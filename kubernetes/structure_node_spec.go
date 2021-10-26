package kubernetes

import (
	v1 "k8s.io/api/core/v1"
)

// Flatteners

func flattenNodeSpec(in v1.NodeSpec) []interface{} {
	att := make(map[string]interface{})

	if in.PodCIDR != "" {
		att["pod_cidr"] = in.PodCIDR
	}

	if in.ProviderID != "" {
		att["provider_id"] = in.ProviderID
	}

	if in.PodCIDRs != nil {
		att["pod_cidrs"] = in.PodCIDRs
	}

	if in.Taints != nil {
		att["taints"] = flattenTaints(in.Taints)
	}

	if in.Unschedulable != false {
		att["unschedulable"] = in.Unschedulable
	}

	return []interface{}{att}
}

func flattenTaints(taints []v1.Taint) []interface{} {
	att := []interface{}{}
	for _, v := range taints {
		obj := map[string]interface{}{}

		if v.Effect != "" {
			obj["effect"] = string(v.Effect)
		}
		if v.Key != "" {
			obj["key"] = v.Key
		}
		if v.Value != "" {
			obj["value"] = v.Value
		}
		att = append(att, obj)
	}
	return att
}

// Expanders - No need
