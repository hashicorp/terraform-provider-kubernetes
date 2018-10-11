package kubernetes

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Expanders

func expandStatefulSetSpec(s []interface{}) (v1.StatefulSetSpec, error) {
	obj := v1.StatefulSetSpec{}
	if len(s) == 0 || s[0] == nil {
		return obj, nil
	}
	in := s[0].(map[string]interface{})

	if v, ok := in["pod_management_policy"].(string); ok {
		obj.PodManagementPolicy = v1.PodManagementPolicyType(v)
	}

	if v, ok := in["replicas"].(int32); ok && v > 0 {
		obj.Replicas = ptrToInt32(int32(v))
	}

	if v, ok := in["revision_history_limit"].(int32); ok {
		obj.RevisionHistoryLimit = ptrToInt32(int32(v))
	}

	if v, ok := in["selector"].([]interface{}); ok && len(v) > 0 {
		obj.Selector = expandLabelSelector(v)
	}

	if v, ok := in["service_name"].(string); ok {
		obj.ServiceName = v
	}

	if v, ok := in["update_strategy"].(map[string]interface{}); ok {
		ust := v1.StatefulSetUpdateStrategy{}
		ust.Type = v["type"].(v1.StatefulSetUpdateStrategyType)
		if r, ok := v["rolling_update"].(map[string]interface{}); ok {
			s := v1.RollingUpdateStatefulSetStrategy{}
			if p, ok := r["partition"].(int32); ok {
				s.Partition = ptrToInt32(int32(p))
			}
			ust.RollingUpdate = &s
		}
		obj.UpdateStrategy = ust
	}

	template, err := expandPodTemplate(in["template"].([]interface{}))
	if err != nil {
		return obj, err
	}
	obj.Template = template

	if v, ok := in["volume_claim_template"].([]interface{}); ok {
		obj.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{}
		if len(v) == 0 || v[0] == nil {
			return obj, nil
		}
		for _, pvc := range v {
			p, err := expandPersistenVolumeClaim(pvc.([]interface{}))
			if err != nil {
				return obj, err
			}
			obj.VolumeClaimTemplates = append(obj.VolumeClaimTemplates, p)
		}
	}
	return obj, nil
}

func expandPersistenVolumeClaim(p []interface{}) (corev1.PersistentVolumeClaim, error) {
	if len(p) == 0 || p[0] == nil {
		return corev1.PersistentVolumeClaim{}, nil
	}
	in := p[0].(map[string]interface{})
	metadata := expandMetadata(in["metadata"].([]interface{}))
	spec, err := expandPersistentVolumeClaimSpec(in["spec"].([]interface{}))
	if err != nil {
		return corev1.PersistentVolumeClaim{}, err
	}
	pvc := corev1.PersistentVolumeClaim{
		ObjectMeta: metadata,
		Spec:       spec,
	}
	return pvc, nil
}

func expandStatefulSetSelectors(s []interface{}) (metav1.LabelSelector, error) {
	obj := metav1.LabelSelector{}
	if len(s) == 0 || s[0] == nil {
		return obj, nil
	}
	in := s[0].(map[string]interface{})
	log.Printf("[DEBUG] StatefulSet Selector: %#v", in)
	if v, ok := in["match_labels"].(map[string]interface{}); ok {
		log.Printf("[DEBUG] StatefulSet Selector MatchLabels: %#v", v)
		ml := make(map[string]string)
		for k, l := range v {
			ml[k] = l.(string)
			log.Printf("[DEBUG] StatefulSet Selector MatchLabel: %#v -> %#v", k, v)
		}
		obj.MatchLabels = ml
	}
	if v, ok := in["match_expressions"].([]interface{}); ok {
		log.Printf("[DEBUG] StatefulSet Selector MatchExpressions: %#v", v)
		me, err := expandMatchExpressions(v)
		if err != nil {
			return obj, err
		}
		obj.MatchExpressions = me
	}
	return obj, nil
}

func expandMatchExpressions(in []interface{}) ([]metav1.LabelSelectorRequirement, error) {
	if len(in) == 0 {
		return []metav1.LabelSelectorRequirement{}, nil
	}
	obj := make([]metav1.LabelSelectorRequirement, len(in))
	for i, c := range in {
		p := c.(map[string]interface{})
		if v, ok := p["key"].(string); ok {
			obj[i].Key = v
		}
		if v, ok := p["operator"].(metav1.LabelSelectorOperator); ok {
			obj[i].Operator = v
		}
		if v, ok := p["values"].(*schema.Set); ok {
			obj[i].Values = schemaSetToStringArray(v)
		}
	}
	return obj, nil
}

// Flattners

func flattenStatefulSetSpec(spec v1.StatefulSetSpec) ([]interface{}, error) {
	att := make(map[string]interface{})

	if spec.PodManagementPolicy != "" {
		att["pod_management_policy"] = spec.PodManagementPolicy
	}
	if spec.Replicas != nil {
		att["replicas"] = *spec.Replicas
	}
	if spec.RevisionHistoryLimit != nil {
		att["revision_history_limit"] = *spec.RevisionHistoryLimit
	}
	if spec.Selector != nil {
		att["selector"] = flattenLabelSelector(spec.Selector)
	}
	if spec.ServiceName != "" {
		att["service_name"] = spec.ServiceName
	}
	template, err := flattenPodTemplateSpec(spec.Template)
	if err != nil {
		return []interface{}{att}, err
	}
	att["template"] = template
	att["volume_claim_template"] = flattenPersistentVolumeClaim(spec.VolumeClaimTemplates)

	return []interface{}{att}, nil
}

func flattenPodTemplateSpec(t corev1.PodTemplateSpec) ([]interface{}, error) {
	template := make(map[string]interface{})

	template["metadata"] = flattenMetadata(t.ObjectMeta)
	spec, err := flattenPodSpec(t.Spec)
	if err != nil {
		return []interface{}{template}, err
	}
	template["spec"] = spec

	return []interface{}{template}, nil
}

func flattenPersistentVolumeClaim(in []corev1.PersistentVolumeClaim) []interface{} {
	pvcs := make([]interface{}, len(in))

	for _, pvc := range in {
		p := make(map[string]interface{})
		p["metadata"] = flattenMetadata(pvc.ObjectMeta)
		p["spec"] = flattenPersistentVolumeClaimSpec(pvc.Spec)
		pvcs = append(pvcs, p)
	}
	return pvcs
}
