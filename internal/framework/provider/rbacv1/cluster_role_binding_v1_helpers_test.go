// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

// TestFilterManagedMetadataKeys covers the core filtering logic.
func TestFilterManagedMetadataKeys(t *testing.T) {
	t.Parallel()

	strVal := func(s string) types.String { return types.StringValue(s) }

	tests := []struct {
		name           string
		input          map[string]string
		current        map[string]types.String
		ignorePatterns []string
		wantKeys       []string // keys that must appear in result
		wantAbsent     []string // keys that must NOT appear in result
	}{
		{
			name:     "ordinary keys are retained",
			input:    map[string]string{"team": "platform", "env": "prod"},
			current:  nil,
			wantKeys: []string{"team", "env"},
		},
		{
			name:           "ignored annotation is removed when untracked",
			input:          map[string]string{"ignored.example.com/owner": "controller", "team": "platform"},
			current:        nil,
			ignorePatterns: []string{`^ignored\.example\.com/`},
			wantKeys:       []string{"team"},
			wantAbsent:     []string{"ignored.example.com/owner"},
		},
		{
			name:           "ignored label is removed when untracked",
			input:          map[string]string{"skip.io/label": "val", "keep": "yes"},
			current:        nil,
			ignorePatterns: []string{`^skip\.io/`},
			wantKeys:       []string{"keep"},
			wantAbsent:     []string{"skip.io/label"},
		},
		{
			name:           "already-managed ignored key is retained",
			input:          map[string]string{"ignored.example.com/owner": "controller"},
			current:        map[string]types.String{"ignored.example.com/owner": strVal("controller")},
			ignorePatterns: []string{`^ignored\.example\.com/`},
			wantKeys:       []string{"ignored.example.com/owner"},
		},
		{
			name:       "internal kubernetes.io key is removed when untracked",
			input:      map[string]string{"kubectl.kubernetes.io/last-applied-configuration": "{}", "team": "platform"},
			current:    nil,
			wantKeys:   []string{"team"},
			wantAbsent: []string{"kubectl.kubernetes.io/last-applied-configuration"},
		},
		{
			name:     "app.kubernetes.io keys are retained",
			input:    map[string]string{"app.kubernetes.io/name": "example", "app.kubernetes.io/version": "1.0"},
			current:  nil,
			wantKeys: []string{"app.kubernetes.io/name", "app.kubernetes.io/version"},
		},
		{
			name:     "service.beta.kubernetes.io keys are retained",
			input:    map[string]string{"service.beta.kubernetes.io/aws-load-balancer-type": "nlb"},
			current:  nil,
			wantKeys: []string{"service.beta.kubernetes.io/aws-load-balancer-type"},
		},
		{
			name:    "empty input returns nil",
			input:   map[string]string{},
			current: nil,
			// result should be nil; wantKeys is empty so no keys are checked
		},
		{
			name:           "invalid regex does not panic",
			input:          map[string]string{"team": "platform"},
			current:        nil,
			ignorePatterns: []string{"[invalid"},
			wantKeys:       []string{"team"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := filterManagedMetadataKeys(tc.input, tc.current, tc.ignorePatterns)

			for _, key := range tc.wantKeys {
				if _, ok := got[key]; !ok {
					t.Errorf("expected key %q to be present in result, but it was absent; result=%v", key, got)
				}
			}
			for _, key := range tc.wantAbsent {
				if _, ok := got[key]; ok {
					t.Errorf("expected key %q to be absent from result, but it was present; result=%v", key, got)
				}
			}
		})
	}
}

