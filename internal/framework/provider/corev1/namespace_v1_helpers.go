// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	corev1api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// expandNamespace converts the model to a Kubernetes Namespace object.
func expandNamespace(model NamespaceV1Model) *corev1api.Namespace {
	return &corev1api.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:         model.Metadata.Name.ValueString(),
			GenerateName: model.Metadata.GenerateName.ValueString(),
			Annotations:  expandStringMap(model.Metadata.Annotations),
			Labels:       expandStringMap(model.Metadata.Labels),
		},
	}
}

// flattenNamespaceMetadata converts a Kubernetes ObjectMeta to the Framework model,
// filtering out internal and ignored keys.
func flattenNamespaceMetadata(meta metav1.ObjectMeta, current NamespaceMetadataModel, ignoreAnnotations, ignoreLabels []string) NamespaceMetadataModel {
	result := NamespaceMetadataModel{
		Name:            types.StringValue(meta.Name),
		Generation:      types.Int64Value(meta.Generation),
		ResourceVersion: types.StringValue(meta.ResourceVersion),
		UID:             types.StringValue(string(meta.UID)),
	}

	if meta.GenerateName != "" {
		result.GenerateName = types.StringValue(meta.GenerateName)
	} else {
		result.GenerateName = types.StringNull()
	}

	filteredAnnotations := filterIgnoredMetadataKeys(meta.Annotations, current.Annotations, ignoreAnnotations)
	if len(filteredAnnotations) > 0 {
		result.Annotations = flattenStringMap(filteredAnnotations)
	}

	filteredLabels := filterIgnoredMetadataKeys(meta.Labels, current.Labels, ignoreLabels)
	if len(filteredLabels) > 0 {
		result.Labels = flattenStringMap(filteredLabels)
	}

	return result
}

// filterIgnoredMetadataKeys removes internal Kubernetes keys and keys matching
// the ignore patterns from a metadata map. Only removes a key if it is not already
// present in the current (Terraform-managed) map.
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

// diffMetadataPatch builds a JSON Patch payload for annotation/label changes.
func diffMetadataPatch(old, new NamespaceMetadataModel) ([]byte, error) {
	ops := make(kubernetes.PatchOperations, 0)
	ops = append(ops, kubernetes.DiffStringMap("/metadata/annotations",
		flattenToStringInterfaceMap(old.Annotations),
		flattenToStringInterfaceMap(new.Annotations))...)
	ops = append(ops, kubernetes.DiffStringMap("/metadata/labels",
		flattenToStringInterfaceMap(old.Labels),
		flattenToStringInterfaceMap(new.Labels))...)
	return json.Marshal(ops)
}

// flattenToStringInterfaceMap converts map[string]types.String → map[string]interface{}
// as required by DiffStringMap.
func flattenToStringInterfaceMap(m map[string]types.String) map[string]interface{} {
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		if !v.IsNull() && !v.IsUnknown() {
			result[k] = v.ValueString()
		}
	}
	return result
}

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
