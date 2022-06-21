package v1

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"

	providermetav1 "github.com/hashicorp/terraform-provider-kubernetes/kubernetes/meta/v1"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes/structures"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes/validators"
)

func ResourceKubernetesConfigMap() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesConfigMapCreate,
		ReadContext:   resourceKubernetesConfigMapRead,
		UpdateContext: resourceKubernetesConfigMapUpdate,
		DeleteContext: resourceKubernetesConfigMapDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"metadata": providermetav1.NamespacedMetadataSchema("config map", true),
			"binary_data": {
				Type:         schema.TypeMap,
				Description:  "BinaryData contains the binary data. Each key must consist of alphanumeric characters, '-', '_' or '.'. BinaryData can contain byte sequences that are not in the UTF-8 range. The keys stored in BinaryData must not overlap with the ones in the Data field, this is enforced during validation process. Using this field will require 1.10+ apiserver and kubelet. This field only accepts base64-encoded payloads that will be decoded/encoded before being sent/received to/from the apiserver.",
				Optional:     true,
				ValidateFunc: validators.ValidateAnnotations,
			},
			"data": {
				Type:        schema.TypeMap,
				Description: "Data contains the configuration data. Each key must consist of alphanumeric characters, '-', '_' or '.'. Values with non-UTF-8 byte sequences must use the BinaryData field. The keys stored in Data must not overlap with the keys in the BinaryData field, this is enforced during validation process.",
				Optional:    true,
			},
		},
	}
}

func resourceKubernetesConfigMapCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(provider.KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := providermetav1.ExpandMetadata(d.Get("metadata").([]interface{}))
	cfgMap := api.ConfigMap{
		ObjectMeta: metadata,
		BinaryData: structures.ExpandBase64MapToByteMap(d.Get("binary_data").(map[string]interface{})),
		Data:       structures.ExpandStringMap(d.Get("data").(map[string]interface{})),
	}
	log.Printf("[INFO] Creating new config map: %#v", cfgMap)
	out, err := conn.CoreV1().ConfigMaps(metadata.Namespace).Create(ctx, &cfgMap, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Submitted new config map: %#v", out)
	d.SetId(providermetav1.BuildId(out.ObjectMeta))

	return resourceKubernetesConfigMapRead(ctx, d, meta)
}

func resourceKubernetesConfigMapRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesConfigMapExists(ctx, d, meta)
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
	log.Printf("[INFO] Reading config map %s", name)
	cfgMap, err := conn.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Received config map: %#v", cfgMap)
	err = d.Set("metadata", providermetav1.FlattenMetadata(cfgMap.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("binary_data", structures.FlattenByteMapToBase64Map(cfgMap.BinaryData))
	d.Set("data", cfgMap.Data)

	return nil
}

func resourceKubernetesConfigMapUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(provider.KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := providermetav1.IdParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	ops := providermetav1.PatchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("binary_data") {
		oldV, newV := d.GetChange("binary_data")
		diffOps := structures.DiffStringMap("/binaryData/", oldV.(map[string]interface{}), newV.(map[string]interface{}))
		ops = append(ops, diffOps...)
	}

	if d.HasChange("data") {
		oldV, newV := d.GetChange("data")
		diffOps := structures.DiffStringMap("/data/", oldV.(map[string]interface{}), newV.(map[string]interface{}))
		ops = append(ops, diffOps...)
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
	d.SetId(providermetav1.BuildId(out.ObjectMeta))

	return resourceKubernetesConfigMapRead(ctx, d, meta)
}

func resourceKubernetesConfigMapDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(provider.KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := providermetav1.IdParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Deleting config map: %#v", name)
	err = conn.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Config map %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesConfigMapExists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(provider.KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	namespace, name, err := providermetav1.IdParts(d.Id())
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
