// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package functions

import (
	"context"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = EnvFromMapFunction{}

func NewEnvFromMapFunction() function.Function {
	return &EnvFromMapFunction{}
}

type EnvFromMapFunction struct{}

// envObjectType is the element type returned by env_from_map: an object with
// name and value attributes, matching the shape of a Kubernetes container
// environment variable.
var envObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"name":  types.StringType,
		"value": types.StringType,
	},
}

func (f EnvFromMapFunction) Metadata(_ context.Context, req function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "env_from_map"
}

func (f EnvFromMapFunction) Definition(_ context.Context, req function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Convert a map of strings into a list of name/value objects",
		MarkdownDescription: "Given a map of strings, returns a list of objects with `name` and `value` attributes, sorted by key. This is useful for populating the `env` field of a container in a `kubernetes_manifest` resource without repeating the `name`/`value` boilerplate for every variable.",
		Parameters: []function.Parameter{
			function.MapParameter{
				Name:                "env",
				ElementType:         types.StringType,
				MarkdownDescription: "A map of environment variable names to values",
			},
		},
		Return: function.ListReturn{
			ElementType: envObjectType,
		},
	}
}

func (f EnvFromMapFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var env map[string]string

	resp.Error = req.Arguments.Get(ctx, &env)
	if resp.Error != nil {
		return
	}

	// Sort the keys so the resulting list is deterministic and does not
	// produce a perpetual diff when the input map is reordered.
	names := make([]string, 0, len(env))
	for name := range env {
		names = append(names, name)
	}
	sort.Strings(names)

	elements := make([]attr.Value, 0, len(env))
	for _, name := range names {
		obj, diags := types.ObjectValue(envObjectType.AttrTypes, map[string]attr.Value{
			"name":  types.StringValue(name),
			"value": types.StringValue(env[name]),
		})
		if diags.HasError() {
			resp.Error = function.FuncErrorFromDiags(ctx, diags)
			return
		}
		elements = append(elements, obj)
	}

	result, diags := types.ListValue(envObjectType, elements)
	if diags.HasError() {
		resp.Error = function.FuncErrorFromDiags(ctx, diags)
		return
	}

	resp.Error = resp.Result.Set(ctx, &result)
}
