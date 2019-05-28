package kubernetes

import (
	"github.com/hashicorp/terraform/helper/schema"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func flattenDaemonSetSpec(in appsv1.DaemonSetSpec, d *schema.ResourceData) ([]interface{}, error) {
	att := make(map[string]interface{})
	att["min_ready_seconds"] = in.MinReadySeconds

	att["strategy"] = flattenDaemonSetStrategy(in.UpdateStrategy)

	if in.RevisionHistoryLimit != nil {
		att["revision_history_limit"] = *in.RevisionHistoryLimit
	}

	if in.Selector != nil {
		att["selector"] = flattenLabelSelector(in.Selector)
	}

	podSpec, err := flattenPodSpec(in.Template.Spec)
	if err != nil {
		return nil, err
	}
	template := make(map[string]interface{})
	template["spec"] = podSpec
	template["metadata"] = flattenMetadata(in.Template.ObjectMeta, d, "spec.0.template.0.")
	att["template"] = []interface{}{template}

	return []interface{}{att}, nil
}

func flattenDaemonSetStrategy(in appsv1.DaemonSetUpdateStrategy) []interface{} {
	att := make(map[string]interface{})
	if in.Type != "" {
		att["type"] = in.Type
	}
	if in.RollingUpdate != nil {
		att["rolling_update"] = flattenDaemonSetStrategyRollingUpdate(in.RollingUpdate)
	}
	return []interface{}{att}
}

func flattenDaemonSetStrategyRollingUpdate(in *appsv1.RollingUpdateDaemonSet) []interface{} {
	att := make(map[string]interface{})
	if in.MaxUnavailable != nil {
		att["max_unavailable"] = in.MaxUnavailable.String()
	}
	return []interface{}{att}
}

func expandDaemonSetSpec(daemonset []interface{}) (appsv1.DaemonSetSpec, error) {
	obj := appsv1.DaemonSetSpec{}

	if len(daemonset) == 0 || daemonset[0] == nil {
		return obj, nil
	}

	in := daemonset[0].(map[string]interface{})

	obj.MinReadySeconds = int32(in["min_ready_seconds"].(int))
	obj.RevisionHistoryLimit = ptrToInt32(int32(in["revision_history_limit"].(int)))

	if v, ok := in["selector"].([]interface{}); ok && len(v) > 0 {
		obj.Selector = expandLabelSelector(v)
	}

	if v, ok := in["strategy"].([]interface{}); ok && len(v) > 0 {
		obj.UpdateStrategy = expandDaemonSetStrategy(v)
	}

	template, err := expandPodTemplate(in["template"].([]interface{}))
	if err != nil {
		return obj, err
	}
	obj.Template = *template

	return obj, nil
}

func expandDaemonSetStrategy(p []interface{}) appsv1.DaemonSetUpdateStrategy {
	obj := appsv1.DaemonSetUpdateStrategy{}

	if len(p) == 0 || p[0] == nil {
		obj.Type = appsv1.RollingUpdateDaemonSetStrategyType
		return obj
	}
	in := p[0].(map[string]interface{})

	if v, ok := in["type"].(string); ok {
		obj.Type = appsv1.DaemonSetUpdateStrategyType(v)
	}
	if v, ok := in["rolling_update"].([]interface{}); ok && len(v) > 0 {
		obj.RollingUpdate = expandRollingUpdateDaemonSet(v)
	}
	return obj
}

func expandRollingUpdateDaemonSet(p []interface{}) *appsv1.RollingUpdateDaemonSet {
	obj := appsv1.RollingUpdateDaemonSet{}
	if len(p) == 0 || p[0] == nil {
		return nil
	}
	in := p[0].(map[string]interface{})

	if v, ok := in["max_unavailable"].(string); ok {
		val := intstr.Parse(v)
		obj.MaxUnavailable = &val
	}
	return &obj
}
