// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	corev1 "k8s.io/api/core/v1"
)

// ExpandTemplatePodSpecForFramework exposes the pure SDKv2 PodSpec expander for
// workload pod templates. Framework sets arrive as slices and are converted to
// the *schema.Set values the shared expander reads.
func ExpandTemplatePodSpecForFramework(spec []interface{}) (*corev1.PodSpec, error) {
	return expandPodSpec(templatePodSpecFrameworkSets(spec).([]interface{}))
}

// FlattenTemplatePodSpecForFramework exposes the pure SDKv2 PodSpec flattener
// with workload template semantics (built-in tolerations are retained), and
// returns sets as slices for the Framework boundary.
func FlattenTemplatePodSpecForFramework(spec corev1.PodSpec) ([]interface{}, error) {
	// The shared flattener removes a token volume in-place from its slice.
	values, err := flattenPodSpec(*spec.DeepCopy(), true)
	if err != nil {
		return nil, err
	}
	return templatePodSpecFrameworkSlices(values).([]interface{}), nil
}

func templatePodSpecFrameworkSets(value interface{}) interface{} {
	switch v := value.(type) {
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, entry := range v {
			result[i] = templatePodSpecFrameworkSets(entry)
		}
		return result
	case map[string]interface{}:
		result := make(map[string]interface{}, len(v))
		for key, entry := range v {
			converted := templatePodSpecFrameworkSets(entry)
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

func templatePodSpecFrameworkSlices(value interface{}) interface{} {
	switch v := value.(type) {
	case *schema.Set:
		return templatePodSpecFrameworkSlices(v.List())
	case []map[string]interface{}:
		result := make([]interface{}, len(v))
		for i, entry := range v {
			result[i] = templatePodSpecFrameworkSlices(entry)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, entry := range v {
			result[i] = templatePodSpecFrameworkSlices(entry)
		}
		return result
	case map[string]interface{}:
		result := make(map[string]interface{}, len(v))
		for key, entry := range v {
			result[key] = templatePodSpecFrameworkSlices(entry)
		}
		return result
	default:
		return value
	}
}
