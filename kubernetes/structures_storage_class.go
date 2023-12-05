// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	v1 "k8s.io/api/core/v1"
	storageapi "k8s.io/api/storage/v1"
)

func flattenStorageClass(in storageapi.StorageClass) map[string]interface{} {
	att := make(map[string]interface{})
	att["parameters"] = in.Parameters
	att["storage_provisioner"] = in.Provisioner
	att["reclaim_policy"] = in.ReclaimPolicy
	if in.VolumeBindingMode != nil {
		att["volume_binding_mode"] = string(*in.VolumeBindingMode)
	}
	if in.AllowedTopologies != nil {
		att["allowed_topologies"] = flattenStorageClassAllowedTopologies(in.AllowedTopologies)
	}
	att["mount_options"] = newStringSet(schema.HashString, in.MountOptions)
	if in.AllowVolumeExpansion != nil {
		att["allow_volume_expansion"] = *in.AllowVolumeExpansion
	}
	return att
}

func expandStorageClassAllowedTopologies(l []interface{}) []v1.TopologySelectorTerm {
	if len(l) == 0 || l[0] == nil {
		return []v1.TopologySelectorTerm{}
	}

	in := l[0].(map[string]interface{})
	topologies := make([]v1.TopologySelectorTerm, 0)
	obj := v1.TopologySelectorTerm{}

	if v, ok := in["match_label_expressions"].([]interface{}); ok && len(v) > 0 {
		obj.MatchLabelExpressions = expandStorageClassMatchLabelExpressions(v)
	}

	topologies = append(topologies, obj)

	return topologies
}

func expandStorageClassMatchLabelExpressions(l []interface{}) []v1.TopologySelectorLabelRequirement {
	if len(l) == 0 || l[0] == nil {
		return []v1.TopologySelectorLabelRequirement{}
	}
	obj := make([]v1.TopologySelectorLabelRequirement, len(l))
	for i, n := range l {
		in := n.(map[string]interface{})
		obj[i] = v1.TopologySelectorLabelRequirement{
			Key:    in["key"].(string),
			Values: sliceOfString(in["values"].(*schema.Set).List()),
		}
	}
	return obj
}

func flattenStorageClassAllowedTopologies(in []v1.TopologySelectorTerm) []interface{} {
	att := make(map[string]interface{})
	for _, n := range in {
		if len(n.MatchLabelExpressions) > 0 {
			att["match_label_expressions"] = flattenStorageClassMatchLabelExpressions(n.MatchLabelExpressions)
		}
	}
	return []interface{}{att}
}

func flattenStorageClassMatchLabelExpressions(in []v1.TopologySelectorLabelRequirement) []interface{} {
	att := make([]interface{}, len(in))
	for i, n := range in {
		m := make(map[string]interface{})
		m["key"] = n.Key
		m["values"] = newStringSet(schema.HashString, n.Values)
		att[i] = m
	}
	return att
}
