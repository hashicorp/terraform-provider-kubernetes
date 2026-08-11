// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package schedulingv1

import (
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// expandStringMap converts map[string]types.String → map[string]string for Kubernetes API calls.
func expandStringMap(m map[string]types.String) map[string]string {
	if m == nil {
		return nil
	}
	result := make(map[string]string, len(m))
	for k, v := range m {
		if !v.IsNull() && !v.IsUnknown() {
			result[k] = v.ValueString()
		}
	}
	return result
}

// flattenStringMap converts map[string]string → map[string]types.String.
func flattenStringMap(m map[string]string) map[string]types.String {
	if m == nil {
		return nil
	}
	result := make(map[string]types.String, len(m))
	for k, v := range m {
		result[k] = types.StringValue(v)
	}
	return result
}

// flattenPriorityClassMetadata converts a Kubernetes ObjectMeta to MetadataModel,
// filtering out internal Kubernetes keys and user-configured ignore patterns.
// current holds the existing Terraform-managed metadata (used to preserve user-managed keys).
func flattenPriorityClassMetadata(meta metav1.ObjectMeta, current MetadataModel, ignoreAnnotations, ignoreLabels []string) MetadataModel {
	result := MetadataModel{
		Name:            types.StringValue(meta.Name),
		Generation:      types.Int64Value(meta.Generation),
		ResourceVersion: types.StringValue(meta.ResourceVersion),
		UID:             types.StringValue(string(meta.UID)),
	}

	// generate_name: only set if non-empty to avoid diff vs nil
	if meta.GenerateName != "" {
		result.GenerateName = types.StringValue(meta.GenerateName)
	} else {
		result.GenerateName = types.StringNull()
	}

	filtered := filterIgnoredMetadataKeys(meta.Annotations, current.Annotations, ignoreAnnotations)
	if len(filtered) > 0 {
		result.Annotations = flattenStringMap(filtered)
	}

	filtered = filterIgnoredMetadataKeys(meta.Labels, current.Labels, ignoreLabels)
	if len(filtered) > 0 {
		result.Labels = flattenStringMap(filtered)
	}

	return result
}

// filterIgnoredMetadataKeys removes internal Kubernetes keys and keys matching
// ignore patterns — unless that key is already present in current (managed by TF).
func filterIgnoredMetadataKeys(meta map[string]string, current map[string]types.String, ignorePatterns []string) map[string]string {
	result := make(map[string]string, len(meta))
	for k, v := range meta {
		_, managedByTF := current[k]
		if !managedByTF && (kubernetes.IsInternalKey(k) || kubernetes.IgnoreKey(k, ignorePatterns)) {
			continue
		}
		result[k] = v
	}
	return result
}

// diffMetadataPatch builds a JSON Patch payload for annotation and label changes.
// Used by Update() to send only the changed keys rather than a full PUT.
func diffMetadataPatch(old, new MetadataModel) ([]byte, error) {
	ops := make(kubernetes.PatchOperations, 0)
	ops = append(ops, kubernetes.DiffStringMap(
		"/metadata/annotations",
		toStringInterfaceMap(old.Annotations),
		toStringInterfaceMap(new.Annotations),
	)...)
	ops = append(ops, kubernetes.DiffStringMap(
		"/metadata/labels",
		toStringInterfaceMap(old.Labels),
		toStringInterfaceMap(new.Labels),
	)...)
	return json.Marshal(ops)
}

// toStringInterfaceMap converts map[string]types.String → map[string]interface{}
// as required by DiffStringMap.
func toStringInterfaceMap(m map[string]types.String) map[string]interface{} {
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		if !v.IsNull() && !v.IsUnknown() {
			result[k] = v.ValueString()
		}
	}
	return result
}
