// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package schedulingv1

import "github.com/hashicorp/terraform-plugin-framework/types"

type PriorityClassModel struct {
	ID               types.String  `tfsdk:"id"`
	Metadata         MetadataModel `tfsdk:"metadata"`
	Value            types.Int64   `tfsdk:"value"`
	Description      types.String  `tfsdk:"description"`
	GlobalDefault    types.Bool    `tfsdk:"global_default"`
	PreemptionPolicy types.String  `tfsdk:"preemption_policy"`
}

type MetadataModel struct {
	Annotations     map[string]types.String `tfsdk:"annotations"`
	GenerateName    types.String            `tfsdk:"generate_name"`
	Generation      types.Int64             `tfsdk:"generation"`
	Labels          map[string]types.String `tfsdk:"labels"`
	Name            types.String            `tfsdk:"name"`
	ResourceVersion types.String            `tfsdk:"resource_version"`
	UID             types.String            `tfsdk:"uid"`
}

type PriorityClassIdentityModel struct {
	APIVersion types.String `tfsdk:"api_version"`
	Kind       types.String `tfsdk:"kind"`
	Name       types.String `tfsdk:"name"`
}
