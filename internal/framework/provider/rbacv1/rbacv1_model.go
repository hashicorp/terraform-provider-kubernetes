// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import "github.com/hashicorp/terraform-plugin-framework/types"

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
