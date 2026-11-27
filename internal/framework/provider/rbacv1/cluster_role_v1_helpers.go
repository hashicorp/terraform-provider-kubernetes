// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	rbacv1api "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func expandClusterRoleRules(rules []ClusterRoleRuleModel) ([]rbacv1api.PolicyRule, diag.Diagnostics) {
	var allDiags diag.Diagnostics
	result := make([]rbacv1api.PolicyRule, len(rules))

	for i, rule := range rules {
		apiGroups, diags := expandStringSet(rule.APIGroups)
		allDiags.Append(diags...)
		nonResourceURLs, diags := expandStringSet(rule.NonResourceURLs)
		allDiags.Append(diags...)
		resourceNames, diags := expandStringSet(rule.ResourceNames)
		allDiags.Append(diags...)
		resources, diags := expandStringSet(rule.Resources)
		allDiags.Append(diags...)
		verbs, diags := expandStringSet(rule.Verbs)
		allDiags.Append(diags...)

		result[i] = rbacv1api.PolicyRule{
			APIGroups:       apiGroups,
			NonResourceURLs: nonResourceURLs,
			ResourceNames:   resourceNames,
			Resources:       resources,
			Verbs:           verbs,
		}
	}
	return result, allDiags
}

func flattenClusterRoleRules(rules []rbacv1api.PolicyRule) ([]ClusterRoleRuleModel, diag.Diagnostics) {
	var allDiags diag.Diagnostics
	result := make([]ClusterRoleRuleModel, len(rules))

	for i, rule := range rules {
		apiGroups, diags := flattenStringSet(rule.APIGroups)
		allDiags.Append(diags...)
		nonResourceURLs, diags := flattenStringSet(rule.NonResourceURLs)
		allDiags.Append(diags...)
		resourceNames, diags := flattenStringSet(rule.ResourceNames)
		allDiags.Append(diags...)
		resources, diags := flattenStringSet(rule.Resources)
		allDiags.Append(diags...)
		verbs, diags := flattenStringSet(rule.Verbs)
		allDiags.Append(diags...)

		result[i] = ClusterRoleRuleModel{
			APIGroups:       apiGroups,
			NonResourceURLs: nonResourceURLs,
			ResourceNames:   resourceNames,
			Resources:       resources,
			Verbs:           verbs,
		}
	}
	return result, allDiags
}

func clusterRoleRulesEqual(a, b []ClusterRoleRuleModel) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].APIGroups.Equal(b[i].APIGroups) ||
			!a[i].NonResourceURLs.Equal(b[i].NonResourceURLs) ||
			!a[i].ResourceNames.Equal(b[i].ResourceNames) ||
			!a[i].Resources.Equal(b[i].Resources) ||
			!a[i].Verbs.Equal(b[i].Verbs) {
			return false
		}
	}
	return true
}

// expandAggregationRule returns nil when the block is absent. A present but empty block
// yields an empty AggregationRule, as SDKv2's expandClusterRoleAggregationRule does.
func expandAggregationRule(ctx context.Context, in []AggregationRuleModel) (*rbacv1api.AggregationRule, diag.Diagnostics) {
	if len(in) == 0 {
		return nil, nil
	}
	var allDiags diag.Diagnostics
	out := &rbacv1api.AggregationRule{}

	for _, sel := range in[0].ClusterRoleSelectors {
		ls := metav1.LabelSelector{}

		if !sel.MatchLabels.IsNull() && !sel.MatchLabels.IsUnknown() {
			labels := map[string]string{}
			allDiags.Append(sel.MatchLabels.ElementsAs(ctx, &labels, false)...)
			if len(labels) > 0 {
				ls.MatchLabels = labels
			}
		}

		for _, expr := range sel.MatchExpressions {
			values, diags := expandStringSet(expr.Values)
			allDiags.Append(diags...)
			ls.MatchExpressions = append(ls.MatchExpressions, metav1.LabelSelectorRequirement{
				Key:      expr.Key.ValueString(),
				Operator: metav1.LabelSelectorOperator(expr.Operator.ValueString()),
				Values:   values,
			})
		}
		out.ClusterRoleSelectors = append(out.ClusterRoleSelectors, ls)
	}
	return out, allDiags
}

