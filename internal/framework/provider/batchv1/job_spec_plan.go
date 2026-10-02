// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	batchapi "k8s.io/api/batch/v1"
)

func jobSpecInt64RequiresReplace() planmodifier.Int64 {
	return int64planmodifier.RequiresReplaceIf(func(ctx context.Context, req planmodifier.Int64Request, resp *int64planmodifier.RequiresReplaceIfFuncResponse) {
		resp.RequiresReplace = workloadPlannedChange(ctx, req.PlanValue, req.StateValue, req.ConfigValue)
		if _, safe := workloadEmptyNormalizationValue(ctx, req.PlanValue, req.StateValue, req.ConfigValue); safe &&
			jobSpecPayloadEquivalent(ctx, req.Plan, req.State, req.Config) {
			resp.RequiresReplace = false
		}
	}, workloadReplacementDescription, workloadReplacementDescription)
}

func jobSpecPolicyRequiresReplace() planmodifier.List {
	return listplanmodifier.RequiresReplaceIf(func(ctx context.Context, req planmodifier.ListRequest, resp *listplanmodifier.RequiresReplaceIfFuncResponse) {
		resp.RequiresReplace = workloadPlannedChange(ctx, req.PlanValue, req.StateValue, req.ConfigValue)
		if _, safe := workloadEmptyNormalizationValue(ctx, req.PlanValue, req.StateValue, req.ConfigValue); safe &&
			jobSpecPayloadEquivalent(ctx, req.Plan, req.State, req.Config) {
			resp.RequiresReplace = false
		}
	}, workloadReplacementDescription, workloadReplacementDescription)
}

func jobSpecStringRequiresReplace() planmodifier.String {
	return stringplanmodifier.RequiresReplaceIf(func(ctx context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
		resp.RequiresReplace = workloadPlannedChange(ctx, req.PlanValue, req.StateValue, req.ConfigValue)
		if _, safe := workloadEmptyNormalizationValue(ctx, req.PlanValue, req.StateValue, req.ConfigValue); safe &&
			jobSpecPayloadEquivalent(ctx, req.Plan, req.State, req.Config) {
			resp.RequiresReplace = false
		}
	}, workloadReplacementDescription, workloadReplacementDescription)
}

func jobSpecMapRequiresReplace() planmodifier.Map {
	return mapplanmodifier.RequiresReplaceIf(func(ctx context.Context, req planmodifier.MapRequest, resp *mapplanmodifier.RequiresReplaceIfFuncResponse) {
		resp.RequiresReplace = workloadPlannedChange(ctx, req.PlanValue, req.StateValue, req.ConfigValue)
		if _, safe := workloadEmptyNormalizationValue(ctx, req.PlanValue, req.StateValue, req.ConfigValue); safe &&
			jobSpecPayloadEquivalent(ctx, req.Plan, req.State, req.Config) {
			resp.RequiresReplace = false
		}
	}, workloadReplacementDescription, workloadReplacementDescription)
}

func jobSpecSetRequiresReplace() planmodifier.Set {
	return setplanmodifier.RequiresReplaceIf(func(ctx context.Context, req planmodifier.SetRequest, resp *setplanmodifier.RequiresReplaceIfFuncResponse) {
		resp.RequiresReplace = workloadPlannedChange(ctx, req.PlanValue, req.StateValue, req.ConfigValue)
		if _, safe := workloadEmptyNormalizationValue(ctx, req.PlanValue, req.StateValue, req.ConfigValue); safe &&
			jobSpecPayloadEquivalent(ctx, req.Plan, req.State, req.Config) {
			resp.RequiresReplace = false
		}
	}, workloadReplacementDescription, workloadReplacementDescription)
}

// jobSpecPayloadEquivalent distinguishes an SDKv2 empty-value representation
// change from a changed immutable Kubernetes payload. The Terraform plan is
// never rewritten. Configured unknowns cannot establish payload equivalence.
func jobSpecPayloadEquivalent(ctx context.Context, plan tfsdk.Plan, state tfsdk.State, config tfsdk.Config) bool {
	if plan.Raw.IsNull() || state.Raw.IsNull() {
		return false
	}
	root, ok := plan.Raw.Type().(tftypes.Object)
	if !ok {
		return false
	}
	specPath := path.Root("spec")
	if _, job := root.AttributeTypes["wait_for_completion"]; !job {
		specPath = specPath.AtListIndex(0).AtName("job_template").AtListIndex(0).AtName("spec")
	}
	var planned, previous, configured types.List
	if plan.GetAttribute(ctx, specPath, &planned).HasError() ||
		state.GetAttribute(ctx, specPath, &previous).HasError() ||
		config.GetAttribute(ctx, specPath, &configured).HasError() {
		return false
	}
	configRaw, err := configured.ToTerraformValue(ctx)
	if err != nil || !configRaw.IsFullyKnown() {
		return false
	}
	planRaw, err := planned.ToTerraformValue(ctx)
	if err != nil {
		return false
	}
	stateRaw, err := previous.ToTerraformValue(ctx)
	if err != nil {
		return false
	}
	resolved, ok := jobComparisonValue(jobSpecValueField(), planRaw, stateRaw, configRaw)
	if !ok || !resolved.IsFullyKnown() {
		return false
	}
	resolvedValue, err := jobSpecValueField().typ.ValueFromTerraform(ctx, resolved)
	if err != nil {
		return false
	}
	list, ok := resolvedValue.(types.List)
	if !ok {
		return false
	}
	before, beforeDiags := expandJobSpec(ctx, previous)
	after, afterDiags := expandJobSpec(ctx, list)
	if beforeDiags.HasError() || afterDiags.HasError() {
		return false
	}
	oldPayload, err := json.Marshal(jobImmutableSpec(before))
	if err != nil {
		return false
	}
	newPayload, err := json.Marshal(jobImmutableSpec(after))
	return err == nil && bytes.Equal(oldPayload, newPayload)
}

