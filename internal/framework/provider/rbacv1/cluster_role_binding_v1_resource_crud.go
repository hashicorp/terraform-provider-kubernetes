// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
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
	if r.SDKv2Meta == nil {
		return nil, fmt.Errorf("provider meta is not configured")
	}
	meta := r.SDKv2Meta()
	if meta == nil {
		return nil, fmt.Errorf("provider meta is not configured")
	}
	clientsets, ok := meta.(kubernetes.KubeClientsets)
	if !ok {
		return nil, fmt.Errorf("provider meta does not implement kubernetes.KubeClientsets")
	}
	return clientsets, nil
}

func (r *ClusterRoleBinding) identity(name string) common.ResourceIdentity {
	return common.ResourceIdentity{
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

	metadata, diags := common.FlattenMetadata(ctx, out.ObjectMeta, state.Metadata, ignoreAnnotations, ignoreLabels)
	if diags.HasError() {
		return diags
	}

	state.Metadata = metadata
	state.RoleRef = flattenRoleRef(out.RoleRef)
	state.Subject = flattenSubjects(out.Subjects)

	return diags
}

// applyPlanResult preserves configured values and resolves server-computed fields.
func applyPlanResult(
	plan *ClusterRoleBindingModel,
	out *api.ClusterRoleBinding,
) {
	plan.ID = types.StringValue(out.Name)
	plan.Metadata[0].Name = types.StringValue(out.Name)
	plan.Metadata[0].Generation = types.Int64Value(out.Generation)
	plan.Metadata[0].ResourceVersion = types.StringValue(out.ResourceVersion)
	plan.Metadata[0].UID = types.StringValue(string(out.UID))
	for i := range plan.Subject {
		if plan.Subject[i].APIGroup.IsUnknown() {
			plan.Subject[i].APIGroup = types.StringValue(out.Subjects[i].APIGroup)
		}
	}
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

	metadata, d := common.ExpandMetadata(ctx, plan.Metadata)
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

	applyPlanResult(&plan, out)
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

	filters, ok := r.SDKv2Meta().(kubernetes.MetadataFilters)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider metadata", "Provider metadata does not implement kubernetes.MetadataFilters.")
		return
	}
	resp.Diagnostics.Append(applyState(ctx, &state, out, filters.GetIgnoreAnnotations(), filters.GetIgnoreLabels())...)
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
	var stateMetadata, planMetadata common.MetadataModel
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
	ops := common.MetadataPatchOps("/metadata/", stateMetadata, planMetadata)

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
		applyPlanResult(&plan, out)
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

	applyPlanResult(&plan, out)
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
		var identityData common.ResourceIdentity
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
	filters, ok := r.SDKv2Meta().(kubernetes.MetadataFilters)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider metadata", "Provider metadata does not implement kubernetes.MetadataFilters.")
		return
	}
	resp.Diagnostics.Append(applyState(ctx, &state, out, filters.GetIgnoreAnnotations(), filters.GetIgnoreLabels())...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, r.identity(out.Name))...)
}