// TestIsInternalMetadataKey exercises the internal-key detection logic.
func TestIsInternalMetadataKey(t *testing.T) {
	t.Parallel()

	internal := []string{
		"kubectl.kubernetes.io/last-applied-configuration",
		"control-plane.alpha.kubernetes.io/leader",
		"deprecated.daemonset.template.generation",
	}
	allowed := []string{
		"app.kubernetes.io/name",
		"app.kubernetes.io/version",
		"service.beta.kubernetes.io/aws-load-balancer-type",
		"team",
		"env",
	}

	for _, key := range internal {
		key := key
		t.Run("internal/"+key, func(t *testing.T) {
			t.Parallel()
			if !isInternalMetadataKey(key) {
				t.Errorf("expected %q to be detected as internal, but it was not", key)
			}
		})
	}

	for _, key := range allowed {
		key := key
		t.Run("allowed/"+key, func(t *testing.T) {
			t.Parallel()
			if isInternalMetadataKey(key) {
				t.Errorf("expected %q to be allowed (not internal), but it was flagged as internal", key)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// JSON Patch helpers
// ---------------------------------------------------------------------------

// marshalOps marshals a PatchOperations slice to a slice of op descriptors for
// easy inspection inside tests.
type patchOp struct {
	op    string
	path  string
	value string // empty for remove; for object-value (whole-map add) use wantObj
}

func marshalOps(t *testing.T, ops kubernetes.PatchOperations) []patchOp {
	t.Helper()
	b, err := ops.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	var raw []struct {
		Op    string          `json:"op"`
		Path  string          `json:"path"`
		Value json.RawMessage `json:"value,omitempty"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	out := make([]patchOp, 0, len(raw))
	for _, r := range raw {
		o := patchOp{op: r.Op, path: r.Path}
		if r.Value != nil {
			var s string
			if err := json.Unmarshal(r.Value, &s); err == nil {
				o.value = s
			}
		}
		out = append(out, o)
	}
	return out
}

func hasOp(got []patchOp, want patchOp) bool {
	for _, o := range got {
		if o.op == want.op && o.path == want.path {
			if want.value == "" || o.value == want.value {
				return true
			}
		}
	}
	return false
}

// TestEscapeJSONPointer verifies RFC 6901 escaping.
func TestEscapeJSONPointer(t *testing.T) {
	t.Parallel()

	cases := []struct{ in, want string }{
		{"plain", "plain"},
		{"a/b", "a~1b"},
		{"a~b", "a~0b"},
		{"a~/b", "a~0~1b"},
		{"kubectl.kubernetes.io/last-applied-configuration", "kubectl.kubernetes.io~1last-applied-configuration"},
	}
	for _, c := range cases {
		c := c
		t.Run(c.in, func(t *testing.T) {
			t.Parallel()
			got := escapeJSONPointer(c.in)
			if got != c.want {
				t.Errorf("escapeJSONPointer(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// TestDiffStringMap covers all patch operation variants.
func TestDiffStringMap(t *testing.T) {
	t.Parallel()

	const prefix = "/metadata/annotations"

	tests := []struct {
		name      string
		oldValues map[string]string
		newValues map[string]string
		wantOps   []patchOp   // each must be present
		wantNone  []patchOp   // none of these must appear
		wantLen   int         // exact op count, -1 to skip
	}{
		{
			name:      "add new key",
			oldValues: map[string]string{"existing": "v1"},
			newValues: map[string]string{"existing": "v1", "new": "v2"},
			wantOps:   []patchOp{{op: "add", path: prefix + "/new", value: "v2"}},
			wantNone:  []patchOp{{op: "remove", path: prefix + "/existing"}},
			wantLen:   1,
		},
		{
			name:      "replace changed value",
			oldValues: map[string]string{"env": "staging"},
			newValues: map[string]string{"env": "prod"},
			wantOps:   []patchOp{{op: "replace", path: prefix + "/env", value: "prod"}},
			wantLen:   1,
		},
		{
			name:      "remove deleted managed key",
			oldValues: map[string]string{"gone": "bye", "keep": "yes"},
			newValues: map[string]string{"keep": "yes"},
			wantOps:   []patchOp{{op: "remove", path: prefix + "/gone"}},
			wantNone:  []patchOp{{op: "remove", path: prefix + "/keep"}},
			wantLen:   1,
		},
		{
			name:      "no-op when maps are identical",
			oldValues: map[string]string{"k": "v"},
			newValues: map[string]string{"k": "v"},
			wantNone: []patchOp{
				{op: "add", path: prefix + "/k"},
				{op: "replace", path: prefix + "/k"},
				{op: "remove", path: prefix + "/k"},
			},
			wantLen: 0,
		},
		{
			// Key with slash: path must use ~1
			name:      "escape slash — replace",
			oldValues: map[string]string{"a/b": "old"},
			newValues: map[string]string{"a/b": "new"},
			wantOps:   []patchOp{{op: "replace", path: prefix + "/a~1b", value: "new"}},
			wantLen:   1,
		},
		{
			// Key with tilde: path must use ~0
			name:      "escape tilde — replace",
			oldValues: map[string]string{"a~b": "old"},
			newValues: map[string]string{"a~b": "new"},
			wantOps:   []patchOp{{op: "replace", path: prefix + "/a~0b", value: "new"}},
			wantLen:   1,
		},
		{
			// External key absent from both maps must never appear in ops.
			name:      "external key never removed",
			oldValues: map[string]string{"managed": "yes"},
			newValues: map[string]string{"managed": "yes"},
			wantNone:  []patchOp{{op: "remove", path: prefix + "/external"}},
			wantLen:   0,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ops := diffStringMap(prefix, tc.oldValues, tc.newValues)
			got := marshalOps(t, ops)

			for _, want := range tc.wantOps {
				if !hasOp(got, want) {
					t.Errorf("missing op %+v; got %+v", want, got)
				}
			}
			for _, none := range tc.wantNone {
				if hasOp(got, none) {
					t.Errorf("unexpected op %+v; got %+v", none, got)
				}
			}
			if tc.wantLen >= 0 && len(ops) != tc.wantLen {
				t.Errorf("len(ops) = %d, want %d; ops=%+v", len(ops), tc.wantLen, got)
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
