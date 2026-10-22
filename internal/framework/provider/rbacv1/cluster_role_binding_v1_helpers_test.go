// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	api "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestApplyPlanResultPreservesMetadata(t *testing.T) {
	for _, labels := range []types.Map{types.MapNull(types.StringType), types.MapValueMust(types.StringType, nil)} {
		t.Run(labels.String(), func(t *testing.T) {
			plan := ClusterRoleBindingModel{
				ID: types.StringUnknown(),
				Metadata: []common.MetadataModel{{
					MetadataBase: common.MetadataBase{Labels: labels, Annotations: labels},
					GenerateName: types.StringValue("tf-acc:"),
				}},
				Subject: []SubjectModel{{APIGroup: types.StringUnknown(), Namespace: types.StringValue("default")}},
			}
			out := &api.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "tf-acc:generated", UID: "uid", ResourceVersion: "42", Generation: 1,
					Labels:      map[string]string{"injected": "true"},
					Annotations: map[string]string{"injected": "true"},
				},
				Subjects: []api.Subject{{APIGroup: "rbac.authorization.k8s.io"}},
			}
			applyPlanResult(&plan, out)
			if !plan.Metadata[0].Labels.Equal(labels) || !plan.Metadata[0].Annotations.Equal(labels) {
				t.Fatalf("apply must echo planned metadata: %#v", plan.Metadata[0])
			}
			if plan.ID.ValueString() != out.Name || plan.Metadata[0].Name.ValueString() != out.Name ||
				plan.Metadata[0].UID.ValueString() != string(out.UID) ||
				plan.Metadata[0].ResourceVersion.ValueString() != out.ResourceVersion ||
				plan.Metadata[0].Generation.ValueInt64() != out.Generation {
				t.Fatalf("server-assigned fields were not populated: %#v", plan)
			}
			if plan.Metadata[0].GenerateName.ValueString() != "tf-acc:" ||
				plan.Subject[0].Namespace.ValueString() != "default" ||
				plan.Subject[0].APIGroup.ValueString() != out.Subjects[0].APIGroup {
				t.Fatalf("unexpected planned or computed values: %#v", plan)
			}
		})
	}
}

// TestSubjectsEqual covers the subject comparison helper.
func TestSubjectsEqual(t *testing.T) {
	t.Parallel()

	sv := types.StringValue

	user := SubjectModel{Kind: sv("User"), Name: sv("alice"), APIGroup: sv("rbac.authorization.k8s.io"), Namespace: sv("default")}
	sa := SubjectModel{Kind: sv("ServiceAccount"), Name: sv("svc"), APIGroup: sv(""), Namespace: sv("kube-system")}
	grp := SubjectModel{Kind: sv("Group"), Name: sv("system:masters"), APIGroup: sv("rbac.authorization.k8s.io"), Namespace: sv("default")}

	if !subjectsEqual([]SubjectModel{user}, []SubjectModel{user}) {
		t.Error("identical slices should be equal")
	}
	if subjectsEqual([]SubjectModel{user}, []SubjectModel{sa}) {
		t.Error("different subjects should not be equal")
	}
	if subjectsEqual([]SubjectModel{user}, []SubjectModel{user, sa}) {
		t.Error("different lengths should not be equal")
	}
	if !subjectsEqual(nil, nil) {
		t.Error("nil slices should be equal")
	}
	if !subjectsEqual([]SubjectModel{}, []SubjectModel{}) {
		t.Error("empty slices should be equal")
	}
	// Order matters — reordering must not be equal.
	if subjectsEqual([]SubjectModel{user, sa}, []SubjectModel{sa, user}) {
		t.Error("reordered subjects should not be equal")
	}
	// Null/unknown values must not panic.
	null := SubjectModel{
		Kind:      types.StringNull(),
		Name:      types.StringNull(),
		APIGroup:  types.StringNull(),
		Namespace: types.StringNull(),
	}
	unknown := SubjectModel{
		Kind:      types.StringUnknown(),
		Name:      types.StringUnknown(),
		APIGroup:  types.StringUnknown(),
		Namespace: types.StringUnknown(),
	}
	_ = subjectsEqual([]SubjectModel{null}, []SubjectModel{user})
	_ = subjectsEqual([]SubjectModel{unknown}, []SubjectModel{grp})
	if !subjectsEqual([]SubjectModel{null}, []SubjectModel{null}) {
		t.Error("identical null-subject slices should be equal")
	}
}

