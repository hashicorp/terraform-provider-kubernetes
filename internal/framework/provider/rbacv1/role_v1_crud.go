// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	rbacv1api "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

const (
	rbacAPIVersion   = "rbac.authorization.k8s.io/v1"
	roleKind         = "Role"
	defaultNamespace = "default"
)

func (r *Role) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RoleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The metadata ListNestedBlock's listvalidator.SizeBetween(1, 1) has
	// already rejected any config with zero or multiple metadata blocks
	// before Create runs.
	planMeta := plan.Metadata[0]

	meta := r.SDKv2Meta().(kubernetes.KubeClientsets)
	conn, err := meta.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	metadata := expandMetadata(planMeta)
	rules, diags := expandPolicyRules(ctx, plan.Rule)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	role := &rbacv1api.Role{
		ObjectMeta: metadata,
		Rules:      rules,
	}

	out, err := conn.RbacV1().Roles(metadata.Namespace).Create(ctx, role, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"error creating Role",
			fmt.Sprintf("Failed to create role %q in namespace %q: %s", metadata.Name, metadata.Namespace, err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(buildID(out.Namespace, out.Name))
	plan.Metadata = []MetadataModel{*flattenMetadata(out.ObjectMeta, planMeta, meta.GetIgnoreAnnotations(), meta.GetIgnoreLabels())}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	identity := RoleIdentityModel{
		APIVersion: types.StringValue(rbacAPIVersion),
		Kind:       types.StringValue(roleKind),
		Namespace:  types.StringValue(out.Namespace),
		Name:       types.StringValue(out.Name),
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
}

func (r *Role) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RoleModel
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

	namespace, name, err := parseID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid resource ID", err.Error())
		return
	}

	role, err := conn.RbacV1().Roles(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"error reading Role",
			fmt.Sprintf("Failed to read role %q in namespace %q: %s", name, namespace, err.Error()),
		)
		return
	}

	var currentMeta MetadataModel
	if len(state.Metadata) > 0 {
		currentMeta = state.Metadata[0]
	}

	state.ID = types.StringValue(buildID(role.Namespace, role.Name))
	state.Metadata = []MetadataModel{*flattenMetadata(role.ObjectMeta, currentMeta, meta.GetIgnoreAnnotations(), meta.GetIgnoreLabels())}

	rules, diags := flattenPolicyRules(role.Rules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Rule = rules

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	identity := RoleIdentityModel{
		APIVersion: types.StringValue(rbacAPIVersion),
		Kind:       types.StringValue(roleKind),
		Namespace:  types.StringValue(role.Namespace),
		Name:       types.StringValue(role.Name),
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
}

func (r *Role) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state RoleModel
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

	namespace, name, err := parseID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid resource ID", err.Error())
		return
	}

	planMeta := plan.Metadata[0]
	var stateMeta MetadataModel
	if len(state.Metadata) > 0 {
		stateMeta = state.Metadata[0]
	}

	ops := buildMetadataPatch(planMeta, stateMeta)

	if !rulesEqual(plan.Rule, state.Rule) {
		rules, diags := expandPolicyRules(ctx, plan.Rule)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		ops = append(ops, &kubernetes.ReplaceOperation{
			Path:  "/rules",
			Value: rules,
		})
	}

	if len(ops) == 0 {
		out, err := conn.RbacV1().Roles(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			resp.Diagnostics.AddError(
				"error reading Role",
				fmt.Sprintf("Failed to read role %q in namespace %q: %s", name, namespace, err.Error()),
			)
			return
		}
		plan.ID = types.StringValue(buildID(out.Namespace, out.Name))
		plan.Metadata = []MetadataModel{*flattenMetadata(out.ObjectMeta, planMeta, meta.GetIgnoreAnnotations(), meta.GetIgnoreLabels())}
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	patchBytes, err := ops.MarshalJSON()
	if err != nil {
		resp.Diagnostics.AddError("failed to marshal patch", err.Error())
		return
	}

	out, err := conn.RbacV1().Roles(namespace).Patch(ctx, name, pkgApi.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"error updating Role",
			fmt.Sprintf("Failed to update role %q in namespace %q: %s", name, namespace, err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(buildID(out.Namespace, out.Name))
	plan.Metadata = []MetadataModel{*flattenMetadata(out.ObjectMeta, planMeta, meta.GetIgnoreAnnotations(), meta.GetIgnoreLabels())}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *Role) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RoleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := r.SDKv2Meta().(kubernetes.KubeClientsets).MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	namespace, name, err := parseID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid resource ID", err.Error())
		return
	}

	err = conn.RbacV1().Roles(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		resp.Diagnostics.AddError(
			"error deleting Role",
			fmt.Sprintf("Failed to delete role %q in namespace %q: %s", name, namespace, err.Error()),
		)
		return
	}
}

func (r *Role) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var namespace, name string

	if req.ID != "" {
		var err error
		namespace, name, err = parseID(req.ID)
		if err != nil {
			resp.Diagnostics.AddError(
				"invalid import ID",
				fmt.Sprintf("Expected format: namespace/name, got: %s", req.ID),
			)
			return
		}
	} else {
		var identityData RoleIdentityModel
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identityData)...)
		if resp.Diagnostics.HasError() {
			return
		}
		namespace = identityData.Namespace.ValueString()
		if namespace == "" {
			namespace = defaultNamespace
		}
		name = identityData.Name.ValueString()
	}

	meta := r.SDKv2Meta().(kubernetes.KubeClientsets)
	conn, err := meta.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	role, err := conn.RbacV1().Roles(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"error importing Role",
			fmt.Sprintf("Failed to import role %q in namespace %q: %s", name, namespace, err.Error()),
		)
		return
	}

	var state RoleModel
	state.ID = types.StringValue(buildID(role.Namespace, role.Name))
	// Nothing is "already managed" yet on a fresh import, so internal keys
	// and ignore_annotations/ignore_labels matches are always filtered out.
	state.Metadata = []MetadataModel{*flattenMetadata(role.ObjectMeta, MetadataModel{}, meta.GetIgnoreAnnotations(), meta.GetIgnoreLabels())}

	rules, diags := flattenPolicyRules(role.Rules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Rule = rules

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	identity := RoleIdentityModel{
		APIVersion: types.StringValue(rbacAPIVersion),
		Kind:       types.StringValue(roleKind),
		Namespace:  types.StringValue(role.Namespace),
		Name:       types.StringValue(role.Name),
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
}
