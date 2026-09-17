// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package schedulingv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
)

var (
	_ resource.Resource                     = (*PriorityClassV1)(nil) // compile-time check: PriorityClassV1 must implement all interface methods
	_ resource.ResourceWithConfigure        = (*PriorityClassV1)(nil)
	_ resource.ResourceWithImportState      = (*PriorityClassV1)(nil)
	_ resource.ResourceWithIdentity         = (*PriorityClassV1)(nil)
	_ resource.ResourceWithMoveState        = (*PriorityClassV1)(nil)
	_ resource.ResourceWithConfigValidators = (*PriorityClassV1)(nil)
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

func (r *PriorityClassV1) MoveState(_ context.Context) []resource.StateMover {
	return moveStateHandlers()
}

// ConfigValidators enforces that exactly one metadata block is present at plan
// time. ListNestedBlock validators fire at apply; this catches the missing block
// before Create/Update/Read are reached, matching SDKv2's Required:true behaviour.
func (r *PriorityClassV1) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		&metadataRequiredValidator{},
	}
}

// metadataRequiredValidator is a plan-time ConfigValidator that rejects configs
// with no metadata block (empty list) before any CRUD method is called.
type metadataRequiredValidator struct{}

func (v *metadataRequiredValidator) Description(_ context.Context) string {
	return "metadata block is required"
}

func (v *metadataRequiredValidator) MarkdownDescription(_ context.Context) string {
	return "`metadata` block is required"
}

func (v *metadataRequiredValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config PriorityClassModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(config.Metadata) == 0 {
		resp.Diagnostics.AddError(
			"Missing required block",
			"A metadata block is required for kubernetes_priority_class_v1. "+
				"Add a metadata { name = \"...\" } block to your configuration.",
		)
	}
}
