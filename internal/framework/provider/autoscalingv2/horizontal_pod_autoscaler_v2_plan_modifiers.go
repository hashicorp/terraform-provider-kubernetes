// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package autoscalingv2

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	kquantity "k8s.io/apimachinery/pkg/api/resource"
)

// quantitiesEqual reports whether two quantity strings have the same value, so "1000m"
// and "1" compare equal. Unparsable strings compare as text.
func quantitiesEqual(a, b types.String) bool {
	if a.IsNull() || a.IsUnknown() || b.IsNull() || b.IsUnknown() {
		return a.Equal(b)
	}
	x, err := kquantity.ParseQuantity(a.ValueString())
	if err != nil {
		return a.Equal(b)
	}
	y, err := kquantity.ParseQuantity(b.ValueString())
	return err == nil && x.Cmp(y) == 0
}

// quantityPlanModifier keeps the prior spelling of a quantity whose configured value is
// equivalent, so Kubernetes canonicalising "1000m" to "1" does not plan an update.
// Terraform accepts the prior value in place of an equivalent configured value.
type quantityPlanModifier struct{}

func (quantityPlanModifier) Description(context.Context) string {
	return "Keeps the prior value when the configured quantity is equivalent."
}

func (m quantityPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (quantityPlanModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.PlanValue.IsNull() && !req.PlanValue.IsUnknown() && quantitiesEqual(req.StateValue, req.PlanValue) {
		resp.PlanValue = req.StateValue
	}
}

// useStateForUnknownInExistingParent is UseStateForUnknown for an attribute nested in an
// Optional+Computed object. The framework modifier copies the prior value even when the
// parent object was null in state, which would plan null for a value Kubernetes defaults
// when the parent is first configured. This one leaves the value unknown in that case.
type useStateForUnknownInExistingParent struct{}

func (useStateForUnknownInExistingParent) Description(context.Context) string {
	return "Keeps the prior value when the enclosing object already exists and the value is not configured."
}

func (m useStateForUnknownInExistingParent) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (useStateForUnknownInExistingParent) use(ctx context.Context, state tfsdk.State, at path.Path, planUnknown, configUnknown bool) bool {
	if state.Raw.IsNull() || !planUnknown || configUnknown {
		return false
	}
	var parent types.Object
	if diags := state.GetAttribute(ctx, at.ParentPath(), &parent); diags.HasError() {
		return false
	}
	return !parent.IsNull() && !parent.IsUnknown()
}

func (m useStateForUnknownInExistingParent) PlanModifyObject(ctx context.Context, req planmodifier.ObjectRequest, resp *planmodifier.ObjectResponse) {
	if m.use(ctx, req.State, req.Path, req.PlanValue.IsUnknown(), req.ConfigValue.IsUnknown()) {
		resp.PlanValue = req.StateValue
	}
}

func (m useStateForUnknownInExistingParent) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if m.use(ctx, req.State, req.Path, req.PlanValue.IsUnknown(), req.ConfigValue.IsUnknown()) {
		resp.PlanValue = req.StateValue
	}
}

func (m useStateForUnknownInExistingParent) PlanModifyInt64(ctx context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	if m.use(ctx, req.State, req.Path, req.PlanValue.IsUnknown(), req.ConfigValue.IsUnknown()) {
		resp.PlanValue = req.StateValue
	}
}
