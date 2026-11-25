// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

// ClusterRoleBindingModel maps the kubernetes_cluster_role_binding_v1 schema.
//
// metadata, role_ref and subject are modeled as slices because they are
// represented as blocks (ListNestedBlock) in the schema, exactly mirroring the
// SDKv2 TypeList representation. metadata and role_ref are constrained to a
// single element by schema validators.
type ClusterRoleBindingModel struct {
	ID       types.String           `tfsdk:"id"`
	Metadata []common.MetadataModel `tfsdk:"metadata"`
	RoleRef  []RoleRefModel         `tfsdk:"role_ref"`
	Subject  []SubjectModel         `tfsdk:"subject"`
}

// sdkv2ClusterRoleBindingStateV0 is kubernetes_cluster_role_binding state as
// SDKv2 stores it at schema version 0 (resourceKubernetesClusterRoleBindingV1).
// It is frozen: it describes stored JSON, not this resource's schema, so
// MoveState keeps working if the framework schema changes. A future SDKv2
// schema version needs its own struct, not edits to this one.
//
// SDKv2 has no null for primitives, so an unset string is "". Maps stay Go maps
// because SDKv2 does keep null and {} apart for them.
type sdkv2ClusterRoleBindingStateV0 struct {
	ID       string                              `json:"id"`
	Metadata []sdkv2ClusterRoleBindingMetadataV0 `json:"metadata"`
	RoleRef  []sdkv2RoleRefV0                    `json:"role_ref"`
	Subject  []sdkv2SubjectV0                    `json:"subject"`
}

type sdkv2ClusterRoleBindingMetadataV0 struct {
	Annotations     map[string]string `json:"annotations"`
	GenerateName    string            `json:"generate_name"`
	Generation      int64             `json:"generation"`
	Labels          map[string]string `json:"labels"`
	Name            string            `json:"name"`
	ResourceVersion string            `json:"resource_version"`
	UID             string            `json:"uid"`
}

type sdkv2RoleRefV0 struct {
	APIGroup string `json:"api_group"`
	Kind     string `json:"kind"`
	Name     string `json:"name"`
}

type sdkv2SubjectV0 struct {
	APIGroup  string `json:"api_group"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}
