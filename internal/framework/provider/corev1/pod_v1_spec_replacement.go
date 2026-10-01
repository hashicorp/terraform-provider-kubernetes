// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Removed list elements do not run their leaf plan modifiers. Carry immutable
// descendants' replacement rules up to the collection's structural boundary.
type podReplacementBoundary interface {
	podRequiresReplacement()
}

type podStringRequiresReplace struct{ planmodifier.String }
type podBoolRequiresReplace struct{ planmodifier.Bool }
type podInt64RequiresReplace struct{ planmodifier.Int64 }
type podListRequiresReplace struct{ planmodifier.List }
type podMapRequiresReplace struct{ planmodifier.Map }
type podSetRequiresReplace struct{ planmodifier.Set }

func (podStringRequiresReplace) podRequiresReplacement()        {}
func (podBoolRequiresReplace) podRequiresReplacement()          {}
func (podInt64RequiresReplace) podRequiresReplacement()         {}
func (podListRequiresReplace) podRequiresReplacement()          {}
func (podMapRequiresReplace) podRequiresReplacement()           {}
func (podSetRequiresReplace) podRequiresReplacement()           {}
func (podListStructureRequiresReplace) podRequiresReplacement() {}

func (m podStringRequiresReplace) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !podZeroEquivalent(req.StateValue, req.PlanValue) {
		m.String.PlanModifyString(ctx, req, resp)
	}
}

func (m podBoolRequiresReplace) PlanModifyBool(ctx context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if !podZeroEquivalent(req.StateValue, req.PlanValue) {
		m.Bool.PlanModifyBool(ctx, req, resp)
	}
}

func (m podInt64RequiresReplace) PlanModifyInt64(ctx context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	if !podZeroEquivalent(req.StateValue, req.PlanValue) {
		m.Int64.PlanModifyInt64(ctx, req, resp)
	}
}

func (m podListRequiresReplace) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if !podZeroEquivalent(req.StateValue, req.PlanValue) {
		m.List.PlanModifyList(ctx, req, resp)
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
