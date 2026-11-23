// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

// endpointsSDKv2ProviderVersion is the last release that serves kubernetes_endpoints_v1
// from SDKv2. SDKv2 never declared a resource identity for Endpoints, so no older
// pre-identity baseline is needed.
const endpointsSDKv2ProviderVersion = "3.5.0"

func endpointsSDKv2Provider() map[string]resource.ExternalProvider {
	return map[string]resource.ExternalProvider{
		"kubernetes": {VersionConstraint: endpointsSDKv2ProviderVersion, Source: "hashicorp/kubernetes"},
	}
}

func testAccEndpointsMigration(t *testing.T, config string) {
	t.Helper()
	checkUID := testAccCheckEndpointsSameUID()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckKubernetesEndpointV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: endpointsSDKv2Provider(),
				Config:            config,
				Check:             checkUID,
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: checkUID,
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

func TestAccEndpoints_UpgradeFromSDKV2_protocolOmitted(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccEndpointsMigration(t, testAccKubernetesEndpointsV1_protocol(name, ""))
}

func TestAccEndpoints_UpgradeFromSDKV2_noSubset(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccEndpointsMigration(t, testAccKubernetesEndpointsV1_noSubset(name))
}

// TestAccEndpoints_UpgradeFromSDKV2_identity verifies that the first refresh after the
// upgrade populates the new resource identity.
func TestAccEndpoints_UpgradeFromSDKV2_identity(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesEndpointsV1_basic(name)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:               func() { testAccPreCheck(t) },
		CheckDestroy:           testAccCheckKubernetesEndpointV1Destroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)},
		Steps: []resource.TestStep{
			{ExternalProviders: endpointsSDKv2Provider(), Config: config},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks:         resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
				ConfigStateChecks:        []statecheck.StateCheck{testAccEndpointsExpectIdentity(name)},
			},
		},
	})
}

// TestAccEndpoints_UpgradeFromSDKV2_explicitEmptyStrings documents the one-time update
// for explicitly empty hostname, node_name and port name values. SDKv2 stored "" for
// unset values too, so the first refresh normalizes them to null and the configured
// "" is written back once. The object is updated in place and then converges.
func TestAccEndpoints_UpgradeFromSDKV2_explicitEmptyStrings(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesEndpointsV1_explicitEmptyStrings(name, 80)
	checkUID := testAccCheckEndpointsSameUID()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckKubernetesEndpointV1Destroy,
		Steps: []resource.TestStep{
			{ExternalProviders: endpointsSDKv2Provider(), Config: config, Check: checkUID},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(endpointsResourceName, plancheck.ResourceActionUpdate)},
					PostApplyPreRefresh:  []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: checkUID,
			},
		},
	})
}

// TestAccEndpoints_UpgradeFromSDKV2_noRefresh documents the one-time in-place update
// shown by `terraform plan -refresh=false` straight after upgrading: SDKv2 stored ""
// for unset hostname, node_name, port name and generate_name, the framework uses null.
func TestAccEndpoints_UpgradeFromSDKV2_noRefresh(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesEndpointsV1_basic(name)
	checkUID := testAccCheckEndpointsSameUID()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:             func() { testAccPreCheck(t) },
		CheckDestroy:         testAccCheckKubernetesEndpointV1Destroy,
		AdditionalCLIOptions: &resource.AdditionalCLIOptions{Plan: resource.PlanOptions{NoRefresh: true}},
		Steps: []resource.TestStep{
			{ExternalProviders: endpointsSDKv2Provider(), Config: config, Check: checkUID},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(endpointsResourceName, plancheck.ResourceActionUpdate)},
					PostApplyPreRefresh:  []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: checkUID,
			},
		},
	})
}

// Metadata written by another controller must survive the upgrade and the first update
// after it, which also replaces /subsets. Mirrors
// TestAccNamespace_UpgradeFromSDKV2_ignoredMetadataSurvivesUpdate: SDKv2 state holds no
// keys for the unmanaged map, so an unguarded Update would replace the live map.
func TestAccEndpoints_UpgradeFromSDKV2_ignoredMetadataSurvivesUpdate(t *testing.T) {
	for _, tc := range []struct{ ignored, managed string }{
		{ignored: "annotations", managed: "labels"},
		{ignored: "labels", managed: "annotations"},
	} {
		t.Run(tc.ignored, func(t *testing.T) {
			name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
			checkUID := testAccCheckEndpointsSameUID()
			before := testAccKubernetesEndpointsV1_managedMetadata(name, tc.managed, "before", "", 80)
			after := testAccKubernetesEndpointsV1_managedMetadata(name, tc.managed, "after", "", 81)

			resource.ParallelTest(t, resource.TestCase{
				PreCheck:     func() { testAccPreCheck(t) },
				CheckDestroy: testAccCheckKubernetesEndpointV1Destroy,
				Steps: []resource.TestStep{
					{
						ExternalProviders: endpointsSDKv2Provider(),
						Config:            before,
						Check:             checkUID,
					},
					{
						// Another controller writes to the unmanaged map while SDKv2 still
						// owns the resource.
						PreConfig:                func() { testAccInjectIgnoredEndpointsMetadata(t, name, tc.ignored, ignoredMetadataKey) },
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Config:                   before,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							checkUID,
							testAccCheckIgnoredEndpointsMetadata(name, tc.ignored, ignoredMetadataKey),
						),
					},
					{
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Config:                   after,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(endpointsResourceName, plancheck.ResourceActionUpdate)},
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							checkUID,
							resource.TestCheckResourceAttr(endpointsResourceName, "metadata.0."+tc.managed+".managed", "after"),
							testAccCheckIgnoredEndpointsMetadata(name, tc.ignored, ignoredMetadataKey),
						),
					},
				},
			})
		})
	}
}
