// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	corev1 "k8s.io/api/core/v1"
	"maps"
)

// ExpandPodSpecForFramework exposes the pure SDKv2 PodSpec expander. The caller
// passes set-typed fields as *schema.Set, as SDKv2 would.
func ExpandPodSpecForFramework(spec []interface{}) (*corev1.PodSpec, error) {
	// Only the Framework public schema uses resource objects. Copy the affected
	// containers so adaptation cannot mutate a plan or a cached schema probe.
	adapted := make([]interface{}, len(spec))
	for i, entry := range spec {
		object, ok := entry.(map[string]interface{})
		if !ok {
			adapted[i] = entry
			continue
		}
		object = maps.Clone(object)
		for _, name := range []string{"container", "init_container"} {
			containers, ok := object[name].([]interface{})
			if !ok {
				continue
			}
			converted := make([]interface{}, len(containers))
			for j, entry := range containers {
				container, ok := entry.(map[string]interface{})
				if !ok {
					converted[j] = entry
					continue
				}
				container = maps.Clone(container)
				if resources, ok := container["resources"].(map[string]interface{}); ok {
					container["resources"] = []interface{}{resources}
				}
				converted[j] = container
			}
			object[name] = converted
		}
		adapted[i] = object
	}
	return expandPodSpec(adapted)
}

// FlattenPodSpecForFramework exposes the pure SDKv2 PodSpec flattener. It
// keeps every toleration, as for a pod template, and drops the injected
// service-account token volume.
func FlattenPodSpecForFramework(spec corev1.PodSpec) ([]interface{}, error) {
	// The flattener removes the token volume from its slice in place.
	flat, err := flattenPodSpec(*spec.DeepCopy(), true)
	if err != nil {
		return nil, err
	}
	for _, entry := range flat {
		object := entry.(map[string]interface{})
		for _, name := range []string{"container", "init_container"} {
			containers, _ := object[name].([]interface{})
			for _, entry := range containers {
				container := entry.(map[string]interface{})
				// The SDK flattener always returns one resource-requirements object.
				resources := container["resources"].([]interface{})
				container["resources"] = resources[0]
			}
		}
	}
	return flat, nil
}

// IsBuiltInToleration reports whether key is a taint whose toleration
// Kubernetes can add to a Pod on its own.
func IsBuiltInToleration(key string) bool {
	_, ok := builtInTolerations[key]
	return ok
}
