// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package storagev1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	storagev1 "github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/storagev1"
)

// ─── Acceptance Move Tests (moved block) ────────────────────────────────────

func testAccStorageClassMoveStateSource(config string) string {
	return strings.ReplaceAll(config, `"kubernetes_storage_class_v1"`, `"kubernetes_storage_class"`)
}

func testAccStorageClassMoveStateTarget(config string) string {
	return config + `
moved {
  from = kubernetes_storage_class.test
  to   = kubernetes_storage_class_v1.test
}
`
}

func testAccStorageClassMoveState(t *testing.T, sdkv2Version, config string) {
	t.Helper()

	tfresource.ParallelTest(t, tfresource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckStorageClassV1Destroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		Steps: []tfresource.TestStep{
			{
				ExternalProviders: map[string]tfresource.ExternalProvider{
					"kubernetes": {
						VersionConstraint: sdkv2Version,
						Source:            "hashicorp/kubernetes",
					},
				},
				Config: testAccStorageClassMoveStateSource(config),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccStorageClassMoveStateTarget(config),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// ─── Moved from SDKv2 (v3.2.1) ───────────────────────────────────────────────

func TestAccStorageClassV1_MoveStateFromUnversioned_basicName(t *testing.T) {
	name := fmt.Sprintf("tf-move-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccStorageClassMoveState(t, sdkv2ProviderVersion, testAccKubernetesStorageClassV1Config_basic(name))
}

func TestAccStorageClassV1_MoveStateFromUnversioned_generateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-move-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccStorageClassMoveState(t, sdkv2ProviderVersion, testAccKubernetesStorageClassV1Config_generateName(prefix))
}

func TestAccStorageClassV1_MoveStateFromUnversioned_annotations(t *testing.T) {
	name := fmt.Sprintf("tf-move-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccStorageClassMoveState(t, sdkv2ProviderVersion, testAccKubernetesStorageClassV1Config_annotations(name))
}

func TestAccStorageClassV1_MoveStateFromUnversioned_labels(t *testing.T) {
	name := fmt.Sprintf("tf-move-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccStorageClassMoveState(t, sdkv2ProviderVersion, testAccKubernetesStorageClassV1Config_labels(name))
}

func TestAccStorageClassV1_MoveStateFromUnversioned_completeName(t *testing.T) {
	name := fmt.Sprintf("tf-move-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccStorageClassMoveState(t, sdkv2ProviderVersion, testAccKubernetesStorageClassV1Config_completeName(name))
}

func TestAccStorageClassV1_MoveStateFromUnversioned_completeGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-move-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccStorageClassMoveState(t, sdkv2ProviderVersion, testAccKubernetesStorageClassV1Config_completeGenerateName(prefix))
}

// ─── Moved from SDKv2 Pre-Identity (v2.37.1) ─────────────────────────────────

func TestAccStorageClassV1_MoveStateFromUnversionedPreIdentity_basicName(t *testing.T) {
	name := fmt.Sprintf("tf-move-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccStorageClassMoveState(t, sdkv2PreIdentityProviderVersion,
		testAccKubernetesStorageClassV1Config_basic(name))
}

func TestAccStorageClassV1_MoveStateFromUnversionedPreIdentity_basicGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-move-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccStorageClassMoveState(t, sdkv2PreIdentityProviderVersion,
		testAccKubernetesStorageClassV1Config_generateName(prefix))
}

func TestAccStorageClassV1_MoveStateFromUnversionedPreIdentity_completeName(t *testing.T) {
	name := fmt.Sprintf("tf-move-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccStorageClassMoveState(t, sdkv2PreIdentityProviderVersion,
		testAccKubernetesStorageClassV1Config_completeName(name))
}

func TestAccStorageClassV1_MoveStateFromUnversionedPreIdentity_completeGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-move-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccStorageClassMoveState(t, sdkv2PreIdentityProviderVersion,
		testAccKubernetesStorageClassV1Config_completeGenerateName(prefix))
}

// ─── MoveState Unit Tests ───────────────────────────────────────────────────

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

	// Parameters is now types.Map — extract value via ElementsAs.
	if got.Parameters.IsNull() || got.Parameters.IsUnknown() {
		t.Fatal("parameters: expected non-null map")
	}
	var params map[string]string
	resp.Diagnostics.Append(got.Parameters.ElementsAs(context.Background(), &params, false)...)
	if resp.Diagnostics.HasError() {
		t.Fatalf("parameters ElementsAs: %s", resp.Diagnostics)
	}
	if params["type"] != "pd-ssd" {
		t.Errorf("parameters.type: got %q, want pd-ssd", params["type"])
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

	// Annotations and Labels are now types.Map — check via Elements().
	annotations := got.Metadata[0].Annotations
	if annotations.IsNull() || annotations.IsUnknown() {
		t.Fatal("annotations: expected non-null map")
	}
	annotationElems := annotations.Elements()
	noteVal, ok := annotationElems["example.com/note"]
	if !ok {
		t.Error("annotations: key 'example.com/note' not found")
	} else if s, ok := noteVal.(types.String); !ok || s.ValueString() != "test" {
		t.Errorf("annotation example.com/note: got %v, want 'test'", noteVal)
	}

	labels := got.Metadata[0].Labels
	if labels.IsNull() || labels.IsUnknown() {
		t.Fatal("labels: expected non-null map")
	}
	labelElems := labels.Elements()
	managedByVal, ok := labelElems["managed-by"]
	if !ok {
		t.Error("labels: key 'managed-by' not found")
	} else if s, ok := managedByVal.(types.String); !ok || s.ValueString() != "terraform" {
		t.Errorf("label managed-by: got %v, want 'terraform'", managedByVal)
	}
}

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

// TestMigration_MoveState_nullAnnotationsAndLabelsAreNull verifies that nil
// (JSON null) maps from SDKv2 state become null types.Map values, not empty maps.
// SDKv2 writes null for unset annotations/labels; MoveState must preserve this.
func TestMigration_MoveState_nullAnnotationsAndLabelsAreNull(t *testing.T) {
	t.Parallel()

	raw := sdkv2SCRawJSON(
		"sc-nil-meta", "sc-nil-meta", "",
		nil, nil, // nil maps → JSON null
		"1", "uid-3", 0,
		"rancher.io/local-path", nil,
		"Delete", "Immediate", true,
		nil, nil,
	)

	resp := runMoveState(t, "kubernetes_storage_class", raw)
	got := readMovedModel(t, resp)

	if !got.Metadata[0].Annotations.IsNull() {
		t.Errorf("annotations: expected null for JSON null, got %v", got.Metadata[0].Annotations)
	}
	if !got.Metadata[0].Labels.IsNull() {
		t.Errorf("labels: expected null for JSON null, got %v", got.Metadata[0].Labels)
	}
}

// TestMigration_MoveState_emptyAnnotationsAndLabelsAreEmpty verifies that empty
// maps {} from SDKv2 state become known empty types.Map values (not null).
// SDKv2 writes {} when the user explicitly sets empty maps; MoveState preserves the distinction.
func TestMigration_MoveState_emptyAnnotationsAndLabelsAreEmpty(t *testing.T) {
	t.Parallel()

	raw := sdkv2SCRawJSON(
		"sc-empty-meta", "sc-empty-meta", "",
		map[string]string{}, map[string]string{}, // empty maps → JSON {}
		"1", "uid-3b", 0,
		"rancher.io/local-path", nil,
		"Delete", "Immediate", true,
		nil, nil,
	)

	resp := runMoveState(t, "kubernetes_storage_class", raw)
	got := readMovedModel(t, resp)

	// {} from SDKv2 becomes a known empty map, not null — preserving the distinction.
	if got.Metadata[0].Annotations.IsNull() {
		t.Error("annotations: expected known empty map for {}, got null")
	}
	if len(got.Metadata[0].Annotations.Elements()) != 0 {
		t.Errorf("annotations: expected 0 elements, got %d", len(got.Metadata[0].Annotations.Elements()))
	}
	if got.Metadata[0].Labels.IsNull() {
		t.Error("labels: expected known empty map for {}, got null")
	}
	if len(got.Metadata[0].Labels.Elements()) != 0 {
		t.Errorf("labels: expected 0 elements, got %d", len(got.Metadata[0].Labels.Elements()))
	}
}

func TestMigration_MoveState_emptyParametersAreNull(t *testing.T) {
	t.Parallel()

	raw := sdkv2SCRawJSON(
		"sc-no-params", "sc-no-params", "",
		nil, nil, "1", "uid-4", 0,
		"rancher.io/local-path",
		map[string]string{},
		"Delete", "Immediate", true,
		nil, nil,
	)

	resp := runMoveState(t, "kubernetes_storage_class", raw)
	got := readMovedModel(t, resp)

	// Empty parameters map → null (no user-configured parameters).
	if !got.Parameters.IsNull() {
		t.Errorf("parameters: expected null for empty map, got %v", got.Parameters)
	}
}

func TestMigration_MoveState_emptyMountOptionsIsNull(t *testing.T) {
	t.Parallel()

	raw := sdkv2SCRawJSON(
		"sc-no-mounts", "sc-no-mounts", "",
		nil, nil, "1", "uid-5", 0,
		"rancher.io/local-path", nil,
		"Delete", "Immediate", true,
		nil, nil,
	)

	resp := runMoveState(t, "kubernetes_storage_class", raw)
	got := readMovedModel(t, resp)

	if !got.MountOptions.IsNull() {
		t.Errorf("mount_options: expected null for empty slice, got %v", got.MountOptions)
	}
}

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

func TestMigration_MoveState_noAllowedTopologies(t *testing.T) {
	t.Parallel()

	raw := sdkv2SCRawJSON(
		"sc-no-topo", "sc-no-topo", "",
		nil, nil, "1", "uid-8", 0,
		"rancher.io/local-path", nil,
		"Delete", "Immediate", true,
		nil, nil,
	)

	resp := runMoveState(t, "kubernetes_storage_class", raw)
	got := readMovedModel(t, resp)

	if len(got.AllowedTopologies) != 0 {
		t.Errorf("allowed_topologies: expected empty, got %d elements", len(got.AllowedTopologies))
	}
}

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

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no errors for unrecognised source type, got: %s", resp.Diagnostics)
	}

	var m storagev1.StorageClassModel
	diags := resp.TargetState.Get(context.Background(), &m)
	if !diags.HasError() && m.ID.ValueString() != "" {
		t.Errorf("expected empty target state for unrecognised source type, got id=%q", m.ID.ValueString())
	}
}
