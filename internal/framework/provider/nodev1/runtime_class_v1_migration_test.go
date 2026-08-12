// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package nodev1_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// sdkv2RuntimeClassProviderVersion is the last release that served
// kubernetes_runtime_class_v1 from the SDKv2 implementation.
// Step 1 of every migration test runs against it to write the old
// TypeList state shape onto disk.
const sdkv2RuntimeClassProviderVersion = "2.35.1"

// testAccRuntimeClassMigration is the shared helper used by every migration
// sub-test. It:
//
//	Step 1: applies sdkv2Config under the last SDKv2 provider —
//	        writes TypeList state:  "metadata": [{ "name": "x", ... }]
//
//	Step 2: re-plans frameworkConfig under the local Framework provider.
//	        UpgradeState() unwraps metadata[0] → metadata object.
//	        The plan MUST be empty — no destroy/recreate.
//
// Note: RuntimeClass uses block syntax in SDKv2 (metadata { name = "x" })
// but attribute syntax in the Framework (metadata = { name = "x" }).
// The two configs are NOT byte-identical between steps — unlike namespace.
func testAccRuntimeClassMigration(t *testing.T, sdkv2Config, frameworkConfig string) {
	t.Helper()

	resource.ParallelTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			// Step 1 — write SDKv2 TypeList state to disk.
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						VersionConstraint: sdkv2RuntimeClassProviderVersion,
						Source:            "hashicorp/kubernetes",
					},
				},
				Config: sdkv2Config,
			},
			// Step 2 — upgrade to the local Framework provider.
			// Terraform detects schema version mismatch and calls UpgradeState()
			// automatically. Plan must be empty after the upgrade.
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   frameworkConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// TestAccRuntimeClassV1_UpgradeFromSDKV2_basic — minimal shape: name only,
// no labels or annotations. The simplest migration path.
func TestAccRuntimeClassV1_UpgradeFromSDKV2_basic(t *testing.T) {
	name := fmt.Sprintf("tf-migration-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccRuntimeClassMigration(t,
		testAccRuntimeClassV1MigConfig_sdkv2(name, "", nil, nil),
		testAccRuntimeClassV1MigConfig_framework(name, "", nil, nil),
	)
}

// TestAccRuntimeClassV1_UpgradeFromSDKV2_labels — labels populated, annotations absent.
// Exercises the branch where Labels is non-nil but Annotations is nil after upgrade.
func TestAccRuntimeClassV1_UpgradeFromSDKV2_labels(t *testing.T) {
	name := fmt.Sprintf("tf-migration-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	labels := map[string]string{"env": "staging", "team": "platform"}
	testAccRuntimeClassMigration(t,
		testAccRuntimeClassV1MigConfig_sdkv2(name, "", labels, nil),
		testAccRuntimeClassV1MigConfig_framework(name, "", labels, nil),
	)
}

// TestAccRuntimeClassV1_UpgradeFromSDKV2_annotations — annotations populated, labels absent.
// Exercises the branch where Annotations is non-nil but Labels is nil after upgrade.
func TestAccRuntimeClassV1_UpgradeFromSDKV2_annotations(t *testing.T) {
	name := fmt.Sprintf("tf-migration-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	annotations := map[string]string{"owner": "team-a", "version": "v1"}
	testAccRuntimeClassMigration(t,
		testAccRuntimeClassV1MigConfig_sdkv2(name, "", nil, annotations),
		testAccRuntimeClassV1MigConfig_framework(name, "", nil, annotations),
	)
}

// TestAccRuntimeClassV1_UpgradeFromSDKV2_generateName — name is server-assigned.
// generate_name is the only shape where metadata.name is unknown at plan time,
// which exercises whether UseStateForUnknown is needed on the name attribute.
func TestAccRuntimeClassV1_UpgradeFromSDKV2_generateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccRuntimeClassMigration(t,
		testAccRuntimeClassV1MigConfig_sdkv2("", prefix, nil, nil),
		testAccRuntimeClassV1MigConfig_framework("", prefix, nil, nil),
	)
}

// TestAccRuntimeClassV1_UpgradeFromSDKV2_emptyMaps — both labels and annotations
// declared as explicit empty maps ({}).
//
// SDKv2 cannot represent the difference between null and a known empty map —
// it writes null to state for `labels = {}`. The Framework Read correctly
// preserves the null it was given. But the config still says `{}`, so Terraform
// plans an update to reconcile the two.
//
// This is NOT a bug — it is the correct and expected behaviour. The assertion
// reflects reality: one non-destructive update on first plan after upgrade.
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
				Config: testAccRuntimeClassV1MigConfig_emptyMaps_sdkv2(name),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccRuntimeClassV1MigConfig_emptyMaps_framework(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						// One update expected — SDKv2 null vs Framework empty map.
						// Non-destructive; idempotent after first apply.
						plancheck.ExpectResourceAction(
							"kubernetes_runtime_class_v1.migrate_test",
							plancheck.ResourceActionUpdate,
						),
					},
				},
			},
		},
	})
}

