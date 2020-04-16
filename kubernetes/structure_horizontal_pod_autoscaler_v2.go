package kubernetes

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func expandHorizontalPodAutoscalerV2Spec(in []interface{}) (*autoscalingv2beta2.HorizontalPodAutoscalerSpec, error) {
	if len(in) == 0 || in[0] == nil {
		return nil, fmt.Errorf("failed to expand HorizontalPodAutoscaler.Spec: null or empty input")
	}

	spec := &autoscalingv2beta2.HorizontalPodAutoscalerSpec{}
	m := in[0].(map[string]interface{})

	if v, ok := m["max_replicas"]; ok {
		spec.MaxReplicas = int32(v.(int))
	}

	if v, ok := m["min_replicas"].(int); ok && v > 0 {
		spec.MinReplicas = ptrToInt32(int32(v))
	}

	if v, ok := m["scale_target_ref"]; ok {
		spec.ScaleTargetRef = expandV2CrossVersionObjectReference(v.([]interface{}))
	}

	if v, ok := m["metric"].([]interface{}); ok {
		spec.Metrics = expandV2Metrics(v)
	}

	return spec, nil
}

func expandV2Metrics(in []interface{}) []autoscalingv2beta2.MetricSpec {
	metrics := []autoscalingv2beta2.MetricSpec{}

	for _, m := range in {
		metrics = append(metrics, expandV2MetricSpec(m.(map[string]interface{})))
	}

	return metrics
}

func expandV2MetricTarget(m map[string]interface{}) autoscalingv2beta2.MetricTarget {
	target := autoscalingv2beta2.MetricTarget{}

	if v, ok := m["type"].(string); ok {
		target.Type = autoscalingv2beta2.MetricTargetType(v)
	}

	switch target.Type {
	case autoscalingv2beta2.AverageValueMetricType:
		if v, ok := m["average_value"].(string); ok && v != "0" && v != "" {
			q := resource.MustParse(v)
			target.AverageValue = &q
		}
	case autoscalingv2beta2.UtilizationMetricType:
		if v, ok := m["average_utilization"].(int); ok && v > 0 {
			target.AverageUtilization = ptrToInt32(int32(v))
		}
	case autoscalingv2beta2.ValueMetricType:
		if v, ok := m["value"].(string); ok && v != "0" && v != "" {
			q := resource.MustParse(v)
			target.Value = &q
		}
	}

	return target
}

func expandV2ResourceMetricSource(m map[string]interface{}) *autoscalingv2beta2.ResourceMetricSource {
	source := &autoscalingv2beta2.ResourceMetricSource{}

	if v, ok := m["name"].(string); ok {
		source.Name = v1.ResourceName(v)
	}

	if v, ok := m["target"].([]interface{}); ok && len(v) == 1 {
		source.Target = expandV2MetricTarget(v[0].(map[string]interface{}))
	}

	return source
}

func expandV2MetricIdentifier(m map[string]interface{}) autoscalingv2beta2.MetricIdentifier {
	identifier := autoscalingv2beta2.MetricIdentifier{}
	identifier.Name = m["name"].(string)

	if v, ok := m["selector"].([]interface{}); ok && len(v) == 1 {
		identifier.Selector = expandLabelSelector(v)
	}

	return identifier
}

func expandV2ExternalMetricSource(m map[string]interface{}) *autoscalingv2beta2.ExternalMetricSource {
	source := &autoscalingv2beta2.ExternalMetricSource{}

	if v, ok := m["metric"].([]interface{}); ok && len(v) == 1 {
		source.Metric = expandV2MetricIdentifier(v[0].(map[string]interface{}))
	}

	if v, ok := m["target"].([]interface{}); ok && len(v) == 1 {
		source.Target = expandV2MetricTarget(v[0].(map[string]interface{}))
	}

	return source
}

func expandV2PodsMetricSource(m map[string]interface{}) *autoscalingv2beta2.PodsMetricSource {
	source := &autoscalingv2beta2.PodsMetricSource{}

	if v, ok := m["metric"].([]interface{}); ok && len(v) == 1 {
		source.Metric = expandV2MetricIdentifier(v[0].(map[string]interface{}))
	}

	if v, ok := m["target"].([]interface{}); ok && len(v) == 1 {
		source.Target = expandV2MetricTarget(v[0].(map[string]interface{}))
	}

	return source
}

func expandV2ObjectMetricSource(m map[string]interface{}) *autoscalingv2beta2.ObjectMetricSource {
	source := &autoscalingv2beta2.ObjectMetricSource{}

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

func expandV2MetricSpec(m map[string]interface{}) autoscalingv2beta2.MetricSpec {
	spec := autoscalingv2beta2.MetricSpec{}

	if v, ok := m["type"].(string); ok {
		spec.Type = autoscalingv2beta2.MetricSourceType(v)
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

	return spec
}

func expandV2CrossVersionObjectReference(in []interface{}) autoscalingv2beta2.CrossVersionObjectReference {
	ref := autoscalingv2beta2.CrossVersionObjectReference{}

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

func flattenV2MetricTarget(target autoscalingv2beta2.MetricTarget) []interface{} {
	m := map[string]interface{}{
		"type": target.Type,
	}

	switch target.Type {
	case autoscalingv2beta2.AverageValueMetricType:
		m["average_value"] = target.AverageValue.String()
	case autoscalingv2beta2.UtilizationMetricType:
		m["average_utilization"] = *target.AverageUtilization
	case autoscalingv2beta2.ValueMetricType:
		m["value"] = target.Value.String()
	}

	return []interface{}{m}
}

func flattenV2MetricIdentifier(identifier autoscalingv2beta2.MetricIdentifier) []interface{} {
	m := map[string]interface{}{
		"name": identifier.Name,
	}

	if identifier.Selector != nil {
		m["selector"] = flattenLabelSelector(identifier.Selector)
	}

	return []interface{}{m}
}

func flattenV2ExternalMetricSource(external *autoscalingv2beta2.ExternalMetricSource) []interface{} {
	m := map[string]interface{}{
		"metric": flattenV2MetricIdentifier(external.Metric),
		"target": flattenV2MetricTarget(external.Target),
	}
	return []interface{}{m}
}

func flattenV2PodsMetricSource(pods *autoscalingv2beta2.PodsMetricSource) []interface{} {
	m := map[string]interface{}{
		"metric": flattenV2MetricIdentifier(pods.Metric),
		"target": flattenV2MetricTarget(pods.Target),
	}
	return []interface{}{m}
}

func flattenV2ObjectMetricSource(object *autoscalingv2beta2.ObjectMetricSource) []interface{} {
	m := map[string]interface{}{
		"described_object": flattenV2CrossVersionObjectReference(object.DescribedObject),
		"metric":           flattenV2MetricIdentifier(object.Metric),
		"target":           flattenV2MetricTarget(object.Target),
	}
	return []interface{}{m}
}

func flattenV2ResourceMetricSource(resource *autoscalingv2beta2.ResourceMetricSource) []interface{} {
	m := map[string]interface{}{
		"name":   resource.Name,
		"target": flattenV2MetricTarget(resource.Target),
	}
	return []interface{}{m}
}

func flattenV2MetricSpec(spec autoscalingv2beta2.MetricSpec) map[string]interface{} {
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

	return m
}

func flattenHorizontalPodAutoscalerV2Spec(spec autoscalingv2beta2.HorizontalPodAutoscalerSpec) []interface{} {
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

	return []interface{}{m}
}

func flattenV2CrossVersionObjectReference(ref autoscalingv2beta2.CrossVersionObjectReference) []interface{} {
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

	return ops
}
