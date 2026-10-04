// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package networkingv1

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	networking "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8types "k8s.io/apimachinery/pkg/types"
)

const ingressClassTimeout = 20 * time.Minute

func (r *IngressClassV1) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan IngressClassV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(validateIngressClassModel(plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, ingressClassTimeout)
	defer cancel()
	client, _, diags := networkingClient(r.SDKv2Meta)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	metadata, diags := common.ExpandMetadata(ctx, plan.Metadata)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := client.NetworkingV1().IngressClasses().Create(ctx, &networking.IngressClass{
		ObjectMeta: metadata, Spec: expandIngressClassSpec(plan.Spec[0]),
	}, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Error creating IngressClass", err.Error())
		return
	}
	resolveIngressClassPlan(&plan, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, ingressClassIdentity(out.Name))...)
	}
}

func (r *IngressClassV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	ctx, cancel := context.WithTimeout(ctx, ingressClassTimeout)
	defer cancel()
	var state IngressClassV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, filters, diags := networkingClient(r.SDKv2Meta)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if state.ID.IsNull() || state.ID.IsUnknown() || state.ID.ValueString() == "" {
		resp.Diagnostics.AddError("Invalid IngressClass state", "The resource id must contain an IngressClass name.")
		return
	}
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, ingressClassIdentity(state.ID.ValueString()))...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	out, err := client.NetworkingV1().IngressClasses().Get(ctx, state.ID.ValueString(), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading IngressClass", err.Error())
		return
	}
	state.Metadata, diags = flattenNetworkingMetadata(ctx, out.ObjectMeta, state.Metadata, filters.GetIgnoreAnnotations(), filters.GetIgnoreLabels())
	resp.Diagnostics.Append(diags...)
	state.ID = types.StringValue(out.Name)
	state.Spec = flattenIngressClassSpec(out.Spec)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, ingressClassIdentity(out.Name))...)
	}
}

func (r *IngressClassV1) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state IngressClassV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(validateIngressClassModel(plan)...)
	resp.Diagnostics.Append(validateIngressClassModel(state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, ingressClassTimeout)
	defer cancel()
	client, _, diags := networkingClient(r.SDKv2Meta)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	live, err := client.NetworkingV1().IngressClasses().Get(ctx, state.ID.ValueString(), metav1.GetOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Error reading IngressClass for update", err.Error())
		return
	}
	ops, diags := networkingMetadataPatch(ctx, live.ObjectMeta, state.Metadata[0].MetadataBase, plan.Metadata[0].MetadataBase)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	priorSpec := expandIngressClassSpec(state.Spec[0])
	plannedSpec := expandIngressClassSpec(plan.Spec[0])
	// Scope is API-owned when omitted. Unknown marking during a metadata-only
	// update must not turn it into a spec edit.
	if plannedSpec.Parameters != nil && priorSpec.Parameters != nil &&
		(plan.Spec[0].Parameters.Scope.IsUnknown() || plan.Spec[0].Parameters.Scope.IsNull()) {
		plannedSpec.Parameters.Scope = priorSpec.Parameters.Scope
	}
	if !reflect.DeepEqual(priorSpec, plannedSpec) {
		ops = append(ops, &kubernetes.ReplaceOperation{Path: "/spec", Value: plannedSpec})
	}
	out := live
	if len(ops) != 0 {
		patch, err := json.Marshal(ops)
		if err != nil {
			resp.Diagnostics.AddError("Error encoding IngressClass patch", err.Error())
			return
		}
		out, err = client.NetworkingV1().IngressClasses().Patch(ctx, state.ID.ValueString(), k8types.JSONPatchType, patch, metav1.PatchOptions{})
		if err != nil {
			resp.Diagnostics.AddError("Error updating IngressClass", err.Error())
			return
		}
	}
	resolveIngressClassPlan(&plan, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, ingressClassIdentity(out.Name))...)
	}
}

func (r *IngressClassV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state IngressClassV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, ingressClassTimeout)
	defer cancel()
	client, _, diags := networkingClient(r.SDKv2Meta)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	name := state.ID.ValueString()
	if name == "" {
		resp.Diagnostics.AddError("Invalid IngressClass state", "The resource id must contain an IngressClass name.")
		return
	}
	err := client.NetworkingV1().IngressClasses().Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error deleting IngressClass", err.Error())
		return
	}
	err = retry.RetryContext(ctx, ingressClassTimeout, func() *retry.RetryError {
		_, err := client.NetworkingV1().IngressClasses().Get(ctx, name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return retry.NonRetryableError(err)
		}
		return retry.RetryableError(fmt.Errorf("IngressClass %q still exists", name))
	})
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for IngressClass deletion", err.Error())
	}
}

func validateIngressClassModel(model IngressClassV1Model) diag.Diagnostics {
	var diags diag.Diagnostics
	if len(model.Metadata) != 1 || len(model.Spec) != 1 {
		diags.AddError("Invalid IngressClass configuration", "Exactly one metadata and spec block are required.")
	}
	return diags
}

func resolveIngressClassPlan(plan *IngressClassV1Model, out *networking.IngressClass) {
	plan.ID = types.StringValue(out.Name)
	plan.Metadata[0].Name = types.StringValue(out.Name)
	plan.Metadata[0].UID = types.StringValue(string(out.UID))
	plan.Metadata[0].Generation = types.Int64Value(out.Generation)
	plan.Metadata[0].ResourceVersion = types.StringValue(out.ResourceVersion)
	if parameters := plan.Spec[0].Parameters; parameters != nil && (parameters.Scope.IsNull() || parameters.Scope.IsUnknown()) {
		parameters.Scope = types.StringValue("")
		if out.Spec.Parameters != nil && out.Spec.Parameters.Scope != nil {
			parameters.Scope = types.StringValue(*out.Spec.Parameters.Scope)
		}
	}
}
