// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	corev1api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

// ── upgrade (same type, SDKv2 state → Framework) ─────────────────────────────

// sdkv2LimitRangeProviderVersion is the last release that served
// kubernetes_limit_range_v1 from the SDKv2 implementation.
const sdkv2LimitRangeProviderVersion = "3.2.1"

// sdkv2PreIdentityLimitRangeProviderVersion predates resource identity (2.38.0).
// State it writes carries identity_schema_version 0 and no identity, so upgrading
// exercises UpgradeIdentity rather than carrying an existing identity across.
const sdkv2PreIdentityLimitRangeProviderVersion = "2.37.1"

// testAccLimitRangeMigration applies config under the last SDKv2 release, then
// re-plans the identical config under the local framework provider and asserts
// the upgrade produces no changes.
//
// Step 2 runs the framework Read path, so these tests exercise FlattenNamespacedMetadata
// and its filtering rather than Create. The two configs must stay byte-identical.
func testAccLimitRangeMigration(t *testing.T, config string) {
	t.Helper()
	testAccLimitRangeMigrationFrom(t, sdkv2LimitRangeProviderVersion, config)
}

func testAccLimitRangeMigrationFrom(t *testing.T, sdkv2Version, config string) {
	t.Helper()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckKubernetesLimitRangeV1Destroy,
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

// testAccLimitRangeMigrationExpectingUpdate is testAccLimitRangeMigration for the one shape
// that cannot migrate cleanly: SDKv2 writes null for a config that says `annotations = {}`
// because its type system cannot represent the difference. The framework can, and Read
// preserves the null it was given — but config still says `{}`, so the first plan after
// upgrade is an in-place update to reconcile them.
//
// An update is the acceptable one-off; a replacement would destroy the object. Asserting
// the specific action (not merely a non-empty plan) keeps that distinction enforced.
func testAccLimitRangeMigrationExpectingUpdate(t *testing.T, config string) {
	t.Helper()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckKubernetesLimitRangeV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						VersionConstraint: sdkv2LimitRangeProviderVersion,
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
						plancheck.ExpectResourceAction(limitRangeResourceName, plancheck.ResourceActionUpdate),
					},
				},
			},
		},
	})
}

// Minimal shape: name + namespace, no spec.
func TestAccLimitRange_UpgradeFromSDKV2_empty(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccLimitRangeMigration(t, testAccKubernetesLimitRangeV1MigrationConfig_empty(name))
}

// Container limit with default and default_request — exercises Optional+Computed map.
func TestAccLimitRange_UpgradeFromSDKV2_basic(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccLimitRangeMigration(t, testAccKubernetesLimitRangeV1MigrationConfig_basic(name))
}

// generate_name: the server assigns a name, so this settles UseStateForUnknown / RequiresReplace.
func TestAccLimitRange_UpgradeFromSDKV2_generatedName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	ns := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccLimitRangeMigration(t, testAccKubernetesLimitRangeV1MigrationConfig_generatedName(prefix, ns))
}

// Annotations populated, labels absent — one map stored null and one not, in the direction
// where the API also adds an internal annotation that has to be filtered back out on Read.
func TestAccLimitRange_UpgradeFromSDKV2_annotations(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccLimitRangeMigration(t, testAccKubernetesLimitRangeV1MigrationConfig_annotations(name))
}

// The mirror: labels populated, annotations absent.
func TestAccLimitRange_UpgradeFromSDKV2_labels(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccLimitRangeMigration(t, testAccKubernetesLimitRangeV1MigrationConfig_labels(name))
}

// A key that the filtering rules would normally strip, declared in config so it must
// survive. Covers the branch where prior state exempts a key from removal.
func TestAccLimitRange_UpgradeFromSDKV2_declaredInternalLabel(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccLimitRangeMigration(t, testAccKubernetesLimitRangeV1MigrationConfig_declaredInternalLabel(name))
}

// Every optional metadata field declared but empty. SDKv2 stores null for `annotations = {}`
// so the first plan after the upgrade reconciles null→{} with an in-place update.
func TestAccLimitRange_UpgradeFromSDKV2_emptyMapsName(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccLimitRangeMigrationExpectingUpdate(t, testAccKubernetesLimitRangeV1MigrationConfig_emptyMapsName(name))
}

// The same empty-maps shape named by generate_name rather than name.
func TestAccLimitRange_UpgradeFromSDKV2_emptyMapsGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	ns := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccLimitRangeMigrationExpectingUpdate(t, testAccKubernetesLimitRangeV1MigrationConfig_emptyMapsGenerateName(prefix, ns))
}

// Every attribute at once — catches a field dropped from FlattenNamespacedMetadata.
func TestAccLimitRange_UpgradeFromSDKV2_completeName(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccLimitRangeMigration(t, testAccKubernetesLimitRangeV1MigrationConfig_complete(name))
}

// The complete shape named by generate_name.
func TestAccLimitRange_UpgradeFromSDKV2_completeGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	ns := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccLimitRangeMigration(t, testAccKubernetesLimitRangeV1MigrationConfig_completeGenerateName(prefix, ns))
}

