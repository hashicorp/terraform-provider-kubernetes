// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	sdkschema "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// legacyValue adapts typed values to the pure workload expanders' input, not to
// ResourceData. Null/unknown computed leaves use the SDK's zero representation.
func legacyValue(ctx context.Context, value attr.Value) (any, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	raw, err := value.ToTerraformValue(ctx)
	if err != nil {
		diagnostics.AddError("Invalid batch value", err.Error())
		return nil, diagnostics
	}
	result, err := legacyTerraformValue(raw)
	if err != nil {
		diagnostics.AddError("Invalid batch value", err.Error())
	}
	return result, diagnostics
}

func legacyTerraformValue(value tftypes.Value) (any, error) {
	known := value.IsKnown() && !value.IsNull()
	switch typ := value.Type().(type) {
	case tftypes.Object:
		fields := map[string]tftypes.Value{}
		if known {
			if err := value.As(&fields); err != nil {
				return nil, err
			}
		}
		result := make(map[string]interface{}, len(typ.AttributeTypes))
		for name, childType := range typ.AttributeTypes {
			child, ok := fields[name]
			if !ok {
				child = tftypes.NewValue(childType, nil)
			}
			converted, err := legacyTerraformValue(child)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			result[name] = converted
		}
		return result, nil
	case tftypes.List, tftypes.Set:
		var elements []tftypes.Value
		if known {
			if err := value.As(&elements); err != nil {
				return nil, err
			}
		}
		result := make([]interface{}, 0, len(elements))
		for _, element := range elements {
			converted, err := legacyTerraformValue(element)
			if err != nil {
				return nil, err
			}
			result = append(result, converted)
		}
		if set, ok := typ.(tftypes.Set); ok {
			switch {
			case set.ElementType.Is(tftypes.String):
				return sdkschema.NewSet(sdkschema.HashString, result), nil
			case set.ElementType.Is(tftypes.Number):
				return sdkschema.NewSet(sdkschema.HashInt, result), nil
			default:
				return nil, fmt.Errorf("unsupported workload set element %s", set.ElementType)
			}
		}
		return result, nil
	case tftypes.Map:
		fields := map[string]tftypes.Value{}
		if known {
			if err := value.As(&fields); err != nil {
				return nil, err
			}
		}
		result := make(map[string]interface{}, len(fields))
		for name, child := range fields {
			converted, err := legacyTerraformValue(child)
			if err != nil {
				return nil, err
			}
			result[name] = converted
		}
		return result, nil
	default:
		switch {
		case value.Type().Is(tftypes.String):
			var result string
			if known {
				if err := value.As(&result); err != nil {
					return nil, err
				}
			}
			return result, nil
		case value.Type().Is(tftypes.Bool):
			var result bool
			if known {
				if err := value.As(&result); err != nil {
					return nil, err
				}
			}
			return result, nil
		case value.Type().Is(tftypes.Number):
			result := 0
			if known {
				var number big.Float
				if err := value.As(&number); err != nil {
					return nil, err
				}
				n, accuracy := number.Int64()
				if accuracy != big.Exact || int64(int(n)) != n {
					return nil, fmt.Errorf("workload integer is out of range: %s", number.String())
				}
				result = int(n)
			}
			return result, nil
		default:
			return nil, fmt.Errorf("unsupported workload type %s", value.Type())
		}
	}
}

type valueField struct {
	typ      attr.Type
	computed bool
	block    bool
	children map[string]valueField
	def      schema.Attribute
}

func blockValueField(block schema.ListNestedBlock) valueField {
	return valueField{
		typ: block.Type(), block: true,
		children: objectValueFields(block.NestedObject.Attributes, block.NestedObject.Blocks),
	}
}

