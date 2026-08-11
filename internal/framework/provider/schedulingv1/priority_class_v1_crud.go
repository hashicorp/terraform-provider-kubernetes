// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package schedulingv1

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

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

	preemptionPolicy := corev1.PreemptionPolicy(plan.PreemptionPolicy.ValueString())
	obj := &schedulingv1.PriorityClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:         plan.Metadata.Name.ValueString(),
			GenerateName: plan.Metadata.GenerateName.ValueString(),
			Labels:       expandStringMap(plan.Metadata.Labels),
			Annotations:  expandStringMap(plan.Metadata.Annotations),
		},
		Value:            int32(plan.Value.ValueInt64()),
		Description:      plan.Description.ValueString(),
		GlobalDefault:    plan.GlobalDefault.ValueBool(),
		PreemptionPolicy: &preemptionPolicy,
	}

	out, err := conn.SchedulingV1().PriorityClasses().Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"error creating PriorityClass",
			fmt.Sprintf("Failed to create PriorityClass %q: %s", plan.Metadata.Name.ValueString(), err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(out.Name)
	plan.Metadata = flattenPriorityClassMetadata(
		out.ObjectMeta,
		plan.Metadata,
		meta.GetIgnoreAnnotations(),
		meta.GetIgnoreLabels(),
	)

	// Reflect server-set scalar fields
	if out.PreemptionPolicy != nil {
		plan.PreemptionPolicy = types.StringValue(string(*out.PreemptionPolicy))
	}
	plan.Description = types.StringValue(out.Description)
	plan.GlobalDefault = types.BoolValue(out.GlobalDefault)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	resp.Diagnostics.Append(resp.Identity.Set(ctx, PriorityClassIdentityModel{
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

	state.Metadata = flattenPriorityClassMetadata(
		out.ObjectMeta,
		state.Metadata,
		meta.GetIgnoreAnnotations(),
		meta.GetIgnoreLabels(),
	)

	state.Value = types.Int64Value(int64(out.Value))
	state.Description = types.StringValue(out.Description)
	state.GlobalDefault = types.BoolValue(out.GlobalDefault)
	if out.PreemptionPolicy != nil {
		state.PreemptionPolicy = types.StringValue(string(*out.PreemptionPolicy))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	resp.Diagnostics.Append(resp.Identity.Set(ctx, PriorityClassIdentityModel{
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

	// Build JSON Patch for metadata (annotations + labels) and mutable scalar fields.
	ops := make(kubernetes.PatchOperations, 0)
	ops = append(ops, kubernetes.DiffStringMap(
		"/metadata/annotations",
		toStringInterfaceMap(state.Metadata.Annotations),
		toStringInterfaceMap(plan.Metadata.Annotations),
	)...)
	ops = append(ops, kubernetes.DiffStringMap(
		"/metadata/labels",
		toStringInterfaceMap(state.Metadata.Labels),
		toStringInterfaceMap(plan.Metadata.Labels),
	)...)
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

	plan.ID = types.StringValue(out.Name)
	plan.Metadata = flattenPriorityClassMetadata(
		out.ObjectMeta,
		plan.Metadata,
		meta.GetIgnoreAnnotations(),
		meta.GetIgnoreLabels(),
	)
	plan.Description = types.StringValue(out.Description)
	plan.GlobalDefault = types.BoolValue(out.GlobalDefault)
	if out.PreemptionPolicy != nil {
		plan.PreemptionPolicy = types.StringValue(string(*out.PreemptionPolicy))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	resp.Diagnostics.Append(resp.Identity.Set(ctx, PriorityClassIdentityModel{
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
		var identityData PriorityClassIdentityModel
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

	var state PriorityClassModel
	state.ID = types.StringValue(out.Name)
	state.Metadata = flattenPriorityClassMetadata(
		out.ObjectMeta,
		MetadataModel{},
		meta.GetIgnoreAnnotations(),
		meta.GetIgnoreLabels(),
	)

	// Only set generate_name if the server actually has one; otherwise leave nil
	// to avoid a perpetual diff against configs that use name instead.
	if out.GenerateName == "" {
		state.Metadata.GenerateName = types.StringNull()
	}

	state.Value = types.Int64Value(int64(out.Value))
	state.Description = types.StringValue(out.Description)
	state.GlobalDefault = types.BoolValue(out.GlobalDefault)
	if out.PreemptionPolicy != nil {
		state.PreemptionPolicy = types.StringValue(string(*out.PreemptionPolicy))
	} else {
		state.PreemptionPolicy = types.StringValue("PreemptLowerPriority")
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	resp.Diagnostics.Append(resp.Identity.Set(ctx, PriorityClassIdentityModel{
		APIVersion: types.StringValue("scheduling.k8s.io/v1"),
		Kind:       types.StringValue("PriorityClass"),
		Name:       types.StringValue(out.Name),
	})...)
}
