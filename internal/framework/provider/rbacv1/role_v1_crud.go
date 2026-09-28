// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	rbacv1api "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

const (
	rbacAPIVersion = "rbac.authorization.k8s.io/v1"
	roleKind       = "Role"
)

func (r *Role) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RoleModel
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

	metadata, diags := common.ExpandNamespacedMetadata(ctx, plan.Metadata)
	resp.Diagnostics.Append(diags...)
	rules, ruleDiags := expandPolicyRules(ctx, plan.Rule)
	resp.Diagnostics.Append(ruleDiags...)
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

	populateIDAndMetadataFromResponse(&plan, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	identity := common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{
			APIVersion: types.StringValue(rbacAPIVersion),
			Kind:       types.StringValue(roleKind),
			Name:       types.StringValue(out.Name),
		},
		Namespace: types.StringValue(out.Namespace),
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

	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
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

	filters := r.SDKv2Meta().(kubernetes.MetadataFilters)
	currentMeta := state.Metadata

	state.ID = types.StringValue(kubernetes.BuildId(role.ObjectMeta))
	flattened, metaDiags := common.FlattenNamespacedMetadata(ctx, role.ObjectMeta, currentMeta, filters.GetIgnoreAnnotations(), filters.GetIgnoreLabels())
	resp.Diagnostics.Append(metaDiags...)
	state.Metadata = flattened

	rules, diags := flattenPolicyRules(role.Rules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Rule = rules

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	identity := common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{
			APIVersion: types.StringValue(rbacAPIVersion),
			Kind:       types.StringValue(roleKind),
			Name:       types.StringValue(role.Name),
		},
		Namespace: types.StringValue(role.Namespace),
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

	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid resource ID", err.Error())
		return
	}

	planMeta := plan.Metadata[0]
	var stateMeta common.NamespacedMetadataModel
	if len(state.Metadata) > 0 {
		stateMeta = state.Metadata[0]
	}

	ops := common.MetadataPatchOps("/metadata/", stateMeta.MetadataModel, planMeta.MetadataModel)

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
		populateIDAndMetadataFromResponse(&plan, out)
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

	populateIDAndMetadataFromResponse(&plan, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func populateIDAndMetadataFromResponse(plan *RoleModel, out *rbacv1api.Role) {
	// State is built from the plan, with only server-assigned fields overwritten from the
	// response. Labels and annotations are deliberately left as the plan wrote them: the
	// API server and admission webhooks add keys of their own, and echoing those back
	// would not match the plan, which Terraform rejects as an inconsistent result.
	// Filtering them is Read's job.
	plan.ID = types.StringValue(kubernetes.BuildId(out.ObjectMeta))
	plan.Metadata[0].Name = types.StringValue(out.Name)
	plan.Metadata[0].Namespace = types.StringValue(out.Namespace)
	plan.Metadata[0].UID = types.StringValue(string(out.UID))
	plan.Metadata[0].ResourceVersion = types.StringValue(out.ResourceVersion)
	plan.Metadata[0].Generation = types.Int64Value(out.Generation)
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

	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
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
		namespace, name, err = kubernetes.IdParts(req.ID)
		if err != nil {
			resp.Diagnostics.AddError(
				"invalid import ID",
				fmt.Sprintf("Expected format: namespace/name, got: %s", req.ID),
			)
			return
		}
	} else {
		var identityData common.NamespacedResourceIdentity
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identityData)...)
		if resp.Diagnostics.HasError() {
			return
		}
		namespace = identityData.Namespace.ValueString()
		if namespace == "" {
			resp.Diagnostics.AddError(
				"invalid identity import",
				"namespace is required when importing by identity; "+
					"provide a namespace in the identity block or use the "+
					"string import format: namespace/name",
			)
			return
		}
		name = identityData.Name.ValueString()
	}

	meta := r.SDKv2Meta().(kubernetes.KubeClientsets)
	filters := r.SDKv2Meta().(kubernetes.MetadataFilters)
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
	state.ID = types.StringValue(kubernetes.BuildId(role.ObjectMeta))
	metadata, metaDiags := common.FlattenNamespacedMetadata(ctx, role.ObjectMeta, nil, filters.GetIgnoreAnnotations(), filters.GetIgnoreLabels())
	resp.Diagnostics.Append(metaDiags...)
	state.Metadata = metadata

	rules, diags := flattenPolicyRules(role.Rules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Rule = rules

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	identity := common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{
			APIVersion: types.StringValue(rbacAPIVersion),
			Kind:       types.StringValue(roleKind),
			Name:       types.StringValue(role.Name),
		},
		Namespace: types.StringValue(role.Namespace),
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
}
