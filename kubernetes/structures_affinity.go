package kubernetes

import (
	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/api/core/v1"
)

// Flatteners

func flattenAffinity(in *v1.Affinity) []interface{} {
	att := make(map[string]interface{})
	if in.NodeAffinity != nil {
		att["node_affinity"] = flattenNodeAffinity(in.NodeAffinity)
	}
	if in.PodAffinity != nil {
		att["pod_affinity"] = flattenPodAffinity(in.PodAffinity)
	}
	if in.PodAntiAffinity != nil {
		att["pod_anti_affinity"] = flattenPodAntiAffinity(in.PodAntiAffinity)
	}
	if len(att) > 0 {
		return []interface{}{att}
	}
	return []interface{}{}
}

func flattenNodeAffinity(in *v1.NodeAffinity) []interface{} {
	att := make(map[string]interface{})
	if in.RequiredDuringSchedulingIgnoredDuringExecution != nil {
		att["required_during_scheduling_ignored_during_execution"] = flattenNodeSelector(in.RequiredDuringSchedulingIgnoredDuringExecution)
	}
	if in.PreferredDuringSchedulingIgnoredDuringExecution != nil {
		att["preferred_during_scheduling_ignored_during_execution"] = flattenPreferredSchedulingTerm(in.PreferredDuringSchedulingIgnoredDuringExecution)
	}
	if len(att) > 0 {
		return []interface{}{att}
	}
	return []interface{}{}
}

func flattenPodAffinity(in *v1.PodAffinity) []interface{} {
	att := make(map[string]interface{})
	if len(in.RequiredDuringSchedulingIgnoredDuringExecution) > 0 {
		att["required_during_scheduling_ignored_during_execution"] = flattenPodAffinityTerms(in.RequiredDuringSchedulingIgnoredDuringExecution)
	}
	if len(in.PreferredDuringSchedulingIgnoredDuringExecution) > 0 {
		att["preferred_during_scheduling_ignored_during_execution"] = flattenWeightedPodAffinityTerms(in.PreferredDuringSchedulingIgnoredDuringExecution)
	}
	if len(att) > 0 {
		return []interface{}{att}
	}
	return []interface{}{}
}

func flattenPodAntiAffinity(in *v1.PodAntiAffinity) []interface{} {
	att := make(map[string]interface{})
	if len(in.RequiredDuringSchedulingIgnoredDuringExecution) > 0 {
		att["required_during_scheduling_ignored_during_execution"] = flattenPodAffinityTerms(in.RequiredDuringSchedulingIgnoredDuringExecution)
	}
	if len(in.PreferredDuringSchedulingIgnoredDuringExecution) > 0 {
		att["preferred_during_scheduling_ignored_during_execution"] = flattenWeightedPodAffinityTerms(in.PreferredDuringSchedulingIgnoredDuringExecution)
	}
	if len(att) > 0 {
		return []interface{}{att}
	}
	return []interface{}{}
}

func flattenWeightedPodAffinityTerms(in []v1.WeightedPodAffinityTerm) []interface{} {
	att := make([]interface{}, len(in), len(in))
	for i, n := range in {
		m := make(map[string]interface{})
		m["weight"] = int(n.Weight)
		m["pod_affinity_term"] = flattenPodAffinityTerms([]v1.PodAffinityTerm{n.PodAffinityTerm})
		att[i] = m
	}
	return att
}

func flattenPodAffinityTerms(in []v1.PodAffinityTerm) []interface{} {
	att := make([]interface{}, len(in), len(in))
	for i, n := range in {
		m := make(map[string]interface{})
		m["namespaces"] = newStringSet(schema.HashString, n.Namespaces)
		m["topology_key"] = n.TopologyKey
		if n.LabelSelector != nil {
			m["label_selector"] = flattenLabelSelector(n.LabelSelector)
		}
		att[i] = m
	}
	return att
}

func flattenNodeSelector(in *v1.NodeSelector) []interface{} {
	att := make(map[string]interface{})
	if len(in.NodeSelectorTerms) > 0 {
		att["node_selector_term"] = flattenNodeSelectorTerms(in.NodeSelectorTerms)
	}
	if len(att) > 0 {
		return []interface{}{att}
	}
	return []interface{}{}
}

