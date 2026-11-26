// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
)

func statefulSetQuantitiesEqual(a, b types.String) bool {
	if a.IsNull() || a.IsUnknown() || b.IsNull() || b.IsUnknown() {
		return a.Equal(b)
	}
	before, err := k8sresource.ParseQuantity(a.ValueString())
	if err != nil {
		return a.Equal(b)
	}
	after, err := k8sresource.ParseQuantity(b.ValueString())
	return err == nil && before.Cmp(after) == 0
}

type statefulSetQuantityMapModifier struct{}

func (statefulSetQuantityMapModifier) Description(context.Context) string {
	return "Preserves equivalent Kubernetes resource quantities."
}

func (m statefulSetQuantityMapModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (statefulSetQuantityMapModifier) PlanModifyMap(_ context.Context, req planmodifier.MapRequest, resp *planmodifier.MapResponse) {
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() || req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}
	planned, previous := req.PlanValue.Elements(), req.StateValue.Elements()
	if len(planned) != len(previous) {
		return
	}
	for key, value := range planned {
		prior, ok := previous[key]
		if !ok || !statefulSetQuantitiesEqual(prior.(types.String), value.(types.String)) {
			return
		}
	}
	// Core permits the entire prior map, not a mixture of prior and configured entries.
	resp.PlanValue = req.StateValue
}

func statefulSetPreserveQuantityMap(current, prior types.Map) (types.Map, diag.Diagnostics) {
	if current.IsNull() || current.IsUnknown() || prior.IsNull() || prior.IsUnknown() {
		return current, nil
	}
	entries := current.Elements()
	previous := prior.Elements()
	for key, value := range entries {
		if old, ok := previous[key]; ok && statefulSetQuantitiesEqual(old.(types.String), value.(types.String)) {
			entries[key] = old
		}
	}
	return types.MapValue(types.StringType, entries)
}
