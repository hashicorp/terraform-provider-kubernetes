// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package storagev1

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	corev1 "k8s.io/api/core/v1"
	storagev1api "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

func (r *StorageClassV1) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan StorageClassModel
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

	m := plan.Metadata[0]
	reclaimPolicy := corev1.PersistentVolumeReclaimPolicy(plan.ReclaimPolicy.ValueString())
	volumeBindingMode := storagev1api.VolumeBindingMode(plan.VolumeBindingMode.ValueString())
	allowVolumeExpansion := plan.AllowVolumeExpansion.ValueBool()

	obj := &storagev1api.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:         m.Name.ValueString(),
			GenerateName: m.GenerateName.ValueString(),
			Labels:       expandStringMap(m.Labels),
			Annotations:  expandStringMap(m.Annotations),
		},
		Provisioner:          plan.StorageProvisioner.ValueString(),
		ReclaimPolicy:        &reclaimPolicy,
		VolumeBindingMode:    &volumeBindingMode,
		AllowVolumeExpansion: &allowVolumeExpansion,
		Parameters:           expandStringMap(plan.Parameters),
		MountOptions:         expandMountOptions(ctx, plan.MountOptions),
		AllowedTopologies:    expandAllowedTopologies(ctx, plan.AllowedTopologies),
	}

	out, err := conn.StorageV1().StorageClasses().Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"error creating StorageClass",
			fmt.Sprintf("Failed to create StorageClass %q: %s", m.Name.ValueString(), err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(out.Name)
	plan.Metadata = []MetadataModel{flattenMetadata(
		out.ObjectMeta, m,
		meta.GetIgnoreAnnotations(),
		meta.GetIgnoreLabels(),
	)}
	setComputedFields(ctx, &plan, out)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, storageClassIdentity(out.Name))...)
}

func (r *StorageClassV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state StorageClassModel
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
	out, err := conn.StorageV1().StorageClasses().Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"error reading StorageClass",
			fmt.Sprintf("Failed to read StorageClass %q: %s", name, err.Error()),
		)
		return
	}

	var currentMeta MetadataModel
	if len(state.Metadata) > 0 {
		currentMeta = state.Metadata[0]
	}
	state.Metadata = []MetadataModel{flattenMetadata(
		out.ObjectMeta, currentMeta,
		meta.GetIgnoreAnnotations(),
		meta.GetIgnoreLabels(),
	)}
	setComputedFields(ctx, &state, out)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, storageClassIdentity(out.Name))...)
}

func (r *StorageClassV1) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan StorageClassModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// id is Computed-only — it lives in state, not in the plan.
	var state StorageClassModel
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
	stateMeta := state.Metadata[0]
	planMeta := plan.Metadata[0]

	// Build JSON Patch: metadata (annotations + labels) and allow_volume_expansion.
	// Only these two items are mutable; everything else is ForceNew.
	ops := make(kubernetes.PatchOperations, 0)
	ops = append(ops, kubernetes.DiffStringMap(
		"/metadata/annotations",
		toStringInterfaceMap(stateMeta.Annotations),
		toStringInterfaceMap(planMeta.Annotations),
	)...)
	ops = append(ops, kubernetes.DiffStringMap(
		"/metadata/labels",
		toStringInterfaceMap(stateMeta.Labels),
		toStringInterfaceMap(planMeta.Labels),
	)...)
	ops = append(ops, &kubernetes.ReplaceOperation{
		Path:  "/allowVolumeExpansion",
		Value: plan.AllowVolumeExpansion.ValueBool(),
	})

	patchBytes, err := json.Marshal(ops)
	if err != nil {
		resp.Diagnostics.AddError("patch serialization error", err.Error())
		return
	}

	out, err := conn.StorageV1().StorageClasses().Patch(
		ctx,
		name,
		k8stypes.JSONPatchType,
		patchBytes,
		metav1.PatchOptions{},
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"error updating StorageClass",
			fmt.Sprintf("Failed to patch StorageClass %q: %s", name, err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(out.Name)
	plan.Metadata = []MetadataModel{flattenMetadata(
		out.ObjectMeta, planMeta,
		meta.GetIgnoreAnnotations(),
		meta.GetIgnoreLabels(),
	)}
	setComputedFields(ctx, &plan, out)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, storageClassIdentity(out.Name))...)
}

