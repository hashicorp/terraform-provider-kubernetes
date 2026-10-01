// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0
package corev1_test

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

	"github.com/hashicorp/terraform-plugin-testing/terraform"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
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

// The baseline shape: name only, both metadata maps absent. Exercises the filtering
// that strips the API server's kubernetes.io/metadata.name back out on Read.
func TestAccNamespace_UpgradeFromSDKV2(t *testing.T) {
	nsName := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMigration(t, testAccKubernetesNamespaceV1Config_basic(nsName))
}

// sdkv2ProviderVersion is the last release that served kubernetes_namespace_v1 from
// the SDKv2 implementation. Step 1 of most migration tests runs against it.
const sdkv2ProviderVersion = "3.2.1"

// sdkv2PreIdentityProviderVersion predates resource identity, which shipped in 2.38.0.
// State it writes has identity_schema_version 0 and no identity, so upgrading from it goes
// through UpgradeIdentity rather than carrying an identity across.
const sdkv2PreIdentityProviderVersion = "2.37.1"

// testAccNamespaceMigration applies config under the last released SDKv2 provider,
// then re-plans the identical config under the local framework provider and asserts
// the upgrade produces no changes.
//
// Step 2 refreshes state written by step 1 through the framework Read, so these tests
// exercise common.FlattenMetadata and its filtering rather than Create. The two configs
// must stay byte-identical: any difference makes a non-empty plan ambiguous.
func testAccNamespaceMigration(t *testing.T, config string) {
	t.Helper()
	testAccNamespaceMigrationFrom(t, sdkv2ProviderVersion, config)
}

// testAccNamespaceMigrationFrom is testAccNamespaceMigration against a chosen SDKv2
// release, so the same shape can be checked from before and after resource identity.
func testAccNamespaceMigrationFrom(t *testing.T, sdkv2Version, config string) {
	t.Helper()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		// Without this a failed run leaks namespaces, and the next run collides on the
		// same generated names.
		CheckDestroy: testAccCheckKubernetesNamespaceV1Destroy,
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

// testAccNamespaceMigrationExpectingUpdate is testAccNamespaceMigration for the shapes
// that cannot migrate cleanly. It asserts the upgrade plans an in-place update instead
// of nothing.
//
// The specific action matters. An update is a one-off upgrade note; a replacement would
// destroy the namespace and is never acceptable. Asserting the action rather than merely
// tolerating a non-empty plan is what keeps that distinction enforced.
func testAccNamespaceMigrationExpectingUpdate(t *testing.T, config string) {
	t.Helper()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		// Without this a failed run leaks namespaces, and the next run collides on the
		// same generated names.
		CheckDestroy: testAccCheckKubernetesNamespaceV1Destroy,
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
						plancheck.ExpectResourceAction("kubernetes_namespace_v1.test", plancheck.ResourceActionUpdate),
					},
				},
			},
		},
	})
}

// name is server-assigned here, so this is the only shape where a metadata value is
// unknown at plan time. It is what settles whether the schema needs UseStateForUnknown.
func TestAccNamespace_UpgradeFromSDKV2_generateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMigration(t, testAccKubernetesNamespaceV1Config_generatedName(prefix))
}

// annotations populated, labels absent — one map null and one not, in the direction
// where the API also adds an internal label that has to be filtered back out.
func TestAccNamespace_UpgradeFromSDKV2_annotations(t *testing.T) {
	nsName := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMigration(t, testAccKubernetesNamespaceV1Config_Annotations(nsName))
}

// The mirror of the above: labels populated, annotations absent.
func TestAccNamespace_UpgradeFromSDKV2_labels(t *testing.T) {
	nsName := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMigration(t, testAccKubernetesNamespaceV1Config_Labels(nsName))
}

// A key that the filtering rules would normally strip, declared in config so that it
// must survive. Covers the branch where prior state exempts a key from removal.
func TestAccNamespace_UpgradeFromSDKV2_declaredInternalLabel(t *testing.T) {
	nsName := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMigration(t, testAccKubernetesNamespaceV1Config_declaredInternalLabel(nsName))
}

// Every optional field declared but empty. This is the one shape that does NOT migrate
// cleanly, and the divergence is inherent rather than a defect in the framework code.
//
// SDKv2 writes `null` to state for a config that declares `annotations = {}`, because
// its type system cannot represent the difference between null and a known empty map.
// The framework can, and Read correctly preserves the null it was given. But the config
// still says `{}`, so Terraform plans an update to reconcile the two.
//
// The provider cannot suppress this: the plan is derived from config and prior state,
// and the only way to make `{}` and null compare equal again would be a plan modifier
// that reintroduces the SDKv2 collapse — losing the ability to ever express an explicit
// empty map.
//
// So the assertion is the real behaviour: one update on first plan after the upgrade.
// It is non-destructive, and the implicit post-apply idempotency plan that every Config
// step runs proves it settles rather than recurring.
func TestAccNamespace_UpgradeFromSDKV2_emptyValuesName(t *testing.T) {
	nsName := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMigrationExpectingUpdate(t, testAccKubernetesNamespaceV1Config_emptyValuesName(nsName))
}

