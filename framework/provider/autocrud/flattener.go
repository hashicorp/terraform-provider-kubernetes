package autocrud

import (
	"fmt"
	"math/big"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// FlattenModel takes a Kubernetes unstructured object and flattens it
// into a Terraform Model
func FlattenManifest(manifest map[string]any, model any) error {
	return flatten(manifest, model)
}

func flattenMap(v any, model any) reflect.Value {
	keyType := reflect.TypeOf(model).Key()
	elemType := reflect.TypeOf(model).Elem()
	mapType := reflect.MapOf(keyType, elemType)
	m := reflect.MakeMap(mapType)
	for k, v := range v.(map[string]any) {
		m.SetMapIndex(reflect.ValueOf(k), flattenValue(reflect.New(elemType).Elem(), v))
	}
	return m
}

func flattenSlice(v any, model any) reflect.Value {
	elemType := reflect.TypeOf(model).Elem()
	sliceType := reflect.SliceOf(elemType)
	sliceVal := v.([]any)
	s := reflect.MakeSlice(sliceType, len(sliceVal), len(sliceVal))
	for k := range sliceVal {
		s.Index(k).Set(flattenValue(reflect.New(elemType).Elem(), sliceVal[k]))
	}
	return s
}

func flattenValue(field reflect.Value, v any) reflect.Value {
	switch field.Type().String() {
	case "basetypes.BoolValue":
		bv := types.BoolValue(v.(bool))
		return reflect.ValueOf(bv)
	case "basetypes.StringValue":
		sv := types.StringValue(v.(string))
		return reflect.ValueOf(sv)
	case "basetypes.NumberValue":
		var bf *big.Float
		switch vv := v.(type) {
		case float32:
		case float64:
			bf = big.NewFloat(float64(vv))
		}
		nv := types.NumberValue(bf)
		return reflect.ValueOf(nv)
	case "basetypes.Int64Value":
		sv := types.Int64Value(v.(int64))
		return reflect.ValueOf(sv)
	default:
		if field.Kind() == reflect.Struct {
			flatten(v.(map[string]any), field.Addr().Interface())
			return field
		}
		if field.Type().Kind() == reflect.Map {
			return flattenMap(v, field.Interface())
		}
		if field.Type().Kind() == reflect.Slice {
			return flattenSlice(v, field.Interface())
		}
	}
	panic(fmt.Sprintf("unsupported value: %v %v", field.Type().Kind(), v))
}

func flatten(manifest map[string]any, model any) error {
	val := reflect.ValueOf(model)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	t := val.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag
		manifestField := tag.Get("manifest")
		field := val.Field(i)
		if v, ok := manifest[manifestField]; ok {
			field.Set(flattenValue(field, v))
		}
	}
	return nil
}
