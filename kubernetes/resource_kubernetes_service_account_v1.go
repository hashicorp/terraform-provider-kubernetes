// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
)

func resourceKubernetesServiceAccountV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesServiceAccountV1Create,
		ReadContext:   resourceKubernetesServiceAccountV1Read,
		UpdateContext: resourceKubernetesServiceAccountV1Update,
		DeleteContext: resourceKubernetesServiceAccountV1Delete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceKubernetesServiceAccountV1ImportState,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Second),
		},

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("service account", true),
			"image_pull_secret": {
				Type:        schema.TypeSet,
				Description: "A list of references to secrets in the same namespace to use for pulling any images in pods that reference this Service Account. More info: https://kubernetes.io/docs/concepts/containers/images/#specifying-imagepullsecrets-on-a-pod",
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Description: "Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
							Optional:    true,
						},
					},
				},
			},
			"secret": {
				Type:        schema.TypeSet,
				Description: "A list of secrets allowed to be used by pods running using this Service Account. More info: https://kubernetes.io/docs/concepts/configuration/secret",
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Description: "Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
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
				Type:       schema.TypeString,
				Computed:   true,
				Deprecated: "Starting from version 1.24.0 Kubernetes does not automatically generate a token for service accounts, in this case, `default_secret_name` will be empty",
			},
		},
	}
}

func resourceKubernetesServiceAccountV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	svcAcc := corev1.ServiceAccount{
		AutomountServiceAccountToken: ptr.To(d.Get("automount_service_account_token").(bool)),
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

	secret, err := getServiceAccountDefaultSecretV1(ctx, out.Name, svcAcc, d.Timeout(schema.TimeoutCreate), conn)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("default_secret_name", secret.Name)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceKubernetesServiceAccountV1Read(ctx, d, meta)
}

func getServiceAccountDefaultSecretV1(ctx context.Context, name string, config corev1.ServiceAccount, timeout time.Duration, conn *kubernetes.Clientset) (*corev1.Secret, error) {
	sv, err := serverVersionGreaterThanOrEqual(conn, "1.24.0")
	if err != nil {
		return &corev1.Secret{}, err
	}
	if sv {
		return &corev1.Secret{}, nil
	}

	var svcAccTokens []corev1.Secret
	err = retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		resp, err := conn.CoreV1().ServiceAccounts(config.Namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return retry.NonRetryableError(err)
		}

		if len(resp.Secrets) == len(config.Secrets) {
			log.Printf("[DEBUG] Configuration contains %d secrets, saw %d, expected %d", len(config.Secrets), len(resp.Secrets), len(config.Secrets)+1)
			return retry.RetryableError(fmt.Errorf("Waiting for default secret of %q to appear", buildId(resp.ObjectMeta)))
		}

		diff := diffObjectReferences(config.Secrets, resp.Secrets)
		secretList, err := conn.CoreV1().Secrets(config.Namespace).List(ctx, metav1.ListOptions{
			FieldSelector: fmt.Sprintf("type=%s", corev1.SecretTypeServiceAccountToken),
		})
		if err != nil {
			return retry.NonRetryableError(err)
		}

		for _, secret := range secretList.Items {
			for _, svcSecret := range diff {
				if secret.Name != svcSecret.Name {
					continue
				}
				svcAccTokens = append(svcAccTokens, secret)
			}
		}

		if len(svcAccTokens) == 0 {
			return retry.RetryableError(fmt.Errorf("Expected 1 generated service account token, %d found", len(svcAccTokens)))
		}

		if len(svcAccTokens) > 1 {
			return retry.NonRetryableError(fmt.Errorf("Expected 1 generated service account token, %d found: %s", len(svcAccTokens), err))
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &svcAccTokens[0], nil
}

func findDefaultServiceAccountV1(ctx context.Context, sa *corev1.ServiceAccount, conn *kubernetes.Clientset) (string, diag.Diagnostics) {
	/*
	   The default service account token secret would have:
	   - been created either at the same moment as the service account or _just_ after (Kubernetes controllers appears to work off a queue)
	   - have a name starting with "[service account name]-token-"

	   See this for where the default token is created in Kubernetes
	   https://github.com/kubernetes/kubernetes/blob/release-1.13/pkg/controller/serviceaccount/tokens_controller.go#L384
	*/
	ds := make([]string, 0)

	for _, saSecret := range sa.Secrets {
		if !strings.HasPrefix(saSecret.Name, fmt.Sprintf("%s-token-", sa.Name)) {
			log.Printf("[DEBUG] Skipping %s as it doesn't have the right name", saSecret.Name)
			continue
		}

		secret, err := conn.CoreV1().Secrets(sa.Namespace).Get(ctx, saSecret.Name, metav1.GetOptions{})
		if err != nil {
			return "", diag.Errorf("Unable to fetch secret %s/%s from Kubernetes: %s", sa.Namespace, saSecret.Name, err)
		}

		if secret.Type != corev1.SecretTypeServiceAccountToken {
			log.Printf("[DEBUG] Skipping %s as it is of the wrong type", saSecret.Name)
			continue
		}

		if secret.Annotations[corev1.ServiceAccountNameKey] != sa.ObjectMeta.Name {
			log.Printf("[DEBUG] Skipping %s as it has a different name than the service account", saSecret.Name)
			continue
		}

		if secret.Annotations[corev1.ServiceAccountUIDKey] != string(sa.ObjectMeta.UID) {
			log.Printf("[DEBUG] Skipping %s as it has a different UID than the service account", saSecret.Name)
			continue
		}

		log.Printf("[DEBUG] Found %s as a candidate for the default service account token", saSecret.Name)
		ds = append(ds, saSecret.Name)
	}

	if len(ds) == 0 {
		return "", diag.Diagnostics{
			diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "Unable to find any service accounts tokens which could have been the default one.",
			},
		}
	}

	if len(ds) == 1 {
		return ds[0], nil
	}

	return "", diag.Diagnostics{
		diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Unable to discover default secret name.",
			Detail:   "There is more than one service account token associated to the service account.",
		},
	}
}

