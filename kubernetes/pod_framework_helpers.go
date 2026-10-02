// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	corev1 "k8s.io/api/core/v1"
)

// ExpandPodSpecForFramework exposes only the pure API conversion shared with the
// SDKv2 Pod. Collection sets are represented as slices at the Framework boundary.
func ExpandPodSpecForFramework(spec []interface{}) (*corev1.PodSpec, error) {
	return expandPodSpec(podFrameworkExpandCollections(spec).([]interface{}))
}

// FlattenPodSpecForFramework retains the SDKv2 Pod's service-account volume and
// mount filtering, built-in toleration filtering, and admission-populated values.
func FlattenPodSpecForFramework(spec corev1.PodSpec) ([]interface{}, error) {
	// The shared flattener removes a token volume in-place from its slice.
	values, err := flattenPodSpec(*spec.DeepCopy(), false)
	if err != nil {
		return nil, err
	}
	return podFrameworkFlattenCollections(values).([]interface{}), nil
}

func podFrameworkExpandCollections(value interface{}) interface{} {
	switch v := value.(type) {
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, entry := range v {
			result[i] = podFrameworkExpandCollections(entry)
		}
		return result
	case map[string]interface{}:
		result := make(map[string]interface{}, len(v))
		for key, entry := range v {
			converted := podFrameworkExpandCollections(entry)
			if items, collection := converted.([]interface{}); collection {
				switch key {
				case "supplemental_groups":
					converted = schema.NewSet(schema.HashInt, items)
				case "values", "namespaces", "match_label_keys", "monitors", "target_ww_ns", "ceph_monitors", "access_modes":
					converted = schema.NewSet(schema.HashString, items)
				}
			}
			result[key] = converted
		}
		return result
	default:
		return value
	}
}

func podFrameworkFlattenCollections(value interface{}) interface{} {
	switch v := value.(type) {
	case *schema.Set:
		return podFrameworkFlattenCollections(v.List())
	case []map[string]interface{}:
		result := make([]interface{}, len(v))
		for i, entry := range v {
			result[i] = podFrameworkFlattenCollections(entry)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, entry := range v {
			result[i] = podFrameworkFlattenCollections(entry)
		}
		return result
	case map[string]interface{}:
		result := make(map[string]interface{}, len(v))
		for key, entry := range v {
			result[key] = podFrameworkFlattenCollections(entry)
		}
		return result
	default:
		return value
	}
}