func objectValueFields(attributes map[string]schema.Attribute, blocks map[string]schema.Block) map[string]valueField {
	result := make(map[string]valueField, len(attributes)+len(blocks))
	for name, attribute := range attributes {
		field := valueField{typ: attribute.GetType(), computed: attribute.IsComputed(), def: attribute}
		switch a := attribute.(type) {
		case schema.ListNestedAttribute:
			field.children = objectValueFields(a.NestedObject.Attributes, nil)
		case schema.SetNestedAttribute:
			field.children = objectValueFields(a.NestedObject.Attributes, nil)
		case schema.SingleNestedAttribute:
			field.children = objectValueFields(a.Attributes, nil)
		}
		result[name] = field
	}
	for name, block := range blocks {
		switch b := block.(type) {
		case schema.ListNestedBlock:
			result[name] = blockValueField(b)
		case schema.SetNestedBlock:
			result[name] = valueField{typ: b.Type(), block: true, children: objectValueFields(b.NestedObject.Attributes, b.NestedObject.Blocks)}
		case schema.SingleNestedBlock:
			result[name] = valueField{typ: b.Type(), block: true, children: objectValueFields(b.Attributes, b.Blocks)}
		}
	}
	return result
}

// valueFromAPI reconstructs the explicit Framework schema from the shared pure
// flatteners. JSON normalizes Kubernetes enum, pointer and integer representations.
func valueFromAPI(ctx context.Context, block schema.ListNestedBlock, raw []interface{}, prior types.List) (types.List, diag.Diagnostics) {
	return valueFromAPIField(ctx, blockValueField(block), raw, prior)
}

// valueFromAPIField is valueFromAPI for a precomputed (memoised) block field.
func valueFromAPIField(ctx context.Context, field valueField, raw []interface{}, prior types.List) (types.List, diag.Diagnostics) {
	elemType := field.typ.(types.ListType).ElemType
	var diagnostics diag.Diagnostics
	data, err := json.Marshal(normalizeLegacyCollections(raw))
	if err != nil {
		diagnostics.AddError("Unable to read batch workload", err.Error())
		return types.ListNull(elemType), diagnostics
	}
	var decoded any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&decoded); err != nil {
		diagnostics.AddError("Unable to read batch workload", err.Error())
		return types.ListNull(elemType), diagnostics
	}
	value, err := apiValue(ctx, field, decoded, prior)
	if err != nil {
		diagnostics.AddError("Unable to read batch workload", err.Error())
		return types.ListNull(elemType), diagnostics
	}
	list, ok := value.(types.List)
	if !ok {
		diagnostics.AddError("Invalid batch schema", fmt.Sprintf("Expected list, received %T", value))
		return types.ListNull(elemType), diagnostics
	}
	return list, diagnostics
}

func normalizeLegacyCollections(raw any) any {
	if set, ok := raw.(*sdkschema.Set); ok {
		return normalizeLegacyCollections(set.List())
	}
	switch value := raw.(type) {
	case []interface{}:
		result := make([]interface{}, len(value))
		for i, child := range value {
			result[i] = normalizeLegacyCollections(child)
		}
		return result
	case map[string]interface{}:
		result := make(map[string]interface{}, len(value))
		for name, child := range value {
			result[name] = normalizeLegacyCollections(child)
		}
		return result
	default:
		reflected := reflect.ValueOf(raw)
		if !reflected.IsValid() {
			return nil
		}
		switch reflected.Kind() {
		case reflect.Slice, reflect.Array:
			if reflected.Kind() == reflect.Slice && reflected.IsNil() {
				return nil
			}
			result := make([]interface{}, reflected.Len())
			for i := range result {
				result[i] = normalizeLegacyCollections(reflected.Index(i).Interface())
			}
			return result
		case reflect.Map:
			if reflected.IsNil() {
				return nil
			}
			if reflected.Type().Key().Kind() == reflect.String {
				result := make(map[string]interface{}, reflected.Len())
				iter := reflected.MapRange()
				for iter.Next() {
					result[iter.Key().String()] = normalizeLegacyCollections(iter.Value().Interface())
				}
				return result
			}
		}
		return raw
	}
}

