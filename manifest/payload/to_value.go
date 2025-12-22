// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package payload

import (
	"math/big"
	"reflect"
	"strconv"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/morph"
)

// ToTFValue converts a Kubernetes dynamic client unstructured value
// into a Terraform specific tftypes.Value type object
// Arguments:
//   - in : the actual raw unstructured value to be converted
//   - st : the expected type of the converted value
//   - th : type hints (optional, describes ambigous encodings such as
//     IntOrString values in more detail).
//     Pass in empty map when not using hints.
//   - at : attribute path which recursively tracks the conversion.
//     Pass in empty tftypes.AttributePath{}
func ToTFValue(in interface{}, st tftypes.Type, th map[string]string, at *tftypes.AttributePath) (tftypes.Value, error) {
	if st == nil {
		return tftypes.Value{}, at.NewErrorf("[%s] type cannot be nil", at.String())
	}
	if in == nil {
		return tftypes.NewValue(st, nil), nil
	}
	switch t := in.(type) {
	case string:
		switch {
		case st.Is(tftypes.String) || st.Is(tftypes.DynamicPseudoType):
			return tftypes.NewValue(tftypes.String, t), nil
		case st.Is(tftypes.Number):
			num, err := strconv.Atoi(t)
			if err != nil {
				return tftypes.Value{}, err
			}
			return tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(int64(num))), nil
		default:
			return tftypes.Value{}, at.NewErrorf(`[%s] cannot convert payload from "string" to "%s"`, at.String(), st.String())
		}
	case bool:
		switch {
		case st.Is(tftypes.Bool) || st.Is(tftypes.DynamicPseudoType):
			return tftypes.NewValue(tftypes.Bool, in), nil
		default:
			return tftypes.Value{}, at.NewErrorf(`[%s] cannot convert payload from "bool" to "%s"`, at.String(), st.String())
		}
	case int:
		switch {
		case st.Is(tftypes.Number) || st.Is(tftypes.DynamicPseudoType):
			return tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(int64(t))), nil
		case st.Is(tftypes.String):
			ht, ok := th[morph.ValueToTypePath(at).String()]
			if ok && ht == "io.k8s.apimachinery.pkg.util.intstr.IntOrString" { // We store this in state as "string"
				return tftypes.NewValue(tftypes.String, strconv.FormatInt(int64(t), 10)), nil
			}
			return tftypes.Value{}, at.NewErrorf(`[%s] cannot convert payload from "int" to "tftypes.String"`, at.String())
		default:
			return tftypes.Value{}, at.NewErrorf(`[%s] cannot convert payload from "int" to "%s"`, at.String(), st.String())
		}
	case int64:
		switch {
		case st.Is(tftypes.Number) || st.Is(tftypes.DynamicPseudoType):
			return tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(t)), nil
		case st.Is(tftypes.String):
			ht, ok := th[morph.ValueToTypePath(at).String()]
			if ok && ht == "io.k8s.apimachinery.pkg.util.intstr.IntOrString" { // We store this in state as "string"
				return tftypes.NewValue(tftypes.String, strconv.FormatInt(t, 10)), nil
			}
			return tftypes.Value{}, at.NewErrorf(`[%s] cannot convert payload from "int64" to "tftypes.String"`, at.String())
		default:
			return tftypes.Value{}, at.NewErrorf(`[%s] cannot convert payload from "int64" to "%s"`, at.String(), st.String())
		}
	case int32:
		switch {
		case st.Is(tftypes.Number) || st.Is(tftypes.DynamicPseudoType):
			return tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(int64(t))), nil
		case st.Is(tftypes.String):
			ht, ok := th[morph.ValueToTypePath(at).String()]
			if ok && ht == "io.k8s.apimachinery.pkg.util.intstr.IntOrString" { // We store this in state as "string"
				return tftypes.NewValue(tftypes.String, strconv.FormatInt(int64(t), 10)), nil
			}
			return tftypes.Value{}, at.NewErrorf(`[%s] cannot convert payload from "int32" to "tftypes.String"`, at.String())
		default:
			return tftypes.Value{}, at.NewErrorf(`[%s] cannot convert payload from "int32" to "%s"`, at.String(), st.String())
		}
	case int16:
		switch {
		case st.Is(tftypes.Number) || st.Is(tftypes.DynamicPseudoType):
			return tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(int64(t))), nil
		case st.Is(tftypes.String):
			ht, ok := th[morph.ValueToTypePath(at).String()]
			if ok && ht == "io.k8s.apimachinery.pkg.util.intstr.IntOrString" { // We store this in state as "string"
				return tftypes.NewValue(tftypes.String, strconv.FormatInt(int64(t), 10)), nil
			}
			return tftypes.Value{}, at.NewErrorf(`[%s] cannot convert payload from "int16" to "tftypes.String"`, at.String())
		default:
			return tftypes.Value{}, at.NewErrorf(`[%s] cannot convert payload from "int32" to "%s"`, at.String(), st.String())
		}
	case float64:
		switch {
		case st.Is(tftypes.Number) || st.Is(tftypes.DynamicPseudoType):
			return tftypes.NewValue(tftypes.Number, new(big.Float).SetFloat64(t)), nil
		default:
			return tftypes.Value{}, at.NewErrorf(`[%s] cannot convert payload from "float64" to "%s"`, at.String(), st.String())
		}
	case []interface{}:
		switch {
		case st.Is(tftypes.List{}):
			return sliceToTFListValue(t, st, th, at)
		case st.Is(tftypes.Tuple{}):
			return sliceToTFTupleValue(t, st, th, at)
		case st.Is(tftypes.Set{}):
			return sliceToTFSetValue(t, st, th, at)
		case st.Is(tftypes.DynamicPseudoType):
			return sliceToTFDynamicValue(t, st, th, at)
		default:
			return tftypes.Value{}, at.NewErrorf(`[%s] cannot convert payload from "[]interface{}" to "%s"`, at.String(), st.String())
		}
	case map[string]interface{}:
		switch {
		case st.Is(tftypes.Object{}):
			return mapToTFObjectValue(t, st, th, at)
		case st.Is(tftypes.Map{}):
			return mapToTFMapValue(t, st, th, at)
		case st.Is(tftypes.DynamicPseudoType):
			return mapToTFDynamicValue(t, st, th, at)
		default:
			return tftypes.Value{}, at.NewErrorf(`[%s] cannot convert payload from "map[string]interface{}" to "%s"`, at.String(), st.String())
		}
	}
	return tftypes.Value{}, at.NewErrorf(`[%s] cannot convert payload of unknown type "%s"`, at.String(), reflect.TypeOf(in).String())
}

