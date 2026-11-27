// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package schedulingv1_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

// Acceptance Move Tests (moved block)

const priorityClassAliasAddress = "kubernetes_priority_class.test"

func testAccPriorityClassMoveStateSource(config string) string {
	return strings.ReplaceAll(config, `"kubernetes_priority_class_v1"`, `"kubernetes_priority_class"`)
}

func testAccPriorityClassMoveStateTarget(config string) string {
	return config + `
moved {
  from = kubernetes_priority_class.test
  to   = kubernetes_priority_class_v1.test
}
`
}

func testAccPriorityClassMoveState(t *testing.T, sdkv2Version, config string) {
	t.Helper()
	testAccPriorityClassMoveStateWithAction(t, sdkv2Version, config, plancheck.ResourceActionNoop)
}

// testAccPriorityClassMoveStateExpectingUpdate is for `labels = {}` and `annotations = {}`:
// SDKv2 stored them as null, so the first plan after the move updates state to `{}`.
func testAccPriorityClassMoveStateExpectingUpdate(t *testing.T, sdkv2Version, config string) {
	t.Helper()
	testAccPriorityClassMoveStateWithAction(t, sdkv2Version, config, plancheck.ResourceActionUpdate)
}

// testAccPriorityClassMoveStateWithAction applies config as kubernetes_priority_class under
// an SDKv2 release, plans the move without applying it (PlanOnly), then applies it. The
// object keeps its uid (not replaced) and resource_version (not modified).
// Cross-type moved needs Terraform 1.8 or later.
func testAccPriorityClassMoveStateWithAction(t *testing.T, sdkv2Version, config string, action plancheck.ResourceActionType) {
	t.Helper()

	sourceConfig := testAccPriorityClassMoveStateSource(config)
	targetConfig := testAccPriorityClassMoveStateTarget(config)
	sameUID := statecheck.CompareValue(compare.ValuesSame())
	sameResourceVersion := statecheck.CompareValue(compare.ValuesSame())
	expectAction := []plancheck.PlanCheck{
		plancheck.ExpectResourceAction(priorityClassV1Address, action),
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckPriorityClassV1Destroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						VersionConstraint: sdkv2Version,
						Source:            "hashicorp/kubernetes",
					},
				},
				Config: sourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					sameUID.AddStateValue(priorityClassAliasAddress, uidPath),
					sameResourceVersion.AddStateValue(priorityClassAliasAddress, resourceVersionPath),
				},
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   targetConfig,
				PlanOnly:                 true,
				ExpectNonEmptyPlan:       action == plancheck.ResourceActionUpdate,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPreRefresh:  expectAction,
					PostApplyPostRefresh: expectAction,
				},
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   targetConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: expectAction,
				},
				ConfigStateChecks: []statecheck.StateCheck{
					sameUID.AddStateValue(priorityClassV1Address, uidPath),
					sameResourceVersion.AddStateValue(priorityClassV1Address, resourceVersionPath),
				},
			},
		},
	})
}

// Moved from SDKv2 (v3.5.0)

func TestAccPriorityClassV1_MoveStateFromUnversioned_basicName(t *testing.T) {
	name := fmt.Sprintf("tf-move-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_basic(name)
	testAccPriorityClassMoveState(t, sdkv2ProviderVersion, config)
}

func TestAccPriorityClassV1_MoveStateFromUnversioned_generateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-move-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_generateName(prefix)
	testAccPriorityClassMoveState(t, sdkv2ProviderVersion, config)
}