func apiValue(ctx context.Context, field valueField, raw any, prior attr.Value) (attr.Value, error) {
	typ := field.typ.TerraformType(ctx)
	null := func() (attr.Value, error) { return field.typ.ValueFromTerraform(ctx, tftypes.NewValue(typ, nil)) }
	if raw == nil && field.def != nil {
		value, ok, err := fieldDefault(ctx, field.def)
		if err != nil {
			return nil, err
		}
		if ok {
			return value, nil
		}
	}
	switch collection := typ.(type) {
	case tftypes.List, tftypes.Set:
		var elementType tftypes.Type
		if list, ok := collection.(tftypes.List); ok {
			elementType = list.ElementType
		} else {
			elementType = collection.(tftypes.Set).ElementType
		}
		elements, ok := raw.([]interface{})
		if raw != nil && !ok {
			return nil, fmt.Errorf("expected collection, received %T", raw)
		}
		if len(elements) == 0 && !field.block && (prior == nil || prior.IsNull()) && !field.computed {
			return null()
		}
		var previous []attr.Value
		switch p := prior.(type) {
		case types.List:
			previous = p.Elements()
		case types.Set:
			previous = p.Elements()
		}
		childType, err := attrTypeForElement(field.typ)
		if err != nil {
			return nil, err
		}
		values := make([]tftypes.Value, len(elements))
		for i, element := range elements {
			var previousElement attr.Value
			if i < len(previous) {
				previousElement = previous[i]
			}
			value, err := apiValue(ctx, valueField{typ: childType, children: field.children, computed: true}, element, previousElement)
			if err != nil {
				return nil, fmt.Errorf("[%d]: %w", i, err)
			}
			values[i], err = value.ToTerraformValue(ctx)
			if err != nil {
				return nil, err
			}
			if !values[i].Type().Equal(elementType) {
				return nil, fmt.Errorf("invalid collection element type %s", values[i].Type())
			}
		}
		return field.typ.ValueFromTerraform(ctx, tftypes.NewValue(typ, values))
	case tftypes.Object:
		if raw == nil {
			return null()
		}
		fields, ok := raw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("expected object, received %T", raw)
		}
		previous := map[string]attr.Value{}
		if object, ok := prior.(types.Object); ok {
			previous = object.Attributes()
		}
		objectType, ok := field.typ.(types.ObjectType)
		if !ok {
			return nil, fmt.Errorf("unsupported workload object type %T", field.typ)
		}
		values := make(map[string]tftypes.Value, len(collection.AttributeTypes))
		for name := range collection.AttributeTypes {
			child, ok := field.children[name]
			if !ok {
				child = valueField{typ: objectType.AttrTypes[name], computed: field.computed}
			}
			value, err := apiValue(ctx, child, fields[name], previous[name])
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			values[name], err = value.ToTerraformValue(ctx)
			if err != nil {
				return nil, err
			}
		}
		return field.typ.ValueFromTerraform(ctx, tftypes.NewValue(typ, values))
	case tftypes.Map:
		fields, ok := raw.(map[string]interface{})
		if raw != nil && !ok {
			return nil, fmt.Errorf("expected map, received %T", raw)
		}
		if len(fields) == 0 && (prior == nil || prior.IsNull() || prior.IsUnknown()) {
			return null()
		}
		childType, err := attrTypeForElement(field.typ)
		if err != nil {
			return nil, err
		}
		var previous map[string]attr.Value
		if m, ok := prior.(types.Map); ok {
			previous = m.Elements()
		}
		values := make(map[string]tftypes.Value, len(fields))
		for name, rawChild := range fields {
			value, err := apiValue(ctx, valueField{typ: childType, computed: true}, rawChild, previous[name])
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			values[name], err = value.ToTerraformValue(ctx)
			if err != nil {
				return nil, err
			}
		}
		return field.typ.ValueFromTerraform(ctx, tftypes.NewValue(typ, values))
	default:
		if raw == nil {
			if prior != nil && !prior.IsUnknown() && !prior.IsNull() {
				previous, err := legacyValue(ctx, prior)
				if !err.HasError() && isZeroScalar(previous) {
					return prior, nil
				}
			}
			return null()
		}
		var value any
		switch {
		case typ.Is(tftypes.String):
			switch v := raw.(type) {
			case string:
				value = v
			case json.Number:
				value = v.String()
			default:
				return nil, fmt.Errorf("expected string, received %T", raw)
			}
		case typ.Is(tftypes.Number):
			number, ok := raw.(json.Number)
			if !ok {
				return nil, fmt.Errorf("expected number, received %T", raw)
			}
			parsed, _, err := big.ParseFloat(number.String(), 10, 512, big.ToNearestEven)
			if err != nil {
				return nil, err
			}
			value = parsed
		case typ.Is(tftypes.Bool):
			boolean, ok := raw.(bool)
			if !ok {
				return nil, fmt.Errorf("expected boolean, received %T", raw)
			}
			value = boolean
		default:
			return nil, fmt.Errorf("unsupported scalar %s", typ)
		}
		if !field.computed && isZeroScalar(value) && (prior == nil || prior.IsNull()) {
			return null()
		}
		return field.typ.ValueFromTerraform(ctx, tftypes.NewValue(typ, value))
	}
}