// TestFlattenSubjects_namespacePreserved verifies that flattenSubjects keeps the
// namespace returned by the Kubernetes API, including an explicitly empty one,
// instead of re-applying the schema default.
func TestFlattenSubjects_namespacePreserved(t *testing.T) {
	t.Parallel()

	input := []api.Subject{
		{
			Kind:      "User",
			Name:      "alice",
			APIGroup:  "rbac.authorization.k8s.io",
			Namespace: "",
		},
		{
			Kind:      "Group",
			Name:      "dev-team",
			APIGroup:  "rbac.authorization.k8s.io",
			Namespace: "default",
		},
		{
			Kind:      "ServiceAccount",
			Name:      "custom-sa",
			APIGroup:  "",
			Namespace: "kube-system",
		},
	}

	got := flattenSubjects(input)

	if len(got) != 3 {
		t.Fatalf("flattenSubjects returned %d subjects, want 3", len(got))
	}

	if got[0].Namespace.IsNull() || got[0].Namespace.ValueString() != "" {
		t.Errorf("User subject namespace = %v, want empty string", got[0].Namespace)
	}
	if got[1].Namespace.ValueString() != "default" {
		t.Errorf("Group subject namespace = %q, want %q", got[1].Namespace.ValueString(), "default")
	}
	if got[2].Namespace.ValueString() != "kube-system" {
		t.Errorf("ServiceAccount subject namespace = %q, want %q", got[2].Namespace.ValueString(), "kube-system")
	}
}

// TestFlattenSubjectsWithPrior verifies that Read keeps a configured empty
// api_group for User and Group subjects, which the API server defaults to
// rbac.authorization.k8s.io, and otherwise reports the API value.
func TestFlattenSubjectsWithPrior(t *testing.T) {
	t.Parallel()

	sv := types.StringValue
	subject := func(kind, name, apiGroup string) SubjectModel {
		return SubjectModel{Kind: sv(kind), Name: sv(name), APIGroup: sv(apiGroup), Namespace: sv("default")}
	}

	tests := []struct {
		name  string
		in    api.Subject
		prior []SubjectModel
		want  string
	}{
		{"user empty prior", api.Subject{Kind: "User", Name: "alice", APIGroup: api.GroupName}, []SubjectModel{subject("User", "alice", "")}, ""},
		{"group empty prior", api.Subject{Kind: "Group", Name: "devs", APIGroup: api.GroupName}, []SubjectModel{subject("Group", "devs", "")}, ""},
		{"user canonical prior", api.Subject{Kind: "User", Name: "alice", APIGroup: api.GroupName}, []SubjectModel{subject("User", "alice", api.GroupName)}, api.GroupName},
		{"no prior (import)", api.Subject{Kind: "User", Name: "alice", APIGroup: api.GroupName}, nil, api.GroupName},
		{"null prior", api.Subject{Kind: "User", Name: "alice", APIGroup: api.GroupName},
			[]SubjectModel{{Kind: sv("User"), Name: sv("alice"), APIGroup: types.StringNull()}}, api.GroupName},
		{"different subject at index", api.Subject{Kind: "User", Name: "bob", APIGroup: api.GroupName}, []SubjectModel{subject("User", "alice", "")}, api.GroupName},
		{"non-canonical api group", api.Subject{Kind: "User", Name: "alice", APIGroup: "example.com"}, []SubjectModel{subject("User", "alice", "")}, "example.com"},
		{"service account", api.Subject{Kind: "ServiceAccount", Name: "sa", Namespace: "default"}, []SubjectModel{subject("ServiceAccount", "sa", "")}, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := flattenSubjectsWithPrior([]api.Subject{tc.in}, tc.prior)
			if len(got) != 1 || got[0].APIGroup.ValueString() != tc.want {
				t.Fatalf("api_group = %v, want %q", got, tc.want)
			}
		})
	}
}

// TestApplyPlanResultFewerAPISubjects verifies that a response with fewer
// subjects than planned does not panic and leaves the missing slots unknown.
func TestApplyPlanResultFewerAPISubjects(t *testing.T) {
	t.Parallel()

	plan := ClusterRoleBindingModel{
		Metadata: []common.MetadataModel{{}},
		Subject: []SubjectModel{
			{APIGroup: types.StringUnknown()},
			{APIGroup: types.StringUnknown()},
		},
	}
	out := &api.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "binding"},
		Subjects:   []api.Subject{{APIGroup: ""}},
	}
	applyPlanResult(&plan, out)
	if plan.Subject[0].APIGroup.IsUnknown() || !plan.Subject[1].APIGroup.IsUnknown() {
		t.Fatalf("unexpected subjects: %#v", plan.Subject)
	}
}
