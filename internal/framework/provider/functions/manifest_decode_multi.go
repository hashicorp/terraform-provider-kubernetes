package functions

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = ManifestDecodeMultiFunction{}

func NewManifestDecodeMultiFunction() function.Function {
	return &ManifestDecodeMultiFunction{}
}

type ManifestDecodeMultiFunction struct{}

func (f ManifestDecodeMultiFunction) Metadata(_ context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "manifest_decode_multi"
}

func (f ManifestDecodeMultiFunction) Definition(_ context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Decode a Kubernetes YAML manifest containing multiple resources",
		MarkdownDescription: "Given a YAML text containing a Kubernetes manifest with multiple resources, will decode the manifest and return a tuple of object representations for each resource.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:                "manifest",
				MarkdownDescription: "The YAML plaintext for a Kubernetes manifest",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f ManifestDecodeMultiFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var manifest string

	resp.Diagnostics.Append(req.Arguments.Get(ctx, &manifest)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tv, diags := decode(manifest)
	resp.Diagnostics = append(resp.Diagnostics, diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	dynamResp := types.DynamicValue(tv)
	resp.Diagnostics.Append(resp.Result.Set(ctx, &dynamResp)...)
}
