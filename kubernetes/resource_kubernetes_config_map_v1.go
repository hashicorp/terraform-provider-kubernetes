// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
)

func resourceKubernetesConfigMapV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesConfigMapV1Create,
		ReadContext:   resourceKubernetesConfigMapV1Read,
		UpdateContext: resourceKubernetesConfigMapV1Update,
		DeleteContext: resourceKubernetesConfigMapV1Delete,
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
			"metadata": namespacedMetadataSchema("config map", true),
			"binary_data": {
				Type:         schema.TypeMap,
				Description:  "BinaryData contains the binary data. Each key must consist of alphanumeric characters, '-', '_' or '.'. BinaryData can contain byte sequences that are not in the UTF-8 range. The keys stored in BinaryData must not overlap with the ones in the Data field, this is enforced during validation process. Using this field will require 1.10+ apiserver and kubelet. This field only accepts base64-encoded payloads that will be decoded/encoded before being sent/received to/from the apiserver.",
				Optional:     true,
				ValidateFunc: validateBase64EncodedMap,
			},
			"data": {
				Type:        schema.TypeMap,
				Description: "Data contains the configuration data. Each key must consist of alphanumeric characters, '-', '_' or '.'. Values with non-UTF-8 byte sequences must use the BinaryData field. The keys stored in Data must not overlap with the keys in the BinaryData field, this is enforced during validation process.",
				Optional:    true,
			},
			"immutable": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Immutable, if set to true, ensures that data stored in the ConfigMap cannot be updated (only object metadata can be modified). If not set to true, the field can be modified at any time. Defaulted to nil.",
			},
		},
	}
}

func resourceKubernetesConfigMapV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	cfgMap := corev1.ConfigMap{
		ObjectMeta: metadata,
		BinaryData: expandBase64MapToByteMap(d.Get("binary_data").(map[string]interface{})),
		Data:       expandStringMap(d.Get("data").(map[string]interface{})),
		Immutable:  ptr.To(d.Get("immutable").(bool)),
	}

	log.Printf("[INFO] Creating new config map: %#v", cfgMap)
	out, err := conn.CoreV1().ConfigMaps(metadata.Namespace).Create(ctx, &cfgMap, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Submitted new config map: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesConfigMapV1Read(ctx, d, meta)
}

func resourceKubernetesConfigMapV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesConfigMapV1Exists(ctx, d, meta)
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
	log.Printf("[INFO] Reading config map %s", name)
	cfgMap, err := conn.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Received config map: %#v", cfgMap)
	err = d.Set("metadata", flattenMetadata(cfgMap.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("binary_data", flattenByteMapToBase64Map(cfgMap.BinaryData))
	d.Set("data", cfgMap.Data)
	d.Set("immutable", cfgMap.Immutable)

	return nil
}

func resourceKubernetesConfigMapV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("binary_data") {
		oldV, newV := d.GetChange("binary_data")
		diffOps := diffStringMap("/binaryData/", oldV.(map[string]interface{}), newV.(map[string]interface{}))
		ops = append(ops, diffOps...)
	}

	if d.HasChange("data") {
		oldV, newV := d.GetChange("data")
		diffOps := diffStringMap("/data/", oldV.(map[string]interface{}), newV.(map[string]interface{}))
		ops = append(ops, diffOps...)
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

	log.Printf("[INFO] Updating config map %q: %v", name, string(data))
	out, err := conn.CoreV1().ConfigMaps(namespace).Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.Errorf("Failed to update Config Map: %s", err)
	}
	log.Printf("[INFO] Submitted updated config map: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesConfigMapV1Read(ctx, d, meta)
}

func resourceKubernetesConfigMapV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Deleting config map: %#v", name)
	err = conn.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return nil
		}
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Config map %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesConfigMapV1Exists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking config map %s", name)
	_, err = conn.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
