// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

func resourceKubernetesHorizontalPodAutoscalerV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesHorizontalPodAutoscalerV1Create,
		ReadContext:   resourceKubernetesHorizontalPodAutoscalerV1Read,
		UpdateContext: resourceKubernetesHorizontalPodAutoscalerV1Update,
		DeleteContext: resourceKubernetesHorizontalPodAutoscalerV1Delete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("horizontal pod autoscaler", true),
			"spec": {
				Type:        schema.TypeList,
				Description: "Behaviour of the autoscaler. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status",
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"max_replicas": {
							Type:        schema.TypeInt,
							Description: "Upper limit for the number of pods that can be set by the autoscaler.",
							Required:    true,
						},
						"min_replicas": {
							Type:        schema.TypeInt,
							Description: "Lower limit for the number of pods that can be set by the autoscaler, defaults to `1`.",
							Optional:    true,
							Default:     1,
						},
						"scale_target_ref": {
							Type:        schema.TypeList,
							Description: "Reference to scaled resource. e.g. Replication Controller",
							Required:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"api_version": {
										Type:        schema.TypeString,
										Description: "API version of the referent",
										Optional:    true,
									},
									"kind": {
										Type:        schema.TypeString,
										Description: "Kind of the referent. e.g. `ReplicationController`. More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#types-kinds",
										Required:    true,
									},
									"name": {
										Type:        schema.TypeString,
										Description: "Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
										Required:    true,
									},
								},
							},
						},
						"target_cpu_utilization_percentage": {
							Type:        schema.TypeInt,
							Description: "Target average CPU utilization (represented as a percentage of requested CPU) over all the pods. If not specified the default autoscaling policy will be used.",
							Optional:    true,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func resourceKubernetesHorizontalPodAutoscalerV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandHorizontalPodAutoscalerSpec(d.Get("spec").([]interface{}))
	if err != nil {
		return diag.FromErr(err)
	}

	hpa := autoscalingv1.HorizontalPodAutoscaler{
		ObjectMeta: metadata,
		Spec:       *spec,
	}
	log.Printf("[INFO] Creating new horizontal pod autoscaler: %#v", hpa)
	out, err := conn.AutoscalingV1().HorizontalPodAutoscalers(metadata.Namespace).Create(ctx, &hpa, metav1.CreateOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Submitted new horizontal pod autoscaler: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesHorizontalPodAutoscalerV1Read(ctx, d, meta)
}

func resourceKubernetesHorizontalPodAutoscalerV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesHorizontalPodAutoscalerV1Exists(ctx, d, meta)
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
	log.Printf("[INFO] Reading horizontal pod autoscaler %s", name)
	hpa, err := conn.AutoscalingV1().HorizontalPodAutoscalers(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Received horizontal pod autoscaler: %#v", hpa)
	err = d.Set("metadata", flattenMetadata(hpa.ObjectMeta, d, meta))
	if err != nil {
		return diag.FromErr(err)
	}

	flattened := flattenHorizontalPodAutoscalerSpec(hpa.Spec)
	log.Printf("[DEBUG] Flattened horizontal pod autoscaler spec: %#v", flattened)
	err = d.Set("spec", flattened)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKubernetesHorizontalPodAutoscalerV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
		diffOps := patchHorizontalPodAutoscalerSpec("spec.0.", "/spec", d)
		ops = append(ops, diffOps...)
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return diag.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating horizontal pod autoscaler %q: %v", name, string(data))
	out, err := conn.AutoscalingV1().HorizontalPodAutoscalers(namespace).Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return diag.Errorf("Failed to update horizontal pod autoscaler: %s", err)
	}
	log.Printf("[INFO] Submitted updated horizontal pod autoscaler: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesHorizontalPodAutoscalerV1Read(ctx, d, meta)
}

func resourceKubernetesHorizontalPodAutoscalerV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Deleting horizontal pod autoscaler: %#v", name)
	err = conn.AutoscalingV1().HorizontalPodAutoscalers(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return nil
		}
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Horizontal Pod Autoscaler %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesHorizontalPodAutoscalerV1Exists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}

	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Checking horizontal pod autoscaler %s", name)
	_, err = conn.AutoscalingV1().HorizontalPodAutoscalers(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
