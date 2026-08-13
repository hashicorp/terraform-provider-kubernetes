// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package schedulingv1_test

// Migration tests for kubernetes_priority_class_v1.
//
// These tests verify that switching from the SDKv2 implementation to the
// Plugin Framework implementation does not alter provider behaviour.
// They follow the HashiCorp migration testing guide:
// https://developer.hashicorp.com/terraform/plugin/framework/migrating/testing
//
// Two categories are covered:
//   A. UpgradeState (v0 → v1): SDKv2 kubernetes_priority_class_v1 state at
//      schema version 0 is transparently upgraded to version 1 the first time
//      the Framework provider reads it.
//   B. MoveState: SDKv2 kubernetes_priority_class (deprecated alias) state is
//      translated into kubernetes_priority_class_v1 state when the user adds a
//      `moved` block.

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	schedulingv1 "github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/schedulingv1"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

// metaObjType is the tftypes shape of one SDKv2 TypeList metadata element.
var metaObjType = tftypes.Object{
	AttributeTypes: map[string]tftypes.Type{
		"name":             tftypes.String,
		"generate_name":    tftypes.String,
		"annotations":      tftypes.Map{ElementType: tftypes.String},
		"labels":           tftypes.Map{ElementType: tftypes.String},
		"resource_version": tftypes.String,
		"uid":              tftypes.String,
		"generation":       tftypes.Number,
	},
}

// v0StateType is the full tftypes shape of an SDKv2 priority_class_v1 state.
var v0StateType = tftypes.Object{
	AttributeTypes: map[string]tftypes.Type{
		"id":                tftypes.String,
		"value":             tftypes.Number,
		"description":       tftypes.String,
		"global_default":    tftypes.Bool,
		"preemption_policy": tftypes.String,
		"metadata":          tftypes.List{ElementType: metaObjType},
	},
}

// buildV0Value constructs a tftypes.Value representing SDKv2 schema version 0
// state. Pass empty strings for optional string fields to simulate omission.
func buildV0Value(id, name, generateName, description, preemptionPolicy string,
	value int, globalDefault bool,
	annotations, labels map[string]tftypes.Value,
	resourceVersion, uid string, generation int,
) tftypes.Value {
	return tftypes.NewValue(v0StateType, map[string]tftypes.Value{
		"id":                tftypes.NewValue(tftypes.String, id),
		"value":             tftypes.NewValue(tftypes.Number, value),
		"description":       tftypes.NewValue(tftypes.String, description),
		"global_default":    tftypes.NewValue(tftypes.Bool, globalDefault),
		"preemption_policy": tftypes.NewValue(tftypes.String, preemptionPolicy),
		"metadata": tftypes.NewValue(
			tftypes.List{ElementType: metaObjType},
			[]tftypes.Value{
				tftypes.NewValue(metaObjType, map[string]tftypes.Value{
					"name":             tftypes.NewValue(tftypes.String, name),
					"generate_name":    tftypes.NewValue(tftypes.String, generateName),
					"annotations":      tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, annotations),
					"labels":           tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, labels),
					"resource_version": tftypes.NewValue(tftypes.String, resourceVersion),
					"uid":              tftypes.NewValue(tftypes.String, uid),
					"generation":       tftypes.NewValue(tftypes.Number, generation),
				}),
			},
		),
	})
}

// runUpgrader retrieves the v0 upgrader and runs it against rawState,
// returning the response for assertion.
func runUpgrader(t *testing.T, rawState tfsdk.State) *resource.UpgradeStateResponse {
	t.Helper()
	r := schedulingv1.NewPriorityClassV1()
	upgraders := r.(interface {
		UpgradeState(context.Context) map[int64]resource.StateUpgrader
	}).UpgradeState(context.Background())

	upgrader, ok := upgraders[0]
	if !ok {
		t.Fatal("expected upgrader registered at key 0")
	}

	req := resource.UpgradeStateRequest{State: &rawState}
	resp := &resource.UpgradeStateResponse{
		State: tfsdk.State{Schema: schedulingv1.PriorityClassV1Schema()},
	}
	upgrader.StateUpgrader(context.Background(), req, resp)
	return resp
}

// readUpgradedModel reads the upgraded PriorityClassModel out of resp.State.
func readUpgradedModel(t *testing.T, resp *resource.UpgradeStateResponse) schedulingv1.PriorityClassModel {
	t.Helper()
	if resp.Diagnostics.HasError() {
		t.Fatalf("upgrade produced errors: %s", resp.Diagnostics)
	}
	var m schedulingv1.PriorityClassModel
	resp.Diagnostics.Append(resp.State.Get(context.Background(), &m)...)
	if resp.Diagnostics.HasError() {
		t.Fatalf("reading upgraded state: %s", resp.Diagnostics)
	}
	if len(m.Metadata) != 1 {
		t.Fatalf("metadata: got %d elements, want 1", len(m.Metadata))
	}
	return m
}

