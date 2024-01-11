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
	storage "k8s.io/api/storage/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

func resourceKubernetesCSIDriverV1Beta1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesCSIDriverV1Beta1Create,
		ReadContext:   resourceKubernetesCSIDriverV1Beta1Read,
		UpdateContext: resourceKubernetesCSIDriverV1Beta1Update,
		DeleteContext: resourceKubernetesCSIDriverV1Beta1Delete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("csi driver", true),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec of the CSIDriver",
				Optional:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"attach_required": {
							Type:        schema.TypeBool,
							Description: "Indicates if the CSI volume driver requires an attach operation",
							Required:    true,
							ForceNew:    true,
						},
						"pod_info_on_mount": {
							Type:        schema.TypeBool,
							Description: "Indicates that the CSI volume driver requires additional pod information (like podName, podUID, etc.) during mount operations",
							Optional:    true,
						},
						"volume_lifecycle_modes": {
							Type:        schema.TypeList,
							Description: "Defines what kind of volumes this CSI volume driver supports",
							Optional:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
								ValidateFunc: validation.StringInSlice([]string{
									"Persistent",
									"Ephemeral",
								}, false),
							},
						},
					},
				},
			},
		},
	}
}

func resourceKubernetesCSIDriverV1Beta1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	CSIDriver := storage.CSIDriver{
		ObjectMeta: expandMetadata(d.Get("metadata").([]interface{})),
		Spec:       expandCSIDriverSpec(d.Get("spec").([]interface{})),
	}

	log.Printf("[INFO] Creating new CSIDriver: %#v", CSIDriver)
	out, err := conn.StorageV1beta1().CSIDrivers().Create(ctx, &CSIDriver, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Submitted new CSIDriver: %#v", out)
	d.SetId(out.Name)

	return resourceKubernetesCSIDriverV1Beta1Read(ctx, d, meta)
}

func resourceKubernetesCSIDriverV1Beta1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesCSIDriverV1Beta1Exists(ctx, d, meta)
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

	name := d.Id()
	log.Printf("[INFO] Reading CSIDriver %s", name)
	CSIDriver, err := conn.StorageV1beta1().CSIDrivers().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Received CSIDriver: %#v", CSIDriver)
	err = d.Set("metadata", flattenMetadata(CSIDriver.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	spec := flattenCSIDriverSpec(CSIDriver.Spec)

	err = d.Set("spec", spec)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesCSIDriverV1Beta1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Id()
	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("spec") {
		diffOps := patchCSIDriverSpec("spec.0.", "/spec", d)
		ops = append(ops, *diffOps...)
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating CSIDriver %q: %v", name, string(data))
	out, err := conn.StorageV1beta1().CSIDrivers().Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.Errorf("Failed to update CSIDriver: %s", err)
	}
	log.Printf("[INFO] Submitted updated CSIDriver: %#v", out)
	d.SetId(out.ObjectMeta.Name)

	return resourceKubernetesCSIDriverV1Beta1Read(ctx, d, meta)
}

func resourceKubernetesCSIDriverV1Beta1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting CSIDriver: %s", d.Id())
	err = conn.StorageV1beta1().CSIDrivers().Delete(ctx, d.Id(), metav1.DeleteOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutDelete), func() *retry.RetryError {
		_, err := conn.StorageV1beta1().CSIDrivers().Get(ctx, d.Id(), metav1.GetOptions{})
		if err != nil {
			if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
				return nil
			}
			return retry.NonRetryableError(err)
		}

		e := fmt.Errorf("CSIDriver (%s) still exists", d.Id())
		return retry.RetryableError(e)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] CSIDriver %s deleted", d.Id())

	d.SetId("")
	return nil
}

func resourceKubernetesCSIDriverV1Beta1Exists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	name := d.Id()
	log.Printf("[INFO] Checking CSIDriver %s", name)
	_, err = conn.StorageV1beta1().CSIDrivers().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
