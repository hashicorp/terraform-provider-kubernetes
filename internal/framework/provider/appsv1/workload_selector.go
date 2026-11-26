// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type LabelSelectorModel struct {
	MatchExpressions []LabelSelectorRequirementModel `tfsdk:"match_expressions"`
	MatchLabels      types.Map                       `tfsdk:"match_labels"`
}

type LabelSelectorRequirementModel struct {
	Key      types.String `tfsdk:"key"`
	Operator types.String `tfsdk:"operator"`
	Values   types.Set    `tfsdk:"values"`
}

func workloadSelectorRequiresReplace() planmodifier.List {
	return listplanmodifier.RequiresReplaceIf(
		func(ctx context.Context, req planmodifier.ListRequest, resp *listplanmodifier.RequiresReplaceIfFuncResponse) {
			plan, err := req.PlanValue.ToTerraformValue(ctx)
			if err != nil {
				resp.Diagnostics.AddError("Unable to compare workload selector", err.Error())
				return
			}
			state, err := req.StateValue.ToTerraformValue(ctx)
			if err != nil {
				resp.Diagnostics.AddError("Unable to compare workload selector", err.Error())
				return
			}
			equal, err := workloadSelectorsEqual(plan, state)
			if err != nil {
				resp.Diagnostics.AddError("Unable to compare workload selector", err.Error())
				return
			}
			resp.RequiresReplace = !equal
		},
		"Replace when the selector changes, ignoring equivalent empty collection representations.",
		"Replace when the selector changes, ignoring equivalent empty collection representations.",
	)
}

func workloadSelectorsEqual(plan, state tftypes.Value) (bool, error) {
	planned, err := workloadSelectorComparisonValue(plan)
	if err != nil {
		return false, err
	}
	prior, err := workloadSelectorComparisonValue(state)
	if err != nil {
		return false, err
	}
	return planned.Equal(prior), nil
}

// Normalize copies for comparison only. Selector presence, expression order and
// nonempty values (including empty-string set/map members) remain significant.
func workloadSelectorComparisonValue(value tftypes.Value) (tftypes.Value, error) {
	return tftypes.Transform(value, func(at *tftypes.AttributePath, current tftypes.Value) (tftypes.Value, error) {
		if current.IsNull() || !current.IsKnown() {
			return current, nil
		}
		switch at.LastStep() {
		case tftypes.AttributeName("match_labels"):
			var elements map[string]tftypes.Value
			if err := current.As(&elements); err != nil {
				return current, err
			}
			if len(elements) == 0 {
				return tftypes.NewValue(current.Type(), nil), nil
			}
		case tftypes.AttributeName("match_expressions"), tftypes.AttributeName("values"):
			var elements []tftypes.Value
			if err := current.As(&elements); err != nil {
				return current, err
			}
			if len(elements) == 0 {
				return tftypes.NewValue(current.Type(), nil), nil
			}
		}
		return current, nil
	})
}

func flattenWorkloadSelector(ctx context.Context, in *metav1.LabelSelector, baseline []LabelSelectorModel) ([]LabelSelectorModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	if in == nil {
		return nil, diags
	}
	prior := LabelSelectorModel{MatchLabels: types.MapNull(types.StringType)}
	if len(baseline) == 1 {
		prior = baseline[0]
	}
	out := LabelSelectorModel{MatchLabels: types.MapNull(types.StringType)}
	if len(in.MatchLabels) > 0 {
		value, d := types.MapValueFrom(ctx, types.StringType, in.MatchLabels)
		diags.Append(d...)
		out.MatchLabels = value
	} else if !prior.MatchLabels.IsNull() && !prior.MatchLabels.IsUnknown() && len(prior.MatchLabels.Elements()) == 0 {
		out.MatchLabels = prior.MatchLabels
	}
	if len(in.MatchExpressions) > 0 || prior.MatchExpressions != nil && len(prior.MatchExpressions) == 0 {
		out.MatchExpressions = make([]LabelSelectorRequirementModel, len(in.MatchExpressions))
	}
	used := make([]bool, len(prior.MatchExpressions))
	for i, expression := range in.MatchExpressions {
		values := types.SetNull(types.StringType)
		if len(expression.Values) > 0 {
			value, d := types.SetValueFrom(ctx, types.StringType, expression.Values)
			diags.Append(d...)
			values = value
		} else {
			// Match by identity and occurrence, not API index: another expression
			// can be reordered or removed, and duplicate requirements are valid.
			for j, previous := range prior.MatchExpressions {
				if used[j] || !previous.Key.Equal(types.StringValue(expression.Key)) ||
					!previous.Operator.Equal(types.StringValue(string(expression.Operator))) ||
					previous.Values.IsUnknown() || len(previous.Values.Elements()) != 0 {
					continue
				}
				used[j] = true
				if !previous.Values.IsNull() {
					values = previous.Values
				}
				break
			}
		}
		out.MatchExpressions[i] = LabelSelectorRequirementModel{
			Key: types.StringValue(expression.Key), Operator: types.StringValue(string(expression.Operator)), Values: values,
		}
	}
	return []LabelSelectorModel{out}, diags
}
