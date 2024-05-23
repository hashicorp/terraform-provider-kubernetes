// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

func expandHorizontalPodAutoscalerV2Spec(in []interface{}) (*autoscalingv2.HorizontalPodAutoscalerSpec, error) {
	if len(in) == 0 || in[0] == nil {
		return nil, fmt.Errorf("failed to expand HorizontalPodAutoscaler.Spec: null or empty input")
	}

	spec := &autoscalingv2.HorizontalPodAutoscalerSpec{}
	m := in[0].(map[string]interface{})

	if v, ok := m["max_replicas"]; ok {
		spec.MaxReplicas = int32(v.(int))
	}

	if v, ok := m["min_replicas"].(int); ok && v > 0 {
		spec.MinReplicas = ptr.To(int32(v))
	}

	if v, ok := m["scale_target_ref"]; ok {
		spec.ScaleTargetRef = expandV2CrossVersionObjectReference(v.([]interface{}))
	}

	if v, ok := m["metric"].([]interface{}); ok {
		spec.Metrics = expandV2Metrics(v)
	}

	if v, ok := m["behavior"].([]interface{}); ok {
		spec.Behavior = expandV2Behavior(v)
	}

	return spec, nil
}

func expandV2Metrics(in []interface{}) []autoscalingv2.MetricSpec {
	metrics := []autoscalingv2.MetricSpec{}

	for _, m := range in {
		metrics = append(metrics, expandV2MetricSpec(m.(map[string]interface{})))
	}

	return metrics
}

func expandV2MetricTarget(m map[string]interface{}) autoscalingv2.MetricTarget {
	target := autoscalingv2.MetricTarget{}

	if v, ok := m["type"].(string); ok {
		target.Type = autoscalingv2.MetricTargetType(v)
	}

	switch target.Type {
	case autoscalingv2.AverageValueMetricType:
		if v, ok := m["average_value"].(string); ok && v != "0" && v != "" {
			q := resource.MustParse(v)
			target.AverageValue = &q
		}
	case autoscalingv2.UtilizationMetricType:
		if v, ok := m["average_utilization"].(int); ok && v > 0 {
			target.AverageUtilization = ptr.To(int32(v))
		}
	case autoscalingv2.ValueMetricType:
		if v, ok := m["value"].(string); ok && v != "0" && v != "" {
			q := resource.MustParse(v)
			target.Value = &q
		}
	}

	return target
}

func expandV2ResourceMetricSource(m map[string]interface{}) *autoscalingv2.ResourceMetricSource {
	source := &autoscalingv2.ResourceMetricSource{}

	if v, ok := m["name"].(string); ok {
		source.Name = corev1.ResourceName(v)
	}

	if v, ok := m["target"].([]interface{}); ok && len(v) == 1 {
		source.Target = expandV2MetricTarget(v[0].(map[string]interface{}))
	}

	return source
}

func expandV2MetricIdentifier(m map[string]interface{}) autoscalingv2.MetricIdentifier {
	identifier := autoscalingv2.MetricIdentifier{}
	identifier.Name = m["name"].(string)

	if v, ok := m["selector"].([]interface{}); ok && len(v) == 1 {
		identifier.Selector = expandLabelSelector(v)
	}

	return identifier
}

func expandV2ExternalMetricSource(m map[string]interface{}) *autoscalingv2.ExternalMetricSource {
	source := &autoscalingv2.ExternalMetricSource{}

	if v, ok := m["metric"].([]interface{}); ok && len(v) == 1 {
		source.Metric = expandV2MetricIdentifier(v[0].(map[string]interface{}))
	}

	if v, ok := m["target"].([]interface{}); ok && len(v) == 1 {
		source.Target = expandV2MetricTarget(v[0].(map[string]interface{}))
	}

	return source
}

func expandV2PodsMetricSource(m map[string]interface{}) *autoscalingv2.PodsMetricSource {
	source := &autoscalingv2.PodsMetricSource{}

	if v, ok := m["metric"].([]interface{}); ok && len(v) == 1 {
		source.Metric = expandV2MetricIdentifier(v[0].(map[string]interface{}))
	}

	if v, ok := m["target"].([]interface{}); ok && len(v) == 1 {
		source.Target = expandV2MetricTarget(v[0].(map[string]interface{}))
	}

	return source
}

