package corev1

import (
	"encoding/base64"

	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func base64EncodeStringMap(m map[string]basetypes.StringValue) map[string]basetypes.StringValue {
	result := make(map[string]basetypes.StringValue)
	for k, v := range m {
		value := v.ValueString()
		output := base64.StdEncoding.EncodeToString([]byte(value))
		result[k] = basetypes.NewStringValue(output)
	}
	return result
}

func base64DecodeStringMap(m map[string]basetypes.StringValue) map[string]basetypes.StringValue {
	result := make(map[string]basetypes.StringValue)
	for k, v := range m {
		value := v.ValueString()
		output, _ := base64.StdEncoding.DecodeString(value)
		result[k] = basetypes.NewStringValue(string(output))
	}
	return result
}