// sdkv2RawJSON produces raw JSON bytes that mirror what the SDKv2 provider
// writes into terraform.tfstate for kubernetes_priority_class. Used by
// MoveState tests which receive req.SourceRawState.JSON.
func sdkv2RawJSON(id, name, generateName, description, preemptionPolicy string,
	value int, globalDefault bool,
	annotations, labels map[string]string,
	resourceVersion, uid string, generation int,
) []byte {
	meta := map[string]interface{}{
		"name":             name,
		"generate_name":    generateName,
		"resource_version": resourceVersion,
		"uid":              uid,
		"generation":       generation,
		"annotations":      annotations,
		"labels":           labels,
	}
	state := map[string]interface{}{
		"id":               id,
		"value":            value,
		"description":      description,
		"global_default":   globalDefault,
		"preemption_policy": preemptionPolicy,
		"metadata":         []interface{}{meta},
	}
	raw, _ := json.Marshal(state)
	return raw
}

// runMoveState calls the MoveState handler with the given source type and raw JSON.
func runMoveState(t *testing.T, sourceTypeName string, rawJSON []byte) *resource.MoveStateResponse {
	t.Helper()
	r := schedulingv1.NewPriorityClassV1()
	movers := r.(interface {
		MoveState(context.Context) []resource.StateMover
	}).MoveState(context.Background())

	if len(movers) == 0 {
		t.Fatal("expected at least 1 StateMover")
	}

	req := resource.MoveStateRequest{
		SourceTypeName: sourceTypeName,
		SourceRawState: &tfprotov6.RawState{JSON: rawJSON},
	}
	resp := &resource.MoveStateResponse{
		TargetState: tfsdk.State{Schema: schedulingv1.PriorityClassV1Schema()},
	}
	movers[0].StateMover(context.Background(), req, resp)
	return resp
}

// readMovedModel reads the moved PriorityClassModel out of resp.TargetState.
func readMovedModel(t *testing.T, resp *resource.MoveStateResponse) schedulingv1.PriorityClassModel {
	t.Helper()
	if resp.Diagnostics.HasError() {
		t.Fatalf("move state produced errors: %s", resp.Diagnostics)
	}
	var m schedulingv1.PriorityClassModel
	resp.Diagnostics.Append(resp.TargetState.Get(context.Background(), &m)...)
	if resp.Diagnostics.HasError() {
		t.Fatalf("reading moved state: %s", resp.Diagnostics)
	}
	return m
}

// ─── A. UpgradeState tests (v0 → v1) ─────────────────────────────────────────

// TestMigration_UpgradeState_handlersRegistered verifies the upgrader is wired
// up at key 0 with a non-nil PriorSchema — a compile-time regression guard.
func TestMigration_UpgradeState_handlersRegistered(t *testing.T) {
	t.Parallel()
	r := schedulingv1.NewPriorityClassV1()
	upgraders := r.(interface {
		UpgradeState(context.Context) map[int64]resource.StateUpgrader
	}).UpgradeState(context.Background())

	upgrader, ok := upgraders[0]
	if !ok {
		t.Fatal("expected upgrader registered at key 0")
	}
	if len(upgraders) != 1 {
		t.Errorf("expected exactly 1 upgrader, got %d", len(upgraders))
	}
	if upgrader.PriorSchema == nil {
		t.Error("PriorSchema must be non-nil so the framework can decode prior state")
	}
}

