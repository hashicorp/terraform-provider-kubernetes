package functions

import (
	"context"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"sigs.k8s.io/yaml"
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
		MarkdownDescription: "manifest_decode Function",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:                "manifest",
				MarkdownDescription: "Manifest to decode",
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

	var data map[string]any
	err := yaml.Unmarshal([]byte(manifest), &data)
	if err != nil {
		// FIXME: handle this error
	}

    dynamObj, objDiags := manifestToValue(data)

	resp.Diagnostics.Append(objDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	dynamResp := types.DynamicValue(dynamObj)
	resp.Diagnostics.Append(resp.Result.Set(ctx, &dynamResp)...)
}

func manifestToValue(manifest map[string]any) (basetypes.ObjectValue, diag.Diagnostics) {
	v := make(map[string]attr.Value)
	t := make(map[string]attr.Type)
	for k, vv := range manifest {
		switch value := vv.(type) {
		case float64:
			t[k] = types.NumberType
			v[k] = types.NumberValue(big.NewFloat(float64(value)))
		case bool:
			t[k] = types.BoolType
			v[k] = types.BoolValue(value)
		case string:
			t[k] = types.StringType
			v[k] = types.StringValue(value)
		case []any:
			tv, _ := manifestToValueList(value)
			// FIXME handle error here
			t[k] = tv.Type(context.TODO())
			v[k] = tv
		case map[string]any:
			ov, _ := manifestToValue(value)
			// FIXME handle error here
			t[k] = ov.Type(context.TODO())
			v[k] = ov
		}
	}

	return types.ObjectValue(t, v)
}

func manifestToValueList(manifest []any) (basetypes.TupleValue, diag.Diagnostics) {
	v := make([]attr.Value, len(manifest))
	t := make([]attr.Type, len(manifest))
	for i, vv := range manifest {
		switch value := vv.(type) {
		case float64:
			t[i] = types.NumberType
			v[i] = types.NumberValue(big.NewFloat(float64(value)))
		case bool:
			t[i] = types.BoolType
			v[i] = types.BoolValue(value)
		case string:
			t[i] = types.StringType
			v[i] = types.StringValue(value)
		case []any:
			tv, _ := manifestToValueList(value)
			// FIXME handle error here
			t[i] = tv.Type(context.TODO())
			v[i] = tv
		case map[string]any:
			ov, _ := manifestToValue(value)
			// FIXME handle error here
			t[i] = ov.Type(context.TODO())
			v[i] = ov
		}
	}

	return types.TupleValue(t, v)
}
