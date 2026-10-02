// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	corev1 "k8s.io/api/core/v1"
)

// ExpandPodSpecForFramework exposes the pure SDKv2 PodSpec expander. The caller
// passes set-typed fields as *schema.Set, as SDKv2 would.
func ExpandPodSpecForFramework(spec []interface{}) (*corev1.PodSpec, error) {
	return expandPodSpec(spec)
}

// FlattenPodSpecForFramework exposes the pure SDKv2 PodSpec flattener. A bare
// Pod (isTemplate false) drops the built-in tolerations Kubernetes adds; both
// drop the injected service-account token volume.
func FlattenPodSpecForFramework(spec corev1.PodSpec, isTemplate bool) ([]interface{}, error) {
	// The flattener removes the token volume from its slice in place.
	return flattenPodSpec(*spec.DeepCopy(), isTemplate)
}