// TestMigration_UpgradeState_basic covers the most common SDKv2 state: name
// only, no generate_name, empty annotations/labels. Verifies that
// generate_name is normalised to null (not "") to avoid perpetual plan diff.
func TestMigration_UpgradeState_basic(t *testing.T) {
	t.Parallel()

	v0 := buildV0Value(
		"my-pc", "my-pc", "",
		"", "PreemptLowerPriority",
		100, false,
		map[string]tftypes.Value{}, map[string]tftypes.Value{},
		"1000", "uid-001", 0,
	)
	r := schedulingv1.NewPriorityClassV1()
	upgraders := r.(interface {
		UpgradeState(context.Context) map[int64]resource.StateUpgrader
	}).UpgradeState(context.Background())
	upgrader := upgraders[0]

	resp := runUpgrader(t, tfsdk.State{Schema: *upgrader.PriorSchema, Raw: v0})
	got := readUpgradedModel(t, resp)

	if got.ID.ValueString() != "my-pc" {
		t.Errorf("id: got %q, want %q", got.ID.ValueString(), "my-pc")
	}
	if got.Metadata[0].Name.ValueString() != "my-pc" {
		t.Errorf("name: got %q, want %q", got.Metadata[0].Name.ValueString(), "my-pc")
	}
	// generate_name was "" in SDKv2 — must become null in Framework
	if !got.Metadata[0].GenerateName.IsNull() {
		t.Errorf("generate_name: expected null, got %q", got.Metadata[0].GenerateName.ValueString())
	}
	// empty maps must remain nil — not empty map — to avoid plan diff
	if got.Metadata[0].Annotations != nil {
		t.Errorf("annotations: expected nil for empty map, got %v", got.Metadata[0].Annotations)
	}
	if got.Metadata[0].Labels != nil {
		t.Errorf("labels: expected nil for empty map, got %v", got.Metadata[0].Labels)
	}
	if got.Value.ValueInt64() != 100 {
		t.Errorf("value: got %d, want 100", got.Value.ValueInt64())
	}
	if got.PreemptionPolicy.ValueString() != "PreemptLowerPriority" {
		t.Errorf("preemption_policy: got %q, want PreemptLowerPriority", got.PreemptionPolicy.ValueString())
	}
}

// TestMigration_UpgradeState_withAnnotationsAndLabels verifies that non-empty
// annotations and labels are preserved exactly after upgrade.
func TestMigration_UpgradeState_withAnnotationsAndLabels(t *testing.T) {
	t.Parallel()

	annotations := map[string]tftypes.Value{
		"example.com/note": tftypes.NewValue(tftypes.String, "hello"),
	}
	labels := map[string]tftypes.Value{
		"env": tftypes.NewValue(tftypes.String, "staging"),
	}
	v0 := buildV0Value(
		"pc-annot", "pc-annot", "",
		"some description", "Never",
		200, false,
		annotations, labels,
		"2000", "uid-002", 1,
	)
	r := schedulingv1.NewPriorityClassV1()
	upgraders := r.(interface {
		UpgradeState(context.Context) map[int64]resource.StateUpgrader
	}).UpgradeState(context.Background())
	upgrader := upgraders[0]

	resp := runUpgrader(t, tfsdk.State{Schema: *upgrader.PriorSchema, Raw: v0})
	got := readUpgradedModel(t, resp)

	if got.Description.ValueString() != "some description" {
		t.Errorf("description: got %q, want %q", got.Description.ValueString(), "some description")
	}
	if got.PreemptionPolicy.ValueString() != "Never" {
		t.Errorf("preemption_policy: got %q, want Never", got.PreemptionPolicy.ValueString())
	}
	if got.Metadata[0].Annotations["example.com/note"].ValueString() != "hello" {
		t.Errorf("annotations[example.com/note]: got %q, want %q",
			got.Metadata[0].Annotations["example.com/note"].ValueString(), "hello")
	}
	if got.Metadata[0].Labels["env"].ValueString() != "staging" {
		t.Errorf("labels[env]: got %q, want %q",
			got.Metadata[0].Labels["env"].ValueString(), "staging")
	}
}

// TestMigration_UpgradeState_withGenerateName verifies that a non-empty
// generate_name is preserved (not nulled out) after upgrade.
func TestMigration_UpgradeState_withGenerateName(t *testing.T) {
	t.Parallel()

	v0 := buildV0Value(
		"pc-gen-xk9p2", "", "pc-gen-",
		"", "PreemptLowerPriority",
		50, false,
		map[string]tftypes.Value{}, map[string]tftypes.Value{},
		"3000", "uid-003", 0,
	)
	r := schedulingv1.NewPriorityClassV1()
	upgraders := r.(interface {
		UpgradeState(context.Context) map[int64]resource.StateUpgrader
	}).UpgradeState(context.Background())
	upgrader := upgraders[0]

	resp := runUpgrader(t, tfsdk.State{Schema: *upgrader.PriorSchema, Raw: v0})
	got := readUpgradedModel(t, resp)

	if got.Metadata[0].GenerateName.IsNull() {
		t.Error("generate_name: expected non-null for 'pc-gen-', got null")
	}
	if got.Metadata[0].GenerateName.ValueString() != "pc-gen-" {
		t.Errorf("generate_name: got %q, want %q",
			got.Metadata[0].GenerateName.ValueString(), "pc-gen-")
	}
}

