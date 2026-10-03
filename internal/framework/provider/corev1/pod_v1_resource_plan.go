// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common/podspec"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// podPlanReadTimeout bounds the read of the live Pod while planning.
const podPlanReadTimeout = 30 * time.Second

func (p *PodV1) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}
	if !rawAttributeUnchanged(req.Plan.Raw, req.State.Raw, "spec") {
		p.planSpecReplacement(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	if rawAttributeUnchanged(req.Plan.Raw, req.State.Raw, "metadata") {
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

	// The quantity modifiers can make the plan equal to state after Framework has
	// marked computed metadata unknown; keep the API revisions only then, so
	// genuine updates still get fresh values.
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
// updatable. Replace the Pod when Update, which patches only the fields that
// podV1ApplySpecPatch sets, cannot make the live Pod match the planned spec.
// The prior state stands in for a Pod that no longer exists; Update then
// reports it as missing.
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
	live, diags := p.livePod(ctx, id.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var replace bool
	if live != nil {
		patched := live.Spec.DeepCopy()
		podV1ApplySpecPatch(patched, &plannedSpec)
		// The patched Pod, flattened as after a write, must satisfy the plan.
		flattened, diags := podV1Spec().FlattenSpec(ctx, *patched, planned, path.Root("spec"))
		replace = diags.HasError() || !podspec.Satisfies(flattened, planned)
	} else {
		priorSpec, diags := podV1Spec().ExpandSpec(ctx, prior, path.Root("spec"))
		replace = diags.HasError() || podV1SpecRequiresReplacement(priorSpec, plannedSpec)
	}
	if replace {
		resp.RequiresReplace = append(resp.RequiresReplace, path.Root("spec"))
	}
}

// rawAttributeUnchanged reports whether plan and state hold the same value for
// a top-level attribute, without decoding either into Framework values.
func rawAttributeUnchanged(plan, state tftypes.Value, name string) bool {
	at := tftypes.NewAttributePath().WithAttributeName(name)
	planned, _, err := tftypes.WalkAttributePath(plan, at)
	if err != nil {
		return false
	}
	prior, _, err := tftypes.WalkAttributePath(state, at)
	if err != nil {
		return false
	}
	plannedValue, ok := planned.(tftypes.Value)
	if !ok {
		return false
	}
	priorValue, ok := prior.(tftypes.Value)
	return ok && plannedValue.Equal(priorValue)
}

// livePod reads the Pod for planning. It returns nil when the Pod no longer exists.
func (p *PodV1) livePod(ctx context.Context, id string) (*corev1.Pod, diag.Diagnostics) {
	clients, _, diags := p.sdkv2Meta()
	if diags.HasError() {
		return nil, diags
	}
	namespace, name, err := kubernetes.IdParts(id)
	if err != nil {
		diags.AddError("Invalid Pod ID", err.Error())
		return nil, diags
	}
	conn, err := clients.MainClientset()
	if err != nil {
		diags.AddError("Kubernetes client error", err.Error())
		return nil, diags
	}
	ctx, cancel := context.WithTimeout(ctx, podPlanReadTimeout)
	defer cancel()
	pod, err := conn.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, diags
	}
	if err != nil {
		diags.AddError("Error reading Pod",
			fmt.Sprintf("Planning a change to the spec of Pod %q requires reading it from Kubernetes: %s", id, err))
		return nil, diags
	}
	return pod, diags
}

// podV1ApplySpecPatch sets the spec fields podV1UpdatePatch can change.
// Kubernetes lets active_deadline_seconds be set or lowered, never raised or removed.
func podV1ApplySpecPatch(spec, planned *corev1.PodSpec) {
	current, next := spec.ActiveDeadlineSeconds, planned.ActiveDeadlineSeconds
	if current == nil || (next != nil && *next <= *current) {
		spec.ActiveDeadlineSeconds = next
	}
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
