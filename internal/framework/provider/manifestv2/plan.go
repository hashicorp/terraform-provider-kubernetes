// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestv2

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// sameObject reports whether a manifest still describes the object in state.
// The API version may change: an object is served in every version of its
// kind. A manifest without a namespace describes the object in "default" when
// the state shows the kind is namespaced.
func sameObject(prior, declared identity) bool {
	namespace := declared.Namespace
	if namespace == "" {
		namespace = metav1.NamespaceDefault
	}
	return prior.group() == declared.group() && prior.Kind == declared.Kind && prior.Name == declared.Name &&
		(prior.Namespace == "" || prior.Namespace == namespace)
}

// sameArguments reports whether a plan leaves every argument that an apply
// uses as it is.
func sameArguments(plan, state model) bool {
	return plan.Manifest.Equal(state.Manifest) && plan.YAMLBody.Equal(state.YAMLBody) &&
		plan.ComputedFields.Equal(state.ComputedFields) && plan.FieldManager.Equal(state.FieldManager) &&
		plan.Wait.Equal(state.Wait) && plan.Timeouts.Object.Equal(state.Timeouts.Object)
}

// pendingIdentity reports whether a part of the identity in a manifest given
// as an object is not known yet, while the rest of it is.
func pendingIdentity(manifest types.Dynamic) bool {
	if manifest.IsNull() || manifest.IsUnknown() || manifest.IsUnderlyingValueUnknown() {
		return false
	}
	for _, names := range [][]string{{"apiVersion"}, {"kind"}, {"metadata"}, {"metadata", "name"}, {"metadata", "namespace"}} {
		if value := member(manifest, names...); value != nil && value.IsUnknown() {
			return true
		}
	}
	return false
}

// ModifyPlan works from the configuration and state alone. It makes no API
// request, so a plan does not depend on the cluster or on the kind being
// installed.
func (r *ManifestV2) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state *model
	drift := types.ListValueMust(types.StringType, nil)
	if !req.State.Raw.IsNull() {
		state = &model{}
		resp.Diagnostics.Append(req.State.Get(ctx, state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		// An update keeps the object as it was last read. Predicting what
		// the cluster will return is what the plan cannot do, and leaving it
		// unknown would make every reference to it unknown as well: other
		// resources would plan changes, or replacements, for a new label.
		// The next refresh shows what the update changed. After an update
		// whose wait failed, the object is read again: that update applies
		// the manifest again and waits again.
		pending, _ := req.Private.GetKey(ctx, pendingKey)
		switch {
		case pending != nil:
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("object"), types.DynamicUnknown())...)
		case !state.Object.IsNull():
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("object"), state.Object)...)
		}
		// After an apply nothing has drifted. Without one, drift stays as
		// it is, which it also does when lifecycle.ignore_changes keeps
		// yaml_body from restoring the fields it names. A value that is not
		// known yet may turn out to change nothing.
		switch {
		case !req.Config.Raw.IsFullyKnown():
			drift = types.ListUnknown(types.StringType)
		case pending == nil && sameArguments(plan, *state):
			drift = state.Drift
		}
	}
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("drift"), drift)...)
	planIdentity(ctx, plan, state, resp)
}

// planIdentity plans the attributes that name the object, and a replacement
// when the manifest names another object than the one in state.
func planIdentity(ctx context.Context, plan model, state *model, resp *resource.ModifyPlanResponse) {
	if state != nil && pendingIdentity(plan.Manifest) {
		// It may name another object, and replacing it is what Terraform
		// cannot decide once the apply has started.
		resp.RequiresReplace = append(resp.RequiresReplace, path.Root("manifest"))
		return
	}
	_, declared, known, err := plan.declared()
	if err != nil {
		return
	}
	if !known {
		// A manifest that is not known at all, such as yaml_body from a
		// template with a value of another resource, is assumed to name the
		// same object. Only its API version can then still change, so that is
		// all the plan leaves open. If it names another object, that apply
		// fails and the next plan replaces the object.
		if state != nil {
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("api_version"), types.StringUnknown())...)
		}
		return
	}
	if state != nil && !sameObject(state.identity(), declared) {
		// Terraform replaces only for an argument that changed, and either
		// one can carry the new identity.
		resp.RequiresReplace = append(resp.RequiresReplace, path.Root("manifest"), path.Root("yaml_body"))
		return
	}
	// The namespace of a new object is known after apply: only the cluster
	// knows whether the kind is namespaced.
	for attribute, value := range map[string]string{"api_version": declared.APIVersion, "kind": declared.Kind, "name": declared.Name} {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root(attribute), types.StringValue(value))...)
	}
}
