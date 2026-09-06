// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	api "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// applyState fully rebuilds a model from a fetched ClusterRoleBinding object.
// It is used by Read and Import where there is no prior plan to preserve.
func applyState(state *ClusterRoleBindingModel, out *api.ClusterRoleBinding) {
	state.ID = types.StringValue(out.Name)
	state.Metadata = flattenMetadata(out.ObjectMeta)
	state.RoleRef = flattenRoleRef(out.RoleRef)
	state.Subject = flattenSubjects(out.Subjects)
}

// applyPlanResult updates a plan-derived model after a Create/Update with the
// server response. It preserves the configured (non-computed) metadata and
// role_ref values to avoid "inconsistent result after apply" errors, filling in
// only the computed fields. Subjects are taken from the response because their
// api_group and namespace fields are computed.
func applyPlanResult(plan *ClusterRoleBindingModel, out *api.ClusterRoleBinding) {
	plan.ID = types.StringValue(out.Name)

	m := &plan.Metadata[0]
	// name is Optional+Computed: this echoes the configured name, or the
	// server-generated name when generate_name is used.
	m.Name = types.StringValue(out.Name)
	m.Generation = types.Int64Value(out.Generation)
	m.ResourceVersion = types.StringValue(out.ResourceVersion)
	m.UID = types.StringValue(string(out.UID))

	plan.Subject = flattenSubjects(out.Subjects)
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

	applyState(&state, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, r.identity(out.Name))...)
}

func (r *ClusterRoleBinding) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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

	name := plan.ID.ValueString()
	cur, err := conn.RbacV1().ClusterRoleBindings().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"read before update failed",
			fmt.Sprintf("Failed to read ClusterRoleBinding %q before update: %s", name, err.Error()),
		)
		return
	}

	// role_ref is immutable (ForceNew / RequiresReplace) so it is never updated
	// here. Only metadata labels/annotations and subjects can change.
	labels, d := expandStringMap(ctx, plan.Metadata[0].Labels)
	resp.Diagnostics.Append(d...)
	annotations, d := expandStringMap(ctx, plan.Metadata[0].Annotations)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	cur.ObjectMeta.Labels = labels
	cur.ObjectMeta.Annotations = annotations
	cur.Subjects = expandSubjects(plan.Subject)

	out, err := conn.RbacV1().ClusterRoleBindings().Update(ctx, cur, metav1.UpdateOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"error updating ClusterRoleBinding",
			fmt.Sprintf("Failed to update ClusterRoleBinding %q: %s", name, err.Error()),
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
	applyState(&state, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, r.identity(out.Name))...)
}
