// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
)

func resourceKubernetesSecretV1() *schema.Resource {
	return &schema.Resource{
		Description:   "The resource provides mechanisms to inject containers with sensitive information, such as passwords, while keeping containers agnostic of Kubernetes. Secrets can be used to store sensitive information either as individual properties or coarse-grained entries like entire files or JSON blobs. The resource will by default create a secret which is available to any pod in the specified (or default) namespace.",
		CreateContext: resourceKubernetesSecretV1Create,
		ReadContext:   resourceKubernetesSecretV1Read,
		UpdateContext: resourceKubernetesSecretV1Update,
		DeleteContext: resourceKubernetesSecretV1Delete,
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
				DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
					if v, ok := d.Get("binary_data_wo_revision").(int); ok && v > 0 {
						return true
					}
					if v, ok := d.Get("data_wo_revision").(int); ok && v > 0 {
						return true
					}
					return false
				},
			},
			"data_wo_revision": {
				Type:         schema.TypeInt,
				Description:  `The current revision of the write-only "data_wo" attribute. Incrementing this integer value will cause Terraform to update the write-only value.`,
				Optional:     true,
				RequiredWith: []string{"data_wo"},
				ValidateFunc: validation.IntAtLeast(1),
			},
			"data_wo": {
				Type:          schema.TypeMap,
				Description:   "(Write-only) A map of the secret data.",
				Optional:      true,
				WriteOnly:     true,
				ConflictsWith: []string{"data"},
			},
			"binary_data": {
				Type:        schema.TypeMap,
				Optional:    true,
				Sensitive:   true,
				Description: "A map of the secret data in base64 encoding. Use this for binary data.",
			},
			"binary_data_wo": {
				Type:          schema.TypeMap,
				Description:   "(Write-only)A map of the secret data in base64 encoding. Use this for binary data.",
				Optional:      true,
				ConflictsWith: []string{"binary_data"},
				WriteOnly:     true,
			},
			"binary_data_wo_revision": {
				Type:         schema.TypeInt,
				Description:  `The current revision of the write-only "binary_data_wo" attribute. Incrementing this integer value will cause Terraform to update the write-only value.`,
				Optional:     true,
				RequiredWith: []string{"binary_data_wo"},
				ValidateFunc: validation.IntAtLeast(1),
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

func resourceKubernetesSecretV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	secret := corev1.Secret{
		ObjectMeta: metadata,
	}

	if datarev, ok := d.Get("data_wo_revision").(int); ok && datarev >= 1 {
		wodata, diags := d.GetRawConfigAt(cty.GetAttrPath("data_wo"))
		if diags.HasError() {
			return diags
		}
		if wodata.IsWhollyKnown() && !wodata.IsNull() {
			secret.StringData = expandCtyStringMap(wodata.AsValueMap())
		}
	} else if v, ok := d.GetOk("data"); ok {
		m := map[string]string{}
		for k, v := range v.(map[string]interface{}) {
			vv := v.(string)
			m[k] = vv
		}
		secret.StringData = m
	}

	if bindatarev, ok := d.Get("binary_data_wo_revision").(int); ok && bindatarev >= 1 {
		wobindata, diags := d.GetRawConfigAt(cty.GetAttrPath("binary_data_wo"))
		if diags.HasError() {
			return diags
		}
		if wobindata.IsWhollyKnown() && !wobindata.IsNull() {
			secret.Data = expandCtyBase64MapToByteMap(wobindata.AsValueMap())
		}
	} else if v, ok := d.GetOk("binary_data"); ok {
		m, err := base64DecodeStringMap(v.(map[string]interface{}))
		if err != nil {
			return diag.FromErr(err)
		}
		secret.Data = m
	}

	if v, ok := d.GetOk("type"); ok {
		secret.Type = corev1.SecretType(v.(string))
	}

	if v, ok := d.GetOk("immutable"); ok {
		secret.Immutable = ptr.To(v.(bool))
	}

	log.Printf("[INFO] Creating new secret: %#v", secret)
	out, err := conn.CoreV1().Secrets(metadata.Namespace).Create(ctx, &secret, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitting new secret: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	if out.Type == corev1.SecretTypeServiceAccountToken && d.Get("wait_for_service_account_token").(bool) {
		log.Printf("[DEBUG] Waiting for secret service account token to be created")

		err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutCreate), func() *retry.RetryError {
			secret, err := conn.CoreV1().Secrets(out.Namespace).Get(ctx, out.Name, metav1.GetOptions{})
			if err != nil {
				log.Printf("[DEBUG] Received error: %#v", err)
				return retry.NonRetryableError(err)
			}

			log.Printf("[INFO] Received secret: %#v", secret.Name)
			if _, ok := secret.Data["token"]; ok {
				log.Println("[INFO] Secret service account token created")
				return nil
			}

			return retry.RetryableError(fmt.Errorf(
				"Waiting for secret %q to create service account token", d.Id()))
		})
		if err != nil {
			lastWarnings, wErr := getLastWarningsForObject(ctx, conn, out.ObjectMeta, "Secret", 3)
			if wErr != nil {
				return diag.FromErr(wErr)
			}
			return diag.Errorf("%s%s", err, stringifyEvents(lastWarnings))
		}
	}

	return resourceKubernetesSecretV1Read(ctx, d, meta)
}

func resourceKubernetesSecretV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesSecretV1Exists(ctx, d, meta)
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

	d.Set("type", secret.Type)
	d.Set("immutable", secret.Immutable)

	// NOTE don't read data if write-only attributes are being used
	if v, ok := d.Get("binary_data_wo_revision").(int); ok && v > 0 {
		return nil
	}
	if v, ok := d.Get("data_wo_revision").(int); ok && v > 0 {
		return nil
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

	return nil
}

func resourceKubernetesSecretV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	if d.HasChange("data_wo_revision") {
		updateData = true
		wodata, diags := d.GetRawConfigAt(cty.GetAttrPath("data_wo"))
		if diags.HasError() {
			return diags
		}
		if wodata.IsWhollyKnown() && !wodata.IsNull() {
			data := expandCtyStringMap(wodata.AsValueMap())
			for k, v := range data {
				newData[k] = base64.StdEncoding.EncodeToString([]byte(v))
			}
		}
	} else if d.HasChange("data") {
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

	if d.HasChange("binary_data_wo_revision") {
		updateData = true
		wobindata, diags := d.GetRawConfigAt(cty.GetAttrPath("binary_data_wo"))
		if diags.HasError() {
			return diags
		}
		if wobindata.IsWhollyKnown() && !wobindata.IsNull() {
			data := expandCtyBase64MapToByteMap(wobindata.AsValueMap())
			for k, v := range data {
				newData[k] = base64.StdEncoding.EncodeToString([]byte(v))
			}
		}
	} else if d.HasChange("binary_data") {
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
			Value: ptr.To(d.Get("immutable").(bool)),
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

	return resourceKubernetesSecretV1Read(ctx, d, meta)
}

func resourceKubernetesSecretV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return nil
		}
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Secret %s deleted", name)

	d.SetId("")

	return nil
}

func resourceKubernetesSecretV1Exists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
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
