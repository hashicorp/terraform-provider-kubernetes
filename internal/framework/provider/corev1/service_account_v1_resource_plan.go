// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func serviceAccountModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() || req.Plan.Raw.Equal(req.State.Raw) {
		return
	}
	var metadata types.List
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata"), &metadata)...)
	if resp.Diagnostics.HasError() || metadata.IsNull() || metadata.IsUnknown() || len(metadata.Elements()) != 1 {
		return
	}
	if metadata.Elements()[0].IsNull() || metadata.Elements()[0].IsUnknown() {
		return
	}
	for _, field := range []string{"annotations", "labels"} {
		fieldPath := path.Root("metadata").AtListIndex(0).AtName(field)
		var before, after types.Map
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, fieldPath, &before)...)
		resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, fieldPath, &after)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if before.IsNull() || before.IsUnknown() || len(before.Elements()) == 0 || !after.IsNull() {
			continue
		}
		// Framework marks computed outputs unknown before attribute modifiers.
		// An omitted Optional+Computed map initially inherits its old value, so
		// clearing it later can introduce the only change after that first pass.
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("metadata").AtListIndex(0).AtName("resource_version"), types.StringUnknown())...)
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("metadata").AtListIndex(0).AtName("generation"), types.Int64Unknown())...)
		return
	}
}
