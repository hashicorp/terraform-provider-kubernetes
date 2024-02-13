package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func GetFunctionsSchema() map[string]*tfprotov5.Function {
	return map[string]*tfprotov5.Function{
		"manifest_decode": {
			Parameters: []*tfprotov5.FunctionParameter{
				{
					Name:        "manifest",
					Type:        tftypes.String,
					Description: "A YAML encoded Kubernetes manifest",
				},
			},
			Return: &tfprotov5.FunctionReturn{
				Type: tftypes.DynamicPseudoType,
			},
			Summary:     "decode a Kubernetes manifest",
			Description: "Take a YAML encoded Kubernetes manifest and decodes it into a Terraform object",
		},
	}
}

func (s *RawProviderServer) GetFunctions(ctx context.Context, req *tfprotov5.GetFunctionsRequest) (*tfprotov5.GetFunctionsResponse, error) {
	resp := &tfprotov5.GetFunctionsResponse{
		Functions: GetFunctionsSchema(),
	}
	return resp, nil
}

func (s *RawProviderServer) CallFunction(ctx context.Context, req *tfprotov5.CallFunctionRequest) (*tfprotov5.CallFunctionResponse, error) {
	v, _ := tfprotov5.NewDynamicValue(tftypes.DynamicPseudoType, tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"test": tftypes.String,
		},
	}, map[string]tftypes.Value{
		"test": tftypes.NewValue(tftypes.String, "hello"),
	}))
	resp := &tfprotov5.CallFunctionResponse{
		Result: &v,
	}
	return resp, nil
}
