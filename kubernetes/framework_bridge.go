// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// This file is the seam between the SDKv2 package and the Plugin Framework
// resources in internal/framework/provider.
//
// Only PURE policy is exported: functions over Kubernetes API types and plain
// Go maps, with no *schema.ResourceData and no Terraform plumbing. The metadata
// ownership policy in particular (rule K8S-CRUD-006) lives entirely in this
// package behind unexported helpers, and a Framework resource that cannot reach
// it would silently delete controller-owned labels and annotations on update.
//
// Deliberately NOT exported: patchMetadata and friends. Those are
// d.HasChange-driven and have no meaning under the Framework, which diffs plan
// against state itself. DiffStringMap below is the pure half they are built on.

// IgnoredMetadataKeys returns the provider-level ignore_annotations and
// ignore_labels settings from the SDKv2 provider meta. The Framework resources
// receive that meta through the mux (rule K8S-CRUD-001).
func IgnoredMetadataKeys(meta interface{}) (ignoreAnnotations, ignoreLabels []string) {
	pm, ok := meta.(providerMetadata)
	if !ok {
		return nil, nil
	}
	return pm.IgnoreAnnotations, pm.IgnoreLabels
}

// FilterMetadataKeys applies the provider's metadata-ownership policy in place,
// dropping internal and provider-ignored annotation/label keys that the
// practitioner did not configure themselves (rule K8S-CRUD-006). It is the
// filtering half of flattenMetadata, without the *schema.ResourceData.
//
// configuredAnnotations/configuredLabels are the keys the practitioner owns. On
// Create and Update those come from the PLAN; on refresh they come from STATE.
// Passing the wrong one silently changes which keys survive, so the caller must
// choose deliberately.
func FilterMetadataKeys(meta *metav1.ObjectMeta, configuredAnnotations, configuredLabels map[string]interface{}, ignoreAnnotations, ignoreLabels []string) {
	removeInternalKeys(meta.Annotations, configuredAnnotations)
	removeKeys(meta.Annotations, configuredAnnotations, ignoreAnnotations)
	removeInternalKeys(meta.Labels, configuredLabels)
	removeKeys(meta.Labels, configuredLabels, ignoreLabels)
}

// DiffStringMap produces the JSON-Patch operations that turn oldMap into
// newMap at pathPrefix, touching only the keys that actually changed.
//
// This is what makes a Framework Update satisfy K8S-CRUD-006 by construction:
// keys Terraform does not manage are never mentioned in the patch, so a
// controller-owned or provider-ignored annotation survives an update that
// changes a managed one, exactly as it does under the SDKv2 implementation.
func DiffStringMap(pathPrefix string, oldMap, newMap map[string]interface{}) PatchOperations {
	return diffStringMap(pathPrefix, oldMap, newMap)
}
