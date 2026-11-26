// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package autoscalingv2

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

var (
	_ resource.Resource                    = (*HorizontalPodAutoscalerV2)(nil)
	_ resource.ResourceWithConfigure       = (*HorizontalPodAutoscalerV2)(nil)
	_ resource.ResourceWithImportState     = (*HorizontalPodAutoscalerV2)(nil)
	_ resource.ResourceWithIdentity        = (*HorizontalPodAutoscalerV2)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*HorizontalPodAutoscalerV2)(nil)
	_ resource.ResourceWithUpgradeState    = (*HorizontalPodAutoscalerV2)(nil)
	_ resource.ResourceWithMoveState       = (*HorizontalPodAutoscalerV2)(nil)
	_ resource.ResourceWithModifyPlan      = (*HorizontalPodAutoscalerV2)(nil)
)

const (
	hpaKind       = "HorizontalPodAutoscaler"
	hpaAPIVersion = "autoscaling/v2"
)

type HorizontalPodAutoscalerV2 struct {
	SDKv2Meta func() any
}

func NewHorizontalPodAutoscalerV2() resource.Resource {
	return &HorizontalPodAutoscalerV2{}
}

func (r *HorizontalPodAutoscalerV2) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_horizontal_pod_autoscaler_v2"
}

func (r *HorizontalPodAutoscalerV2) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	sdkv2Meta, ok := req.ProviderData.(func() any)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data",
			fmt.Sprintf("Expected func() any, got %T. This is a bug in the provider.", req.ProviderData))
		return
	}
	r.SDKv2Meta = sdkv2Meta
}

// ModifyPlan keeps the prior state when the plan differs from it only in unknown computed values.
// Terraform proposes null for an omitted metric or behavior whose stored value has non-computed
// attributes, so the framework marks every computed attribute unknown even though the attribute
// plan modifiers then restore the stored value.
func (r *HorizontalPodAutoscalerV2) ModifyPlan(_ context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() || req.Config.Raw.IsNull() {
		return
	}
	plan, usePriorState, err := common.NoOpPlan(req.Config.Raw, req.Plan.Raw, req.State.Raw)
	if err != nil {
		resp.Diagnostics.AddError("Unable to normalize HorizontalPodAutoscaler plan", err.Error())
		return
	}
	if usePriorState {
		resp.Plan.Raw = plan
	}
}

func (r *HorizontalPodAutoscalerV2) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = common.NamespacedIdentitySchema()
}

func (r *HorizontalPodAutoscalerV2) UpgradeIdentity(_ context.Context) map[int64]resource.IdentityUpgrader {
	return common.UpgradeNamespacedIdentity(hpaKind, hpaAPIVersion)
}

func hpaIdentity(namespace, name string) common.NamespacedResourceIdentity {
	return common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{
			Name:       types.StringValue(name),
			Kind:       types.StringValue(hpaKind),
			APIVersion: types.StringValue(hpaAPIVersion),
		},
		Namespace: types.StringValue(namespace),
	}
}
