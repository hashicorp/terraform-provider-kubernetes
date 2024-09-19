package authenticationv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
)

var _ ephemeral.EphemeralResource = (*TokenRequestEphemeralResource)(nil)

type TokenRequestEphemeralResource struct{}

type TokenRequestModel struct {
	Token types.String `tfsdk:"token"`
}

func NewTokenRequestEphemeralResource() ephemeral.EphemeralResource {
	return &TokenRequestEphemeralResource{}
}

func (r *TokenRequestEphemeralResource) Metadata(ctx context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_token_request_v1"
}

func (r *TokenRequestEphemeralResource) Schema(ctx context.Context, req ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "TokenRequest requests a token for a given service account.",
		Attributes: map[string]schema.Attribute{
			"token": schema.StringAttribute{
				Description: "Token is the opaque bearer token.",
			},
		},
	}
}

func (r *TokenRequestEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var data TokenRequestModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
