// Copyright IBM Corp. 2017, 2026

package rbacv1

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
)

var (
	_ resource.Resource                   = (*Role)(nil)
	_ resource.ResourceWithConfigure      = (*Role)(nil)
	_ resource.ResourceWithImportState    = (*Role)(nil)
	_ resource.ResourceWithIdentity       = (*Role)(nil)
	_ resource.ResourceWithValidateConfig = (*Role)(nil)
)

type Role struct {
	SDKv2Meta func() any
}

func NewRole() resource.Resource {
	return &Role{}
}

func (r *Role) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_v1"
}

func (r *Role) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.SDKv2Meta = req.ProviderData.(func() any)
}

func (r *Role) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config RoleModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(config.Metadata) == 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("metadata"),
			"Missing required block",
			fmt.Sprintf("At least one %q block is required.", "metadata"),
		)
	}
	if len(config.Rule) == 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("rule"),
			"Missing required block",
			fmt.Sprintf("At least one %q block is required.", "rule"),
		)
	}
}

func (r *Role) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		// Version must match resourceIdentitySchemaNamespaced() in the SDKv2
		// implementation so that identity state written before this resource
		// migrated to Framework is still considered current.
		Version: 1,
		Attributes: map[string]identityschema.Attribute{
			"api_version": identityschema.StringAttribute{
				RequiredForImport: true,
			},
			"kind": identityschema.StringAttribute{
				RequiredForImport: true,
			},
			"namespace": identityschema.StringAttribute{
				OptionalForImport: true,
			},
			"name": identityschema.StringAttribute{
				RequiredForImport: true,
			},
		},
	}
}
