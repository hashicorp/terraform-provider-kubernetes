package autocrud

import (
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ExpandModel takes a framework Model struct and converts it
// to a map compatible with kubernetes unstructured.Object
func ExpandModel(model any) map[string]any {
	return expand(model)
}

func expandMap(v any) map[string]any {
	val := reflect.ValueOf(v)
	m := map[string]any{}
	for _, k := range val.MapKeys() {
		m[k.String()] = expandValue(val.MapIndex(k))
	}
	return m
}

func expandSlice(v any) any {
	val := reflect.ValueOf(v)
	l := make([]any, val.Len())
	for i := 0; i < val.Len(); i++ {
		l[i] = expandValue(val.Index(i))
	}
	return l
}

func expandValue(field reflect.Value) any {
	v := field.Interface()
	switch field.Type().String() {
	case "basetypes.BoolValue":
		if val, ok := v.(types.Bool); ok && !val.IsNull() && !val.IsUnknown() {
			return val.ValueBool()
		}
	case "basetypes.StringValue":
		if val, ok := v.(types.String); ok && !val.IsNull() && !val.IsUnknown() {
			return val.ValueString()
		}
	case "basetypes.NumberValue":
		if val, ok := v.(types.Number); ok && !val.IsNull() && !val.IsUnknown() {
			vv := val.ValueBigFloat()
			if vv.IsInt() {
				intVal, _ := vv.Int64()
				return intVal
			}
			// TODO handle float64
		}
	case "basetypes.Int64Value":
		if val, ok := v.(types.Int64); ok && !val.IsNull() && !val.IsUnknown() {
			return val.ValueInt64()
		}
	default:
		if field.Type().Kind() == reflect.Struct {
			return expand(field.Interface())
		}
		if field.Type().Kind() == reflect.Map {
			return expandMap(field.Interface())
		}
		if field.Type().Kind() == reflect.Slice {
			return expandSlice(field.Interface())
		}
	}
	return nil
}

func expand(model any) map[string]interface{} {
	val := reflect.ValueOf(model)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	t := val.Type()
	m := map[string]interface{}{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag
		manifestField := tag.Get("manifest")
		field := val.Field(i)
		m[manifestField] = expandValue(field)
	}
	return m
}
