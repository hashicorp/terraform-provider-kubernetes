// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func flattenReplicationControllerSpec(in corev1.ReplicationControllerSpec, d *schema.ResourceData, meta interface{}) ([]interface{}, error) {
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
		template["spec"] = podSpec
		template["metadata"] = flattenMetadata(in.Template.ObjectMeta, d, meta)
		att["template"] = []interface{}{template}
	}

	return []interface{}{att}, nil
}

func expandReplicationControllerSpec(rc []interface{}) (*corev1.ReplicationControllerSpec, error) {
	obj := &corev1.ReplicationControllerSpec{}
	if len(rc) == 0 || rc[0] == nil {
		return obj, nil
	}
	in := rc[0].(map[string]interface{})
	obj.MinReadySeconds = int32(in["min_ready_seconds"].(int))
	obj.Replicas = ptr.To(int32(in["replicas"].(int)))
	obj.Selector = expandStringMap(in["selector"].(map[string]interface{}))

	template, err := expandReplicationControllerTemplate(in["template"].([]interface{}))
	if err != nil {
		return obj, err
	}

	obj.Template = template

	return obj, nil
}

func expandReplicationControllerTemplate(rct []interface{}) (*corev1.PodTemplateSpec, error) {
	obj := &corev1.PodTemplateSpec{}
	in := rct[0].(map[string]interface{})
	metadata := in["metadata"].([]interface{})
	obj.ObjectMeta = expandMetadata(metadata)

	podSpec, err := expandPodSpec(in["spec"].([]interface{}))
	if err != nil {
		return obj, err
	}
	obj.Spec = *podSpec

	return obj, nil
}
