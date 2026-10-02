// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (p *PodV1) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}
	p.planSpecReplacement(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	var planned, prior types.List
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata"), &planned)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata"), &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if planned.IsUnknown() || prior.IsUnknown() || len(planned.Elements()) != 1 || len(prior.Elements()) != 1 {
		return
	}

	// Quantity modifiers can restore semantic equality after Framework has
	// marked metadata unknown. Restore API revisions only for an otherwise
	// identical plan; genuine updates must still receive fresh API values.
	candidate := req.Plan
	plannedMetadata := planned.Elements()[0].(types.Object).Attributes()
	priorMetadata := prior.Elements()[0].(types.Object).Attributes()
	for _, name := range []string{"generation", "resource_version"} {
		if plannedMetadata[name].IsUnknown() && !priorMetadata[name].IsUnknown() {
			resp.Diagnostics.Append(candidate.SetAttribute(ctx,
				path.Root("metadata").AtListIndex(0).AtName(name), priorMetadata[name])...)
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}
	if candidate.Raw.Equal(req.State.Raw) {
		resp.Plan = candidate
	}
}

// Kubernetes rejects most Pod spec updates, including fields SDKv2 left
// updatable. Replace the Pod when the planned spec differs in anything
// podV1UpdatePatch cannot send. A difference from the prior state alone is
// checked against the live Pod, since state from -refresh=false may be stale.
func (p *PodV1) planSpecReplacement(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	var planned, prior types.List
	var id types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec"), &planned)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("spec"), &prior)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() || planned.IsUnknown() || planned.Equal(prior) {
		return
	}
	// An unknown value cannot be patched in, so it may change an immutable field.
	plannedSpec, diags := podV1Spec().ExpandSpec(ctx, planned, path.Root("spec"))
	if diags.HasError() {
		resp.RequiresReplace = append(resp.RequiresReplace, path.Root("spec"))
		return
	}
	priorSpec, diags := podV1Spec().ExpandSpec(ctx, prior, path.Root("spec"))
	if !diags.HasError() && !podV1SpecRequiresReplacement(priorSpec, plannedSpec) {
		return
	}
	if live, ok := p.livePod(ctx, id.ValueString()); ok {
		patched := live.Spec.DeepCopy()
		podV1ApplySpecPatch(patched, &plannedSpec)
		// Update stores this flattening; matching the plan means no replacement.
		flattened, diags := podV1Spec().FlattenSpec(ctx, *patched, planned, path.Root("spec"))
		if !diags.HasError() && podV1SameBlocks(flattened, planned) {
			return
		}
	}
	resp.RequiresReplace = append(resp.RequiresReplace, path.Root("spec"))
}

// podV1SameBlocks is Equal with an empty block list matching a null one, as
// Terraform compares nested blocks.
func podV1SameBlocks(a, b attr.Value) bool {
	switch x := a.(type) {
	case types.List:
		y, ok := b.(types.List)
		if !ok || x.IsUnknown() || y.IsUnknown() {
			return a.Equal(b)
		}
		if len(x.Elements()) != len(y.Elements()) {
			return false
		}
		for i, element := range x.Elements() {
			if !podV1SameBlocks(element, y.Elements()[i]) {
				return false
			}
		}
		return true
	case types.Object:
		y, ok := b.(types.Object)
		if !ok || x.IsNull() || x.IsUnknown() || y.IsNull() || y.IsUnknown() {
			return a.Equal(b)
		}
		for name, value := range x.Attributes() {
			if !podV1SameBlocks(value, y.Attributes()[name]) {
				return false
			}
		}
		return len(x.Attributes()) == len(y.Attributes())
	}
	return a.Equal(b)
}

func (p *PodV1) livePod(ctx context.Context, id string) (*corev1.Pod, bool) {
	clients, _, diags := p.sdkv2Meta()
	if diags.HasError() {
		return nil, false
	}
	namespace, name, err := kubernetes.IdParts(id)
	if err != nil {
		return nil, false
	}
	conn, err := clients.MainClientset()
	if err != nil {
		return nil, false
	}
	pod, err := conn.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, false
	}
	return pod, true
}

// podV1ApplySpecPatch sets the spec fields podV1UpdatePatch can change.
func podV1ApplySpecPatch(spec, planned *corev1.PodSpec) {
	spec.ActiveDeadlineSeconds = planned.ActiveDeadlineSeconds
}

// podV1SpecRequiresReplacement compares two specs as the API sees them.
func podV1SpecRequiresReplacement(prior, planned corev1.PodSpec) bool {
	patched := prior.DeepCopy()
	podV1ApplySpecPatch(patched, &planned)
	return !reflect.DeepEqual(podV1SpecValue(*patched), podV1SpecValue(planned))
}

// podV1SpecValue is the spec as API JSON without null, zero or empty values,
// which the provider treats as unset.
func podV1SpecValue(spec corev1.PodSpec) interface{} {
	data, err := json.Marshal(spec)
	if err != nil {
		return nil
	}
	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		return nil
	}
	return podV1PruneZero(value)
}

func podV1PruneZero(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		for key, child := range v {
			if child = podV1PruneZero(child); child == nil {
				delete(v, key)
			} else {
				v[key] = child
			}
		}
		if len(v) == 0 {
			return nil
		}
	case []interface{}:
		if len(v) == 0 {
			return nil
		}
		for i, child := range v {
			v[i] = podV1PruneZero(child)
		}
	case string:
		if v == "" {
			return nil
		}
	case bool:
		if !v {
			return nil
		}
	case float64:
		if v == 0 {
			return nil
		}
	}
	return value
}
