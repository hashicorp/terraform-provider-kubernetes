// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

func resourceKubernetesResourceQuotaV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesResourceQuotaV1Create,
		ReadContext:   resourceKubernetesResourceQuotaV1Read,
		UpdateContext: resourceKubernetesResourceQuotaV1Update,
		DeleteContext: resourceKubernetesResourceQuotaV1Delete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(1 * time.Minute),
			Update: schema.DefaultTimeout(1 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("resource quota", true),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec defines the desired quota. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status",
				Optional:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"hard": {
							Type:             schema.TypeMap,
							Description:      "The set of desired hard limits for each named resource. More info: http://releases.k8s.io/HEAD/docs/design/admission_control_resource_quota.md#admissioncontrol-plugin-resourcequota",
							Optional:         true,
							Elem:             schema.TypeString,
							ValidateFunc:     validateResourceList,
							DiffSuppressFunc: suppressEquivalentResourceQuantity,
						},
						"scopes": {
							Type:        schema.TypeSet,
							Description: "A collection of filters that must match each object tracked by a quota. If not specified, the quota matches all objects.",
							Optional:    true,
							ForceNew:    true,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.StringInSlice([]string{"Terminating", "NotTerminating", "BestEffort", "NotBestEffort", "PriorityClass"}, false),
							},
							Set: schema.HashString,
						},
						"scope_selector": {
							Type:        schema.TypeList,
							Description: "A collection of filters like scopes that must match each object tracked by a quota but expressed using ScopeSelectorOperator in combination with possible values. For a resource to match, both scopes AND scopeSelector (if specified in spec), must be matched.",
							Optional:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"match_expression": {
										Type:        schema.TypeList,
										Description: "A list of scope selector requirements by scope of the resources.",
										Optional:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"scope_name": {
													Type:         schema.TypeString,
													Description:  "The name of the scope that the selector applies to.",
													Required:     true,
													ValidateFunc: validation.StringInSlice([]string{"Terminating", "NotTerminating", "BestEffort", "NotBestEffort", "PriorityClass"}, false),
												},
												"operator": {
													Type:         schema.TypeString,
													Description:  "Represents a scope's relationship to a set of values.",
													Required:     true,
													ValidateFunc: validation.StringInSlice([]string{"In", "NotIn", "Exists", "DoesNotExist"}, false),
												},
												"values": {
													Type:        schema.TypeSet,
													Description: "A list of scope selector requirements by scope of the resources.",
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
				},
			},
		},
	}
}

func resourceKubernetesResourceQuotaV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandResourceQuotaSpec(d.Get("spec").([]interface{}))
	if err != nil {
		return diag.FromErr(err)
	}
	resQuota := api.ResourceQuota{
		ObjectMeta: metadata,
		Spec:       *spec,
	}
	log.Printf("[INFO] Creating new resource quota: %#v", resQuota)
	out, err := conn.CoreV1().ResourceQuotas(metadata.Namespace).Create(ctx, &resQuota, metav1.CreateOptions{})
	if err != nil {
		return diag.Errorf("Failed to create resource quota: %s", err)
	}
	log.Printf("[INFO] Submitted new resource quota: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutCreate), func() *retry.RetryError {
		quota, err := conn.CoreV1().ResourceQuotas(out.Namespace).Get(ctx, out.Name, metav1.GetOptions{})
		if err != nil {
			return retry.NonRetryableError(err)
		}
		if resourceListEquals(spec.Hard, quota.Status.Hard) {
			return nil
		}
		err = fmt.Errorf("Quotas don't match after creation.\nExpected: %#v\nGiven: %#v",
			spec.Hard, quota.Status.Hard)
		return retry.RetryableError(err)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceKubernetesResourceQuotaV1Read(ctx, d, meta)
}

func resourceKubernetesResourceQuotaV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesResourceQuotaV1Exists(ctx, d, meta)
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

	log.Printf("[INFO] Reading resource quota %s", name)
	resQuota, err := conn.CoreV1().ResourceQuotas(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Received resource quota: %#v", resQuota)

	// This is to work around K8S bug
	// See https://github.com/kubernetes/kubernetes/issues/44539
	if resQuota.ObjectMeta.GenerateName == "" {
		if v, ok := d.GetOk("metadata.0.generate_name"); ok {
			resQuota.ObjectMeta.GenerateName = v.(string)
		}
	}

	err = d.Set("metadata", flattenMetadata(resQuota.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("spec", flattenResourceQuotaSpec(resQuota.Spec))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesResourceQuotaV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	var spec *api.ResourceQuotaSpec
	waitForChangedSpec := false
	if d.HasChange("spec") {
		spec, err = expandResourceQuotaSpec(d.Get("spec").([]interface{}))
		if err != nil {
			return diag.FromErr(err)
		}
		ops = append(ops, &ReplaceOperation{
			Path:  "/spec",
			Value: *spec,
		})
		waitForChangedSpec = true
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating resource quota %q: %v", name, string(data))
	out, err := conn.CoreV1().ResourceQuotas(namespace).Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.Errorf("Failed to update resource quota: %s", err)
	}
	log.Printf("[INFO] Submitted updated resource quota: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	if waitForChangedSpec {
		err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutUpdate), func() *retry.RetryError {
			quota, err := conn.CoreV1().ResourceQuotas(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return retry.NonRetryableError(err)
			}
			if resourceListEquals(spec.Hard, quota.Status.Hard) {
				return nil
			}
			err = fmt.Errorf("Quotas don't match after update.\nExpected: %#v\nGiven: %#v",
				spec.Hard, quota.Status.Hard)
			return retry.RetryableError(err)
		})
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceKubernetesResourceQuotaV1Read(ctx, d, meta)
}

func resourceKubernetesResourceQuotaV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Deleting resource quota: %#v", name)
	err = conn.CoreV1().ResourceQuotas(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return nil
		}
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Resource quota %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesResourceQuotaV1Exists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking resource quota %s", name)
	_, err = conn.CoreV1().ResourceQuotas(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
