// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	api "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestExpandClusterRoleAggregationRule(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *api.AggregationRule
	}{
		{
			[]interface{}{},
			&api.AggregationRule{},
		},
		{
			[]interface{}{
				map[string]interface{}{
					"cluster_role_selectors": []interface{}{
						map[string]interface{}{
							"match_labels": map[string]interface{}{"key": "value"},
						},
					},
				},
			},
			&api.AggregationRule{
				ClusterRoleSelectors: []metav1.LabelSelector{
					{
						MatchLabels: map[string]string{"key": "value"},
					},
				},
			},
		},
		{
			[]interface{}{
				map[string]interface{}{
					"cluster_role_selectors": []interface{}{
						map[string]interface{}{
							"match_labels": map[string]interface{}{"key": "value"},
						},
						map[string]interface{}{
							"match_labels": map[string]interface{}{"foo": "bar"},
						},
					},
				},
			},
			&api.AggregationRule{
				ClusterRoleSelectors: []metav1.LabelSelector{
					{
						MatchLabels: map[string]string{"key": "value"},
					},
					{
						MatchLabels: map[string]string{"foo": "bar"},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		output := expandClusterRoleAggregationRule(tc.Input)
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from expander.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestFlattenClusterRoleAggregationRule(t *testing.T) {
	cases := []struct {
		Input          *api.AggregationRule
		ExpectedOutput []interface{}
	}{
		{
			&api.AggregationRule{},
			[]interface{}{map[string]interface{}{}},
		},
		{
			&api.AggregationRule{
				ClusterRoleSelectors: []metav1.LabelSelector{
					{
						MatchLabels: map[string]string{"key": "value"},
					},
				},
			},
			[]interface{}{
				map[string]interface{}{
					"cluster_role_selectors": []interface{}{
						map[string]interface{}{
							"match_labels": map[string]string{"key": "value"},
						},
					},
				},
			},
		},
		{
			&api.AggregationRule{
				ClusterRoleSelectors: []metav1.LabelSelector{
					{
						MatchLabels: map[string]string{"key": "value"},
					},
					{
						MatchLabels: map[string]string{"foo": "bar"},
					},
				},
			},
			[]interface{}{
				map[string]interface{}{
					"cluster_role_selectors": []interface{}{
						map[string]interface{}{
							"match_labels": map[string]string{"key": "value"},
						},
						map[string]interface{}{
							"match_labels": map[string]string{"foo": "bar"},
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		output := flattenClusterRoleAggregationRule(tc.Input)
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from flattener.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}

func TestExpandRBACSubjects(t *testing.T) {
	cases := []struct {
		Name           string
		Input          []interface{}
		ExpectedOutput []api.Subject
	}{
		{
			"empty",
			[]interface{}{},
			[]api.Subject{},
		},
		{
			"Group and User subjects are not namespaced",
			[]interface{}{
				map[string]interface{}{
					"api_group": "rbac.authorization.k8s.io",
					"kind":      "Group",
					"name":      "admins",
					"namespace": "",
				},
				map[string]interface{}{
					"api_group": "rbac.authorization.k8s.io",
					"kind":      "User",
					"name":      "jane",
					"namespace": "",
				},
			},
			[]api.Subject{
				{APIGroup: "rbac.authorization.k8s.io", Kind: "Group", Name: "admins"},
				{APIGroup: "rbac.authorization.k8s.io", Kind: "User", Name: "jane"},
			},
		},
		{
			"ServiceAccount without a namespace uses the default namespace",
			[]interface{}{
				map[string]interface{}{
					"api_group": "",
					"kind":      "ServiceAccount",
					"name":      "default",
					"namespace": "",
				},
			},
			[]api.Subject{
				{Kind: "ServiceAccount", Name: "default", Namespace: "default"},
			},
		},
		{
			"explicit namespaces are preserved",
			[]interface{}{
				map[string]interface{}{
					"api_group": "",
					"kind":      "ServiceAccount",
					"name":      "builder",
					"namespace": "ci",
				},
				map[string]interface{}{
					"api_group": "rbac.authorization.k8s.io",
					"kind":      "Group",
					"name":      "admins",
					"namespace": "ci",
				},
			},
			[]api.Subject{
				{Kind: "ServiceAccount", Name: "builder", Namespace: "ci"},
				{APIGroup: "rbac.authorization.k8s.io", Kind: "Group", Name: "admins", Namespace: "ci"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			output := expandRBACSubjects(tc.Input)
			if !reflect.DeepEqual(output, tc.ExpectedOutput) {
				t.Fatalf("Unexpected output from expander.\nExpected: %#v\nGiven:    %#v",
					tc.ExpectedOutput, output)
			}
		})
	}
}

func TestFlattenRBACSubjects(t *testing.T) {
	cases := []struct {
		Input          []api.Subject
		ExpectedOutput []interface{}
	}{
		{
			[]api.Subject{},
			[]interface{}{},
		},
		{
			[]api.Subject{
				{APIGroup: "rbac.authorization.k8s.io", Kind: "Group", Name: "admins"},
				{Kind: "ServiceAccount", Name: "builder", Namespace: "ci"},
			},
			[]interface{}{
				map[string]interface{}{
					"api_group": "rbac.authorization.k8s.io",
					"kind":      "Group",
					"name":      "admins",
				},
				map[string]interface{}{
					"kind":      "ServiceAccount",
					"name":      "builder",
					"namespace": "ci",
				},
			},
		},
	}

	for _, tc := range cases {
		output := flattenRBACSubjects(tc.Input)
		if !reflect.DeepEqual(output, tc.ExpectedOutput) {
			t.Fatalf("Unexpected output from flattener.\nExpected: %#v\nGiven:    %#v",
				tc.ExpectedOutput, output)
		}
	}
}

// Subjects of kind User and Group are not namespaced, so a configuration that omits
// the namespace must not send one to the API server. Only ServiceAccount subjects
// fall back to the "default" namespace.
func TestRBACSubjectSchema_namespace(t *testing.T) {
	s := map[string]*schema.Schema{
		"subject": {
			Type:     schema.TypeList,
			Required: true,
			Elem: &schema.Resource{
				Schema: rbacSubjectSchema(),
			},
		},
	}
	d := schema.TestResourceDataRaw(t, s, map[string]interface{}{
		"subject": []interface{}{
			map[string]interface{}{"kind": "Group", "name": "admins", "api_group": "rbac.authorization.k8s.io"},
			map[string]interface{}{"kind": "User", "name": "jane", "api_group": "rbac.authorization.k8s.io"},
			map[string]interface{}{"kind": "ServiceAccount", "name": "default"},
			map[string]interface{}{"kind": "ServiceAccount", "name": "builder", "namespace": "ci"},
		},
	})
	expected := []api.Subject{
		{APIGroup: "rbac.authorization.k8s.io", Kind: "Group", Name: "admins"},
		{APIGroup: "rbac.authorization.k8s.io", Kind: "User", Name: "jane"},
		{Kind: "ServiceAccount", Name: "default", Namespace: "default"},
		{Kind: "ServiceAccount", Name: "builder", Namespace: "ci"},
	}

	output := expandRBACSubjects(d.Get("subject").([]interface{}))
	if !reflect.DeepEqual(output, expected) {
		t.Fatalf("Unexpected subjects from schema.\nExpected: %#v\nGiven:    %#v", expected, output)
	}
}

// State written by older provider versions holds namespace = "default" for every
// subject. An unset namespace must not produce a diff against it, but the
// suppression must not carry a stale "default" over to a different subject when
// the list shifts, e.g. because a leading subject was removed.
func TestRBACSubjectSchema_namespaceDiffSuppress(t *testing.T) {
	if rbacSubjectSchema()["namespace"].DiffSuppressFunc == nil {
		t.Fatal("expected a DiffSuppressFunc on the subject namespace attribute")
	}
	r := &schema.Resource{
		Schema: map[string]*schema.Schema{
			"subject": {
				Type:     schema.TypeList,
				Required: true,
				Elem:     &schema.Resource{Schema: rbacSubjectSchema()},
			},
		},
	}
	// State as written by a provider version that defaulted every namespace.
	state := &terraform.InstanceState{
		ID: "test",
		Attributes: map[string]string{
			"subject.#":           "2",
			"subject.0.api_group": "",
			"subject.0.kind":      "ServiceAccount",
			"subject.0.name":      "default",
			"subject.0.namespace": "default",
			"subject.1.api_group": "rbac.authorization.k8s.io",
			"subject.1.kind":      "Group",
			"subject.1.name":      "reviewers",
			"subject.1.namespace": "default",
		},
	}
	sa := map[string]interface{}{"kind": "ServiceAccount", "name": "default"}
	group := map[string]interface{}{"api_group": "rbac.authorization.k8s.io", "kind": "Group", "name": "reviewers"}

	plan := func(t *testing.T, subjects ...interface{}) []api.Subject {
		cfg := terraform.NewResourceConfigRaw(map[string]interface{}{"subject": subjects})
		diff, err := r.Diff(context.Background(), state, cfg, nil)
		if err != nil {
			t.Fatalf("diff: %v", err)
		}
		if diff == nil {
			return nil
		}
		d, err := schema.InternalMap(r.Schema).Data(state, diff)
		if err != nil {
			t.Fatalf("data: %v", err)
		}
		return expandRBACSubjects(d.Get("subject").([]interface{}))
	}

	t.Run("unset namespace does not diff against legacy default", func(t *testing.T) {
		if got := plan(t, sa, group); got != nil {
			t.Fatalf("expected no diff, planned subjects: %#v", got)
		}
	})
	t.Run("removing a leading subject does not carry its namespace over", func(t *testing.T) {
		expected := []api.Subject{{APIGroup: "rbac.authorization.k8s.io", Kind: "Group", Name: "reviewers"}}
		if got := plan(t, group); !reflect.DeepEqual(got, expected) {
			t.Fatalf("Unexpected planned subjects.\nExpected: %#v\nGiven:    %#v", expected, got)
		}
	})
	t.Run("explicit namespace is not suppressed", func(t *testing.T) {
		withNs := map[string]interface{}{"api_group": "rbac.authorization.k8s.io", "kind": "Group", "name": "reviewers", "namespace": "kube-system"}
		expected := []api.Subject{
			{Kind: "ServiceAccount", Name: "default", Namespace: "default"},
			{APIGroup: "rbac.authorization.k8s.io", Kind: "Group", Name: "reviewers", Namespace: "kube-system"},
		}
		if got := plan(t, sa, withNs); !reflect.DeepEqual(got, expected) {
			t.Fatalf("Unexpected planned subjects.\nExpected: %#v\nGiven:    %#v", expected, got)
		}
	})
}
