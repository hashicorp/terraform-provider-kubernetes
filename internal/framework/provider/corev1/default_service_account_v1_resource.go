// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

var (
	_ resource.Resource                    = (*DefaultServiceAccountV1)(nil)
	_ resource.ResourceWithConfigure       = (*DefaultServiceAccountV1)(nil)
	_ resource.ResourceWithIdentity        = (*DefaultServiceAccountV1)(nil)
	_ resource.ResourceWithImportState     = (*DefaultServiceAccountV1)(nil)
	_ resource.ResourceWithMoveState       = (*DefaultServiceAccountV1)(nil)
	_ resource.ResourceWithModifyPlan      = (*DefaultServiceAccountV1)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*DefaultServiceAccountV1)(nil)
)

type DefaultServiceAccountV1 struct {
	SDKv2Meta func() any
}

func NewDefaultServiceAccountV1() resource.Resource {
	return &DefaultServiceAccountV1{}
}

func (r *DefaultServiceAccountV1) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_default_service_account_v1"
}

func (r *DefaultServiceAccountV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	serviceAccountConfigure(req, resp, &r.SDKv2Meta)
}

func (r *DefaultServiceAccountV1) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = serviceAccountSchema(ctx, true)
}

func (r *DefaultServiceAccountV1) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = common.NamespacedIdentitySchema()
}

func (r *DefaultServiceAccountV1) UpgradeIdentity(context.Context) map[int64]resource.IdentityUpgrader {
	return common.UpgradeNamespacedIdentity(serviceAccountKind, serviceAccountAPIVersion)
}

func (r *DefaultServiceAccountV1) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	serviceAccountCreate(ctx, req, resp, r.SDKv2Meta, true)
}

func (r *DefaultServiceAccountV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	serviceAccountRead(ctx, req, resp, r.SDKv2Meta, true)
}

func (r *DefaultServiceAccountV1) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	serviceAccountUpdate(ctx, req, resp, r.SDKv2Meta, true)
}

// The SDKv2 default resource inherits ordinary deletion, not state-only detachment.
func (r *DefaultServiceAccountV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	serviceAccountDelete(ctx, req, resp, r.SDKv2Meta, true)
}

func (r *DefaultServiceAccountV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	serviceAccountImport(ctx, req, resp, r.SDKv2Meta, true)
}

func (r *DefaultServiceAccountV1) MoveState(ctx context.Context) []resource.StateMover {
	return serviceAccountMoveState(ctx, true)
}

func (r *DefaultServiceAccountV1) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	serviceAccountModifyPlan(ctx, req, resp)
}
