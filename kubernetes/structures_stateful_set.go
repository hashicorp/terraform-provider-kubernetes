package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/apis/apps/v1beta1"
)

func flattenStatefulSetSpec(in v1beta1.StatefulSetSpec) ([]interface{}, error) {
	att := make(map[string]interface{})

	if in.Replicas != nil {
		att["replicas"] = *in.Replicas
	}
	att["service_name"] = in.ServiceName
	att["selector"] = in.Selector.MatchLabels

	podSpec, err := flattenPodSpec(in.Template.Spec)
	if err != nil {
		return nil, err
	}
	att["template"] = podSpec

	return []interface{}{att}, nil
}

func expandStatefulSetSpec(statefulSet []interface{}) (v1beta1.StatefulSetSpec, error) {
	obj := v1beta1.StatefulSetSpec{}
	if len(statefulSet) == 0 || statefulSet[0] == nil {
		return obj, nil
	}
	in := statefulSet[0].(map[string]interface{})

	obj.Replicas = ptrToInt32(int32(in["replicas"].(int)))
	obj.Selector = &metav1.LabelSelector{
		MatchLabels: expandStringMap(in["selector"].(map[string]interface{})),
	}
	obj.ServiceName = in["service_name"].(string)

	podSpec, err := expandPodSpec(in["template"].([]interface{}))
	if err != nil {
		return obj, err
	}
	obj.Template = v1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: obj.Selector.MatchLabels,
		},
		Spec: podSpec,
	}

	return obj, nil
}
