// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"
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
	pkgApi "k8s.io/apimachinery/pkg/types"
)

const (
	clusterRoleAPIVersion = "rbac.authorization.k8s.io/v1"
	clusterRoleKind       = "ClusterRole"
)

func (r *ClusterRoleV1) conn() (kubernetes.KubeClientsets, error) {
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

func (r *ClusterRoleV1) identity(name string) common.ResourceIdentity {
	return common.ResourceIdentity{
		APIVersion: types.StringValue(clusterRoleAPIVersion),
		Kind:       types.StringValue(clusterRoleKind),
		Name:       types.StringValue(name),
	}
}

// applyClusterRolePlanResult builds state from the plan, overwriting only the fields the
// server assigns. Labels and annotations stay as planned: the API server and admission
// webhooks add keys of their own, and echoing those back would not match the plan, which
// Terraform rejects as an inconsistent result. Filtering them is Read's job.
func applyClusterRolePlanResult(plan *ClusterRoleModel, out *rbacv1api.ClusterRole) {
	plan.ID = types.StringValue(out.Name)
	if len(plan.Metadata) > 0 {
		common.PopulateMetadataFromResponse(&plan.Metadata[0], out.ObjectMeta)
	}
}

func (r *ClusterRoleV1) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ClusterRoleModel
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

	metadata, diags := common.ExpandMetadata(ctx, plan.Metadata)
	resp.Diagnostics.Append(diags...)
	rules, diags := expandClusterRoleRules(plan.Rule)
	resp.Diagnostics.Append(diags...)
	aggregation, diags := expandAggregationRule(ctx, plan.AggregationRule)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusterRole := &rbacv1api.ClusterRole{
		ObjectMeta:      metadata,
		Rules:           rules,
		AggregationRule: aggregation,
	}

	tflog.Info(ctx, "Creating new ClusterRole", map[string]any{"cluster_role": fmt.Sprintf("%#v", clusterRole)})
	out, err := conn.RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"error creating ClusterRole",
			fmt.Sprintf("Failed to create ClusterRole %q: %s", clusterRole.Name, err.Error()),
		)
		return
	}
	tflog.Info(ctx, "Submitted new ClusterRole", map[string]any{"cluster_role": fmt.Sprintf("%#v", out)})

	applyClusterRolePlanResult(&plan, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, r.identity(out.Name))...)
}

func (r *ClusterRoleV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ClusterRoleModel
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
	tflog.Info(ctx, "Reading ClusterRole", map[string]any{"name": name})
	out, err := conn.RbacV1().ClusterRoles().Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		// Terraform before 1.12 does not send identity on refresh, but the framework
		// still requires it when Read removes a missing resource.
		if resp.Identity.Raw.IsNull() {
			resp.Diagnostics.Append(resp.Identity.Set(ctx, r.identity(name))...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		tflog.Debug(ctx, "Received error", map[string]any{"error": fmt.Sprintf("%#v", err)})
		resp.Diagnostics.AddError(
			"error reading ClusterRole",
			fmt.Sprintf("Failed to read ClusterRole %q: %s", name, err.Error()),
		)
		return
	}
	tflog.Info(ctx, "Received ClusterRole", map[string]any{"cluster_role": fmt.Sprintf("%#v", out)})

	filters, ok := r.SDKv2Meta().(kubernetes.MetadataFilters)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider metadata", "Provider metadata does not implement kubernetes.MetadataFilters.")
		return
	}
	metadata, diags := common.FlattenMetadata(ctx, out.ObjectMeta, state.Metadata, filters.GetIgnoreAnnotations(), filters.GetIgnoreLabels())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.ID = types.StringValue(out.Name)
	state.Metadata = metadata

	if out.AggregationRule != nil {
		// With an aggregation rule the Kubernetes controller manager owns .rules and
		// rewrites them from the selected ClusterRoles, discarding anything configured.
		// A framework block cannot be Computed, so reading them back would plan their
		// removal on every run. Keep rule empty, which also drops the rules that SDKv2
		// recorded in state, and let aggregation_rule describe the object.
		aggregation, diags := flattenAggregationRule(out.AggregationRule, state.AggregationRule)
		resp.Diagnostics.Append(diags...)
		state.AggregationRule = aggregation
		state.Rule = []ClusterRoleRuleModel{}
	} else {
		rules, diags := flattenClusterRoleRules(out.Rules)
		resp.Diagnostics.Append(diags...)
		state.Rule = rules
		state.AggregationRule = []AggregationRuleModel{}
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, r.identity(out.Name))...)
}

