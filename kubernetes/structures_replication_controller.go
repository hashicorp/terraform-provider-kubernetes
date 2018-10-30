package kubernetes

import (
	"github.com/imdario/mergo"
	"k8s.io/api/core/v1"
)

func flattenReplicationControllerSpec(in v1.ReplicationControllerSpec) ([]interface{}, error) {
	att := make(map[string]interface{})
	att["min_ready_seconds"] = in.MinReadySeconds

	if in.Replicas != nil {
		att["replicas"] = *in.Replicas
	}

	att["selector"] = in.Selector

	if in.Template != nil {
		template, err := flattenReplicationControllerTemplate(*in.Template)
		if err != nil {
			return nil, err
		}
		att["template"] = template
	}

	return []interface{}{att}, nil
}

func flattenReplicationControllerTemplate(in v1.PodTemplateSpec) ([]interface{}, error) {
	att := make(map[string]interface{})

	podSpec, err := flattenPodSpec(in.Spec)
	if err != nil {
		return nil, err
	}

	// Put the pod spec directly at the base template field to support deprecated fields
	att = podSpec[0].(map[string]interface{})

	// Also put the pod spec in the new spec field
	att["spec"] = podSpec[0].(map[string]interface{})

	// HINT: use diffSuppressFunc for labels automatically added from selector?
	att["metadata"] = flattenMetadata(in.ObjectMeta)

	return []interface{}{att}, nil
}

func expandReplicationControllerSpec(rc []interface{}) (*v1.ReplicationControllerSpec, error) {
	obj := &v1.ReplicationControllerSpec{}
	if len(rc) == 0 || rc[0] == nil {
		return obj, nil
	}
	in := rc[0].(map[string]interface{})
	obj.MinReadySeconds = int32(in["min_ready_seconds"].(int))
	obj.Replicas = ptrToInt32(int32(in["replicas"].(int)))
	obj.Selector = expandStringMap(in["selector"].(map[string]interface{}))

	template, err := expandReplicationControllerTemplate(in["template"].([]interface{}), obj.Selector)
	if err != nil {
		return obj, err
	}

	obj.Template = template

	return obj, nil
}

func expandReplicationControllerTemplate(rct []interface{}, selector map[string]string) (*v1.PodTemplateSpec, error) {
	obj := &v1.PodTemplateSpec{}
	in := rct[0].(map[string]interface{})

	// Get user defined metadata
	metadata := expandMetadata(in["metadata"].([]interface{}))

	// Add labels from selector to ensure proper selection of pods by the replication controller for deprecated use case
	if metadata.Labels == nil {
		metadata.Labels = selector
	}
	obj.ObjectMeta = metadata

	// Get pod spec from deprecated fields
	podSpecDeprecated, err := expandPodSpec(rct)
	if err != nil {
		return obj, err
	}

	// Get pod spec from new fields
	podSpec, err := expandPodSpec(in["spec"].([]interface{}))
	if err != nil {
		return obj, err
	}

	// Merge them overriding the deprecated ones by the new ones
	if err = mergo.MergeWithOverwrite(&podSpecDeprecated, podSpec); err != nil {
		return obj, err
	}

	obj.Spec = *podSpecDeprecated

	return obj, nil
}
