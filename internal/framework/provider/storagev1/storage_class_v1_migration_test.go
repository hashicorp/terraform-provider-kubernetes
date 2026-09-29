// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package storagev1_test

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

// sdkv2ProviderVersion is the last release that served kubernetes_storage_class_v1 from
// the SDKv2 implementation. Step 1 of most migration tests runs against it.
const sdkv2ProviderVersion = "3.2.1"

// sdkv2PreIdentityProviderVersion predates resource identity, which shipped in 2.38.0.
// State it writes has identity_schema_version 0 and no identity, so upgrading from it goes
// through UpgradeIdentity rather than carrying an identity across.
const sdkv2PreIdentityProviderVersion = "2.37.1"

func testAccStorageClassV1Migration(t *testing.T, config string) {
	t.Helper()
	testAccStorageClassV1MigrationFrom(t, sdkv2ProviderVersion, config)
}

func testAccStorageClassV1MigrationFrom(t *testing.T, sdkv2Version, config string) {
	t.Helper()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckStorageClassV1Destroy,
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

// testAccStorageClassV1MigrationExpectingUpdate asserts the upgrade plans an in-place update
// instead of nothing (e.g. for configs declaring explicit empty maps {} where SDKv2 wrote null).
func testAccStorageClassV1MigrationExpectingUpdate(t *testing.T, config string) {
	t.Helper()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckStorageClassV1Destroy,
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
						plancheck.ExpectResourceAction("kubernetes_storage_class_v1.test", plancheck.ResourceActionUpdate),
					},
				},
			},
		},
	})
}

// ─── Upgrade from SDKv2 (v3.2.1) ─────────────────────────────────────────────

func TestAccStorageClassV1_UpgradeFromSDKV2_basicName(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccStorageClassV1Migration(t, testAccKubernetesStorageClassV1Config_basic(name))
}

func TestAccStorageClassV1_UpgradeFromSDKV2_generateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccStorageClassV1Migration(t, testAccKubernetesStorageClassV1Config_generateName(prefix))
}

func TestAccStorageClassV1_UpgradeFromSDKV2_annotations(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccStorageClassV1Migration(t, testAccKubernetesStorageClassV1Config_annotations(name))
}

func TestAccStorageClassV1_UpgradeFromSDKV2_labels(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccStorageClassV1Migration(t, testAccKubernetesStorageClassV1Config_labels(name))
}

func TestAccStorageClassV1_UpgradeFromSDKV2_emptyValuesName(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	// SDKv2 v3.2.1 normalises annotations={} and labels={} to null in state.
	// The Framework plan against null state with an {} config is a no-op (semantic
	// equality), not an update. The one-time patch only appears after the first
	// Framework apply, which is outside this test's scope.
	testAccStorageClassV1Migration(t, testAccKubernetesStorageClassV1Config_emptyValuesName(name))
}

func TestAccStorageClassV1_UpgradeFromSDKV2_emptyValuesGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	// Same reasoning as emptyValuesName above.
	testAccStorageClassV1Migration(t, testAccKubernetesStorageClassV1Config_emptyValuesGenerateName(prefix))
}

func TestAccStorageClassV1_UpgradeFromSDKV2_completeName(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccStorageClassV1Migration(t, testAccKubernetesStorageClassV1Config_completeName(name))
}

func TestAccStorageClassV1_UpgradeFromSDKV2_completeGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccStorageClassV1Migration(t, testAccKubernetesStorageClassV1Config_completeGenerateName(prefix))
}

// ─── Upgrade from SDKv2 Pre-Identity (v2.37.1) ───────────────────────────────

func TestAccStorageClassV1_UpgradeFromSDKV2PreIdentity_basicName(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccStorageClassV1MigrationFrom(t, sdkv2PreIdentityProviderVersion,
		testAccKubernetesStorageClassV1Config_basic(name))
}

func TestAccStorageClassV1_UpgradeFromSDKV2PreIdentity_basicGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccStorageClassV1MigrationFrom(t, sdkv2PreIdentityProviderVersion,
		testAccKubernetesStorageClassV1Config_generateName(prefix))
}

func TestAccStorageClassV1_UpgradeFromSDKV2PreIdentity_completeName(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccStorageClassV1MigrationFrom(t, sdkv2PreIdentityProviderVersion,
		testAccKubernetesStorageClassV1Config_completeName(name))
}

func TestAccStorageClassV1_UpgradeFromSDKV2PreIdentity_completeGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccStorageClassV1MigrationFrom(t, sdkv2PreIdentityProviderVersion,
		testAccKubernetesStorageClassV1Config_completeGenerateName(prefix))
}

// ─── HCL configs shared between migration and move tests ─────────────────────

func testAccKubernetesStorageClassV1Config_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_storage_class_v1" "test" {
  metadata {
    name = %[1]q
  }

  storage_provisioner = "kubernetes.io/no-provisioner"
}
`, name)
}

func testAccKubernetesStorageClassV1Config_generateName(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_storage_class_v1" "test" {
  metadata {
    generate_name = %[1]q
  }

  storage_provisioner = "kubernetes.io/no-provisioner"
}
`, prefix)
}

func testAccKubernetesStorageClassV1Config_annotations(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_storage_class_v1" "test" {
  metadata {
    name = %[1]q
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }
  }

  storage_provisioner = "kubernetes.io/no-provisioner"
}
`, name)
}

func testAccKubernetesStorageClassV1Config_labels(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_storage_class_v1" "test" {
  metadata {
    name = %[1]q
    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }
  }

  storage_provisioner = "kubernetes.io/no-provisioner"
}
`, name)
}

func testAccKubernetesStorageClassV1Config_emptyValuesName(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_storage_class_v1" "test" {
  metadata {
    name        = %[1]q
    annotations = {}
    labels      = {}
  }

  storage_provisioner = "kubernetes.io/no-provisioner"
}
`, name)
}

func testAccKubernetesStorageClassV1Config_emptyValuesGenerateName(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_storage_class_v1" "test" {
  metadata {
    generate_name = %[1]q
    annotations   = {}
    labels        = {}
  }

  storage_provisioner = "kubernetes.io/no-provisioner"
}
`, prefix)
}

func testAccKubernetesStorageClassV1Config_completeName(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_storage_class_v1" "test" {
  metadata {
    name = %[1]q
    labels = {
      environment = "test"
    }
    annotations = {
      "example.com/note" = "migration"
    }
  }

  storage_provisioner    = "kubernetes.io/no-provisioner"
  reclaim_policy         = "Delete"
  volume_binding_mode    = "Immediate"
  allow_volume_expansion = true
  mount_options          = ["debug"]
}
`, name)
}

func testAccKubernetesStorageClassV1Config_completeGenerateName(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_storage_class_v1" "test" {
  metadata {
    generate_name = %[1]q
    labels = {
      environment = "test"
    }
    annotations = {
      "example.com/note" = "migration"
    }
  }

  storage_provisioner    = "kubernetes.io/no-provisioner"
  reclaim_policy         = "Delete"
  volume_binding_mode    = "Immediate"
  allow_volume_expansion = true
  mount_options          = ["debug"]
}
`, prefix)
}