func flattenNodeSelectorTerms(in []v1.NodeSelectorTerm) []interface{} {
	att := make([]interface{}, len(in), len(in))
	for i, n := range in {
		m := make(map[string]interface{})
		if len(n.MatchExpressions) > 0 {
			m["match_expressions"] = flattenNodeSelectorRequirements(n.MatchExpressions)
		}
		att[i] = m
	}
	return att
}

func flattenNodeSelectorRequirements(in []v1.NodeSelectorRequirement) []interface{} {
	att := make([]interface{}, len(in), len(in))
	for i, n := range in {
		m := make(map[string]interface{})
		m["key"] = n.Key
		m["operator"] = n.Operator
		m["values"] = newStringSet(schema.HashString, n.Values)
		att[i] = m
	}
	return att
}

func flattenPreferredSchedulingTerm(in []v1.PreferredSchedulingTerm) []interface{} {
	att := make([]interface{}, len(in), len(in))
	for i, n := range in {
		m := make(map[string]interface{})
		m["weight"] = int(n.Weight)
		m["preference"] = flattenNodeSelectorTerm(n.Preference)
		att[i] = m
	}
	return att
}

func flattenNodeSelectorTerm(in v1.NodeSelectorTerm) []interface{} {
	if len(in.MatchExpressions) > 0 {
		m := make(map[string]interface{})
		m["match_expressions"] = flattenNodeSelectorRequirements(in.MatchExpressions)
		return []interface{}{m}
	}
	return []interface{}{}
}

// Expanders

func expandAffinity(a []interface{}) (*v1.Affinity, error) {
	if len(a) == 0 || a[0] == nil {
		return &v1.Affinity{}, nil
	}
	in := a[0].(map[string]interface{})
	obj := v1.Affinity{}
	if v, ok := in["node_affinity"].([]interface{}); ok && len(v) > 0 {
		obj.NodeAffinity = expandNodeAffinity(v)
	}
	if v, ok := in["pod_affinity"].([]interface{}); ok && len(v) > 0 {
		obj.PodAffinity = expandPodAffinity(v)
	}
	if v, ok := in["pod_anti_affinity"].([]interface{}); ok && len(v) > 0 {
		obj.PodAntiAffinity = expandPodAntiAffinity(v)
	}
	return &obj, nil
}

func expandNodeAffinity(a []interface{}) *v1.NodeAffinity {
	if len(a) == 0 || a[0] == nil {
		return &v1.NodeAffinity{}
	}
	in := a[0].(map[string]interface{})
	obj := v1.NodeAffinity{}
	if v, ok := in["required_during_scheduling_ignored_during_execution"].([]interface{}); ok && len(v) > 0 {
		obj.RequiredDuringSchedulingIgnoredDuringExecution = expandNodeSelector(v)
	}
	if v, ok := in["preferred_during_scheduling_ignored_during_execution"].([]interface{}); ok && len(v) > 0 {
		obj.PreferredDuringSchedulingIgnoredDuringExecution = expandPreferredSchedulingTerms(v)
	}
	return &obj
}

func expandPodAffinity(a []interface{}) *v1.PodAffinity {
	if len(a) == 0 || a[0] == nil {
		return &v1.PodAffinity{}
	}
	in := a[0].(map[string]interface{})
	obj := v1.PodAffinity{}
	if v, ok := in["required_during_scheduling_ignored_during_execution"].([]interface{}); ok && len(v) > 0 {
		obj.RequiredDuringSchedulingIgnoredDuringExecution = expandPodAffinityTerms(v)
	}
	if v, ok := in["preferred_during_scheduling_ignored_during_execution"].([]interface{}); ok && len(v) > 0 {
		obj.PreferredDuringSchedulingIgnoredDuringExecution = expandWeightedPodAffinityTerms(v)
	}
	return &obj
}

func expandPodAntiAffinity(a []interface{}) *v1.PodAntiAffinity {
	if len(a) == 0 || a[0] == nil {
		return &v1.PodAntiAffinity{}
	}
	in := a[0].(map[string]interface{})
	obj := v1.PodAntiAffinity{}
	if v, ok := in["required_during_scheduling_ignored_during_execution"].([]interface{}); ok && len(v) > 0 {
		obj.RequiredDuringSchedulingIgnoredDuringExecution = expandPodAffinityTerms(v)
	}
	if v, ok := in["preferred_during_scheduling_ignored_during_execution"].([]interface{}); ok && len(v) > 0 {
		obj.PreferredDuringSchedulingIgnoredDuringExecution = expandWeightedPodAffinityTerms(v)
	}
	return &obj
}