func jobImmutableSpec(spec batchapi.JobSpec) batchapi.JobSpec {
	spec.ActiveDeadlineSeconds = nil
	spec.BackoffLimit = nil
	spec.ManualSelector = nil
	spec.MaxFailedIndexes = nil
	spec.Parallelism = nil
	spec.TTLSecondsAfterFinished = nil
	return spec
}

func jobComparisonValue(field valueField, plan, state, config tftypes.Value) (tftypes.Value, bool) {
	if !config.IsKnown() {
		return tftypes.Value{}, false
	}
	if !plan.IsKnown() {
		if field.computed && config.IsNull() && state.IsFullyKnown() {
			return state, true
		}
		return tftypes.Value{}, false
	}
	if plan.IsNull() {
		return plan, true
	}
	switch typ := plan.Type().(type) {
	case tftypes.Object:
		var planned, previous, configured map[string]tftypes.Value
		if plan.As(&planned) != nil {
			return tftypes.Value{}, false
		}
		if !state.IsNull() && state.As(&previous) != nil {
			return tftypes.Value{}, false
		}
		if !config.IsNull() && config.As(&configured) != nil {
			return tftypes.Value{}, false
		}
		result := make(map[string]tftypes.Value, len(planned))
		for name, value := range planned {
			prior, found := previous[name]
			if !found {
				prior = tftypes.NewValue(value.Type(), nil)
			}
			conf, found := configured[name]
			if !found {
				conf = tftypes.NewValue(value.Type(), nil)
			}
			if (name == "container" || name == "init_container") && value.IsKnown() && !value.IsNull() {
				var ok bool
				prior, ok = jobComparisonContainerState(value, prior)
				if !ok {
					return tftypes.Value{}, false
				}
			}
			resolved, ok := jobComparisonValue(field.children[name], value, prior, conf)
			if !ok {
				return tftypes.Value{}, false
			}
			result[name] = resolved
		}
		return tftypes.NewValue(typ, result), true
	case tftypes.List:
		var planned, previous, configured []tftypes.Value
		if plan.As(&planned) != nil {
			return tftypes.Value{}, false
		}
		if !state.IsNull() && state.As(&previous) != nil {
			return tftypes.Value{}, false
		}
		if !config.IsNull() && config.As(&configured) != nil {
			return tftypes.Value{}, false
		}
		result := make([]tftypes.Value, len(planned))
		for i, value := range planned {
			prior, conf := tftypes.NewValue(value.Type(), nil), tftypes.NewValue(value.Type(), nil)
			if i < len(previous) {
				prior = previous[i]
			}
			if i < len(configured) {
				conf = configured[i]
			}
			resolved, ok := jobComparisonValue(valueField{children: field.children}, value, prior, conf)
			if !ok {
				return tftypes.Value{}, false
			}
			result[i] = resolved
		}
		return tftypes.NewValue(typ, result), true
	default:
		return plan, plan.IsFullyKnown()
	}
}

// Container identity is its name, not its position in the Terraform list.
// Align only the comparison state; new names deliberately have no prior values.
func jobComparisonContainerState(plan, state tftypes.Value) (tftypes.Value, bool) {
	var planned, previous []tftypes.Value
	if plan.As(&planned) != nil || (!state.IsNull() && state.As(&previous) != nil) {
		return tftypes.Value{}, false
	}
	containerName := func(value tftypes.Value) (string, bool) {
		var fields map[string]tftypes.Value
		if !value.IsKnown() || value.IsNull() || value.As(&fields) != nil {
			return "", false
		}
		name, ok := fields["name"]
		if !ok || !name.IsKnown() || name.IsNull() {
			return "", false
		}
		var result string
		if name.As(&result) != nil || result == "" {
			return "", false
		}
		return result, true
	}
	byName := make(map[string]tftypes.Value, len(previous))
	for _, value := range previous {
		name, ok := containerName(value)
		if !ok {
			return tftypes.Value{}, false
		}
		if _, duplicate := byName[name]; duplicate {
			return tftypes.Value{}, false
		}
		byName[name] = value
	}
	aligned := make([]tftypes.Value, len(planned))
	seen := make(map[string]struct{}, len(planned))
	for i, value := range planned {
		name, ok := containerName(value)
		if !ok {
			return tftypes.Value{}, false
		}
		if _, duplicate := seen[name]; duplicate {
			return tftypes.Value{}, false
		}
		seen[name] = struct{}{}
		prior, found := byName[name]
		if !found {
			prior = tftypes.NewValue(value.Type(), nil)
		}
		aligned[i] = prior
	}
	return tftypes.NewValue(plan.Type(), aligned), true
}
