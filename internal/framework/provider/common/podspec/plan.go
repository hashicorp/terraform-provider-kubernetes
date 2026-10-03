// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package podspec

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	kquantity "k8s.io/apimachinery/pkg/api/resource"
)

// SDKv2's ForceNew on a list of objects governs its structural diff, not all of
// its descendants. Leaf replacement rules are declared on their own attributes.
//
// absentZero marks a list that Kubernetes holds as one element whether or not
// one is sent, which SDKv2 recorded as empty when it held only zero values.
type podListStructureRequiresReplace struct{ absentZero bool }

func (podListStructureRequiresReplace) Description(context.Context) string {
	return "changes to the collection size require replacement"
}
func (v podListStructureRequiresReplace) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (m podListStructureRequiresReplace) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() || req.StateValue.IsUnknown() {
		return
	}
	if m.absentZero && len(req.StateValue.Elements()) == 0 && len(req.PlanValue.Elements()) == 1 && podZeroElement(req.ConfigValue) {
		// The configured zero-valued element is the object Kubernetes already
		// holds; plan its unset maps as the empty maps it reports.
		element := req.PlanValue.Elements()[0].(types.Object)
		attributes := element.Attributes()
		for name, value := range attributes {
			if quantities, ok := value.(types.Map); ok && quantities.IsUnknown() {
				attributes[name] = types.MapValueMust(quantities.ElementType(ctx), map[string]attr.Value{})
			}
		}
		resp.PlanValue = types.ListValueMust(element.Type(ctx), []attr.Value{types.ObjectValueMust(element.AttributeTypes(ctx), attributes)})
		return
	}
	// As in SDKv2, a collection known only after apply may change its size.
	resp.RequiresReplace = req.PlanValue.IsUnknown() || len(req.StateValue.Elements()) != len(req.PlanValue.Elements())
}

func podZeroElement(config types.List) bool {
	return !config.IsNull() && !config.IsUnknown() && len(config.Elements()) == 1 && podZeroValue(config.Elements()[0])
}

func (b builder) quantityString(f forceNew, fallback string) schema.StringAttribute {
	a := b.str(false, false, updatable, fallback, podStringRule("quantity"))
	a.PlanModifiers = []planmodifier.String{podQuantityStringPlanModifier{}}
	if b.replace(f) {
		a.PlanModifiers = append(a.PlanModifiers, podStringRequiresReplace{stringplanmodifier.RequiresReplace()})
	}
	return a
}

func (b builder) quantityMap(computed bool, f forceNew) schema.MapAttribute {
	a := b.mapping(computed, updatable)
	a.PlanModifiers = append(a.PlanModifiers, podQuantityMapPlanModifier{})
	if b.replace(f) {
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

// podSpelling returns how to keep the spelling of a string Kubernetes stores
// as a number, such as run_as_user "01000", or nil for other strings.
func podSpelling(validators []validator.String) func(prior, current types.String) types.String {
	for _, v := range validators {
		switch v {
		case podStringRule("nullable-int"):
			return common.KeepIntSpelling
		case podStringRule("port"):
			return common.KeepIntOrStringSpelling
		case podStringRule("mode"):
			return common.KeepOctalSpelling
		}
	}
	return nil
}

// As with quantities, respelling a number Kubernetes stores, such as "1000" as
// "01000", plans the prior value instead of a change or a replacement.
type podSpellingPlanModifier struct {
	keep func(prior, current types.String) types.String
}

func (podSpellingPlanModifier) Description(context.Context) string {
	return "preserves the spelling of an equal number"
}
func (m podSpellingPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}
func (m podSpellingPlanModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.PlanValue.IsUnknown() {
		resp.PlanValue = m.keep(req.StateValue, req.PlanValue)
	}
}

// The API defaults an empty http_get path to "/", so the two are equivalent:
// an unset path keeps a prior "/" when planning and keeps "" when flattening.
const httpGetDefaultPath = "/"

func (b builder) httpGetPath() schema.StringAttribute {
	a := b.str(false, false, updatable, "")
	a.PlanModifiers = append([]planmodifier.String{podHTTPGetPathPlanModifier{}}, a.PlanModifiers...)
	return a
}

type podHTTPGetPathPlanModifier struct{}

func (podHTTPGetPathPlanModifier) Description(context.Context) string {
	return "treats an empty path as the API default \"/\""
}
func (v podHTTPGetPathPlanModifier) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (podHTTPGetPathPlanModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.PlanValue.Equal(types.StringValue("")) && req.StateValue.Equal(types.StringValue(httpGetDefaultPath)) {
		resp.PlanValue = req.StateValue
	}
}

func podHTTPGetPath(path []string) bool {
	n := len(path)
	return n > 1 && path[n-1] == "path" && path[n-2] == "http_get"
}

func podPreserveHTTPGetPath(prior, current attr.Value) attr.Value {
	if prior.Equal(types.StringValue("")) && current.Equal(types.StringValue(httpGetDefaultPath)) {
		return prior
	}
	return current
}

// As in SDKv2, "" on an API-defaulted string means "unset": it keeps the value
// already in state instead of planning a change to "".
type podEmptyStringKeepsState struct{}

func (podEmptyStringKeepsState) Description(context.Context) string {
	return "an empty string keeps the API-populated value"
}
func (m podEmptyStringKeepsState) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}
func (podEmptyStringKeepsState) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.State.Raw.IsNull() || req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || req.ConfigValue.ValueString() != "" {
		return
	}
	if !req.StateValue.IsNull() && !req.StateValue.IsUnknown() {
		resp.PlanValue = req.StateValue
	}
}

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

