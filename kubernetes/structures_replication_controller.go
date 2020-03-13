package kubernetes

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"k8s.io/api/core/v1"
)

func flattenReplicationControllerSpec(in v1.ReplicationControllerSpec, d *schema.ResourceData, useDeprecatedSpecFields bool) ([]interface{}, error) {
	att := make(map[string]interface{})
	att["min_ready_seconds"] = in.MinReadySeconds

	if in.Replicas != nil {
		att["replicas"] = *in.Replicas
	}

	if in.Selector != nil {
		att["selector"] = in.Selector
	}

	if in.Template != nil {
		podSpec, err := flattenPodSpec(in.Template.Spec)
		if err != nil {
			return nil, err
		}
		template := make(map[string]interface{})

		if useDeprecatedSpecFields {
			// Use deprecated fields
			for k, v := range podSpec[0].(map[string]interface{}) {
				template[k] = v
			}
		} else {
			// Use new fields
			template["spec"] = podSpec
			template["metadata"] = flattenMetadata(in.Template.ObjectMeta, d)
		}

		att["template"] = []interface{}{template}
	}

	return []interface{}{att}, nil
}

func expandReplicationControllerSpec(rc []interface{}, useDeprecatedSpecFields bool) (*v1.ReplicationControllerSpec, error) {
	obj := &v1.ReplicationControllerSpec{}
	if len(rc) == 0 || rc[0] == nil {
		return obj, nil
	}
	in := rc[0].(map[string]interface{})
	obj.MinReadySeconds = int32(in["min_ready_seconds"].(int))
	obj.Replicas = ptrToInt32(int32(in["replicas"].(int)))
	obj.Selector = expandStringMap(in["selector"].(map[string]interface{}))

	template, err := expandReplicationControllerTemplate(in["template"].([]interface{}), obj.Selector, useDeprecatedSpecFields)
	if err != nil {
		return obj, err
	}

	obj.Template = template

	return obj, nil
}

func expandReplicationControllerTemplate(rct []interface{}, selector map[string]string, useDeprecatedSpecFields bool) (*v1.PodTemplateSpec, error) {
	obj := &v1.PodTemplateSpec{}

	if useDeprecatedSpecFields {
		// Add labels from selector to ensure proper selection of pods by the replication controller for deprecated use case
		obj.ObjectMeta.Labels = selector

		// Get pod spec from deprecated fields
		podSpecDeprecated, err := expandPodSpec(rct)
		if err != nil {
			return obj, err
		}
		obj.Spec = *podSpecDeprecated
	} else {
		in := rct[0].(map[string]interface{})
		metadata := in["metadata"].([]interface{})

		// Return an error if new spec fields are used but no metadata is defined to preserve the Required property of the metadata field
		// cf. https://www.terraform.io/docs/extend/best-practices/deprecations.html#renaming-a-required-attribute
		if len(metadata) < 1 {
			return obj, errors.New("`spec.template.metadata` is Required when new 'spec.template.spec' fields are used.")
		}

		// Get user defined metadata
		obj.ObjectMeta = expandMetadata(metadata)

		// Get pod spec from new fields
		podSpec, err := expandPodSpec(in["spec"].([]interface{}))
		if err != nil {
			return obj, err
		}
		obj.Spec = *podSpec
	}

	return obj, nil
}
