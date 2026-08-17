// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package schedulingv1_test

// Migration tests for kubernetes_priority_class_v1.
//
// These tests verify MoveState — the mechanism that allows users to migrate
// from the deprecated kubernetes_priority_class (SDKv2) resource to
// kubernetes_priority_class_v1 (Framework) using a `moved` block:
//
//	moved {
//	  from = kubernetes_priority_class.example
//	  to   = kubernetes_priority_class_v1.example
//	}
//
// No UpgradeState handler is needed because the Framework schema uses
// ListNestedBlock for metadata, which produces an identical JSON state shape
// to the SDKv2 TypeList{MaxItems:1} — schema_version stays at 0 and Terraform
// reads the existing state directly without any upgrade step.

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	schedulingv1 "github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/schedulingv1"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

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
		"id":                id,
		"value":             value,
		"description":       description,
		"global_default":    globalDefault,
		"preemption_policy": preemptionPolicy,
		"metadata":          []interface{}{meta},
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

// ─── MoveState tests ──────────────────────────────────────────────────────────

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
