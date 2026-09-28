// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	api "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

const (
	clusterRoleBindingAPIVersion = "rbac.authorization.k8s.io/v1"
	clusterRoleBindingKind       = "ClusterRoleBinding"
)

func (r *ClusterRoleBinding) conn() (kubernetes.KubeClientsets, error) {
	meta := r.SDKv2Meta()
	if meta == nil {
		return nil, fmt.Errorf("provider meta is not configured")
	}
	return meta.(kubernetes.KubeClientsets), nil
}

func (r *ClusterRoleBinding) identity(name string) ClusterRoleBindingIdentityModel {
	return ClusterRoleBindingIdentityModel{
		APIVersion: types.StringValue(clusterRoleBindingAPIVersion),
		Kind:       types.StringValue(clusterRoleBindingKind),
		Name:       types.StringValue(name),
	}
}

// applyState fully rebuilds a model from a fetched ClusterRoleBinding object,
// filtering internal and provider-ignored metadata keys that are not already
// tracked in state. It is used by Read and ImportState.
func applyState(
	ctx context.Context,
	state *ClusterRoleBindingModel,
	out *api.ClusterRoleBinding,
	ignoreAnnotations []string,
	ignoreLabels []string,
) diag.Diagnostics {
	state.ID = types.StringValue(out.Name)

	var currentMeta ClusterRoleBindingMetadataModel
	if len(state.Metadata) > 0 {
		currentMeta = state.Metadata[0]
	}

	metadata, diags := flattenMetadata(ctx, out.ObjectMeta, currentMeta, ignoreAnnotations, ignoreLabels)
	if diags.HasError() {
		return diags
	}

	state.Metadata = metadata
	state.RoleRef = flattenRoleRef(out.RoleRef)
	state.Subject = flattenSubjects(out.Subjects)

	return nil
}

// applyPlanResult updates a plan-derived model after a Create/Update using the
// server response, filtering metadata using the planned state as the "current"
// reference so that provider-ignored keys do not cause plan drift. Subjects are
// taken from the response because their api_group and namespace fields are
// computed. The server-assigned name is always echoed back (required for
// generate_name).
func applyPlanResult(
	ctx context.Context,
	plan *ClusterRoleBindingModel,
	out *api.ClusterRoleBinding,
	ignoreAnnotations []string,
	ignoreLabels []string,
) diag.Diagnostics {
	plan.ID = types.StringValue(out.Name)

	var currentMeta ClusterRoleBindingMetadataModel
	if len(plan.Metadata) > 0 {
		currentMeta = plan.Metadata[0]
	}

	metadata, diags := flattenMetadata(ctx, out.ObjectMeta, currentMeta, ignoreAnnotations, ignoreLabels)
	if diags.HasError() {
		return diags
	}

	plan.Metadata = metadata
	// name is Optional+Computed: echo the server value to handle generate_name.
	plan.Metadata[0].Name = types.StringValue(out.Name)
	plan.Subject = flattenSubjects(out.Subjects)

	return nil
}

func (r *ClusterRoleBinding) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ClusterRoleBindingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clientset, err := r.conn()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}
	conn, err := clientset.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	metadata, d := expandMetadata(ctx, plan.Metadata)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	binding := &api.ClusterRoleBinding{
		ObjectMeta: metadata,
		RoleRef:    expandRoleRef(plan.RoleRef),
		Subjects:   expandSubjects(plan.Subject),
	}

	out, err := conn.RbacV1().ClusterRoleBindings().Create(ctx, binding, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"error creating ClusterRoleBinding",
			fmt.Sprintf("Failed to create ClusterRoleBinding %q: %s", binding.Name, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(applyPlanResult(ctx, &plan, out, clientset.GetIgnoreAnnotations(), clientset.GetIgnoreLabels())...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, r.identity(out.Name))...)
}

func (r *ClusterRoleBinding) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ClusterRoleBindingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clientset, err := r.conn()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}
	conn, err := clientset.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	name := state.ID.ValueString()
	out, err := conn.RbacV1().ClusterRoleBindings().Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"error reading ClusterRoleBinding",
			fmt.Sprintf("Failed to read ClusterRoleBinding %q: %s", name, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(applyState(ctx, &state, out, clientset.GetIgnoreAnnotations(), clientset.GetIgnoreLabels())...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, r.identity(out.Name))...)
}

