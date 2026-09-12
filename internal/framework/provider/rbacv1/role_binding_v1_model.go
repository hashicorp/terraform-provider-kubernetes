// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import "github.com/hashicorp/terraform-plugin-framework/types"

// RoleBindingModel is the top-level Terraform state model for kubernetes_role_binding_v1.
type RoleBindingModel struct {
	ID       types.String              `tfsdk:"id"`
	Metadata []NamespacedMetadataModel `tfsdk:"metadata"`
	RoleRef  []RoleRefModel            `tfsdk:"role_ref"`
	Subject  []SubjectModel            `tfsdk:"subject"`
}

// NamespacedMetadataModel holds the standard Kubernetes object metadata
// for a namespaced resource (includes Namespace, unlike cluster-scoped resources).
type NamespacedMetadataModel struct {
	Annotations     map[string]types.String `tfsdk:"annotations"`
	GenerateName    types.String            `tfsdk:"generate_name"`
	Generation      types.Int64             `tfsdk:"generation"`
	Labels          map[string]types.String `tfsdk:"labels"`
	Name            types.String            `tfsdk:"name"`
	Namespace       types.String            `tfsdk:"namespace"`
	ResourceVersion types.String            `tfsdk:"resource_version"`
	UID             types.String            `tfsdk:"uid"`
}

// RoleRefModel represents the role_ref block — which Role or ClusterRole to grant.
// All fields are ForceNew (RequiresReplace) because the Kubernetes API does not
// allow patching roleRef after creation.
type RoleRefModel struct {
	APIGroup types.String `tfsdk:"api_group"`
	Kind     types.String `tfsdk:"kind"`
	Name     types.String `tfsdk:"name"`
}

// SubjectModel represents a single subject block — who receives the permissions.
// Subjects can be a User, ServiceAccount, or Group.
type SubjectModel struct {
	APIGroup  types.String `tfsdk:"api_group"`
	Kind      types.String `tfsdk:"kind"`
	Name      types.String `tfsdk:"name"`
	Namespace types.String `tfsdk:"namespace"`
}

// RoleBindingIdentityModel is the identity schema for kubernetes_role_binding_v1.
// Namespaced resources carry api_version, kind, namespace, and name.
type RoleBindingIdentityModel struct {
	APIVersion types.String `tfsdk:"api_version"`
	Kind       types.String `tfsdk:"kind"`
	Namespace  types.String `tfsdk:"namespace"`
	Name       types.String `tfsdk:"name"`
}
