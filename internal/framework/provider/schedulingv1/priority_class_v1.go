// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package schedulingv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
)

var (
	_ resource.Resource                 = (*PriorityClassV1)(nil) // compile-time check: PriorityClassV1 must implement all interface methods
	_ resource.ResourceWithConfigure    = (*PriorityClassV1)(nil)
	_ resource.ResourceWithImportState  = (*PriorityClassV1)(nil)
	_ resource.ResourceWithIdentity     = (*PriorityClassV1)(nil)
	_ resource.ResourceWithUpgradeState = (*PriorityClassV1)(nil)
)

type PriorityClassV1 struct {
	SDKv2Meta func() any
}

func NewPriorityClassV1() resource.Resource {
	return &PriorityClassV1{}
}

func (r *PriorityClassV1) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_priority_class_v1"
}

func (r *PriorityClassV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.SDKv2Meta = req.ProviderData.(func() any)
}

func (r *PriorityClassV1) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
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

func (r *PriorityClassV1) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	return upgradeStateHandlers()
}