func (r *ClusterRoleV1) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ClusterRoleModel
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

	name := state.ID.ValueString()

	var stateMetadata, planMetadata common.MetadataModel
	if len(state.Metadata) > 0 {
		stateMetadata = state.Metadata[0]
	}
	if len(plan.Metadata) > 0 {
		planMetadata = plan.Metadata[0]
	}

	// Build the patch from prior Terraform state to plan so that only Terraform-managed
	// keys are modified. Externally added and provider-ignored keys are absent from
	// state (filtered during Read) and so never appear in the diff.
	ops := common.MetadataPatchOps("/metadata/", stateMetadata, planMetadata)

	// SDKv2 patched rules index by index (remove, replace, add) and ended with the
	// planned list. Replacing /rules in one operation produces the same result.
	if !clusterRoleRulesEqual(state.Rule, plan.Rule) {
		rules, diags := expandClusterRoleRules(plan.Rule)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		ops = append(ops, &kubernetes.AddOperation{Path: "/rules", Value: rules})
	}

	if !aggregationRulesEqual(state.AggregationRule, plan.AggregationRule) {
		if len(plan.AggregationRule) == 0 {
			ops = append(ops, &kubernetes.RemoveOperation{Path: "/aggregationRule"})
		} else {
			aggregation, diags := expandAggregationRule(ctx, plan.AggregationRule)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			// Add replaces an existing member, and also works when the ClusterRole
			// had no aggregationRule yet, which Replace does not.
			ops = append(ops, &kubernetes.AddOperation{Path: "/aggregationRule", Value: aggregation})
		}
	}

	// Nothing to patch: fetch the object to refresh computed fields rather than send an
	// empty patch, which the API rejects.
	if len(ops) == 0 {
		out, err := conn.RbacV1().ClusterRoles().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			resp.Diagnostics.AddError(
				"error reading ClusterRole",
				fmt.Sprintf("Failed to read ClusterRole %q: %s", name, err.Error()),
			)
			return
		}
		applyClusterRolePlanResult(&plan, out)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.Append(resp.Identity.Set(ctx, r.identity(out.Name))...)
		return
	}

	patchBytes, err := ops.MarshalJSON()
	if err != nil {
		resp.Diagnostics.AddError("failed to marshal patch", err.Error())
		return
	}
	tflog.Info(ctx, "Updating ClusterRole", map[string]any{"name": name, "patch": string(patchBytes)})

	out, err := conn.RbacV1().ClusterRoles().Patch(ctx, name, pkgApi.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"error updating ClusterRole",
			fmt.Sprintf("Failed to update ClusterRole %q: %s", name, err.Error()),
		)
		return
	}
	tflog.Info(ctx, "Submitted updated ClusterRole", map[string]any{"cluster_role": fmt.Sprintf("%#v", out)})

	applyClusterRolePlanResult(&plan, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, r.identity(out.Name))...)
}

func (r *ClusterRoleV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ClusterRoleModel
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
	tflog.Info(ctx, "Deleting ClusterRole", map[string]any{"name": name})
	err = conn.RbacV1().ClusterRoles().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		resp.Diagnostics.AddError(
			"error deleting ClusterRole",
			fmt.Sprintf("Failed to delete ClusterRole %q: %s", name, err.Error()),
		)
		return
	}
	tflog.Info(ctx, "ClusterRole deleted", map[string]any{"name": name})
}

func (r *ClusterRoleV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	name := req.ID
	if name == "" {
		var identityData common.ResourceIdentity
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identityData)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if identityData.Kind.ValueString() != clusterRoleKind || identityData.APIVersion.ValueString() != clusterRoleAPIVersion {
			resp.Diagnostics.AddError("invalid identity import",
				fmt.Sprintf("Expected kind %q and api_version %q, got kind %q and api_version %q.",
					clusterRoleKind, clusterRoleAPIVersion, identityData.Kind.ValueString(), identityData.APIVersion.ValueString()))
			return
		}
		name = identityData.Name.ValueString()
		if name == "" {
			resp.Diagnostics.AddError("invalid identity import", "Identity attribute \"name\" must not be empty.")
			return
		}
	}

	// Read fetches the object and populates state and identity.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), name)...)
}
