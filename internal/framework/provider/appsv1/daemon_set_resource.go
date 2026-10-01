// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

const (
	daemonSetAPIVersion = "apps/v1"
	daemonSetKind       = "DaemonSet"

	defaultDaemonSetCreateTimeout = 10 * time.Minute
	defaultDaemonSetUpdateTimeout = 10 * time.Minute
	defaultDaemonSetDeleteTimeout = 10 * time.Minute
)

var (
	_ resource.Resource                    = (*DaemonSetV1)(nil)
	_ resource.ResourceWithConfigure       = (*DaemonSetV1)(nil)
	_ resource.ResourceWithImportState     = (*DaemonSetV1)(nil)
	_ resource.ResourceWithIdentity        = (*DaemonSetV1)(nil)
	_ resource.ResourceWithModifyPlan      = (*DaemonSetV1)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*DaemonSetV1)(nil)
	_ resource.ResourceWithMoveState       = (*DaemonSetV1)(nil)
	_ resource.ResourceWithUpgradeState    = (*DaemonSetV1)(nil)
)

type DaemonSetV1 struct {
	SDKv2Meta func() any
}

type DaemonSetV1Model struct {
	ID             types.String                     `tfsdk:"id"`
	Metadata       []common.NamespacedMetadataModel `tfsdk:"metadata"`
	Spec           []DaemonSetV1SpecModel           `tfsdk:"spec"`
	WaitForRollout types.Bool                       `tfsdk:"wait_for_rollout"`
	Timeouts       timeouts.Value                   `tfsdk:"timeouts"`
}

type DaemonSetV1SpecModel struct {
	MinReadySeconds      types.Int64                   `tfsdk:"min_ready_seconds"`
	RevisionHistoryLimit types.Int64                   `tfsdk:"revision_history_limit"`
	Selector             []DaemonSetLabelSelectorModel `tfsdk:"selector"`
	Strategy             types.List                    `tfsdk:"strategy"`
	Template             []DaemonSetTemplateModel      `tfsdk:"template"`
}

type DaemonSetTemplateModel struct {
	Metadata []common.NamespacedMetadataModel `tfsdk:"metadata"`
	Spec     types.List                       `tfsdk:"spec"`
}

type DaemonSetLabelSelectorModel = LabelSelectorModel

type DaemonSetMatchExpressionModel = LabelSelectorRequirementModel

type DaemonSetStrategyModel struct {
	Type          types.String `tfsdk:"type"`
	RollingUpdate types.List   `tfsdk:"rolling_update"`
}

type DaemonSetRollingUpdateModel struct {
	MaxSurge       types.String `tfsdk:"max_surge"`
	MaxUnavailable types.String `tfsdk:"max_unavailable"`
}

func NewDaemonSetV1() resource.Resource {
	return &DaemonSetV1{}
}

func (d *DaemonSetV1) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_daemon_set_v1"
}

func (d *DaemonSetV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (d *DaemonSetV1) ModifyPlan(_ context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() || req.Config.Raw.IsNull() {
		return
	}

	plan, usePriorState, err := workloadNoOpPlan(req.Config.Raw, req.Plan.Raw, req.State.Raw)
	if err != nil {
		resp.Diagnostics.AddError("Unable to normalize daemonset plan", err.Error())
		return
	}
	if usePriorState {
		resp.Plan.Raw = plan
	}
}

func (d *DaemonSetV1) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = common.NamespacedIdentitySchema()
}

func (d *DaemonSetV1) UpgradeIdentity(context.Context) map[int64]resource.IdentityUpgrader {
	return common.UpgradeNamespacedIdentity(daemonSetKind, daemonSetAPIVersion)
}

func (d *DaemonSetV1) sdkv2Meta() (kubernetes.KubeClientsets, kubernetes.MetadataFilters, diag.Diagnostics) {
	var diagnostics diag.Diagnostics

	if d.SDKv2Meta == nil {
		diagnostics.AddError("Provider not configured",
			"The SDKv2 provider metadata is unavailable. This is a bug in the provider.")
		return nil, nil, diagnostics
	}

	meta := d.SDKv2Meta()
	clients, ok := meta.(kubernetes.KubeClientsets)
	if !ok {
		diagnostics.AddError("Unexpected provider data",
			fmt.Sprintf("Expected kubernetes.KubeClientsets, got %T. This is a bug in the provider.", meta))
		return nil, nil, diagnostics
	}

	filters, ok := meta.(kubernetes.MetadataFilters)
	if !ok {
		diagnostics.AddError("Unexpected provider data",
			fmt.Sprintf("Expected kubernetes.MetadataFilters, got %T. This is a bug in the provider.", meta))
		return nil, nil, diagnostics
	}

	return clients, filters, diagnostics
}

func daemonSetIdentity(namespace, name string) common.NamespacedResourceIdentity {
	return common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{
			APIVersion: types.StringValue(daemonSetAPIVersion),
			Kind:       types.StringValue(daemonSetKind),
			Name:       types.StringValue(name),
		},
		Namespace: types.StringValue(namespace),
	}
}

func daemonSetIdentitySchema() identityschema.Schema {
	return common.NamespacedIdentitySchema()
}
