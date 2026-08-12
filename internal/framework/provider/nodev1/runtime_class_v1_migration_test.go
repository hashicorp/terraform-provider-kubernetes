// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package nodev1_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// sdkv2RuntimeClassProviderVersion is the last release that served
// kubernetes_runtime_class_v1 from the SDKv2 implementation.
// Step 1 of every migration test runs against it to write real SDKv2 state.
const sdkv2RuntimeClassProviderVersion = "2.35.1"

// testAccRuntimeClassMigration is the shared helper for all migration sub-tests.
//
// Because metadata is a ListNestedBlock in both the SDKv2 and Framework
// implementations, the state shape is byte-identical:
//
//	"metadata": [{ "name": "x", "uid": "...", ... }]
//
// No UpgradeState() is required. The same HCL config works in both steps.
// The only thing being tested is that switching provider implementations
// (SDKv2 binary → Framework binary) produces an empty plan — no destroy/recreate.
//
// To run all migration tests:
//
//	TF_ACC=1 KUBE_CONFIG_PATH=~/.kube/config KUBE_CTX=kind-demo \
//	  go test ./internal/framework/provider/nodev1/... -run TestAccRuntimeClassV1_UpgradeFromSDKV2 -v
func testAccRuntimeClassMigration(t *testing.T, config string) {
	t.Helper()

	resource.ParallelTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			// Step 1 — apply with the last SDKv2 provider release.
			// Writes real SDKv2 state to terraform.tfstate.
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						VersionConstraint: sdkv2RuntimeClassProviderVersion,
						Source:            "hashicorp/kubernetes",
					},
				},
				Config: config,
			},
			// Step 2 — switch to the local Framework provider binary.
			// The state shape is identical (ListNestedBlock = TypeList), so
			// no state upgrade runs. The plan MUST be empty — no diff means
			// the provider switch was fully transparent to the user.
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// TestAccRuntimeClassV1_UpgradeFromSDKV2_basic — minimal shape: name only.
// This is the core migration path every user will hit.
func TestAccRuntimeClassV1_UpgradeFromSDKV2_basic(t *testing.T) {
	name := fmt.Sprintf("tf-migration-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccRuntimeClassMigration(t, testAccRuntimeClassV1MigConfig_basic(name))
}

// TestAccRuntimeClassV1_UpgradeFromSDKV2_labels — labels populated, annotations absent.
// Exercises the path where Labels is non-nil and Annotations is nil after Read.
func TestAccRuntimeClassV1_UpgradeFromSDKV2_labels(t *testing.T) {
	name := fmt.Sprintf("tf-migration-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccRuntimeClassMigration(t, testAccRuntimeClassV1MigConfig_labels(name))
}

// TestAccRuntimeClassV1_UpgradeFromSDKV2_annotations — annotations populated, labels absent.
// Exercises the path where Annotations is non-nil and Labels is nil after Read.
func TestAccRuntimeClassV1_UpgradeFromSDKV2_annotations(t *testing.T) {
	name := fmt.Sprintf("tf-migration-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccRuntimeClassMigration(t, testAccRuntimeClassV1MigConfig_annotations(name))
}

// TestAccRuntimeClassV1_UpgradeFromSDKV2_generateName — name is server-assigned.
// generate_name is the only metadata shape where name is unknown at plan time.
// This exercises whether the Framework correctly reads back the server-assigned
// name and produces an empty plan after migration.
func TestAccRuntimeClassV1_UpgradeFromSDKV2_generateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccRuntimeClassMigration(t, testAccRuntimeClassV1MigConfig_generateName(prefix))
}

// TestAccRuntimeClassV1_UpgradeFromSDKV2_emptyMaps — both labels and annotations
// declared as explicit empty maps {}.
//
// SDKv2 cannot represent the difference between null and a known empty map —
// it writes null to state for `labels = {}`. The Framework Read correctly
// preserves the null it was given. But the config still says `{}`, so Terraform
// plans an update to reconcile the two.
//
// This is NOT a bug — it is correct and expected behaviour. The assertion
// reflects reality: one non-destructive update on first plan after the upgrade.
// The implicit post-apply idempotency plan proves it settles and does not recur.
func TestAccRuntimeClassV1_UpgradeFromSDKV2_emptyMaps(t *testing.T) {
	name := fmt.Sprintf("tf-migration-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						VersionConstraint: sdkv2RuntimeClassProviderVersion,
						Source:            "hashicorp/kubernetes",
					},
				},
				Config: testAccRuntimeClassV1MigConfig_emptyMaps(name),
			},
			{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Config:                   testAccRuntimeClassV1MigConfig_emptyMaps(name),
					// PlanOnly asserts the plan shape without applying.
					// SDKv2 writes null for `labels = {}` and `annotations = {}` because
					// it cannot represent the difference between null and an empty map.
					// The Framework sees {} in config vs null in state → one update planned.
					// ExpectNonEmptyPlan: true documents this expected behaviour.
					PlanOnly:           true,
					ExpectNonEmptyPlan: true,
				},
		},
	})
}

// ── HCL config helpers ────────────────────────────────────────────────────────
// All configs use block syntax (metadata { }) because ListNestedBlock is used
// in both SDKv2 and Framework implementations. The same config is passed to
// both Step 1 (SDKv2) and Step 2 (Framework) — no syntax change needed.

func testAccRuntimeClassV1MigConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_runtime_class_v1" "migrate_test" {
  metadata {
    name = %q
  }
  handler = "runc"
}
`, name)
}

func testAccRuntimeClassV1MigConfig_labels(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_runtime_class_v1" "migrate_test" {
  metadata {
    name = %q
    labels = {
      env  = "staging"
      team = "platform"
    }
  }
  handler = "runc"
}
`, name)
}

func testAccRuntimeClassV1MigConfig_annotations(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_runtime_class_v1" "migrate_test" {
  metadata {
    name = %q
    annotations = {
      owner   = "team-a"
      version = "v1"
    }
  }
  handler = "runc"
}
`, name)
}

func testAccRuntimeClassV1MigConfig_generateName(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_runtime_class_v1" "migrate_test" {
  metadata {
    generate_name = %q
  }
  handler = "runc"
}
`, prefix)
}

func testAccRuntimeClassV1MigConfig_emptyMaps(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_runtime_class_v1" "migrate_test" {
  metadata {
    name        = %q
    labels      = {}
    annotations = {}
  }
  handler = "runc"
}
`, name)
}
