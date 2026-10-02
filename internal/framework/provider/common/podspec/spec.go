// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package podspec

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
	sdkschema "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	corev1 "k8s.io/api/core/v1"
)

var podSpecOmitted = &struct{}{}

// Built is the PodSpec schema and conversions for one Options value. It is
// shared by every resource with those Options and must be treated as read-only.
type Built struct {
	// Spec is the frozen "spec" block.
	Spec schema.ListNestedBlock

	template   bool
	refresh    bool
	objectType basetypes.ObjectTypable
	computed   map[string]bool
	blocks     map[string]bool
}

var built sync.Map // Options -> *Built

// For returns the PodSpec built for o, building and freezing it on first use.
func For(o Options) *Built {
	if b, ok := built.Load(o); ok {
		return b.(*Built)
	}
	b, _ := built.LoadOrStore(o, build(o))
	return b.(*Built)
}

func build(o Options) *Built {
	spec := common.FreezeListNestedBlock(builder{o: o}.specBlock())
	b := &Built{
		Spec:       spec,
		template:   o.Template,
		objectType: spec.NestedObject.CustomType,
		computed:   map[string]bool{},
		blocks:     map[string]bool{},
	}
	podSpecComputedPaths(spec.NestedObject.Attributes, spec.NestedObject.Blocks, "spec", b.computed)
	podSpecBlockPaths(spec.NestedObject, "spec", b.blocks)
	return b
}

// ObjectType returns the type of one "spec" element.
func (b *Built) ObjectType() basetypes.ObjectTypable {
	return b.objectType
}

// ExpandSpec converts the "spec" list at the given path into a PodSpec
// through the shared pure SDKv2 expander. A null or empty list yields an
// empty PodSpec, as SDKv2 did. Unknown values are rejected unless the schema
// marks them API-computed, in which case the API default is selected.
func (b *Built) ExpandSpec(ctx context.Context, value types.List, at path.Path) (corev1.PodSpec, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if value.IsUnknown() {
		diagnostics.AddAttributeError(at, "Unknown Pod Template Specification", "The pod template spec must be known before it is sent to Kubernetes.")
		return corev1.PodSpec{}, diagnostics
	}
	if len(value.Elements()) > 1 {
		diagnostics.AddAttributeError(at, "Invalid Pod Template Specification", "At most one pod template spec block is allowed.")
		return corev1.PodSpec{}, diagnostics
	}
	raw := podSpecAPIValue(ctx, value, at, "spec", b.computed, &diagnostics)
	if diagnostics.HasError() {
		return corev1.PodSpec{}, diagnostics
	}
	result, err := kubernetes.ExpandPodSpecForFramework(raw.([]interface{}))
	if err != nil {
		diagnostics.AddAttributeError(at, "Unable to Expand Pod Template Specification", err.Error())
		return corev1.PodSpec{}, diagnostics
	}
	return *result, diagnostics
}

// FlattenSpec converts an API PodSpec into the "spec" list after a write. The
// baseline is the plan; it decides null versus empty collection ownership,
// retains semantically equal quantity spellings and keeps a planned "" on
// API-defaulted strings.
func (b *Built) FlattenSpec(ctx context.Context, spec corev1.PodSpec, baseline types.List, at path.Path) (types.List, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	raw, err := kubernetes.FlattenPodSpecForFramework(spec, b.template)
	if err != nil {
		diagnostics.AddAttributeError(at, "Unable to Flatten Pod Template Specification", err.Error())
		return types.ListNull(b.objectType), diagnostics
	}
	preserveProjectedSourceGroups(ctx, spec, baseline, raw)
	value := podSpecStateValue(ctx, types.ListType{ElemType: b.objectType}, raw, baseline, []string{"spec"}, "spec", b, &diagnostics)
	if diagnostics.HasError() {
		return types.ListNull(b.objectType), diagnostics
	}
	return value.(types.List), diagnostics
}