func expandV2ObjectMetricSource(m map[string]interface{}) *autoscalingv2.ObjectMetricSource {
	source := &autoscalingv2.ObjectMetricSource{}

	if v, ok := m["described_object"].([]interface{}); ok && len(v) == 1 {
		source.DescribedObject = expandV2CrossVersionObjectReference(v)
	}

	if v, ok := m["metric"].([]interface{}); ok && len(v) == 1 {
		source.Metric = expandV2MetricIdentifier(v[0].(map[string]interface{}))
	}

	if v, ok := m["target"].([]interface{}); ok && len(v) == 1 {
		source.Target = expandV2MetricTarget(v[0].(map[string]interface{}))
	}

	return source
}

func expandV2ContainerResourceMetricSource(m map[string]interface{}) *autoscalingv2.ContainerResourceMetricSource {
	source := &autoscalingv2.ContainerResourceMetricSource{}

	if v, ok := m["container"].(string); ok {
		source.Container = v
	}

	if v, ok := m["name"].(string); ok {
		source.Name = corev1.ResourceName(v)
	}

	if v, ok := m["target"].([]interface{}); ok && len(v) == 1 {
		source.Target = expandV2MetricTarget(v[0].(map[string]interface{}))
	}

	return source
}

func expandV2MetricSpec(m map[string]interface{}) autoscalingv2.MetricSpec {
	spec := autoscalingv2.MetricSpec{}

	if v, ok := m["type"].(string); ok {
		spec.Type = autoscalingv2.MetricSourceType(v)
	}

	if v, ok := m["resource"].([]interface{}); ok && len(v) == 1 {
		spec.Resource = expandV2ResourceMetricSource(v[0].(map[string]interface{}))
	}

	if v, ok := m["external"].([]interface{}); ok && len(v) == 1 {
		spec.External = expandV2ExternalMetricSource(v[0].(map[string]interface{}))
	}

	if v, ok := m["pods"].([]interface{}); ok && len(v) == 1 {
		spec.Pods = expandV2PodsMetricSource(v[0].(map[string]interface{}))
	}

	if v, ok := m["object"].([]interface{}); ok && len(v) == 1 {
		spec.Object = expandV2ObjectMetricSource(v[0].(map[string]interface{}))
	}

	if v, ok := m["container_resource"].([]interface{}); ok && len(v) == 1 {
		spec.ContainerResource = expandV2ContainerResourceMetricSource(v[0].(map[string]interface{}))
	}

	return spec
}

func expandV2Behavior(in []interface{}) *autoscalingv2.HorizontalPodAutoscalerBehavior {
	spec := &autoscalingv2.HorizontalPodAutoscalerBehavior{}

	if len(in) == 0 || in[0] == nil {
		return spec
	}

	b := in[0].(map[string]interface{})

	if v, ok := b["scale_up"].([]interface{}); ok {
		spec.ScaleUp = expandV2ScalingRules(v)
	}

	if v, ok := b["scale_down"].([]interface{}); ok {
		spec.ScaleDown = expandV2ScalingRules(v)
	}

	return spec
}

func expandV2ScalingRules(in []interface{}) *autoscalingv2.HPAScalingRules {
	spec := &autoscalingv2.HPAScalingRules{}

	if len(in) == 0 || in[0] == nil {
		return spec
	}

	r := in[0].(map[string]interface{})

	spec.Policies = expandV2ScalingPolicies(r["policy"].([]interface{}))

	if v, ok := r["select_policy"].(string); ok {
		spec.SelectPolicy = (*autoscalingv2.ScalingPolicySelect)(&v)
	}

	if v, ok := r["stabilization_window_seconds"].(int); ok {
		spec.StabilizationWindowSeconds = ptr.To(int32(v))
	}

	return spec
}