// TestMigration_UpgradeState_allScalarFields verifies description, global_default,
// value, resource_version and uid are all preserved correctly.
func TestMigration_UpgradeState_allScalarFields(t *testing.T) {
	t.Parallel()

	v0 := buildV0Value(
		"pc-full", "pc-full", "",
		"critical workloads", "Never",
		1000, true,
		map[string]tftypes.Value{}, map[string]tftypes.Value{},
		"9999", "uid-full", 3,
	)
	r := schedulingv1.NewPriorityClassV1()
	upgraders := r.(interface {
		UpgradeState(context.Context) map[int64]resource.StateUpgrader
	}).UpgradeState(context.Background())
	upgrader := upgraders[0]

	resp := runUpgrader(t, tfsdk.State{Schema: *upgrader.PriorSchema, Raw: v0})
	got := readUpgradedModel(t, resp)

	if got.Value.ValueInt64() != 1000 {
		t.Errorf("value: got %d, want 1000", got.Value.ValueInt64())
	}
	if !got.GlobalDefault.ValueBool() {
		t.Error("global_default: expected true")
	}
	if got.Description.ValueString() != "critical workloads" {
		t.Errorf("description: got %q, want %q", got.Description.ValueString(), "critical workloads")
	}
	if got.PreemptionPolicy.ValueString() != "Never" {
		t.Errorf("preemption_policy: got %q, want Never", got.PreemptionPolicy.ValueString())
	}
	if got.Metadata[0].ResourceVersion.ValueString() != "9999" {
		t.Errorf("resource_version: got %q, want 9999", got.Metadata[0].ResourceVersion.ValueString())
	}
	if got.Metadata[0].UID.ValueString() != "uid-full" {
		t.Errorf("uid: got %q, want uid-full", got.Metadata[0].UID.ValueString())
	}
	if got.Metadata[0].Generation.ValueInt64() != 3 {
		t.Errorf("generation: got %d, want 3", got.Metadata[0].Generation.ValueInt64())
	}
}

// ─── B. MoveState tests ───────────────────────────────────────────────────────

// TestMigration_MoveState_handlersRegistered verifies the StateMover is wired
// up — a compile-time regression guard.
func TestMigration_MoveState_handlersRegistered(t *testing.T) {
	t.Parallel()
	r := schedulingv1.NewPriorityClassV1()
	movers := r.(interface {
		MoveState(context.Context) []resource.StateMover
	}).MoveState(context.Background())

	if len(movers) == 0 {
		t.Error("expected at least 1 StateMover registered")
	}
}

// TestMigration_MoveState_basic verifies the core translation from
// kubernetes_priority_class raw JSON state to PriorityClassModel.
func TestMigration_MoveState_basic(t *testing.T) {
	t.Parallel()

	raw := sdkv2RawJSON(
		"my-pc", "my-pc", "",
		"Demo class", "PreemptLowerPriority",
		500, false,
		map[string]string{"example.com/note": "test"},
		map[string]string{"managed-by": "terraform"},
		"567273", "a6da86ec-b80d-44c0-9007-aafa4d982d4a", 1,
	)

	resp := runMoveState(t, "kubernetes_priority_class", raw)
	got := readMovedModel(t, resp)

	if got.ID.ValueString() != "my-pc" {
		t.Errorf("id: got %q, want my-pc", got.ID.ValueString())
	}
	if len(got.Metadata) != 1 {
		t.Fatalf("metadata: got %d elements, want 1", len(got.Metadata))
	}
	if got.Metadata[0].Name.ValueString() != "my-pc" {
		t.Errorf("name: got %q, want my-pc", got.Metadata[0].Name.ValueString())
	}
	if got.Value.ValueInt64() != 500 {
		t.Errorf("value: got %d, want 500", got.Value.ValueInt64())
	}
	if got.Description.ValueString() != "Demo class" {
		t.Errorf("description: got %q, want %q", got.Description.ValueString(), "Demo class")
	}
	if got.GlobalDefault.ValueBool() {
		t.Error("global_default: expected false")
	}
	if got.PreemptionPolicy.ValueString() != "PreemptLowerPriority" {
		t.Errorf("preemption_policy: got %q, want PreemptLowerPriority", got.PreemptionPolicy.ValueString())
	}
	if got.Metadata[0].Annotations["example.com/note"].ValueString() != "test" {
		t.Errorf("annotation: got %q, want test",
			got.Metadata[0].Annotations["example.com/note"].ValueString())
	}
	if got.Metadata[0].Labels["managed-by"].ValueString() != "terraform" {
		t.Errorf("label: got %q, want terraform",
			got.Metadata[0].Labels["managed-by"].ValueString())
	}
}

