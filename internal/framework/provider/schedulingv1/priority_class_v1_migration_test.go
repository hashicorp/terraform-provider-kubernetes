// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package schedulingv1_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	sdkv2 "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

func testAccPreCheck(t *testing.T) {
	ctx := context.TODO()
	hasFileCfg := (os.Getenv("KUBE_CTX_AUTH_INFO") != "" && os.Getenv("KUBE_CTX_CLUSTER") != "") ||
		os.Getenv("KUBE_CTX") != "" ||
		os.Getenv("KUBE_CONFIG_PATH") != ""
	hasUserCredentials := os.Getenv("KUBE_USER") != "" && os.Getenv("KUBE_PASSWORD") != ""
	hasClientCert := os.Getenv("KUBE_CLIENT_CERT_DATA") != "" && os.Getenv("KUBE_CLIENT_KEY_DATA") != ""
	hasStaticCfg := (os.Getenv("KUBE_HOST") != "" &&
		os.Getenv("KUBE_CLUSTER_CA_CERT_DATA") != "") &&
		(hasUserCredentials || hasClientCert || os.Getenv("KUBE_TOKEN") != "")

	if !hasFileCfg && !hasStaticCfg && !hasUserCredentials {
		t.Fatalf("File config (KUBE_CTX_AUTH_INFO and KUBE_CTX_CLUSTER) or static configuration"+
			"(%s) or (%s) must be set for acceptance tests",
			strings.Join([]string{
				"KUBE_HOST",
				"KUBE_USER",
				"KUBE_PASSWORD",
				"KUBE_CLUSTER_CA_CERT_DATA",
			}, ", "),
			strings.Join([]string{
				"KUBE_HOST",
				"KUBE_CLIENT_CERT_DATA",
				"KUBE_CLIENT_KEY_DATA",
				"KUBE_CLUSTER_CA_CERT_DATA",
			}, ", "),
		)
	}

	diags := kubernetes.Provider().Configure(ctx, sdkv2.NewResourceConfigRaw(nil))
	if diags.HasError() {
		t.Fatal(diags[0].Summary)
	}
}

// sdkv2ProviderVersion is the last release that served kubernetes_priority_class_v1 from
// the SDKv2 implementation. The initial step of most migration tests runs against it.
const sdkv2ProviderVersion = "3.5.0"

// sdkv2PreIdentityProviderVersion is a release from before the provider declared resource
// identity for any type (2.38.0). No SDKv2 release declared identity for this type, so it
// is kept as the oldest supported source rather than for a difference in identity.
const sdkv2PreIdentityProviderVersion = "2.37.1"

func testAccPriorityClassV1Migration(t *testing.T, config string) {
	t.Helper()
	testAccPriorityClassV1MigrationFrom(t, sdkv2ProviderVersion, config)
}

func testAccPriorityClassV1MigrationFrom(t *testing.T, sdkv2Version, config string) {
	t.Helper()
	testAccPriorityClassV1MigrationWithAction(t, sdkv2Version, config, plancheck.ResourceActionNoop)
}

// testAccPriorityClassV1MigrationExpectingUpdate is testAccPriorityClassV1Migration for the
// shapes that cannot migrate with an empty plan: SDKv2 stored null for `labels = {}`.
func testAccPriorityClassV1MigrationExpectingUpdate(t *testing.T, config string) {
	t.Helper()
	testAccPriorityClassV1MigrationWithAction(t, sdkv2ProviderVersion, config, plancheck.ResourceActionUpdate)
}

// testAccPriorityClassV1MigrationWithAction applies config under an SDKv2 release, then
// under the framework provider, and asserts the planned action. Whether the action is a
// no-op or an update, the object keeps its uid (not replaced) and resource_version (not
// modified), and an import afterwards matches state.
func testAccPriorityClassV1MigrationWithAction(t *testing.T, sdkv2Version, config string, action plancheck.ResourceActionType) {
	t.Helper()

	sameUID := statecheck.CompareValue(compare.ValuesSame())
	sameResourceVersion := statecheck.CompareValue(compare.ValuesSame())

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckPriorityClassV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						VersionConstraint: sdkv2Version,
						Source:            "hashicorp/kubernetes",
					},
				},
				Config: config,
				ConfigStateChecks: []statecheck.StateCheck{
					sameUID.AddStateValue(priorityClassV1Address, uidPath),
					sameResourceVersion.AddStateValue(priorityClassV1Address, resourceVersionPath),
				},
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(priorityClassV1Address, action),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					sameUID.AddStateValue(priorityClassV1Address, uidPath),
					sameResourceVersion.AddStateValue(priorityClassV1Address, resourceVersionPath),
				},
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

