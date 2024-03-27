// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package functions

import (
	"context"
	"fmt"
	"math/big"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"sigs.k8s.io/yaml"
)

var documentSeparator = regexp.MustCompile(`(:?^|\s*\n)---\s*`)

func decode(ctx context.Context, manifest string) (v types.Tuple, diags diag.Diagnostics) {
	docs := documentSeparator.Split(manifest, -1)
	dtypes := []attr.Type{}
	dvalues := []attr.Value{}
	diags = diag.Diagnostics{}

	for _, d := range docs {
		var data map[string]any
		err := yaml.Unmarshal([]byte(d), &data)
		if err != nil {
			diags.Append(diag.NewErrorDiagnostic("Invalid YAML document", err.Error()))
			return
		}

		if len(data) == 0 {
			diags.Append(diag.NewWarningDiagnostic("Empty document", "encountered a YAML document with no values"))
			continue
		}

		if err := validateKubernetesManifest(data); err != nil {
			diags.Append(diag.NewErrorDiagnostic("Invalid Kubernetes manifest", err.Error()))
			return
		}

		obj, d := decodeScalar(ctx, data)
		diags.Append(d...)
		if diags.HasError() {
			return
		}
		dtypes = append(dtypes, obj.Type(ctx))
		dvalues = append(dvalues, obj)
	}

	return types.TupleValue(dtypes, dvalues)
}

func decodeMapping(ctx context.Context, m map[string]any) (attr.Value, diag.Diagnostics) {
	vm := make(map[string]attr.Value, len(m))
	tm := make(map[string]attr.Type, len(m))

	for k, v := range m {
		vv, diags := decodeScalar(ctx, v)
		if diags.HasError() {
			return nil, diags
		}
		vm[k] = vv
		tm[k] = vv.Type(ctx)
	}

	return types.ObjectValue(tm, vm)
}

func decodeSequence(ctx context.Context, s []any) (attr.Value, diag.Diagnostics) {
	vl := make([]attr.Value, len(s))
	tl := make([]attr.Type, len(s))

	for i, v := range s {
		vv, diags := decodeScalar(ctx, v)
		if diags.HasError() {
			return nil, diags
		}
		vl[i] = vv
		tl[i] = vv.Type(ctx)
	}

	return types.TupleValue(tl, vl)
}

func decodeScalar(ctx context.Context, m any) (value attr.Value, diags diag.Diagnostics) {
	switch v := m.(type) {
	case float64:
		value = types.NumberValue(big.NewFloat(float64(v)))
	case bool:
		value = types.BoolValue(v)
	case string:
		value = types.StringValue(v)
	case []any:
		return decodeSequence(ctx, v)
	case map[string]any:
		return decodeMapping(ctx, v)
	default:
		diags.Append(diag.NewErrorDiagnostic("failed to decode", fmt.Sprintf("unexpected type: %T for value %#v", v, v)))
	}
	return
}

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
