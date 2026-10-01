// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package storagev1

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
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

	clients, _, diags := r.sdkv2Meta()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	objMeta, expandDiags := common.ExpandMetadata(ctx, plan.Metadata)
	resp.Diagnostics.Append(expandDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	reclaimPolicy := corev1.PersistentVolumeReclaimPolicy(plan.ReclaimPolicy.ValueString())
	volumeBindingMode := storagev1api.VolumeBindingMode(plan.VolumeBindingMode.ValueString())
	allowVolumeExpansion := plan.AllowVolumeExpansion.ValueBool()

	var parameters map[string]string
	if !plan.Parameters.IsNull() && !plan.Parameters.IsUnknown() {
		resp.Diagnostics.Append(plan.Parameters.ElementsAs(ctx, &parameters, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	obj := &storagev1api.StorageClass{
		ObjectMeta:           objMeta,
		Provisioner:          plan.StorageProvisioner.ValueString(),
		ReclaimPolicy:        &reclaimPolicy,
		VolumeBindingMode:    &volumeBindingMode,
		AllowVolumeExpansion: &allowVolumeExpansion,
		Parameters:           parameters,
		MountOptions:         expandMountOptions(ctx, plan.MountOptions),
		AllowedTopologies:    expandAllowedTopologies(ctx, plan.AllowedTopologies),
	}

	out, err := conn.StorageV1().StorageClasses().Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"error creating StorageClass",
			fmt.Sprintf("Failed to create StorageClass %q: %s", objMeta.Name, err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(out.Name)
	populateMetadataFromResponse(&plan, out.ObjectMeta)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, storageClassIdentity(out.Name))...)
}

func (r *StorageClassV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state StorageClassModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clients, filters, diags := r.sdkv2Meta()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := clients.MainClientset()
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

	// Read uses the API response as source of truth and filters.
	flatMeta, flatDiags := common.FlattenMetadata(ctx, out.ObjectMeta, state.Metadata, filters.GetIgnoreAnnotations(), filters.GetIgnoreLabels())
	resp.Diagnostics.Append(flatDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Metadata = flatMeta
	setComputedFields(ctx, &state, out)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Identity == nil {
		return
	}

	// Always set identity unconditionally. UpgradeIdentity returns all-null for v0 state
	// (no prior identity), and framework v1.16.1+ allows Read to populate it afterwards.
	// Guarding on the prior identity value caused "Missing Resource Identity After Read"
	// on every migration test (SDKv2 state carries no identity at all).
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

	clients, _, diags := r.sdkv2Meta()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	name := state.ID.ValueString()

	// Build JSON Patch: metadata (annotations + labels) and allow_volume_expansion.
	// Only these two items are mutable; everything else is ForceNew.
	ops := common.MetadataPatchOps("/metadata/", state.Metadata[0], plan.Metadata[0])
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

	// Update echoes the plan for user-controlled fields; only server-assigned
	// metadata fields (name, uid, resource_version, generation) come from the response.
	plan.ID = types.StringValue(out.Name)
	plan.Metadata[0].MetadataBase.Name = types.StringValue(out.Name)
	plan.Metadata[0].MetadataBase.UID = types.StringValue(string(out.UID))
	plan.Metadata[0].MetadataBase.ResourceVersion = types.StringValue(out.ResourceVersion)
	plan.Metadata[0].MetadataBase.Generation = types.Int64Value(out.Generation)
	setComputedFields(ctx, &plan, out)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Identity == nil {
		return
	}

	// Set identity unconditionally — same reasoning as Read.
	resp.Diagnostics.Append(resp.Identity.Set(ctx, storageClassIdentity(out.Name))...)
}

func (r *StorageClassV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state StorageClassModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clients, _, diags := r.sdkv2Meta()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := clients.MainClientset()
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
		var identityData common.ResourceIdentity
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identityData)...)
		if resp.Diagnostics.HasError() {
			return
		}
		name = identityData.Name.ValueString()
	}

	clients, filters, diags := r.sdkv2Meta()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := clients.MainClientset()
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

	// Import uses API response as source of truth (same as Read).
	flatMeta, flatDiags := common.FlattenMetadata(ctx, out.ObjectMeta, nil, filters.GetIgnoreAnnotations(), filters.GetIgnoreLabels())
	resp.Diagnostics.Append(flatDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state StorageClassModel
	state.ID = types.StringValue(out.Name)
	state.Metadata = flatMeta
	state.MountOptions = types.SetNull(types.StringType)
	setComputedFields(ctx, &state, out)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, storageClassIdentity(out.Name))...)
}

// ── Private helpers ───────────────────────────────────────────────────────────

// sdkv2Meta resolves the SDKv2 provider metadata into the two interfaces this resource needs.
// The call is deferred until the request rather than made in Configure, because the mux server
// configures the SDKv2 provider independently.
func (r *StorageClassV1) sdkv2Meta() (kubernetes.KubeClientsets, kubernetes.MetadataFilters, diag.Diagnostics) {
	meta := r.SDKv2Meta()
	clients := meta.(kubernetes.KubeClientsets)
	filters := meta.(kubernetes.MetadataFilters)
	return clients, filters, nil
}

// populateMetadataFromResponse overwrites only server-assigned metadata fields
// in the plan from the API response, preserving the plan's user-controlled fields
// (labels, annotations). Called after Create — not Update (Update uses plan directly
// plus MetadataPatchOps).
//
// Takes the model by pointer so that field assignments land in the caller's model.
// A value receiver would silently discard assignments to non-slice fields.
func populateMetadataFromResponse(plan *StorageClassModel, obj metav1.ObjectMeta) {
	if len(plan.Metadata) == 0 {
		return
	}
	plan.Metadata[0].MetadataBase.Name = types.StringValue(obj.Name)
	plan.Metadata[0].MetadataBase.UID = types.StringValue(string(obj.UID))
	plan.Metadata[0].MetadataBase.ResourceVersion = types.StringValue(obj.ResourceVersion)
	plan.Metadata[0].MetadataBase.Generation = types.Int64Value(obj.Generation)
}

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

	// parameters: reflect server data when present; otherwise ensure a typed null.
	// A zero-value types.Map{} has no element type and causes a decode error when
	// stored to state — e.g. on ImportState where Parameters starts uninitialized.
	if len(out.Parameters) > 0 {
		p, _ := types.MapValueFrom(ctx, types.StringType, out.Parameters)
		m.Parameters = p
	} else {
		// Preserve a non-null empty map if the plan had one (user explicitly configured
		// an empty parameters block), but always ensure a typed value, never zero-value.
		if m.Parameters.IsNull() || m.Parameters.ElementType(ctx) == nil {
			m.Parameters = types.MapNull(types.StringType)
		}
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
func storageClassIdentity(name string) common.ResourceIdentity {
	return common.ResourceIdentity{
		APIVersion: types.StringValue(storageClassAPIVersion),
		Kind:       types.StringValue(storageClassKind),
		Name:       types.StringValue(name),
	}
}
