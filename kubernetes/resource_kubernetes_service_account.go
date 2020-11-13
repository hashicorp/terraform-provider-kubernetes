package kubernetes

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

func resourceKubernetesServiceAccount() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesServiceAccountCreate,
		ReadContext:   resourceKubernetesServiceAccountRead,
		UpdateContext: resourceKubernetesServiceAccountUpdate,
		DeleteContext: resourceKubernetesServiceAccountDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceKubernetesServiceAccountImportState,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Second),
		},

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("service account", true),
			"image_pull_secret": {
				Type:        schema.TypeSet,
				Description: "A list of references to secrets in the same namespace to use for pulling any images in pods that reference this Service Account. More info: http://kubernetes.io/docs/user-guide/secrets#manually-specifying-an-imagepullsecret",
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Description: "Name of the referent. More info: http://kubernetes.io/docs/user-guide/identifiers#names",
							Optional:    true,
						},
					},
				},
			},
			"secret": {
				Type:        schema.TypeSet,
				Description: "A list of secrets allowed to be used by pods running using this Service Account. More info: http://kubernetes.io/docs/user-guide/secrets",
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Description: "Name of the referent. More info: http://kubernetes.io/docs/user-guide/identifiers#names",
							Optional:    true,
						},
					},
				},
			},
			"automount_service_account_token": {
				Type:        schema.TypeBool,
				Description: "Enable automatic mounting of the service account token",
				Optional:    true,
				Default:     true,
			},
			"default_secret_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceKubernetesServiceAccountCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	svcAcc := api.ServiceAccount{
		AutomountServiceAccountToken: ptrToBool(d.Get("automount_service_account_token").(bool)),
		ObjectMeta:                   metadata,
		ImagePullSecrets:             expandLocalObjectReferenceArray(d.Get("image_pull_secret").(*schema.Set).List()),
		Secrets:                      expandServiceAccountSecrets(d.Get("secret").(*schema.Set).List(), ""),
	}
	log.Printf("[INFO] Creating new service account: %#v", svcAcc)
	out, err := conn.CoreV1().ServiceAccounts(metadata.Namespace).Create(ctx, &svcAcc, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Submitted new service account: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	secret, err := getServiceAccountDefaultSecret(ctx, out.Name, svcAcc, d.Timeout(schema.TimeoutCreate), conn)
	d.Set("default_secret_name", secret.Name)
	return resourceKubernetesServiceAccountRead(ctx, d, meta)
}

