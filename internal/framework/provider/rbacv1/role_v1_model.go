// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type RoleModel struct {
	ID       types.String    `tfsdk:"id"`
	Metadata []MetadataModel `tfsdk:"metadata"`
	Rule     []RuleModel     `tfsdk:"rule"`
}

type MetadataModel struct {
	Annotations     map[string]types.String `tfsdk:"annotations"`
	GenerateName    types.String            `tfsdk:"generate_name"`
	Generation      types.Int64             `tfsdk:"generation"`
	Labels          map[string]types.String `tfsdk:"labels"`
	Name            types.String            `tfsdk:"name"`
	Namespace       types.String            `tfsdk:"namespace"`
	ResourceVersion types.String            `tfsdk:"resource_version"`
	UID             types.String            `tfsdk:"uid"`
}

type RuleModel struct {
	APIGroups     types.Set `tfsdk:"api_groups"`
	Resources     types.Set `tfsdk:"resources"`
	ResourceNames types.Set `tfsdk:"resource_names"`
	Verbs         types.Set `tfsdk:"verbs"`
}

type RoleIdentityModel struct {
	APIVersion types.String `tfsdk:"api_version"`
	Kind       types.String `tfsdk:"kind"`
	Namespace  types.String `tfsdk:"namespace"`
	Name       types.String `tfsdk:"name"`
}

type RoleSourceState struct {
	ID       string                 `json:"id"`
	Metadata []roleSourceMetadata   `json:"metadata"`
	Rule     []roleSourcePolicyRule `json:"rule"`
}

type roleSourceMetadata struct {
	Annotations     map[string]string `json:"annotations"`
	GenerateName    string            `json:"generate_name"`
	Generation      int64             `json:"generation"`
	Labels          map[string]string `json:"labels"`
	Name            string            `json:"name"`
	Namespace       string            `json:"namespace"`
	ResourceVersion string            `json:"resource_version"`
	UID             string            `json:"uid"`
}

type roleSourcePolicyRule struct {
	APIGroups     []string `json:"api_groups"`
	Resources     []string `json:"resources"`
	ResourceNames []string `json:"resource_names"`
	Verbs         []string `json:"verbs"`
}