func expandPreferredSchedulingTerms(t []interface{}) []v1.PreferredSchedulingTerm {
	if len(t) == 0 || t[0] == nil {
		return []v1.PreferredSchedulingTerm{}
	}
	obj := make([]v1.PreferredSchedulingTerm, len(t), len(t))
	for i, n := range t {
		in := n.(map[string]interface{})
		if v, ok := in["weight"].(int); ok {
			obj[i].Weight = int32(v)
		}
		if v, ok := in["preference"].([]interface{}); ok && len(v) > 0 {
			obj[i].Preference = expandNodeSelectorTerm(v)
		}
	}
	return obj
}

func expandNodeSelectorTerm(t []interface{}) v1.NodeSelectorTerm {
	if len(t) == 0 || t[0] == nil {
		return v1.NodeSelectorTerm{}
	}
	in := t[0].(map[string]interface{})
	obj := v1.NodeSelectorTerm{}
	if v, ok := in["match_expressions"].([]interface{}); ok && len(v) > 0 {
		obj.MatchExpressions = expandNodeSelectorRequirements(v)
	}
	return obj
}

func expandNodeSelector(s []interface{}) *v1.NodeSelector {
	if len(s) == 0 || s[0] == nil {
		return &v1.NodeSelector{}
	}
	in := s[0].(map[string]interface{})
	obj := v1.NodeSelector{}
	if v, ok := in["node_selector_term"].([]interface{}); ok && len(v) > 0 {
		obj.NodeSelectorTerms = expandNodeSelectorTerms(v)
	}
	return &obj
}

func expandNodeSelectorTerms(t []interface{}) []v1.NodeSelectorTerm {
	if len(t) == 0 || t[0] == nil {
		return []v1.NodeSelectorTerm{}
	}
	obj := make([]v1.NodeSelectorTerm, len(t), len(t))
	for i, n := range t {
		in := n.(map[string]interface{})
		if v, ok := in["match_expressions"].([]interface{}); ok && len(v) > 0 {
			obj[i].MatchExpressions = expandNodeSelectorRequirements(v)
		}
	}
	return obj
}

func expandNodeSelectorRequirements(r []interface{}) []v1.NodeSelectorRequirement {
	if len(r) == 0 || r[0] == nil {
		return []v1.NodeSelectorRequirement{}
	}
	obj := make([]v1.NodeSelectorRequirement, len(r), len(r))
	for i, n := range r {
		in := n.(map[string]interface{})
		obj[i] = v1.NodeSelectorRequirement{
			Key:      in["key"].(string),
			Operator: v1.NodeSelectorOperator(in["operator"].(string)),
			Values:   sliceOfString(in["values"].(*schema.Set).List()),
		}
	}
	return obj
}

func expandPodAffinityTerms(t []interface{}) []v1.PodAffinityTerm {
	if len(t) == 0 || t[0] == nil {
		return []v1.PodAffinityTerm{}
	}
	obj := make([]v1.PodAffinityTerm, len(t), len(t))
	for i, n := range t {
		in := n.(map[string]interface{})
		if v, ok := in["label_selector"].([]interface{}); ok && len(v) > 0 {
			obj[i].LabelSelector = expandLabelSelector(v)
		}
		if v, ok := in["namespaces"].(*schema.Set); ok {
			obj[i].Namespaces = sliceOfString(v.List())
		}
		if v, ok := in["topology_key"].(string); ok {
			obj[i].TopologyKey = v
		}
	}
	return obj
}

func expandWeightedPodAffinityTerms(t []interface{}) []v1.WeightedPodAffinityTerm {
	if len(t) == 0 || t[0] == nil {
		return []v1.WeightedPodAffinityTerm{}
	}
	obj := make([]v1.WeightedPodAffinityTerm, len(t), len(t))
	for i, n := range t {
		in := n.(map[string]interface{})
		if v, ok := in["weight"].(int); ok {
			obj[i].Weight = int32(v)
		}
		if v, ok := in["pod_affinity_term"].([]interface{}); ok && len(v) > 0 {
			obj[i].PodAffinityTerm = expandPodAffinityTerms(v)[0]
		}
	}
	return obj
}
