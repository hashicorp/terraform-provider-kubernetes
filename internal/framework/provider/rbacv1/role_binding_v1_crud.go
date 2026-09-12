// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	rbacv1api "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

// buildID returns the canonical "namespace/name" ID used as the Terraform resource ID.
func buildID(namespace, name string) string {
	return namespace + "/" + name
}

// splitID splits a "namespace/name" ID into its components.
func splitID(id string) (namespace, name string, err error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid RoleBinding ID %q: expected \"namespace/name\"", id)
	}
	return parts[0], parts[1], nil
}

// ── Create ────────────────────────────────────────────────────────────────────

func (r *RoleBindingV1) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RoleBindingModel
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
	obj := &rbacv1api.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:         m.Name.ValueString(),
			GenerateName: m.GenerateName.ValueString(),
			Namespace:    m.Namespace.ValueString(),
			Labels:       expandStringMap(m.Labels),
			Annotations:  expandStringMap(m.Annotations),
		},
		RoleRef:  expandRoleRef(plan.RoleRef[0]),
		Subjects: expandSubjects(plan.Subject),
	}

	out, err := conn.RbacV1().RoleBindings(obj.Namespace).Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"error creating RoleBinding",
			fmt.Sprintf("Failed to create RoleBinding %q/%q: %s", obj.Namespace, obj.Name, err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(buildID(out.Namespace, out.Name))

	var currentMeta NamespacedMetadataModel
	if len(plan.Metadata) > 0 {
		currentMeta = plan.Metadata[0]
	}
	plan.Metadata = []NamespacedMetadataModel{flattenNamespacedMetadata(
		out.ObjectMeta,
		currentMeta,
		meta.GetIgnoreAnnotations(),
		meta.GetIgnoreLabels(),
	)}
	plan.RoleRef = []RoleRefModel{flattenRoleRef(out.RoleRef)}
	plan.Subject = flattenSubjects(out.Subjects)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, RoleBindingIdentityModel{
		APIVersion: types.StringValue("rbac.authorization.k8s.io/v1"),
		Kind:       types.StringValue("RoleBinding"),
		Namespace:  types.StringValue(out.Namespace),
		Name:       types.StringValue(out.Name),
	})...)
}

// ── Read ──────────────────────────────────────────────────────────────────────

func (r *RoleBindingV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RoleBindingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	namespace, name, err := splitID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid resource ID", err.Error())
		return
	}

	meta := r.SDKv2Meta().(kubernetes.KubeClientsets)
	conn, err := meta.MainClientset()
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

	var currentMeta NamespacedMetadataModel
	if len(state.Metadata) > 0 {
		currentMeta = state.Metadata[0]
	}
	state.Metadata = []NamespacedMetadataModel{flattenNamespacedMetadata(
		out.ObjectMeta,
		currentMeta,
		meta.GetIgnoreAnnotations(),
		meta.GetIgnoreLabels(),
	)}
	state.RoleRef = []RoleRefModel{flattenRoleRef(out.RoleRef)}
	state.Subject = flattenSubjects(out.Subjects)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, RoleBindingIdentityModel{
		APIVersion: types.StringValue("rbac.authorization.k8s.io/v1"),
		Kind:       types.StringValue("RoleBinding"),
		Namespace:  types.StringValue(out.Namespace),
		Name:       types.StringValue(out.Name),
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

	namespace, name, err := splitID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid resource ID", err.Error())
		return
	}

	meta := r.SDKv2Meta().(kubernetes.KubeClientsets)
	conn, err := meta.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	stateMeta := state.Metadata[0]
	planMeta := plan.Metadata[0]

	// Build JSON Patch: metadata (annotations + labels) and subject list.
	// Only call DiffStringMap when at least one side is non-empty. When both
	// sides are nil/empty, DiffStringMap would emit an Add operation that
	// replaces the entire map with {}, erasing any externally managed ignored keys.
	ops := make(kubernetes.PatchOperations, 0)
	if len(stateMeta.Annotations) > 0 || len(planMeta.Annotations) > 0 {
		ops = append(ops, kubernetes.DiffStringMap(
			"/metadata/annotations",
			toStringInterfaceMap(stateMeta.Annotations),
			toStringInterfaceMap(planMeta.Annotations),
		)...)
	}
	if len(stateMeta.Labels) > 0 || len(planMeta.Labels) > 0 {
		ops = append(ops, kubernetes.DiffStringMap(
			"/metadata/labels",
			toStringInterfaceMap(stateMeta.Labels),
			toStringInterfaceMap(planMeta.Labels),
		)...)
	}
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

	plan.ID = types.StringValue(buildID(out.Namespace, out.Name))
	plan.Metadata = []NamespacedMetadataModel{flattenNamespacedMetadata(
		out.ObjectMeta,
		planMeta,
		meta.GetIgnoreAnnotations(),
		meta.GetIgnoreLabels(),
	)}
	plan.RoleRef = []RoleRefModel{flattenRoleRef(out.RoleRef)}
	plan.Subject = flattenSubjects(out.Subjects)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, RoleBindingIdentityModel{
		APIVersion: types.StringValue("rbac.authorization.k8s.io/v1"),
		Kind:       types.StringValue("RoleBinding"),
		Namespace:  types.StringValue(out.Namespace),
		Name:       types.StringValue(out.Name),
	})...)
}

// ── Delete ────────────────────────────────────────────────────────────────────

func (r *RoleBindingV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RoleBindingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	namespace, name, err := splitID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid resource ID", err.Error())
		return
	}

	conn, err := r.SDKv2Meta().(kubernetes.KubeClientsets).MainClientset()
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
		namespace, name, err = splitID(req.ID)
		if err != nil {
			resp.Diagnostics.AddError("invalid import ID", err.Error())
			return
		}
	} else {
		var identityData RoleBindingIdentityModel
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identityData)...)
		if resp.Diagnostics.HasError() {
			return
		}
		namespace = identityData.Namespace.ValueString()
		name = identityData.Name.ValueString()
	}

	meta := r.SDKv2Meta().(kubernetes.KubeClientsets)
	conn, err := meta.MainClientset()
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

	flatMeta := flattenNamespacedMetadata(
		out.ObjectMeta,
		NamespacedMetadataModel{},
		meta.GetIgnoreAnnotations(),
		meta.GetIgnoreLabels(),
	)

	// If the server has no generate_name, set to null to avoid perpetual diff
	// against configs that use name instead of generate_name.
	if out.GenerateName == "" {
		flatMeta.GenerateName = types.StringNull()
	}

	state := RoleBindingModel{
		ID:       types.StringValue(buildID(out.Namespace, out.Name)),
		Metadata: []NamespacedMetadataModel{flatMeta},
		RoleRef:  []RoleRefModel{flattenRoleRef(out.RoleRef)},
		Subject:  flattenSubjects(out.Subjects),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, RoleBindingIdentityModel{
		APIVersion: types.StringValue("rbac.authorization.k8s.io/v1"),
		Kind:       types.StringValue("RoleBinding"),
		Namespace:  types.StringValue(out.Namespace),
		Name:       types.StringValue(out.Name),
	})...)
}
