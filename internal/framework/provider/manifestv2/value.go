// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestv2

import (
	"context"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// toJSON converts a known Terraform value to the JSON form Kubernetes
// accepts. Null attributes are left out: in a manifest they mean "not set".
func toJSON(v attr.Value) (any, error) {
	if v == nil || v.IsNull() {
		return nil, nil
	}
	if v.IsUnknown() {
		return nil, fmt.Errorf("the value is not known yet")
	}
	switch v := v.(type) {
	case basetypes.DynamicValue:
		return toJSON(v.UnderlyingValue())
	case basetypes.StringValue:
		return v.ValueString(), nil
	case basetypes.BoolValue:
		return v.ValueBool(), nil
	case basetypes.NumberValue:
		f := v.ValueBigFloat()
		if i, accuracy := f.Int64(); accuracy == big.Exact {
			return i, nil
		}
		n, _ := f.Float64()
		return n, nil
	case basetypes.ObjectValue:
		return membersToJSON(v.Attributes())
	case basetypes.MapValue:
		return membersToJSON(v.Elements())
	case basetypes.TupleValue:
		return elementsToJSON(v.Elements())
	case basetypes.ListValue:
		return elementsToJSON(v.Elements())
	case basetypes.SetValue:
		return elementsToJSON(v.Elements())
	}
	return nil, fmt.Errorf("unsupported value type %T", v)
}

func membersToJSON(members map[string]attr.Value) (any, error) {
	out := make(map[string]any, len(members))
	for name, member := range members {
		if member.IsNull() {
			continue
		}
		value, err := toJSON(member)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		out[name] = value
	}
	return out, nil
}

func elementsToJSON(elements []attr.Value) (any, error) {
	out := make([]any, len(elements))
	for i, element := range elements {
		value, err := toJSON(element)
		if err != nil {
			return nil, fmt.Errorf("[%d]: %w", i, err)
		}
		out[i] = value
	}
	return out, nil
}

// fromJSON converts JSON read from the API to a Terraform value.
func fromJSON(ctx context.Context, v any) (attr.Value, error) {
	switch v := v.(type) {
	case nil:
		return types.DynamicNull(), nil
	case string:
		return types.StringValue(v), nil
	case bool:
		return types.BoolValue(v), nil
	case int64:
		return types.NumberValue(new(big.Float).SetInt64(v)), nil
	case float64:
		return types.NumberValue(big.NewFloat(v)), nil
	case map[string]any:
		attrTypes := make(map[string]attr.Type, len(v))
		attrs := make(map[string]attr.Value, len(v))
		for name, member := range v {
			// A null member, such as the creationTimestamp of a pod
			// template, says nothing and does not survive a plan unchanged.
			if member == nil {
				continue
			}
			value, err := fromJSON(ctx, member)
			if err != nil {
				return nil, err
			}
			attrTypes[name], attrs[name] = value.Type(ctx), value
		}
		object, diags := types.ObjectValue(attrTypes, attrs)
		if diags.HasError() {
			return nil, fmt.Errorf("%s", diags[0].Detail())
		}
		return object, nil
	case []any:
		elemTypes := make([]attr.Type, len(v))
		elems := make([]attr.Value, len(v))
		for i, element := range v {
			value, err := fromJSON(ctx, element)
			if err != nil {
				return nil, err
			}
			elemTypes[i], elems[i] = value.Type(ctx), value
		}
		tuple, diags := types.TupleValue(elemTypes, elems)
		if diags.HasError() {
			return nil, fmt.Errorf("%s", diags[0].Detail())
		}
		return tuple, nil
	}
	return nil, fmt.Errorf("unsupported JSON value %T", v)
}

// member returns a nested attribute of an object or map value, or nil.
func member(v attr.Value, names ...string) attr.Value {
	for _, name := range names {
		if dynamic, ok := v.(basetypes.DynamicValue); ok {
			v = dynamic.UnderlyingValue()
		}
		switch container := v.(type) {
		case basetypes.ObjectValue:
			v = container.Attributes()[name]
		case basetypes.MapValue:
			v = container.Elements()[name]
		default:
			return nil
		}
		if v == nil {
			return nil
		}
	}
	return v
}