func TestAccPriorityClassV1_MoveStateFromUnversioned_annotations(t *testing.T) {
	name := fmt.Sprintf("tf-move-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_annotations(name)
	testAccPriorityClassMoveState(t, sdkv2ProviderVersion, config)
}

func TestAccPriorityClassV1_MoveStateFromUnversioned_labels(t *testing.T) {
	name := fmt.Sprintf("tf-move-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_labels(name)
	testAccPriorityClassMoveState(t, sdkv2ProviderVersion, config)
}

func TestAccPriorityClassV1_MoveStateFromUnversioned_emptyValuesName(t *testing.T) {
	name := fmt.Sprintf("tf-move-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_emptyValuesName(name)
	testAccPriorityClassMoveStateExpectingUpdate(t, sdkv2ProviderVersion, config)
}

func TestAccPriorityClassV1_MoveStateFromUnversioned_emptyValuesGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-move-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_emptyValuesGenerateName(prefix)
	testAccPriorityClassMoveStateExpectingUpdate(t, sdkv2ProviderVersion, config)
}

func TestAccPriorityClassV1_MoveStateFromUnversioned_completeName(t *testing.T) {
	name := fmt.Sprintf("tf-move-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_completeName(name)
	testAccPriorityClassMoveState(t, sdkv2ProviderVersion, config)
}

func TestAccPriorityClassV1_MoveStateFromUnversioned_completeGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-move-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_completeGenerateName(prefix)
	testAccPriorityClassMoveState(t, sdkv2ProviderVersion, config)
}

// After moving a generate_name object, an update must keep id and metadata.name known in
// the plan, and must not replace the object.
func TestAccPriorityClassV1_MoveStateFromUnversioned_generateNameUpdate(t *testing.T) {
	prefix := fmt.Sprintf("tf-move-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_completeGenerateName(prefix)
	sourceConfig := testAccPriorityClassMoveStateSource(config)
	targetConfig := testAccPriorityClassMoveStateTarget(config)
	updatedConfig := testAccPriorityClassV1Config_completeGenerateNameUpdated(prefix)
	nameMatchesPrefix := knownvalue.StringRegexp(regexp.MustCompile("^" + prefix))
	namePath := tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("name")
	sameUID := statecheck.CompareValue(compare.ValuesSame())

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckPriorityClassV1Destroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						VersionConstraint: sdkv2ProviderVersion,
						Source:            "hashicorp/kubernetes",
					},
				},
				Config: sourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					sameUID.AddStateValue(priorityClassAliasAddress, uidPath),
				},
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   targetConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   updatedConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(priorityClassV1Address, plancheck.ResourceActionUpdate),
						plancheck.ExpectKnownValue(priorityClassV1Address, tfjsonpath.New("id"), nameMatchesPrefix),
						plancheck.ExpectKnownValue(priorityClassV1Address, namePath, nameMatchesPrefix),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					sameUID.AddStateValue(priorityClassV1Address, uidPath),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(priorityClassV1Address, "description", "Updated priority class"),
					resource.TestCheckResourceAttr(priorityClassV1Address, "metadata.0.labels.environment", "updated"),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				ResourceName:             priorityClassV1Address,
				ImportState:              true,
				ImportStateVerify:        true,
			},
		},
	})
}

// The SDKv2 alias never had an identity, so MoveState builds one. Identity is visible in
// state only from Terraform 1.12.
func TestAccPriorityClassV1_MoveStateFromUnversioned_identity(t *testing.T) {
	name := fmt.Sprintf("tf-move-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_basic(name)
	sourceConfig := testAccPriorityClassMoveStateSource(config)
	targetConfig := testAccPriorityClassMoveStateTarget(config)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckPriorityClassV1Destroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						VersionConstraint: sdkv2ProviderVersion,
						Source:            "hashicorp/kubernetes",
					},
				},
				Config: sourceConfig,
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   targetConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(priorityClassV1Address, map[string]knownvalue.Check{
						"name":        knownvalue.StringExact(name),
						"api_version": knownvalue.StringExact("scheduling.k8s.io/v1"),
						"kind":        knownvalue.StringExact("PriorityClass"),
					}),
				},
			},
		},
	})
}

// MoveState must accept only kubernetes_priority_class. A move from any other type is
// skipped, and Terraform then fails the move rather than accepting state decoded with the
// wrong shape. The provider-address and schema-version guards cannot be reached from a
// configuration, so this covers the source-type guard only.
func TestAccPriorityClassV1_MoveStateFromOtherTypeIsRejected(t *testing.T) {
	name := fmt.Sprintf("tf-move-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	sourceConfig := testAccPriorityClassV1Config_configMap(name)
	targetConfig := testAccPriorityClassV1Config_configMapMovedToPriorityClass(name)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPriorityClassV1Destroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		Steps: []resource.TestStep{
			{
				Config: sourceConfig,
			},
			{
				Config:      targetConfig,
				ExpectError: regexp.MustCompile(`Unable to Move Resource State`),
			},
			// Leave the config map under its own address so the test can destroy it.
			{
				Config: sourceConfig,
			},
		},
	})
}

func testAccPriorityClassV1Config_configMap(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_config_map_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = "default"
  }
}
`, name)
}

func testAccPriorityClassV1Config_configMapMovedToPriorityClass(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata {
    name = %[1]q
  }

  value = 100
}

moved {
  from = kubernetes_config_map_v1.test
  to   = kubernetes_priority_class_v1.test
}
`, name)
}

// Moved from SDKv2 (v2.37.1)

func TestAccPriorityClassV1_MoveStateFromUnversionedPreIdentity_basicName(t *testing.T) {
	name := fmt.Sprintf("tf-move-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_basic(name)
	testAccPriorityClassMoveState(t, sdkv2PreIdentityProviderVersion, config)
}

func TestAccPriorityClassV1_MoveStateFromUnversionedPreIdentity_basicGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-move-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_generateName(prefix)
	testAccPriorityClassMoveState(t, sdkv2PreIdentityProviderVersion, config)
}

func TestAccPriorityClassV1_MoveStateFromUnversionedPreIdentity_completeName(t *testing.T) {
	name := fmt.Sprintf("tf-move-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_completeName(name)
	testAccPriorityClassMoveState(t, sdkv2PreIdentityProviderVersion, config)
}

func TestAccPriorityClassV1_MoveStateFromUnversionedPreIdentity_completeGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-move-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_completeGenerateName(prefix)
	testAccPriorityClassMoveState(t, sdkv2PreIdentityProviderVersion, config)
}
