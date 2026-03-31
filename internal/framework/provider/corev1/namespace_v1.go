// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
)

var (
	_ resource.Resource                 = (*NamespaceV1)(nil)
	_ resource.ResourceWithConfigure    = (*NamespaceV1)(nil)
	_ resource.ResourceWithImportState  = (*NamespaceV1)(nil)
	_ resource.ResourceWithIdentity     = (*NamespaceV1)(nil)
	_ resource.ResourceWithMoveState    = (*NamespaceV1)(nil)
	_ resource.ResourceWithUpgradeState = (*NamespaceV1)(nil)
)

type NamespaceV1 struct {
	SDKv2Meta func() any
}

func NewNamespaceV1() resource.Resource {
	return &NamespaceV1{}
}

func (r *NamespaceV1) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_namespace_v1"
}

func (r *NamespaceV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	metaFunc, ok := req.ProviderData.(func() any)
	if !ok {
		resp.Diagnostics.AddError(
			"provider configuration error",
			"unexpected provider data type",
		)
		return
	}
	r.SDKv2Meta = metaFunc
}

func (r *NamespaceV1) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
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

func (r *NamespaceV1) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return upgradeStateHandlers()
}

func (r *NamespaceV1) MoveState(ctx context.Context) []resource.StateMover {
	return moveStateHandlers()
}
