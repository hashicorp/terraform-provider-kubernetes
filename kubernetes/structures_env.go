// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

func expandEnv(e []interface{}) []map[string]interface{} {
	envs := []map[string]interface{}{}
	if len(e) == 0 {
		return envs
	}

	for _, c := range e {
		p, ok := c.(map[string]interface{})
		if !ok {
			continue
		}

		newEnv := make(map[string]interface{})
		if name, ok := p["name"]; ok {
			newEnv["name"] = name
		}
		if value, ok := p["value"]; ok {
			newEnv["value"] = value
		}
		if v, ok := p["value_from"].([]interface{}); ok && len(v) > 0 {
			newEnv["valueFrom"] = expandEnvValueFromMap(v[0])
		}
		envs = append(envs, newEnv)
	}

	return envs
}

func expandEnvValueFromMap(e interface{}) map[string]interface{} {
	if e == nil {
		return nil
	}

	in := e.(map[string]interface{})
	expandedValues := make(map[string]interface{})

	if v, ok := in["config_map_key_ref"].([]interface{}); ok && len(v) > 0 {
		expandedValues["configMapKeyRef"] = v[0]
	}
	if v, ok := in["field_ref"].([]interface{}); ok && len(v) > 0 {
		expandedValues["fieldRef"] = expandFieldRefMap(v[0])
	}
	if v, ok := in["resource_field_ref"].([]interface{}); ok && len(v) > 0 {
		expandedValues["resourceFieldRef"] = expandResourceFieldMap(v[0])
	}
	if v, ok := in["secret_key_ref"].([]interface{}); ok && len(v) > 0 {
		expandedValues["secretKeyRef"] = v[0]
	}

	return expandedValues
}

func expandFieldRefMap(e interface{}) map[string]interface{} {
	if e == nil {
		return nil
	}

	in := e.(map[string]interface{})
	expandedValues := make(map[string]interface{})

	if v, ok := in["api_version"].(interface{}); ok && v != nil {
		expandedValues["apiVersion"] = v
	}
	if v, ok := in["field_path"].([]interface{}); ok && v != nil {
		expandedValues["fieldPath"] = v
	}

	return expandedValues
}

func expandResourceFieldMap(e interface{}) map[string]interface{} {
	if e == nil {
		return nil
	}

	in := e.(map[string]interface{})
	expandedValues := make(map[string]interface{})

	if v, ok := in["container_name"].(interface{}); ok && v != nil {
		expandedValues["containerName"] = v
	}
	if v, ok := in["divisor"].([]interface{}); ok && v != nil {
		expandedValues["divisor"] = v
	}
	if v, ok := in["resource"].([]interface{}); ok && v != nil {
		expandedValues["resource"] = v
	}

	return expandedValues
}

func flattenEnv(e []interface{}) []interface{} {
	envs := []interface{}{}
	if len(e) == 0 {
		return envs
	}
	for _, c := range e {
		p, ok := c.(map[string]interface{})
		if !ok {
			continue
		}

		newEnv := make(map[string]interface{})
		if name, ok := p["name"]; ok {
			newEnv["name"] = name
		}
		if value, ok := p["value"]; ok && value != "" {
			newEnv["value"] = value
		}
		if v, ok := p["valueFrom"].(map[string]interface{}); ok && len(v) > 0 {
			newEnv["value_from"] = flattenEnvValueFromMap(v)
		}
		envs = append(envs, newEnv)
	}

	return envs
}

func flattenEnvValueFromMap(e interface{}) []interface{} {
	if e == nil {
		return nil
	}

	in := e.(map[string]interface{})
	expandedValues := make(map[string]interface{})

	if v, ok := in["configMapKeyRef"].(interface{}); ok && v != nil {
		expandedValues["config_map_key_ref"] = []interface{}{v}
	}
	if v, ok := in["fieldRef"].(interface{}); ok && v != nil {
		expandedValues["field_ref"] = flattenFieldRefMap(v)
	}
	if v, ok := in["resourceFieldRef"].(interface{}); ok && v != nil {
		expandedValues["resource_field_ref"] = flattenResourceFieldMap(v)
	}
	if v, ok := in["secretKeyRef"].(interface{}); ok && v != nil {
		expandedValues["secret_key_ref"] = []interface{}{v}
	}

	return []interface{}{expandedValues}
}

func flattenFieldRefMap(e interface{}) []interface{} {
	if e == nil {
		return nil
	}

	in := e.(map[string]interface{})
	expandedValues := make(map[string]interface{})

	if v, ok := in["apiVersion"].(interface{}); ok && v != nil {
		expandedValues["api_version"] = v
	}
	if v, ok := in["fieldPath"].(interface{}); ok && v != nil {
		expandedValues["field_path"] = v
	}

	return []interface{}{expandedValues}
}

func flattenResourceFieldMap(e interface{}) []interface{} {
	if e == nil {
		return nil
	}

	in := e.(map[string]interface{})
	expandedValues := make(map[string]interface{})

	if v, ok := in["containerName"].(interface{}); ok && v != nil {
		expandedValues["container_name"] = v
	}
	if v, ok := in["divisor"].(interface{}); ok && v != nil {
		expandedValues["divisor"] = v
	}
	if v, ok := in["resource"].(interface{}); ok && v != nil {
		expandedValues["resource"] = v
	}

	return []interface{}{expandedValues}
}
