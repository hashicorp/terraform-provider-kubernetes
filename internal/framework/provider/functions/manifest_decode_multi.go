// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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

	resp.Error = req.Arguments.Get(ctx, &manifest)
	if resp.Error != nil {
		return
	}

	tv, diags := decode(ctx, manifest)
	if diags.HasError() {
		resp.Error = function.FuncErrorFromDiags(ctx, diags)
		return
	}

	dynamResp := types.DynamicValue(tv)
	resp.Error = resp.Result.Set(ctx, &dynamResp)
}
