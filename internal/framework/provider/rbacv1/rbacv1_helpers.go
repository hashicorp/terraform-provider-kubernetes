// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	rbacv1api "k8s.io/api/rbac/v1"
)

func expandStringSet(set types.Set) ([]string, diag.Diagnostics) {
	if set.IsNull() || set.IsUnknown() {
		return nil, nil
	}

	elements := set.Elements()
	result := make([]string, 0, len(elements))
	for _, element := range elements {
		v, ok := element.(types.String)
		if !ok {
			return nil, diag.Diagnostics{diag.NewErrorDiagnostic(
				"Unexpected Set Element Type", "Expected a string element.",
			)}
		}
		if !v.IsNull() && !v.IsUnknown() {
			result = append(result, v.ValueString())
		}
	}
	return result, nil
}

func flattenStringSet(slice []string) (types.Set, diag.Diagnostics) {
	elements := make([]attr.Value, len(slice))
	for i, v := range slice {
		elements[i] = types.StringValue(v)
	}

	return types.SetValue(types.StringType, elements)
}

func expandPolicyRules(rules []RuleModel) ([]rbacv1api.PolicyRule, diag.Diagnostics) {
	var allDiags diag.Diagnostics
	result := make([]rbacv1api.PolicyRule, len(rules))

	for i, rule := range rules {
		apiGroups, diags := expandStringSet(rule.APIGroups)
		allDiags.Append(diags...)

		resources, diags := expandStringSet(rule.Resources)
		allDiags.Append(diags...)

		resourceNames, diags := expandStringSet(rule.ResourceNames)
		allDiags.Append(diags...)

		verbs, diags := expandStringSet(rule.Verbs)
		allDiags.Append(diags...)

		result[i] = rbacv1api.PolicyRule{
			APIGroups:     apiGroups,
			Resources:     resources,
			ResourceNames: resourceNames,
			Verbs:         verbs,
		}
	}

	return result, allDiags
}

func flattenPolicyRules(rules []rbacv1api.PolicyRule) ([]RuleModel, diag.Diagnostics) {
	var allDiags diag.Diagnostics
	result := make([]RuleModel, len(rules))

	for i, rule := range rules {
		apiGroups, diags := flattenStringSet(rule.APIGroups)
		allDiags.Append(diags...)

		resources, diags := flattenStringSet(rule.Resources)
		allDiags.Append(diags...)

		resourceNames, diags := flattenStringSet(rule.ResourceNames)
		allDiags.Append(diags...)

		verbs, diags := flattenStringSet(rule.Verbs)
		allDiags.Append(diags...)

		result[i] = RuleModel{
			APIGroups:     apiGroups,
			Resources:     resources,
			ResourceNames: resourceNames,
			Verbs:         verbs,
		}
	}

	return result, allDiags
}

func rulesEqual(a, b []RuleModel) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].APIGroups.Equal(b[i].APIGroups) ||
			!a[i].Resources.Equal(b[i].Resources) ||
			!a[i].ResourceNames.Equal(b[i].ResourceNames) ||
			!a[i].Verbs.Equal(b[i].Verbs) {
			return false
		}
	}
	return true
}
