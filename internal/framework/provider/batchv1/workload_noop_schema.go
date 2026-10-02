// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// preserveWorkloadNoopPlan runs after attribute plan modifiers. Quantity
// normalization can eliminate the only configured change after Framework has
// already marked Computed fields unknown. Restore those fields only if doing so
// yields exactly the existing state, never for known null/empty changes or an
// actual update that could change server-generated values.
func preserveWorkloadNoopPlan(_ context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() || !req.Config.Raw.IsFullyKnown() || !req.State.Raw.IsFullyKnown() {
		return
	}
	resourceSchema, ok := req.Plan.Schema.(schema.Schema)
	if !ok {
		return
	}
	field := valueField{children: objectValueFields(resourceSchema.Attributes, resourceSchema.Blocks)}
	if workloadNoopValuesMatch(field, req.Plan.Raw, req.State.Raw, req.Config.Raw) {
		resp.Plan.Raw = req.State.Raw.Copy()
	}
}

func workloadNoopValuesMatch(field valueField, plan, state, config tftypes.Value) bool {
	if plan.Equal(state) {
		return true
	}
	if !plan.IsKnown() {
		return field.computed && config.IsNull() && state.IsFullyKnown()
	}
	if plan.IsNull() || state.IsNull() || !state.IsKnown() {
		return false
	}
	switch plan.Type().(type) {
	case tftypes.Object:
		var planned, prior, configured map[string]tftypes.Value
		if plan.As(&planned) != nil || state.As(&prior) != nil {
			return false
		}
		if !config.IsNull() && config.As(&configured) != nil {
			return false
		}
		if len(planned) != len(prior) {
			return false
		}
		for name, child := range planned {
			old, ok := prior[name]
			if !ok {
				return false
			}
			conf, ok := configured[name]
			if !ok {
				conf = tftypes.NewValue(child.Type(), nil)
			}
			if !workloadNoopValuesMatch(field.children[name], child, old, conf) {
				return false
			}
		}
		return true
	case tftypes.List:
		var planned, prior, configured []tftypes.Value
		if plan.As(&planned) != nil || state.As(&prior) != nil {
			return false
		}
		if !config.IsNull() && config.As(&configured) != nil {
			return false
		}
		if len(planned) != len(prior) {
			return false
		}
		for index, child := range planned {
			conf := tftypes.NewValue(child.Type(), nil)
			if index < len(configured) {
				conf = configured[index]
			}
			if !workloadNoopValuesMatch(valueField{children: field.children}, child, prior[index], conf) {
				return false
			}
		}
		return true
	}
	return false
}
