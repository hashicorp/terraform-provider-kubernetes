package v1

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes/structures"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func flattenNodeSelectorRequirementList(in []corev1.NodeSelectorRequirement) []map[string]interface{} {
	att := make([]map[string]interface{}, len(in))
	for i, v := range in {
		m := map[string]interface{}{}
		m["key"] = v.Key
		m["values"] = structures.NewStringSet(schema.HashString, v.Values)
		m["operator"] = string(v.Operator)
		att[i] = m
	}
	return att
}

func expandNodeSelectorRequirementList(in []interface{}) []corev1.NodeSelectorRequirement {
	att := []corev1.NodeSelectorRequirement{}
	if len(in) < 1 {
		return att
	}
	att = make([]corev1.NodeSelectorRequirement, len(in))
	for i, c := range in {
		p := c.(map[string]interface{})
		att[i].Key = p["key"].(string)
		att[i].Operator = corev1.NodeSelectorOperator(p["operator"].(string))
		att[i].Values = structures.ExpandStringSlice(p["values"].(*schema.Set).List())
	}
	return att
}

func flattenNodeSelectorTerm(in corev1.NodeSelectorTerm) []interface{} {
	att := make(map[string]interface{})
	if len(in.MatchExpressions) > 0 {
		att["match_expressions"] = flattenNodeSelectorRequirementList(in.MatchExpressions)
	}
	if len(in.MatchFields) > 0 {
		att["match_fields"] = flattenNodeSelectorRequirementList(in.MatchFields)
	}
	return []interface{}{att}
}

func expandNodeSelectorTerm(l []interface{}) *corev1.NodeSelectorTerm {
	if len(l) == 0 || l[0] == nil {
		return &corev1.NodeSelectorTerm{}
	}
	in := l[0].(map[string]interface{})
	obj := corev1.NodeSelectorTerm{}
	if v, ok := in["match_expressions"].([]interface{}); ok && len(v) > 0 {
		obj.MatchExpressions = expandNodeSelectorRequirementList(v)
	}
	if v, ok := in["match_fields"].([]interface{}); ok && len(v) > 0 {
		obj.MatchFields = expandNodeSelectorRequirementList(v)
	}
	return &obj
}

func flattenNodeSelectorTerms(in []corev1.NodeSelectorTerm) []interface{} {
	att := make([]interface{}, len(in), len(in))
	for i, n := range in {
		att[i] = flattenNodeSelectorTerm(n)[0]
	}
	return att
}

func expandNodeSelectorTerms(l []interface{}) []corev1.NodeSelectorTerm {
	if len(l) == 0 || l[0] == nil {
		return []corev1.NodeSelectorTerm{}
	}
	obj := make([]corev1.NodeSelectorTerm, len(l), len(l))
	for i, n := range l {
		obj[i] = *expandNodeSelectorTerm([]interface{}{n})
	}
	return obj
}

func flattenResourceList(l corev1.ResourceList) map[string]string {
	m := make(map[string]string)
	for k, v := range l {
		m[string(k)] = v.String()
	}
	return m
}

func expandMapToResourceList(m map[string]interface{}) (*corev1.ResourceList, error) {
	out := make(corev1.ResourceList)
	for stringKey, origValue := range m {
		key := corev1.ResourceName(stringKey)
		var value resource.Quantity

		if v, ok := origValue.(int); ok {
			q := resource.NewQuantity(int64(v), resource.DecimalExponent)
			value = *q
		} else if v, ok := origValue.(string); ok {
			var err error
			value, err = resource.ParseQuantity(v)
			if err != nil {
				return &out, err
			}
		} else {
			return &out, fmt.Errorf("Unexpected value type: %#v", origValue)
		}

		out[key] = value
	}
	return &out, nil
}

func flattenPersistentVolumeAccessModes(in []corev1.PersistentVolumeAccessMode) *schema.Set {
	var out = make([]interface{}, len(in), len(in))
	for i, v := range in {
		out[i] = string(v)
	}
	return schema.NewSet(schema.HashString, out)
}

func expandPersistentVolumeAccessModes(s []interface{}) []corev1.PersistentVolumeAccessMode {
	out := make([]corev1.PersistentVolumeAccessMode, len(s), len(s))
	for i, v := range s {
		out[i] = corev1.PersistentVolumeAccessMode(v.(string))
	}
	return out
}