func sliceToTFDynamicValue(in []interface{}, st tftypes.Type, th map[string]string, at *tftypes.AttributePath) (tftypes.Value, error) {
	il := make([]tftypes.Value, len(in))
	oTypes := make([]tftypes.Type, len(in))
	for k, v := range in {
		eap := at.WithElementKeyInt(k)
		var iv tftypes.Value
		iv, err := ToTFValue(v, tftypes.DynamicPseudoType, th, at.WithElementKeyInt(k))
		if err != nil {
			return tftypes.Value{}, eap.NewErrorf("[%s] cannot convert list element '%s' as DynamicPseudoType", eap, err)
		}
		il[k] = iv
		oTypes[k] = iv.Type()
	}
	return tftypes.NewValue(tftypes.Tuple{ElementTypes: oTypes}, il), nil
}

func sliceToTFListValue(in []interface{}, st tftypes.Type, th map[string]string, at *tftypes.AttributePath) (tftypes.Value, error) {
	il := make([]tftypes.Value, 0, len(in))
	schemaElementType := st.(tftypes.List).ElementType
	for k, v := range in {
		eap := at.WithElementKeyInt(k)
		iv, err := ToTFValue(v, schemaElementType, th, at.WithElementKeyInt(k))
		if err != nil {
			return tftypes.Value{}, eap.NewErrorf("[%s] cannot convert list element value: %s", eap, err)
		}
		il = append(il, iv)
	}
	// Use the schema type directly to preserve DynamicPseudoType and other schema-defined types
	return tftypes.NewValue(st, il), nil
}

