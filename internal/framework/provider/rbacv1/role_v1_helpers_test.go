// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
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

func TestMetadataMapValidatorsNullValues(t *testing.T) {
	for _, v := range []struct {
		name      string
		validator validator.Map
	}{
		{name: "labels", validator: labelValidator{}},
		{name: "annotations", validator: annotationKeyValidator{}},
	} {
		for _, tc := range []struct {
			name      string
			key       string
			value     types.String
			wantError bool
		}{
			{name: "null value", key: "optional", value: types.StringNull()},
			{name: "null omits invalid API key", key: "Invalid Key!", value: types.StringNull()},
			{name: "unknown value", key: "optional", value: types.StringUnknown()},
			{name: "non-null invalid API key", key: "Invalid Key!", value: types.StringValue("present"), wantError: true},
		} {
			t.Run(v.name+"/"+tc.name, func(t *testing.T) {
				req := validator.MapRequest{
					Path:        path.Root(v.name),
					ConfigValue: types.MapValueMust(types.StringType, map[string]attr.Value{tc.key: tc.value}),
				}
				var resp validator.MapResponse
				v.validator.ValidateMap(context.Background(), req, &resp)
				if resp.Diagnostics.HasError() != tc.wantError {
					t.Fatalf("HasError() = %t, want %t: %v", resp.Diagnostics.HasError(), tc.wantError, resp.Diagnostics)
				}
			})
		}
	}
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
	if got := flattenStringMap(nil); got == nil || len(got) != 0 {
		t.Errorf("flattenStringMap(nil) = %v, want non-nil empty map", got)
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

func TestParseIDNamespace(t *testing.T) {

	t.Run("parseID rejects missing namespace separator", func(t *testing.T) {
		_, _, err := parseID("name-only")
		if err == nil {
			t.Fatal("parseID: expected error for ID without namespace, got nil")
		}
	})

	t.Run("parseID accepts namespace/name", func(t *testing.T) {
		ns, name, err := parseID("default/my-role")
		if err != nil {
			t.Fatalf("parseID: unexpected error: %v", err)
		}
		if ns != "default" || name != "my-role" {
			t.Fatalf("parseID: got ns=%q name=%q, want default/my-role", ns, name)
		}
	})
}

func TestFlattenMetadataMapNullValues(t *testing.T) {
	current := map[string]types.String{
		"keep":                         types.StringValue("retained"),
		"optional":                     types.StringNull(),
		"ignored.example.com/optional": types.StringNull(),
	}
	for _, tc := range []struct {
		name   string
		remote map[string]string
		want   map[string]types.String
	}{
		{
			name:   "absent keys retain null markers",
			remote: map[string]string{"keep": "retained"},
			want:   current,
		},
		{
			name:   "remote values reveal drift",
			remote: map[string]string{"keep": "retained", "optional": "drift"},
			want: map[string]types.String{
				"keep":                         types.StringValue("retained"),
				"optional":                     types.StringValue("drift"),
				"ignored.example.com/optional": types.StringNull(),
			},
		},
		{
			name:   "null entries do not claim ignored keys",
			remote: map[string]string{"keep": "retained", "ignored.example.com/optional": "external"},
			want:   current,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := flattenMetadataMap(tc.remote, current, []string{`^ignored\.example\.com/`})
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("metadata mismatch (-want +got):\n%s", diff)
			}
		})
	}
	if got := flattenMetadataMap(nil, nil, nil); got == nil || len(got) != 0 {
		t.Errorf("import without prior metadata = %v, want empty map", got)
	}
}

func TestDiffStringMapNullValues(t *testing.T) {
	for _, tc := range []struct {
		name string
		old  map[string]types.String
		new  map[string]types.String
		want kubernetes.PatchOperations
	}{
		{
			name: "new null marker does not patch Kubernetes",
			new:  map[string]types.String{"optional": types.StringNull()},
			want: kubernetes.PatchOperations{},
		},
		{
			name: "removing a null marker does not remove an absent API key",
			old:  map[string]types.String{"optional": types.StringNull()},
			want: kubernetes.PatchOperations{},
		},
		{
			name: "string to null removes the API key",
			old:  map[string]types.String{"example.com/key": types.StringValue("present")},
			new:  map[string]types.String{"example.com/key": types.StringNull()},
			want: kubernetes.PatchOperations{&kubernetes.RemoveOperation{Path: "/metadata/labels/example.com~1key"}},
		},
		{
			name: "null to string creates a missing API map",
			old:  map[string]types.String{"optional": types.StringNull()},
			new:  map[string]types.String{"optional": types.StringValue("present")},
			want: kubernetes.PatchOperations{&kubernetes.AddOperation{
				Path: "/metadata/labels", Value: map[string]string{"optional": "present"},
			}},
		},
		{
			name: "null markers do not mask updates to other keys",
			old:  map[string]types.String{"optional": types.StringNull(), "keep": types.StringValue("old")},
			new:  map[string]types.String{"optional": types.StringNull(), "keep": types.StringValue("new")},
			want: kubernetes.PatchOperations{&kubernetes.ReplaceOperation{Path: "/metadata/labels/keep", Value: "new"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := diffStringMap("/metadata/labels", tc.old, tc.new)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("patch mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
