package payload

import (
	"math/big"
	"reflect"
	"strconv"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// ToTFValue converts a Kubernetes dynamic client unstructured object
// into a Terraform specific tftypes.Value type object
func ToTFValue(in interface{}, st tftypes.Type, at *tftypes.AttributePath) (tftypes.Value, error) {
	if st == nil {
		return tftypes.Value{}, at.NewErrorf("[%s] type cannot be nil", at.String())
	}
	if in == nil {
		return tftypes.NewValue(st, nil), nil
	}
	switch in.(type) {
	case string:
		switch {
		case st.Is(tftypes.String) || st.Is(tftypes.DynamicPseudoType):
			return tftypes.NewValue(tftypes.String, in.(string)), nil
		case st.Is(tftypes.Number):
			num, err := strconv.Atoi(in.(string))
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
			return tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(int64(in.(int)))), nil
		default:
			return tftypes.Value{}, at.NewErrorf(`[%s] cannot convert payload from "int" to "%s"`, at.String(), st.String())
		}
	case int64:
		switch {
		case st.Is(tftypes.Number) || st.Is(tftypes.DynamicPseudoType):
			return tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(in.(int64))), nil
		default:
			return tftypes.Value{}, at.NewErrorf(`[%s] cannot convert payload from "in64" to "%s"`, at.String(), st.String())
		}
	case int32:
		switch {
		case st.Is(tftypes.Number) || st.Is(tftypes.DynamicPseudoType):
			return tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(int64(in.(int32)))), nil
		default:
			return tftypes.Value{}, at.NewErrorf(`[%s] cannot convert payload from "int32" to "%s"`, at.String(), st.String())
		}
	case int16:
		switch {
		case st.Is(tftypes.Number) || st.Is(tftypes.DynamicPseudoType):
			return tftypes.NewValue(tftypes.Number, new(big.Float).SetInt64(int64(in.(int16)))), nil
		default:
			return tftypes.Value{}, at.NewErrorf(`[%s] cannot convert payload from "int32" to "%s"`, at.String(), st.String())
		}
	case float64:
		switch {
		case st.Is(tftypes.Number) || st.Is(tftypes.DynamicPseudoType):
			return tftypes.NewValue(tftypes.Number, new(big.Float).SetFloat64(in.(float64))), nil
		default:
			return tftypes.Value{}, at.NewErrorf(`[%s] cannot convert payload from "float64" to "%s"`, at.String(), st.String())
		}
	case []interface{}:
		switch {
		case st.Is(tftypes.List{}):
			return sliceToTFListValue(in.([]interface{}), st, at)
		case st.Is(tftypes.Tuple{}):
			return sliceToTFTupleValue(in.([]interface{}), st, at)
		case st.Is(tftypes.Set{}):
			return sliceToTFSetValue(in.([]interface{}), st, at)
		case st.Is(tftypes.DynamicPseudoType):
			return sliceToTFDynamicValue(in.([]interface{}), st, at)
		default:
			return tftypes.Value{}, at.NewErrorf(`[%s] cannot convert payload from "[]interface{}" to "%s"`, at.String(), st.String())
		}
	case map[string]interface{}:
		switch {
		case st.Is(tftypes.Object{}):
			return mapToTFObjectValue(in.(map[string]interface{}), st, at)
		case st.Is(tftypes.Map{}):
			return mapToTFMapValue(in.(map[string]interface{}), st, at)
		case st.Is(tftypes.DynamicPseudoType):
			return mapToTFDynamicValue(in.(map[string]interface{}), st, at)
		default:
			return tftypes.Value{}, at.NewErrorf(`[%s] cannot convert payload from "map[string]interface{}" to "%s"`, at.String(), st.String())
		}
	}
	return tftypes.Value{}, at.NewErrorf(`[%s] cannot convert payload of unknown type "%s"`, at.String(), reflect.TypeOf(in).String())
}

func sliceToTFDynamicValue(in []interface{}, st tftypes.Type, at *tftypes.AttributePath) (tftypes.Value, error) {
	il := make([]tftypes.Value, len(in), len(in))
	oTypes := make([]tftypes.Type, len(in), len(in))
	for k, v := range in {
		eap := at.WithElementKeyInt(int64(k))
		var iv tftypes.Value
		iv, err := ToTFValue(v, tftypes.DynamicPseudoType, at.WithElementKeyInt(int64(k)))
		if err != nil {
			return tftypes.Value{}, eap.NewErrorf("[%s] cannot convert list element '%s' as DynamicPseudoType", eap, err)
		}
		il[k] = iv
		oTypes[k] = iv.Type()
	}
	return tftypes.NewValue(tftypes.Tuple{ElementTypes: oTypes}, il), nil
}

func sliceToTFListValue(in []interface{}, st tftypes.Type, at *tftypes.AttributePath) (tftypes.Value, error) {
	il := make([]tftypes.Value, 0, len(in))
	var oType tftypes.Type = tftypes.Type(nil)
	for k, v := range in {
		eap := at.WithElementKeyInt(int64(k))
		iv, err := ToTFValue(v, st.(tftypes.List).ElementType, at.WithElementKeyInt(int64(k)))
		if err != nil {
			return tftypes.Value{}, eap.NewErrorf("[%s] cannot convert list element value: %s", eap, err)
		}
		il = append(il, iv)
		if oType == tftypes.Type(nil) {
			oType = iv.Type()
		}
		if !oType.Is(iv.Type()) {
			return tftypes.Value{}, eap.NewErrorf("[%s] conflicting list element type: %s", eap.String(), iv.Type())
		}
	}
	// fallback for empty list, just use the requested type
	if oType == tftypes.Type(nil) {
		oType = st.(tftypes.List).ElementType
	}
	return tftypes.NewValue(tftypes.List{ElementType: oType}, il), nil
}

