package kubernetes

import (
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

func flattenDeploymentSpec(in v1beta1.DeploymentSpec, d *schema.ResourceData) ([]interface{}, error) {
	att := make(map[string]interface{})

	att["min_ready_seconds"] = in.MinReadySeconds
	att["paused"] = in.Paused
	if in.ProgressDeadlineSeconds != nil {
		att["progress_deadline_seconds"] = int(*in.ProgressDeadlineSeconds)
	} else {
		// nil pointer means the this is set to the default value (600)
		att["progress_deadline_seconds"] = 600
	}

	if in.Replicas != nil {
		att["replicas"] = *in.Replicas
	}

	if in.RevisionHistoryLimit != nil {
		att["revision_history_limit"] = *in.RevisionHistoryLimit
	} else {
		// nil pointer means the this is set to the default value (10)
		att["revision_history_limit"] = 10
	}

	att["selector"] = in.Selector.MatchLabels
	att["strategy"] = flattenDeploymentStrategy(in.Strategy)

	templateMetadata := flattenMetadata(in.Template.ObjectMeta, d)
	podSpec, err := flattenPodSpec(in.Template.Spec)
	if err != nil {
		return nil, err
	}
	template := make(map[string]interface{})
	template["metadata"] = templateMetadata
	template["spec"] = podSpec
	att["template"] = []interface{}{template}

	return []interface{}{att}, nil
}

func flattenDeploymentStrategy(in v1beta1.DeploymentStrategy) []interface{} {
	att := make(map[string]interface{})
	if in.Type != "" {
		att["type"] = in.Type
	}
	if in.RollingUpdate != nil {
		att["rolling_update"] = flattenDeploymentStrategyRollingUpdate(in.RollingUpdate)
	}
	return []interface{}{att}
}

func flattenDeploymentStrategyRollingUpdate(in *v1beta1.RollingUpdateDeployment) []interface{} {
	att := make(map[string]interface{})
	if in.MaxSurge != nil {
		att["max_surge"] = in.MaxSurge.String()
	}
	if in.MaxUnavailable != nil {
		att["max_unavailable"] = in.MaxUnavailable.String()
	}

	return []interface{}{att}
}

func expandDeploymentSpec(deployment []interface{}) (v1beta1.DeploymentSpec, error) {
	obj := v1beta1.DeploymentSpec{}
	if len(deployment) == 0 || deployment[0] == nil {
		return obj, nil
	}
	in := deployment[0].(map[string]interface{})

	obj.MinReadySeconds = int32(in["min_ready_seconds"].(int))
	obj.Paused = in["paused"].(bool)
	if in["progress_deadline_seconds"].(int) > 0 {
		obj.ProgressDeadlineSeconds = ptrToInt32(int32(in["progress_deadline_seconds"].(int)))
	}
	obj.Replicas = ptrToInt32(int32(in["replicas"].(int)))

	if in["revision_history_limit"] != nil {
		obj.RevisionHistoryLimit = ptrToInt32(int32(in["revision_history_limit"].(int)))
	}

	obj.Selector = &metav1.LabelSelector{
		MatchLabels: expandStringMap(in["selector"].(map[string]interface{})),
	}
	obj.Strategy = expandDeploymentStrategy(in["strategy"].([]interface{}))

	for _, v := range in["template"].([]interface{}) {
		template := v.(map[string]interface{})
		podSpec, err := expandPodSpec(template["spec"].([]interface{}))
		if err != nil {
			return obj, err
		}
		obj.Template = v1.PodTemplateSpec{
			Spec: podSpec,
		}

		if metaCfg, ok := template["metadata"]; ok {
			metadata := expandMetadata(metaCfg.([]interface{}))
			obj.Template.ObjectMeta = metadata
		}
	}

	return obj, nil
}

func expandDeploymentStrategy(p []interface{}) v1beta1.DeploymentStrategy {
	obj := v1beta1.DeploymentStrategy{}
	if len(p) == 0 || p[0] == nil {
		return obj
	}
	in := p[0].(map[string]interface{})

	if v, ok := in["type"]; ok {
		obj.Type = v1beta1.DeploymentStrategyType(v.(string))
	}
	if v, ok := in["rolling_update"]; ok {
		obj.RollingUpdate = expandRollingUpdateDeployment(v.([]interface{}))
	}
	return obj
}

func expandRollingUpdateDeployment(p []interface{}) *v1beta1.RollingUpdateDeployment {
	obj := v1beta1.RollingUpdateDeployment{}
	if len(p) == 0 || p[0] == nil {
		return &obj
	}
	in := p[0].(map[string]interface{})

	if v, ok := in["max_surge"]; ok {
		obj.MaxSurge = expandRollingUpdateDeploymentIntOrString(v.(string))
	}
	if v, ok := in["max_unavailable"]; ok {
		obj.MaxUnavailable = expandRollingUpdateDeploymentIntOrString(v.(string))
	}
	return &obj
}

func expandRollingUpdateDeploymentIntOrString(v string) *intstr.IntOrString {
	i, err := strconv.Atoi(v)
	if err != nil {
		return &intstr.IntOrString{
			Type:   intstr.String,
			StrVal: v,
		}
	}
	return &intstr.IntOrString{
		Type:   intstr.Int,
		IntVal: int32(i),
	}
}
