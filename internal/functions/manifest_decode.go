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

    dynamObj, objDiags := manifestToObjectValue(data)

	resp.Diagnostics.Append(objDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	dynamResp := types.DynamicValue(dynamObj)
	resp.Diagnostics.Append(resp.Result.Set(ctx, &dynamResp)...)
}

func manifestToObjectValue(manifest map[string]any) (basetypes.ObjectValue, diag.Diagnostics) {
	val := make(map[string]attr.Value)
	typ := make(map[string]attr.Type)

	for mk, mv := range manifest {
		switch value := mv.(type) {
		case bool, float64, string:
			typ[mk], val[mk] = manifestToBaseValue(value)
		case []any:
			v, _ := manifestToTupleValue(value)
			// FIXME handle error here
			typ[mk] = v.Type(context.TODO())
			val[mk] = v
		case map[string]any:
			v, _ := manifestToObjectValue(value)
			// FIXME handle error here
			typ[mk] = v.Type(context.TODO())
			val[mk] = v
		}
	}

	return types.ObjectValue(typ, val)
}

func manifestToTupleValue(manifest []any) (basetypes.TupleValue, diag.Diagnostics) {
	val := make([]attr.Value, len(manifest))
	typ := make([]attr.Type, len(manifest))

	for mi, mv := range manifest {
		switch value := mv.(type) {
		case bool, float64, string:
			typ[mi], val[mi] = manifestToBaseValue(value)
		case []any:
			v, _ := manifestToTupleValue(value)
			// FIXME handle error here
			typ[mi] = v.Type(context.TODO())
			val[mi] = v
		case map[string]any:
			v, _ := manifestToObjectValue(value)
			// FIXME handle error here
			typ[mi] = v.Type(context.TODO())
			val[mi] = v
		}
	}

	return types.TupleValue(typ, val)
}

func manifestToBaseValue(manifest any) (attr.Type, attr.Value) {
	var val attr.Value
	var typ attr.Type

	switch value := manifest.(type) {
	case float64:
		typ = types.NumberType
		val = types.NumberValue(big.NewFloat(float64(value)))
	case bool:
		typ = types.BoolType
		val = types.BoolValue(value)
	case string:
		typ = types.StringType
		val = types.StringValue(value)
	}

	return typ, val
}
