// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package storagev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
)

// StorageClassV1 is the Plugin Framework resource for kubernetes_storage_class_v1.
type StorageClassV1 struct {
	SDKv2Meta func() any
}

var (
	_ resource.Resource                = (*StorageClassV1)(nil)
	_ resource.ResourceWithConfigure   = (*StorageClassV1)(nil)
	_ resource.ResourceWithImportState = (*StorageClassV1)(nil)
	_ resource.ResourceWithIdentity    = (*StorageClassV1)(nil)
	_ resource.ResourceWithMoveState   = (*StorageClassV1)(nil)
	_ resource.ResourceWithModifyPlan  = (*StorageClassV1)(nil)
)

func NewStorageClassV1() resource.Resource {
	return &StorageClassV1{}
}

func (r *StorageClassV1) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storage_class_v1"
}

func (r *StorageClassV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.SDKv2Meta = req.ProviderData.(func() any)
}

func (r *StorageClassV1) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"api_version": identityschema.StringAttribute{
				RequiredForImport: true,
			},
			"kind": identityschema.StringAttribute{
				RequiredForImport: true,
			},
			"name": identityschema.StringAttribute{
				RequiredForImport: true,
			},
		},
	}
}

func (r *StorageClassV1) MoveState(_ context.Context) []resource.StateMover {
	return moveStateHandlers()
}

func (r *StorageClassV1) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// If creating or destroying, no plan modification needed
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var state, plan StorageClassModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If allowed_topologies block count or content changed, force replacement
	if len(state.AllowedTopologies) != len(plan.AllowedTopologies) {
		resp.RequiresReplace.Append(path.Root("allowed_topologies"))
	}
}
