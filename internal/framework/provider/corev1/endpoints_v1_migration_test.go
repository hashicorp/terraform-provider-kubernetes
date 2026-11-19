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

func testAccEndpointsMigration(t *testing.T, config string) {
	t.Helper()
	testAccEndpointsMigrationFrom(t, sdkv2ProviderVersion, config)
}

func testAccEndpointsMigrationFrom(t *testing.T, sdkv2Version, config string) {
	t.Helper()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckKubernetesEndpointV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						VersionConstraint: sdkv2Version,
						Source:            "hashicorp/kubernetes",
					},
				},
				Config: config,
			},
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

func TestAccEndpoints_UpgradeFromSDKV2(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccEndpointsMigration(t, testAccKubernetesEndpointsV1_basic(name))
}

func TestAccEndpoints_UpgradeFromSDKV2_generatedName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccEndpointsMigration(t, testAccKubernetesEndpointsV1_generatedName(prefix))
}

func TestAccEndpoints_UpgradeFromSDKV2_modified(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccEndpointsMigration(t, testAccKubernetesEndpointsV1_modified(name))
}

func TestAccEndpoints_UpgradeFromSDKV2PreIdentity_basic(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccEndpointsMigrationFrom(t, sdkv2PreIdentityProviderVersion, testAccKubernetesEndpointsV1_basic(name))
}

func TestAccEndpoints_UpgradeFromSDKV2PreIdentity_modified(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccEndpointsMigrationFrom(t, sdkv2PreIdentityProviderVersion, testAccKubernetesEndpointsV1_modified(name))
}
