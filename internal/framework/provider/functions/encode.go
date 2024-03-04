package functions

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"sigs.k8s.io/yaml"
)

func encodeValue(v attr.Value) (any, error) {
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
		// FIXME: we should support map, list here too
	default:
		return nil, fmt.Errorf("tried to encode unsupported type: %T: %v", v, vv)
	}
}

func encodeTuple(t basetypes.TupleValue) ([]any, error) {
	size := len(t.Elements())
	l := make([]any, size)
	for i := 0; i < size; i++ {
		var err error
		l[i], err = encodeValue(t.Elements()[i])
		if err != nil {
			return nil, err
		}
	}
	return l, nil
}

func encodeObject(o basetypes.ObjectValue) (map[string]any, error) {
	m := map[string]any{}
	for k, v := range o.Attributes() {
		var err error
		m[k], err = encodeValue(v)
		if err != nil {
			return nil, err
		}
	}
	return m, nil
}

func encode(v attr.Value) (string, diag.Diagnostics) {
	val, err := encodeValue(v)
	if err != nil {
		return "", diag.Diagnostics{diag.NewErrorDiagnostic("Error decoding manifest", err.Error())}
	}
	encoded := []byte{}
	if l, ok := val.([]any); ok {
		for _, vv := range l {
			e, err := yaml.Marshal(vv)
			if err != nil {
				return "", diag.Diagnostics{diag.NewErrorDiagnostic("Error marshalling yaml", err.Error())}
			}
			encoded = append(encoded, []byte("---\n")...)
			encoded = append(encoded, e...)
		}
		return string(encoded), nil
	}

	encoded, err = yaml.Marshal(val)
	if err != nil {
		return "", diag.Diagnostics{diag.NewErrorDiagnostic("Error marshalling yaml", err.Error())}
	}
	return string(encoded), nil
}
