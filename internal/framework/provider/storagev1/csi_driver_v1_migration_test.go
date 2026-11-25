// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package storagev1_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	storagev1api "k8s.io/api/storage/v1"
)

// Migration tests: state written by a released SDKv2 provider, planned by this build under
// the same type name, with byte-identical configuration in both steps.

// csiDriverSDKv2ProviderVersion is the last release that served kubernetes_csi_driver_v1
// from SDKv2.
const csiDriverSDKv2ProviderVersion = "3.5.0"

// csiDriverPreIdentityProviderVersion predates resource identity (2.38.0). The CSI driver
// never had an identity under SDKv2, so state from every release reaches UpgradeIdentity;
// this version also exercises state that has no identity block at all.
const csiDriverPreIdentityProviderVersion = "2.37.1"

func testAccSDKv2Provider(version string) map[string]resource.ExternalProvider {
	return map[string]resource.ExternalProvider{
		"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: version},
	}
}

// csiDriverIdentityChecks asserts the identity this build writes, comparing the name with
// state so it also works for a server-generated name.
func csiDriverIdentityChecks() []statecheck.StateCheck {
	return []statecheck.StateCheck{
		statecheck.ExpectIdentityValue(csiDriverResourceName, tfjsonpath.New("kind"), knownvalue.StringExact("CSIDriver")),
		statecheck.ExpectIdentityValue(csiDriverResourceName, tfjsonpath.New("api_version"), knownvalue.StringExact("storage.k8s.io/v1")),
		statecheck.ExpectIdentityValueMatchesStateAtPath(csiDriverResourceName,
			tfjsonpath.New("name"), tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("name")),
	}
}

// testAccCSIDriverV1UpgradeFrom applies config under the given SDKv2 release, then plans it
// under this build and checks the first plan with firstPlan. The driver must keep its UID,
// and an import afterwards must reproduce the upgraded state.
//
// sdkv2Drifts marks configurations SDKv2 itself never settles on, so its own post-apply
// plan is expected to be non-empty.
func testAccCSIDriverV1UpgradeFrom(t *testing.T, sdkv2Version, config string, sdkv2Drifts bool, firstPlan plancheck.PlanCheck) {
	t.Helper()
	var before, after storagev1api.CSIDriver
	importStep := testAccCSIDriverV1ImportStep()
	importStep.ProtoV6ProviderFactories = testAccProtoV6ProviderFactories

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		// Identity is only exchanged with Terraform 1.12 and later, and it is what this
		// upgrade changes.
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		CheckDestroy: testAccCheckCSIDriverV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders:  testAccSDKv2Provider(sdkv2Version),
				Config:             config,
				Check:              testAccCheckCSIDriverV1Exists(csiDriverResourceName, &before),
				ExpectNonEmptyPlan: sdkv2Drifts,
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{firstPlan},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				ConfigStateChecks: csiDriverIdentityChecks(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCSIDriverV1Exists(csiDriverResourceName, &after),
					testAccCheckCSIDriverV1Recreated(&before, &after, false),
				),
			},
			importStep,
		},
	})
}

func TestAccCSIDriverV1_UpgradeFromSDKV2_basicName(t *testing.T) {
	testAccCSIDriverV1UpgradeFrom(t, csiDriverSDKv2ProviderVersion,
		testAccCSIDriverV1Config_basic(acctest.RandomWithPrefix(csiDriverNamePrefix), true),
		false, plancheck.ExpectEmptyPlan())
}

func TestAccCSIDriverV1_UpgradeFromSDKV2_completeName(t *testing.T) {
	testAccCSIDriverV1UpgradeFrom(t, csiDriverSDKv2ProviderVersion,
		testAccCSIDriverV1Config_completeName(acctest.RandomWithPrefix(csiDriverNamePrefix), "one", true),
		false, plancheck.ExpectEmptyPlan())
}

// SDKv2 stores "" for an unset generate_name and the real prefix here; both must upgrade
// without a replacement.
func TestAccCSIDriverV1_UpgradeFromSDKV2_completeGenerateName(t *testing.T) {
	testAccCSIDriverV1UpgradeFrom(t, csiDriverSDKv2ProviderVersion,
		testAccCSIDriverV1Config_completeGenerateName(csiDriverNamePrefix, "one", true),
		false, plancheck.ExpectEmptyPlan())
}

// SDKv2 plans to remove the API-defaulted volume_lifecycle_modes on every run. After the
// upgrade the attribute is Computed, so the same configuration plans nothing.
func TestAccCSIDriverV1_UpgradeFromSDKV2_apiDefaults(t *testing.T) {
	testAccCSIDriverV1UpgradeFrom(t, csiDriverSDKv2ProviderVersion,
		testAccCSIDriverV1Config_apiDefaults(acctest.RandomWithPrefix(csiDriverNamePrefix)),
		true, plancheck.ExpectEmptyPlan())
}

// SDKv2 stores null for labels = {} and annotations = {}; the framework can hold the {} the
// configuration asks for, so the first plan is a one-off in-place update, never a replacement.
func TestAccCSIDriverV1_UpgradeFromSDKV2_emptyMaps(t *testing.T) {
	testAccCSIDriverV1UpgradeFrom(t, csiDriverSDKv2ProviderVersion,
		testAccCSIDriverV1Config_emptyMaps(acctest.RandomWithPrefix(csiDriverNamePrefix)),
		false, plancheck.ExpectResourceAction(csiDriverResourceName, plancheck.ResourceActionUpdate))
}

// The pre-identity tests cover state with no identity at all. Without UpgradeIdentity
// their first plan fails with "Unable to Upgrade Resource Identity".
func TestAccCSIDriverV1_UpgradeFromSDKV2PreIdentity_basicName(t *testing.T) {
	testAccCSIDriverV1UpgradeFrom(t, csiDriverPreIdentityProviderVersion,
		testAccCSIDriverV1Config_basic(acctest.RandomWithPrefix(csiDriverNamePrefix), true),
		false, plancheck.ExpectEmptyPlan())
}

func TestAccCSIDriverV1_UpgradeFromSDKV2PreIdentity_completeName(t *testing.T) {
	testAccCSIDriverV1UpgradeFrom(t, csiDriverPreIdentityProviderVersion,
		testAccCSIDriverV1Config_completeName(acctest.RandomWithPrefix(csiDriverNamePrefix), "one", true),
		false, plancheck.ExpectEmptyPlan())
}

func TestAccCSIDriverV1_UpgradeFromSDKV2PreIdentity_completeGenerateName(t *testing.T) {
	testAccCSIDriverV1UpgradeFrom(t, csiDriverPreIdentityProviderVersion,
		testAccCSIDriverV1Config_completeGenerateName(csiDriverNamePrefix, "one", true),
		false, plancheck.ExpectEmptyPlan())
}
