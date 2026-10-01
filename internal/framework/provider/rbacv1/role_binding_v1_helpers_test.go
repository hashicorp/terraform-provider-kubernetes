// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

// Internal-package unit tests for unexported helpers.
// These complement the external rbacv1_test suite; they live in package rbacv1
// so they can access package-private functions.

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	rbacv1api "k8s.io/api/rbac/v1"
)

// ── applySubjectComputedFields ─────────────────────────────────────────────────

// TestApplySubjectComputedFields_unknownAPIGroupResolvedFromResponse is the
// exact scenario reported by the reviewer:
//
//   - A subject is configured without api_group (ServiceAccount case).
//   - The plan therefore carries an *unknown* api_group value.
//   - Kubernetes returns a concrete value ("" for ServiceAccount).
//   - After applySubjectComputedFields the plan must contain the concrete
//     value, not unknown.
//
// This is a pure unit test; no cluster is required.
func TestApplySubjectComputedFields_unknownAPIGroupResolvedFromResponse(t *testing.T) {
	t.Parallel()

	// Simulate the plan as the Framework produces it: api_group is unknown
	// because the attribute is Optional+Computed with no Default and the
	// practitioner omitted it.
	plan := []SubjectModel{
		{
			Kind:      types.StringValue("ServiceAccount"),
			Name:      types.StringValue("my-sa"),
			APIGroup:  types.StringUnknown(), // omitted → unknown in plan
			Namespace: types.StringValue("kube-system"),
		},
	}

	// Kubernetes returns the concrete api_group value ("" for ServiceAccount).
	apiResponse := []rbacv1api.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      "my-sa",
			APIGroup:  "", // Kubernetes fills this in
			Namespace: "kube-system",
		},
	}

	applySubjectComputedFields(&plan, apiResponse)

	got := plan[0]

	// api_group must be resolved to the concrete value, not remain unknown.
	if got.APIGroup.IsUnknown() {
		t.Fatal("api_group is still unknown after applySubjectComputedFields; expected concrete value")
	}
	if got.APIGroup.ValueString() != "" {
		t.Errorf("api_group = %q, want %q (empty string for ServiceAccount)", got.APIGroup.ValueString(), "")
	}

	// Configured values must be preserved unchanged.
	if got.Kind.ValueString() != "ServiceAccount" {
		t.Errorf("Kind = %q, want %q", got.Kind.ValueString(), "ServiceAccount")
	}
	if got.Name.ValueString() != "my-sa" {
		t.Errorf("Name = %q, want %q", got.Name.ValueString(), "my-sa")
	}
	if got.Namespace.ValueString() != "kube-system" {
		t.Errorf("Namespace = %q, want %q", got.Namespace.ValueString(), "kube-system")
	}
}