// RefreshSpec is FlattenSpec for reads, with the prior state as the baseline.
// An API-defaulted string configured as "" records the live value, as SDKv2 did.
func (b *Built) RefreshSpec(ctx context.Context, spec corev1.PodSpec, baseline types.List, at path.Path) (types.List, diag.Diagnostics) {
	r := *b
	r.refresh = true
	return r.FlattenSpec(ctx, spec, baseline, at)
}

// SDKv2 passed a block with nothing configured as [nil]. For the pod security
// context that omits runAsNonRoot rather than sending false, so a block such as
// security_context { supplemental_groups = [] } leaves the API object unchanged.
var podZeroBlockUnset = map[string]bool{"spec.security_context": true}

// This boundary translates known Framework values to the input of the shared
// pure PodSpec API helpers, never to ResourceData. Unknowns cannot reach client-go.
func podSpecAPIValue(ctx context.Context, value attr.Value, at path.Path, key string, computed map[string]bool, diagnostics *diag.Diagnostics) interface{} {
	if value.IsUnknown() {
		if computed[key] {
			return podSpecOmitted
		}
		diagnostics.AddAttributeError(at, "Unknown Pod Template Specification", "This value must be known before the pod template spec is sent to Kubernetes.")
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
			diagnostics.AddAttributeError(at, "Invalid Pod Template Specification", "A nested pod template spec object cannot be null.")
			return nil
		}
		result := make(map[string]interface{}, len(v.Attributes()))
		for name, entry := range v.Attributes() {
			child := podSpecAPIValue(ctx, entry, at.AtName(name), key+"."+name, computed, diagnostics)
			if child != podSpecOmitted {
				result[name] = child
			}
		}
		return result
	case types.Map:
		result := make(map[string]interface{}, len(v.Elements()))
		for name, entry := range v.Elements() {
			if entry.IsUnknown() {
				diagnostics.AddAttributeError(at.AtMapKey(name), "Unknown Pod Template Specification", "A configured map entry must be known before it is sent to Kubernetes.")
				continue
			}
			child := podSpecAPIValue(ctx, entry, at.AtMapKey(name), key, computed, diagnostics)
			if child != podSpecOmitted {
				result[name] = child
			}
		}
		return result
	case types.List:
		if podZeroBlockUnset[key] && len(v.Elements()) == 1 && podZeroValue(v.Elements()[0]) {
			return []interface{}{nil}
		}
		result := make([]interface{}, len(v.Elements()))
		for index, entry := range v.Elements() {
			if entry.IsUnknown() {
				diagnostics.AddAttributeError(at.AtListIndex(index), "Unknown Pod Template Specification", "A configured list element must be known before it is sent to Kubernetes.")
				continue
			}
			result[index] = podSpecAPIValue(ctx, entry, at.AtListIndex(index), key, computed, diagnostics)
		}
		return result
	case types.Set:
		result := make([]interface{}, 0, len(v.Elements()))
		for index, entry := range v.Elements() {
			if entry.IsUnknown() {
				diagnostics.AddAttributeError(at, "Unknown Pod Template Specification", "A configured set element must be known before it is sent to Kubernetes.")
				continue
			}
			result = append(result, podSpecAPIValue(ctx, entry, at.AtListIndex(index), key, computed, diagnostics))
		}
		// The SDKv2 expanders read sets as *schema.Set.
		hash := sdkschema.HashString
		if v.ElementType(ctx).Equal(types.Int64Type) {
			hash = sdkschema.HashInt
		}
		return sdkschema.NewSet(hash, result)
	default:
		diagnostics.AddAttributeError(at, "Unable to Expand Pod Template Specification", fmt.Sprintf("Unsupported value type %T.", value))
		return nil
	}
}