// podAttributeRequiresReplacement reports whether an attribute is itself
// ForceNew, regardless of its descendants.
func podAttributeRequiresReplacement(attribute schema.Attribute) bool {
	switch a := attribute.(type) {
	case schema.StringAttribute:
		return podModifiersRequireReplacement(a.PlanModifiers)
	case schema.BoolAttribute:
		return podModifiersRequireReplacement(a.PlanModifiers)
	case schema.Int64Attribute:
		return podModifiersRequireReplacement(a.PlanModifiers)
	case schema.ListAttribute:
		return podModifiersRequireReplacement(a.PlanModifiers)
	case schema.MapAttribute:
		return podModifiersRequireReplacement(a.PlanModifiers)
	case schema.SetAttribute:
		return podModifiersRequireReplacement(a.PlanModifiers)
	case schema.ListNestedAttribute:
		return podModifiersRequireReplacement(a.PlanModifiers)
	}
	return false
}

func podObjectRequiresStructuralReplacement(object schema.NestedBlockObject) bool {
	for _, attribute := range object.Attributes {
		if podAttributeRequiresReplacement(attribute) {
			return true
		}
		if a, ok := attribute.(schema.ListNestedAttribute); ok &&
			podObjectRequiresStructuralReplacement(schema.NestedBlockObject{Attributes: a.NestedObject.Attributes}) {
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

// As in SDKv2, an unknown value replaces only when it is itself ForceNew for
// this owner; a dynamic block over an unknown collection is updated in place
// even when its elements could hold values that force replacement.
func podObjectHasImmutableValue(object schema.NestedBlockObject, value attr.Value) bool {
	if value.IsNull() || value.IsUnknown() {
		return false
	}
	values := value.(types.Object).Attributes()
	for name, attribute := range object.Attributes {
		if !podNonzeroValue(values[name]) {
			continue
		}
		if podAttributeRequiresReplacement(attribute) {
			return true
		}
		nested, ok := attribute.(schema.ListNestedAttribute)
		if !ok || values[name].IsUnknown() {
			continue
		}
		for _, element := range values[name].(types.List).Elements() {
			if podObjectHasImmutableValue(schema.NestedBlockObject{Attributes: nested.NestedObject.Attributes}, element) {
				return true
			}
		}
	}
	for name, block := range object.Blocks {
		b, ok := block.(schema.ListNestedBlock)
		if !ok || !podNonzeroValue(values[name]) {
			continue
		}
		if podModifiersRequireReplacement(b.PlanModifiers) {
			return true
		}
		if values[name].IsUnknown() {
			continue
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
