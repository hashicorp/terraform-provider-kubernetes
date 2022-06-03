package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

func resourceKubernetesSecret() *schema.Resource {
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
			"metadata": namespacedMetadataSchema("secret", true),
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
				Description: "A map of the secret data in base64 encoding. Use this for binary data.",
			},
			"immutable": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Ensures that data stored in the Secret cannot be updated (only object metadata can be modified).",
			},
			"type": {
				Type:        schema.TypeString,
				Description: "Type of secret",
				Default:     string(api.SecretTypeOpaque),
				Optional:    true,
				ForceNew:    true,
			},
		},
	}
}

func resourceKubernetesSecretCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	secret := api.Secret{
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
		m, err := base64DecodeStringMap(v.(map[string]interface{}))
		if err != nil {
			return diag.FromErr(err)
		}
		secret.Data = m
	}

	if v, ok := d.GetOk("type"); ok {
		secret.Type = api.SecretType(v.(string))
	}

	if v, ok := d.GetOkExists("immutable"); ok {
		secret.Immutable = ptrToBool(v.(bool))
	}

	log.Printf("[INFO] Creating new secret: %#v", secret)
	out, err := conn.CoreV1().Secrets(metadata.Namespace).Create(ctx, &secret, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitting new secret: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

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
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Reading secret %s", name)
	secret, err := conn.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Received secret: %#v", secret.ObjectMeta)
	err = d.Set("metadata", flattenMetadata(secret.ObjectMeta, d, meta))
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
		d.Set("binary_data", base64EncodeByteMap(binaryData))
	}

	for _, k := range binaryDataKeys {
		delete(secret.Data, k)
	}
	d.Set("data", flattenByteMapToStringMap(secret.Data))
	d.Set("type", secret.Type)
	d.Set("immutable", secret.Immutable)

	return nil
}

func resourceKubernetesSecretUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)

	newData := map[string]interface{}{}
	updateData := false
	if d.HasChange("data") {
		_, new := d.GetChange("data")
		new = base64EncodeStringMap(new.(map[string]interface{}))
		for k, v := range new.(map[string]interface{}) {
			newData[k] = v
		}
		updateData = true
	} else if v, ok := d.GetOk("data"); ok {
		for k, vv := range base64EncodeStringMap(v.(map[string]interface{})) {
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
		ops = append(ops, &AddOperation{
			Path:  "/data",
			Value: newData,
		})
	}

	if d.HasChange("immutable") {
		ops = append(ops, &ReplaceOperation{
			Path:  "/immutable",
			Value: ptrToBool(d.Get("immutable").(bool)),
		})
	}

	data, err := ops.MarshalJSON()
	if err != nil {
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}

	log.Printf("[INFO] Updating secret %q: %v", name, data)
	out, err := conn.CoreV1().Secrets(namespace).Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.Errorf("Failed to update secret: %s", err)
	}

	log.Printf("[INFO] Submitting updated secret: %#v", out.ObjectMeta)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesSecretRead(ctx, d, meta)
}

func resourceKubernetesSecretDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
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
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	namespace, name, err := idParts(d.Id())
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
