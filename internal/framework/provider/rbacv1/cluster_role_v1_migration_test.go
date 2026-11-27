// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// State-migration compatibility tests for kubernetes_cluster_role_v1.
//
// Each test creates a ClusterRole with a released, SDKv2-based provider and then plans
// the identical configuration with the locally built, muxed Plugin Framework provider.
// An empty plan proves existing state and configuration carry over in place, with no
// diff and no destroy/recreate. The baseline is roleSDKv2ProviderVersion, the same
// release the role_v1 migration is tested against. rolePreIdentityProviderVersion is the
// last release before resource identity, so those cases also exercise UpgradeIdentity.

// testAccClusterRoleCheckSameUID records the ClusterRole UID on its first call and fails
// if a later step sees a different one, i.e. the role was replaced instead of upgraded.
func testAccClusterRoleCheckSameUID() resource.TestCheckFunc {
	var prior string
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[clusterRoleResourceName]
		if !ok {
			return fmt.Errorf("missing %s", clusterRoleResourceName)
		}
		uid := rs.Primary.Attributes["metadata.0.uid"]
		if uid == "" {
			return fmt.Errorf("missing ClusterRole UID")
		}
		if prior != "" && uid != prior {
			return fmt.Errorf("ClusterRole replaced: UID %q became %q", prior, uid)
		}
		prior = uid
		return nil
	}
}

// testAccClusterRoleMigrationFrom applies config with the released provider at version,
// then with the Framework provider, which must plan check. A follow-up plan after the
// apply must always be empty.
func testAccClusterRoleMigrationFrom(t *testing.T, version, config string, check plancheck.PlanCheck, extra ...resource.TestCheckFunc) {
	t.Helper()
	checkUID := testAccClusterRoleCheckSameUID()

	resource.Test(t, resource.TestCase{
		CheckDestroy: testAccClusterRoleCheckDestroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: roleExternalProvider(version),
				Config:            config,
				Check:             checkUID,
			},
			{
				ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{check},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(append(extra, checkUID)...),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(clusterRoleResourceName, tfjsonpath.New("id"), knownvalue.NotNull()),
				},
			},
			{
				ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
				ResourceName:             clusterRoleResourceName,
				ImportState:              true,
				ImportStateVerify:        true,
			},
		},
	})
}

// testAccClusterRoleMigrationScenarios runs every shape from one SDKv2 release. Shapes
// that SDKv2 cannot store exactly as configured expect one in-place update instead of an
// empty plan.
func testAccClusterRoleMigrationScenarios(t *testing.T, version string) {
	t.Helper()
	empty := plancheck.ExpectEmptyPlan()
	update := plancheck.ExpectResourceAction(clusterRoleResourceName, plancheck.ResourceActionUpdate)

	t.Run("basic", func(t *testing.T) {
		name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
		testAccClusterRoleMigrationFrom(t, version, testAccClusterRoleConfig_basic(name), empty,
			resource.TestCheckResourceAttr(clusterRoleResourceName, "metadata.0.name", name),
			resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.#", "1"),
			resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.0.resources.#", "2"),
			resource.TestCheckResourceAttr(clusterRoleResourceName, "aggregation_rule.#", "0"),
		)
	})
	t.Run("generatedName", func(t *testing.T) {
		prefix := "tf-acc-test-gen:" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
		testAccClusterRoleMigrationFrom(t, version, testAccClusterRoleConfig_generateName(prefix), empty,
			resource.TestCheckResourceAttr(clusterRoleResourceName, "metadata.0.generate_name", prefix),
			resource.TestCheckResourceAttrSet(clusterRoleResourceName, "metadata.0.name"),
		)
	})
	t.Run("multipleRules", func(t *testing.T) {
		// Includes a rule that sets only non_resource_urls and verbs.
		name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
		testAccClusterRoleMigrationFrom(t, version, testAccClusterRoleConfig_modified(name), empty,
			resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.#", "3"),
			resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.2.non_resource_urls.#", "1"),
			resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.2.api_groups.#", "0"),
		)
	})
	t.Run("aggregationRule", func(t *testing.T) {
		name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
		testAccClusterRoleMigrationFrom(t, version, testAccClusterRoleConfig_aggRule(name), empty,
			resource.TestCheckResourceAttr(clusterRoleResourceName, "aggregation_rule.#", "1"),
			resource.TestCheckResourceAttr(clusterRoleResourceName, "aggregation_rule.0.cluster_role_selectors.0.match_expressions.0.operator", "In"),
			resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.#", "0"),
		)
	})
	t.Run("multipleSelectors", func(t *testing.T) {
		name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
		testAccClusterRoleMigrationFrom(t, version, testAccClusterRoleConfig_aggRuleMultiple(name, "foo", "bar", "bar", "foo"), empty,
			resource.TestCheckResourceAttr(clusterRoleResourceName, "aggregation_rule.0.cluster_role_selectors.#", "2"),
		)
	})
	t.Run("aggregatedRulesFilledByController", func(t *testing.T) {
		// The controller manager fills the aggregated role's rules and SDKv2 records
		// them in state. The Framework resource keeps rule empty for such a role, so
		// the upgrade must not plan their removal.
		name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
		testAccClusterRoleMigrationFrom(t, version, testAccClusterRoleConfig_aggRuleSource(name), empty,
			resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.#", "0"),
			resource.TestCheckResourceAttr("kubernetes_cluster_role_v1.source", "rule.#", "1"),
		)
	})
	t.Run("emptyMetadataMaps", func(t *testing.T) {
		// SDKv2 stores explicitly empty labels and annotations as null, so the first
		// plan updates them to {} once, as it does for the other migrated resources.
		name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
		testAccClusterRoleMigrationFrom(t, version, testAccClusterRoleConfig_emptyMetadata(name), update)
	})
}