// Multiple limits in a single spec block.
func TestAccLimitRange_UpgradeFromSDKV2_multipleLimits(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccLimitRangeMigration(t, testAccKubernetesLimitRangeV1MigrationConfig_multipleLimits(name))
}

// Pre-identity: state from before 2.38.0 has no identity at all — exercises UpgradeIdentity.
func TestAccLimitRange_UpgradeFromSDKV2PreIdentity_basicName(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccLimitRangeMigrationFrom(t, sdkv2PreIdentityLimitRangeProviderVersion,
		testAccKubernetesLimitRangeV1MigrationConfig_basic(name))
}

func TestAccLimitRange_UpgradeFromSDKV2PreIdentity_basicGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	ns := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccLimitRangeMigrationFrom(t, sdkv2PreIdentityLimitRangeProviderVersion,
		testAccKubernetesLimitRangeV1MigrationConfig_generatedName(prefix, ns))
}

func TestAccLimitRange_UpgradeFromSDKV2PreIdentity_completeName(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccLimitRangeMigrationFrom(t, sdkv2PreIdentityLimitRangeProviderVersion,
		testAccKubernetesLimitRangeV1MigrationConfig_complete(name))
}

func TestAccLimitRange_UpgradeFromSDKV2PreIdentity_completeGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	ns := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccLimitRangeMigrationFrom(t, sdkv2PreIdentityLimitRangeProviderVersion,
		testAccKubernetesLimitRangeV1MigrationConfig_completeGenerateName(prefix, ns))
}

// ignoredLimitRangeMetadataKey is filtered out of state by FlattenNamespacedMetadata
// so Terraform never manages it — the API server does not restore it, which makes it
// usable as evidence in a live-object assertion.
const ignoredLimitRangeMetadataKey = "example.kubernetes.io/owner"

// testAccInjectIgnoredLimitRangeMetadata writes ignoredLimitRangeMetadataKey into one
// of the limit range's metadata maps, standing in for another controller.
func testAccInjectIgnoredLimitRangeMetadata(t *testing.T, ns, name, field, key string) {
	t.Helper()
	client, err := mainClientset()
	if err != nil {
		t.Fatal(err)
	}
	patch := fmt.Sprintf(`{"metadata":{%q:{%q:"controller-owned"}}}`, field, key)
	if _, err := client.CoreV1().LimitRanges(ns).Patch(context.Background(), name,
		k8stypes.MergePatchType, []byte(patch), metav1.PatchOptions{}); err != nil {
		t.Fatal(err)
	}
}

// testAccCheckIgnoredLimitRangeMetadata reads the live object: the ignored key is absent
// from both plan and state, so only the API server can say whether it survived.
func testAccCheckIgnoredLimitRangeMetadata(ns, name, field, key string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		client, err := mainClientset()
		if err != nil {
			return err
		}
		lr, err := client.CoreV1().LimitRanges(ns).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		live := map[string]map[string]string{"annotations": lr.Annotations, "labels": lr.Labels}
		if got := live[field][key]; got != "controller-owned" {
			return fmt.Errorf("ignored %s[%q] = %q, expected %q (live: %v)",
				field, key, got, "controller-owned", live[field])
		}
		return nil
	}
}

// testAccCheckLimitRangeNotRecreated confirms the object's UID did not change between two
// captured snapshots, which would indicate a destroy+create.
func testAccCheckLimitRangeNotRecreated(before, after *corev1api.LimitRange) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if before.UID != after.UID {
			return fmt.Errorf("LimitRange was recreated: before UID %s, after UID %s", before.UID, after.UID)
		}
		return nil
	}
}

// testAccStoreLimitRange records the live limit range for a before/after UID comparison.
func testAccStoreLimitRange(ns, name string, out *corev1api.LimitRange) resource.TestCheckFunc {
	return func(*terraform.State) error {
		client, err := mainClientset()
		if err != nil {
			return err
		}
		lr, err := client.CoreV1().LimitRanges(ns).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		*out = *lr
		return nil
	}
}

