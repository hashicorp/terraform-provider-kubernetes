package corev1

// func (r *Secret) BeforeCreate(m *SecretModel) {
// 	m.Data = base64EncodeStringMap(m.Data)
// }

// func (r *Secret) AfterCreate(m *SecretModel) {
// 	m.Data = base64DecodeStringMap(m.Data)
// }

// func (r *Namespace) AfterCreate(m *NamespaceModel) {
// 	panic("damn")
// }

// func base64EncodeStringMap(m map[string]basetypes.StringValue) map[string]basetypes.StringValue {
// 	result := make(map[string]basetypes.StringValue)
// 	for k, v := range m {
// 		value := v.ValueString()
// 		output := base64.StdEncoding.EncodeToString([]byte(value))
// 		result[k] = basetypes.NewStringValue(output)
// 	}
// 	return result
// }

// func base64DecodeStringMap(m map[string]basetypes.StringValue) map[string]basetypes.StringValue {
// 	result := make(map[string]basetypes.StringValue)
// 	for k, v := range m {
// 		value := v.ValueString()
// 		output, _ := base64.StdEncoding.DecodeString(value)
// 		result[k] = basetypes.NewStringValue(string(output))
// 	}
// 	return result
// }
