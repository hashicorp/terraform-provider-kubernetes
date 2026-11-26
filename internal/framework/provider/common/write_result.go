// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// SetWriteResult sets state after a Create or Update to the plan, with each
// value the plan leaves unknown taken from the state that set records, which
// is normally flattened from the object Kubernetes returned. Known planned
// values are kept, so fields added by admission or other clients show as drift
// on the next refresh instead of failing the apply. set starts from the
// current state.
func SetWriteResult(ctx context.Context, state *tfsdk.State, plan tfsdk.Plan, set func(*tfsdk.State) diag.Diagnostics) diag.Diagnostics {
	actual := tfsdk.State{Schema: state.Schema, Raw: state.Raw}
	diags := set(&actual)
	w := writeResult{diags: &diags, computed: func(at path.Path) bool {
		attribute, d := state.Schema.AttributeAtPath(ctx, at)
		return !d.HasError() && attribute.IsComputed()
	}}
	state.Raw = w.fill(path.Empty(), plan.Raw, actual.Raw, true)
	return diags
}

type writeResult struct {
	diags    *diag.Diagnostics
	computed func(path.Path) bool
}

func (w writeResult) fill(at path.Path, plan, actual tftypes.Value, found bool) tftypes.Value {
	if plan.IsFullyKnown() {
		return plan
	}
	found = found && actual.Type() != nil && actual.Type().Equal(plan.Type())
	if !plan.IsKnown() {
		if found && actual.IsFullyKnown() {
			return actual
		}
		return w.unresolved(at, plan.Type())
	}
	found = found && actual.IsKnown() && !actual.IsNull()
	switch typ := plan.Type().(type) {
	case tftypes.Object:
		var planned, current map[string]tftypes.Value
		_ = plan.As(&planned)
		if found {
			_ = actual.As(&current)
		}
		out := make(map[string]tftypes.Value, len(planned))
		for name, value := range planned {
			match, ok := current[name]
			out[name] = w.fill(at.AtName(name), value, match, ok)
		}
		return tftypes.NewValue(typ, out)
	case tftypes.Map:
		var planned, current map[string]tftypes.Value
		_ = plan.As(&planned)
		if found {
			_ = actual.As(&current)
		}
		out := make(map[string]tftypes.Value, len(planned))
		for key, value := range planned {
			match, ok := current[key]
			out[key] = w.fill(at.AtMapKey(key), value, match, ok)
		}
		return tftypes.NewValue(typ, out)
	case tftypes.List:
		var planned, current []tftypes.Value
		_ = plan.As(&planned)
		if found {
			_ = actual.As(&current)
		}
		out := make([]tftypes.Value, len(planned))
		for i, j := range listMatches(planned, current) {
			var match tftypes.Value
			if j >= 0 {
				match = current[j]
			}
			out[i] = w.fill(at.AtListIndex(i), planned[i], match, j >= 0)
		}
		return tftypes.NewValue(typ, out)
	case tftypes.Set:
		var planned, current []tftypes.Value
		_ = plan.As(&planned)
		if found {
			_ = actual.As(&current)
		}
		out := make([]tftypes.Value, len(planned))
		claimed := make([]bool, len(current))
		for i, value := range planned {
			out[i] = value
			if value.IsFullyKnown() {
				claim(current, claimed, func(v tftypes.Value) bool { return v.Equal(value) })
			}
		}
		// A set element has no identity besides its value, so an unknown in one is
		// filled only from the single element that matches the rest of it.
		for i, value := range planned {
			if value.IsFullyKnown() {
				continue
			}
			j := -1
			for k, candidate := range current {
				if !claimed[k] && agrees(value, candidate) {
					if j >= 0 {
						j = -1
						break
					}
					j = k
				}
			}
			if j < 0 {
				w.diags.AddAttributeError(at, "Unable to Record Applied Value",
					"The planned value of a set element was unknown and Kubernetes returned no single matching element. "+
						"This is always a problem with the provider. Please report it to the provider developers.")
				return tftypes.NewValue(typ, nil)
			}
			claimed[j] = true
			out[i] = current[j]
		}
		return tftypes.NewValue(typ, out)
	}
	return w.unresolved(at, plan.Type())
}

// unresolved is the value for a planned unknown Kubernetes did not return:
// null where the attribute is computed, and an error anywhere else.
func (w writeResult) unresolved(at path.Path, typ tftypes.Type) tftypes.Value {
	if !w.computed(at) {
		w.diags.AddAttributeError(at, "Unable to Record Applied Value",
			"The planned value was unknown and Kubernetes returned none. "+
				"This is always a problem with the provider. Please report it to the provider developers.")
	}
	return tftypes.NewValue(typ, nil)
}

// listMatches returns, for each planned list element, the index of the actual
// element that holds its values, or -1. An element with a known name matches
// the actual element of that name, so elements added or reordered by
// Kubernetes are not taken for planned ones. Other elements match the rest in
// order, those without a name first.
func listMatches(planned, actual []tftypes.Value) []int {
	plannedNames, actualNames := elementNames(planned), elementNames(actual)
	matches := make([]int, len(planned))
	claimed := make([]bool, len(actual))
	for i, name := range plannedNames {
		matches[i] = -1
		if name == "" {
			continue
		}
		for j, other := range actualNames {
			if !claimed[j] && other == name {
				matches[i], claimed[j] = j, true
				break
			}
		}
	}
	var rest []int
	for _, named := range []bool{false, true} {
		for j, name := range actualNames {
			if !claimed[j] && (name != "") == named {
				rest = append(rest, j)
			}
		}
	}
	for i, name := range plannedNames {
		if name == "" && len(rest) > 0 {
			matches[i], rest = rest[0], rest[1:]
		}
	}
	return matches
}

// elementNames returns the known, non-empty "name" attribute of each object.
func elementNames(values []tftypes.Value) []string {
	names := make([]string, len(values))
	for i, value := range values {
		if _, ok := value.Type().(tftypes.Object); !ok || !value.IsKnown() || value.IsNull() {
			continue
		}
		var attributes map[string]tftypes.Value
		if value.As(&attributes) != nil {
			continue
		}
		name, ok := attributes["name"]
		if ok && name.Type().Is(tftypes.String) && name.IsKnown() && !name.IsNull() {
			_ = name.As(&names[i])
		}
	}
	return names
}

func claim(values []tftypes.Value, claimed []bool, match func(tftypes.Value) bool) {
	for i, value := range values {
		if !claimed[i] && match(value) {
			claimed[i] = true
			return
		}
	}
}

// agrees reports whether actual has every known value of plan.
func agrees(plan, actual tftypes.Value) bool {
	switch {
	case !plan.IsKnown():
		return true
	case plan.IsFullyKnown() || plan.IsNull() || actual.IsNull() || !actual.IsKnown():
		return plan.Equal(actual)
	}
	var plannedMap, actualMap map[string]tftypes.Value
	if plan.As(&plannedMap) == nil && actual.As(&actualMap) == nil {
		if len(plannedMap) != len(actualMap) {
			return false
		}
		for key, value := range plannedMap {
			if other, ok := actualMap[key]; !ok || !agrees(value, other) {
				return false
			}
		}
		return true
	}
	var plannedList, actualList []tftypes.Value
	if plan.As(&plannedList) == nil && actual.As(&actualList) == nil && len(plannedList) == len(actualList) {
		for i, value := range plannedList {
			if !agrees(value, actualList[i]) {
				return false
			}
		}
		return true
	}
	return false
}