// testAccPriorityClassV1MigrationNoRefresh is the upgrade as `terraform plan -refresh=false`
// sees it; a refreshing plan rewrites state through Read first and hides it.
//
// Step 2 is a PlanOnly step directly after the SDKv2 step. NoRefresh applies to every plan
// in the test case, so both of its plans compare config with the upgraded state itself.
// SDKv2 stored an unset generate_name as "", which the framework holds as null, so shapes
// configured with name plan a one-time update. Step 3 applies it: the update must not
// replace or modify the object, so uid and resource_version stay the same.
func testAccPriorityClassV1MigrationNoRefresh(t *testing.T, config string, action plancheck.ResourceActionType) {
	t.Helper()

	sameUID := statecheck.CompareValue(compare.ValuesSame())
	sameResourceVersion := statecheck.CompareValue(compare.ValuesSame())
	expectAction := []plancheck.PlanCheck{
		plancheck.ExpectResourceAction(priorityClassV1Address, action),
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckPriorityClassV1Destroy,
		AdditionalCLIOptions: &resource.AdditionalCLIOptions{
			Plan: resource.PlanOptions{NoRefresh: true},
		},
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						VersionConstraint: sdkv2ProviderVersion,
						Source:            "hashicorp/kubernetes",
					},
				},
				Config: config,
				ConfigStateChecks: []statecheck.StateCheck{
					sameUID.AddStateValue(priorityClassV1Address, uidPath),
					sameResourceVersion.AddStateValue(priorityClassV1Address, resourceVersionPath),
				},
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				PlanOnly:                 true,
				ExpectNonEmptyPlan:       action == plancheck.ResourceActionUpdate,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPreRefresh:  expectAction,
					PostApplyPostRefresh: expectAction,
				},
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
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

// Upgrade from SDKv2 (v3.5.0)

func TestAccPriorityClassV1_UpgradeFromSDKV2_basicName(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_basic(name)
	testAccPriorityClassV1Migration(t, config)
}

func TestAccPriorityClassV1_UpgradeFromSDKV2_generateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_generateName(prefix)
	testAccPriorityClassV1Migration(t, config)
}

func TestAccPriorityClassV1_UpgradeFromSDKV2_annotations(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_annotations(name)
	testAccPriorityClassV1Migration(t, config)
}

func TestAccPriorityClassV1_UpgradeFromSDKV2_labels(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_labels(name)
	testAccPriorityClassV1Migration(t, config)
}

func TestAccPriorityClassV1_UpgradeFromSDKV2_emptyValuesName(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_emptyValuesName(name)
	testAccPriorityClassV1MigrationExpectingUpdate(t, config)
}

func TestAccPriorityClassV1_UpgradeFromSDKV2_emptyValuesGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_emptyValuesGenerateName(prefix)
	testAccPriorityClassV1MigrationExpectingUpdate(t, config)
}

func TestAccPriorityClassV1_UpgradeFromSDKV2_completeName(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_completeName(name)
	testAccPriorityClassV1Migration(t, config)
}

func TestAccPriorityClassV1_UpgradeFromSDKV2_completeGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_completeGenerateName(prefix)
	testAccPriorityClassV1Migration(t, config)
}

// State upgraded from SDKv2 carries an all-null identity. If the object is then deleted
// outside Terraform, Read must still set an identity before removing it from state, or
// every plan fails with "Missing Resource Identity After Read".
func TestAccPriorityClassV1_UpgradeFromSDKV2_disappears(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_basic(name)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckPriorityClassV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						VersionConstraint: sdkv2ProviderVersion,
						Source:            "hashicorp/kubernetes",
					},
				},
				Config: config,
			},
			// Delete in this step's PreConfig, not in the SDKv2 step's Check: there the SDKv2
			// provider could refresh the object out of state (TF_ACC_REFRESH_AFTER_APPLY), and
			// the create plan below would pass without the framework Read seeing the 404.
			{
				PreConfig: func() {
					if err := testAccPriorityClassV1DeleteOutOfBand(name)(nil); err != nil {
						t.Fatal(err)
					}
				},
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(priorityClassV1Address, plancheck.ResourceActionCreate),
					},
				},
			},
		},
	})
}

