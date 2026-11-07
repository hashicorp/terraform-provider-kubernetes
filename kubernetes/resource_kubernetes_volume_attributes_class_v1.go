package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	storage "k8s.io/api/storage/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func resourceKubernetesVolumeAttributesClassV1() *schema.Resource {
	return &schema.Resource{
		Description:   "A VolumeAttributesClass represents a specification of mutable volume attributes.",
		CreateContext: resourceKubernetesVolumeAttributesClassV1Create,
		ReadContext:   resourceKubernetesVolumeAttributesClassV1Read,
		UpdateContext: resourceKubernetesVolumeAttributesClassV1Update,
		DeleteContext: resourceKubernetesVolumeAttributesClassV1Delete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("volume attributes class", true),
			"driver_name": {
				Type:        schema.TypeString,
				Description: "Name of the CSI driver. This field is immutable.",
				Required:    true,
				ForceNew:    true,
			},
			"parameters": {
				Type:        schema.TypeMap,
				Description: "parameters hold volume attributes defined by the CSI driver. This field is immutable.",
				Optional:    true,
				ForceNew:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceKubernetesVolumeAttributesClassV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	params := expandStringMap(d.Get("parameters").(map[string]interface{}))

	volumeAttrsClass := &storage.VolumeAttributesClass{
		ObjectMeta: metadata,
		DriverName: d.Get("driver_name").(string),
		Parameters: params,
	}

	log.Printf("[INFO] Creating new VolumeAttributesClass: %#v", volumeAttrsClass)
	result, err := conn.StorageV1beta1().VolumeAttributesClasses().Create(ctx, volumeAttrsClass, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Submitted new VolumeAttributesClass: %#v", result)

	d.SetId(buildId(result.ObjectMeta))

	return resourceKubernetesVolumeAttributesClassV1Read(ctx, d, meta)
}

func resourceKubernetesVolumeAttributesClassV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Id()
	log.Printf("[INFO] Reading VolumeAttributesClass %s", name)
	volumeAttrsClass, err := conn.StorageV1beta1().VolumeAttributesClasses().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			d.SetId("")
			return nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Received VolumeAttributesClass: %#v", volumeAttrsClass)

	err = d.Set("metadata", flattenMetadata(volumeAttrsClass.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("driver_name", volumeAttrsClass.DriverName)
	if err != nil {
		return diag.FromErr(err)
	}

	if len(volumeAttrsClass.Parameters) > 0 {
		err = d.Set("parameters", volumeAttrsClass.Parameters)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourceKubernetesVolumeAttributesClassV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Id()
	metadata := expandMetadata(d.Get("metadata").([]interface{}))

	volumeAttrsClass := &storage.VolumeAttributesClass{
		ObjectMeta: metadata,
		DriverName: d.Get("driver_name").(string),
	}

	if d.HasChange("parameters") {
		volumeAttrsClass.Parameters = expandStringMap(d.Get("parameters").(map[string]interface{}))
	}

	log.Printf("[INFO] Updating VolumeAttributesClass %s", name)
	_, err = conn.StorageV1beta1().VolumeAttributesClasses().Update(ctx, volumeAttrsClass, metav1.UpdateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceKubernetesVolumeAttributesClassV1Read(ctx, d, meta)
}

func resourceKubernetesVolumeAttributesClassV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Id()
	log.Printf("[INFO] Deleting VolumeAttributesClass %s", name)
	err = conn.StorageV1beta1().VolumeAttributesClasses().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return diag.FromErr(err)
	}

	log.Printf("[INFO] VolumeAttributesClass %s deleted", name)
	return nil
}
