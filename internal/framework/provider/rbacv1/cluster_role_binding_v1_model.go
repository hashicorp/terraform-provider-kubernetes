// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ClusterRoleBindingModel maps the kubernetes_cluster_role_binding_v1 schema.
//
// metadata, role_ref and subject are modeled as slices because they are
// represented as blocks (ListNestedBlock) in the schema, exactly mirroring the
// SDKv2 TypeList representation. metadata and role_ref are constrained to a
// single element by schema validators.
type ClusterRoleBindingModel struct {
	ID       types.String                      `tfsdk:"id"`
	Metadata []ClusterRoleBindingMetadataModel `tfsdk:"metadata"`
	RoleRef  []RoleRefModel                    `tfsdk:"role_ref"`
	Subject  []SubjectModel                    `tfsdk:"subject"`
}

// ClusterRoleBindingMetadataModel is the non-namespaced metadata block. It has
// no namespace field because a ClusterRoleBinding is a cluster-scoped resource.
type ClusterRoleBindingMetadataModel struct {
	Annotations     types.Map    `tfsdk:"annotations"`
	GenerateName    types.String `tfsdk:"generate_name"`
	Generation      types.Int64  `tfsdk:"generation"`
	Labels          types.Map    `tfsdk:"labels"`
	Name            types.String `tfsdk:"name"`
	ResourceVersion types.String `tfsdk:"resource_version"`
	UID             types.String `tfsdk:"uid"`
}

type RoleRefModel struct {
	APIGroup types.String `tfsdk:"api_group"`
	Kind     types.String `tfsdk:"kind"`
	Name     types.String `tfsdk:"name"`
}

type SubjectModel struct {
	APIGroup  types.String `tfsdk:"api_group"`
	Kind      types.String `tfsdk:"kind"`
	Name      types.String `tfsdk:"name"`
	Namespace types.String `tfsdk:"namespace"`
}

type ClusterRoleBindingIdentityModel struct {
	APIVersion types.String `tfsdk:"api_version"`
	Kind       types.String `tfsdk:"kind"`
	Name       types.String `tfsdk:"name"`
}
