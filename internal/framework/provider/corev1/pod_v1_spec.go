// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	api "k8s.io/api/core/v1"
)

// podSpecMeta holds the schema-derived values that the Pod spec expander and
// flattener need on every RPC. They depend only on the static schema, so they
// are computed once per process and shared read-only.
type podSpecMeta struct {
	objectType basetypes.ObjectTypable
	computed   map[string]bool
	blocks     map[string]bool
}

var podSpecDerived = sync.OnceValue(func() podSpecMeta {
	object := common.FreezeNestedBlockObject(podSpecObject())
	meta := podSpecMeta{objectType: object.Type(), computed: map[string]bool{}, blocks: map[string]bool{}}
	podSpecComputedPaths(object.Attributes, object.Blocks, "spec", meta.computed)
	podSpecBlockPaths(object, "spec", meta.blocks)
	return meta
})

func expandPodV1Spec(ctx context.Context, spec types.List) (*api.PodSpec, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if spec.IsNull() || spec.IsUnknown() || len(spec.Elements()) != 1 {
		diagnostics.AddAttributeError(path.Root("spec"), "Invalid Pod Specification", "A known spec block is required before creating or updating a Pod.")
		return nil, diagnostics
	}
	raw := podSpecAPIValue(ctx, spec, path.Root("spec"), "spec", podSpecDerived().computed, &diagnostics)
	if diagnostics.HasError() {
		return nil, diagnostics
	}
	result, err := kubernetes.ExpandPodSpecForFramework(raw.([]interface{}))
	if err != nil {
		diagnostics.AddError("Unable to Expand Pod Specification", err.Error())
	}
	return result, diagnostics
}

func flattenPodV1Spec(ctx context.Context, spec api.PodSpec, prior types.List) (types.List, diag.Diagnostics) {
	derived := podSpecDerived()
	objectType := derived.objectType
	var diagnostics diag.Diagnostics
	raw, err := kubernetes.FlattenPodSpecForFramework(spec)
	if err != nil {
		diagnostics.AddError("Unable to Flatten Pod Specification", err.Error())
		return types.ListNull(objectType), diagnostics
	}
	value := podSpecStateValue(ctx, types.ListType{ElemType: objectType}, raw, prior, []string{"spec"}, "spec", derived.blocks, &diagnostics)
	if diagnostics.HasError() {
		return types.ListNull(objectType), diagnostics
	}
	return value.(types.List), diagnostics
}

// This boundary translates known Framework values to the input of the shared
// pure Pod API helpers, never to ResourceData. Unknowns cannot reach client-go.
func podSpecAPIValue(ctx context.Context, value attr.Value, at path.Path, key string, computed map[string]bool, diagnostics *diag.Diagnostics) interface{} {
	if value.IsUnknown() && !computed[key] {
		diagnostics.AddAttributeError(at, "Unknown Pod Specification", "This value must be known before sending the Pod specification to Kubernetes.")
		return nil
	}
	switch v := value.(type) {
	case types.String:
		return v.ValueString()
	case types.Bool:
		return v.ValueBool()
	case types.Int64:
		return int(v.ValueInt64())
	case types.Object:
		if v.IsNull() {
			diagnostics.AddAttributeError(at, "Invalid Pod Specification", "A nested Pod specification object cannot be null.")
			return nil
		}
		result := make(map[string]interface{}, len(v.Attributes()))
		for name, entry := range v.Attributes() {
			result[name] = podSpecAPIValue(ctx, entry, at.AtName(name), key+"."+name, computed, diagnostics)
		}
		return result
	case types.Map:
		result := make(map[string]interface{}, len(v.Elements()))
		for name, entry := range v.Elements() {
			result[name] = podSpecAPIValue(ctx, entry, at.AtMapKey(name), key, computed, diagnostics)
		}
		return result
	case types.List:
		result := make([]interface{}, len(v.Elements()))
		for index, entry := range v.Elements() {
			result[index] = podSpecAPIValue(ctx, entry, at.AtListIndex(index), key, computed, diagnostics)
		}
		return result
	case types.Set:
		result := make([]interface{}, len(v.Elements()))
		for index, entry := range v.Elements() {
			result[index] = podSpecAPIValue(ctx, entry, at.AtListIndex(index), key, computed, diagnostics)
		}

		return result
	default:
		diagnostics.AddAttributeError(at, "Unable to Expand Pod Specification", fmt.Sprintf("Unsupported Pod value type %T.", value))
		return nil
	}
}

func podSpecComputedPaths(attributes map[string]schema.Attribute, blocks map[string]schema.Block, prefix string, computed map[string]bool) {
	for name, attribute := range attributes {
		key := prefix + "." + name
		if attribute.IsComputed() {
			computed[key] = true
		}
		if nested, ok := attribute.(schema.ListNestedAttribute); ok {
			podSpecComputedPaths(nested.NestedObject.Attributes, nil, key, computed)
		}
	}
	for name, block := range blocks {
		if nested, ok := block.(schema.ListNestedBlock); ok {
			podSpecComputedPaths(nested.NestedObject.Attributes, nested.NestedObject.Blocks, prefix+"."+name, computed)
		}
	}
}

