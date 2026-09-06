// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	api "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// expandStringMap converts a Terraform types.Map into a map[string]string.
// A null/unknown map yields a nil map (so it is omitted from the API object).
func expandStringMap(ctx context.Context, m types.Map) (map[string]string, diag.Diagnostics) {
	if m.IsNull() || m.IsUnknown() {
		return nil, nil
	}
	result := make(map[string]string, len(m.Elements()))
	diags := m.ElementsAs(ctx, &result, false)
	if len(result) == 0 {
		return nil, diags
	}
	return result, diags
}

// flattenStringMap converts a map[string]string into a Terraform types.Map.
// An empty map is represented as a null map to match the SDKv2 behaviour of
// omitting empty metadata maps from state.
func flattenStringMap(m map[string]string) types.Map {
	if len(m) == 0 {
		return types.MapNull(types.StringType)
	}
	elements := make(map[string]interface{}, len(m))
	for k, v := range m {
		elements[k] = v
	}
	result, _ := types.MapValueFrom(context.Background(), types.StringType, elements)
	return result
}

// expandMetadata builds a Kubernetes ObjectMeta from the (single) metadata block.
func expandMetadata(ctx context.Context, in []ClusterRoleBindingMetadataModel) (metav1.ObjectMeta, diag.Diagnostics) {
	var diags diag.Diagnostics
	meta := metav1.ObjectMeta{}
	if len(in) == 0 {
		return meta, diags
	}
	m := in[0]

	annotations, d := expandStringMap(ctx, m.Annotations)
	diags.Append(d...)
	meta.Annotations = annotations

	labels, d := expandStringMap(ctx, m.Labels)
	diags.Append(d...)
	meta.Labels = labels

	if !m.GenerateName.IsNull() && !m.GenerateName.IsUnknown() {
		meta.GenerateName = m.GenerateName.ValueString()
	}
	if !m.Name.IsNull() && !m.Name.IsUnknown() {
		meta.Name = m.Name.ValueString()
	}

	return meta, diags
}

// flattenMetadata produces the metadata block from a Kubernetes ObjectMeta.
func flattenMetadata(meta metav1.ObjectMeta) []ClusterRoleBindingMetadataModel {
	model := ClusterRoleBindingMetadataModel{
		Annotations:     flattenStringMap(meta.Annotations),
		Labels:          flattenStringMap(meta.Labels),
		Name:            types.StringValue(meta.Name),
		Generation:      types.Int64Value(meta.Generation),
		ResourceVersion: types.StringValue(meta.ResourceVersion),
		UID:             types.StringValue(string(meta.UID)),
	}
	if meta.GenerateName != "" {
		model.GenerateName = types.StringValue(meta.GenerateName)
	} else {
		model.GenerateName = types.StringNull()
	}
	return []ClusterRoleBindingMetadataModel{model}
}

// expandRoleRef builds a Kubernetes RoleRef from the (single) role_ref block.
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

// flattenRoleRef produces the role_ref block from a Kubernetes RoleRef.
func flattenRoleRef(in api.RoleRef) []RoleRefModel {
	return []RoleRefModel{{
		APIGroup: types.StringValue(in.APIGroup),
		Kind:     types.StringValue(in.Kind),
		Name:     types.StringValue(in.Name),
	}}
}

// expandSubjects builds the list of Kubernetes Subjects from the subject blocks.
func expandSubjects(in []SubjectModel) []api.Subject {
	if len(in) == 0 {
		return []api.Subject{}
	}
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

// flattenSubjects produces the subject blocks from the list of Kubernetes Subjects.
func flattenSubjects(in []api.Subject) []SubjectModel {
	subjects := make([]SubjectModel, 0, len(in))
	for _, s := range in {
		model := SubjectModel{
			// api_group is Optional+Computed; store the concrete value
			// (empty for ServiceAccount subjects) to mirror the SDKv2 state.
			APIGroup: types.StringValue(s.APIGroup),
			Kind:     types.StringValue(s.Kind),
			Name:     types.StringValue(s.Name),
		}
		if s.Namespace != "" {
			model.Namespace = types.StringValue(s.Namespace)
		} else {
			// Preserve the SDKv2 "default" default so cluster-scoped
			// (User/Group) subjects that omit a namespace do not drift.
			model.Namespace = types.StringValue("default")
		}
		subjects = append(subjects, model)
	}
	return subjects
}
