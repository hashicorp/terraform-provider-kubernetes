// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package functions

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = ManifestEncodeFunction{}

func NewManifestEncodeFunction() function.Function {
	return &ManifestEncodeFunction{}
}

type ManifestEncodeFunction struct{}

func (f ManifestEncodeFunction) Metadata(_ context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "manifest_encode"
}

func (f ManifestEncodeFunction) Definition(_ context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Encode an object to Kubernetes YAML",
		MarkdownDescription: "Given an object representation of a Kubernetes manifest, will encode and return a YAML string for that resource.",
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:                "manifest",
				MarkdownDescription: "The object representation of a Kubernetes manifest",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f ManifestEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var manifest types.Dynamic

	resp.Error = req.Arguments.Get(ctx, &manifest)
	if resp.Error != nil {
		return
	}

	uv := manifest.UnderlyingValue()
	s, diags := encode(uv)
	if diags.HasError() {
		resp.Error = function.FuncErrorFromDiags(ctx, diags)
		return
	}

	svalue := types.StringValue(s)
	resp.Error = resp.Result.Set(ctx, &svalue)
}
