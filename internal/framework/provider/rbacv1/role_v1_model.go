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
