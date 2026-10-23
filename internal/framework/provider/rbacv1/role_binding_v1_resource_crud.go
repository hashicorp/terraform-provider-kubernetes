// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	rbacv1api "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

const (
	roleBindingKind = "RoleBinding"
)

// ── Create ────────────────────────────────────────────────────────────────────

func (r *RoleBindingV1) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RoleBindingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clients, _, metaDiags := common.ResolveSDKv2Meta(r.SDKv2Meta)
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

	tflog.Debug(ctx, "Creating RoleBinding", map[string]any{"namespace": objMeta.Namespace, "name": objMeta.Name})
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
	common.PopulateNamespacedMetadataFromResponse(&plan.Metadata[0], out.ObjectMeta)

	// Preserve configured subject values; resolve only unknown computed fields
	// (e.g. api_group omitted by the caller) from the Kubernetes response.
	applySubjectComputedFields(&plan.Subject, out.Subjects)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, roleBindingIdentity(out.Namespace, out.Name))...)
}

// ── Read ──────────────────────────────────────────────────────────────────────

func (r *RoleBindingV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RoleBindingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// IdParts checks separators, but permits an empty namespace or name.
	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil || namespace == "" || name == "" {
		resp.Diagnostics.AddError("invalid resource ID",
			fmt.Sprintf("Expected format: namespace/name with a non-empty namespace and name, got: %s", state.ID.ValueString()))
		return
	}

	clients, filters, metaDiags := common.ResolveSDKv2Meta(r.SDKv2Meta)
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	tflog.Debug(ctx, "Reading RoleBinding", map[string]any{"namespace": namespace, "name": name})
	out, err := conn.RbacV1().RoleBindings(namespace).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		// Terraform before 1.12 does not send identity on refresh, but the
		// framework still requires it when Read removes a missing resource.
		if resp.Identity.Raw.IsNull() {
			resp.Diagnostics.Append(resp.Identity.Set(ctx, roleBindingIdentity(namespace, name))...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
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
	state.Subject = flattenSubjectsWithPrior(out.Subjects, state.Subject)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, roleBindingIdentity(out.Namespace, out.Name))...)
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

	clients, _, metaDiags := common.ResolveSDKv2Meta(r.SDKv2Meta)
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

	var current metav1.ObjectMeta
	if (len(stateMeta.Annotations.Elements()) == 0 && len(planMeta.Annotations.Elements()) > 0) ||
		(len(stateMeta.Labels.Elements()) == 0 && len(planMeta.Labels.Elements()) > 0) {
		out, err := conn.RbacV1().RoleBindings(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			resp.Diagnostics.AddError("error reading RoleBinding",
				fmt.Sprintf("Failed to read RoleBinding %q/%q before updating metadata: %s", namespace, name, err))
			return
		}
		current = out.ObjectMeta
	}
	ops := roleBindingMetadataPatchOps(stateMeta.MetadataModel, planMeta.MetadataModel, current)
	// Patch subjects only when they changed, mirroring SDKv2 d.HasChange("subject").
	// One replace of the whole list is simpler than index-based operations and yields
	// the same final list.
	changedSubjects := subjectsChanged(state.Subject, plan.Subject)
	if changedSubjects {
		ops = append(ops, &kubernetes.ReplaceOperation{
			Path:  "/subjects",
			Value: expandSubjects(plan.Subject),
		})
	} else {
		// Unchanged subjects keep their prior state values, including any computed
		// field the plan left unknown.
		plan.Subject = state.Subject
	}

	patchBytes, err := json.Marshal(ops)
	if err != nil {
		resp.Diagnostics.AddError("patch serialization error", err.Error())
		return
	}

	tflog.Debug(ctx, "Updating RoleBinding", map[string]any{"namespace": namespace, "name": name})
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
	common.PopulateNamespacedMetadataFromResponse(&plan.Metadata[0], out.ObjectMeta)

	// Preserve configured subject values; resolve only unknown computed fields
	// (e.g. api_group omitted by the caller) from the Kubernetes response.
	if changedSubjects {
		applySubjectComputedFields(&plan.Subject, out.Subjects)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, roleBindingIdentity(out.Namespace, out.Name))...)
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

	clients, _, metaDiags := common.ResolveSDKv2Meta(r.SDKv2Meta)
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	tflog.Debug(ctx, "Deleting RoleBinding", map[string]any{"namespace": namespace, "name": name})
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
	id := req.ID
	if id == "" {
		var identityData common.NamespacedResourceIdentity
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identityData)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if identityData.Kind.ValueString() != roleBindingKind || identityData.APIVersion.ValueString() != rbacAPIVersion {
			resp.Diagnostics.AddError("invalid identity import",
				fmt.Sprintf("Expected kind %q and api_version %q, got kind %q and api_version %q.",
					roleBindingKind, rbacAPIVersion, identityData.Kind.ValueString(), identityData.APIVersion.ValueString()))
			return
		}
		namespace := identityData.Namespace.ValueString()
		if namespace == "" {
			namespace = metav1.NamespaceDefault
		}
		id = kubernetes.BuildId(metav1.ObjectMeta{Namespace: namespace, Name: identityData.Name.ValueString()})
	}

	// Read validates the namespaced ID and populates state and identity.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}
