// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	v1 "k8s.io/api/core/v1"
	api "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

func resourceKubernetesStorageClassV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesStorageClassV1Create,
		ReadContext:   resourceKubernetesStorageClassV1Read,
		UpdateContext: resourceKubernetesStorageClassV1Update,
		DeleteContext: resourceKubernetesStorageClassV1Delete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("storage class", true),
			"parameters": {
				Type:        schema.TypeMap,
				Description: "The parameters for the provisioner that should create volumes of this storage class",
				Optional:    true,
				ForceNew:    true,
			},
			"storage_provisioner": {
				Type:        schema.TypeString,
				Description: "Indicates the type of the provisioner",
				Required:    true,
				ForceNew:    true,
			},
			"reclaim_policy": {
				Type:        schema.TypeString,
				Description: "Indicates the type of the reclaim policy",
				Optional:    true,
				Default:     string(v1.PersistentVolumeReclaimDelete),
				ValidateFunc: validation.StringInSlice([]string{
					string(v1.PersistentVolumeReclaimRecycle),
					string(v1.PersistentVolumeReclaimDelete),
					string(v1.PersistentVolumeReclaimRetain),
				}, false),
			},
			"volume_binding_mode": {
				Type:        schema.TypeString,
				Description: "Indicates when volume binding and dynamic provisioning should occur",
				Optional:    true,
				ForceNew:    true,
				Default:     string(api.VolumeBindingImmediate),
				ValidateFunc: validation.StringInSlice([]string{
					string(api.VolumeBindingImmediate),
					string(api.VolumeBindingWaitForFirstConsumer),
				}, false),
			},
			"allow_volume_expansion": {
				Type:        schema.TypeBool,
				Description: "Indicates whether the storage class allow volume expand",
				Optional:    true,
				Default:     true,
			},
			"mount_options": {
				Type:        schema.TypeSet,
				Description: "Persistent Volumes that are dynamically created by a storage class will have the mount options specified",
				Optional:    true,
				ForceNew:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Set:         schema.HashString,
			},
			"allowed_topologies": {
				Type:        schema.TypeList,
				Description: "Restrict the node topologies where volumes can be dynamically provisioned.",
				Optional:    true,
				ForceNew:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"match_label_expressions": {
							Type:        schema.TypeList,
							Description: "A list of topology selector requirements by labels.",
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"key": {
										Type:        schema.TypeString,
										Description: "The label key that the selector applies to.",
										Optional:    true,
									},
									"values": {
										Type:        schema.TypeSet,
										Description: "An array of string values. One value must match the label to be selected.",
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func resourceKubernetesStorageClassV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	reclaimPolicy := v1.PersistentVolumeReclaimPolicy(d.Get("reclaim_policy").(string))
	volumeBindingMode := api.VolumeBindingMode(d.Get("volume_binding_mode").(string))
	allowVolumeExpansion := d.Get("allow_volume_expansion").(bool)
	storageClass := api.StorageClass{
		ObjectMeta:           metadata,
		Provisioner:          d.Get("storage_provisioner").(string),
		ReclaimPolicy:        &reclaimPolicy,
		VolumeBindingMode:    &volumeBindingMode,
		AllowVolumeExpansion: &allowVolumeExpansion,
	}

	if v, ok := d.GetOk("parameters"); ok {
		storageClass.Parameters = expandStringMap(v.(map[string]interface{}))
	}

	if v, ok := d.GetOk("mount_options"); ok {
		storageClass.MountOptions = schemaSetToStringArray(v.(*schema.Set))
	}
	if v, ok := d.GetOk("allowed_topologies"); ok && len(v.([]interface{})) > 0 {
		storageClass.AllowedTopologies = expandStorageClassAllowedTopologies(v.([]interface{}))
	}

	log.Printf("[INFO] Creating new storage class: %#v", storageClass)
	out, err := conn.StorageV1().StorageClasses().Create(ctx, &storageClass, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Submitted new storage class: %#v", out)
	d.SetId(out.Name)

	return resourceKubernetesStorageClassV1Read(ctx, d, meta)
}

func resourceKubernetesStorageClassV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	diags := diag.Diagnostics{}

	exists, err := resourceKubernetesStorageClassV1Exists(ctx, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		d.SetId("")
		return diags
	}
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Id()
	log.Printf("[INFO] Reading storage class %s", name)
	storageClass, err := conn.StorageV1().StorageClasses().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Received storage class: %#v", storageClass)

	err = d.Set("metadata", flattenMetadata(storageClass.ObjectMeta, d, meta))
	if err != nil {
		diags = append(diags, diag.FromErr(err)[0])
	}

	sc := flattenStorageClass(*storageClass)
	for k, v := range sc {
		err = d.Set(k, v)
		if err != nil {
			diags = append(diags, diag.FromErr(err)[0])
		}
	}
	return diags
}

func resourceKubernetesStorageClassV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Id()
	ops := patchMetadata("metadata.0.", "/metadata/", d)

	if d.HasChange("allow_volume_expansion") {
		newVal := d.Get("allow_volume_expansion").(bool)
		ops = append(ops, &ReplaceOperation{
			Path:  "/allowVolumeExpansion",
			Value: newVal,
		})
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}

	log.Printf("[INFO] Updating storage class %q: %v", name, string(data))
	out, err := conn.StorageV1().StorageClasses().Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.Errorf("Failed to update storage class: %s", err)
	}
	log.Printf("[INFO] Submitted updated storage class: %#v", out)
	d.SetId(out.ObjectMeta.Name)

	return resourceKubernetesStorageClassV1Read(ctx, d, meta)
}

func resourceKubernetesStorageClassV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Id()
	log.Printf("[INFO] Deleting storage class: %#v", name)
	err = conn.StorageV1().StorageClasses().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return nil
		}
		return diag.FromErr(err)
	}

	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutDelete), func() *retry.RetryError {
		_, err := conn.StorageV1().StorageClasses().Get(ctx, d.Id(), metav1.GetOptions{})
		if err != nil {
			if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
				return nil
			}
			return retry.NonRetryableError(err)
		}

		e := fmt.Errorf("storage class (%s) still exists", d.Id())
		return retry.RetryableError(e)
	})
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Storage class %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesStorageClassV1Exists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	name := d.Id()
	log.Printf("[INFO] Checking storage class %s", name)
	_, err = conn.StorageV1().StorageClasses().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
