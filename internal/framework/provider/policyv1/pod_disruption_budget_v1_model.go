// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package policyv1

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	policy "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type podDisruptionBudgetV1Model struct {
	ID       types.String                     `tfsdk:"id"`
	Metadata []common.NamespacedMetadataModel `tfsdk:"metadata"`
	Spec     []podDisruptionBudgetSpecModel   `tfsdk:"spec"`
}

type podDisruptionBudgetSpecModel struct {
	MinAvailable   types.String           `tfsdk:"min_available"`
	MaxUnavailable types.String           `tfsdk:"max_unavailable"`
	Selector       *pdbLabelSelectorModel `tfsdk:"selector"`
}

type pdbLabelSelectorModel struct {
	MatchLabels      types.Map                  `tfsdk:"match_labels"`
	MatchExpressions []pdbLabelRequirementModel `tfsdk:"match_expressions"`
}

type pdbLabelRequirementModel struct {
	Key      types.String `tfsdk:"key"`
	Operator types.String `tfsdk:"operator"`
	Values   types.Set    `tfsdk:"values"`
}

func expandPodDisruptionBudgetSpec(ctx context.Context, in []podDisruptionBudgetSpecModel) (policy.PodDisruptionBudgetSpec, diag.Diagnostics) {
	var diags diag.Diagnostics
	var out policy.PodDisruptionBudgetSpec
	if len(in) != 1 || in[0].Selector == nil {
		diags.AddError("Invalid pod disruption budget spec", "Exactly one spec block and a non-null selector object are required.")
		return out, diags
	}
	expandThreshold := func(value types.String) *intstr.IntOrString {
		if value.IsUnknown() {
			diags.AddError("Unknown pod disruption budget threshold", "Thresholds must be known before creating the pod disruption budget.")
			return nil
		}
		if value.IsNull() || value.ValueString() == "" {
			return nil
		}
		if err := validateNullableIntOrPercent(value.ValueString()); err != nil {
			diags.AddError("Invalid integer or percentage", err.Error())
			return nil
		}
		parsed := intstr.Parse(value.ValueString())
		return &parsed
	}
	out.MinAvailable = expandThreshold(in[0].MinAvailable)
	out.MaxUnavailable = expandThreshold(in[0].MaxUnavailable)
	selector := in[0].Selector
	// An empty policy/v1 selector matches all pods; it must not become nil.
	out.Selector = &metav1.LabelSelector{}
	diags.Append(selector.MatchLabels.ElementsAs(ctx, &out.Selector.MatchLabels, false)...)
	for _, expression := range selector.MatchExpressions {
		if expression.Key.IsUnknown() || expression.Operator.IsUnknown() {
			diags.AddError("Unknown selector expression", "Selector expression keys and operators must be known before creating the pod disruption budget.")
			continue
		}
		requirement := metav1.LabelSelectorRequirement{
			Key: expression.Key.ValueString(), Operator: metav1.LabelSelectorOperator(expression.Operator.ValueString()),
		}
		diags.Append(expression.Values.ElementsAs(ctx, &requirement.Values, false)...)
		out.Selector.MatchExpressions = append(out.Selector.MatchExpressions, requirement)
	}
	return out, diags
}

func flattenPodDisruptionBudgetSpec(ctx context.Context, in policy.PodDisruptionBudgetSpec, prior []podDisruptionBudgetSpecModel) ([]podDisruptionBudgetSpecModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var old podDisruptionBudgetSpecModel
	if len(prior) > 0 {
		old = prior[0]
	}
	out := podDisruptionBudgetSpecModel{
		MinAvailable:   flattenPDBThreshold(in.MinAvailable, old.MinAvailable),
		MaxUnavailable: flattenPDBThreshold(in.MaxUnavailable, old.MaxUnavailable),
		Selector:       nil,
	}
	if in.Selector != nil {
		var previous pdbLabelSelectorModel
		if old.Selector != nil {
			previous = *old.Selector
		}
		selector := pdbLabelSelectorModel{}
		// Preserve an explicitly empty expression list; omitted attributes stay null.
		if previous.MatchExpressions != nil {
			selector.MatchExpressions = []pdbLabelRequirementModel{}
		}
		selector.MatchLabels = types.MapNull(types.StringType)
		if len(in.Selector.MatchLabels) > 0 || !previous.MatchLabels.IsNull() {
			labels := in.Selector.MatchLabels
			if labels == nil {
				labels = map[string]string{}
			}
			var d diag.Diagnostics
			selector.MatchLabels, d = types.MapValueFrom(ctx, types.StringType, labels)
			diags.Append(d...)
		}
		for i, expression := range in.Selector.MatchExpressions {
			values := types.SetNull(types.StringType)
			var previousValues types.Set
			if i < len(previous.MatchExpressions) {
				previousValues = previous.MatchExpressions[i].Values
			}
			if len(expression.Values) > 0 || !previousValues.IsNull() {
				v := expression.Values
				if v == nil {
					v = []string{}
				}
				var d diag.Diagnostics
				values, d = types.SetValueFrom(ctx, types.StringType, v)
				diags.Append(d...)
			}
			selector.MatchExpressions = append(selector.MatchExpressions, pdbLabelRequirementModel{
				Key: types.StringValue(expression.Key), Operator: types.StringValue(string(expression.Operator)), Values: values,
			})
		}
		out.Selector = &selector
	}
	return []podDisruptionBudgetSpecModel{out}, diags
}

func flattenPDBThreshold(value *intstr.IntOrString, prior types.String) types.String {
	if value == nil {
		return types.StringValue("")
	}
	// The API canonicalizes integer spellings (e.g. +01 to 1). Keep the prior
	// known spelling only for equal int32 values; percentages remain exact strings.
	if value.Type == intstr.Int && !prior.IsNull() && !prior.IsUnknown() {
		if n, err := strconv.ParseInt(prior.ValueString(), 10, 32); err == nil && n == int64(value.IntVal) {
			return prior
		}
	}
	return types.StringValue(value.String())
}
