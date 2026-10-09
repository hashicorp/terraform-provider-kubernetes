// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package podspec

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
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
	// zeroAbsent marks blocks that Kubernetes returns as absent when sent with
	// only zero values, so a read keeps such a configured block.
	zeroAbsent map[string]bool
	// absentZero marks blocks that Kubernetes returns with one zero-valued
	// element when sent without them, so a write keeps a planned empty list.
	absentZero map[string]bool
	// spelling keeps a prior numeric string that Kubernetes stores as a number.
	spelling map[string]func(prior, current types.String) types.String
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
		spelling:   map[string]func(prior, current types.String) types.String{},
	}
	podSpecComputedPaths(spec.NestedObject.Attributes, spec.NestedObject.Blocks, "spec", b.computed)
	podSpecSpellingPaths(spec.NestedObject, "spec", b.spelling)
	podSpecBlockPaths(spec.NestedObject, "spec", b.blocks)
	b.zeroAbsent = podZeroAbsentBlocks(b)
	b.absentZero = podAbsentZeroBlocks(b)
	return b
}

// ObjectType returns the type of one "spec" element.
func (b *Built) ObjectType() basetypes.ObjectTypable {
	return b.objectType
}

// ExpandSpec converts the "spec" list at path at into a PodSpec with the SDKv2
// expander. Unknown values are rejected unless the schema marks them
// API-computed. A write payload passes the configuration, which decides how
// zero-valued podZeroBlockUnset blocks are sent; without one they are unset,
// so expansions of a plan and a state agree.
func (b *Built) ExpandSpec(ctx context.Context, value types.List, config *tfsdk.Config, at path.Path) (corev1.PodSpec, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	configured := types.ListNull(b.objectType)
	if config != nil {
		diagnostics.Append(config.GetAttribute(ctx, at, &configured)...)
	}
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
	podUnsetZeroBlocks(raw.([]interface{}), value, configured)
	result, err := expandSDKPodSpec(raw.([]interface{}))
	if err != nil {
		diagnostics.AddAttributeError(at, "Unable to Expand Pod Template Specification", err.Error())
		return corev1.PodSpec{}, diagnostics
	}
	return *result, diagnostics
}

