// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package storagev1_test

// Migration unit tests for kubernetes_storage_class_v1.
//
// These tests verify MoveState — the mechanism that allows users to migrate
// from the deprecated kubernetes_storage_class (SDKv2) resource to
// kubernetes_storage_class_v1 (Framework) using a `moved` block:
//
//	moved {
//	  from = kubernetes_storage_class.example
//	  to   = kubernetes_storage_class_v1.example
//	}
//
// No cluster is required — all tests run entirely in memory.
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

	storagev1 "github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/storagev1"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// sdkv2SCRawJSON produces raw JSON bytes that mirror what the SDKv2 provider
// writes into terraform.tfstate for kubernetes_storage_class.
func sdkv2SCRawJSON(
	id, name, generateName string,
	annotations, labels map[string]string,
	resourceVersion, uid string,
	generation int,
	provisioner string,
	parameters map[string]string,
	reclaimPolicy, volumeBindingMode string,
	allowVolumeExpansion bool,
	mountOptions []string,
	allowedTopologies []map[string]interface{},
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
		"id":                     id,
		"metadata":               []interface{}{meta},
		"storage_provisioner":    provisioner,
		"parameters":             parameters,
		"reclaim_policy":         reclaimPolicy,
		"volume_binding_mode":    volumeBindingMode,
		"allow_volume_expansion": allowVolumeExpansion,
		"mount_options":          mountOptions,
		"allowed_topologies":     allowedTopologies,
	}
	raw, _ := json.Marshal(state)
	return raw
}

// runMoveState invokes the registered MoveState handler with the given source
// type and raw JSON, returning the response for inspection.
func runMoveState(t *testing.T, sourceTypeName string, rawJSON []byte) *resource.MoveStateResponse {
	t.Helper()
	r := storagev1.NewStorageClassV1()
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
		TargetState: tfsdk.State{Schema: storagev1.StorageClassV1Schema()},
	}
	movers[0].StateMover(context.Background(), req, resp)
	return resp
}

// readMovedModel extracts the StorageClassModel from the MoveState response.
func readMovedModel(t *testing.T, resp *resource.MoveStateResponse) storagev1.StorageClassModel {
	t.Helper()
	if resp.Diagnostics.HasError() {
		t.Fatalf("move state produced errors: %s", resp.Diagnostics)
	}
	var m storagev1.StorageClassModel
	resp.Diagnostics.Append(resp.TargetState.Get(context.Background(), &m)...)
	if resp.Diagnostics.HasError() {
		t.Fatalf("reading moved state: %s", resp.Diagnostics)
	}
	return m
}

// ── tests ─────────────────────────────────────────────────────────────────────

// TestMigration_MoveState_handlersRegistered verifies the StateMover is wired
// up — a compile-time regression guard.
func TestMigration_MoveState_handlersRegistered(t *testing.T) {
	t.Parallel()
	r := storagev1.NewStorageClassV1()
	movers := r.(interface {
		MoveState(context.Context) []resource.StateMover
	}).MoveState(context.Background())

	if len(movers) == 0 {
		t.Error("expected at least 1 StateMover registered")
	}
}

// TestMigration_MoveState_basic verifies a full-featured state translation:
// all scalar fields, parameters, mount_options, and allowed_topologies.
func TestMigration_MoveState_basic(t *testing.T) {
	t.Parallel()

	topologies := []map[string]interface{}{
		{
			"match_label_expressions": []map[string]interface{}{
				{
					"key":    "topology.kubernetes.io/zone",
					"values": []string{"us-east-1a", "us-east-1b"},
				},
			},
		},
	}

	raw := sdkv2SCRawJSON(
		"my-sc", "my-sc", "",
		map[string]string{"example.com/note": "test"},
		map[string]string{"managed-by": "terraform"},
		"123456", "a6da86ec-b80d-44c0-9007-aafa4d982d4a", 1,
		"rancher.io/local-path",
		map[string]string{"type": "pd-ssd"},
		"Retain", "WaitForFirstConsumer", true,
		[]string{"noatime", "nodiratime"},
		topologies,
	)

	resp := runMoveState(t, "kubernetes_storage_class", raw)
	got := readMovedModel(t, resp)

	if got.ID.ValueString() != "my-sc" {
		t.Errorf("id: got %q, want my-sc", got.ID.ValueString())
	}
	if len(got.Metadata) != 1 {
		t.Fatalf("metadata: got %d elements, want 1", len(got.Metadata))
	}
	if got.Metadata[0].Name.ValueString() != "my-sc" {
		t.Errorf("name: got %q, want my-sc", got.Metadata[0].Name.ValueString())
	}
	if got.StorageProvisioner.ValueString() != "rancher.io/local-path" {
		t.Errorf("storage_provisioner: got %q, want rancher.io/local-path", got.StorageProvisioner.ValueString())
	}
	if got.ReclaimPolicy.ValueString() != "Retain" {
		t.Errorf("reclaim_policy: got %q, want Retain", got.ReclaimPolicy.ValueString())
	}
	if got.VolumeBindingMode.ValueString() != "WaitForFirstConsumer" {
		t.Errorf("volume_binding_mode: got %q, want WaitForFirstConsumer", got.VolumeBindingMode.ValueString())
	}
	if !got.AllowVolumeExpansion.ValueBool() {
		t.Error("allow_volume_expansion: expected true")
	}
	if got.Parameters["type"].ValueString() != "pd-ssd" {
		t.Errorf("parameters.type: got %q, want pd-ssd", got.Parameters["type"].ValueString())
	}
	if got.MountOptions.IsNull() || len(got.MountOptions.Elements()) != 2 {
		t.Errorf("mount_options: expected 2 elements, got %v", got.MountOptions)
	}
	if len(got.AllowedTopologies) != 1 {
		t.Fatalf("allowed_topologies: expected 1, got %d", len(got.AllowedTopologies))
	}
	if len(got.AllowedTopologies[0].MatchLabelExpressions) != 1 {
		t.Fatalf("match_label_expressions: expected 1, got %d", len(got.AllowedTopologies[0].MatchLabelExpressions))
	}
	expr := got.AllowedTopologies[0].MatchLabelExpressions[0]
	if expr.Key.ValueString() != "topology.kubernetes.io/zone" {
		t.Errorf("topology key: got %q, want topology.kubernetes.io/zone", expr.Key.ValueString())
	}
	if len(expr.Values.Elements()) != 2 {
		t.Errorf("topology values: expected 2, got %d", len(expr.Values.Elements()))
	}
	if got.Metadata[0].Annotations["example.com/note"].ValueString() != "test" {
		t.Errorf("annotation: got %q, want test", got.Metadata[0].Annotations["example.com/note"].ValueString())
	}
	if got.Metadata[0].Labels["managed-by"].ValueString() != "terraform" {
		t.Errorf("label: got %q, want terraform", got.Metadata[0].Labels["managed-by"].ValueString())
	}
}

