// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

var (
	_ resource.Resource                    = (*DeploymentV1)(nil)
	_ resource.ResourceWithConfigure       = (*DeploymentV1)(nil)
	_ resource.ResourceWithIdentity        = (*DeploymentV1)(nil)
	_ resource.ResourceWithImportState     = (*DeploymentV1)(nil)
	_ resource.ResourceWithModifyPlan      = (*DeploymentV1)(nil)
	_ resource.ResourceWithMoveState       = (*DeploymentV1)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*DeploymentV1)(nil)
	_ resource.ResourceWithUpgradeState    = (*DeploymentV1)(nil)
)

const (
	deploymentAPIVersion = "apps/v1"
	deploymentKind       = "Deployment"

	sdkv2ProviderAddressSuffix = "/hashicorp/kubernetes"
	moveStateErrSummary        = "Unable to move kubernetes_deployment state"
)

type DeploymentV1 struct {
	SDKv2Meta func() any
}

type DeploymentV1Model struct {
	ID             types.String                     `tfsdk:"id"`
	Metadata       []common.NamespacedMetadataModel `tfsdk:"metadata"`
	Spec           types.List                       `tfsdk:"spec"`
	WaitForRollout types.Bool                       `tfsdk:"wait_for_rollout"`
	Timeouts       timeouts.Value                   `tfsdk:"timeouts"`
}

func NewDeploymentV1() resource.Resource {
	return &DeploymentV1{}
}

func (d *DeploymentV1) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment_v1"
}

func (d *DeploymentV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	sdkv2Meta, ok := req.ProviderData.(func() any)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data",
			fmt.Sprintf("Expected func() any, got %T. This is a bug in the provider.", req.ProviderData),
		)
		return
	}
	d.SDKv2Meta = sdkv2Meta
}

func (d *DeploymentV1) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() || req.Config.Raw.IsNull() {
		return
	}
	plan, usePriorState, err := workloadNoOpPlan(req.Config.Raw, req.Plan.Raw, req.State.Raw)
	if err != nil {
		resp.Diagnostics.AddError("Unable to normalize deployment plan", err.Error())
		return
	}
	if usePriorState {
		resp.Plan.Raw = plan
	}
}

func (d *DeploymentV1) sdkv2Meta() (kubernetes.KubeClientsets, kubernetes.MetadataFilters, diag.Diagnostics) {
	var diags diag.Diagnostics

	if d.SDKv2Meta == nil {
		diags.AddError("Provider not configured", "The SDKv2 provider metadata is unavailable. This is a bug in the provider.")
		return nil, nil, diags
	}

	meta := d.SDKv2Meta()
	clients, ok := meta.(kubernetes.KubeClientsets)
	if !ok {
		diags.AddError("Unexpected provider data",
			fmt.Sprintf("Expected kubernetes.KubeClientsets, got %T. This is a bug in the provider.", meta))
		return nil, nil, diags
	}
	filters, ok := meta.(kubernetes.MetadataFilters)
	if !ok {
		diags.AddError("Unexpected provider data",
			fmt.Sprintf("Expected kubernetes.MetadataFilters, got %T. This is a bug in the provider.", meta))
		return nil, nil, diags
	}

	return clients, filters, diags
}

func (d *DeploymentV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != "" {
		namespace, name, err := kubernetes.IdParts(req.ID)
		if err != nil || namespace == "" || name == "" {
			resp.Diagnostics.AddError("Invalid deployment import ID", "Expected a non-empty namespace/name ID.")
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
		return
	}

	if req.Identity == nil {
		resp.Diagnostics.AddError("Missing deployment import identity", "Provide a namespace/name ID or an identity containing namespace and name.")
		return
	}
	var identity common.NamespacedResourceIdentity
	resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
	if resp.Diagnostics.HasError() {
		return
	}

	namespace := identity.Namespace.ValueString()
	if identity.Namespace.IsNull() || identity.Namespace.IsUnknown() || namespace == "" ||
		identity.Name.IsNull() || identity.Name.IsUnknown() || identity.Name.ValueString() == "" {
		resp.Diagnostics.AddError("Invalid deployment import identity", "Import identity must contain known, non-empty namespace and name values.")
		return
	}
	id := namespace + "/" + identity.Name.ValueString()
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
	// SDKv2 identity imports leave this local policy at its zero value.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("wait_for_rollout"), false)...)
}

func (d *DeploymentV1) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = common.NamespacedIdentitySchema()
}

func (d *DeploymentV1) UpgradeIdentity(ctx context.Context) map[int64]resource.IdentityUpgrader {
	return common.UpgradeNamespacedIdentity(deploymentKind, deploymentAPIVersion)
}

func isSDKv2SourceType(req resource.MoveStateRequest, expectedType string) bool {
	return req.SourceTypeName == expectedType &&
		(req.SourceSchemaVersion == 0 || req.SourceSchemaVersion == 1) &&
		strings.HasSuffix(req.SourceProviderAddress, sdkv2ProviderAddressSuffix)
}
