// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

const (
	statefulSetAPIVersion = "apps/v1"
	statefulSetKind       = "StatefulSet"
)

var (
	_ resource.Resource                    = (*StatefulSetV1)(nil)
	_ resource.ResourceWithConfigure       = (*StatefulSetV1)(nil)
	_ resource.ResourceWithIdentity        = (*StatefulSetV1)(nil)
	_ resource.ResourceWithImportState     = (*StatefulSetV1)(nil)
	_ resource.ResourceWithMoveState       = (*StatefulSetV1)(nil)
	_ resource.ResourceWithModifyPlan      = (*StatefulSetV1)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*StatefulSetV1)(nil)
	_ resource.ResourceWithUpgradeState    = (*StatefulSetV1)(nil)
)

type StatefulSetV1 struct {
	SDKv2Meta func() any
}

func NewStatefulSetV1() resource.Resource {
	return &StatefulSetV1{}
}

func (r *StatefulSetV1) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_stateful_set_v1"
}

func (r *StatefulSetV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	meta, ok := req.ProviderData.(func() any)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected func() any, got %T", req.ProviderData))
		return
	}
	r.SDKv2Meta = meta
}

func (r *StatefulSetV1) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}
	var plan StatefulSetV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state StatefulSetV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || len(plan.Spec) != 1 || len(state.Spec) != 1 {
		return
	}
	replicas := plan.Spec[0].Replicas
	if replicas.IsNull() || replicas.IsUnknown() || replicas.ValueString() != "" {
		return
	}
	prior := state.Spec[0].Replicas
	if prior.IsNull() || prior.IsUnknown() {
		return
	}
	plan.Spec[0].Replicas = prior
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r *StatefulSetV1) sdkv2Meta() (kubernetes.KubeClientsets, kubernetes.MetadataFilters, diag.Diagnostics) {
	var diags diag.Diagnostics
	if r.SDKv2Meta == nil {
		diags.AddError("Provider not configured", "The SDKv2 provider metadata is unavailable. This is a bug in the provider.")
		return nil, nil, diags
	}

	meta := r.SDKv2Meta()
	clients, ok := meta.(kubernetes.KubeClientsets)
	if !ok {
		diags.AddError("Unexpected provider data", fmt.Sprintf("Expected kubernetes.KubeClientsets, got %T", meta))
		return nil, nil, diags
	}
	filters, ok := meta.(kubernetes.MetadataFilters)
	if !ok {
		diags.AddError("Unexpected provider data", fmt.Sprintf("Expected kubernetes.MetadataFilters, got %T", meta))
		return nil, nil, diags
	}
	return clients, filters, diags
}

func (r *StatefulSetV1) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = common.NamespacedIdentitySchema()
}

func (r *StatefulSetV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != "" {
		resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
		return
	}

	var ident statefulSetIdentityModel
	resp.Diagnostics.Append(req.Identity.Get(ctx, &ident)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ns := ident.Namespace.ValueString()
	name := ident.Name.ValueString()
	if name == "" {
		resp.Diagnostics.AddError("Unable to import StatefulSet", "Missing identity name")
		return
	}
	if ns == "" {
		ns = "default"
	}
	id := fmt.Sprintf("%s/%s", ns, name)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func (r *StatefulSetV1) UpgradeIdentity(ctx context.Context) map[int64]resource.IdentityUpgrader {
	return common.UpgradeNamespacedIdentity(statefulSetKind, statefulSetAPIVersion)
}

func hasProviderSuffix(address string) bool {
	return strings.HasSuffix(address, "/hashicorp/kubernetes")
}
