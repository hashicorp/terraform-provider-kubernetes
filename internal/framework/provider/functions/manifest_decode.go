// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package functions

import (
	"context"

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
		Summary:             "Decode a Kubernetes YAML manifest",
		MarkdownDescription: "Given a YAML text containing a Kubernetes manifest, will decode and return an object representation of that resource.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:                "manifest",
				MarkdownDescription: "The YAML text for a Kubernetes manifest",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f ManifestDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
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

	elems := tv.Elements()
	if len(elems) == 0 {
		resp.Error = function.NewFuncError("Invalid manifest: YAML document is empty")
		return
	} else if len(elems) > 1 {
		resp.Error = function.NewFuncError("YAML manifest contains multiple resources: use decode_manifest_multi to decode manifests containing more than one resource")
		return
	}

	dynamResp := types.DynamicValue(elems[0])
	resp.Error = resp.Result.Set(ctx, &dynamResp)
}