func expandV2ScalingPolicies(in []interface{}) []autoscalingv2.HPAScalingPolicy {
	policies := []autoscalingv2.HPAScalingPolicy{}

	for _, m := range in {
		policies = append(policies, expandV2ScalingPolicy(m.(map[string]interface{})))
	}

	return policies
}

func expandV2ScalingPolicy(in map[string]interface{}) autoscalingv2.HPAScalingPolicy {
	spec := autoscalingv2.HPAScalingPolicy{}

	if v, ok := in["period_seconds"].(int); ok {
		spec.PeriodSeconds = int32(v)
	}

	if v, ok := in["type"].(string); ok {
		spec.Type = autoscalingv2.HPAScalingPolicyType(v)
	}

	if v, ok := in["value"].(int); ok {
		spec.Value = int32(v)
	}

	return spec
}

func expandV2CrossVersionObjectReference(in []interface{}) autoscalingv2.CrossVersionObjectReference {
	ref := autoscalingv2.CrossVersionObjectReference{}

	if len(in) == 0 || in[0] == nil {
		return ref
	}

	m := in[0].(map[string]interface{})

	if v, ok := m["api_version"]; ok {
		ref.APIVersion = v.(string)
	}

	if v, ok := m["kind"]; ok {
		ref.Kind = v.(string)
	}

	if v, ok := m["name"]; ok {
		ref.Name = v.(string)
	}
	return ref
}

func flattenV2MetricTarget(target autoscalingv2.MetricTarget) []interface{} {
	m := map[string]interface{}{
		"type": target.Type,
	}

	switch target.Type {
	case autoscalingv2.AverageValueMetricType:
		m["average_value"] = target.AverageValue.String()
	case autoscalingv2.UtilizationMetricType:
		m["average_utilization"] = *target.AverageUtilization
	case autoscalingv2.ValueMetricType:
		m["value"] = target.Value.String()
	}

	return []interface{}{m}
}

func flattenV2MetricIdentifier(identifier autoscalingv2.MetricIdentifier) []interface{} {
	m := map[string]interface{}{
		"name": identifier.Name,
	}

	if identifier.Selector != nil {
		m["selector"] = flattenLabelSelector(identifier.Selector)
	}

	return []interface{}{m}
}

func flattenV2ExternalMetricSource(external *autoscalingv2.ExternalMetricSource) []interface{} {
	m := map[string]interface{}{
		"metric": flattenV2MetricIdentifier(external.Metric),
		"target": flattenV2MetricTarget(external.Target),
	}
	return []interface{}{m}
}

func flattenV2PodsMetricSource(pods *autoscalingv2.PodsMetricSource) []interface{} {
	m := map[string]interface{}{
		"metric": flattenV2MetricIdentifier(pods.Metric),
		"target": flattenV2MetricTarget(pods.Target),
	}
	return []interface{}{m}
}

func flattenV2ObjectMetricSource(object *autoscalingv2.ObjectMetricSource) []interface{} {
	m := map[string]interface{}{
		"described_object": flattenV2CrossVersionObjectReference(object.DescribedObject),
		"metric":           flattenV2MetricIdentifier(object.Metric),
		"target":           flattenV2MetricTarget(object.Target),
	}
	return []interface{}{m}
}

func flattenV2ContainerResourceMetricSource(cr *autoscalingv2.ContainerResourceMetricSource) []interface{} {
	m := map[string]interface{}{
		"name":      cr.Name.String(),
		"container": cr.Container,
		"target":    flattenV2MetricTarget(cr.Target),
	}
	return []interface{}{m}
}

func flattenV2ResourceMetricSource(resource *autoscalingv2.ResourceMetricSource) []interface{} {
	m := map[string]interface{}{
		"name":   resource.Name,
		"target": flattenV2MetricTarget(resource.Target),
	}
	return []interface{}{m}
}

