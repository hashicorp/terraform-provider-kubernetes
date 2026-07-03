// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubemanifest

import (
	"context"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// AttrToGo converts a Framework attr.Value (typically the underlying value of a
// types.Dynamic patch) into a Go value suitable for JSON-encoding a Server-Side Apply
// body. Explicit nulls are PRESERVED as nil (which json.Marshal renders as `null`, the
// SSA signal to remove an owned field); absent keys are simply not present. Unknown
// values are an error — a patch must be fully known at apply time.
func AttrToGo(ctx context.Context, v attr.Value) (interface{}, error) {
	if v == nil || v.IsNull() {
		return nil, nil
	}
	if v.IsUnknown() {
		return nil, fmt.Errorf("patch contains an unknown value; all patch values must be known at apply time")
	}
	switch val := v.(type) {
	case basetypes.DynamicValue:
		return AttrToGo(ctx, val.UnderlyingValue())
	case basetypes.ObjectValue:
		return attrMap(ctx, val.Attributes())
	case basetypes.MapValue:
		return attrMap(ctx, val.Elements())
	case basetypes.ListValue:
		return attrSlice(ctx, val.Elements())
	case basetypes.SetValue:
		return attrSlice(ctx, val.Elements())
	case basetypes.TupleValue:
		return attrSlice(ctx, val.Elements())
	case basetypes.StringValue:
		return val.ValueString(), nil
	case basetypes.BoolValue:
		return val.ValueBool(), nil
	case basetypes.NumberValue:
		return numberToGo(val.ValueBigFloat()), nil
	case basetypes.Int64Value:
		return val.ValueInt64(), nil
	case basetypes.Float64Value:
		return val.ValueFloat64(), nil
	default:
		return nil, fmt.Errorf("unsupported patch value type %T", v)
	}
}

func attrMap(ctx context.Context, in map[string]attr.Value) (map[string]interface{}, error) {
	out := make(map[string]interface{}, len(in))
	for k, av := range in {
		g, err := AttrToGo(ctx, av)
		if err != nil {
			return nil, err
		}
		out[k] = g // preserve nil → JSON null (SSA field removal)
	}
	return out, nil
}

func attrSlice(ctx context.Context, in []attr.Value) ([]interface{}, error) {
	out := make([]interface{}, 0, len(in))
	for _, av := range in {
		g, err := AttrToGo(ctx, av)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, nil
}

// numberToGo renders a big.Float as an int64 when it is integral (so replicas=3
// serializes as 3, not 3.0), else as a float64.
func numberToGo(f *big.Float) interface{} {
	if f == nil {
		return nil
	}
	if i, acc := f.Int64(); acc == big.Exact {
		return i
	}
	v, _ := f.Float64()
	return v
}

// DeepMerge recursively merges src into dst: nested maps are merged; for any other
// value (including explicit nil) src wins. Returns dst. Used to overlay a patch onto
// the object's identity fields (apiVersion/kind/metadata) without clobbering them.
func DeepMerge(dst, src map[string]interface{}) map[string]interface{} {
	if dst == nil {
		dst = map[string]interface{}{}
	}
	for k, sv := range src {
		if sm, ok := sv.(map[string]interface{}); ok {
			if dm, ok := dst[k].(map[string]interface{}); ok {
				dst[k] = DeepMerge(dm, sm)
				continue
			}
		}
		dst[k] = sv
	}
	return dst
}

// PruneNulls recursively removes map entries whose value is nil, and recurses into
// nested maps and maps nested in slices. This is how "set a leaf to null to remove it"
// works for Server-Side Apply: a field the manager owned but no longer declares is
// pruned by the API server. (Sending an explicit JSON null does NOT remove typed
// string-map entries such as annotations/labels/ConfigMap data — the API coerces null
// to ""; omitting the field is the portable removal signal.)
func PruneNulls(v interface{}) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		for k, val := range t {
			if val == nil {
				delete(t, k)
				continue
			}
			t[k] = PruneNulls(val)
		}
		return t
	case []interface{}:
		for i, e := range t {
			t[i] = PruneNulls(e)
		}
		return t
	default:
		return v
	}
}
