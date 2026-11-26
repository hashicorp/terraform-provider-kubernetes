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
		// The next refresh shows what the update changed.
		if !state.Object.IsNull() {
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("object"), state.Object)...)
		}
	}
	// After an apply nothing has drifted. A list that Read filled therefore
	// shows as a change, which is what makes Terraform apply again.
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("drift"), types.ListValueMust(types.StringType, nil))...)
	_, declared, known, err := plan.declared()
	if err != nil {
		return
	}
	if !known {
		// The object is assumed to stay the same one. Only its API version
		// can then still change, so that is all the plan leaves open. A
		// different object would need a replacement, which Terraform cannot
		// plan once the apply has started: that apply fails and the next plan
		// replaces the object.
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