// TestMigration_MoveState_emptyGenerateNameIsNull verifies that an empty
// generate_name from SDKv2 state is normalised to null to prevent plan drift.
func TestMigration_MoveState_emptyGenerateNameIsNull(t *testing.T) {
	t.Parallel()

	raw := sdkv2SCRawJSON(
		"sc-no-gen", "sc-no-gen", "",
		nil, nil, "1", "uid-1", 0,
		"rancher.io/local-path", nil,
		"Delete", "Immediate", true,
		nil, nil,
	)

	resp := runMoveState(t, "kubernetes_storage_class", raw)
	got := readMovedModel(t, resp)

	if !got.Metadata[0].GenerateName.IsNull() {
		t.Errorf("generate_name: expected null for empty string, got %q",
			got.Metadata[0].GenerateName.ValueString())
	}
}

// TestMigration_MoveState_nonEmptyGenerateNamePreserved verifies that a set
// generate_name prefix is preserved as-is after the move.
func TestMigration_MoveState_nonEmptyGenerateNamePreserved(t *testing.T) {
	t.Parallel()

	raw := sdkv2SCRawJSON(
		"sc-gen-xk9p2", "", "sc-gen-",
		nil, nil, "2", "uid-2", 0,
		"rancher.io/local-path", nil,
		"Delete", "Immediate", true,
		nil, nil,
	)

	resp := runMoveState(t, "kubernetes_storage_class", raw)
	got := readMovedModel(t, resp)

	if got.Metadata[0].GenerateName.IsNull() {
		t.Error("generate_name: expected non-null for 'sc-gen-', got null")
	}
	if got.Metadata[0].GenerateName.ValueString() != "sc-gen-" {
		t.Errorf("generate_name: got %q, want sc-gen-",
			got.Metadata[0].GenerateName.ValueString())
	}
}

// TestMigration_MoveState_emptyAnnotationsAndLabelsAreNil verifies that empty
// annotation and label maps become nil in the Framework model, preventing a
// perpetual plan diff against configs that omit them entirely.
func TestMigration_MoveState_emptyAnnotationsAndLabelsAreNil(t *testing.T) {
	t.Parallel()

	raw := sdkv2SCRawJSON(
		"sc-empty-meta", "sc-empty-meta", "",
		map[string]string{}, map[string]string{},
		"1", "uid-3", 0,
		"rancher.io/local-path", nil,
		"Delete", "Immediate", true,
		nil, nil,
	)

	resp := runMoveState(t, "kubernetes_storage_class", raw)
	got := readMovedModel(t, resp)

	if got.Metadata[0].Annotations != nil {
		t.Errorf("annotations: expected nil for empty map, got %v", got.Metadata[0].Annotations)
	}
	if got.Metadata[0].Labels != nil {
		t.Errorf("labels: expected nil for empty map, got %v", got.Metadata[0].Labels)
	}
}

