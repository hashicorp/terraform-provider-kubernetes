// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package schedulingv1

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
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
	plan.ID = types.StringValue(out.Name)
	common.PopulateMetadataFromResponse(&plan.Metadata[0], out.ObjectMeta)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, priorityClassIdentity(out.Name))...)
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
	if err != nil {
		if apierrors.IsNotFound(err) {
			// The framework requires an identity even when Read removes the resource.
			// Terraform before 1.12 sends none (null), and state upgraded from SDKv2
			// carries the all-null identity common.UpgradeIdentity returns; the
			// framework rejects both, so set it from the name in state.
			if resp.Identity != nil && (resp.Identity.Raw.IsNull() || resp.Identity.Raw.IsFullyNull()) {
				resp.Diagnostics.Append(resp.Identity.Set(ctx, priorityClassIdentity(name))...)
				if resp.Diagnostics.HasError() {
					return
				}
			}
			resp.State.RemoveResource(ctx)
			return
		}
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

	state.ID = types.StringValue(out.Name)
	state.Metadata = flatMeta
	state.Value = types.Int64Value(int64(out.Value))
	state.Description = types.StringValue(out.Description)
	state.GlobalDefault = types.BoolValue(out.GlobalDefault)
	// The API server defaults preemptionPolicy, so nil only appears on servers
	// older than the field. Fall back to the same default.
	state.PreemptionPolicy = types.StringValue(string(corev1.PreemptLowerPriority))
	if out.PreemptionPolicy != nil {
		state.PreemptionPolicy = types.StringValue(string(*out.PreemptionPolicy))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, priorityClassIdentity(out.Name))...)
}

func (r *PriorityClassV1) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state PriorityClassModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
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

	// id is Computed-only with UseStateForUnknown, so state is the authoritative source.
	name := state.ID.ValueString()

	ops := common.MetadataPatchOps("/metadata/", state.Metadata[0], plan.Metadata[0])

	// Both fields are omitempty in the API, so an empty description or a false
	// globalDefault can be absent from the stored object: "add" sets a member whether
	// or not it exists, where "replace" requires it to. Each op is sent only when the
	// field changed, as SDKv2 did with d.HasChange, so an update that touches only
	// labels does not overwrite a value changed outside Terraform after planning.
	if !plan.Description.Equal(state.Description) {
		ops = append(ops, &kubernetes.AddOperation{
			Path:  "/description",
			Value: plan.Description.ValueString(),
		})
	}
	if !plan.GlobalDefault.Equal(state.GlobalDefault) {
		ops = append(ops, &kubernetes.AddOperation{
			Path:  "/globalDefault",
			Value: plan.GlobalDefault.ValueBool(),
		})
	}

	var out *schedulingv1.PriorityClass
	if len(ops) == 0 {
		// Nothing to send: the plan differs from state only in Terraform-side values
		// (for example "" → null for generate_name after an upgrade). Read the object
		// instead of sending an empty PATCH, which would still need patch RBAC.
		out, err = conn.SchedulingV1().PriorityClasses().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			resp.Diagnostics.AddError(
				"error reading PriorityClass",
				fmt.Sprintf("Failed to read PriorityClass %q: %s", name, err.Error()),
			)
			return
		}
	} else {
		patchBytes, err := ops.MarshalJSON()
		if err != nil {
			resp.Diagnostics.AddError("failed to marshal patch", err.Error())
			return
		}

		out, err = conn.SchedulingV1().PriorityClasses().Patch(ctx, name, k8stypes.JSONPatchType, patchBytes, metav1.PatchOptions{})
		if err != nil {
			resp.Diagnostics.AddError(
				"error updating PriorityClass",
				fmt.Sprintf("Failed to patch PriorityClass %q: %s", name, err.Error()),
			)
			return
		}
	}
	plan.ID = types.StringValue(out.Name)
	common.PopulateMetadataFromResponse(&plan.Metadata[0], out.ObjectMeta)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, priorityClassIdentity(out.Name))...)
}

func (r *PriorityClassV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
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
	err = conn.SchedulingV1().PriorityClasses().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		resp.Diagnostics.AddError(
			"error deleting PriorityClass",
			fmt.Sprintf("Failed to delete PriorityClass %q: %s", name, err.Error()),
		)
	}
}

func (r *PriorityClassV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	name := req.ID
	if name == "" {
		var identityData common.ResourceIdentity
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identityData)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if identityData.Kind.ValueString() != priorityClassKind || identityData.APIVersion.ValueString() != priorityClassAPIVersion {
			resp.Diagnostics.AddError("invalid identity import",
				fmt.Sprintf("Expected kind %q and api_version %q, got kind %q and api_version %q.",
					priorityClassKind, priorityClassAPIVersion, identityData.Kind.ValueString(), identityData.APIVersion.ValueString()))
			return
		}
		name = identityData.Name.ValueString()
		if name == "" {
			resp.Diagnostics.AddError("invalid identity import", "Identity attribute \"name\" must not be empty.")
			return
		}
	}

	// Read fetches the object and populates state and identity.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), name)...)
}
