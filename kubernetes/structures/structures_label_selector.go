package structures

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Flatteners

func FlattenLabelSelector(in *metav1.LabelSelector) []interface{} {
	att := make(map[string]interface{})
	if len(in.MatchLabels) > 0 {
		att["match_labels"] = in.MatchLabels
	}
	if len(in.MatchExpressions) > 0 {
		att["match_expressions"] = FlattenLabelSelectorRequirement(in.MatchExpressions)
	}
	return []interface{}{att}
}

func FlattenLabelSelectorRequirement(in []metav1.LabelSelectorRequirement) []interface{} {
	att := make([]interface{}, len(in), len(in))
	for i, n := range in {
		m := make(map[string]interface{})
		m["key"] = n.Key
		m["operator"] = n.Operator
		m["values"] = NewStringSet(schema.HashString, n.Values)
		att[i] = m
	}
	return att
}

// Expanders

func ExpandLabelSelector(l []interface{}) *metav1.LabelSelector {
	if len(l) == 0 || l[0] == nil {
		return &metav1.LabelSelector{}
	}
	in := l[0].(map[string]interface{})
	obj := &metav1.LabelSelector{}
	if v, ok := in["match_labels"].(map[string]interface{}); ok && len(v) > 0 {
		obj.MatchLabels = ExpandStringMap(v)
	}
	if v, ok := in["match_expressions"].([]interface{}); ok && len(v) > 0 {
		obj.MatchExpressions = ExpandLabelSelectorRequirement(v)
	}
	return obj
}

func ExpandLabelSelectorRequirement(l []interface{}) []metav1.LabelSelectorRequirement {
	if len(l) == 0 || l[0] == nil {
		return []metav1.LabelSelectorRequirement{}
	}
	obj := make([]metav1.LabelSelectorRequirement, len(l), len(l))
	for i, n := range l {
		in := n.(map[string]interface{})
		obj[i] = metav1.LabelSelectorRequirement{
			Key:      in["key"].(string),
			Operator: metav1.LabelSelectorOperator(in["operator"].(string)),
			Values:   SliceOfString(in["values"].(*schema.Set).List()),
		}
	}
	return obj
}
