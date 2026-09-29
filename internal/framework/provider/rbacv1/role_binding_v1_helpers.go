// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	rbacv1api "k8s.io/api/rbac/v1"
)

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

	commonLen := len(newExpanded)
	if commonLen > len(oldExpanded) {
		commonLen = len(oldExpanded)
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
	for i, v := range newExpanded[:commonLen] {
		ops = append(ops, &kubernetes.ReplaceOperation{
			Path:  "/subjects/" + strconv.Itoa(i),
			Value: v,
		})
	}

	// Add new entries beyond the old length
	if len(newExpanded) > len(oldExpanded) {
		for i, v := range newExpanded[commonLen:] {
			ops = append(ops, &kubernetes.AddOperation{
				Path:  "/subjects/" + strconv.Itoa(commonLen+i),
				Value: v,
			})
		}
	}

	return ops
}