func flattenV2MetricSpec(spec autoscalingv2.MetricSpec) map[string]interface{} {
	m := map[string]interface{}{}

	m["type"] = spec.Type

	if spec.Resource != nil {
		m["resource"] = flattenV2ResourceMetricSource(spec.Resource)
	}

	if spec.External != nil {
		m["external"] = flattenV2ExternalMetricSource(spec.External)
	}

	if spec.Pods != nil {
		m["pods"] = flattenV2PodsMetricSource(spec.Pods)
	}

	if spec.Object != nil {
		m["object"] = flattenV2ObjectMetricSource(spec.Object)
	}

	if spec.ContainerResource != nil {
		m["container_resource"] = flattenV2ContainerResourceMetricSource(spec.ContainerResource)
	}

	return m
}

func flattenHorizontalPodAutoscalerV2Spec(spec autoscalingv2.HorizontalPodAutoscalerSpec) []interface{} {
	m := make(map[string]interface{}, 0)

	m["max_replicas"] = spec.MaxReplicas

	if spec.MinReplicas != nil {
		m["min_replicas"] = *spec.MinReplicas
	}

	m["scale_target_ref"] = flattenV2CrossVersionObjectReference(spec.ScaleTargetRef)

	metrics := []interface{}{}
	for _, m := range spec.Metrics {
		metrics = append(metrics, flattenV2MetricSpec(m))
	}
	m["metric"] = metrics

	if spec.Behavior != nil {
		m["behavior"] = flattenV2Behavior(*spec.Behavior)
	}

	return []interface{}{m}
}

func flattenV2CrossVersionObjectReference(ref autoscalingv2.CrossVersionObjectReference) []interface{} {
	m := make(map[string]interface{}, 0)

	if ref.APIVersion != "" {
		m["api_version"] = ref.APIVersion
	}

	if ref.Kind != "" {
		m["kind"] = ref.Kind
	}

	if ref.Name != "" {
		m["name"] = ref.Name
	}

	return []interface{}{m}
}

func flattenV2Behavior(spec autoscalingv2.HorizontalPodAutoscalerBehavior) []interface{} {
	b := map[string]interface{}{}

	if spec.ScaleUp != nil {
		b["scale_up"] = flattenV2ScalingRules(*spec.ScaleUp)
	}

	if spec.ScaleDown != nil {
		b["scale_down"] = flattenV2ScalingRules(*spec.ScaleDown)
	}

	return []interface{}{b}
}

func flattenV2ScalingRules(spec autoscalingv2.HPAScalingRules) []interface{} {
	r := map[string]interface{}{}

	if spec.Policies != nil {
		policies := []interface{}{}
		for _, m := range spec.Policies {
			policies = append(policies, flattenV2ScalingPolicy(m))
		}

		r["policy"] = policies
	}

	if spec.SelectPolicy != nil {
		r["select_policy"] = string(*spec.SelectPolicy)
	}

	if spec.StabilizationWindowSeconds != nil {
		r["stabilization_window_seconds"] = int(*spec.StabilizationWindowSeconds)
	}

	return []interface{}{r}
}

func flattenV2ScalingPolicy(spec autoscalingv2.HPAScalingPolicy) map[string]interface{} {
	return map[string]interface{}{
		"type":           string(spec.Type),
		"value":          int(spec.Value),
		"period_seconds": int(spec.PeriodSeconds),
	}
}

func patchHorizontalPodAutoscalerV2Spec(prefix string, pathPrefix string, d *schema.ResourceData) []PatchOperation {
	ops := make([]PatchOperation, 0)

	if d.HasChange(prefix + "max_replicas") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/maxReplicas",
			Value: d.Get(prefix + "max_replicas").(int),
		})
	}

	if d.HasChange(prefix + "min_replicas") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/minReplicas",
			Value: d.Get(prefix + "min_replicas").(int),
		})
	}

	if d.HasChange(prefix + "scale_target_ref") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/scaleTargetRef",
			Value: expandCrossVersionObjectReference(d.Get(prefix + "scale_target_ref").([]interface{})),
		})
	}

	if d.HasChange(prefix + "metric") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/metrics",
			Value: expandV2Metrics(d.Get(prefix + "metric").([]interface{})),
		})
	}

	if d.HasChange(prefix + "behavior") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/behavior",
			Value: expandV2Behavior(d.Get(prefix + "behavior").([]interface{})),
		})
	}

	return ops
}