// Metadata written by another controller must survive the upgrade AND the first update
// after it. The dangerous half is the update: prior SDKv2 state has no keys for the
// unmanaged map, so Update would replace the live map with {} without common.MetadataPatchOps'
// skip guard. Both annotation↔label directions are covered.
func TestAccLimitRange_UpgradeFromSDKV2_ignoredMetadataSurvivesUpdate(t *testing.T) {
	for _, tc := range []struct{ ignored, managed string }{
		{ignored: "annotations", managed: "labels"},
		{ignored: "labels", managed: "annotations"},
	} {
		t.Run(tc.ignored, func(t *testing.T) {
			var before, after corev1api.LimitRange
			name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
			sdkv2Providers := map[string]resource.ExternalProvider{
				"kubernetes": {VersionConstraint: sdkv2LimitRangeProviderVersion, Source: "hashicorp/kubernetes"},
			}

			resource.ParallelTest(t, resource.TestCase{
				PreCheck:     func() { testAccPreCheck(t) },
				CheckDestroy: testAccCheckKubernetesLimitRangeV1Destroy,
				Steps: []resource.TestStep{
					{
						ExternalProviders: sdkv2Providers,
						Config:            testAccKubernetesLimitRangeV1MigrationConfig_managedMetadata(name, tc.managed, "before"),
						Check:             testAccStoreLimitRange(name, name, &before),
					},
					{
						// Another controller writes to the unmanaged map while SDKv2 still owns it.
						PreConfig:                func() { testAccInjectIgnoredLimitRangeMetadata(t, name, name, tc.ignored, ignoredLimitRangeMetadataKey) },
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Config:                   testAccKubernetesLimitRangeV1MigrationConfig_managedMetadata(name, tc.managed, "before"),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: testAccCheckIgnoredLimitRangeMetadata(name, name, tc.ignored, ignoredLimitRangeMetadataKey),
					},
					{
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Config:                   testAccKubernetesLimitRangeV1MigrationConfig_managedMetadata(name, tc.managed, "after"),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(limitRangeResourceName, plancheck.ResourceActionUpdate)},
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							testAccCheckKubernetesLimitRangeV1Exists(limitRangeResourceName, &after),
							testAccCheckLimitRangeNotRecreated(&before, &after),
							resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0."+tc.managed+".managed", "after"),
							testAccCheckIgnoredLimitRangeMetadata(name, name, tc.ignored, ignoredLimitRangeMetadataKey),
						),
					},
				},
			})
		})
	}
}

// ── migration HCL configs ─────────────────────────────────────────────────────

func testAccKubernetesLimitRangeV1MigrationConfig_empty(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
}
`, name)
}

func testAccKubernetesLimitRangeV1MigrationConfig_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }

  spec {
    limit {
      type = "Container"

      default = {
        cpu    = "200m"
        memory = "512M"
      }

      default_request = {
        cpu    = "100m"
        memory = "256M"
      }
    }
  }
}
`, name)
}

func testAccKubernetesLimitRangeV1MigrationConfig_generatedName(prefix, ns string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
  metadata {
    generate_name = %[2]q
    namespace     = kubernetes_namespace_v1.test.metadata.0.name
  }

  spec {
    limit {
      type = "Pod"
    }
  }
}
`, ns, prefix)
}

func testAccKubernetesLimitRangeV1MigrationConfig_multipleLimits(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }

  spec {
    limit {
      type = "Pod"

      max = {
        cpu    = "200m"
        memory = "1024M"
      }
    }

    limit {
      type = "PersistentVolumeClaim"

      min = {
        storage = "24M"
      }
    }

    limit {
      type = "Container"

      default = {
        cpu    = "50m"
        memory = "24M"
      }
    }
  }
}
`, name)
}

func testAccKubernetesLimitRangeV1MigrationConfig_annotations(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }

    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
}
`, name)
}

func testAccKubernetesLimitRangeV1MigrationConfig_labels(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
  metadata {
    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
}
`, name)
}

// kubernetes.io/metadata.name is set by the API server to the namespace name; the
// filtering rules drop it unless the practitioner declared it. Declaring it with the
// value the server would assign keeps the config valid while exercising that exemption.
func testAccKubernetesLimitRangeV1MigrationConfig_declaredInternalLabel(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
  metadata {
    labels = {
      "kubernetes.io/metadata.name" = %[1]q
    }

    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
}
`, name)
}

// generate_name is omitted rather than set to "": both providers validate it as a DNS
// label prefix, and an empty string fails validation before the migration is exercised.
func testAccKubernetesLimitRangeV1MigrationConfig_emptyMapsName(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
  metadata {
    annotations = {}
    labels      = {}
    name        = %[1]q
    namespace   = kubernetes_namespace_v1.test.metadata.0.name
  }
}
`, name)
}

func testAccKubernetesLimitRangeV1MigrationConfig_emptyMapsGenerateName(prefix, ns string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
  metadata {
    annotations  = {}
    generate_name = %[2]q
    labels       = {}
    namespace    = kubernetes_namespace_v1.test.metadata.0.name
  }
}
`, ns, prefix)
}

func testAccKubernetesLimitRangeV1MigrationConfig_complete(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
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

    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }

  spec {
    limit {
      type = "Container"

      default = {
        cpu    = "200m"
        memory = "512M"
      }

      default_request = {
        cpu    = "100m"
        memory = "256M"
      }
    }
  }
}
`, name)
}

func testAccKubernetesLimitRangeV1MigrationConfig_completeGenerateName(prefix, ns string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }

    generate_name = %[2]q

    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }

  spec {
    limit {
      type = "Container"

      default = {
        cpu    = "200m"
        memory = "512M"
      }

      default_request = {
        cpu    = "100m"
        memory = "256M"
      }
    }
  }
}
`, ns, prefix)
}

// testAccKubernetesLimitRangeV1MigrationConfig_managedMetadata is the shape both
// upgrade and move-state tests use for the ignored-metadata coverage: one managed key
// in the given field, the other map never mentioned.
func testAccKubernetesLimitRangeV1MigrationConfig_managedMetadata(name, field, value string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
  metadata {
    %[2]s = {
      managed = %[3]q
    }

    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
}
`, name, field, value)
}
