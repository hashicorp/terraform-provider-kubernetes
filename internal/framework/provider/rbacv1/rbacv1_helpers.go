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

func expandRoleRef(in []RoleRefModel) rbacv1api.RoleRef {
	if len(in) == 0 {
		return rbacv1api.RoleRef{}
	}
	r := in[0]
	return rbacv1api.RoleRef{
		APIGroup: r.APIGroup.ValueString(),
		Kind:     r.Kind.ValueString(),
		Name:     r.Name.ValueString(),
	}
}

func flattenRoleRef(in rbacv1api.RoleRef) []RoleRefModel {
	return []RoleRefModel{{
		APIGroup: types.StringValue(in.APIGroup),
		Kind:     types.StringValue(in.Kind),
		Name:     types.StringValue(in.Name),
	}}
}

func expandSubjects(in []SubjectModel) []rbacv1api.Subject {
	subjects := make([]rbacv1api.Subject, 0, len(in))
	for _, s := range in {
		subject := rbacv1api.Subject{
			Kind: s.Kind.ValueString(),
			Name: s.Name.ValueString(),
		}
		if !s.APIGroup.IsNull() && !s.APIGroup.IsUnknown() {
			subject.APIGroup = s.APIGroup.ValueString()
		}
		if !s.Namespace.IsNull() && !s.Namespace.IsUnknown() {
			subject.Namespace = s.Namespace.ValueString()
		}
		subjects = append(subjects, subject)
	}
	return subjects
}

// flattenSubjects converts API subjects to the model. When prior (the subject list
// from state) is given, it keeps a prior-state api_group of "" when the API returns
// the canonical rbac.authorization.k8s.io for a User or Group subject: the API
// server defaults an empty api_group for those kinds, so the two values are
// equivalent and rewriting the configured "" would cause a perpetual diff. SDKv2
// has no equivalent code: it stores the API value, and its SDK treats "" as unset
// for this Optional+Computed attribute, so it shows no diff. Pass nil for prior
// when there is no state.
func flattenSubjects(in []rbacv1api.Subject, prior []SubjectModel) []SubjectModel {
	result := make([]SubjectModel, 0, len(in))
	for _, s := range in {
		model := SubjectModel{
			// SDKv2 stores an empty api_group for ServiceAccount subjects.
			APIGroup: types.StringValue(s.APIGroup),
			Kind:     types.StringValue(s.Kind),
			Name:     types.StringValue(s.Name),
			// The API echoes back the namespace that was sent, so an empty
			// value is kept as-is; the schema default applies only at plan time.
			Namespace: types.StringValue(s.Namespace),
		}
		result = append(result, model)
	}
	for i := range result {
		if i >= len(prior) {
			break
		}
		p := prior[i]
		if p.APIGroup.IsNull() || p.APIGroup.IsUnknown() || p.APIGroup.ValueString() != "" {
			continue
		}
		if p.Kind.ValueString() != in[i].Kind || p.Name.ValueString() != in[i].Name {
			continue
		}
		if (in[i].Kind == rbacv1api.UserKind || in[i].Kind == rbacv1api.GroupKind) && in[i].APIGroup == rbacv1api.GroupName {
			result[i].APIGroup = types.StringValue("")
		}
	}
	return result
}