func flattenResourceQuotaSpec(in corev1.ResourceQuotaSpec) []interface{} {
	out := make([]interface{}, 1)

	m := make(map[string]interface{}, 0)
	m["hard"] = flattenResourceList(in.Hard)
	m["scopes"] = flattenResourceQuotaScopes(in.Scopes)

	if in.ScopeSelector != nil {
		m["scope_selector"] = flattenResourceQuotaScopeSelector(in.ScopeSelector)
	}

	out[0] = m
	return out
}

func expandResourceQuotaSpec(s []interface{}) (*corev1.ResourceQuotaSpec, error) {
	out := &corev1.ResourceQuotaSpec{}
	if len(s) < 1 {
		return out, nil
	}
	m := s[0].(map[string]interface{})

	if v, ok := m["hard"]; ok {
		list, err := expandMapToResourceList(v.(map[string]interface{}))
		if err != nil {
			return out, err
		}
		out.Hard = *list
	}

	if v, ok := m["scopes"]; ok {
		out.Scopes = expandResourceQuotaScopes(v.(*schema.Set).List())
	}

	if v, ok := m["scope_selector"]; ok {
		out.ScopeSelector = expandResourceQuotaScopeSelector(v.([]interface{}))
	}

	return out, nil
}

func flattenResourceQuotaScopes(in []corev1.ResourceQuotaScope) *schema.Set {
	out := make([]string, len(in), len(in))
	for i, scope := range in {
		out[i] = string(scope)
	}
	return structures.NewStringSet(schema.HashString, out)
}

func expandResourceQuotaScopes(s []interface{}) []corev1.ResourceQuotaScope {
	out := make([]corev1.ResourceQuotaScope, len(s), len(s))
	for i, scope := range s {
		out[i] = corev1.ResourceQuotaScope(scope.(string))
	}
	return out
}

func expandResourceQuotaScopeSelector(s []interface{}) *corev1.ScopeSelector {
	if len(s) < 1 {
		return nil
	}
	m := s[0].(map[string]interface{})

	att := &corev1.ScopeSelector{}

	if v, ok := m["match_expression"].([]interface{}); ok {
		att.MatchExpressions = expandResourceQuotaScopeSelectorMatchExpressions(v)
	}

	return att
}

func expandResourceQuotaScopeSelectorMatchExpressions(s []interface{}) []corev1.ScopedResourceSelectorRequirement {
	out := make([]corev1.ScopedResourceSelectorRequirement, len(s), len(s))

	for i, raw := range s {
		matchExp := raw.(map[string]interface{})

		if v, ok := matchExp["scope_name"].(string); ok {
			out[i].ScopeName = corev1.ResourceQuotaScope(v)
		}

		if v, ok := matchExp["operator"].(string); ok {
			out[i].Operator = corev1.ScopeSelectorOperator(v)
		}

		if v, ok := matchExp["values"].(*schema.Set); ok && v.Len() > 0 {
			out[i].Values = structures.SliceOfString(v.List())
		}
	}
	return out
}

func flattenResourceQuotaScopeSelector(in *corev1.ScopeSelector) []interface{} {
	out := make([]interface{}, 1)

	m := make(map[string]interface{}, 0)
	m["match_expression"] = flattenResourceQuotaScopeSelectorMatchExpressions(in.MatchExpressions)

	out[0] = m
	return out
}

func flattenResourceQuotaScopeSelectorMatchExpressions(in []corev1.ScopedResourceSelectorRequirement) []interface{} {
	if len(in) == 0 {
		return []interface{}{}
	}
	out := make([]interface{}, len(in))

	for i, l := range in {
		m := make(map[string]interface{}, 0)
		m["operator"] = string(l.Operator)
		m["scope_name"] = string(l.ScopeName)

		if l.Values != nil && len(l.Values) > 0 {
			m["values"] = structures.NewStringSet(schema.HashString, l.Values)
		}

		out[i] = m
	}
	return out
}

func resourceListEquals(x, y corev1.ResourceList) bool {
	for k, v := range x {
		yValue, ok := y[k]
		if !ok {
			return false
		}
		if v.Cmp(yValue) != 0 {
			return false
		}
	}
	for k, v := range y {
		xValue, ok := x[k]
		if !ok {
			return false
		}
		if v.Cmp(xValue) != 0 {
			return false
		}
	}
	return true
}

