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
		{"nil slice", nil, makeStringSet(nil)},
		{"empty slice", []string{}, makeStringSet(nil)},
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
