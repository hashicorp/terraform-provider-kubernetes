package v1

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	providermetav1 "github.com/hashicorp/terraform-provider-kubernetes/kubernetes/meta/v1"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes/structures"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func ResourceKubernetesSecret() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesSecretCreate,
		ReadContext:   resourceKubernetesSecretRead,
		UpdateContext: resourceKubernetesSecretUpdate,
		DeleteContext: resourceKubernetesSecretDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		CustomizeDiff: func(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
			if diff.Id() == "" {
				return nil
			}

			// ForceNew if immutable has been set to true
			// and there are any changes to data, binary_data, or immutable
			immutable, _ := diff.GetChange("immutable")
			if immutable.(bool) {
				immutableFields := []string{
					"data",
					"binary_data",
					"immutable",
				}
				for _, f := range immutableFields {
					if diff.HasChange(f) {
						diff.ForceNew(f)
					}
				}
			}

			return nil
		},

		Schema: map[string]*schema.Schema{
			"metadata": providermetav1.NamespacedMetadataSchema("secret", true),
			"data": {
				Type:        schema.TypeMap,
				Description: "A map of the secret data.",
				Optional:    true,
				Computed:    true,
				Sensitive:   true,
			},
			"binary_data": {
				Type:        schema.TypeMap,
				Optional:    true,
				Sensitive:   true,
				Description: "A map of the secret data in structures.Base64 encoding. Use this for binary data.",
			},
			"immutable": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Ensures that data stored in the Secret cannot be updated (only object metadata can be modified).",
			},
			"type": {
				Type:        schema.TypeString,
				Description: "Type of secret",
				Default:     string(corev1.SecretTypeOpaque),
				Optional:    true,
				ForceNew:    true,
			},
			"wait_for_service_account_token": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Terraform will wait for the service account token to be created.",
			},
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(1 * time.Minute),
		},
	}
}

func resourceKubernetesSecretCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(provider.KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := providermetav1.ExpandMetadata(d.Get("metadata").([]interface{}))
	secret := corev1.Secret{
		ObjectMeta: metadata,
	}

	if v, ok := d.GetOk("data"); ok {
		m := map[string]string{}
		for k, v := range v.(map[string]interface{}) {
			vv := v.(string)
			m[k] = vv
		}
		secret.StringData = m
	}

	if v, ok := d.GetOk("binary_data"); ok {
		m, err := structures.Base64DecodeStringMap(v.(map[string]interface{}))
		if err != nil {
			return diag.FromErr(err)
		}
		secret.Data = m
	}

	if v, ok := d.GetOk("type"); ok {
		secret.Type = corev1.SecretType(v.(string))
	}

	if v, ok := d.GetOkExists("immutable"); ok {
		secret.Immutable = structures.PtrToBool(v.(bool))
	}

	log.Printf("[INFO] Creating new secret: %#v", secret)
	out, err := conn.CoreV1().Secrets(metadata.Namespace).Create(ctx, &secret, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitting new secret: %#v", out)
	d.SetId(providermetav1.BuildId(out.ObjectMeta))

	if out.Type == corev1.SecretTypeServiceAccountToken && d.Get("wait_for_service_account_token").(bool) {
		log.Printf("[DEBUG] Waiting for secret service account token to be created")

		err = resource.RetryContext(ctx, d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
			secret, err := conn.CoreV1().Secrets(out.Namespace).Get(ctx, out.Name, metav1.GetOptions{})
			if err != nil {
				log.Printf("[DEBUG] Received error: %#v", err)
				return resource.NonRetryableError(err)
			}

			log.Printf("[INFO] Received secret: %#v", secret.Name)
			if _, ok := secret.Data["token"]; ok {
				log.Println("[INFO] Secret service account token created")
				return nil
			}

			return resource.RetryableError(fmt.Errorf(
				"Waiting for secret %q to create service account token", d.Id()))
		})
		if err != nil {
			lastWarnings, wErr := structures.GetLastWarningsForObject(ctx, conn, out.ObjectMeta, "Secret", 3)
			if wErr != nil {
				return diag.FromErr(wErr)
			}
			return diag.Errorf("%s%s", err, structures.StringifyEvents(lastWarnings))
		}
	}

	return resourceKubernetesSecretRead(ctx, d, meta)
}

func resourceKubernetesSecretRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesSecretExists(ctx, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		d.SetId("")
		return diag.Diagnostics{}
	}
	conn, err := meta.(provider.KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := providermetav1.IdParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Reading secret %s", name)
	secret, err := conn.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Received secret: %#v", secret.ObjectMeta)
	err = d.Set("metadata", providermetav1.FlattenMetadata(secret.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	binaryDataKeys := []string{}
	if v, ok := d.GetOk("binary_data"); ok {
		binaryData := map[string][]byte{}
		for k := range v.(map[string]interface{}) {
			binaryData[k] = secret.Data[k]
			binaryDataKeys = append(binaryDataKeys, k)
		}
		d.Set("binary_data", structures.Base64EncodeByteMap(binaryData))
	}

	for _, k := range binaryDataKeys {
		delete(secret.Data, k)
	}
	d.Set("data", structures.FlattenByteMapToStringMap(secret.Data))
	d.Set("type", secret.Type)
	d.Set("immutable", secret.Immutable)

	return nil
}

func resourceKubernetesSecretUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(provider.KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := providermetav1.IdParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	ops := providermetav1.PatchMetadata("metadata.0.", "/metadata/", d)

	newData := map[string]interface{}{}
	updateData := false
	if d.HasChange("data") {
		_, new := d.GetChange("data")
		new = structures.Base64EncodeStringMap(new.(map[string]interface{}))
		for k, v := range new.(map[string]interface{}) {
			newData[k] = v
		}
		updateData = true
	} else if v, ok := d.GetOk("data"); ok {
		for k, vv := range structures.Base64EncodeStringMap(v.(map[string]interface{})) {
			newData[k] = vv
		}
	}
	if d.HasChange("binary_data") {
		_, new := d.GetChange("binary_data")
		for k, v := range new.(map[string]interface{}) {
			newData[k] = v
		}
		updateData = true
	} else if v, ok := d.GetOk("binary_data"); ok {
		for k, vv := range v.(map[string]interface{}) {
			newData[k] = vv
		}
	}

	if updateData {
		ops = append(ops, &structures.AddOperation{
			Path:  "/data",
			Value: newData,
		})
	}

	if d.HasChange("immutable") {
		ops = append(ops, &structures.ReplaceOperation{
			Path:  "/immutable",
			Value: structures.PtrToBool(d.Get("immutable").(bool)),
		})
	}

	data, err := ops.MarshalJSON()
	if err != nil {
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}

	log.Printf("[INFO] Updating secret %q: %v", name, data)
	out, err := conn.CoreV1().Secrets(namespace).Patch(ctx, name, types.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.Errorf("Failed to update secret: %s", err)
	}

	log.Printf("[INFO] Submitting updated secret: %#v", out.ObjectMeta)
	d.SetId(providermetav1.BuildId(out.ObjectMeta))

	return resourceKubernetesSecretRead(ctx, d, meta)
}

func resourceKubernetesSecretDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(provider.KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := providermetav1.IdParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting secret: %q", name)
	err = conn.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Secret %s deleted", name)

	d.SetId("")

	return nil
}

func resourceKubernetesSecretExists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(provider.KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	namespace, name, err := providermetav1.IdParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking secret %s", name)
	_, err = conn.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}

	return true, err
}
