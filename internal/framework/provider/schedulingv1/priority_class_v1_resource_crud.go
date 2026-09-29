// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package schedulingv1

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"

	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

func (r *PriorityClassV1) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PriorityClassModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	meta := r.SDKv2Meta().(kubernetes.KubeClientsets)
	conn, err := meta.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	k8sMeta, diags := common.ExpandMetadata(ctx, plan.Metadata)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	preemptionPolicy := corev1.PreemptionPolicy(plan.PreemptionPolicy.ValueString())
	obj := &schedulingv1.PriorityClass{
		ObjectMeta:       k8sMeta,
		Value:            int32(plan.Value.ValueInt64()),
		Description:      plan.Description.ValueString(),
		GlobalDefault:    plan.GlobalDefault.ValueBool(),
		PreemptionPolicy: &preemptionPolicy,
	}

	out, err := conn.SchedulingV1().PriorityClasses().Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"error creating PriorityClass",
			fmt.Sprintf("Failed to create PriorityClass %q: %s", k8sMeta.Name, err.Error()),
		)
		return
	}

	// Build state from plan, overwriting only server-assigned fields.
	// Create must echo the plan — not filter the API response — to satisfy
	// the apply-consistency contract (§2.3 of the migration guide).
	plan.ID = types.StringValue(out.Name)
	populateMetadataFromResponse(&plan, out)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	resp.Diagnostics.Append(resp.Identity.Set(ctx, common.ResourceIdentity{
		APIVersion: types.StringValue("scheduling.k8s.io/v1"),
		Kind:       types.StringValue("PriorityClass"),
		Name:       types.StringValue(out.Name),
	})...)
}

func (r *PriorityClassV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PriorityClassModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	meta := r.SDKv2Meta().(kubernetes.KubeClientsets)
	conn, err := meta.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	name := state.ID.ValueString()
	out, err := conn.SchedulingV1().PriorityClasses().Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"error reading PriorityClass",
			fmt.Sprintf("Failed to read PriorityClass %q: %s", name, err.Error()),
		)
		return
	}

	// Read uses the API response as the source of truth and filters internal/ignored keys.
	metaFilters := meta.(kubernetes.MetadataFilters)
	flatMeta, diags := common.FlattenMetadata(ctx, out.ObjectMeta, state.Metadata, metaFilters.GetIgnoreAnnotations(), metaFilters.GetIgnoreLabels())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Metadata = flatMeta

	state.Value = types.Int64Value(int64(out.Value))
	state.Description = types.StringValue(out.Description)
	state.GlobalDefault = types.BoolValue(out.GlobalDefault)
	if out.PreemptionPolicy != nil {
		state.PreemptionPolicy = types.StringValue(string(*out.PreemptionPolicy))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	// Framework v1.16.1+: Read must always write identity unconditionally.
	// The framework handles the "unexpected identity change" guard internally.
	resp.Diagnostics.Append(resp.Identity.Set(ctx, common.ResourceIdentity{
		APIVersion: types.StringValue("scheduling.k8s.io/v1"),
		Kind:       types.StringValue("PriorityClass"),
		Name:       types.StringValue(out.Name),
	})...)
}