func sliceToTFTupleValue(in []interface{}, st tftypes.Type, at *tftypes.AttributePath) (tftypes.Value, error) {
	il := make([]tftypes.Value, len(in), len(in))
	oTypes := make([]tftypes.Type, len(in), len(in))
	ttypes := st.(tftypes.Tuple).ElementTypes
	if len(ttypes) == 1 && len(il) > 1 {
		ttypes = make([]tftypes.Type, len(in), len(in))
		for i := range il {
			ttypes[i] = st.(tftypes.Tuple).ElementTypes[0]
		}
	}
	for k, v := range in {
		eap := at.WithElementKeyInt(int64(k))
		et := ttypes[k]
		iv, err := ToTFValue(v, et, at.WithElementKeyInt(int64(k)))
		if err != nil {
			return tftypes.Value{}, eap.NewErrorf("[%s] cannot convert list element [%d] to '%s': %s", eap.String(), k, et.String(), err)
		}
		il[k] = iv
		oTypes[k] = iv.Type()
	}
	return tftypes.NewValue(tftypes.Tuple{ElementTypes: oTypes}, il), nil
}

func sliceToTFSetValue(in []interface{}, st tftypes.Type, at *tftypes.AttributePath) (tftypes.Value, error) {
	il := make([]tftypes.Value, len(in), len(in))
	var oType tftypes.Type = tftypes.Type(nil)
	for k, v := range in {
		eap := at.WithElementKeyInt(int64(k))
		iv, err := ToTFValue(v, st.(tftypes.Set).ElementType, at.WithElementKeyInt(int64(k)))
		if err != nil {
			return tftypes.Value{}, eap.NewErrorf("[%s] cannot convert list element [%d] to '%s': %s", eap, k, st.(tftypes.Set).ElementType.String(), err)
		}
		il[k] = iv
		if oType == tftypes.Type(nil) {
			oType = iv.Type()
		}
		if !oType.Is(iv.Type()) {
			return tftypes.Value{}, eap.NewErrorf("[%s] conflicting list element type: %s", eap.String(), iv.Type())
		}
	}
	// fallback for empty list, just use the requested type
	if oType == tftypes.Type(nil) {
		oType = st.(tftypes.Set).ElementType
	}
	return tftypes.NewValue(tftypes.Set{ElementType: oType}, il), nil
}

func mapToTFMapValue(in map[string]interface{}, st tftypes.Type, at *tftypes.AttributePath) (tftypes.Value, error) {
	im := make(map[string]tftypes.Value)
	var oType tftypes.Type
	for k, v := range in {
		eap := at.WithAttributeName(k)
		mv, err := ToTFValue(v, st.(tftypes.Map).AttributeType, eap)
		if err != nil {
			return tftypes.Value{}, eap.NewErrorf("[%s] cannot convert map element '%s' to '%s': err", eap, st.(tftypes.Map).AttributeType.String(), err)
		}
		im[k] = mv
		if oType == tftypes.Type(nil) {
			oType = mv.Type()
		}
		if !oType.Is(im[k].Type()) {
			return tftypes.Value{}, eap.NewErrorf("[%s] conflicting map element type: %s", eap.String(), mv.Type())
		}
	}
	if oType == tftypes.Type(nil) {
		oType = st.(tftypes.Map).AttributeType
	}
	return tftypes.NewValue(tftypes.Map{AttributeType: oType}, im), nil
}

func mapToTFObjectValue(in map[string]interface{}, st tftypes.Type, at *tftypes.AttributePath) (tftypes.Value, error) {
	im := make(map[string]tftypes.Value)
	oTypes := make(map[string]tftypes.Type)
	for k, kt := range st.(tftypes.Object).AttributeTypes {
		eap := at.WithAttributeName(k)
		v, ok := in[k]
		if !ok {
			v = nil
		}
		nv, err := ToTFValue(v, kt, eap)
		if err != nil {
			return tftypes.Value{}, eap.NewErrorf("[%s] cannot convert map element value: %s", eap, err)
		}
		im[k] = nv
		oTypes[k] = nv.Type()
	}
	return tftypes.NewValue(tftypes.Object{AttributeTypes: oTypes}, im), nil
}

func mapToTFDynamicValue(in map[string]interface{}, st tftypes.Type, at *tftypes.AttributePath) (tftypes.Value, error) {
	im := make(map[string]tftypes.Value)
	oTypes := make(map[string]tftypes.Type)
	for k, v := range in {
		eap := at.WithAttributeName(k)
		nv, err := ToTFValue(v, tftypes.DynamicPseudoType, eap)
		if err != nil {
			return tftypes.Value{}, eap.NewErrorf("[%s] cannot convert map element value: %s", eap, err)
		}
		im[k] = nv
		oTypes[k] = nv.Type()
	}
	return tftypes.NewValue(tftypes.Object{AttributeTypes: oTypes}, im), nil
}
