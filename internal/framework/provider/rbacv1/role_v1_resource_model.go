// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

type RoleModel struct {
	ID       types.String                     `tfsdk:"id"`
	Metadata []common.NamespacedMetadataModel `tfsdk:"metadata"`
	Rule     []RuleModel                      `tfsdk:"rule"`
}

type RuleModel struct {
	APIGroups     types.Set `tfsdk:"api_groups"`
	Resources     types.Set `tfsdk:"resources"`
	ResourceNames types.Set `tfsdk:"resource_names"`
	Verbs         types.Set `tfsdk:"verbs"`
}

type sdkv2RoleStateV0 struct {
	ID       string                             `json:"id"`
	Metadata []common.SDKv2NamespacedMetadataV0 `json:"metadata"`
	Rule     []sdkv2RolePolicyRuleV0            `json:"rule"`
}

type sdkv2RolePolicyRuleV0 struct {
	APIGroups     []string `json:"api_groups"`
	Resources     []string `json:"resources"`
	ResourceNames []string `json:"resource_names"`
	Verbs         []string `json:"verbs"`
}
