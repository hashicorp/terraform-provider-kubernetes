// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package functions

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"sigs.k8s.io/yaml"
)

func encodeValue(v attr.Value) (any, error) {
	if v.IsNull() {
		return nil, nil
	}

	switch vv := v.(type) {
	case basetypes.StringValue:
		return vv.ValueString(), nil
	case basetypes.NumberValue:
		f, _ := vv.ValueBigFloat().Float64()
		return f, nil
	case basetypes.Float64Value:
		return vv.ValueFloat64(), nil
	case basetypes.Int64Value:
		return vv.ValueInt64(), nil
	case basetypes.BoolValue:
		return vv.ValueBool(), nil
	case basetypes.ObjectValue:
		return encodeObject(vv)
	case basetypes.TupleValue:
		return encodeTuple(vv)
	case basetypes.MapValue:
		return encodeMap(vv)
	case basetypes.ListValue:
		return encodeList(vv)
	case basetypes.SetValue:
		return encodeSet(vv)
	default:
		return nil, fmt.Errorf("tried to encode unsupported type: %T: %v", v, vv)
	}
}

func encodeSet(sv basetypes.SetValue) ([]any, error) {
	elems := sv.Elements()
	size := len(elems)
	l := make([]any, size)
	for i := 0; i < size; i++ {
		var err error
		l[i], err = encodeValue(elems[i])
		if err != nil {
			return nil, err
		}
	}
	return l, nil
}

func encodeList(lv basetypes.ListValue) ([]any, error) {
	elems := lv.Elements()
	size := len(elems)
	l := make([]any, size)
	for i := 0; i < size; i++ {
		var err error
		l[i], err = encodeValue(elems[i])
		if err != nil {
			return nil, err
		}
	}
	return l, nil
}

func encodeTuple(tv basetypes.TupleValue) ([]any, error) {
	elems := tv.Elements()
	size := len(elems)
	l := make([]any, size)
	for i := 0; i < size; i++ {
		var err error
		l[i], err = encodeValue(elems[i])
		if err != nil {
			return nil, err
		}
	}
	return l, nil
}

func encodeObject(ov basetypes.ObjectValue) (map[string]any, error) {
	attrs := ov.Attributes()
	m := make(map[string]any, len(attrs))
	for k, v := range attrs {
		var err error
		m[k], err = encodeValue(v)
		if err != nil {
			return nil, err
		}
	}
	return m, nil
}

func encodeMap(mv basetypes.MapValue) (map[string]any, error) {
	elems := mv.Elements()
	m := make(map[string]any, len(elems))
	for k, v := range elems {
		var err error
		m[k], err = encodeValue(v)
		if err != nil {
			return nil, err
		}
	}
	return m, nil
}

func marshal(m map[string]any) (encoded string, diags diag.Diagnostics) {
	if err := validateKubernetesManifest(m); err != nil {
		diags.Append(diag.NewErrorDiagnostic("Invalid Kubernetes manifest", err.Error()))
		return
	}
	b, err := yaml.Marshal(m)
	if err != nil {
		diags.Append(diag.NewErrorDiagnostic("Error marshalling yaml", err.Error()))
		return
	}
	return string(b), nil
}

func encode(v attr.Value) (encoded string, diags diag.Diagnostics) {
	val, err := encodeValue(v)
	if err != nil {
		return "", diag.Diagnostics{diag.NewErrorDiagnostic("Error decoding manifest", err.Error())}
	}

	if m, ok := val.(map[string]any); ok {
		return marshal(m)
	} else if l, ok := val.([]any); ok {
		for _, vv := range l {
			m, ok := vv.(map[string]any)
			if !ok {
				diags.Append(diag.NewErrorDiagnostic(
					"List of manifests contained an invalid resource", fmt.Sprintf("value doesn't seem to be a manifest: %#v", vv)))
			}
			s, diags := marshal(m)
			if diags.HasError() {
				return "", diags
			}
			encoded = strings.Join([]string{encoded, s}, "---\n")
		}
		return string(encoded), nil
	}

	diags.Append(diag.NewErrorDiagnostic(
		"Invalid manifest", fmt.Sprintf("value doesn't seem to be a manifest: %#v", val)))
	return
}
