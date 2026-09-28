// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	api "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

			current, diags := types.MapValueFrom(context.Background(), types.StringType, tc.current)
			if diags.HasError() {
				t.Fatal(diags)
			}
			gotMeta, diags := common.FlattenBaseMetadata(context.Background(),
				metav1.ObjectMeta{Labels: tc.input}, common.MetadataBase{Labels: current}, nil, tc.ignorePatterns)
			if diags.HasError() {
				t.Fatal(diags)
			}
			got := gotMeta.Labels.Elements()

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
			meta, diags := common.FlattenBaseMetadata(context.Background(),
				metav1.ObjectMeta{Labels: map[string]string{key: "value"}}, common.MetadataBase{}, nil, nil)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if len(meta.Labels.Elements()) != 0 {
				t.Errorf("expected %q to be detected as internal, but it was not", key)
			}
		})
	}

	for _, key := range allowed {
		key := key
		t.Run("allowed/"+key, func(t *testing.T) {
			t.Parallel()
			meta, diags := common.FlattenBaseMetadata(context.Background(),
				metav1.ObjectMeta{Labels: map[string]string{key: "value"}}, common.MetadataBase{}, nil, nil)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if len(meta.Labels.Elements()) != 1 {
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
			ops := kubernetes.DiffStringMap("/metadata/annotations",
				map[string]interface{}{c.in: "old"}, map[string]interface{}{c.in: "new"})
			got := marshalOps(t, ops)
			if len(got) != 1 || got[0].path != "/metadata/annotations/"+c.want {
				t.Errorf("escaped patch path for %q = %v, want %q", c.in, got, c.want)
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
		wantOps   []patchOp // each must be present
		wantNone  []patchOp // none of these must appear
		wantLen   int       // exact op count, -1 to skip
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
			oldValues, diags := types.MapValueFrom(context.Background(), types.StringType, tc.oldValues)
			if diags.HasError() {
				t.Fatal(diags)
			}
			newValues, diags := types.MapValueFrom(context.Background(), types.StringType, tc.newValues)
			if diags.HasError() {
				t.Fatal(diags)
			}
			ops := common.BaseMetadataPatchOps("/metadata/",
				common.MetadataBase{Annotations: oldValues}, common.MetadataBase{Annotations: newValues})
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

func TestClusterRoleBindingGenerateNameWithoutRefresh(t *testing.T) {
	ctx := context.Background()
	r := &ClusterRoleBinding{}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	model := ClusterRoleBindingModel{
		ID: types.StringValue("binding"),
		Metadata: []common.MetadataModel{{
			MetadataBase: common.MetadataBase{
				Name:        types.StringValue("binding"),
				Labels:      types.MapNull(types.StringType),
				Annotations: types.MapNull(types.StringType),
			},
			GenerateName: types.StringValue(""),
		}},
	}
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(ctx, model); diags.HasError() {
		t.Fatal(diags)
	}
	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	model.Metadata[0].GenerateName = types.StringNull()
	if diags := plan.Set(ctx, model); diags.HasError() {
		t.Fatal(diags)
	}
	block := schemaResp.Schema.Blocks["metadata"].(schema.ListNestedBlock)
	attr := block.NestedObject.Attributes["generate_name"].(schema.StringAttribute)
	for _, tc := range []struct {
		state   types.String
		replace bool
	}{
		{types.StringValue(""), false},
		{types.StringValue("real-prefix-"), true},
	} {
		req := planmodifier.StringRequest{
			State: state, Plan: plan, StateValue: tc.state, PlanValue: types.StringNull(),
		}
		resp := planmodifier.StringResponse{PlanValue: req.PlanValue}
		for _, modifier := range attr.PlanModifiers {
			modifier.PlanModifyString(ctx, req, &resp)
		}
		if resp.Diagnostics.HasError() || resp.RequiresReplace != tc.replace {
			t.Fatalf("state %s: replacement=%t, want %t; diagnostics=%v",
				tc.state, resp.RequiresReplace, tc.replace, resp.Diagnostics)
		}
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