func attrTypeForElement(typ attr.Type) (attr.Type, error) {
	if collection, ok := typ.(attr.TypeWithElementType); ok {
		return collection.ElementType(), nil
	}
	return nil, fmt.Errorf("unsupported collection type %T", typ)
}

func isZeroScalar(value any) bool {
	switch v := value.(type) {
	case string:
		return v == ""
	case bool:
		return !v
	case *big.Float:
		return v.Sign() == 0
	default:
		return value != nil && reflect.ValueOf(value).IsZero()
	}
}

func fieldDefault(ctx context.Context, attribute schema.Attribute) (attr.Value, bool, error) {
	switch a := attribute.(type) {
	case schema.StringAttribute:
		if a.Default != nil {
			var response defaults.StringResponse
			a.Default.DefaultString(ctx, defaults.StringRequest{}, &response)
			if response.Diagnostics.HasError() {
				return nil, false, fmt.Errorf("invalid workload default: %v", response.Diagnostics)
			}
			return response.PlanValue, true, nil
		}
	case schema.BoolAttribute:
		if a.Default != nil {
			var response defaults.BoolResponse
			a.Default.DefaultBool(ctx, defaults.BoolRequest{}, &response)
			if response.Diagnostics.HasError() {
				return nil, false, fmt.Errorf("invalid workload default: %v", response.Diagnostics)
			}
			return response.PlanValue, true, nil
		}
	case schema.Int64Attribute:
		if a.Default != nil {
			var response defaults.Int64Response
			a.Default.DefaultInt64(ctx, defaults.Int64Request{}, &response)
			if response.Diagnostics.HasError() {
				return nil, false, fmt.Errorf("invalid workload default: %v", response.Diagnostics)
			}
			return response.PlanValue, true, nil
		}
	}
	return nil, false, nil
}

// preservePlannedValue resolves computed values without overwriting known plan
// values with API normalization. Read independently detects subsequent drift.
func preservePlannedValue(ctx context.Context, block schema.ListNestedBlock, plan, actual types.List) (types.List, diag.Diagnostics) {
	return preservePlannedValueField(ctx, blockValueField(block), plan, actual)
}

// preservePlannedValueField is preservePlannedValue for a precomputed (memoised) block field.
func preservePlannedValueField(ctx context.Context, field valueField, plan, actual types.List) (types.List, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	plannedRaw, err := plan.ToTerraformValue(ctx)
	if err != nil {
		diagnostics.AddError("Invalid batch plan", err.Error())
		return plan, diagnostics
	}
	actualRaw, err := actual.ToTerraformValue(ctx)
	if err != nil {
		diagnostics.AddError("Invalid batch API value", err.Error())
		return plan, diagnostics
	}
	value, err := resolveComputedValues(ctx, field, plannedRaw, actualRaw)
	if err != nil {
		diagnostics.AddError("Unable to reconcile batch state", err.Error())
		return plan, diagnostics
	}
	converted, err := field.typ.ValueFromTerraform(ctx, value)
	if err != nil {
		diagnostics.AddError("Unable to reconcile batch state", err.Error())
		return plan, diagnostics
	}
	result, ok := converted.(types.List)
	if !ok {
		diagnostics.AddError("Invalid batch schema", fmt.Sprintf("Expected list, got %T", converted))
		return plan, diagnostics
	}
	return result, diagnostics
}