func (r *StorageClassV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state StorageClassModel
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
	err = conn.StorageV1().StorageClasses().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		resp.Diagnostics.AddError(
			"error deleting StorageClass",
			fmt.Sprintf("Failed to delete StorageClass %q: %s", name, err.Error()),
		)
	}
}

func (r *StorageClassV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var name string

	if req.ID != "" {
		name = req.ID
	} else {
		var identityData StorageClassIdentityModel
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

	out, err := conn.StorageV1().StorageClasses().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"error importing StorageClass",
			fmt.Sprintf("Failed to import StorageClass %q: %s", name, err.Error()),
		)
		return
	}

	flatMeta := flattenMetadata(
		out.ObjectMeta,
		MetadataModel{},
		meta.GetIgnoreAnnotations(),
		meta.GetIgnoreLabels(),
	)
	// Only set generate_name if the server actually has one; otherwise leave null
	// to avoid a perpetual diff against configs that use name instead.
	if out.GenerateName == "" {
		flatMeta.GenerateName = types.StringNull()
	}

	var state StorageClassModel
	state.ID = types.StringValue(out.Name)
	state.Metadata = []MetadataModel{flatMeta}
	// Initialise mount_options to a typed empty set so setComputedFields has
	// a non-null baseline to work from. Import has no prior plan state, so
	// we must pre-populate optional Set attributes with their zero value.
	state.MountOptions = types.SetValueMust(types.StringType, []attr.Value{})
	setComputedFields(ctx, &state, out)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, storageClassIdentity(out.Name))...)
}

// ── Private helpers ───────────────────────────────────────────────────────────

// setComputedFields populates all Computed fields from the Kubernetes API
// response into the model. Called after Create, Read, Update and ImportState.
//
// For optional-only fields (mount_options, allowed_topologies, parameters) we
// must preserve null when the config omitted the field. The Framework requires
// that the post-apply state matches the planned value for optional attributes:
// if the plan had null, the state must also be null — not an empty collection.
// We achieve this by only overwriting those fields when the server actually
// returned non-empty data OR when the model already holds a non-null value
// (meaning the user configured it and we should reflect the server's copy).
func setComputedFields(ctx context.Context, m *StorageClassModel, out *storagev1api.StorageClass) {
	m.StorageProvisioner = types.StringValue(out.Provisioner)

	if out.ReclaimPolicy != nil {
		m.ReclaimPolicy = types.StringValue(string(*out.ReclaimPolicy))
	}
	if out.VolumeBindingMode != nil {
		m.VolumeBindingMode = types.StringValue(string(*out.VolumeBindingMode))
	}
	if out.AllowVolumeExpansion != nil {
		m.AllowVolumeExpansion = types.BoolValue(*out.AllowVolumeExpansion)
	}

	// parameters: only write if server returned data or user had it configured.
	if len(out.Parameters) > 0 {
		m.Parameters = flattenStringMap(out.Parameters)
	} else if m.Parameters != nil {
		// User had parameters set; server echoed empty — clear to nil.
		m.Parameters = nil
	}

	// mount_options: only write if server returned data or user had it configured.
	// If the user omitted mount_options (null in plan) and the server returns
	// an empty list, leave the model value as-is (null) to avoid the
	// "was null, but now empty set" inconsistency error.
	if len(out.MountOptions) > 0 {
		m.MountOptions = flattenMountOptions(ctx, out.MountOptions)
	} else if !m.MountOptions.IsNull() {
		// User had mount_options set; server echoed empty — reflect as empty set.
		m.MountOptions = flattenMountOptions(ctx, out.MountOptions)
	}
	// else: user omitted mount_options (null) and server returned empty — leave null.

	// allowed_topologies: same null-preservation logic.
	if len(out.AllowedTopologies) > 0 {
		m.AllowedTopologies = flattenAllowedTopologies(ctx, out.AllowedTopologies)
	} else if len(m.AllowedTopologies) > 0 {
		// User had topologies set; server echoed empty — clear.
		m.AllowedTopologies = nil
	}
	// else: user omitted allowed_topologies — leave nil.
}

// storageClassIdentity returns the identity model for a given StorageClass name.
func storageClassIdentity(name string) StorageClassIdentityModel {
	return StorageClassIdentityModel{
		APIVersion: types.StringValue("storage.k8s.io/v1"),
		Kind:       types.StringValue("StorageClass"),
		Name:       types.StringValue(name),
	}
}
