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

func resourceKubernetesHorizontalPodAutoscaler() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceKubernetesHorizontalPodAutoscalerCreate,
		ReadContext:   resourceKubernetesHorizontalPodAutoscalerRead,
		UpdateContext: resourceKubernetesHorizontalPodAutoscalerUpdate,
		DeleteContext: resourceKubernetesHorizontalPodAutoscalerDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: horizontalPodAutoscalerSchemaV2(),
	}
}

func resourceKubernetesHorizontalPodAutoscalerCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if useV2Beta2(d) {
		return resourceKubernetesHorizontalPodAutoscalerV2Beta2Create(ctx, d, meta)
	}

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

	return resourceKubernetesHorizontalPodAutoscalerRead(ctx, d, meta)
}

func resourceKubernetesHorizontalPodAutoscalerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	exists, err := resourceKubernetesHorizontalPodAutoscalerExists(ctx, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		d.SetId("")
		return diag.Diagnostics{}
	}
	if useV2Beta2(d) {
		return resourceKubernetesHorizontalPodAutoscalerV2Beta2Read(ctx, d, meta)
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

	// NOTE: this is needed for import
	if _, exists := hpa.ObjectMeta.GetAnnotations()["autoscaling.alpha.kubernetes.io/metrics"]; exists {
		return resourceKubernetesHorizontalPodAutoscalerV2Beta2Read(ctx, d, meta)
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

func resourceKubernetesHorizontalPodAutoscalerUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if useV2Beta2(d) {
		return resourceKubernetesHorizontalPodAutoscalerV2Beta2Update(ctx, d, meta)
	}

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

	return resourceKubernetesHorizontalPodAutoscalerRead(ctx, d, meta)
}

func resourceKubernetesHorizontalPodAutoscalerDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if useV2Beta2(d) {
		return resourceKubernetesHorizontalPodAutoscalerV2Beta2Delete(ctx, d, meta)
	}

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

func resourceKubernetesHorizontalPodAutoscalerExists(ctx context.Context, d *schema.ResourceData, meta interface{}) (bool, error) {
	if useV2Beta2(d) {
		return resourceKubernetesHorizontalPodAutoscalerV2Beta2Exists(ctx, d, meta)
	}

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

func useV2Beta2(d *schema.ResourceData) bool {
	if len(d.Get("spec.0.metric").([]interface{})) > 0 {
		log.Printf("[INFO] Using autoscaling/v2beta2 because this resource has a metric field")
		return true
	}

	if len(d.Get("spec.0.behavior").([]interface{})) > 0 {
		log.Printf("[INFO] Using autoscaling/v2beta2 because this resource has a behavior field")
		return true
	}

	return false
}