// The same shape named by generate_name rather than name. Both are configs a user could
// have written under the SDKv2 provider, so both are in scope for the upgrade contract.
func TestAccNamespace_UpgradeFromSDKV2_emptyValuesGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMigrationExpectingUpdate(t, testAccKubernetesNamespaceV1Config_emptyValuesGeneratedName(prefix))
}

// Every optional field declared with a real value.
func TestAccNamespace_UpgradeFromSDKV2_completeName(t *testing.T) {
	nsName := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMigration(t, testAccKubernetesNamespaceV1Config_completeName(nsName))
}

// The complete shape named by generate_name rather than name.
func TestAccNamespace_UpgradeFromSDKV2_completeGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMigration(t, testAccKubernetesNamespaceV1Config_completeGeneratedName(prefix))
}

// The pre-identity tests below are the only coverage of state written before resource
// identity existed (2.38.0). Upgrading it exercises UpgradeIdentity, where the stored
// identity is absent rather than merely older. Four shapes: the minimal one and the fully
// populated one, each with name and with generate_name, since a server-assigned name is
// what the identity carries.
func TestAccNamespace_UpgradeFromSDKV2PreIdentity_basicName(t *testing.T) {
	nsName := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMigrationFrom(t, sdkv2PreIdentityProviderVersion,
		testAccKubernetesNamespaceV1Config_basic(nsName))
}

func TestAccNamespace_UpgradeFromSDKV2PreIdentity_basicGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMigrationFrom(t, sdkv2PreIdentityProviderVersion,
		testAccKubernetesNamespaceV1Config_generatedName(prefix))
}

func TestAccNamespace_UpgradeFromSDKV2PreIdentity_completeName(t *testing.T) {
	nsName := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMigrationFrom(t, sdkv2PreIdentityProviderVersion,
		testAccKubernetesNamespaceV1Config_completeName(nsName))
}

func TestAccNamespace_UpgradeFromSDKV2PreIdentity_completeGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMigrationFrom(t, sdkv2PreIdentityProviderVersion,
		testAccKubernetesNamespaceV1Config_completeGeneratedName(prefix))
}

func testAccKubernetesNamespaceV1Config_basic(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = "%s"
  }
}
`, nsName)
}

func testAccKubernetesNamespaceV1Config_Annotations(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }
    name = "%s"
  }
}
`, nsName)
}

func testAccKubernetesNamespaceV1Config_Labels(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    name = "%s"
  }
}
`, nsName)
}

func testAccKubernetesNamespaceV1Config_Annotations_Labels(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    name = "%s"
  }
}
`, nsName)
}

func testAccKubernetesNamespaceV1Config_generatedName(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    generate_name = "%s"
  }
}
`, prefix)
}

func testAccKubernetesNamespaceV1Config_timeouts(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = "%s"
  }

  timeouts {
    delete = "10m"
  }
}
`, nsName)
}

// kubernetes.io/metadata.name is set by the API server to the namespace name, and the
// filtering rules drop it unless the practitioner declared it. Declaring it with the
// value the server would assign keeps the config valid while exercising that exemption.
func testAccKubernetesNamespaceV1Config_declaredInternalLabel(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    labels = {
      "kubernetes.io/metadata.name" = "%s"
    }

    name = "%s"
  }
}
`, nsName, nsName)
}

// generate_name is omitted rather than set to "": both providers validate it as a DNS
// label prefix, and an empty string fails validation before the migration is exercised.
func testAccKubernetesNamespaceV1Config_emptyValuesName(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    annotations = {}
    labels      = {}
    name        = "%s"
  }

  wait_for_default_service_account = false

  timeouts {}
}
`, nsName)
}

func testAccKubernetesNamespaceV1Config_completeName(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    name = "%s"
  }

  wait_for_default_service_account = true

  timeouts {
    delete = "10m"
  }
}
`, nsName)
}

func testAccKubernetesNamespaceV1Config_emptyValuesGeneratedName(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    annotations   = {}
    generate_name = "%s"
    labels        = {}
  }

  wait_for_default_service_account = false

  timeouts {}
}
`, prefix)
}

