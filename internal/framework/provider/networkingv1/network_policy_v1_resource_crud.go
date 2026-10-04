// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package networkingv1

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	networking "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

const networkPolicyTimeout = 20 * time.Minute

func (r *NetworkPolicyV1) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	ctx, cancel := context.WithTimeout(ctx, networkPolicyTimeout)
	defer cancel()
	var plan NetworkPolicyV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(plan.Metadata) != 1 {
		resp.Diagnostics.AddError("Invalid network policy metadata", "Exactly one metadata block is required.")
		return
	}
	client, _, d := networkingClient(r.SDKv2Meta)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	metadata, d := common.ExpandNamespacedMetadata(ctx, plan.Metadata)
	resp.Diagnostics.Append(d...)
	spec, d := networkPolicyExpandSpec(ctx, plan.Spec)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := client.NetworkingV1().NetworkPolicies(metadata.Namespace).Create(ctx,
		&networking.NetworkPolicy{ObjectMeta: metadata, Spec: spec}, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Error creating network policy", err.Error())
		return
	}
	// Retain the created object even if its follow-up read or flattening fails.
	resp.Diagnostics.Append(networkPolicyWriteState(ctx, &resp.State, resp.Identity, &plan, out.ObjectMeta)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err = client.NetworkingV1().NetworkPolicies(out.Namespace).Get(ctx, out.Name, metav1.GetOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Error reading created network policy", err.Error())
		return
	}
	// Preserve the planned spec; Read exposes admission changes as drift.
	resp.Diagnostics.Append(networkPolicyWriteState(ctx, &resp.State, resp.Identity, &plan, out.ObjectMeta)...)
}

func (r *NetworkPolicyV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	ctx, cancel := context.WithTimeout(ctx, networkPolicyTimeout)
	defer cancel()
	var state NetworkPolicyV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid network policy ID", err.Error())
		return
	}
	client, filters, d := networkingClient(r.SDKv2Meta)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, networkPolicyIdentity(namespace, name))...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	out, err := client.NetworkingV1().NetworkPolicies(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading network policy", err.Error())
		return
	}
	state.Metadata, d = flattenNetworkingNamespacedMetadata(ctx, out.ObjectMeta, state.Metadata,
		filters.GetIgnoreAnnotations(), filters.GetIgnoreLabels())
	resp.Diagnostics.Append(d...)
	state.Spec, d = networkPolicyFlattenSpec(ctx, out.Spec, state.Spec)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(networkPolicyWriteState(ctx, &resp.State, resp.Identity, &state, out.ObjectMeta)...)
}

func (r *NetworkPolicyV1) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	ctx, cancel := context.WithTimeout(ctx, networkPolicyTimeout)
	defer cancel()
	var plan, prior NetworkPolicyV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(plan.Metadata) != 1 || len(prior.Metadata) != 1 {
		resp.Diagnostics.AddError("Invalid network policy metadata", "Exactly one metadata block is required.")
		return
	}
	namespace, name, err := kubernetes.IdParts(prior.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid network policy ID", err.Error())
		return
	}
	client, _, d := networkingClient(r.SDKv2Meta)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := client.NetworkingV1().NetworkPolicies(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Error reading network policy for update", err.Error())
		return
	}
	ops, d := networkingMetadataPatch(ctx, out.ObjectMeta, prior.Metadata[0].MetadataBase, plan.Metadata[0].MetadataBase)
	resp.Diagnostics.Append(d...)
	spec, d := networkPolicyExpandSpec(ctx, plan.Spec)
	resp.Diagnostics.Append(d...)
	priorSpec, d := networkPolicyExpandSpec(ctx, prior.Spec)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	// An empty list clears rules. JSON patch "add" also replaces existing members,
	// so optional spec fields need not already exist on the live object.
	if spec.Ingress == nil {
		spec.Ingress = []networking.NetworkPolicyIngressRule{}
	}
	if spec.Egress == nil {
		spec.Egress = []networking.NetworkPolicyEgressRule{}
	}
	if priorSpec.Ingress == nil {
		priorSpec.Ingress = []networking.NetworkPolicyIngressRule{}
	}
	if priorSpec.Egress == nil {
		priorSpec.Egress = []networking.NetworkPolicyEgressRule{}
	}
	for _, field := range []struct {
		apiName string
		prior   any
		value   any
	}{
		{"ingress", priorSpec.Ingress, spec.Ingress},
		{"egress", priorSpec.Egress, spec.Egress},
		{"podSelector", priorSpec.PodSelector, spec.PodSelector},
		{"policyTypes", priorSpec.PolicyTypes, spec.PolicyTypes},
	} {
		before, err := json.Marshal(field.prior)
		if err != nil {
			resp.Diagnostics.AddError("Error encoding prior network policy", err.Error())
			return
		}
		after, err := json.Marshal(field.value)
		if err != nil {
			resp.Diagnostics.AddError("Error encoding planned network policy", err.Error())
			return
		}
		// Empty-map/set state normalization must not write an equivalent API spec.
		if !bytes.Equal(before, after) {
			ops = append(ops, &kubernetes.AddOperation{Path: "/spec/" + field.apiName, Value: field.value})
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}
	if len(ops) > 0 {
		data, err := ops.MarshalJSON()
		if err != nil {
			resp.Diagnostics.AddError("Error encoding network policy update", err.Error())
			return
		}
		out, err = client.NetworkingV1().NetworkPolicies(namespace).Patch(ctx, name, k8stypes.JSONPatchType, data, metav1.PatchOptions{})
		if err != nil {
			resp.Diagnostics.AddError("Error updating network policy", err.Error())
			return
		}
	}
	// Preserve the planned spec; Read exposes admission changes as drift.
	resp.Diagnostics.Append(networkPolicyWriteState(ctx, &resp.State, resp.Identity, &plan, out.ObjectMeta)...)
}

func (r *NetworkPolicyV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	ctx, cancel := context.WithTimeout(ctx, networkPolicyTimeout)
	defer cancel()
	var state NetworkPolicyV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid network policy ID", err.Error())
		return
	}
	client, _, d := networkingClient(r.SDKv2Meta)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	err = client.NetworkingV1().NetworkPolicies(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting network policy", err.Error())
	}
}

func networkPolicyWriteState(ctx context.Context, state *tfsdk.State, identity *tfsdk.ResourceIdentity, model *NetworkPolicyV1Model, metadata metav1.ObjectMeta) diag.Diagnostics {
	var diags diag.Diagnostics
	model.ID = types.StringValue(kubernetes.BuildId(metadata))
	model.Metadata[0].Name = types.StringValue(metadata.Name)
	model.Metadata[0].Namespace = types.StringValue(metadata.Namespace)
	model.Metadata[0].UID = types.StringValue(string(metadata.UID))
	model.Metadata[0].ResourceVersion = types.StringValue(metadata.ResourceVersion)
	model.Metadata[0].Generation = types.Int64Value(metadata.Generation)
	// Configured metadata maps remain planned values during writes; Read handles
	// API-added keys using the provider's ownership filters.
	diags.Append(state.Set(ctx, model)...)
	if identity != nil {
		diags.Append(identity.Set(ctx, networkPolicyIdentity(metadata.Namespace, metadata.Name))...)
	}
	return diags
}
