package functions

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = ManifestDecodeFunction{}

func NewManifestDecodeFunction() function.Function {
	return &ManifestDecodeFunction{}
}

type ManifestDecodeFunction struct{}

func (f ManifestDecodeFunction) Metadata(_ context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "manifest_decode"
}

func (f ManifestDecodeFunction) Definition(_ context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "manifest_decode Function",
		MarkdownDescription: "Decode a single Kubernetes manifest from a YAML document",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:                "manifest",
				MarkdownDescription: "The YAML plaintext for a Kubernetes manifest",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f ManifestDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
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

	elems := tv.Elements()
	if len(elems) == 0 {
		resp.Diagnostics.Append(diag.NewArgumentErrorDiagnostic(1, "Invalid manifest", "YAML document is empty"))
		return
	} else if len(elems) > 1 {
		resp.Diagnostics.Append(diag.NewArgumentWarningDiagnostic(1, "YAML manifest contains multiple resources, only the first resource will be used", "Use decode_manifest_multi to decode manifests containing more than one resource"))
	}

	dynamResp := types.DynamicValue(elems[0])
	resp.Diagnostics.Append(resp.Result.Set(ctx, &dynamResp)...)
}
