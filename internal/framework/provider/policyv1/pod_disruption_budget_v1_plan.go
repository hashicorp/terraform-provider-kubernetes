// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package policyv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type podDisruptionBudgetSpecRequiresReplace struct{}

func (podDisruptionBudgetSpecRequiresReplace) Description(context.Context) string {
	return "Changes to the spec require replacement, except for legacy null/empty collection normalization."
}

func (m podDisruptionBudgetSpecRequiresReplace) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (podDisruptionBudgetSpecRequiresReplace) PlanModifyList(_ context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	// A deep RequiresReplace on the parent runs before child modifiers. Compare the
	// whole spec here, without rewriting configured null/empty values in the plan.
	resp.RequiresReplace = !pdbSpecValuesEqual(req.StateValue, req.PlanValue)
}

func pdbSpecValuesEqual(a, b attr.Value) bool {
	if a.Equal(b) {
		return true
	}
	if a.IsUnknown() || b.IsUnknown() {
		return false
	}
	switch a := a.(type) {
	case types.List:
		b, ok := b.(types.List)
		if !ok {
			return false
		}
		left, right := a.Elements(), b.Elements()
		if len(left) != len(right) {
			return false
		}
		for i := range left {
			if !pdbSpecValuesEqual(left[i], right[i]) {
				return false
			}
		}
		return true
	case types.Object:
		b, ok := b.(types.Object)
		if !ok || a.IsNull() || b.IsNull() {
			return false
		}
		left, right := a.Attributes(), b.Attributes()
		if len(left) != len(right) {
			return false
		}
		for name, value := range left {
			other, ok := right[name]
			if !ok || !pdbSpecValuesEqual(value, other) {
				return false
			}
		}
		return true
	case types.Map:
		b, ok := b.(types.Map)
		return ok && len(a.Elements()) == 0 && len(b.Elements()) == 0
	case types.Set:
		b, ok := b.(types.Set)
		return ok && len(a.Elements()) == 0 && len(b.Elements()) == 0
	case types.String:
		b, ok := b.(types.String)
		// All spec strings have SDKv2's empty-string default.
		return ok && a.ValueString() == "" && b.ValueString() == ""
	default:
		return false
	}
}