func (r *ClusterRoleBinding) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ClusterRoleBindingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clientset, err := r.conn()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}
	conn, err := clientset.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	name := plan.ID.ValueString()

	// Guard metadata access — schema requires one block but avoids a panic on
	// incomplete state.
	var stateMetadata, planMetadata ClusterRoleBindingMetadataModel
	if len(state.Metadata) > 0 {
		stateMetadata = state.Metadata[0]
	}
	if len(plan.Metadata) > 0 {
		planMetadata = plan.Metadata[0]
	}

	// Build the patch from prior Terraform state → plan so that only
	// Terraform-managed keys are modified. Externally added and
	// provider-ignored keys are absent from Terraform state (filtered during
	// Read) and therefore never appear in the diff.
	ops, d := buildMetadataPatch(ctx, stateMetadata, planMetadata)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	// role_ref is immutable (RequiresReplace) — only subjects can change here.
	// Replace /subjects atomically with one operation to avoid per-index
	// ordering and stale-entry bugs.
	if !subjectsEqual(state.Subject, plan.Subject) {
		ops = append(ops, &kubernetes.ReplaceOperation{
			Path:  "/subjects",
			Value: expandSubjects(plan.Subject),
		})
	}

	// No-op update: nothing changed — fetch current state to refresh computed
	// fields without sending an empty patch (which the API rejects).
	if len(ops) == 0 {
		out, err := conn.RbacV1().ClusterRoleBindings().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			resp.Diagnostics.AddError(
				"error reading ClusterRoleBinding",
				fmt.Sprintf("Failed to read ClusterRoleBinding %q: %s", name, err.Error()),
			)
			return
		}
		resp.Diagnostics.Append(applyPlanResult(ctx, &plan, out, clientset.GetIgnoreAnnotations(), clientset.GetIgnoreLabels())...)
		if resp.Diagnostics.HasError() {
			return
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.Append(resp.Identity.Set(ctx, r.identity(out.Name))...)
		return
	}

	patchBytes, err := ops.MarshalJSON()
	if err != nil {
		resp.Diagnostics.AddError("failed to marshal patch", err.Error())
		return
	}

	out, err := conn.RbacV1().ClusterRoleBindings().Patch(
		ctx,
		name,
		pkgApi.JSONPatchType,
		patchBytes,
		metav1.PatchOptions{},
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"error updating ClusterRoleBinding",
			fmt.Sprintf("Failed to patch ClusterRoleBinding %q: %s", name, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(applyPlanResult(ctx, &plan, out, clientset.GetIgnoreAnnotations(), clientset.GetIgnoreLabels())...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, r.identity(out.Name))...)
}

func (r *ClusterRoleBinding) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ClusterRoleBindingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clientset, err := r.conn()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}
	conn, err := clientset.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	name := state.ID.ValueString()
	err = conn.RbacV1().ClusterRoleBindings().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		resp.Diagnostics.AddError(
			"error deleting ClusterRoleBinding",
			fmt.Sprintf("Failed to delete ClusterRoleBinding %q: %s", name, err.Error()),
		)
		return
	}
}

func (r *ClusterRoleBinding) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var name string
	if req.ID != "" {
		name = req.ID
	} else {
		var identityData ClusterRoleBindingIdentityModel
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identityData)...)
		if resp.Diagnostics.HasError() {
			return
		}
		name = identityData.Name.ValueString()
	}

	clientset, err := r.conn()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}
	conn, err := clientset.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	out, err := conn.RbacV1().ClusterRoleBindings().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"error importing ClusterRoleBinding",
			fmt.Sprintf("Failed to import ClusterRoleBinding %q: %s", name, err.Error()),
		)
		return
	}

	var state ClusterRoleBindingModel
	resp.Diagnostics.Append(applyState(ctx, &state, out, clientset.GetIgnoreAnnotations(), clientset.GetIgnoreLabels())...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, r.identity(out.Name))...)
}