// flattenAggregationRule converts the API aggregation rule to the model. prior is the
// aggregation_rule from state, or nil when there is none (import). SDKv2 stores an
// unset match_labels or values as null, so an empty API value becomes null, unless the
// prior state held a known empty value at the same position: that is a configured
// `{}` or `[]`, and rewriting it to null would be a perpetual diff.
func flattenAggregationRule(in *rbacv1api.AggregationRule, prior []AggregationRuleModel) ([]AggregationRuleModel, diag.Diagnostics) {
	if in == nil {
		return []AggregationRuleModel{}, nil
	}
	var allDiags diag.Diagnostics

	var priorSelectors []ClusterRoleSelectorModel
	if len(prior) > 0 {
		priorSelectors = prior[0].ClusterRoleSelectors
	}

	selectors := make([]ClusterRoleSelectorModel, len(in.ClusterRoleSelectors))
	for i, ls := range in.ClusterRoleSelectors {
		var priorSel ClusterRoleSelectorModel
		if i < len(priorSelectors) {
			priorSel = priorSelectors[i]
		}

		matchLabels := types.MapNull(types.StringType)
		if len(ls.MatchLabels) > 0 {
			elems := make(map[string]attr.Value, len(ls.MatchLabels))
			for k, v := range ls.MatchLabels {
				elems[k] = types.StringValue(v)
			}
			m, diags := types.MapValue(types.StringType, elems)
			allDiags.Append(diags...)
			matchLabels = m
		} else if isKnownEmptyMap(priorSel.MatchLabels) {
			matchLabels = priorSel.MatchLabels
		}

		exprs := make([]MatchExpressionModel, len(ls.MatchExpressions))
		for j, req := range ls.MatchExpressions {
			var priorExpr MatchExpressionModel
			if j < len(priorSel.MatchExpressions) {
				priorExpr = priorSel.MatchExpressions[j]
			}

			values := types.SetNull(types.StringType)
			if len(req.Values) > 0 {
				v, diags := flattenStringSet(req.Values)
				allDiags.Append(diags...)
				values = v
			} else if isKnownEmptySet(priorExpr.Values) {
				values = priorExpr.Values
			}
			exprs[j] = MatchExpressionModel{
				Key:      types.StringValue(req.Key),
				Operator: types.StringValue(string(req.Operator)),
				Values:   values,
			}
		}

		selectors[i] = ClusterRoleSelectorModel{
			MatchLabels:      matchLabels,
			MatchExpressions: exprs,
		}
	}
	return []AggregationRuleModel{{ClusterRoleSelectors: selectors}}, allDiags
}

func isKnownEmptySet(s types.Set) bool {
	return !s.IsNull() && !s.IsUnknown() && len(s.Elements()) == 0
}

func isKnownEmptyMap(m types.Map) bool {
	return !m.IsNull() && !m.IsUnknown() && len(m.Elements()) == 0
}

func aggregationRulesEqual(a, b []AggregationRuleModel) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return true
	}
	as, bs := a[0].ClusterRoleSelectors, b[0].ClusterRoleSelectors
	if len(as) != len(bs) {
		return false
	}
	for i := range as {
		if !as[i].MatchLabels.Equal(bs[i].MatchLabels) || len(as[i].MatchExpressions) != len(bs[i].MatchExpressions) {
			return false
		}
		for j := range as[i].MatchExpressions {
			x, y := as[i].MatchExpressions[j], bs[i].MatchExpressions[j]
			if !x.Key.Equal(y.Key) || !x.Operator.Equal(y.Operator) || !x.Values.Equal(y.Values) {
				return false
			}
		}
	}
	return true
}