func podSpecComputedPaths(attributes map[string]schema.Attribute, blocks map[string]schema.Block, prefix string, computed map[string]bool) {
	for name, attribute := range attributes {
		key := prefix + "." + name
		if attribute.IsComputed() && !podAttributeHasDefault(attribute) {
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

// Zero-value defaults make SDK-optional scalars Computed in the Framework, but
// only API-computed paths may legitimately remain unknown during apply.
func podAttributeHasDefault(attribute schema.Attribute) bool {
	switch a := attribute.(type) {
	case schema.StringAttribute:
		return a.Default != nil
	case schema.BoolAttribute:
		return a.Default != nil
	case schema.Int64Attribute:
		return a.Default != nil
	case schema.ListAttribute:
		return a.Default != nil
	case schema.MapAttribute:
		return a.Default != nil
	case schema.SetAttribute:
		return a.Default != nil
	}
	return false
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
// Read those values through the native template PodSpec type so every SDKv2 path
// has its declared Terraform type without erasing null/empty collection ownership.
func podSpecStateValue(ctx context.Context, typ attr.Type, raw interface{}, prior attr.Value, names []string, key string, b *Built, diagnostics *diag.Diagnostics) attr.Value {
	if set, ok := raw.(*sdkschema.Set); ok {
		raw = set.List()
	}
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
			entries[name] = podSpecStateValue(ctx, childType, child, old[name], childNames, key+"."+name, b, diagnostics)
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
		if count == 0 && !b.blocks[key] && prior != nil && prior.IsNull() {
			return types.ListNull(t.ElemType)
		}
		if count == 0 && !rv.IsValid() && !b.blocks[key] && prior == nil {
			return types.ListNull(t.ElemType)
		}
		// As in SDKv2, a block holding only zero values is the same as no block.
		if count == 0 && b.blocks[key] && len(previous) == 1 && podZeroValue(previous[0]) {
			return prior
		}
		entries := make([]attr.Value, count)
		for i := 0; i < count; i++ {
			var old attr.Value
			if i < len(previous) {
				old = previous[i]
			}
			entries[i] = podSpecStateValue(ctx, t.ElemType, rv.Index(i).Interface(), old, names, key, b, diagnostics)
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
				entries = append(entries, podSpecStateValue(ctx, t.ElemType, rv.Index(i).Interface(), nil, names, key, b, diagnostics))
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
				entries[name] = podSpecStateValue(ctx, t.ElemType, iter.Value().Interface(), nil, nil, key, b, diagnostics)
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
			if p, ok := prior.(types.String); ok && b.computed[key] && podKeepUnsetString(p, text, b.refresh) {
				result = p
			}
		case typ.Equal(types.BoolType):
			result = types.BoolValue(rv.IsValid() && rv.Bool())
		case typ.Equal(types.Int64Type):
			number := int64(0)
			if rv.IsValid() {
				number = rv.Int()
			}
			result = types.Int64Value(number)
		default:
			diagnostics.AddError("Unable to Flatten Pod Template Specification", fmt.Sprintf("Unsupported type %T at %s.", typ, key))
			result = types.StringNull()
		}
	}
	if prior != nil && podQuantityPath(names) {
		result = podPreserveQuantity(prior, result)
	}
	if prior != nil && podHTTPGetPath(names) {
		result = podPreserveHTTPGetPath(prior, result)
	}
	return result
}

// An API-defaulted string the configuration leaves unset keeps its baseline:
// null when older state never stored the field and the API reports it empty,
// and a planned "" after a write. A read records the live value instead.
func podKeepUnsetString(baseline types.String, api string, refresh bool) bool {
	switch {
	case baseline.IsUnknown():
		return false
	case baseline.IsNull():
		return api == ""
	default:
		return !refresh && baseline.ValueString() == ""
	}
}

func podZeroValue(value attr.Value) bool {
	switch v := value.(type) {
	case types.Object:
		for _, child := range v.Attributes() {
			if !podZeroValue(child) {
				return false
			}
		}
		return !v.IsUnknown()
	case types.List:
		for _, element := range v.Elements() {
			if !podZeroValue(element) {
				return false
			}
		}
		return !v.IsUnknown()
	}
	return !podNonzeroValue(value)
}
