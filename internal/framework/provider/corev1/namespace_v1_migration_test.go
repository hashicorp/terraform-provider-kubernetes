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
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
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

// The baseline shape: name only, both metadata maps absent. Exercises the filtering
// that strips the API server's kubernetes.io/metadata.name back out on Read.
func TestAccNamespace_UpgradeFromSDKV2(t *testing.T) {
	nsName := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMigration(t, testAccKubernetesNamespaceV1Config_basic(nsName))
}

// sdkv2ProviderVersion is the last release that served kubernetes_namespace_v1 from
// the SDKv2 implementation. Step 1 of every migration test runs against it.
const sdkv2ProviderVersion = "3.2.1"

// testAccNamespaceMigration applies config under the last released SDKv2 provider,
// then re-plans the identical config under the local framework provider and asserts
// the upgrade produces no changes.
//
// Step 2 refreshes state written by step 1 through the framework Read, so these tests
// exercise common.FlattenMetadata and its filtering rather than Create. The two configs
// must stay byte-identical: any difference makes a non-empty plan ambiguous.
func testAccNamespaceMigration(t *testing.T, config string) {
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

// testAccNamespaceMoveState applies a kubernetes_namespace under the released SDKv2
// provider, then switches to kubernetes_namespace_v1 under the local framework build with
// a `moved` block, and asserts the upgrade plans nothing.
//
//	moved {
//	  from = kubernetes_namespace.test
//	  to   = kubernetes_namespace_v1.test
//	}
//
// An empty plan proves MoveState produced state this provider already considers correct.
// Without MoveState these fail outright — Terraform cannot move state across resource types
// unless the destination provider implements it. Cross-type `moved` needs Terraform 1.8+,
// a lower floor than the 1.12 the identity tests need.
//
// CheckDestroy matches both type names: a failure at step 2 leaves state under the old
// address, which would otherwise leak a namespace silently.
func testAccNamespaceMoveState(t *testing.T, sourceConfig, movedConfig string) {
	t.Helper()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		CheckDestroy: testAccCheckKubernetesNamespaceV1Destroy,
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
				Config:                   movedConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// The baseline shape: a named namespace with labels and annotations.
func TestAccNamespace_MoveStateFromUnversioned(t *testing.T) {
	nsName := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMoveState(t,
		testAccKubernetesNamespaceConfig_unversioned(nsName),
		testAccKubernetesNamespaceConfig_movedToV1(nsName))
}

// generate_name: metadata.name is server-assigned, and is Optional+Computed with both
// UseStateForUnknown and RequiresReplace. If the move mishandled it — dropping it, or
// leaving it unknown — the plan would be a replacement rather than empty, destroying the
// namespace. This is also the shape where SDKv2 stores a real generate_name value rather
// than the "" it writes when the field was never set.
func TestAccNamespace_MoveStateFromUnversioned_generateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMoveState(t,
		testAccKubernetesNamespaceConfig_unversionedGenerateName(prefix),
		testAccKubernetesNamespaceConfig_movedToV1GenerateName(prefix))
}

// timeouts is the one attribute whose type cannot be eyeballed — SDKv2 injects it as a
// single-nested block of strings (helper/schema/core_schema.go:328), and a MoveState
// implementation that rebuilds it by hand can easily get the attribute set wrong or drop
// it entirely. Dropping it leaves state null against a config that sets it, so the plan is
// not empty and this fails. The other move tests declare no timeouts block and would pass
// either way.
func TestAccNamespace_MoveStateFromUnversioned_deleteTimeout(t *testing.T) {
	nsName := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMoveState(t,
		testAccKubernetesNamespaceConfig_unversionedDeleteTimeout(nsName),
		testAccKubernetesNamespaceConfig_movedToV1DeleteTimeout(nsName))
}

func testAccKubernetesNamespaceConfig_unversioned(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
    }

    labels = {
      TestLabelOne = "one"
    }

    name = "%s"
  }
}
`, nsName)
}

func testAccKubernetesNamespaceConfig_movedToV1(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
    }

    labels = {
      TestLabelOne = "one"
    }

    name = "%s"
  }
}

moved {
  from = kubernetes_namespace.test
  to   = kubernetes_namespace_v1.test
}
`, nsName)
}

func testAccKubernetesNamespaceConfig_unversionedGenerateName(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace" "test" {
  metadata {
    generate_name = "%s"
  }
}
`, prefix)
}

func testAccKubernetesNamespaceConfig_movedToV1GenerateName(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    generate_name = "%s"
  }
}

moved {
  from = kubernetes_namespace.test
  to   = kubernetes_namespace_v1.test
}
`, prefix)
}

func testAccKubernetesNamespaceConfig_unversionedDeleteTimeout(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace" "test" {
  metadata {
    name = "%s"
  }

  timeouts {
    delete = "20s"
  }
}
`, nsName)
}

func testAccKubernetesNamespaceConfig_movedToV1DeleteTimeout(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = "%s"
  }

  timeouts {
    delete = "20s"
  }
}

moved {
  from = kubernetes_namespace.test
  to   = kubernetes_namespace_v1.test
}
`, nsName)
}
