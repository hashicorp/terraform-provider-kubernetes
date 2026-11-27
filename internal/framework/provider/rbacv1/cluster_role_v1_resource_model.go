// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

// ClusterRoleModel maps the kubernetes_cluster_role_v1 schema. metadata, rule and
// aggregation_rule are blocks, so they are slices, mirroring the SDKv2 TypeList shape.
type ClusterRoleModel struct {
	ID              types.String           `tfsdk:"id"`
	Metadata        []common.MetadataModel `tfsdk:"metadata"`
	Rule            []ClusterRoleRuleModel `tfsdk:"rule"`
	AggregationRule []AggregationRuleModel `tfsdk:"aggregation_rule"`
}

type ClusterRoleRuleModel struct {
	APIGroups       types.Set `tfsdk:"api_groups"`
	NonResourceURLs types.Set `tfsdk:"non_resource_urls"`
	ResourceNames   types.Set `tfsdk:"resource_names"`
	Resources       types.Set `tfsdk:"resources"`
	Verbs           types.Set `tfsdk:"verbs"`
}

type AggregationRuleModel struct {
	ClusterRoleSelectors []ClusterRoleSelectorModel `tfsdk:"cluster_role_selectors"`
}

type ClusterRoleSelectorModel struct {
	MatchLabels      types.Map              `tfsdk:"match_labels"`
	MatchExpressions []MatchExpressionModel `tfsdk:"match_expressions"`
}

type MatchExpressionModel struct {
	Key      types.String `tfsdk:"key"`
	Operator types.String `tfsdk:"operator"`
	Values   types.Set    `tfsdk:"values"`
}

// sdkv2ClusterRoleStateV0 is kubernetes_cluster_role state as SDKv2 stores it at schema
// version 0. It is frozen: it describes stored JSON, not this resource's schema, so
// MoveState keeps working if the framework schema changes. A future SDKv2 schema version
// needs its own struct, not edits to this one.
//
// SDKv2 has no null for primitives, so an unset string is "". Lists and maps stay Go
// slices and maps because SDKv2 keeps null and empty apart for them.
type sdkv2ClusterRoleStateV0 struct {
	ID              string                          `json:"id"`
	Metadata        []sdkv2ClusterRoleMetadataV0    `json:"metadata"`
	Rule            []sdkv2ClusterRolePolicyRuleV0  `json:"rule"`
	AggregationRule []sdkv2ClusterRoleAggregationV0 `json:"aggregation_rule"`
}

type sdkv2ClusterRoleMetadataV0 struct {
	Annotations     map[string]string `json:"annotations"`
	GenerateName    string            `json:"generate_name"`
	Generation      int64             `json:"generation"`
	Labels          map[string]string `json:"labels"`
	Name            string            `json:"name"`
	ResourceVersion string            `json:"resource_version"`
	UID             string            `json:"uid"`
}

type sdkv2ClusterRolePolicyRuleV0 struct {
	APIGroups       []string `json:"api_groups"`
	NonResourceURLs []string `json:"non_resource_urls"`
	ResourceNames   []string `json:"resource_names"`
	Resources       []string `json:"resources"`
	Verbs           []string `json:"verbs"`
}

type sdkv2ClusterRoleAggregationV0 struct {
	ClusterRoleSelectors []sdkv2ClusterRoleSelectorV0 `json:"cluster_role_selectors"`
}

type sdkv2ClusterRoleSelectorV0 struct {
	MatchLabels      map[string]string              `json:"match_labels"`
	MatchExpressions []sdkv2ClusterRoleExpressionV0 `json:"match_expressions"`
}

type sdkv2ClusterRoleExpressionV0 struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values"`
}
