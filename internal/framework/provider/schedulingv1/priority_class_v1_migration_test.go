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
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
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
// the SDKv2 implementation. Step 1 of most migration tests runs against it.
const sdkv2ProviderVersion = "3.2.1"

// sdkv2PreIdentityProviderVersion predates resource identity, which shipped in 2.38.0.
// State it writes has identity_schema_version 0 and no identity, so upgrading from it goes
// through UpgradeIdentity rather than carrying an identity across.
const sdkv2PreIdentityProviderVersion = "2.37.1"

func testAccPriorityClassV1Migration(t *testing.T, config string) {
	t.Helper()
	testAccPriorityClassV1MigrationFrom(t, sdkv2ProviderVersion, config)
}

func testAccPriorityClassV1MigrationFrom(t *testing.T, sdkv2Version, config string) {
	t.Helper()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
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

// testAccPriorityClassV1MigrationExpectingUpdate is testAccPriorityClassV1Migration for the shapes
// that cannot migrate with an empty plan due to SDKv2 null vs Framework empty map handling.
// It asserts the upgrade plans an in-place update instead of nothing.
func testAccPriorityClassV1MigrationExpectingUpdate(t *testing.T, config string) {
	t.Helper()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
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
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("kubernetes_priority_class_v1.test", plancheck.ResourceActionUpdate),
					},
				},
			},
		},
	})
}

// ─── Upgrade from SDKv2 (v3.2.1) ─────────────────────────────────────────────

func TestAccPriorityClassV1_UpgradeFromSDKV2_basicName(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccPriorityClassV1Migration(t, testAccKubernetesPriorityClassV1Config_basic(name))
}

func TestAccPriorityClassV1_UpgradeFromSDKV2_generateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccPriorityClassV1Migration(t, testAccKubernetesPriorityClassV1Config_generateName(prefix))
}

func TestAccPriorityClassV1_UpgradeFromSDKV2_annotations(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccPriorityClassV1Migration(t, testAccKubernetesPriorityClassV1Config_annotations(name))
}

func TestAccPriorityClassV1_UpgradeFromSDKV2_labels(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccPriorityClassV1Migration(t, testAccKubernetesPriorityClassV1Config_labels(name))
}

func TestAccPriorityClassV1_UpgradeFromSDKV2_emptyValuesName(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccPriorityClassV1MigrationExpectingUpdate(t, testAccKubernetesPriorityClassV1Config_emptyValuesName(name))
}

func TestAccPriorityClassV1_UpgradeFromSDKV2_emptyValuesGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccPriorityClassV1MigrationExpectingUpdate(t, testAccKubernetesPriorityClassV1Config_emptyValuesGenerateName(prefix))
}

func TestAccPriorityClassV1_UpgradeFromSDKV2_completeName(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccPriorityClassV1Migration(t, testAccKubernetesPriorityClassV1Config_completeName(name))
}

func TestAccPriorityClassV1_UpgradeFromSDKV2_completeGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccPriorityClassV1Migration(t, testAccKubernetesPriorityClassV1Config_completeGenerateName(prefix))
}

// ─── Upgrade from SDKv2 Pre-Identity (v2.37.1) ───────────────────────────────

func TestAccPriorityClassV1_UpgradeFromSDKV2PreIdentity_basicName(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccPriorityClassV1MigrationFrom(t, sdkv2PreIdentityProviderVersion,
		testAccKubernetesPriorityClassV1Config_basic(name))
}

func TestAccPriorityClassV1_UpgradeFromSDKV2PreIdentity_basicGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccPriorityClassV1MigrationFrom(t, sdkv2PreIdentityProviderVersion,
		testAccKubernetesPriorityClassV1Config_generateName(prefix))
}

func TestAccPriorityClassV1_UpgradeFromSDKV2PreIdentity_completeName(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccPriorityClassV1MigrationFrom(t, sdkv2PreIdentityProviderVersion,
		testAccKubernetesPriorityClassV1Config_completeName(name))
}

func TestAccPriorityClassV1_UpgradeFromSDKV2PreIdentity_completeGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccPriorityClassV1MigrationFrom(t, sdkv2PreIdentityProviderVersion,
		testAccKubernetesPriorityClassV1Config_completeGenerateName(prefix))
}

// ─── HCL configs shared between migration and move tests ─────────────────────

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
