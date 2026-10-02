// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

var _ resource.ResourceWithModifyPlan = (*StatefulSetV1)(nil)

func (r *StatefulSetV1) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	candidate, unchanged, err := common.NoOpPlan(req.Config.Raw, req.Plan.Raw, req.State.Raw)
	if err != nil {
		resp.Diagnostics.AddError("Unable to compare StatefulSet plan", err.Error())
		return
	}
	// Resource versions and generations can change on updates. Only retain them
	// once every configured field and every known planned value proves a no-op.
	if unchanged {
		resp.Plan.Raw = candidate
	}
}

// statefulSetVolumeClaimRequiresReplace replaces the StatefulSet where SDKv2
// did (claim templates added, removed or unknown; access modes or limits
// changed) and otherwise plans claim-template edits in place with a warning.
type statefulSetVolumeClaimRequiresReplace struct{}

func (statefulSetVolumeClaimRequiresReplace) Description(context.Context) string {
	return "Adding or removing a volume claim template, or changing its access modes or limits, requires replacement."
}

func (m statefulSetVolumeClaimRequiresReplace) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (statefulSetVolumeClaimRequiresReplace) PlanModifyList(_ context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() || req.PlanValue.Equal(req.StateValue) {
		return
	}
	if req.PlanValue.IsUnknown() {
		resp.RequiresReplace = true
		return
	}
	planned, prior := req.PlanValue.Elements(), req.StateValue.Elements()
	if len(planned) != len(prior) {
		resp.RequiresReplace = true
		return
	}
	inPlace := false
	for i := range planned {
		if planned[i].IsUnknown() ||
			!claimFieldEquivalent(planned[i], prior[i], stringSetsEqual, "spec", "access_modes") ||
			!claimFieldEquivalent(planned[i], prior[i], quantityMapsEqual, "spec", "resources", "limits") {
			resp.RequiresReplace = true
			return
		}
		inPlace = inPlace ||
			!claimFieldEquivalent(planned[i], prior[i], quantityMapsEqual, "spec", "resources", "requests") ||
			!claimFieldEquivalent(planned[i], prior[i], stringMapsEqual, "metadata", "labels") ||
			!claimFieldEquivalent(planned[i], prior[i], stringMapsEqual, "metadata", "annotations")
	}
	if inPlace {
		resp.Diagnostics.AddAttributeWarning(req.Path, "Volume claim template change is not applied",
			"Kubernetes does not allow changing the volume claim templates of an existing StatefulSet, so the new "+
				"requests, labels or annotations are only recorded in Terraform state. The StatefulSet and its "+
				"PersistentVolumeClaims keep their current values, and the next refresh shows the difference again. "+
				"To apply the change, replace the StatefulSet, for example with terraform apply -replace.")
	}
}

// claimFieldEquivalent compares one field of a planned and a prior claim
// template. An unknown planned value is never equivalent.
func claimFieldEquivalent(planned, prior attr.Value, equal func(a, b attr.Value) bool, names ...string) bool {
	a, b := claimTemplateField(planned, names...), claimTemplateField(prior, names...)
	if a != nil && a.IsUnknown() {
		return false
	}
	return equal(a, b)
}

// claimTemplateField walks nested blocks by name, taking the only element of
// each single-element block list. It returns nil for a missing value.
func claimTemplateField(value attr.Value, names ...string) attr.Value {
	for _, name := range names {
		if list, ok := value.(types.List); ok {
			if list.IsUnknown() {
				return list
			}
			elements := list.Elements()
			if len(elements) == 0 {
				return nil
			}
			value = elements[0]
		}
		object, ok := value.(types.Object)
		if !ok || object.IsNull() {
			return nil
		}
		if object.IsUnknown() {
			return object
		}
		value = object.Attributes()[name]
	}
	return value
}

// The comparisons below treat null and empty collections as equal.

func stringSetsEqual(a, b attr.Value) bool {
	x, y := stringElements(a), stringElements(b)
	if len(x) != len(y) {
		return false
	}
	for value := range x {
		if _, ok := y[value]; !ok {
			return false
		}
	}
	return true
}

func stringMapsEqual(a, b attr.Value) bool {
	x, y := stringElements(a), stringElements(b)
	if len(x) != len(y) {
		return false
	}
	for key, value := range x {
		if other, ok := y[key]; !ok || !other.Equal(value) {
			return false
		}
	}
	return true
}

func quantityMapsEqual(a, b attr.Value) bool {
	x, y := stringElements(a), stringElements(b)
	if len(x) != len(y) {
		return false
	}
	for key, value := range x {
		if other, ok := y[key]; !ok || !statefulSetQuantitiesEqual(value, other) {
			return false
		}
	}
	return true
}

// stringElements indexes a string set by value, or a string map by key.
func stringElements(value attr.Value) map[string]types.String {
	out := map[string]types.String{}
	switch v := value.(type) {
	case types.Set:
		for _, element := range v.Elements() {
			if s, ok := element.(types.String); ok {
				out[s.String()] = s
			}
		}
	case types.Map:
		for key, element := range v.Elements() {
			if s, ok := element.(types.String); ok {
				out[key] = s
			}
		}
	}
	return out
}
