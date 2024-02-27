package functions

import (
	"context"
	"fmt"
	"math/big"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"

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
		MarkdownDescription: "Decode a Kubernetes manifest from YAML",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:                "manifest",
				MarkdownDescription: "The YAML plaintext for a Kubernetes manifest",
			},
		},
		Return: function.DynamicReturn{},
	}
}

var documentSeparator = regexp.MustCompile(`(:?^|\s*\n)---\s*`)

func validateKubernetesManifest(m map[string]any) error {
	// NOTE: a Kubernetes manifest should have:
	//       - an apiVersion
	//       - a kind
	//       - a metadata field
	for _, k := range []string{"apiVersion", "kind", "metadata"} {
		_, ok := m[k]
		if !ok {
			return fmt.Errorf("missing field %q", k)
		}
	}
	return nil
}

func (f ManifestDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var manifest string

	resp.Diagnostics.Append(req.Arguments.Get(ctx, &manifest)...)
	if resp.Diagnostics.HasError() {
		return
	}

	docs := documentSeparator.Split(manifest, -1)
	dtypes := []attr.Type{}
	dvalues := []attr.Value{}

	for _, d := range docs {
		var data map[string]any
		err := yaml.Unmarshal([]byte(d), &data)
		if err != nil {
			resp.Diagnostics.Append(diag.NewArgumentErrorDiagnostic(1, "Invalid YAML document", err.Error()))
			return
		}

		if len(data) == 0 {
			resp.Diagnostics.Append(diag.NewArgumentWarningDiagnostic(1, "Empty document", "encountered a YAML document with no values"))
			continue
		}

		if err := validateKubernetesManifest(data); err != nil {
			resp.Diagnostics.Append(diag.NewArgumentErrorDiagnostic(1, "Invalid Kubernetes manifest", err.Error()))
			return
		}

		obj, diags := decode(data)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		dtypes = append(dtypes, obj.Type(ctx))
		dvalues = append(dvalues, obj)
	}

	tv, diags := types.TupleValue(dtypes, dvalues)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var dynamResp types.Dynamic
	if len(dvalues) == 1 {
		dynamResp = types.DynamicValue(dvalues[0])
	} else {
		dynamResp = types.DynamicValue(tv)
	}
	resp.Diagnostics.Append(resp.Result.Set(ctx, &dynamResp)...)
}

func decodeMap(m map[string]any) (attr.Value, diag.Diagnostics) {
	vm := make(map[string]attr.Value, len(m))
	tm := make(map[string]attr.Type, len(m))

	for k, v := range m {
		vv, diags := decode(v)
		if diags.HasError() {
			return nil, diags
		}
		vm[k] = vv
		tm[k] = vv.Type(context.TODO())
	}

	return types.ObjectValue(tm, vm)
}

func decodeList(l []any) (attr.Value, diag.Diagnostics) {
	vl := make([]attr.Value, len(l))
	tl := make([]attr.Type, len(l))

	for i, v := range l {
		vv, diags := decode(v)
		if diags.HasError() {
			return nil, diags
		}
		vl[i] = vv
		tl[i] = vv.Type(context.TODO())
	}

	return types.TupleValue(tl, vl)
}

func decode(m any) (value attr.Value, diags diag.Diagnostics) {
	switch v := m.(type) {
	case float64:
		value = types.NumberValue(big.NewFloat(float64(v)))
	case bool:
		value = types.BoolValue(v)
	case string:
		value = types.StringValue(v)
	case []any:
		return decodeList(v)
	case map[string]any:
		return decodeMap(v)
	default:
		diags.Append(diag.NewErrorDiagnostic("failed to decode", fmt.Sprintf("unexpected type: %T for value %#v", v, v)))
	}
	return
}
