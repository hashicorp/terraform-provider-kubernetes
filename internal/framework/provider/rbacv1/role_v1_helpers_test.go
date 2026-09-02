// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	rbacv1api "k8s.io/api/rbac/v1"
)

func makeStringSet(values []string) types.Set {
	elements := make([]attr.Value, len(values))
	for i, v := range values {
		elements[i] = types.StringValue(v)
	}
	set, _ := types.SetValue(types.StringType, elements)
	return set
}

func TestExpandStringSet(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		input    types.Set
		expected []string
	}{
		{"normal set", makeStringSet([]string{"a", "b", "c"}), []string{"a", "b", "c"}},
		{"null set", types.SetNull(types.StringType), nil},
		{"empty set", makeStringSet([]string{}), []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, diags := expandStringSet(ctx, tt.input)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if diff := cmp.Diff(tt.expected, got, cmpopts.SortSlices(func(a, b string) bool { return a < b })); diff != "" {
				t.Errorf("expandStringSet() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFlattenStringSet(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected types.Set
	}{
		{"normal slice", []string{"a", "b", "c"}, makeStringSet([]string{"a", "b", "c"})},
		{"nil slice", nil, types.SetNull(types.StringType)},
		{"empty slice", []string{}, types.SetNull(types.StringType)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, diags := flattenStringSet(tt.input)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if !got.Equal(tt.expected) {
				t.Errorf("flattenStringSet() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBuildID(t *testing.T) {
	tests := []struct {
		namespace string
		name      string
		expected  string
	}{
		{"default", "my-role", "default/my-role"},
		{"kube-system", "admin-role", "kube-system/admin-role"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := buildID(tt.namespace, tt.name)
			if got != tt.expected {
				t.Errorf("buildID(%q, %q) = %q, want %q", tt.namespace, tt.name, got, tt.expected)
			}
		})
	}
}

func TestParseID(t *testing.T) {
	tests := []struct {
		input         string
		wantNamespace string
		wantName      string
		wantErr       bool
	}{
		{"default/my-role", "default", "my-role", false},
		{"kube-system/admin-role", "kube-system", "admin-role", false},
		{"invalid", "", "", true},
		{"too/many/parts", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			gotNS, gotName, err := parseID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseID(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if gotNS != tt.wantNamespace || gotName != tt.wantName {
					t.Errorf("parseID(%q) = (%q, %q), want (%q, %q)",
						tt.input, gotNS, gotName, tt.wantNamespace, tt.wantName)
				}
			}
		})
	}
}

func TestExpandFlattenPolicyRules(t *testing.T) {
	ctx := context.Background()

	input := []RuleModel{
		{
			APIGroups: makeStringSet([]string{""}),
			Resources: makeStringSet([]string{"pods"}),
			Verbs:     makeStringSet([]string{"get", "list"}),
		},
	}

	got, diags := expandPolicyRules(ctx, input)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	expected := []rbacv1api.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"get", "list"},
		},
	}

	if len(got) != len(expected) {
		t.Fatalf("expandPolicyRules() returned %d rules, want %d", len(got), len(expected))
	}
	if diff := cmp.Diff(expected[0].APIGroups, got[0].APIGroups); diff != "" {
		t.Errorf("APIGroups mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(expected[0].Resources, got[0].Resources); diff != "" {
		t.Errorf("Resources mismatch (-want +got):\n%s", diff)
	}

	roundTripped, diags := flattenPolicyRules(got)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(roundTripped) != 1 {
		t.Fatalf("flattenPolicyRules() returned %d rules, want 1", len(roundTripped))
	}
	if !roundTripped[0].APIGroups.Equal(input[0].APIGroups) {
		t.Errorf("APIGroups round-trip mismatch: got %v, want %v", roundTripped[0].APIGroups, input[0].APIGroups)
	}
}

func TestRulesEqual(t *testing.T) {
	a := []RuleModel{{
		APIGroups:     makeStringSet([]string{""}),
		Resources:     makeStringSet([]string{"pods"}),
		ResourceNames: types.SetNull(types.StringType),
		Verbs:         makeStringSet([]string{"get", "list"}),
	}}
	b := []RuleModel{{
		APIGroups:     makeStringSet([]string{""}),
		Resources:     makeStringSet([]string{"pods"}),
		ResourceNames: types.SetNull(types.StringType),
		Verbs:         makeStringSet([]string{"list", "get"}), // different order, same set
	}}
	c := []RuleModel{{
		APIGroups:     makeStringSet([]string{""}),
		Resources:     makeStringSet([]string{"pods"}),
		ResourceNames: types.SetNull(types.StringType),
		Verbs:         makeStringSet([]string{"get"}),
	}}

	if !rulesEqual(a, b) {
		t.Errorf("rulesEqual() = false, want true for sets with same elements in different order")
	}
	if rulesEqual(a, c) {
		t.Errorf("rulesEqual() = true, want false for sets with different elements")
	}
	if rulesEqual(a, nil) {
		t.Errorf("rulesEqual() = true, want false for different lengths")
	}
}

func TestExpandFlattenStringMap(t *testing.T) {
	m := map[string]types.String{
		"a": types.StringValue("1"),
		"b": types.StringValue("2"),
	}
	expanded := expandStringMap(m)
	if len(expanded) != 2 || expanded["a"] != "1" || expanded["b"] != "2" {
		t.Errorf("expandStringMap() = %v", expanded)
	}

	flattened := flattenStringMap(expanded)
	if len(flattened) != 2 || !flattened["a"].Equal(types.StringValue("1")) {
		t.Errorf("flattenStringMap() = %v", flattened)
	}

	if expandStringMap(nil) != nil {
		t.Errorf("expandStringMap(nil) should be nil")
	}
	if flattenStringMap(nil) != nil {
		t.Errorf("flattenStringMap(nil) should be nil")
	}
}

func TestIsInternalMetadataKey(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{"kubectl.kubernetes.io/last-applied-configuration", true},
		{"pv.kubernetes.io/bound-by-controller", true},
		{"app.kubernetes.io/name", false},                 // explicitly allow-listed
		{"service.beta.kubernetes.io/aws-lb-type", false}, // explicitly allow-listed
		{"example.com/owner", false},
		{"deprecated.daemonset.template.generation", true},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got := isInternalMetadataKey(tt.key); got != tt.want {
				t.Errorf("isInternalMetadataKey(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestMatchesIgnorePattern(t *testing.T) {
	patterns := []string{"^ignore\\..*", "^also-ignore$"}

	if !matchesIgnorePattern("ignore.me", patterns) {
		t.Error("expected ignore.me to match")
	}
	if !matchesIgnorePattern("also-ignore", patterns) {
		t.Error("expected also-ignore to match")
	}
	if matchesIgnorePattern("keep.me", patterns) {
		t.Error("expected keep.me not to match")
	}
	if matchesIgnorePattern("anything", nil) {
		t.Error("expected no patterns to match nothing")
	}
}

func TestFilterManagedMetadataKeys(t *testing.T) {
	fromCluster := map[string]string{
		"kubectl.kubernetes.io/last-applied-configuration": "{...}",    // internal, unmanaged -> dropped
		"ignored.example.com/pattern":                      "x",        // matches ignore pattern, unmanaged -> dropped
		"team":                                             "platform", // ordinary key -> kept
	}

	t.Run("nothing managed, internal and ignored keys dropped", func(t *testing.T) {
		got := filterManagedMetadataKeys(fromCluster, nil, []string{"^ignored\\..*"})
		want := map[string]string{"team": "platform"}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("keys already managed by TF are kept even if internal/ignored", func(t *testing.T) {
		current := map[string]types.String{
			"kubectl.kubernetes.io/last-applied-configuration": types.StringValue("{...}"),
			"ignored.example.com/pattern":                      types.StringValue("x"),
		}
		got := filterManagedMetadataKeys(fromCluster, current, []string{"^ignored\\..*"})
		if diff := cmp.Diff(fromCluster, got); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("empty input returns nil", func(t *testing.T) {
		if got := filterManagedMetadataKeys(nil, nil, nil); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
}

func TestBuildMetadataPatch(t *testing.T) {
	state := MetadataModel{
		Labels: map[string]types.String{"a": types.StringValue("1")},
	}
	plan := MetadataModel{
		Labels: map[string]types.String{"a": types.StringValue("2"), "b": types.StringValue("3")},
	}

	ops := buildMetadataPatch(plan, state)
	if len(ops) == 0 {
		t.Fatalf("expected patch operations for changed labels")
	}

	same := buildMetadataPatch(state, state)
	if len(same) != 0 {
		t.Errorf("expected no patch operations for unchanged metadata, got %d", len(same))
	}
}