func resolveComputedValues(ctx context.Context, field valueField, plan, actual tftypes.Value) (tftypes.Value, error) {
	if !plan.IsKnown() {
		return actual, nil
	}
	if plan.Equal(actual) {
		return plan, nil
	}
	if plan.IsNull() || actual.IsNull() {
		if emptyTerraformValue(plan) && emptyTerraformValue(actual) {
			return plan, nil
		}
		return tftypes.Value{}, fmt.Errorf("Kubernetes returned a different value than planned")
	}
	switch plan.Type().(type) {
	case tftypes.Object, tftypes.Map:
		var plannedFields, actualFields map[string]tftypes.Value
		if err := plan.As(&plannedFields); err != nil {
			return tftypes.Value{}, err
		}
		if !actual.IsNull() {
			if err := actual.As(&actualFields); err != nil {
				return tftypes.Value{}, err
			}
		}
		if len(plannedFields) != len(actualFields) {
			return tftypes.Value{}, fmt.Errorf("Kubernetes returned different map keys than planned")
		}
		for name, planned := range plannedFields {
			api, ok := actualFields[name]
			if !ok {
				api = tftypes.NewValue(planned.Type(), nil)
			}
			child, ok := field.children[name]
			if !ok {
				element, err := attrTypeForElement(field.typ)
				if err != nil {
					return tftypes.Value{}, err
				}
				child = valueField{typ: element}
			}
			value, err := resolveComputedValues(ctx, child, planned, api)
			if err != nil {
				return tftypes.Value{}, fmt.Errorf("%s: %w", name, err)
			}
			plannedFields[name] = value
		}
		return tftypes.NewValue(plan.Type(), plannedFields), nil
	case tftypes.List, tftypes.Set:
		var plannedElements, actualElements []tftypes.Value
		if err := plan.As(&plannedElements); err != nil {
			return tftypes.Value{}, err
		}
		if !actual.IsNull() {
			if err := actual.As(&actualElements); err != nil {
				return tftypes.Value{}, err
			}
		}
		if len(plannedElements) != len(actualElements) {
			return tftypes.Value{}, fmt.Errorf("API collection size %d does not match planned size %d",
				len(actualElements), len(plannedElements))
		}
		// Workload sets contain primitive values; ordering is not identity.
		if _, ok := plan.Type().(tftypes.Set); ok {
			return tftypes.Value{}, fmt.Errorf("Kubernetes returned different set elements than planned")
		}
		element, err := attrTypeForElement(field.typ)
		if err != nil {
			return tftypes.Value{}, err
		}
		for i := range plannedElements {
			value, err := resolveComputedValues(ctx, valueField{typ: element, children: field.children},
				plannedElements[i], actualElements[i])
			if err != nil {
				return tftypes.Value{}, fmt.Errorf("[%d]: %w", i, err)
			}
			plannedElements[i] = value
		}
		return tftypes.NewValue(plan.Type(), plannedElements), nil
	default:
		planned, err := field.typ.ValueFromTerraform(ctx, plan)
		if err != nil {
			return tftypes.Value{}, err
		}
		api, err := field.typ.ValueFromTerraform(ctx, actual)
		if err != nil {
			return tftypes.Value{}, err
		}
		if semantic, ok := planned.(basetypes.StringValuableWithSemanticEquals); ok {
			if stringValue, ok := api.(basetypes.StringValuable); ok {
				equal, diagnostics := semantic.StringSemanticEquals(ctx, stringValue)
				if diagnostics.HasError() {
					return tftypes.Value{}, fmt.Errorf("unable to compare Kubernetes quantity: %v", diagnostics)
				}
				if equal {
					return plan, nil
				}
			}
		}
		return tftypes.Value{}, fmt.Errorf("Kubernetes returned a different value than planned")
	}
}

func emptyTerraformValue(value tftypes.Value) bool {
	if !value.IsKnown() {
		return false
	}
	if value.IsNull() {
		return true
	}
	switch value.Type().(type) {
	case tftypes.Map:
		var entries map[string]tftypes.Value
		return value.As(&entries) == nil && len(entries) == 0
	case tftypes.List, tftypes.Set:
		var entries []tftypes.Value
		return value.As(&entries) == nil && len(entries) == 0
	default:
		return false
	}
}
