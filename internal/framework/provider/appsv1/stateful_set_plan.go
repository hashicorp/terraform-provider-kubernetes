// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var _ resource.ResourceWithModifyPlan = (*StatefulSetV1)(nil)

func (r *StatefulSetV1) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	candidate, unchanged, err := workloadNoOpPlan(req.Config.Raw, req.Plan.Raw, req.State.Raw)
	if err != nil {
		resp.Diagnostics.AddError("Unable to compare StatefulSet plan", err.Error())
		return
	}
	// Resource versions and generations can change on updates. Only retain them
	// once every configured field and every known planned value proves a no-op.
	if unchanged {
		resp.Plan.Raw = candidate
	}
}

type statefulSetVolumeClaimRequiresReplace struct{}

func (statefulSetVolumeClaimRequiresReplace) Description(context.Context) string {
	return "Changes to volume claim templates require replacement."
}

func (m statefulSetVolumeClaimRequiresReplace) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (statefulSetVolumeClaimRequiresReplace) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() || req.PlanValue.IsUnknown() || req.PlanValue.Equal(req.StateValue) {
		return
	}
	plan, err := req.PlanValue.ToTerraformValue(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to compare volume claim templates", err.Error())
		return
	}
	state, err := req.StateValue.ToTerraformValue(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to compare volume claim templates", err.Error())
		return
	}
	config, err := req.ConfigValue.ToTerraformValue(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to compare volume claim templates", err.Error())
		return
	}
	comparisonState, err := statefulSetClaimComparisonState(plan, state)
	if err != nil {
		resp.Diagnostics.AddError("Unable to compare volume claim templates", err.Error())
		return
	}
	_, unchanged, err := workloadNoOpPlan(config, plan, comparisonState)
	if err != nil {
		resp.Diagnostics.AddError("Unable to compare volume claim templates", err.Error())
		return
	}
	resp.RequiresReplace = !unchanged
}

// SDKv2 persisted some unset fields as empty values. Normalize those only for
// replacement comparison; the configured plan and the persisted state are not
// rewritten, and removing nonempty metadata must still replace the StatefulSet.
func statefulSetClaimComparisonState(plan, state tftypes.Value) (tftypes.Value, error) {
	return tftypes.Transform(state, func(at *tftypes.AttributePath, value tftypes.Value) (tftypes.Value, error) {
		if value.IsNull() || !value.IsKnown() {
			return value, nil
		}
		planned, _, err := tftypes.WalkAttributePath(plan, at)
		if err != nil {
			return value, nil
		}
		plannedValue := planned.(tftypes.Value)
		if at.LastStep() == tftypes.AttributeName("selector") {
			equal, err := workloadSelectorsEqual(plannedValue, value)
			if err != nil {
				return value, err
			}
			if equal {
				return plannedValue, nil
			}
		}
		if !plannedValue.IsNull() {
			if _, quantityEntry := at.LastStep().(tftypes.ElementKeyString); quantityEntry && plannedValue.IsKnown() {
				switch at.WithoutLastStep().LastStep() {
				case tftypes.AttributeName("limits"), tftypes.AttributeName("requests"):
					var before, after string
					if err := value.As(&before); err != nil {
						return value, err
					}
					if err := plannedValue.As(&after); err != nil {
						return value, err
					}
					if statefulSetQuantitiesEqual(types.StringValue(before), types.StringValue(after)) {
						return plannedValue, nil
					}
				}
			}
			return value, nil
		}
		switch at.LastStep() {
		case tftypes.AttributeName("annotations"), tftypes.AttributeName("labels"),
			tftypes.AttributeName("limits"), tftypes.AttributeName("requests"),
			tftypes.AttributeName("match_labels"):
			var elements map[string]tftypes.Value
			if err := value.As(&elements); err != nil {
				return value, err
			}
			if len(elements) == 0 {
				return tftypes.NewValue(value.Type(), nil), nil
			}
		case tftypes.AttributeName("generate_name"):
			var name string
			if err := value.As(&name); err != nil {
				return value, err
			}
			if name == "" {
				return tftypes.NewValue(value.Type(), nil), nil
			}
		}
		return value, nil
	})
}