func podSpecBlockPaths(object schema.NestedBlockObject, prefix string, blocks map[string]bool) {
	for name, block := range object.Blocks {
		key := prefix + "." + name
		blocks[key] = true
		if nested, ok := block.(schema.ListNestedBlock); ok {
			podSpecBlockPaths(nested.NestedObject, key, blocks)
		}
	}
}

// The existing pure flatteners return maps, slices, enum aliases, and pointers.
// Read those values through the native Pod type so all 756 historical paths have
// their declared Terraform type without erasing null/empty collection ownership.
func podSpecStateValue(ctx context.Context, typ attr.Type, raw interface{}, prior attr.Value, names []string, key string, blocks map[string]bool, diagnostics *diag.Diagnostics) attr.Value {
	rv := reflect.ValueOf(raw)
	for rv.IsValid() && (rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface) {
		if rv.IsNil() {
			rv = reflect.Value{}
			break
		}
		rv = rv.Elem()
	}
	var result attr.Value
	switch t := typ.(type) {
	case types.ObjectType:
		entries := map[string]attr.Value{}
		var old map[string]attr.Value
		if object, ok := prior.(types.Object); ok && !object.IsNull() && !object.IsUnknown() {
			old = object.Attributes()
		}
		for name, childType := range t.AttrTypes {
			var child interface{}
			if rv.IsValid() && rv.Kind() == reflect.Map {
				if entry := rv.MapIndex(reflect.ValueOf(name)); entry.IsValid() {
					child = entry.Interface()
				}
			}
			childNames := append(append([]string(nil), names...), name)
			entries[name] = podSpecStateValue(ctx, childType, child, old[name], childNames, key+"."+name, blocks, diagnostics)
		}
		v, d := types.ObjectValue(t.AttrTypes, entries)
		diagnostics.Append(d...)
		result = v
	case types.ListType:
		var previous []attr.Value
		if list, ok := prior.(types.List); ok && !list.IsNull() && !list.IsUnknown() {
			previous = list.Elements()
		}
		count := 0
		if rv.IsValid() && (rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array) {
			count = rv.Len()
		}
		if count == 0 && !blocks[key] && prior != nil && prior.IsNull() {
			return types.ListNull(t.ElemType)
		}
		if count == 0 && !rv.IsValid() && !blocks[key] && prior == nil {
			return types.ListNull(t.ElemType)
		}
		entries := make([]attr.Value, count)
		for i := 0; i < count; i++ {
			var old attr.Value
			if i < len(previous) {
				old = previous[i]
			}
			entries[i] = podSpecStateValue(ctx, t.ElemType, rv.Index(i).Interface(), old, names, key, blocks, diagnostics)
		}
		v, d := types.ListValue(t.ElemType, entries)
		diagnostics.Append(d...)
		result = v
	case types.SetType:
		if (!rv.IsValid() || rv.Len() == 0) && (prior == nil || prior.IsNull()) {
			return types.SetNull(t.ElemType)
		}
		var entries []attr.Value
		if rv.IsValid() {
			for i := 0; i < rv.Len(); i++ {
				entries = append(entries, podSpecStateValue(ctx, t.ElemType, rv.Index(i).Interface(), nil, names, key, blocks, diagnostics))
			}
		}
		v, d := types.SetValue(t.ElemType, entries)
		diagnostics.Append(d...)
		result = v
	case types.MapType:
		if (!rv.IsValid() || rv.Len() == 0) && prior != nil && prior.IsNull() {
			return types.MapNull(t.ElemType)
		}
		if !rv.IsValid() && prior == nil {
			return types.MapNull(t.ElemType)
		}
		entries := map[string]attr.Value{}
		if rv.IsValid() {
			iter := rv.MapRange()
			for iter.Next() {
				name := iter.Key().String()
				entries[name] = podSpecStateValue(ctx, t.ElemType, iter.Value().Interface(), nil, nil, key, blocks, diagnostics)
			}
		}
		v, d := types.MapValue(t.ElemType, entries)
		diagnostics.Append(d...)
		result = v
	default:
		switch {
		case typ.Equal(types.StringType):
			text := ""
			if rv.IsValid() {
				text = fmt.Sprint(rv.Interface())
			}
			result = types.StringValue(text)
		case typ.Equal(types.BoolType):
			result = types.BoolValue(rv.IsValid() && rv.Bool())
		case typ.Equal(types.Int64Type):
			number := int64(0)
			if rv.IsValid() {
				number = rv.Int()
			}
			result = types.Int64Value(number)
		default:
			diagnostics.AddError("Unable to Flatten Pod Specification", fmt.Sprintf("Unsupported Pod type %T at %s.", typ, key))
			result = types.StringNull()
		}
	}
	if prior != nil && podQuantityPath(names) {
		result = podPreserveQuantity(prior, result)
	}
	return result
}
