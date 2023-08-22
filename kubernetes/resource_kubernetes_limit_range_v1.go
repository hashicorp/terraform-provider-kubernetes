// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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

func resourceKubernetesLimitRangeV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesLimitRangeV1Create,
		ReadContext:   resourceKubernetesLimitRangeV1Read,
		UpdateContext: resourceKubernetesLimitRangeV1Update,
		DeleteContext: resourceKubernetesLimitRangeV1Delete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("limit range", true),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec defines the limits enforced. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status",
				Optional:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"limit": {
							Type:        schema.TypeList,
							Description: "Limits is the list of objects that are enforced.",
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"default": {
										Type:        schema.TypeMap,
										Description: "Default resource requirement limit value by resource name if resource limit is omitted.",
										Optional:    true,
									},
									"default_request": {
										Type:        schema.TypeMap,
										Description: "The default resource requirement request value by resource name if resource request is omitted.",
										Optional:    true,
										Computed:    true,
									},
									"max": {
										Type:        schema.TypeMap,
										Description: "Max usage constraints on this kind by resource name.",
										Optional:    true,
									},
									"max_limit_request_ratio": {
										Type:        schema.TypeMap,
										Description: "The named resource must have a request and limit that are both non-zero where limit divided by request is less than or equal to the enumerated value; this represents the max burst for the named resource.",
										Optional:    true,
									},
									"min": {
										Type:        schema.TypeMap,
										Description: "Min usage constraints on this kind by resource name.",
										Optional:    true,
									},
									"type": {
										Type:        schema.TypeString,
										Description: "Type of resource that this limit applies to.",
										Optional:    true,
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

func resourceKubernetesLimitRangeV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandLimitRangeSpec(d.Get("spec").([]interface{}), d.IsNewResource())
	if err != nil {
		return diag.FromErr(err)
	}
	limitRange := api.LimitRange{
		ObjectMeta: metadata,
		Spec:       *spec,
	}
	log.Printf("[INFO] Creating new limit range: %#v", limitRange)
	out, err := conn.CoreV1().LimitRanges(metadata.Namespace).Create(ctx, &limitRange, metav1.CreateOptions{})
	if err != nil {
		return diag.Errorf("Failed to create limit range: %s", err)
	}
	log.Printf("[INFO] Submitted new limit range: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesLimitRangeV1Read(ctx, d, meta)
}

func resourceKubernetesLimitRangeV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesLimitRangeV1Exists(ctx, d, meta)
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
	log.Printf("[INFO] Reading limit range %s", name)
	limitRange, err := conn.CoreV1().LimitRanges(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Received limit range: %#v", limitRange)

	err = d.Set("metadata", flattenMetadata(limitRange.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("spec", flattenLimitRangeSpec(limitRange.Spec))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesLimitRangeV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("spec") {
		spec, err := expandLimitRangeSpec(d.Get("spec").([]interface{}), d.IsNewResource())
		if err != nil {
			return diag.FromErr(err)
		}
		ops = append(ops, &ReplaceOperation{
			Path:  "/spec",
			Value: spec,
		})
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating limit range %q: %v", name, string(data))
	out, err := conn.CoreV1().LimitRanges(namespace).Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.Errorf("Failed to update limit range: %s", err)
	}
	log.Printf("[INFO] Submitted updated limit range: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesLimitRangeV1Read(ctx, d, meta)
}

func resourceKubernetesLimitRangeV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting limit range: %#v", name)
	err = conn.CoreV1().LimitRanges(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Limit range %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesLimitRangeV1Exists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking limit range %s", name)
	_, err = conn.CoreV1().LimitRanges(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