// FlattenSpec converts an API PodSpec into the "spec" list after a write. The
// plan, as baseline, decides null versus empty values, kept spellings and the
// built-in tolerations a bare Pod records.
func (b *Built) FlattenSpec(ctx context.Context, spec corev1.PodSpec, baseline types.List, at path.Path) (types.List, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if !b.template {
		spec.Tolerations = podKeptTolerations(spec.Tolerations, baseline)
	}
	raw, err := flattenSDKPodSpec(spec)
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

// podKeptTolerations drops the tolerations of built-in taints, which
// Kubernetes adds to a bare Pod on its own, unless the baseline holds an equal
// one. Without a baseline, as on import, it drops them all.
func podKeptTolerations(tolerations []corev1.Toleration, baseline types.List) []corev1.Toleration {
	configured := podSpecBlock(baseline, "toleration")
	kept := make([]corev1.Toleration, 0, len(tolerations))
	for _, toleration := range tolerations {
		if kubernetes.IsBuiltInToleration(toleration.Key) {
			i := slices.IndexFunc(configured, func(element attr.Value) bool { return podTolerationMatches(element, toleration) })
			if i < 0 {
				continue
			}
			configured = slices.Delete(configured, i, i+1)
		}
		kept = append(kept, toleration)
	}
	return kept
}

// podTolerationMatches reports whether a "toleration" element equals
// toleration as the expander converts it. An unknown attribute matches any
// value, so a planned element still finds its live toleration.
func podTolerationMatches(element attr.Value, toleration corev1.Toleration) bool {
	object, ok := element.(types.Object)
	if !ok || object.IsNull() || object.IsUnknown() {
		return false
	}
	live := map[string]string{
		"key":      toleration.Key,
		"operator": string(toleration.Operator),
		"value":    toleration.Value,
		"effect":   string(toleration.Effect),
	}
	for name, value := range object.Attributes() {
		s, ok := value.(types.String)
		if !ok {
			return false
		}
		if s.IsUnknown() {
			continue
		}
		if name != "toleration_seconds" {
			if s.ValueString() != live[name] {
				return false
			}
			continue
		}
		if s.ValueString() == "" {
			if toleration.TolerationSeconds != nil {
				return false
			}
			continue
		}
		n, err := strconv.ParseInt(s.ValueString(), 10, 64)
		if err != nil || toleration.TolerationSeconds == nil || *toleration.TolerationSeconds != n {
			return false
		}
	}
	return true
}

// A pod security_context holding only zero values is sent as unset, as SDKv2
// sent it as [nil], unless its configuration sets a bool or a nested block:
// SDKv2 then sent runAsNonRoot = false, which Pod Security admission checks.
var podZeroBlockUnset = map[string]bool{"spec.security_context": true}

// podUnsetZeroBlocks sends the zero-valued podZeroBlockUnset blocks of raw as
// unset unless configured sets a value in them.
func podUnsetZeroBlocks(raw []interface{}, value, configured types.List) {
	if len(raw) != 1 {
		return
	}
	spec, _ := raw[0].(map[string]interface{})
	for key := range podZeroBlockUnset {
		name := strings.TrimPrefix(key, "spec.")
		planned := podSpecBlock(value, name)
		if len(planned) == 1 && podZeroValue(planned[0]) && !slices.ContainsFunc(podSpecBlock(configured, name), podConfigSets) {
			spec[name] = []interface{}{nil}
		}
	}
}

// podSpecBlock returns the elements of the named block in a "spec" list.
func podSpecBlock(spec types.List, name string) []attr.Value {
	if len(spec.Elements()) != 1 {
		return nil
	}
	object, _ := spec.Elements()[0].(types.Object)
	block, _ := object.Attributes()[name].(types.List)
	return block.Elements()
}

// podConfigSets reports whether a configured value is one SDKv2 sent: any
// bool, non-empty string or collection, or nested block element.
func podConfigSets(value attr.Value) bool {
	if value == nil || value.IsNull() {
		return false
	}
	switch v := value.(type) {
	case types.Object:
		for _, child := range v.Attributes() {
			if podConfigSets(child) {
				return true
			}
		}
		return false
	case types.Bool:
		return true
	}
	return podNonzeroValue(value)
}

// podSpecAPIValue converts a known Framework value into SDKv2 expander input.
func podSpecAPIValue(ctx context.Context, value attr.Value, at path.Path, key string, computed map[string]bool, diagnostics *diag.Diagnostics) interface{} {
	// These pointer fields were absent from older state. In particular, an
	// omitted host_users must never become false through ValueBool().
	if value.IsNull() && (key == "spec.host_users" || podProcMountPath(key)) {
		return podSpecOmitted
	}
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
			if podContainerResourcesPath(key) || key == "spec.volume.image" {
				return podSpecOmitted
			}
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
		switch nested := attribute.(type) {
		case schema.ListNestedAttribute:
			podSpecComputedPaths(nested.NestedObject.Attributes, nil, key, computed)
		case schema.SingleNestedAttribute:
			podSpecComputedPaths(nested.Attributes, nil, key, computed)
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

// podSpecSpellingPaths records the strings Kubernetes stores as numbers.
func podSpecSpellingPaths(object schema.NestedBlockObject, prefix string, spelling map[string]func(prior, current types.String) types.String) {
	for name, attribute := range object.Attributes {
		if a, ok := attribute.(schema.StringAttribute); ok {
			if keep := podSpelling(a.Validators); keep != nil {
				spelling[prefix+"."+name] = keep
			}
		}
	}
	for name, block := range object.Blocks {
		if nested, ok := block.(schema.ListNestedBlock); ok {
			podSpecSpellingPaths(nested.NestedObject, prefix+"."+name, spelling)
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

// podSpecStateValue converts SDKv2 flattener output into a value of typ, with
// prior deciding null versus empty.
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
		if key == "spec.volume.image" {
			// Older state could not record admission-supplied image sources.
			// Keep those unconfigured sources unowned; import has no baseline
			// and records the live source so a matching configuration is safe.
			if !rv.IsValid() || (prior != nil && prior.IsNull()) {
				return types.ObjectNull(t.AttrTypes)
			}
		}
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
		if podContainerResourcesPath(key) && !b.refresh && prior != nil && prior.IsNull() && podZeroValue(v) {
			return prior
		}
		result = v
	case types.ListType:
		var previous []attr.Value
		plannedEmpty := false
		if list, ok := prior.(types.List); ok && !list.IsNull() && !list.IsUnknown() {
			previous = list.Elements()
			plannedEmpty = len(previous) == 0
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
		// A block holding only zero values is the same as no block after a write;
		// a read keeps it only if Kubernetes cannot hold it.
		if count == 0 && b.blocks[key] && len(previous) == 1 && podZeroValue(previous[0]) && (!b.refresh || b.zeroAbsent[key]) {
			return prior
		}
		entries := make([]attr.Value, count)
		var volumesByName map[string]attr.Value
		if key == "spec.volume" {
			volumesByName = make(map[string]attr.Value, len(previous))
			for _, value := range previous {
				if volume, ok := value.(types.Object); ok {
					if name, ok := volume.Attributes()["name"].(types.String); ok && !name.IsUnknown() && !name.IsNull() {
						volumesByName[name.ValueString()] = value
					}
				}
			}
		}
		for i := 0; i < count; i++ {
			var old attr.Value
			if i < len(previous) {
				old = previous[i]
			}
			// Admission may reorder volumes; image ownership belongs to a
			// named volume, not the element previously at this index.
			if volumesByName != nil {
				volume, _ := rv.Index(i).Interface().(map[string]interface{})
				name, _ := volume["name"].(string)
				old = volumesByName[name]
			}
			entries[i] = podSpecStateValue(ctx, t.ElemType, rv.Index(i).Interface(), old, names, key, b, diagnostics)
		}
		// Kubernetes holds some blocks, such as container resources, even when
		// none was sent, so a planned empty list stays empty after a write.
		if count == 1 && plannedEmpty && !b.refresh && b.absentZero[key] && podZeroValue(entries[0]) {
			return prior
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
			if podProcMountPath(key) && !rv.IsValid() {
				if prior == nil || prior.IsNull() || prior.IsUnknown() {
					return types.StringNull()
				}
				return types.StringValue(string(corev1.DefaultProcMount))
			}
			text := ""
			if rv.IsValid() {
				text = fmt.Sprint(rv.Interface())
			}
			result = types.StringValue(text)
			if p, ok := prior.(types.String); ok {
				if b.computed[key] && podKeepUnsetString(p, text, b.refresh) {
					result = p
				} else if keep := b.spelling[key]; keep != nil {
					result = keep(p, types.StringValue(text))
				}
			}
		case typ.Equal(types.BoolType):
			if key == "spec.host_users" && !rv.IsValid() {
				if prior == nil || prior.IsNull() || prior.IsUnknown() {
					return types.BoolNull()
				}
				return types.BoolValue(true)
			}
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

// podZeroAbsentBlocks records the blocks that come back absent when sent with
// only zero values through the expander, JSON and the flattener.
func podZeroAbsentBlocks(b *Built) map[string]bool {
	result := map[string]bool{}
	for key := range b.blocks {
		flat, ok := podRoundTrip(b, podZeroRaw(b.objectType, "spec", key))
		if ok && podRawAbsent(flat, strings.Split(key, ".")[1:]) {
			result[key] = true
		}
	}
	return result
}

// podAbsentZeroBlocks records the lists of objects that come back present when
// left out of a zero-valued enclosing element.
func podAbsentZeroBlocks(b *Built) map[string]bool {
	keys := map[string]bool{}
	podObjectListPaths(b.objectType, "spec", keys)
	result := map[string]bool{}
	for key := range keys {
		parent := key[:strings.LastIndex(key, ".")]
		flat, ok := podRoundTrip(b, podZeroRaw(b.objectType, "spec", parent))
		if ok && !podRawAbsent(flat, strings.Split(key, ".")[1:]) {
			result[key] = true
		}
	}
	return result
}

// expandSDKPodSpec adapts object arguments to SDKv2 singleton lists without
// changing the caller's value.
func expandSDKPodSpec(spec []interface{}) (*corev1.PodSpec, error) {
	adapted := make([]interface{}, len(spec))
	for i, entry := range spec {
		object, ok := entry.(map[string]interface{})
		if !ok {
			adapted[i] = entry
			continue
		}
		object = maps.Clone(object)
		for _, name := range []string{"container", "init_container"} {
			containers, ok := object[name].([]interface{})
			if !ok {
				continue
			}
			converted := make([]interface{}, len(containers))
			for j, entry := range containers {
				container, ok := entry.(map[string]interface{})
				if !ok {
					converted[j] = entry
					continue
				}
				container = maps.Clone(container)
				if resources, ok := container["resources"].(map[string]interface{}); ok {
					container["resources"] = []interface{}{resources}
				}
				converted[j] = container
			}
			object[name] = converted
		}
		if volumes, ok := object["volume"].([]interface{}); ok {
			converted := make([]interface{}, len(volumes))
			for j, entry := range volumes {
				volume, ok := entry.(map[string]interface{})
				if !ok {
					converted[j] = entry
					continue
				}
				volume = maps.Clone(volume)
				if image, ok := volume["image"].(map[string]interface{}); ok {
					volume["image"] = []interface{}{image}
				}
				converted[j] = volume
			}
			object["volume"] = converted
		}
		adapted[i] = object
	}
	return kubernetes.ExpandPodSpec(adapted)
}

// flattenSDKPodSpec adapts SDKv2 singleton lists to object arguments.
func flattenSDKPodSpec(spec corev1.PodSpec) ([]interface{}, error) {
	flat, err := kubernetes.FlattenPodSpec(spec)
	if err != nil {
		return nil, err
	}
	for _, entry := range flat {
		object := entry.(map[string]interface{})
		for _, name := range []string{"container", "init_container"} {
			containers, _ := object[name].([]interface{})
			for _, entry := range containers {
				container := entry.(map[string]interface{})
				// The SDK flattener always returns one resource-requirements object.
				container["resources"] = container["resources"].([]interface{})[0]
			}
		}
		volumes, _ := object["volume"].([]interface{})
		for _, entry := range volumes {
			volume := entry.(map[string]interface{})
			if image, ok := volume["image"].([]interface{}); ok && len(image) == 1 {
				volume["image"] = image[0]
			} else {
				delete(volume, "image")
			}
		}
	}
	return flat, nil
}

func podProcMountPath(key string) bool {
	return key == "spec.container.security_context.proc_mount" || key == "spec.init_container.security_context.proc_mount"
}

// NormalizeFeatureDefaults makes effective API defaults compare equal to
// omission. Callers use a copy; this does not change write payloads or state.
func NormalizeFeatureDefaults(spec *corev1.PodSpec) {
	if spec.HostUsers != nil && *spec.HostUsers {
		spec.HostUsers = nil
	}
	for _, containers := range [][]corev1.Container{spec.Containers, spec.InitContainers} {
		for i := range containers {
			if sc := containers[i].SecurityContext; sc != nil && sc.ProcMount != nil && *sc.ProcMount == corev1.DefaultProcMount {
				sc.ProcMount = nil
			}
		}
	}
}

func podRoundTrip(b *Built, element interface{}) (interface{}, bool) {
	spec, err := expandSDKPodSpec([]interface{}{element})
	if err != nil {
		return nil, false
	}
	var echo corev1.PodSpec
	if data, err := json.Marshal(spec); err != nil || json.Unmarshal(data, &echo) != nil {
		return nil, false
	}
	flat, err := flattenSDKPodSpec(echo)
	return flat, err == nil
}

func podObjectListPaths(typ attr.Type, key string, keys map[string]bool) {
	switch t := typ.(type) {
	case basetypes.ObjectTypable:
		object, _ := t.ValueType(context.Background()).(basetypes.ObjectValuable).ToObjectValue(context.Background())
		for name, child := range object.AttributeTypes(context.Background()) {
			podObjectListPaths(child, key+"."+name, keys)
		}
	case types.ListType:
		if _, ok := t.ElemType.(basetypes.ObjectTypable); ok {
			keys[key] = true
			podObjectListPaths(t.ElemType, key, keys)
		}
	}
}

// podZeroRaw is the expander input for a spec holding one zero-valued block at
// target, with one zero-valued element in each enclosing block.
func podZeroRaw(typ attr.Type, key, target string) interface{} {
	switch t := typ.(type) {
	case basetypes.ObjectTypable:
		object, _ := t.ValueType(context.Background()).(basetypes.ObjectValuable).ToObjectValue(context.Background())
		result := map[string]interface{}{}
		for name, child := range object.AttributeTypes(context.Background()) {
			result[name] = podZeroRaw(child, key+"."+name, target)
		}
		return result
	case types.ListType:
		if key == target && podZeroBlockUnset[key] {
			return []interface{}{nil}
		}
		if key == target || strings.HasPrefix(target, key+".") {
			return []interface{}{podZeroRaw(t.ElemType, key, target)}
		}
		return []interface{}{}
	case types.SetType:
		if t.ElemType.Equal(types.Int64Type) {
			return sdkschema.NewSet(sdkschema.HashInt, nil)
		}
		return sdkschema.NewSet(sdkschema.HashString, nil)
	case types.MapType:
		return map[string]interface{}{}
	}
	switch {
	case typ.Equal(types.BoolType):
		return false
	case typ.Equal(types.Int64Type):
		return 0
	}
	return ""
}

// podRawAbsent reports whether the flattened spec has no element at the path
// of block names, following the first element of each enclosing block.
func podRawAbsent(raw interface{}, names []string) bool {
	rv := reflect.ValueOf(raw)
	for i := 0; ; i++ {
		for rv.IsValid() && (rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface) {
			rv = rv.Elem()
		}
		if !rv.IsValid() || rv.Kind() != reflect.Slice || rv.Len() == 0 {
			return true
		}
		if i == len(names) {
			return false
		}
		rv = rv.Index(0).Elem()
		if !rv.IsValid() || rv.Kind() != reflect.Map {
			return true
		}
		rv = rv.MapIndex(reflect.ValueOf(names[i]))
	}
}

// Satisfies reports whether a flattened value is one Terraform accepts for the
// planned value after an apply: an unknown planned value matches anything and
// an empty block list matches a null one.
func Satisfies(actual, planned attr.Value) bool {
	if planned.IsUnknown() {
		return true
	}
	switch x := actual.(type) {
	case types.List:
		y, ok := planned.(types.List)
		if !ok || x.IsUnknown() {
			return actual.Equal(planned)
		}
		if len(x.Elements()) != len(y.Elements()) {
			return false
		}
		for i, element := range x.Elements() {
			if !Satisfies(element, y.Elements()[i]) {
				return false
			}
		}
		return true
	case types.Object:
		y, ok := planned.(types.Object)
		if !ok || x.IsNull() || x.IsUnknown() || y.IsNull() {
			return actual.Equal(planned)
		}
		for name, value := range y.Attributes() {
			child, ok := x.Attributes()[name]
			if !ok || !Satisfies(child, value) {
				return false
			}
		}
		return len(x.Attributes()) == len(y.Attributes())
	}
	return actual.Equal(planned)
}

func podContainerResourcesPath(key string) bool {
	return key == "spec.container.resources" || key == "spec.init_container.resources"
}