func (r *PriorityClassV1) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PriorityClassModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// id is Computed-only — it lives in state, not in the plan. Read it from state.
	var state PriorityClassModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	meta := r.SDKv2Meta().(kubernetes.KubeClientsets)
	conn, err := meta.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	name := state.ID.ValueString()

	// Build JSON Patch: metadata maps + mutable scalar fields.
	ops := common.MetadataPatchOps("/metadata/", state.Metadata[0], plan.Metadata[0])
	ops = append(ops, &kubernetes.AddOperation{
		Path:  "/description",
		Value: plan.Description.ValueString(),
	})
	ops = append(ops, &kubernetes.AddOperation{
		Path:  "/globalDefault",
		Value: plan.GlobalDefault.ValueBool(),
	})

	patchBytes, err := json.Marshal(ops)
	if err != nil {
		resp.Diagnostics.AddError("patch serialization error", err.Error())
		return
	}

	out, err := conn.SchedulingV1().PriorityClasses().Patch(
		ctx,
		name,
		k8stypes.JSONPatchType,
		patchBytes,
		metav1.PatchOptions{},
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"error updating PriorityClass",
			fmt.Sprintf("Failed to patch PriorityClass %q: %s", name, err.Error()),
		)
		return
	}

	// Build state from plan, overwriting only server-assigned fields.
	// Update must echo the plan — not filter the API response — to satisfy
	// the apply-consistency contract (§2.3 of the migration guide).
	plan.ID = types.StringValue(out.Name)
	populateMetadataFromResponse(&plan, out)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	resp.Diagnostics.Append(resp.Identity.Set(ctx, common.ResourceIdentity{
		APIVersion: types.StringValue("scheduling.k8s.io/v1"),
		Kind:       types.StringValue("PriorityClass"),
		Name:       types.StringValue(out.Name),
	})...)
}

func (r *PriorityClassV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PriorityClassModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := r.SDKv2Meta().(kubernetes.KubeClientsets).MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	name := state.ID.ValueString()
	err = conn.SchedulingV1().PriorityClasses().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		resp.Diagnostics.AddError(
			"error deleting PriorityClass",
			fmt.Sprintf("Failed to delete PriorityClass %q: %s", name, err.Error()),
		)
	}
}

func (r *PriorityClassV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var name string

	if req.ID != "" {
		name = req.ID
	} else {
		var identityData common.ResourceIdentity
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identityData)...)
		if resp.Diagnostics.HasError() {
			return
		}
		name = identityData.Name.ValueString()
	}

	meta := r.SDKv2Meta().(kubernetes.KubeClientsets)
	conn, err := meta.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	out, err := conn.SchedulingV1().PriorityClasses().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"error importing PriorityClass",
			fmt.Sprintf("Failed to import PriorityClass %q: %s", name, err.Error()),
		)
		return
	}

	// Import reads from the API and filters, same as Read.
	metaFilters := meta.(kubernetes.MetadataFilters)
	flatMeta, diags := common.FlattenMetadata(ctx, out.ObjectMeta, nil, metaFilters.GetIgnoreAnnotations(), metaFilters.GetIgnoreLabels())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state PriorityClassModel
	state.ID = types.StringValue(out.Name)
	state.Metadata = flatMeta
	state.Value = types.Int64Value(int64(out.Value))
	state.Description = types.StringValue(out.Description)
	state.GlobalDefault = types.BoolValue(out.GlobalDefault)
	if out.PreemptionPolicy != nil {
		state.PreemptionPolicy = types.StringValue(string(*out.PreemptionPolicy))
	} else {
		state.PreemptionPolicy = types.StringValue("PreemptLowerPriority")
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	resp.Diagnostics.Append(resp.Identity.Set(ctx, common.ResourceIdentity{
		APIVersion: types.StringValue("scheduling.k8s.io/v1"),
		Kind:       types.StringValue("PriorityClass"),
		Name:       types.StringValue(out.Name),
	})...)
}

// populateMetadataFromResponse overwrites the server-assigned metadata fields
// (name, uid, resource_version, generation) in the plan model with values from
// the API response. All other metadata fields are left as planned.
//
// The plan pointer is required: assignments to a value receiver's fields are
// silently discarded for non-slice fields like ID (§4 of MIGRATION_CONVERGENCE_GUIDE.md).
func populateMetadataFromResponse(plan *PriorityClassModel, out *schedulingv1.PriorityClass) {
	if len(plan.Metadata) == 0 {
		return
	}
	m := plan.Metadata[0]
	m.Name = types.StringValue(out.Name)
	m.UID = types.StringValue(string(out.UID))
	m.ResourceVersion = types.StringValue(out.ResourceVersion)
	m.Generation = types.Int64Value(out.Generation)
	plan.Metadata[0] = m
}
