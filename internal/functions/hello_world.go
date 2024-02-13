package functions

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

var _ function.Function = HelloWorldFunction{}

func NewHelloWorldFunction() function.Function {
	return &HelloWorldFunction{}
}

type HelloWorldFunction struct{}

func (f HelloWorldFunction) Metadata(_ context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "hello_world"
}

func (f HelloWorldFunction) Definition(_ context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "hello_world Function",
		MarkdownDescription: "hello_world Function",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:                "name",
				MarkdownDescription: "Name to send Hello to",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f HelloWorldFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
    var name string

    resp.Diagnostics.Append(req.Arguments.Get(ctx, &name)...)
    if resp.Diagnostics.HasError() {
        return
    }

    resp.Diagnostics.Append(resp.Result.Set(ctx, fmt.Sprintf("Hello, %s", name))...)
}