// Upgrade from SDKv2 (v3.5.0), planned with -refresh=false

func TestAccPriorityClassV1_UpgradeFromSDKV2NoRefresh_basicName(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_basic(name)
	testAccPriorityClassV1MigrationNoRefresh(t, config, plancheck.ResourceActionUpdate)
}

func TestAccPriorityClassV1_UpgradeFromSDKV2NoRefresh_generateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_generateName(prefix)
	testAccPriorityClassV1MigrationNoRefresh(t, config, plancheck.ResourceActionNoop)
}

func TestAccPriorityClassV1_UpgradeFromSDKV2NoRefresh_emptyValuesName(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_emptyValuesName(name)
	testAccPriorityClassV1MigrationNoRefresh(t, config, plancheck.ResourceActionUpdate)
}

func TestAccPriorityClassV1_UpgradeFromSDKV2NoRefresh_completeName(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_completeName(name)
	testAccPriorityClassV1MigrationNoRefresh(t, config, plancheck.ResourceActionUpdate)
}

func TestAccPriorityClassV1_UpgradeFromSDKV2NoRefresh_completeGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_completeGenerateName(prefix)
	testAccPriorityClassV1MigrationNoRefresh(t, config, plancheck.ResourceActionNoop)
}

// Upgrade from SDKv2 (v2.37.1)

func TestAccPriorityClassV1_UpgradeFromSDKV2PreIdentity_basicName(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_basic(name)
	testAccPriorityClassV1MigrationFrom(t, sdkv2PreIdentityProviderVersion, config)
}

func TestAccPriorityClassV1_UpgradeFromSDKV2PreIdentity_basicGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_generateName(prefix)
	testAccPriorityClassV1MigrationFrom(t, sdkv2PreIdentityProviderVersion, config)
}

func TestAccPriorityClassV1_UpgradeFromSDKV2PreIdentity_completeName(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_completeName(name)
	testAccPriorityClassV1MigrationFrom(t, sdkv2PreIdentityProviderVersion, config)
}

func TestAccPriorityClassV1_UpgradeFromSDKV2PreIdentity_completeGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesPriorityClassV1Config_completeGenerateName(prefix)
	testAccPriorityClassV1MigrationFrom(t, sdkv2PreIdentityProviderVersion, config)
}

// HCL configs shared between migration and move tests

func testAccKubernetesPriorityClassV1Config_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata {
    name = %[1]q
  }

  value = 100
}
`, name)
}

func testAccKubernetesPriorityClassV1Config_generateName(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata {
    generate_name = %[1]q
  }

  value = 100
}
`, prefix)
}

func testAccKubernetesPriorityClassV1Config_annotations(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata {
    name = %[1]q
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }
  }

  value = 100
}
`, name)
}

func testAccKubernetesPriorityClassV1Config_labels(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata {
    name = %[1]q
    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }
  }

  value = 100
}
`, name)
}

func testAccKubernetesPriorityClassV1Config_emptyValuesName(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata {
    name        = %[1]q
    annotations = {}
    labels      = {}
  }

  value = 100
}
`, name)
}

func testAccKubernetesPriorityClassV1Config_emptyValuesGenerateName(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata {
    generate_name = %[1]q
    annotations   = {}
    labels        = {}
  }

  value = 100
}
`, prefix)
}

func testAccKubernetesPriorityClassV1Config_completeName(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata {
    name = %[1]q
    labels = {
      environment = "test"
    }
    annotations = {
      "example.com/note" = "migration"
    }
  }

  value             = 500
  description       = "Migration test priority class"
  global_default    = false
  preemption_policy = "PreemptLowerPriority"
}
`, name)
}

func testAccKubernetesPriorityClassV1Config_completeGenerateName(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata {
    generate_name = %[1]q
    labels = {
      environment = "test"
    }
    annotations = {
      "example.com/note" = "migration"
    }
  }

  value             = 500
  description       = "Migration test generated priority class"
  global_default    = false
  preemption_policy = "PreemptLowerPriority"
}
`, prefix)
}