// ── SDKv2-style HCL config builders (block syntax — no = on metadata) ──────────

// testAccRuntimeClassV1MigConfig_sdkv2 builds HCL using old SDKv2 block syntax.
// Pass name="" to use generate_name instead. Pass nil maps to omit those fields.
func testAccRuntimeClassV1MigConfig_sdkv2(name, generateName string, labels, annotations map[string]string) string {
	var sb strings.Builder
	sb.WriteString("resource \"kubernetes_runtime_class_v1\" \"migrate_test\" {\n")
	sb.WriteString("  metadata {\n") // NO = sign — SDKv2 TypeList block syntax
	if name != "" {
		fmt.Fprintf(&sb, "    name = %q\n", name)
	}
	if generateName != "" {
		fmt.Fprintf(&sb, "    generate_name = %q\n", generateName)
	}
	if len(labels) > 0 {
		sb.WriteString("    labels = {\n")
		for k, v := range labels {
			fmt.Fprintf(&sb, "      %q = %q\n", k, v)
		}
		sb.WriteString("    }\n")
	}
	if len(annotations) > 0 {
		sb.WriteString("    annotations = {\n")
		for k, v := range annotations {
			fmt.Fprintf(&sb, "      %q = %q\n", k, v)
		}
		sb.WriteString("    }\n")
	}
	sb.WriteString("  }\n")
	sb.WriteString("  handler = \"runc\"\n")
	sb.WriteString("}\n")
	return sb.String()
}

// ── Framework-style HCL config builders (attribute syntax — = on metadata) ─────

// testAccRuntimeClassV1MigConfig_framework builds HCL using new Framework attribute syntax.
// Pass name="" to use generate_name instead. Pass nil maps to omit those fields.
func testAccRuntimeClassV1MigConfig_framework(name, generateName string, labels, annotations map[string]string) string {
	var sb strings.Builder
	sb.WriteString("resource \"kubernetes_runtime_class_v1\" \"migrate_test\" {\n")
	sb.WriteString("  metadata = {\n") // WITH = sign — Framework SingleNestedAttribute syntax
	if name != "" {
		fmt.Fprintf(&sb, "    name = %q\n", name)
	}
	if generateName != "" {
		fmt.Fprintf(&sb, "    generate_name = %q\n", generateName)
	}
	if len(labels) > 0 {
		sb.WriteString("    labels = {\n")
		for k, v := range labels {
			fmt.Fprintf(&sb, "      %q = %q\n", k, v)
		}
		sb.WriteString("    }\n")
	}
	if len(annotations) > 0 {
		sb.WriteString("    annotations = {\n")
		for k, v := range annotations {
			fmt.Fprintf(&sb, "      %q = %q\n", k, v)
		}
		sb.WriteString("    }\n")
	}
	sb.WriteString("  }\n")
	sb.WriteString("  handler = \"runc\"\n")
	sb.WriteString("}\n")
	return sb.String()
}

func testAccRuntimeClassV1MigConfig_emptyMaps_sdkv2(name string) string {
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

func testAccRuntimeClassV1MigConfig_emptyMaps_framework(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_runtime_class_v1" "migrate_test" {
  metadata = {
    name        = %q
    labels      = {}
    annotations = {}
  }
  handler = "runc"
}
`, name)
}
