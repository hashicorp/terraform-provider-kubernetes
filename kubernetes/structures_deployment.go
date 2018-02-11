package kubernetes

import (
	corev1 "k8s.io/client-go/pkg/api/v1"
	v1 "k8s.io/client-go/pkg/apis/apps/v1beta1"
)

func flattenDeploymentStrategy(in v1.DeploymentStrategy) []interface{} {
	att := make(map[string]interface{})

	// TODO: These need to be flattened further
	att["rolling_update"] = in.RollingUpdate
	att["type"] = in.Type

	return []interface{}{att}
}

func flattenDeploymentSpec(in v1.DeploymentSpec) ([]interface{}, error) {
	att := make(map[string]interface{})
	att["min_ready_seconds"] = in.MinReadySeconds
	att["paused"] = in.Paused
	if in.ProgressDeadlineSeconds != nil {
		att["progress_deadline_seconds"] = *in.ProgressDeadlineSeconds
	}
	if in.Replicas != nil {
		att["replicas"] = *in.Replicas
	}
	if in.RevisionHistoryLimit != nil {
		att["revision_history_limit"] = *in.RevisionHistoryLimit
	}
	if in.Selector != nil {
		att["selector"] = flattenLabelSelector(in.Selector)
	}
	att["strategy"] = flattenDeploymentStrategy(in.Strategy)
	podSpec, err := flattenPodSpec(in.Template.Spec)
	if err != nil {
		return nil, err
	}
	att["template"] = podSpec

	return []interface{}{att}, nil
}

func expandDeploymentStrategy(in []interface{}) v1.DeploymentStrategy {
	obj := v1.DeploymentStrategy{}
	// TODO: expand the strategy
	return obj
}

func expandDeploymentSpec(d []interface{}) (v1.DeploymentSpec, error) {
	obj := v1.DeploymentSpec{}
	if len(d) == 0 || d[0] == nil {
		return obj, nil
	}
	in := d[0].(map[string]interface{})
	obj.MinReadySeconds = int32(in["min_ready_seconds"].(int))
	if v, ok := in["paused"]; ok {
		obj.Paused = v.(bool)
	}
	obj.ProgressDeadlineSeconds = ptrToInt32(int32(in["progress_deadline_seconds"].(int)))
	obj.Replicas = ptrToInt32(int32(in["replicas"].(int)))
	obj.RevisionHistoryLimit = ptrToInt32(int32(in["revision_history_limit"].(int)))
	if v, ok := in["selector"].([]interface{}); ok && len(v) > 0 {
		obj.Selector = expandLabelSelector(v)
	}
	if v, ok := in["strategy"].([]interface{}); ok && len(v) > 0 {
		obj.Strategy = expandDeploymentStrategy(v)
	}
	podSpec, err := expandPodSpec(in["template"].([]interface{}))
	if err != nil {
		return obj, err
	}
	obj.Template = corev1.PodTemplateSpec{
		Spec: podSpec,
	}

	return obj, nil
}