func testAccKubernetesNamespaceV1Config_completeGeneratedName(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }

    generate_name = "%s"

    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }
  }

  wait_for_default_service_account = true

  timeouts {
    delete = "10m"
  }
}
`, prefix)
}

// ignoredMetadataKey is filtered out of state by common.FlattenMetadata for every resource,
// so Terraform never manages it — the same position a key matching ignore_annotations is in,
// without needing provider configuration. The API server does not restore it, which is what
// makes it usable as evidence.
const ignoredMetadataKey = "example.kubernetes.io/owner"

// testAccInjectIgnoredNamespaceMetadata writes ignoredMetadataKey into one of the namespace's
// metadata maps, standing in for another controller.
func testAccInjectIgnoredNamespaceMetadata(t *testing.T, nsName, field, key string) {
	t.Helper()

	client, err := mainClientset()
	if err != nil {
		t.Fatal(err)
	}
	patch := fmt.Sprintf(`{"metadata":{%q:{%q:"controller-owned"}}}`, field, key)
	if _, err := client.CoreV1().Namespaces().Patch(context.Background(), nsName,
		k8stypes.MergePatchType, []byte(patch), metav1.PatchOptions{}); err != nil {
		t.Fatal(err)
	}
}

// testAccStoreNamespace records the live namespace by name, for capturing a baseline before
// the resource exists at the Terraform address under test — during a move, step 1 manages it
// as kubernetes_namespace, so the usual Exists check cannot run there.
func testAccStoreNamespace(nsName string, out *k8sv1.Namespace) resource.TestCheckFunc {
	return func(*terraform.State) error {
		client, err := mainClientset()
		if err != nil {
			return err
		}
		ns, err := client.CoreV1().Namespaces().Get(context.Background(), nsName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		*out = *ns
		return nil
	}
}

// testAccCheckIgnoredNamespaceMetadata reads the live object: the ignored key is absent from
// both the plan and state, so only the API server can say whether it survived.
func testAccCheckIgnoredNamespaceMetadata(nsName, field, key string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		client, err := mainClientset()
		if err != nil {
			return err
		}
		ns, err := client.CoreV1().Namespaces().Get(context.Background(), nsName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		live := map[string]map[string]string{"annotations": ns.Annotations, "labels": ns.Labels}
		if got := live[field][key]; got != "controller-owned" {
			return fmt.Errorf("ignored %s[%q] = %q, expected %q (live: %v)",
				field, key, got, "controller-owned", live[field])
		}
		return nil
	}
}

// testAccKubernetesNamespaceV1Config_managedMetadata is the shape both upgrade paths use for
// ignored-metadata coverage: one managed key in field, the other map never mentioned.
func testAccKubernetesNamespaceV1Config_managedMetadata(nsName, field, value string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    %s = {
      managed = %q
    }

    name = %q
  }
}
`, field, value, nsName)
}

// Metadata written by another controller must survive the upgrade to this provider AND the
// first update after it.
//
// The upgrade itself is the easy half — it plans nothing, so nothing can be destroyed. The
// dangerous half is the update that follows: prior state written by SDKv2 has no keys for the
// other map, so Update would diff an empty map against an empty map and, without the skip in
// common.MetadataPatchOps, replace the live map with {}.
//
// Both directions are covered, since annotations and labels are separate operations built
// from separate values.
func TestAccNamespace_UpgradeFromSDKV2_ignoredMetadataSurvivesUpdate(t *testing.T) {
	for _, tc := range []struct{ ignored, managed string }{
		{ignored: "annotations", managed: "labels"},
		{ignored: "labels", managed: "annotations"},
	} {
		t.Run(tc.ignored, func(t *testing.T) {
			var before, after k8sv1.Namespace
			nsName := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
			sdkv2Providers := map[string]resource.ExternalProvider{
				"kubernetes": {VersionConstraint: sdkv2ProviderVersion, Source: "hashicorp/kubernetes"},
			}

			resource.ParallelTest(t, resource.TestCase{
				PreCheck:     func() { testAccPreCheck(t) },
				CheckDestroy: testAccCheckKubernetesNamespaceV1Destroy,
				Steps: []resource.TestStep{
					{
						ExternalProviders: sdkv2Providers,
						Config:            testAccKubernetesNamespaceV1Config_managedMetadata(nsName, tc.managed, "before"),
						Check:             testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &before),
					},
					{
						// Another controller writes to the map Terraform does not manage,
						// while SDKv2 still owns the resource.
						PreConfig:                func() { testAccInjectIgnoredNamespaceMetadata(t, nsName, tc.ignored, ignoredMetadataKey) },
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Config:                   testAccKubernetesNamespaceV1Config_managedMetadata(nsName, tc.managed, "before"),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: testAccCheckIgnoredNamespaceMetadata(nsName, tc.ignored, ignoredMetadataKey),
					},
					{
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Config:                   testAccKubernetesNamespaceV1Config_managedMetadata(nsName, tc.managed, "after"),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(namespaceResourceName, plancheck.ResourceActionUpdate)},
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &after),
							testAccCheckNamespaceNotRecreated(&before, &after),
							resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0."+tc.managed+".managed", "after"),
							testAccCheckIgnoredNamespaceMetadata(nsName, tc.ignored, ignoredMetadataKey),
						),
					},
				},
			})
		})
	}
}
