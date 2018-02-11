package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
)

func flattenDeploymentStrategy(in v1.DeploymentStrategy) []interface{} {

}

func flattenDeploymentSpec(in v1.DeploymentSpec) ([]interface{}, error) {
	att := make(map[string]interface{})
	if in.MinReadySeconds != nill {
		att["min_ready_seconds"] = *in.MinReadySeconds
	}
	att["paused"] = in.Paused
	if in.ProgressDeadlineSeconds != nill {
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
	if in.Strategy != nil {
		att["strategy"] = flattenStrategy(in.Strategy)
	}
	podSpec, err := flattenPodSpec(in.Template.Spec)
	if err != nil {
		return nil, err
	}
	att["template"] = podSpec

	return []interface{}{att}, nil
}

func expandDeploymentSpec(rc []interface{}) (v1.DeploymentSpec, error) {
	obj := v1.DeploymentSpec{}
	if len(rc) == 0 || rc[0] == nil {
		return obj, nil
	}
	in := rc[0].(map[string]interface{})
	obj.MinReadySeconds = int32(in["min_ready_seconds"].(int))
	obj.Replicas = ptrToInt32(int32(in["replicas"].(int)))
	obj.Selector = expandStringMap(in["selector"].(map[string]interface{}))
	podSpec, err := expandPodSpec(in["template"].([]interface{}))
	if err != nil {
		return obj, err
	}
	obj.Template = &v1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: obj.Selector,
		},
		Spec: podSpec,
	}

	return obj, nil
}
