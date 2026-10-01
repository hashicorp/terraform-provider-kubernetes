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
// When a subject has no namespace (e.g. User or Group kinds), the Kubernetes API
// returns an empty string. The schema declares Default: "default" for namespace, which
// Terraform applies during planning but not during Read. To keep state consistent with
// the plan default — and to match SDKv2 behaviour (Default: "default" in schema_rbac.go)
// — we write "default" whenever the API returns an empty namespace.
func flattenSubjects(in []rbacv1api.Subject) []SubjectModel {
	result := make([]SubjectModel, 0, len(in))
	for _, s := range in {
		ns := s.Namespace
		if ns == "" {
			ns = "default"
		}
		m := SubjectModel{
			Kind:      types.StringValue(s.Kind),
			Name:      types.StringValue(s.Name),
			APIGroup:  types.StringValue(s.APIGroup),
			Namespace: types.StringValue(ns),
		}
		result = append(result, m)
	}
	return result
}

// applySubjectComputedFields resolves unknown computed fields in plan subjects from the
// Kubernetes API response, while preserving all configured values.
//
// subject.api_group is Optional+Computed with no default. When a subject is configured
// without api_group (e.g. a ServiceAccount), the plan carries an unknown value for that
// field. After the API call succeeds, Kubernetes returns a concrete value. This helper
// fills in only those unknown slots; configured (known) values are left untouched.
//
// If the response has a different number of subjects than the plan — which should not
// happen in practice but is possible if the server normalised the request — we fall back
// to a full flatten for the mismatched tail so the state is still consistent.
func applySubjectComputedFields(plan *[]SubjectModel, apiSubjects []rbacv1api.Subject) {
	for i := range *plan {
		if i >= len(apiSubjects) {
			break
		}
		s := &(*plan)[i]
		api := apiSubjects[i]

		// Resolve api_group only when the plan left it unknown (omitted by the caller).
		if s.APIGroup.IsUnknown() {
			s.APIGroup = types.StringValue(api.APIGroup)
		}
		// Resolve namespace only when the plan left it unknown.
		if s.Namespace.IsUnknown() {
			ns := api.Namespace
			if ns == "" {
				ns = "default"
			}
			s.Namespace = types.StringValue(ns)
		}
	}

	// Append any extra API subjects that have no matching plan entry.
	for i := len(*plan); i < len(apiSubjects); i++ {
		api := apiSubjects[i]
		ns := api.Namespace
		if ns == "" {
			ns = "default"
		}
		*plan = append(*plan, SubjectModel{
			Kind:      types.StringValue(api.Kind),
			Name:      types.StringValue(api.Name),
			APIGroup:  types.StringValue(api.APIGroup),
			Namespace: types.StringValue(ns),
		})
	}
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
