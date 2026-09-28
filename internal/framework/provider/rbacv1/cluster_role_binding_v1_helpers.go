// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	api "k8s.io/api/rbac/v1"
)

// subjectsEqual is order-sensitive because subject is a list.
func subjectsEqual(a, b []SubjectModel) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Kind.Equal(b[i].Kind) ||
			!a[i].Name.Equal(b[i].Name) ||
			!a[i].APIGroup.Equal(b[i].APIGroup) ||
			!a[i].Namespace.Equal(b[i].Namespace) {
			return false
		}
	}
	return true
}

func expandRoleRef(in []RoleRefModel) api.RoleRef {
	if len(in) == 0 {
		return api.RoleRef{}
	}
	r := in[0]
	return api.RoleRef{
		APIGroup: r.APIGroup.ValueString(),
		Kind:     r.Kind.ValueString(),
		Name:     r.Name.ValueString(),
	}
}

func flattenRoleRef(in api.RoleRef) []RoleRefModel {
	return []RoleRefModel{{
		APIGroup: types.StringValue(in.APIGroup),
		Kind:     types.StringValue(in.Kind),
		Name:     types.StringValue(in.Name),
	}}
}

func expandSubjects(in []SubjectModel) []api.Subject {
	subjects := make([]api.Subject, 0, len(in))
	for _, s := range in {
		subject := api.Subject{
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

func flattenSubjects(in []api.Subject) []SubjectModel {
	subjects := make([]SubjectModel, 0, len(in))
	for _, s := range in {
		model := SubjectModel{
			// SDKv2 stores an empty api_group for ServiceAccount subjects.
			APIGroup: types.StringValue(s.APIGroup),
			Kind:     types.StringValue(s.Kind),
			Name:     types.StringValue(s.Name),
		}
		if s.Namespace != "" {
			model.Namespace = types.StringValue(s.Namespace)
		} else {
			model.Namespace = types.StringValue("default")
		}
		subjects = append(subjects, model)
	}
	return subjects
}