func diffObjectReferences(origOrs []corev1.ObjectReference, ors []corev1.ObjectReference) []corev1.ObjectReference {
	var diff []corev1.ObjectReference
	uniqueRefs := make(map[string]*corev1.ObjectReference, 0)
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

func resourceKubernetesServiceAccountV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesServiceAccountV1Exists(ctx, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		d.SetId("")
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
	err = d.Set("metadata", flattenMetadata(svcAcc.ObjectMeta, d, meta))
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
	err = d.Set("image_pull_secret", flattenLocalObjectReferenceArray(svcAcc.ImagePullSecrets))
	if err != nil {
		return diag.FromErr(err)
	}

	defaultSecretName := d.Get("default_secret_name").(string)
	log.Printf("[DEBUG] Default secret name is %q", defaultSecretName)
	secrets := flattenServiceAccountSecrets(svcAcc.Secrets, defaultSecretName)
	log.Printf("[DEBUG] Flattened secrets: %#v", secrets)
	err = d.Set("secret", secrets)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesServiceAccountV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	return resourceKubernetesServiceAccountV1Read(ctx, d, meta)
}

func resourceKubernetesServiceAccountV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return nil
		}
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Service account %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesServiceAccountV1Exists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
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
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

func resourceKubernetesServiceAccountV1ImportState(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
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

	defaultSecret, diagMsg := findDefaultServiceAccountV1(ctx, sa, conn)
	if diagMsg.HasError() {
		log.Print("[WARN] Failed to discover the default service account token")
	}

	err = d.Set("default_secret_name", defaultSecret)
	if err != nil {
		return nil, fmt.Errorf("Unable to set default_secret_name: %s", err)
	}

	d.SetId(buildId(sa.ObjectMeta))

	return []*schema.ResourceData{d}, nil
}