func sliceToTFTupleValue(in []interface{}, st tftypes.Type, th map[string]string, at *tftypes.AttributePath) (tftypes.Value, error) {
	il := make([]tftypes.Value, len(in))
	schemaTypes := st.(tftypes.Tuple).ElementTypes
	// Handle case where schema has one element type but data has multiple elements
	if len(schemaTypes) == 1 && len(il) > 1 {
		schemaTypes = make([]tftypes.Type, len(in))
		for i := range il {
			schemaTypes[i] = st.(tftypes.Tuple).ElementTypes[0]
		}
	}
	for k, v := range in {
		eap := at.WithElementKeyInt(k)
		et := schemaTypes[k]
		iv, err := ToTFValue(v, et, th, at.WithElementKeyInt(k))
		if err != nil {
			return tftypes.Value{}, eap.NewErrorf("[%s] cannot convert list element [%d] to '%s': %s", eap.String(), k, et.String(), err)
		}
		il[k] = iv
	}
	// Use the schema types to preserve DynamicPseudoType and other schema-defined types
	return tftypes.NewValue(tftypes.Tuple{ElementTypes: schemaTypes}, il), nil
}

func sliceToTFSetValue(in []interface{}, st tftypes.Type, th map[string]string, at *tftypes.AttributePath) (tftypes.Value, error) {
	il := make([]tftypes.Value, len(in))
	schemaElementType := st.(tftypes.Set).ElementType
	for k, v := range in {
		eap := at.WithElementKeyInt(k)
		iv, err := ToTFValue(v, schemaElementType, th, at.WithElementKeyInt(k))
		if err != nil {
			return tftypes.Value{}, eap.NewErrorf("[%s] cannot convert set element [%d] to '%s': %s", eap, k, schemaElementType.String(), err)
		}
		il[k] = iv
	}
	// Use the schema type directly to preserve DynamicPseudoType and other schema-defined types
	return tftypes.NewValue(st, il), nil
}

func mapToTFMapValue(in map[string]interface{}, st tftypes.Type, th map[string]string, at *tftypes.AttributePath) (tftypes.Value, error) {
	im := make(map[string]tftypes.Value)
	schemaElementType := st.(tftypes.Map).ElementType
	for k, v := range in {
		eap := at.WithElementKeyString(k)
		mv, err := ToTFValue(v, schemaElementType, th, eap)
		if err != nil {
			return tftypes.Value{}, eap.NewErrorf("[%s] cannot convert map element '%s' to '%s': err", eap, schemaElementType.String(), err)
		}
		im[k] = mv
	}
	// Use the schema type directly to preserve DynamicPseudoType and other schema-defined types
	return tftypes.NewValue(st, im), nil
}

func mapToTFObjectValue(in map[string]interface{}, st tftypes.Type, th map[string]string, at *tftypes.AttributePath) (tftypes.Value, error) {
	im := make(map[string]tftypes.Value)
	schemaTypes := st.(tftypes.Object).AttributeTypes
	oTypes := make(map[string]tftypes.Type, len(schemaTypes))
	for k, kt := range schemaTypes {
		eap := at.WithAttributeName(k)
		v, ok := in[k]
		if !ok {
			v = nil
		}
		nv, err := ToTFValue(v, kt, th, eap)
		if err != nil {
			return tftypes.Value{}, eap.NewErrorf("[%s] cannot convert map element value: %s", eap, err)
		}
		im[k] = nv
		// Preserve DynamicPseudoType from schema, otherwise use actual type
		// to allow for tuple expansion
		if kt.Is(tftypes.DynamicPseudoType) {
			oTypes[k] = tftypes.DynamicPseudoType
		} else {
			oTypes[k] = nv.Type()
		}
	}
	return tftypes.NewValue(tftypes.Object{AttributeTypes: oTypes}, im), nil
}

func mapToTFDynamicValue(in map[string]interface{}, st tftypes.Type, th map[string]string, at *tftypes.AttributePath) (tftypes.Value, error) {
	im := make(map[string]tftypes.Value)
	oTypes := make(map[string]tftypes.Type)
	for k, v := range in {
		eap := at.WithAttributeName(k)
		nv, err := ToTFValue(v, tftypes.DynamicPseudoType, th, eap)
		if err != nil {
			return tftypes.Value{}, eap.NewErrorf("[%s] cannot convert map element value: %s", eap, err)
		}
		im[k] = nv
		oTypes[k] = nv.Type()
	}
	return tftypes.NewValue(tftypes.Object{AttributeTypes: oTypes}, im), nil
}
