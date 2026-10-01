// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"k8s.io/apimachinery/pkg/api/resource"
)

// SDKv2's ForceNew on a list of objects governs its structural diff, not all of
// its descendants. Leaf replacement rules are declared on their own attributes.
type podListStructureRequiresReplace struct{}

func (podListStructureRequiresReplace) Description(context.Context) string {
	return "changes to the collection size require replacement"
}
func (v podListStructureRequiresReplace) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (podListStructureRequiresReplace) PlanModifyList(_ context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() || req.PlanValue.IsUnknown() {
		return
	}
	if req.StateValue.IsUnknown() {
		return
	}
	resp.RequiresReplace = len(req.StateValue.Elements()) != len(req.PlanValue.Elements())
}

func podQuantityString(required, replace bool, fallback string) schema.StringAttribute {
	a := podString(required, false, false, fallback, podStringRule("quantity"))
	a.PlanModifiers = []planmodifier.String{podQuantityStringPlanModifier{}}
	if replace {
		a.PlanModifiers = append(a.PlanModifiers, podStringRequiresReplace{stringplanmodifier.RequiresReplace()})
	}
	return a
}

func podResourceQuantityMap() schema.MapAttribute {
	return podQuantityMap(true, true)
}

func podQuantityMap(computed, replace bool) schema.MapAttribute {
	a := podMap(computed, false)
	a.PlanModifiers = append(a.PlanModifiers, podQuantityMapPlanModifier{})
	if replace {
		a.PlanModifiers = append(a.PlanModifiers, podMapRequiresReplace{mapplanmodifier.RequiresReplace()})
	}
	return a
}

func podQuantitiesEqual(a, b types.String) bool {
	if a.IsNull() || a.IsUnknown() || b.IsNull() || b.IsUnknown() {
		return a.Equal(b)
	}
	x, err := resource.ParseQuantity(a.ValueString())
	if err != nil {
		return a.Equal(b)
	}
	y, err := resource.ParseQuantity(b.ValueString())
	return err == nil && x.Cmp(y) == 0
}

type podQuantityStringPlanModifier struct{}

func (podQuantityStringPlanModifier) Description(context.Context) string {
	return "preserves semantically equivalent Kubernetes quantities"
}
func (v podQuantityStringPlanModifier) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (podQuantityStringPlanModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.PlanValue.IsNull() && !req.PlanValue.IsUnknown() && podQuantitiesEqual(req.StateValue, req.PlanValue) {
		resp.PlanValue = req.StateValue
	}
}

type podQuantityMapPlanModifier struct{}

func (podQuantityMapPlanModifier) Description(context.Context) string {
	return "preserves semantically equivalent Kubernetes quantity map entries"
}
func (v podQuantityMapPlanModifier) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (podQuantityMapPlanModifier) PlanModifyMap(_ context.Context, req planmodifier.MapRequest, resp *planmodifier.MapResponse) {
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() || req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}
	entries := req.PlanValue.Elements()
	oldEntries := req.StateValue.Elements()
	if len(entries) != len(oldEntries) {
		return
	}
	for key, value := range entries {
		old, ok := oldEntries[key]
		if !ok || !podQuantitiesEqual(old.(types.String), value.(types.String)) {
			return
		}
	}
	// Core permits the configured map or the prior map, not a mixture of them.
	resp.PlanValue = req.StateValue
}

func podQuantityPath(path []string) bool {
	if len(path) == 0 {
		return false
	}
	switch path[len(path)-1] {
	case "size_limit", "divisor":
		return true
	case "limits", "requests":
		return len(path) > 1 && path[len(path)-2] == "resources"
	default:
		return false
	}
}

func podPreserveQuantity(prior, current attr.Value) attr.Value {
	p, ok := prior.(types.String)
	c, stringValue := current.(types.String)
	if ok && stringValue && podQuantitiesEqual(p, c) {
		return p
	}
	pm, ok := prior.(types.Map)
	cm, mapValue := current.(types.Map)
	if !ok || !mapValue || pm.IsNull() || pm.IsUnknown() || cm.IsNull() || cm.IsUnknown() {
		return current
	}
	entries := cm.Elements()
	for key, entry := range entries {
		old, exists := pm.Elements()[key]
		if exists && podQuantitiesEqual(old.(types.String), entry.(types.String)) {
			entries[key] = old
		}
	}
	value, _ := types.MapValue(types.StringType, entries)
	return value
}