func expandLimitRangeSpec(s []interface{}, isNew bool) (*corev1.LimitRangeSpec, error) {
	out := &corev1.LimitRangeSpec{}
	if len(s) < 1 || s[0] == nil {
		return out, nil
	}
	m := s[0].(map[string]interface{})

	if limits, ok := m["limit"].([]interface{}); ok {
		newLimits := make([]corev1.LimitRangeItem, len(limits), len(limits))

		for i, l := range limits {
			lrItem := corev1.LimitRangeItem{}
			limit := l.(map[string]interface{})

			if v, ok := limit["type"]; ok {
				lrItem.Type = corev1.LimitType(v.(string))
			}

			// defaultRequest is forbidden for Pod limits, even though it's set & returned by API
			// this is how we avoid sending it back
			if v, ok := limit["default_request"]; ok {
				drm := v.(map[string]interface{})
				if lrItem.Type == corev1.LimitTypePod && len(drm) > 0 {
					if isNew {
						return out, fmt.Errorf("limit.%d.default_request cannot be set for Pod limit", i)
					}
				} else {
					el, err := expandMapToResourceList(drm)
					if err != nil {
						return out, err
					}
					lrItem.DefaultRequest = *el
				}
			}

			if v, ok := limit["default"]; ok {
				el, err := expandMapToResourceList(v.(map[string]interface{}))
				if err != nil {
					return out, err
				}
				lrItem.Default = *el
			}
			if v, ok := limit["max"]; ok {
				el, err := expandMapToResourceList(v.(map[string]interface{}))
				if err != nil {
					return out, err
				}
				lrItem.Max = *el
			}
			if v, ok := limit["max_limit_request_ratio"]; ok {
				el, err := expandMapToResourceList(v.(map[string]interface{}))
				if err != nil {
					return out, err
				}
				lrItem.MaxLimitRequestRatio = *el
			}
			if v, ok := limit["min"]; ok {
				el, err := expandMapToResourceList(v.(map[string]interface{}))
				if err != nil {
					return out, err
				}
				lrItem.Min = *el
			}

			newLimits[i] = lrItem
		}

		out.Limits = newLimits
	}

	return out, nil
}

func flattenLimitRangeSpec(in corev1.LimitRangeSpec) []interface{} {
	if len(in.Limits) == 0 {
		return []interface{}{}
	}

	out := make([]interface{}, 1)
	limits := make([]interface{}, len(in.Limits), len(in.Limits))

	for i, l := range in.Limits {
		m := make(map[string]interface{}, 0)
		m["default"] = flattenResourceList(l.Default)
		m["default_request"] = flattenResourceList(l.DefaultRequest)
		m["max"] = flattenResourceList(l.Max)
		m["max_limit_request_ratio"] = flattenResourceList(l.MaxLimitRequestRatio)
		m["min"] = flattenResourceList(l.Min)
		m["type"] = string(l.Type)

		limits[i] = m
	}
	out[0] = map[string]interface{}{
		"limit": limits,
	}
	return out
}

func flattenLocalObjectReferenceArray(in []corev1.LocalObjectReference) []interface{} {
	att := []interface{}{}
	for _, v := range in {
		m := map[string]interface{}{
			"name": v.Name,
		}
		att = append(att, m)
	}
	return att
}

func expandLocalObjectReferenceArray(in []interface{}) []corev1.LocalObjectReference {
	att := []corev1.LocalObjectReference{}
	if len(in) < 1 {
		return att
	}
	att = make([]corev1.LocalObjectReference, len(in))
	for i, c := range in {
		p := c.(map[string]interface{})
		if name, ok := p["name"]; ok {
			att[i].Name = name.(string)
		}
	}
	return att
}

func flattenServiceAccountSecrets(in []corev1.ObjectReference, defaultSecretName string) []interface{} {
	att := make([]interface{}, 0)
	for _, v := range in {
		if v.Name == defaultSecretName {
			continue
		}
		m := map[string]interface{}{}
		if v.Name != "" {
			m["name"] = v.Name
		}
		att = append(att, m)
	}
	return att
}

func expandServiceAccountSecrets(in []interface{}, defaultSecretName string) []corev1.ObjectReference {
	att := make([]corev1.ObjectReference, 0)

	for _, c := range in {
		p := c.(map[string]interface{})
		if name, ok := p["name"]; ok {
			att = append(att, corev1.ObjectReference{Name: name.(string)})
		}
	}
	if defaultSecretName != "" {
		att = append(att, corev1.ObjectReference{Name: defaultSecretName})
	}

	return att
}

func flattenPersistentVolumeMountOptions(in []string) *schema.Set {
	var out = make([]interface{}, len(in), len(in))
	for i, v := range in {
		out[i] = string(v)
	}
	return schema.NewSet(schema.HashString, out)
}