// TestMigration_MoveState_emptyParametersAreNil verifies that an empty
// parameters map from SDKv2 state becomes nil to prevent plan drift.
func TestMigration_MoveState_emptyParametersAreNil(t *testing.T) {
	t.Parallel()

	raw := sdkv2SCRawJSON(
		"sc-no-params", "sc-no-params", "",
		nil, nil, "1", "uid-4", 0,
		"rancher.io/local-path",
		map[string]string{}, // empty parameters
		"Delete", "Immediate", true,
		nil, nil,
	)

	resp := runMoveState(t, "kubernetes_storage_class", raw)
	got := readMovedModel(t, resp)

	if got.Parameters != nil {
		t.Errorf("parameters: expected nil for empty map, got %v", got.Parameters)
	}
}

// TestMigration_MoveState_emptyMountOptionsIsNull verifies that a nil or
// empty mount_options slice becomes a null types.Set so that the moved state
// matches a Framework config that omits mount_options entirely, preventing
// the "was null, but now empty set" plan inconsistency after the move.
func TestMigration_MoveState_emptyMountOptionsIsNull(t *testing.T) {
	t.Parallel()

	raw := sdkv2SCRawJSON(
		"sc-no-mounts", "sc-no-mounts", "",
		nil, nil, "1", "uid-5", 0,
		"rancher.io/local-path", nil,
		"Delete", "Immediate", true,
		nil, nil, // empty mount_options
	)

	resp := runMoveState(t, "kubernetes_storage_class", raw)
	got := readMovedModel(t, resp)

	if !got.MountOptions.IsNull() {
		t.Errorf("mount_options: expected null for empty slice, got %v", got.MountOptions)
	}
}

// TestMigration_MoveState_reclaimPolicyDefaultsToDelete verifies that an
// empty reclaim_policy string is normalised to "Delete".
func TestMigration_MoveState_reclaimPolicyDefaultsToDelete(t *testing.T) {
	t.Parallel()

	raw := sdkv2SCRawJSON(
		"sc-no-reclaim", "sc-no-reclaim", "",
		nil, nil, "1", "uid-6", 0,
		"rancher.io/local-path", nil,
		"" /* empty reclaim_policy */, "Immediate", true,
		nil, nil,
	)

	resp := runMoveState(t, "kubernetes_storage_class", raw)
	got := readMovedModel(t, resp)

	if got.ReclaimPolicy.ValueString() != "Delete" {
		t.Errorf("reclaim_policy: got %q, want Delete", got.ReclaimPolicy.ValueString())
	}
}

// TestMigration_MoveState_volumeBindingModeDefaultsToImmediate verifies that
// an empty volume_binding_mode string is normalised to "Immediate".
func TestMigration_MoveState_volumeBindingModeDefaultsToImmediate(t *testing.T) {
	t.Parallel()

	raw := sdkv2SCRawJSON(
		"sc-no-binding", "sc-no-binding", "",
		nil, nil, "1", "uid-7", 0,
		"rancher.io/local-path", nil,
		"Delete", "" /* empty volume_binding_mode */, true,
		nil, nil,
	)

	resp := runMoveState(t, "kubernetes_storage_class", raw)
	got := readMovedModel(t, resp)

	if got.VolumeBindingMode.ValueString() != "Immediate" {
		t.Errorf("volume_binding_mode: got %q, want Immediate", got.VolumeBindingMode.ValueString())
	}
}

// TestMigration_MoveState_noAllowedTopologies verifies that absent
// allowed_topologies produces a nil slice (not an empty block list).
func TestMigration_MoveState_noAllowedTopologies(t *testing.T) {
	t.Parallel()

	raw := sdkv2SCRawJSON(
		"sc-no-topo", "sc-no-topo", "",
		nil, nil, "1", "uid-8", 0,
		"rancher.io/local-path", nil,
		"Delete", "Immediate", true,
		nil, nil, // no allowed_topologies
	)

	resp := runMoveState(t, "kubernetes_storage_class", raw)
	got := readMovedModel(t, resp)

	if len(got.AllowedTopologies) != 0 {
		t.Errorf("allowed_topologies: expected empty, got %d elements", len(got.AllowedTopologies))
	}
}

// TestMigration_MoveState_wrongSourceTypeIsIgnored verifies that the handler
// returns early without error or writing any state when SourceTypeName does
// not match kubernetes_storage_class.
func TestMigration_MoveState_wrongSourceTypeIsIgnored(t *testing.T) {
	t.Parallel()

	raw := sdkv2SCRawJSON(
		"some-other", "some-other", "",
		nil, nil, "1", "uid-9", 0,
		"rancher.io/local-path", nil,
		"Delete", "Immediate", true,
		nil, nil,
	)

	resp := runMoveState(t, "kubernetes_some_other_resource", raw)

	// No errors expected — handler must silently return.
	if resp.Diagnostics.HasError() {
		t.Errorf("expected no errors for unrecognised source type, got: %s", resp.Diagnostics)
	}

	// TargetState must be empty — handler must not have written anything.
	var m storagev1.StorageClassModel
	diags := resp.TargetState.Get(context.Background(), &m)
	if !diags.HasError() && m.ID.ValueString() != "" {
		t.Errorf("expected empty target state for unrecognised source type, got id=%q", m.ID.ValueString())
	}
}