// TestMigration_MoveState_emptyGenerateName verifies that an empty string
// generate_name from SDKv2 is normalised to null to prevent plan drift.
func TestMigration_MoveState_emptyGenerateName(t *testing.T) {
	t.Parallel()

	raw := sdkv2RawJSON(
		"pc-no-gen", "pc-no-gen", "",
		"", "PreemptLowerPriority", 100, false,
		nil, nil, "1", "uid-1", 0,
	)

	resp := runMoveState(t, "kubernetes_priority_class", raw)
	got := readMovedModel(t, resp)

	if !got.Metadata[0].GenerateName.IsNull() {
		t.Errorf("generate_name: expected null for empty string, got %q",
			got.Metadata[0].GenerateName.ValueString())
	}
}

// TestMigration_MoveState_nonEmptyGenerateName verifies that a set
// generate_name is preserved as-is after the move.
func TestMigration_MoveState_nonEmptyGenerateName(t *testing.T) {
	t.Parallel()

	raw := sdkv2RawJSON(
		"pc-gen-xk9p2", "", "pc-gen-",
		"", "PreemptLowerPriority", 50, false,
		nil, nil, "2", "uid-2", 0,
	)

	resp := runMoveState(t, "kubernetes_priority_class", raw)
	got := readMovedModel(t, resp)

	if got.Metadata[0].GenerateName.IsNull() {
		t.Error("generate_name: expected non-null for 'pc-gen-', got null")
	}
	if got.Metadata[0].GenerateName.ValueString() != "pc-gen-" {
		t.Errorf("generate_name: got %q, want pc-gen-",
			got.Metadata[0].GenerateName.ValueString())
	}
}

// TestMigration_MoveState_emptyMapsAreNil verifies that empty annotation and
// label maps from SDKv2 become nil in the Framework model, preventing a
// perpetual plan diff against configs that omit annotations/labels entirely.
func TestMigration_MoveState_emptyMapsAreNil(t *testing.T) {
	t.Parallel()

	raw := sdkv2RawJSON(
		"pc-empty-maps", "pc-empty-maps", "",
		"", "PreemptLowerPriority", 100, false,
		map[string]string{}, map[string]string{},
		"1", "uid-3", 0,
	)

	resp := runMoveState(t, "kubernetes_priority_class", raw)
	got := readMovedModel(t, resp)

	if got.Metadata[0].Annotations != nil {
		t.Errorf("annotations: expected nil for empty map, got %v",
			got.Metadata[0].Annotations)
	}
	if got.Metadata[0].Labels != nil {
		t.Errorf("labels: expected nil for empty map, got %v",
			got.Metadata[0].Labels)
	}
}

// TestMigration_MoveState_missingPreemptionPolicyDefaultsToPreemptLowerPriority
// verifies that an empty preemption_policy string (which SDKv2 could produce
// on older state files) is defaulted to PreemptLowerPriority.
func TestMigration_MoveState_missingPreemptionPolicyDefaultsToPreemptLowerPriority(t *testing.T) {
	t.Parallel()

	raw := sdkv2RawJSON(
		"pc-no-policy", "pc-no-policy", "",
		"", "" /* empty preemption_policy */, 100, false,
		nil, nil, "1", "uid-4", 0,
	)

	resp := runMoveState(t, "kubernetes_priority_class", raw)
	got := readMovedModel(t, resp)

	if got.PreemptionPolicy.ValueString() != "PreemptLowerPriority" {
		t.Errorf("preemption_policy: got %q, want PreemptLowerPriority",
			got.PreemptionPolicy.ValueString())
	}
}

// TestMigration_MoveState_wrongSourceTypeIsIgnored verifies that the handler
// returns early without error or writing any state when SourceTypeName does
// not match kubernetes_priority_class. This ensures the handler is safe to
// call for any moved block in the configuration.
func TestMigration_MoveState_wrongSourceTypeIsIgnored(t *testing.T) {
	t.Parallel()

	raw := sdkv2RawJSON(
		"some-other", "some-other", "",
		"", "PreemptLowerPriority", 100, false,
		nil, nil, "1", "uid-5", 0,
	)

	resp := runMoveState(t, "kubernetes_some_other_resource", raw)

	// No errors expected — handler must silently return
	if resp.Diagnostics.HasError() {
		t.Errorf("expected no errors for unrecognised source type, got: %s",
			resp.Diagnostics)
	}

	// TargetState must be empty — handler must not have written anything
	var m schedulingv1.PriorityClassModel
	diags := resp.TargetState.Get(context.Background(), &m)
	if !diags.HasError() && m.ID.ValueString() != "" {
		t.Errorf("expected empty target state for unrecognised source type, got id=%q",
			m.ID.ValueString())
	}
}
