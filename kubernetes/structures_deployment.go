package kubernetes

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func flattenDeploymentSpec(in appsv1.DeploymentSpec) ([]interface{}, error) {
	att := make(map[string]interface{})
	att["min_ready_seconds"] = in.MinReadySeconds

	if in.Replicas != nil {
		att["replicas"] = *in.Replicas
	}

	att["selector"] = in.Selector
	podSpec, err := flattenPodSpec(in.Template.Spec)
	if err != nil {
		return nil, err
	}
	att["template"] = podSpec

	return []interface{}{att}, nil
}

func expandDeploymentSpec(deployment []interface{}) (appsv1.DeploymentSpec, error) {
	obj := appsv1.DeploymentSpec{}

	if len(deployment) == 0 || deployment[0] == nil {
		return obj, nil
	}

	in := deployment[0].(map[string]interface{})

	obj.MinReadySeconds = int32(in["min_ready_seconds"].(int))
	obj.Paused = in["paused"].(bool)
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
		ObjectMeta: expandMetadata(in["template"].([]interface{})),
		Spec:       podSpec,
	}

	return obj, nil
}

func expandDeploymentStrategy(l []interface{}) appsv1.DeploymentStrategy {
	if len(l) == 0 || l[0] == nil {
		return appsv1.DeploymentStrategy{}
	}
	in := l[0].(map[string]interface{})
	obj := appsv1.DeploymentStrategy{}
	if v, ok := in["type"].(map[string]interface{}); ok && len(v) > 0 {
		obj.Type = appsv1.DeploymentStrategyType(in["type"].(string))
	}
	if v, ok := in["rolling_update"].([]interface{}); ok && len(v) > 0 {
		obj.RollingUpdate = expandRollingUpdateDeployment(v)
	}
	return obj
}

func expandRollingUpdateDeployment(l []interface{}) *appsv1.RollingUpdateDeployment {
	if len(l) == 0 || l[0] == nil {
		return &appsv1.RollingUpdateDeployment{}
	}

	in := l[0].(map[string]interface{})
	obj := &appsv1.RollingUpdateDeployment{}
	if v, ok := in["max_surge"].(map[string]interface{}); ok && len(v) > 0 {
		maxSurge := intstr.IntOrString{
			Type:   intstr.String,
			StrVal: in["max_surge"].(string),
		}
		obj.MaxSurge = &maxSurge
	}
	if v, ok := in["max_unavailable"].([]interface{}); ok && len(v) > 0 {
		maxUnavailable := intstr.IntOrString{
			Type:   intstr.String,
			StrVal: in["max_unavailable"].(string),
		}
		obj.MaxUnavailable = &maxUnavailable
	}
	return obj
}
