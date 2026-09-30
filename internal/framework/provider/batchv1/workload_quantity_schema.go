// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	_ basetypes.StringTypable                    = workloadQuantityType{}
	_ basetypes.StringValuableWithSemanticEquals = workloadQuantityValue{}
)

// Quantity values retain the configured spelling when the API normalizes an
// equivalent quantity, matching the SDKv2 quantity DiffSuppressFunc.
type workloadQuantityType struct {
	basetypes.StringType
}

func (workloadQuantityType) Equal(other attr.Type) bool {
	_, ok := other.(workloadQuantityType)
	return ok
}

func (workloadQuantityType) String() string {
	return "batchv1.workloadQuantityType"
}

func (workloadQuantityType) ValueFromString(_ context.Context, value basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return workloadQuantityValue{StringValue: value}, nil
}

func (t workloadQuantityType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	converted, err := t.StringType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}
	return workloadQuantityValue{StringValue: converted.(basetypes.StringValue)}, nil
}

func (workloadQuantityType) ValueType(context.Context) attr.Value {
	return workloadQuantityValue{}
}

type workloadQuantityValue struct {
	basetypes.StringValue
}

func (workloadQuantityValue) Type(context.Context) attr.Type {
	return workloadQuantityType{}
}

func (v workloadQuantityValue) Equal(other attr.Value) bool {
	quantity, ok := other.(workloadQuantityValue)
	return ok && v.StringValue.Equal(quantity.StringValue)
}

func (v workloadQuantityValue) StringSemanticEquals(ctx context.Context, other basetypes.StringValuable) (bool, diag.Diagnostics) {
	otherString, diags := other.ToStringValue(ctx)
	if diags.HasError() || v.IsNull() || v.IsUnknown() || otherString.IsNull() || otherString.IsUnknown() {
		return false, diags
	}
	left, err := resource.ParseQuantity(v.ValueString())
	if err != nil {
		return false, diags
	}
	right, err := resource.ParseQuantity(otherString.ValueString())
	if err != nil {
		return false, diags
	}
	return left.Cmp(right) == 0, diags
}

type workloadQuantityStringPlanModifier struct {
	requiresReplace bool
}

func (workloadQuantityStringPlanModifier) Description(context.Context) string {
	return "Preserves the previous spelling of an equivalent Kubernetes resource quantity."
}

func (m workloadQuantityStringPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m workloadQuantityStringPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	quantity := workloadQuantityValue{StringValue: req.PlanValue}
	equal, diags := quantity.StringSemanticEquals(ctx, req.StateValue)
	resp.Diagnostics.Append(diags...)
	if equal {
		resp.PlanValue = req.StateValue
	}
	resp.RequiresReplace = m.requiresReplace && !resp.PlanValue.Equal(req.StateValue) &&
		!(req.ConfigValue.IsNull() && req.PlanValue.IsUnknown()) &&
		!workloadTemplateEmptyNormalization(ctx, req.Path, req.Config, req.Plan, req.State)
}

type workloadQuantityMapPlanModifier struct {
	requiresReplace bool
}

func (workloadQuantityMapPlanModifier) Description(context.Context) string {
	return "Preserves the previous spelling of equivalent Kubernetes resource quantities."
}

func (m workloadQuantityMapPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m workloadQuantityMapPlanModifier) PlanModifyMap(ctx context.Context, req planmodifier.MapRequest, resp *planmodifier.MapResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	if !req.PlanValue.IsNull() && !req.PlanValue.IsUnknown() && !req.StateValue.IsNull() && !req.StateValue.IsUnknown() {
		elements, prior := req.PlanValue.Elements(), req.StateValue.Elements()
		for key, value := range elements {
			old, ok := prior[key]
			if !ok {
				continue
			}
			quantity, ok := value.(workloadQuantityValue)
			if !ok {
				continue
			}
			previous, ok := old.(basetypes.StringValuable)
			if !ok {
				continue
			}
			equal, diags := quantity.StringSemanticEquals(ctx, previous)
			resp.Diagnostics.Append(diags...)
			if equal {
				elements[key] = old
			}
		}
		normalized, diags := types.MapValue(req.PlanValue.ElementType(ctx), elements)
		resp.Diagnostics.Append(diags...)
		resp.PlanValue = normalized
	}
	resp.RequiresReplace = m.requiresReplace && workloadPlannedChange(ctx, resp.PlanValue, req.StateValue, req.ConfigValue) &&
		!workloadTemplateEmptyNormalization(ctx, req.Path, req.Config, req.Plan, req.State)
}
