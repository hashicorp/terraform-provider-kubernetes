// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// Migration tests: state written by the last SDKv2 release of kubernetes_config_map_v1 must
// plan clean under the Framework resource, with no replacement and no spurious update.
//
// Step 2 refreshes that state through the Framework Read, so these exercise the flattening
// of metadata, data and binary_data and the null-versus-empty handling, not Create. The
// two steps use byte-identical configs, so a non-empty plan can only come from the upgrade.
//
// sdkv2ProviderVersion and sdkv2PreIdentityProviderVersion are shared with
// namespace_v1_migration_test.go.

func testAccConfigMapMigrationFrom(t *testing.T, sdkv2Version, config string) {
	t.Helper()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		// Without this a failed run leaks ConfigMaps, and the next run collides on the
		// same names.
		CheckDestroy: testAccCheckKubernetesConfigMapV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {VersionConstraint: sdkv2Version, Source: "hashicorp/kubernetes"},
				},
				Config: config,
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func testAccConfigMapMigrationName() string {
	return fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
}

// Name, both metadata maps and data: the baseline shape.
func TestAccConfigMap_UpgradeFromSDKV2(t *testing.T) {
	testAccConfigMapMigrationFrom(t, sdkv2ProviderVersion, testAccConfigMapV1Config_basic(testAccConfigMapMigrationName()))
}

// Name only: data, binary_data, labels and annotations all stored null.
func TestAccConfigMap_UpgradeFromSDKV2_noData(t *testing.T) {
	testAccConfigMapMigrationFrom(t, sdkv2ProviderVersion, testAccConfigMapV1Config_noData(testAccConfigMapMigrationName()))
}

// metadata.name is server-assigned, the only shape where a metadata value is unknown at
// plan time. SDKv2 also stored generate_name, which the Framework must carry across.
func TestAccConfigMap_UpgradeFromSDKV2_generateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccConfigMapMigrationFrom(t, sdkv2ProviderVersion, testAccConfigMapV1Config_generatedName(prefix))
}

// binary_data: SDKv2 stored the base64 text; the Framework re-encodes the API's bytes on
// Read and must land on the same string.
func TestAccConfigMap_UpgradeFromSDKV2_binaryData(t *testing.T) {
	testAccConfigMapMigrationFrom(t, sdkv2ProviderVersion,
		testAccConfigMapV1Config_binaryData2("tf-migration-test-gen-", testAccConfigMapFixtureDir(t)))
}

// immutable = true: the Framework's replacement rule must not fire on an unchanged config.
func TestAccConfigMap_UpgradeFromSDKV2_immutable(t *testing.T) {
	testAccConfigMapMigrationFrom(t, sdkv2ProviderVersion,
		testAccConfigMapV1Config_immutable(testAccConfigMapMigrationName(), true, "second"))
}

func TestAccConfigMap_UpgradeFromSDKV2_namespace(t *testing.T) {
	testAccConfigMapMigrationFrom(t, sdkv2ProviderVersion,
		testAccConfigMapV1Config_namespace(testAccConfigMapMigrationName()))
}

// State written before resource identity existed (2.38.0) has no identity, so this goes
// through UpgradeIdentity.
func TestAccConfigMap_UpgradeFromSDKV2PreIdentity_basic(t *testing.T) {
	testAccConfigMapMigrationFrom(t, sdkv2PreIdentityProviderVersion,
		testAccConfigMapV1Config_basic(testAccConfigMapMigrationName()))
}

func TestAccConfigMap_UpgradeFromSDKV2PreIdentity_generateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccConfigMapMigrationFrom(t, sdkv2PreIdentityProviderVersion, testAccConfigMapV1Config_generatedName(prefix))
}