func TestAccMigrateClusterRole(t *testing.T) {
	testAccClusterRoleMigrationScenarios(t, roleSDKv2ProviderVersion)
}

func TestAccMigrateClusterRole_preIdentity(t *testing.T) {
	testAccClusterRoleMigrationScenarios(t, rolePreIdentityProviderVersion)
}

// TestAccMigrateClusterRole_identity verifies the upgraded resource has an identity, for
// state written both with and without one.
func TestAccMigrateClusterRole_identity(t *testing.T) {
	for _, version := range []string{roleSDKv2ProviderVersion, rolePreIdentityProviderVersion} {
		t.Run(version, func(t *testing.T) {
			name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
			config := testAccClusterRoleConfig_basic(name)
			checkUID := testAccClusterRoleCheckSameUID()

			resource.Test(t, resource.TestCase{
				CheckDestroy: testAccClusterRoleCheckDestroy,
				Steps: []resource.TestStep{
					{
						ExternalProviders: roleExternalProvider(version),
						Config:            config,
						Check:             checkUID,
					},
					{
						ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
						Config:                   config,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: checkUID,
						ConfigStateChecks: []statecheck.StateCheck{
							statecheck.ExpectIdentity(clusterRoleResourceName, map[string]knownvalue.Check{
								"api_version": knownvalue.StringExact("rbac.authorization.k8s.io/v1"),
								"kind":        knownvalue.StringExact("ClusterRole"),
								"name":        knownvalue.StringExact(name),
							}),
						},
					},
				},
			})
		})
	}
}

// TestAccMigrateClusterRole_noRefresh verifies SDKv2-written state, which stores an unset
// generate_name as "", is not replaced when the Framework provider plans with
// -refresh=false, i.e. without Read normalising the value.
func TestAccMigrateClusterRole_noRefresh(t *testing.T) {
	for _, version := range []string{roleSDKv2ProviderVersion, rolePreIdentityProviderVersion} {
		t.Run(version, func(t *testing.T) {
			name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
			config := testAccClusterRoleConfig_basic(name)

			resource.Test(t, resource.TestCase{
				CheckDestroy: testAccClusterRoleCheckDestroy,
				AdditionalCLIOptions: &resource.AdditionalCLIOptions{
					Plan: resource.PlanOptions{NoRefresh: true},
				},
				Steps: []resource.TestStep{
					{
						ExternalProviders: roleExternalProvider(version),
						Config:            config,
					},
					{
						ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
						Config:                   config,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{expectNotReplaced(clusterRoleResourceName)},
						},
						Check: resource.TestCheckResourceAttr(clusterRoleResourceName, "metadata.0.name", name),
					},
				},
			})
		})
	}
}

// TestAccMigrateClusterRole_thenUpdate changes the rules as the first Framework apply
// after the upgrade. The update must keep the role (same UID) and a refresh and import
// must read back what was stored.
func TestAccMigrateClusterRole_thenUpdate(t *testing.T) {
	for _, version := range []string{roleSDKv2ProviderVersion, rolePreIdentityProviderVersion} {
		t.Run(version, func(t *testing.T) {
			name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
			checkUID := testAccClusterRoleCheckSameUID()

			resource.Test(t, resource.TestCase{
				CheckDestroy: testAccClusterRoleCheckDestroy,
				Steps: []resource.TestStep{
					{
						ExternalProviders: roleExternalProvider(version),
						Config:            testAccClusterRoleConfig_basic(name),
						Check:             checkUID,
					},
					{
						ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
						Config:                   testAccClusterRoleConfig_modified(name),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{
								plancheck.ExpectResourceAction(clusterRoleResourceName, plancheck.ResourceActionUpdate),
							},
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.#", "3"),
							resource.TestCheckResourceAttr(clusterRoleResourceName, "metadata.0.labels.%", "2"),
							checkUID,
						),
					},
					{
						ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
						ResourceName:             clusterRoleResourceName,
						ImportState:              true,
						ImportStateVerify:        true,
					},
				},
			})
		})
	}
}

// TestAccMigrateClusterRole_thenAggregate gives an upgraded ClusterRole an aggregation
// rule as the first Framework apply: the role keeps its UID and later plans are empty.
func TestAccMigrateClusterRole_thenAggregate(t *testing.T) {
	for _, version := range []string{roleSDKv2ProviderVersion, rolePreIdentityProviderVersion} {
		t.Run(version, func(t *testing.T) {
			name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
			checkUID := testAccClusterRoleCheckSameUID()

			resource.Test(t, resource.TestCase{
				CheckDestroy: testAccClusterRoleCheckDestroy,
				Steps: []resource.TestStep{
					{
						ExternalProviders: roleExternalProvider(version),
						Config:            testAccClusterRoleConfig_noAggregation(name),
						Check:             checkUID,
					},
					{
						ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
						Config:                   testAccClusterRoleConfig_aggregationOnly(name),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{
								plancheck.ExpectResourceAction(clusterRoleResourceName, plancheck.ResourceActionUpdate),
							},
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.#", "0"),
							resource.TestCheckResourceAttr(clusterRoleResourceName, "aggregation_rule.#", "1"),
							checkUID,
						),
					},
				},
			})
		})
	}
}

func testAccClusterRoleConfig_emptyMetadata(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_cluster_role_v1" "test" {
  metadata {
    name        = %[1]q
    labels      = {}
    annotations = {}
  }

  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["get"]
  }
}
`, name)
}
