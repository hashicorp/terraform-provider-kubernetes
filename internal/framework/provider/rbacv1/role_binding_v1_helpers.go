// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	rbacv1api "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ── String map helpers ────────────────────────────────────────────────────────

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

// toStringInterfaceMap converts map[string]types.String → map[string]interface{}
// as required by kubernetes.DiffStringMap.
func toStringInterfaceMap(m map[string]types.String) map[string]interface{} {
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		if !v.IsNull() && !v.IsUnknown() {
			result[k] = v.ValueString()
		}
	}
	return result
}

// ── Metadata helpers ──────────────────────────────────────────────────────────

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

// flattenNamespacedMetadata converts a Kubernetes ObjectMeta to NamespacedMetadataModel,
// filtering out internal Kubernetes keys and user-configured ignore patterns.
// current holds the existing Terraform-managed metadata (used to preserve user-managed keys).
func flattenNamespacedMetadata(meta metav1.ObjectMeta, current NamespacedMetadataModel, ignoreAnnotations, ignoreLabels []string) NamespacedMetadataModel {
	result := NamespacedMetadataModel{
		Name:            types.StringValue(meta.Name),
		Namespace:       types.StringValue(meta.Namespace),
		Generation:      types.Int64Value(meta.Generation),
		ResourceVersion: types.StringValue(meta.ResourceVersion),
		UID:             types.StringValue(string(meta.UID)),
	}

	// generate_name: only set if non-empty to avoid perpetual diff vs nil
	if meta.GenerateName != "" {
		result.GenerateName = types.StringValue(meta.GenerateName)
	} else {
		result.GenerateName = types.StringNull()
	}

	filtered := filterIgnoredMetadataKeys(meta.Annotations, current.Annotations, ignoreAnnotations)
	// Preserve non-null empty map when the user explicitly configured annotations = {}.
	// Without this, annotations={} in config would flatten to null and cause an
	// inconsistent-result error on the next plan.
	if len(filtered) > 0 {
		result.Annotations = flattenStringMap(filtered)
	} else if current.Annotations != nil {
		result.Annotations = map[string]types.String{}
	}

	filtered = filterIgnoredMetadataKeys(meta.Labels, current.Labels, ignoreLabels)
	if len(filtered) > 0 {
		result.Labels = flattenStringMap(filtered)
	} else if current.Labels != nil {
		result.Labels = map[string]types.String{}
	}

	return result
}

// ── RoleRef helpers ───────────────────────────────────────────────────────────

// expandRoleRef converts a RoleRefModel to a Kubernetes RoleRef API object.
func expandRoleRef(m RoleRefModel) rbacv1api.RoleRef {
	return rbacv1api.RoleRef{
		APIGroup: m.APIGroup.ValueString(),
		Kind:     m.Kind.ValueString(),
		Name:     m.Name.ValueString(),
	}
}

// flattenRoleRef converts a Kubernetes RoleRef API object to a RoleRefModel.
func flattenRoleRef(in rbacv1api.RoleRef) RoleRefModel {
	return RoleRefModel{
		APIGroup: types.StringValue(in.APIGroup),
		Kind:     types.StringValue(in.Kind),
		Name:     types.StringValue(in.Name),
	}
}

// ── Subject helpers ───────────────────────────────────────────────────────────

// expandSubjects converts a slice of SubjectModel to Kubernetes Subject API objects.
func expandSubjects(in []SubjectModel) []rbacv1api.Subject {
	subjects := make([]rbacv1api.Subject, 0, len(in))
	for _, s := range in {
		subject := rbacv1api.Subject{
			Kind:      s.Kind.ValueString(),
			Name:      s.Name.ValueString(),
			APIGroup:  s.APIGroup.ValueString(),
			Namespace: s.Namespace.ValueString(),
		}
		subjects = append(subjects, subject)
	}
	return subjects
}

// flattenSubjects converts Kubernetes Subject API objects to a slice of SubjectModel.
func flattenSubjects(in []rbacv1api.Subject) []SubjectModel {
	result := make([]SubjectModel, 0, len(in))
	for _, s := range in {
		m := SubjectModel{
			Kind:      types.StringValue(s.Kind),
			Name:      types.StringValue(s.Name),
			APIGroup:  types.StringValue(s.APIGroup),
			Namespace: types.StringValue(s.Namespace),
		}
		result = append(result, m)
	}
	return result
}

// ── Subject patch helper ──────────────────────────────────────────────────────

// patchSubjects generates the minimal JSON Patch operations to reconcile the
// subject list from old → new. Uses index-based Replace/Add/Remove operations
// to match the SDKv2 patchRbacSubject behaviour exactly.
func patchSubjects(old, new []SubjectModel) kubernetes.PatchOperations {
	oldExpanded := expandSubjects(old)
	newExpanded := expandSubjects(new)

	ops := make(kubernetes.PatchOperations, 0, len(newExpanded)+len(oldExpanded))

	common := len(newExpanded)
	if common > len(oldExpanded) {
		common = len(oldExpanded)
	}

	// Remove trailing old entries first (reverse order to keep indices stable)
	if len(oldExpanded) > len(newExpanded) {
		for i := len(newExpanded); i < len(oldExpanded); i++ {
			ops = append(ops, &kubernetes.RemoveOperation{
				Path: "/subjects/" + strconv.Itoa(len(oldExpanded)-i),
			})
		}
	}

	// Replace entries that exist in both old and new
	for i, v := range newExpanded[:common] {
		ops = append(ops, &kubernetes.ReplaceOperation{
			Path:  "/subjects/" + strconv.Itoa(i),
			Value: v,
		})
	}

	// Add new entries beyond the old length
	if len(newExpanded) > len(oldExpanded) {
		for i, v := range newExpanded[common:] {
			ops = append(ops, &kubernetes.AddOperation{
				Path:  "/subjects/" + strconv.Itoa(common+i),
				Value: v,
			})
		}
	}

	return ops
}
