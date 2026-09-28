// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package storagev1

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// StorageClassV1 is the Plugin Framework resource for kubernetes_storage_class_v1.
type StorageClassV1 struct {
	SDKv2Meta func() any
}

var (
	_ resource.Resource                    = (*StorageClassV1)(nil)
	_ resource.ResourceWithConfigure       = (*StorageClassV1)(nil)
	_ resource.ResourceWithImportState     = (*StorageClassV1)(nil)
	_ resource.ResourceWithIdentity        = (*StorageClassV1)(nil)
	_ resource.ResourceWithMoveState       = (*StorageClassV1)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*StorageClassV1)(nil)
	_ resource.ResourceWithModifyPlan      = (*StorageClassV1)(nil)
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
		Version: 1,
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

// UpgradeIdentity handles version-0 identity from provider versions before 2.38.0
// (e.g. v2.37.1) where identity_schema_version was 0 or identity was absent.
func (r *StorageClassV1) UpgradeIdentity(ctx context.Context) map[int64]resource.IdentityUpgrader {
	return map[int64]resource.IdentityUpgrader{
		0: {
			IdentityUpgrader: func(ctx context.Context, req resource.UpgradeIdentityRequest, resp *resource.UpgradeIdentityResponse) {
				if resp.Identity == nil {
					return
				}

				identity := StorageClassIdentityModel{
					Name:       types.StringNull(),
					Kind:       types.StringNull(),
					APIVersion: types.StringNull(),
				}

				if req.RawIdentity != nil && len(req.RawIdentity.JSON) > 0 {
					var prior struct {
						Name string `json:"name"`
					}
					if err := json.Unmarshal(req.RawIdentity.JSON, &prior); err != nil {
						resp.Diagnostics.AddError(
							"Unable to upgrade resource identity",
							fmt.Sprintf("Could not decode the stored identity: %s", err),
						)
						return
					}
					identity.Name = types.StringValue(prior.Name)
					identity.Kind = types.StringValue("StorageClass")
					identity.APIVersion = types.StringValue("storage.k8s.io/v1")
				}

				resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
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
