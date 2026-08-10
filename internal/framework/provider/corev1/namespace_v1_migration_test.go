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

func TestAccNamespace_UpgradeFromSDKV2(t *testing.T){
	nsName := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resource.ParallelTest(t,resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes":{
						VersionConstraint:"3.2.1",
						Source: "hashicorp/kubernetes",
					},
				},
				Config: testAccKubernetesNamespaceV1Config_basic(nsName),
				Check: 	resource.ComposeTestCheckFunc(),
			},
			{
				ProtoV6ProviderFactories:testAccProtoV6ProviderFactories,
				Config: testAccKubernetesNamespaceV1Config_basic(nsName),
				ConfigPlanChecks: resource.ConfigPlanChecks{
                    PreApply: []plancheck.PlanCheck{
                        plancheck.ExpectEmptyPlan(),
                    },
                },
			},
		},
	})
}

// sdkv2ProviderVersion is the last release that served kubernetes_namespace_v1 from
// the SDKv2 implementation. Step 1 of every migration test runs against it.
const sdkv2ProviderVersion = "3.2.1"

// testAccNamespaceMigration applies config under the last released SDKv2 provider,
// then re-plans the identical config under the local framework provider and asserts
// the upgrade produces no changes.
//
// Step 2 refreshes state written by step 1 through the framework Read, so these tests
// exercise flattenMetadata and filterMetadataMap rather than Create. The two configs
// must stay byte-identical: any difference makes a non-empty plan ambiguous.
func testAccNamespaceMigration(t *testing.T, config string) {
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
						plancheck.ExpectEmptyPlan(),
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
func TestAccNamespace_UpgradeFromSDKV2_emptyValues(t *testing.T) {
	nsName := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesNamespaceV1Config_emptyValues(nsName)

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
						plancheck.ExpectResourceAction("kubernetes_namespace_v1.test", plancheck.ResourceActionUpdate),
					},
				},
			},
		},
	})
}

// Every optional field declared with a real value.
func TestAccNamespace_UpgradeFromSDKV2_complete(t *testing.T) {
	nsName := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMigration(t, testAccKubernetesNamespaceV1Config_complete(nsName))
}

// timeouts in isolation, kept separate so a failure here is not confounded with the
// metadata shapes. SDKv2 turns out to persist timeouts as a real state attribute
// (`"timeouts": {"delete": null}`), not as out-of-band instance metadata, so the
// framework's timeouts block reads it back directly and this shape migrates cleanly.
func TestAccNamespace_UpgradeFromSDKV2_timeouts(t *testing.T) {
	nsName := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMigration(t, testAccKubernetesNamespaceV1Config_timeouts(nsName))
}

// A key that the filtering rules would normally strip, declared in config so that it
// must survive. Covers the branch where prior state exempts a key from removal.
func TestAccNamespace_UpgradeFromSDKV2_declaredInternalLabel(t *testing.T) {
	nsName := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMigration(t, testAccKubernetesNamespaceV1Config_declaredInternalLabel(nsName))
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

// generate_name is omitted rather than set to "": both providers validate it as a DNS
// label prefix, and an empty string fails validation before the migration is exercised.
func testAccKubernetesNamespaceV1Config_emptyValues(nsName string) string {
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

func testAccKubernetesNamespaceV1Config_complete(nsName string) string {
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



