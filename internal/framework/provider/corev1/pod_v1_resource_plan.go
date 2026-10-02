// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (p *PodV1) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}
	var planned, prior types.List
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata"), &planned)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata"), &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if planned.IsUnknown() || prior.IsUnknown() || len(planned.Elements()) != 1 || len(prior.Elements()) != 1 {
		return
	}

	// Quantity modifiers can restore semantic equality after Framework has
	// marked metadata unknown. Restore API revisions only for an otherwise
	// identical plan; genuine updates must still receive fresh API values.
	candidate := req.Plan
	plannedMetadata := planned.Elements()[0].(types.Object).Attributes()
	priorMetadata := prior.Elements()[0].(types.Object).Attributes()
	for _, name := range []string{"generation", "resource_version"} {
		if plannedMetadata[name].IsUnknown() && !priorMetadata[name].IsUnknown() {
			resp.Diagnostics.Append(candidate.SetAttribute(ctx,
				path.Root("metadata").AtListIndex(0).AtName(name), priorMetadata[name])...)
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}
	if candidate.Raw.Equal(req.State.Raw) {
		resp.Plan = candidate
	}
}