func getServiceAccountDefaultSecret(ctx context.Context, name string, config api.ServiceAccount, timeout time.Duration, conn *kubernetes.Clientset) (*api.Secret, error) {
	var svcAccTokens []api.Secret
	err := resource.Retry(timeout, func() *resource.RetryError {
		resp, err := conn.CoreV1().ServiceAccounts(config.Namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return resource.NonRetryableError(err)
		}

		if len(resp.Secrets) == len(config.Secrets) {
			log.Printf("[DEBUG] Configuration contains %d secrets, saw %d, expected %d", len(config.Secrets), len(resp.Secrets), len(config.Secrets)+1)
			return resource.RetryableError(fmt.Errorf("Waiting for default secret of %q to appear", buildId(resp.ObjectMeta)))
		}

		diff := diffObjectReferences(config.Secrets, resp.Secrets)
		secretList, err := conn.CoreV1().Secrets(config.Namespace).List(ctx, metav1.ListOptions{
			FieldSelector: fmt.Sprintf("type=%s", api.SecretTypeServiceAccountToken),
		})
		for _, secret := range secretList.Items {
			for _, svcSecret := range diff {
				if secret.Name != svcSecret.Name {
					continue
				}
				svcAccTokens = append(svcAccTokens, secret)
			}
		}

		if len(svcAccTokens) == 0 {
			return resource.RetryableError(fmt.Errorf("Expected 1 generated service account token, %d found", len(svcAccTokens)))
		}

		if len(svcAccTokens) > 1 {
			return resource.NonRetryableError(fmt.Errorf("Expected 1 generated service account token, %d found: %s", len(svcAccTokens), err))
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &svcAccTokens[0], nil
}

func findDefaultServiceAccount(ctx context.Context, sa *api.ServiceAccount, conn *kubernetes.Clientset) (string, error) {
	/*
	   The default service account token secret would have:
	   - been created either at the same moment as the service account or _just_ after (Kubernetes controllers appears to work off a queue)
	   - have a name starting with "[service account name]-token-"

	   See this for where the default token is created in Kubernetes
	   https://github.com/kubernetes/kubernetes/blob/release-1.13/pkg/controller/serviceaccount/tokens_controller.go#L384
	*/
	for _, saSecret := range sa.Secrets {
		if !strings.HasPrefix(saSecret.Name, fmt.Sprintf("%s-token-", sa.Name)) {
			log.Printf("[DEBUG] Skipping %s as it doesn't have the right name", saSecret.Name)
			continue
		}

		secret, err := conn.CoreV1().Secrets(sa.Namespace).Get(ctx, saSecret.Name, metav1.GetOptions{})
		if err != nil {
			return "", fmt.Errorf("Unable to fetch secret %s/%s from Kubernetes: %s", sa.Namespace, saSecret.Name, err)
		}

		if secret.Type != api.SecretTypeServiceAccountToken {
			log.Printf("[DEBUG] Skipping %s as it is of the wrong type", saSecret.Name)
			continue
		}

		if secret.CreationTimestamp.Before(&sa.CreationTimestamp) {
			log.Printf("[DEBUG] Skipping %s as it existed before the service account", saSecret.Name)
			continue
		}

		if secret.CreationTimestamp.Sub(sa.CreationTimestamp.Time) > (3 * time.Second) {
			log.Printf("[DEBUG] Skipping %s as it wasn't created at the same time as the service account", saSecret.Name)
			continue
		}

		log.Printf("[DEBUG] Found %s as a candidate for the default service account token", saSecret.Name)

		return saSecret.Name, nil
	}

	return "", fmt.Errorf("Unable to find any service accounts tokens which could have been the default one")
}

func diffObjectReferences(origOrs []api.ObjectReference, ors []api.ObjectReference) []api.ObjectReference {
	var diff []api.ObjectReference
	uniqueRefs := make(map[string]*api.ObjectReference, 0)
	for _, or := range origOrs {
		uniqueRefs[or.Name] = &or
	}

	for _, or := range ors {
		_, found := uniqueRefs[or.Name]
		if !found {
			diff = append(diff, or)
		}
	}

	return diff
}

func resourceKubernetesServiceAccountRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesServiceAccountExists(ctx, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		return diag.Diagnostics{}
	}
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Reading service account %s", name)
	svcAcc, err := conn.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Received service account: %#v", svcAcc)
	err = d.Set("metadata", flattenMetadata(svcAcc.ObjectMeta, d))
	if err != nil {
		return diag.FromErr(err)
	}

	if svcAcc.AutomountServiceAccountToken == nil {
		err = d.Set("automount_service_account_token", false)
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		err = d.Set("automount_service_account_token", *svcAcc.AutomountServiceAccountToken)
		if err != nil {
			return diag.FromErr(err)
		}
	}
	d.Set("image_pull_secret", flattenLocalObjectReferenceArray(svcAcc.ImagePullSecrets))

	defaultSecretName := d.Get("default_secret_name").(string)
	log.Printf("[DEBUG] Default secret name is %q", defaultSecretName)
	secrets := flattenServiceAccountSecrets(svcAcc.Secrets, defaultSecretName)
	log.Printf("[DEBUG] Flattened secrets: %#v", secrets)
	d.Set("secret", secrets)

	return nil
}

func resourceKubernetesServiceAccountUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("image_pull_secret") {
		v := d.Get("image_pull_secret").(*schema.Set).List()
		ops = append(ops, &ReplaceOperation{
			Path:  "/imagePullSecrets",
			Value: expandLocalObjectReferenceArray(v),
		})
	}
	if d.HasChange("secret") {
		v := d.Get("secret").(*schema.Set).List()
		defaultSecretName := d.Get("default_secret_name").(string)

		ops = append(ops, &ReplaceOperation{
			Path:  "/secrets",
			Value: expandServiceAccountSecrets(v, defaultSecretName),
		})
	}
	if d.HasChange("automount_service_account_token") {
		v := d.Get("automount_service_account_token").(bool)
		ops = append(ops, &ReplaceOperation{
			Path:  "/automountServiceAccountToken",
			Value: v,
		})
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating service account %q: %v", name, string(data))
	out, err := conn.CoreV1().ServiceAccounts(namespace).Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.Errorf("Failed to update service account: %s", err)
	}
	log.Printf("[INFO] Submitted updated service account: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesServiceAccountRead(ctx, d, meta)
}

func resourceKubernetesServiceAccountDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting service account: %#v", name)
	err = conn.CoreV1().ServiceAccounts(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Service account %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesServiceAccountExists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking service account %s", name)
	_, err = conn.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

func resourceKubernetesServiceAccountImportState(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return nil, err
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return nil, fmt.Errorf("Unable to parse identifier %s: %s", d.Id(), err)
	}

	sa, err := conn.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("Unable to fetch service account from Kubernetes: %s", err)
	}
	defaultSecret, err := findDefaultServiceAccount(ctx, sa, conn)
	if err != nil {
		return nil, fmt.Errorf("Failed to discover the default service account token: %s", err)
	}

	err = d.Set("default_secret_name", defaultSecret)
	if err != nil {
		return nil, fmt.Errorf("Unable to set default_secret_name: %s", err)
	}
	d.SetId(buildId(sa.ObjectMeta))

	return []*schema.ResourceData{d}, nil
}
