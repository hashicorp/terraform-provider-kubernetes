// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"

	policy "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func expandPodDisruptionBudgetV1Spec(in []interface{}) (*policy.PodDisruptionBudgetSpec, error) {
	spec := &policy.PodDisruptionBudgetSpec{}
	if len(in) == 0 || in[0] == nil {
		return nil, fmt.Errorf("failed to expand PodDisruptionBudget.Spec: null or empty input")
	}
	m := in[0].(map[string]interface{})
	if v, ok := m["max_unavailable"].(string); ok && len(v) > 0 {
		val := intstr.Parse(v)
		spec.MaxUnavailable = &val
	}
	if v, ok := m["min_available"].(string); ok && len(v) > 0 {
		val := intstr.Parse(v)
		spec.MinAvailable = &val
	}
	if v, ok := m["selector"].([]interface{}); ok && len(v) > 0 {
		spec.Selector = expandLabelSelector(v)
	}

	return spec, nil
}

func flattenPodDisruptionBudgetV1Spec(spec policy.PodDisruptionBudgetSpec) []interface{} {
	m := make(map[string]interface{}, 0)
	if spec.MaxUnavailable != nil {
		m["max_unavailable"] = spec.MaxUnavailable.String()
	}
	if spec.MinAvailable != nil {
		m["min_available"] = spec.MinAvailable.String()
	}
	if spec.Selector != nil {
		m["selector"] = flattenLabelSelector(spec.Selector)
	}

	return []interface{}{m}
}
