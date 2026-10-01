// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	rbacv1api "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

const (
	roleBindingAPIVersion = "rbac.authorization.k8s.io/v1"
	roleBindingKind       = "RoleBinding"
)

// ── Create ────────────────────────────────────────────────────────────────────

func (r *RoleBindingV1) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RoleBindingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clients, _, metaDiags := r.sdkv2Meta()
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	objMeta, diags := common.ExpandNamespacedMetadata(ctx, plan.Metadata)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	obj := &rbacv1api.RoleBinding{
		ObjectMeta: objMeta,
		RoleRef:    expandRoleRef(plan.RoleRef[0]),
		Subjects:   expandSubjects(plan.Subject),
	}

	out, err := conn.RbacV1().RoleBindings(objMeta.Namespace).Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"error creating RoleBinding",
			fmt.Sprintf("Failed to create RoleBinding %q/%q: %s", objMeta.Namespace, objMeta.Name, err.Error()),
		)
		return
	}

	// Echo the plan, overwriting only server-assigned fields.
	// Filtering server-added metadata keys is Read's job, not Create's.
	plan.ID = types.StringValue(kubernetes.BuildId(out.ObjectMeta))
	plan.Metadata[0].Name = types.StringValue(out.Name)
	plan.Metadata[0].Namespace = types.StringValue(out.Namespace)
	plan.Metadata[0].UID = types.StringValue(string(out.UID))
	plan.Metadata[0].ResourceVersion = types.StringValue(out.ResourceVersion)
	plan.Metadata[0].Generation = types.Int64Value(out.Generation)

	// Preserve configured subject values; resolve only unknown computed fields
	// (e.g. api_group omitted by the caller) from the Kubernetes response.
	applySubjectComputedFields(&plan.Subject, out.Subjects)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{
			APIVersion: types.StringValue(roleBindingAPIVersion),
			Kind:       types.StringValue(roleBindingKind),
			Name:       types.StringValue(out.Name),
		},
		Namespace: types.StringValue(out.Namespace),
	})...)
}

// ── Read ──────────────────────────────────────────────────────────────────────

func (r *RoleBindingV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RoleBindingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid resource ID", err.Error())
		return
	}

	clients, filters, metaDiags := r.sdkv2Meta()
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	out, err := conn.RbacV1().RoleBindings(namespace).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"error reading RoleBinding",
			fmt.Sprintf("Failed to read RoleBinding %q/%q: %s", namespace, name, err.Error()),
		)
		return
	}

	// Read filters the API response against prior state and ignore lists.
	metadata, diags := common.FlattenNamespacedMetadata(ctx, out.ObjectMeta, state.Metadata,
		filters.GetIgnoreAnnotations(), filters.GetIgnoreLabels())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Metadata = metadata
	state.RoleRef = []RoleRefModel{flattenRoleRef(out.RoleRef)}
	state.Subject = flattenSubjects(out.Subjects)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{
			APIVersion: types.StringValue(roleBindingAPIVersion),
			Kind:       types.StringValue(roleBindingKind),
			Name:       types.StringValue(out.Name),
		},
		Namespace: types.StringValue(out.Namespace),
	})...)
}

// ── Update ────────────────────────────────────────────────────────────────────

func (r *RoleBindingV1) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RoleBindingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// ID is Computed-only — lives in state, not the plan. Read it from state.
	var state RoleBindingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid resource ID", err.Error())
		return
	}

	clients, _, metaDiags := r.sdkv2Meta()
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	stateMeta := state.Metadata[0]
	planMeta := plan.Metadata[0]

	// Build metadata patch ops via common helper — mirrors SDKv2 patchMetadata.
	ops := common.MetadataPatchOps("/metadata/", stateMeta.MetadataModel, planMeta.MetadataModel)
	// Add subject patch ops on top.
	ops = append(ops, patchSubjects(state.Subject, plan.Subject)...)

	patchBytes, err := json.Marshal(ops)
	if err != nil {
		resp.Diagnostics.AddError("patch serialization error", err.Error())
		return
	}

	out, err := conn.RbacV1().RoleBindings(namespace).Patch(
		ctx,
		name,
		k8stypes.JSONPatchType,
		patchBytes,
		metav1.PatchOptions{},
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"error updating RoleBinding",
			fmt.Sprintf("Failed to patch RoleBinding %q/%q: %s", namespace, name, err.Error()),
		)
		return
	}

	// Echo the plan, overwriting only server-assigned fields.
	// Filtering server-added metadata keys is Read's job, not Update's.
	plan.ID = types.StringValue(kubernetes.BuildId(out.ObjectMeta))
	plan.Metadata[0].Name = types.StringValue(out.Name)
	plan.Metadata[0].Namespace = types.StringValue(out.Namespace)
	plan.Metadata[0].UID = types.StringValue(string(out.UID))
	plan.Metadata[0].ResourceVersion = types.StringValue(out.ResourceVersion)
	plan.Metadata[0].Generation = types.Int64Value(out.Generation)

	// Preserve configured subject values; resolve only unknown computed fields
	// (e.g. api_group omitted by the caller) from the Kubernetes response.
	applySubjectComputedFields(&plan.Subject, out.Subjects)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{
			APIVersion: types.StringValue(roleBindingAPIVersion),
			Kind:       types.StringValue(roleBindingKind),
			Name:       types.StringValue(out.Name),
		},
		Namespace: types.StringValue(out.Namespace),
	})...)
}

// ── Delete ────────────────────────────────────────────────────────────────────

func (r *RoleBindingV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RoleBindingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid resource ID", err.Error())
		return
	}

	clients, _, metaDiags := r.sdkv2Meta()
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	err = conn.RbacV1().RoleBindings(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		resp.Diagnostics.AddError(
			"error deleting RoleBinding",
			fmt.Sprintf("Failed to delete RoleBinding %q/%q: %s", namespace, name, err.Error()),
		)
	}
}

// ── ImportState ───────────────────────────────────────────────────────────────

func (r *RoleBindingV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var namespace, name string

	// Accept either a plain "namespace/name" string ID or an identity object.
	if req.ID != "" {
		var err error
		namespace, name, err = kubernetes.IdParts(req.ID)
		if err != nil {
			resp.Diagnostics.AddError("invalid import ID", err.Error())
			return
		}
	} else {
		var identityData common.NamespacedResourceIdentity
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identityData)...)
		if resp.Diagnostics.HasError() {
			return
		}
		namespace = identityData.Namespace.ValueString()
		name = identityData.Name.ValueString()
	}

	clients, filters, metaDiags := r.sdkv2Meta()
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	out, err := conn.RbacV1().RoleBindings(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"error importing RoleBinding",
			fmt.Sprintf("Failed to import RoleBinding %q/%q: %s", namespace, name, err.Error()),
		)
		return
	}

	// Import uses an empty prior state — nothing declared, so nothing is exempt from filtering.
	flatMetadata, diags := common.FlattenNamespacedMetadata(ctx, out.ObjectMeta, nil,
		filters.GetIgnoreAnnotations(), filters.GetIgnoreLabels())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := RoleBindingModel{
		ID:       types.StringValue(kubernetes.BuildId(out.ObjectMeta)),
		Metadata: flatMetadata,
		RoleRef:  []RoleRefModel{flattenRoleRef(out.RoleRef)},
		Subject:  flattenSubjects(out.Subjects),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{
			APIVersion: types.StringValue(roleBindingAPIVersion),
			Kind:       types.StringValue(roleBindingKind),
			Name:       types.StringValue(out.Name),
		},
		Namespace: types.StringValue(out.Namespace),
	})...)
}
