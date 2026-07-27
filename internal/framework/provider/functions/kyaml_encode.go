package functions

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"sigs.k8s.io/yaml"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

var _ function.Function = KyamlEncodeFunction{}

func NewKyamlEncodeFunction() function.Function {
	return &KyamlEncodeFunction{}
}

type KyamlEncodeFunction struct{}

func (f KyamlEncodeFunction) Metadata(_ context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "kyaml_encode"
}

func (f KyamlEncodeFunction) Definition(_ context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Encode an object to Kubernetes YAML (KYAML)",
		MarkdownDescription: "Given an object representation of a Kubernetes manifest, will encode and return a KYAML string for that resource. KYAML is a Kubernetes dialect of YAML.",
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:                "manifest",
				MarkdownDescription: "The object representation of a Kubernetes manifest",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f KyamlEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var manifest types.Dynamic

	resp.Error = req.Arguments.Get(ctx, &manifest)
	if resp.Error != nil {
		return
	}

	uv := manifest.UnderlyingValue()
	val, err := encodeValue(uv)
	if err != nil {
		resp.Error = function.FuncErrorFromDiags(ctx, diag.Diagnostics{diag.NewErrorDiagnostic("Error decoding manifest", err.Error())})
		return
	}

	var encoded string
	if m, ok := val.(map[string]any); ok {
		s, diags := marshalKyaml(m)
		if diags.HasError() {
			resp.Error = function.FuncErrorFromDiags(ctx, diags)
			return
		}
		encoded = s
	} else if l, ok := val.([]any); ok {
		var parts []string
		for _, vv := range l {
			m, ok := vv.(map[string]any)
			if !ok {
				resp.Error = function.FuncErrorFromDiags(ctx, diag.Diagnostics{diag.NewErrorDiagnostic(
					"List of manifests contained an invalid resource", fmt.Sprintf("value doesn't seem to be a manifest: %#v", vv))})
				return
			}
			s, diags := marshalKyaml(m)
			if diags.HasError() {
				resp.Error = function.FuncErrorFromDiags(ctx, diags)
				return
			}
			parts = append(parts, s)
		}
		encoded = strings.Join(parts, "---\n")
	} else {
		resp.Error = function.FuncErrorFromDiags(ctx, diag.Diagnostics{diag.NewErrorDiagnostic(
			"Invalid manifest", fmt.Sprintf("value doesn't seem to be a manifest: %#v", val))})
		return
	}

	svalue := types.StringValue(encoded)
	resp.Error = resp.Result.Set(ctx, &svalue)
}

func marshalKyaml(in map[string]any) (string, diag.Diagnostics) {
	b, err := yaml.Marshal(in)
	if err != nil {
		return "", diag.Diagnostics{diag.NewErrorDiagnostic("Error marshalling yaml", err.Error())}
	}
	return string(b), nil
}
