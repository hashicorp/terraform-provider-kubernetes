// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

// RoleBindingModel is the top-level Terraform state model for kubernetes_role_binding_v1.
type RoleBindingModel struct {
	ID       types.String                     `tfsdk:"id"`
	Metadata []common.NamespacedMetadataModel `tfsdk:"metadata"`
	RoleRef  []RoleRefModel                   `tfsdk:"role_ref"`
	Subject  []SubjectModel                   `tfsdk:"subject"`
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

// These raw shapes describe the frozen SDKv2 schema-version-0 state.
type sdkv2RBMetadataElement struct {
	Name            string            `json:"name"`
	GenerateName    string            `json:"generate_name"`
	Namespace       string            `json:"namespace"`
	Annotations     map[string]string `json:"annotations"`
	Labels          map[string]string `json:"labels"`
	ResourceVersion string            `json:"resource_version"`
	UID             string            `json:"uid"`
	Generation      int64             `json:"generation"`
}

type sdkv2RBRoleRefElement struct {
	APIGroup string `json:"api_group"`
	Kind     string `json:"kind"`
	Name     string `json:"name"`
}

type sdkv2RBSubjectElement struct {
	APIGroup  string `json:"api_group"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type sdkv2RoleBindingStateV0 struct {
	ID       string                   `json:"id"`
	Metadata []sdkv2RBMetadataElement `json:"metadata"`
	RoleRef  []sdkv2RBRoleRefElement  `json:"role_ref"`
	Subjects []sdkv2RBSubjectElement  `json:"subject"`
}
