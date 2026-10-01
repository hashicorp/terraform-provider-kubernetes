// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package podtemplate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	kquantity "k8s.io/apimachinery/pkg/api/resource"
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

func podQuantityString(fallback string) schema.StringAttribute {
	a := podString(false, false, false, fallback, podStringRule("quantity"))
	a.PlanModifiers = []planmodifier.String{podQuantityStringPlanModifier{}}
	return a
}

func podResourceQuantityMap() schema.MapAttribute {
	return podQuantityMap(true, false)
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
	x, err := kquantity.ParseQuantity(a.ValueString())
	if err != nil {
		return a.Equal(b)
	}
	y, err := kquantity.ParseQuantity(b.ValueString())
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

// Removed list elements do not run their leaf plan modifiers. Carry immutable
// descendants' replacement rules up to the collection's structural boundary.
type podReplacementBoundary interface {
	podRequiresReplacement()
}

type podStringRequiresReplace struct{ planmodifier.String }
type podMapRequiresReplace struct{ planmodifier.Map }
type podSetRequiresReplace struct{ planmodifier.Set }

func (podStringRequiresReplace) podRequiresReplacement()        {}
func (podMapRequiresReplace) podRequiresReplacement()           {}
func (podSetRequiresReplace) podRequiresReplacement()           {}
func (podListStructureRequiresReplace) podRequiresReplacement() {}

func (m podStringRequiresReplace) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !podZeroEquivalent(req.StateValue, req.PlanValue) {
		m.String.PlanModifyString(ctx, req, resp)
	}
}

func (m podMapRequiresReplace) PlanModifyMap(ctx context.Context, req planmodifier.MapRequest, resp *planmodifier.MapResponse) {
	if !podZeroEquivalent(req.StateValue, req.PlanValue) {
		m.Map.PlanModifyMap(ctx, req, resp)
	}
}

func (m podSetRequiresReplace) PlanModifySet(ctx context.Context, req planmodifier.SetRequest, resp *planmodifier.SetResponse) {
	if !podZeroEquivalent(req.StateValue, req.PlanValue) {
		m.Set.PlanModifySet(ctx, req, resp)
	}
}

// SDKv2's absent scalar is zero; zero-to-absent is likewise not an API mutation.
func podZeroEquivalent(before, after attr.Value) bool {
	return !podNonzeroValue(before) && !podNonzeroValue(after)
}

func podModifiersRequireReplacement[T any](modifiers []T) bool {
	for _, modifier := range modifiers {
		if _, ok := any(modifier).(podReplacementBoundary); ok {
			return true
		}
	}
	return false
}

func podObjectRequiresStructuralReplacement(object schema.NestedBlockObject) bool {
	for _, attribute := range object.Attributes {
		var replace bool
		switch a := attribute.(type) {
		case schema.StringAttribute:
			replace = podModifiersRequireReplacement(a.PlanModifiers)
		case schema.BoolAttribute:
			replace = podModifiersRequireReplacement(a.PlanModifiers)
		case schema.Int64Attribute:
			replace = podModifiersRequireReplacement(a.PlanModifiers)
		case schema.ListAttribute:
			replace = podModifiersRequireReplacement(a.PlanModifiers)
		case schema.MapAttribute:
			replace = podModifiersRequireReplacement(a.PlanModifiers)
		case schema.SetAttribute:
			replace = podModifiersRequireReplacement(a.PlanModifiers)
		case schema.ListNestedAttribute:
			replace = podModifiersRequireReplacement(a.PlanModifiers) ||
				podObjectRequiresStructuralReplacement(schema.NestedBlockObject{Attributes: a.NestedObject.Attributes})
		}

		if replace {
			return true
		}
	}
	for _, block := range object.Blocks {
		if b, ok := block.(schema.ListNestedBlock); ok &&
			(podModifiersRequireReplacement(b.PlanModifiers) || podObjectRequiresStructuralReplacement(b.NestedObject)) {
			return true
		}
	}
	return false
}

type podListInheritedRequiresReplace struct {
	object schema.NestedBlockObject
}

func (podListInheritedRequiresReplace) Description(context.Context) string {
	return "adding or removing immutable nested values requires replacement"
}

func (m podListInheritedRequiresReplace) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m podListInheritedRequiresReplace) PlanModifyList(_ context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() || req.PlanValue.IsUnknown() || req.StateValue.IsUnknown() {
		return
	}
	before, after := req.StateValue.Elements(), req.PlanValue.Elements()
	common := min(len(before), len(after))
	for _, removed := range before[common:] {
		if podObjectHasImmutableValue(m.object, removed) {
			resp.RequiresReplace = true
			return
		}
	}
	for _, added := range after[common:] {
		if podObjectHasImmutableValue(m.object, added) {
			resp.RequiresReplace = true
			return
		}
	}
}

func podObjectHasImmutableValue(object schema.NestedBlockObject, value attr.Value) bool {
	if value.IsNull() {
		return false
	}
	if value.IsUnknown() {
		return true
	}
	values := value.(types.Object).Attributes()
	for name, attribute := range object.Attributes {
		if podObjectRequiresStructuralReplacement(schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{name: attribute},
		}) && podNonzeroValue(values[name]) {
			return true
		}
	}
	for name, block := range object.Blocks {
		b, ok := block.(schema.ListNestedBlock)
		if !ok || !podNonzeroValue(values[name]) {
			continue
		}
		if podModifiersRequireReplacement(b.PlanModifiers) || values[name].IsUnknown() {
			return true
		}
		for _, nested := range values[name].(types.List).Elements() {
			if podObjectHasImmutableValue(b.NestedObject, nested) {
				return true
			}
		}
	}
	return false
}

// SDKv2 treats absent scalar values as their zero value during removal.
func podNonzeroValue(value attr.Value) bool {
	if value == nil || value.IsNull() {
		return false
	}
	if value.IsUnknown() {
		return true
	}
	switch v := value.(type) {
	case types.String:
		return v.ValueString() != ""
	case types.Bool:
		return v.ValueBool()
	case types.Int64:
		return v.ValueInt64() != 0
	case types.List:
		return len(v.Elements()) != 0
	case types.Map:
		return len(v.Elements()) != 0
	case types.Set:
		return len(v.Elements()) != 0
	}
	return true
}
