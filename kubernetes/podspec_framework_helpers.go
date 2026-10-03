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

// FlattenPodSpecForFramework exposes the pure SDKv2 PodSpec flattener. It
// keeps every toleration, as for a pod template, and drops the injected
// service-account token volume.
func FlattenPodSpecForFramework(spec corev1.PodSpec) ([]interface{}, error) {
	// The flattener removes the token volume from its slice in place.
	return flattenPodSpec(*spec.DeepCopy(), true)
}

// IsBuiltInToleration reports whether key is a taint whose toleration
// Kubernetes can add to a Pod on its own.
func IsBuiltInToleration(key string) bool {
	_, ok := builtInTolerations[key]
	return ok
}