// TestApplySubjectComputedFields_knownAPIGroupPreserved verifies that a
// configured (known) api_group is never overwritten by the API response.
func TestApplySubjectComputedFields_knownAPIGroupPreserved(t *testing.T) {
	t.Parallel()

	plan := []SubjectModel{
		{
			Kind:      types.StringValue("User"),
			Name:      types.StringValue("alice"),
			APIGroup:  types.StringValue("rbac.authorization.k8s.io"), // explicitly configured
			Namespace: types.StringValue("default"),
		},
	}

	// Kubernetes echoes the same value back; we are verifying the helper does
	// not overwrite a known value even when the API agrees.
	apiResponse := []rbacv1api.Subject{
		{
			Kind:     "User",
			Name:     "alice",
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	applySubjectComputedFields(&plan, apiResponse)

	got := plan[0]
	if got.APIGroup.IsUnknown() {
		t.Fatal("api_group became unknown unexpectedly")
	}
	if got.APIGroup.ValueString() != "rbac.authorization.k8s.io" {
		t.Errorf("api_group = %q, want %q", got.APIGroup.ValueString(), "rbac.authorization.k8s.io")
	}
}

// TestApplySubjectComputedFields_unknownNamespaceResolved verifies that an
// unknown namespace is resolved from the API response with the same
// empty-string → "default" normalisation used by flattenSubjects.
func TestApplySubjectComputedFields_unknownNamespaceResolved(t *testing.T) {
	t.Parallel()

	plan := []SubjectModel{
		{
			Kind:      types.StringValue("User"),
			Name:      types.StringValue("bob"),
			APIGroup:  types.StringValue("rbac.authorization.k8s.io"),
			Namespace: types.StringUnknown(), // unknown: schema Default not yet applied
		},
	}

	// Kubernetes returns "" for namespace on a User subject (non-namespaced).
	apiResponse := []rbacv1api.Subject{
		{Kind: "User", Name: "bob", APIGroup: "rbac.authorization.k8s.io", Namespace: ""},
	}

	applySubjectComputedFields(&plan, apiResponse)

	got := plan[0]
	if got.Namespace.IsUnknown() {
		t.Fatal("namespace is still unknown after applySubjectComputedFields")
	}
	// Empty string from API → "default" per flattenSubjects convention.
	if got.Namespace.ValueString() != "default" {
		t.Errorf("namespace = %q, want %q", got.Namespace.ValueString(), "default")
	}
}

// TestApplySubjectComputedFields_knownNamespacePreserved verifies that a
// known configured namespace is NOT overwritten even if the API returns a
// different value.
func TestApplySubjectComputedFields_knownNamespacePreserved(t *testing.T) {
	t.Parallel()

	plan := []SubjectModel{
		{
			Kind:      types.StringValue("ServiceAccount"),
			Name:      types.StringValue("sa"),
			APIGroup:  types.StringValue(""),
			Namespace: types.StringValue("production"), // explicitly configured
		},
	}

	// API might normalise or echo back; the plan value must win.
	apiResponse := []rbacv1api.Subject{
		{Kind: "ServiceAccount", Name: "sa", APIGroup: "", Namespace: "production"},
	}

	applySubjectComputedFields(&plan, apiResponse)

	got := plan[0]
	if got.Namespace.ValueString() != "production" {
		t.Errorf("namespace = %q, want %q", got.Namespace.ValueString(), "production")
	}
}

// TestApplySubjectComputedFields_multipleSubjectsPartialUnknown covers the
// mixed case: two subjects, one with a known api_group and one with an unknown
// api_group. Only the unknown slot must be resolved.
func TestApplySubjectComputedFields_multipleSubjectsPartialUnknown(t *testing.T) {
	t.Parallel()

	plan := []SubjectModel{
		{
			Kind:      types.StringValue("User"),
			Name:      types.StringValue("alice"),
			APIGroup:  types.StringValue("rbac.authorization.k8s.io"), // known
			Namespace: types.StringValue("default"),
		},
		{
			Kind:      types.StringValue("ServiceAccount"),
			Name:      types.StringValue("ci-runner"),
			APIGroup:  types.StringUnknown(), // unknown: omitted in config
			Namespace: types.StringValue("ci"),
		},
	}

	apiResponse := []rbacv1api.Subject{
		{Kind: "User", Name: "alice", APIGroup: "rbac.authorization.k8s.io"},
		{Kind: "ServiceAccount", Name: "ci-runner", APIGroup: "", Namespace: "ci"},
	}

	applySubjectComputedFields(&plan, apiResponse)

	if plan[0].APIGroup.ValueString() != "rbac.authorization.k8s.io" {
		t.Errorf("subject[0] api_group = %q, want %q", plan[0].APIGroup.ValueString(), "rbac.authorization.k8s.io")
	}
	if plan[1].APIGroup.IsUnknown() {
		t.Fatal("subject[1] api_group is still unknown")
	}
	if plan[1].APIGroup.ValueString() != "" {
		t.Errorf("subject[1] api_group = %q, want %q (empty for ServiceAccount)", plan[1].APIGroup.ValueString(), "")
	}
}

// TestApplySubjectComputedFields_groupSubjectNonEmptyAPIGroup reproduces the
// reviewer's second key scenario: Group subject with api_group = "rbac.authorization.k8s.io"
// omitted from config → unknown in plan → resolved from Kubernetes response.
func TestApplySubjectComputedFields_groupSubjectNonEmptyAPIGroup(t *testing.T) {
	t.Parallel()

	plan := []SubjectModel{
		{
			Kind:      types.StringValue("Group"),
			Name:      types.StringValue("dev-team"),
			APIGroup:  types.StringUnknown(), // omitted
			Namespace: types.StringValue("default"),
		},
	}

	// Kubernetes returns the concrete group api_group.
	apiResponse := []rbacv1api.Subject{
		{Kind: "Group", Name: "dev-team", APIGroup: "rbac.authorization.k8s.io", Namespace: ""},
	}

	applySubjectComputedFields(&plan, apiResponse)

	got := plan[0]
	if got.APIGroup.IsUnknown() {
		t.Fatal("api_group still unknown for Group subject")
	}
	if got.APIGroup.ValueString() != "rbac.authorization.k8s.io" {
		t.Errorf("api_group = %q, want %q", got.APIGroup.ValueString(), "rbac.authorization.k8s.io")
	}
}
