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
//
// Namespace is written exactly as the API returns it. The schema default ("default")
// is applied by Terraform at plan time when namespace is omitted, so the API then
// stores and returns "default". An empty namespace coming back from the API therefore
// means the user explicitly configured "" (or the object was created outside Terraform);
// rewriting it to "default" would make the next plan differ from state. This matches the
// SDKv2 flatten, which kept an empty namespace empty.
func flattenSubjects(in []rbacv1api.Subject) []SubjectModel {
	result := make([]SubjectModel, 0, len(in))
	for _, s := range in {
		result = append(result, flattenSubject(s))
	}
	return result
}

// flattenSubjectsWithPrior is flattenSubjects for Read. It keeps a prior-state
// api_group of "" when the API returns the canonical rbac.authorization.k8s.io for a
// User or Group subject: the API server defaults an empty api_group for those kinds,
// so the two values are equivalent and rewriting the configured "" would cause a
// perpetual diff.
func flattenSubjectsWithPrior(in []rbacv1api.Subject, prior []SubjectModel) []SubjectModel {
	result := flattenSubjects(in)
	for i := range result {
		if i >= len(prior) {
			break
		}
		p := prior[i]
		if p.APIGroup.IsNull() || p.APIGroup.IsUnknown() || p.APIGroup.ValueString() != "" {
			continue
		}
		if p.Kind.ValueString() != in[i].Kind || p.Name.ValueString() != in[i].Name {
			continue
		}
		if (in[i].Kind == rbacv1api.UserKind || in[i].Kind == rbacv1api.GroupKind) &&
			in[i].APIGroup == rbacv1api.GroupName {
			result[i].APIGroup = types.StringValue("")
		}
	}
	return result
}

func flattenSubject(s rbacv1api.Subject) SubjectModel {
	return SubjectModel{
		Kind:      types.StringValue(s.Kind),
		Name:      types.StringValue(s.Name),
		APIGroup:  types.StringValue(s.APIGroup),
		Namespace: types.StringValue(s.Namespace),
	}
}

// applySubjectComputedFields resolves unknown computed fields in plan subjects from the
// Kubernetes API response, while preserving all configured values.
//
// subject.api_group is Optional+Computed with no default. When a subject is configured
// without api_group (e.g. a ServiceAccount), the plan carries an unknown value for that
// field. After the API call succeeds, Kubernetes returns a concrete value. This helper
// fills in only those unknown slots; configured (known) values are left untouched.
//
// The number of subject blocks is never changed: Terraform requires the applied block
// count to equal the planned count, so any extra API subjects are ignored here and
// surface as drift on the next Read instead.
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
		// Resolve namespace only when the plan left it unknown; keep the API value as-is.
		if s.Namespace.IsUnknown() {
			s.Namespace = types.StringValue(api.Namespace)
		}
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
