package structures

import (
	"encoding/base64"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ExpandStringMap(m map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		result[k] = v.(string)
	}
	return result
}

func ExpandBase64MapToByteMap(m map[string]interface{}) map[string][]byte {
	result := make(map[string][]byte)
	for k, v := range m {
		b, err := base64.StdEncoding.DecodeString(v.(string))
		if err == nil {
			result[k] = b
		}
	}
	return result
}

func ExpandStringMapToByteMap(m map[string]interface{}) map[string][]byte {
	result := make(map[string][]byte)
	for k, v := range m {
		result[k] = []byte(v.(string))
	}
	return result
}

func ExpandStringSlice(s []interface{}) []string {
	result := make([]string, len(s), len(s))
	for k, v := range s {
		// Handle the Terraform parser bug which turns empty strings in lists to nil.
		if v == nil {
			result[k] = ""
		} else {
			result[k] = v.(string)
		}
	}
	return result
}

func FlattenByteMapToBase64Map(m map[string][]byte) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		result[k] = base64.StdEncoding.EncodeToString([]byte(v))
	}
	return result
}

func FlattenByteMapToStringMap(m map[string][]byte) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		result[k] = string(v)
	}
	return result
}

func PtrToString(s string) *string {
	return &s
}

func PtrToBool(b bool) *bool {
	return &b
}

func PtrToInt32(i int32) *int32 {
	return &i
}

func PtrToInt64(i int64) *int64 {
	return &i
}

func SliceOfString(slice []interface{}) []string {
	result := make([]string, len(slice), len(slice))
	for i, s := range slice {
		result[i] = s.(string)
	}
	return result
}

func Base64EncodeStringMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		value := v.(string)
		result[k] = base64.StdEncoding.EncodeToString([]byte(value))
	}
	return result
}

func Base64EncodeByteMap(m map[string][]byte) map[string]interface{} {
	result := map[string]interface{}{}
	for k, v := range m {
		result[k] = base64.StdEncoding.EncodeToString(v)
	}
	return result
}

func Base64DecodeStringMap(m map[string]interface{}) (map[string][]byte, error) {
	mm := map[string][]byte{}
	for k, v := range m {
		d, err := base64.StdEncoding.DecodeString(v.(string))
		if err != nil {
			return nil, err
		}
		mm[k] = []byte(d)
	}
	return mm, nil
}

func NewStringSet(f schema.SchemaSetFunc, in []string) *schema.Set {
	var out = make([]interface{}, len(in), len(in))
	for i, v := range in {
		out[i] = v
	}
	return schema.NewSet(f, out)
}
func NewInt64Set(f schema.SchemaSetFunc, in []int64) *schema.Set {
	var out = make([]interface{}, len(in), len(in))
	for i, v := range in {
		out[i] = int(v)
	}
	return schema.NewSet(f, out)
}

func SchemaSetToStringArray(set *schema.Set) []string {
	array := make([]string, 0, set.Len())
	for _, elem := range set.List() {
		e := elem.(string)
		array = append(array, e)
	}
	return array
}

func SchemaSetToInt64Array(set *schema.Set) []int64 {
	array := make([]int64, 0, set.Len())
	for _, elem := range set.List() {
		e := elem.(int)
		array = append(array, int64(e))
	}
	return array
}
