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

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("secret", true),
			"data": {
				Type:        schema.TypeMap,
				Description: "A map of the secret data.",
				Optional:    true,
				Sensitive:   true,
			},
			"type": {
				Type:        schema.TypeString,
				Description: "Type of secret",
				Default:     "Opaque",
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
	var secret api.Secret
	if data, ok := d.GetOk("data"); ok {
		secret = api.Secret{
			ObjectMeta: metadata,
			Data:       expandStringMapToByteMap(data.(map[string]interface{})),
		}
	} else {
		secret = api.Secret{
			ObjectMeta: metadata,
		}
	}

	if v, ok := d.GetOk("type"); ok {
		secret.Type = api.SecretType(v.(string))
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

	log.Printf("[INFO] Received secret: %#v", secret)
	err = d.Set("metadata", flattenMetadata(secret.ObjectMeta, d))
	if err != nil {
		return diag.FromErr(err)
	}

	if _, ok := d.GetOk("data"); ok {
		d.Set("data", flattenByteMapToStringMap(secret.Data))
	}
	d.Set("type", secret.Type)

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
	if d.HasChange("data") {
		oldV, newV := d.GetChange("data")

		oldV = base64EncodeStringMap(oldV.(map[string]interface{}))
		newV = base64EncodeStringMap(newV.(map[string]interface{}))

		diffOps := diffStringMap("/data/", oldV.(map[string]interface{}), newV.(map[string]interface{}))

		ops = append(ops, diffOps...)
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

	log.Printf("[INFO] Submitting updated secret: %#v", out)
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
