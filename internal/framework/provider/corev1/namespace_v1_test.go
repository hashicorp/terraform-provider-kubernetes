// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
)

const namespaceResourceName = "kubernetes_namespace_v1.test"

// mainClientset resolves a Kubernetes client the same way the resource does at runtime:
// through the SDKv2 provider meta that the framework provider is handed at configure
// time. The SDKv2 test file uses the package-level testAccProvider, which does not exist
// in this package.
func mainClientset() (*k8sclient.Clientset, error) {
	return sdkv2providerMeta()().(kubernetes.KubeClientsets).MainClientset()
}

// nullMetadataMap asserts that a metadata map is null rather than an empty map.
//
// SDKv2 could not tell the two apart, so its tests assert `metadata.0.labels.% == 0`.
// The framework can, and an omitted map stays null — which has no "%" entry in the
// flatmap at all, so the SDKv2 form of the assertion fails with "attribute not found"
// rather than a value mismatch. Asserting null directly also catches a regression that
// turns null into an empty map, which is the failure mode findings §3 is about.
func nullMetadataMap(attr string) statecheck.StateCheck {
	return statecheck.ExpectKnownValue(
		namespaceResourceName,
		tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey(attr),
		knownvalue.Null(),
	)
}

func TestAccKubernetesNamespaceV1_basic(t *testing.T) {
	var conf k8sv1.Namespace
	nsName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceV1Config_basic(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &conf),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.name", nsName),
					resource.TestCheckResourceAttrSet(namespaceResourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(namespaceResourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(namespaceResourceName, "metadata.0.uid"),
				),
				// The API server always adds kubernetes.io/metadata.name. These two
				// assertions are the acceptance-level proof that Read filters it out.
				ConfigStateChecks: []statecheck.StateCheck{
					nullMetadataMap("annotations"),
					nullMetadataMap("labels"),
				},
			},
			{
				ResourceName:      namespaceResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// resource_version changes on every server-side write, so it can never
				// round-trip. wait_for_default_service_account is deliberately NOT
				// ignored here even though the SDKv2 suite ignores it: SDKv2's import
				// leaves it unset too, and inheriting the exemption would hide whether
				// the framework has the same gap. If this fails on that attribute, the
				// fix is in ImportState or Read, not in this list.
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesNamespaceV1Config_Annotations(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &conf),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.name", nsName),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					nullMetadataMap("labels"),
				},
			},
			{
				Config: testAccKubernetesNamespaceV1Config_Annotations_Labels(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &conf),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.labels.TestLabelThree", "three"),
				),
			},
			{
				// Shrinks both maps and changes a value: exercises Remove, Replace and
				// Add operations in a single patch.
				Config: testAccKubernetesNamespaceV1Config_smallerLists(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &conf),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.annotations.Different", "1234"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.labels.TestLabelThree", "three"),
				),
			},
			{
				// Removing the maps entirely. Every key the practitioner declared is
				// patched away, but the server's own label must not come back into state.
				Config: testAccKubernetesNamespaceV1Config_basic(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &conf),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.name", nsName),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					nullMetadataMap("annotations"),
					nullMetadataMap("labels"),
				},
			},
		},
	})
}

func TestAccKubernetesNamespaceV1_identity(t *testing.T) {
	nsName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		CheckDestroy: testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceV1Config_basic(nsName),
				// api_version and kind are literals because client-go's typed clients
				// clear TypeMeta on responses — the resource hardcodes them for the
				// same reason.
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(namespaceResourceName, map[string]knownvalue.Check{
						"name":        knownvalue.StringExact(nsName),
						"api_version": knownvalue.StringExact("v1"),
						"kind":        knownvalue.StringExact("Namespace"),
					}),
				},
			},
			{
				ResourceName:    namespaceResourceName,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func TestAccKubernetesNamespaceV1_default_service_account(t *testing.T) {
	var nsConf k8sv1.Namespace
	var saConf k8sv1.ServiceAccount
	nsName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceV1Config_waitForDefaultServiceAccount(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &nsConf),
					testAccCheckKubernetesDefaultServiceAccountExists(namespaceResourceName, &saConf),
					resource.TestCheckResourceAttr(namespaceResourceName, "wait_for_default_service_account", "true"),
				),
			},
			{
				ResourceName:            namespaceResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
		},
	})
}

func TestAccKubernetesNamespaceV1_generatedName(t *testing.T) {
	var conf k8sv1.Namespace
	prefix := "tf-acc-test-gen-"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceV1Config_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &conf),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.generate_name", prefix),
					// name is server-assigned, so this is the shape where it is unknown
					// at plan time — the case that decides whether the schema needs
					// UseStateForUnknown.
					resource.TestMatchResourceAttr(namespaceResourceName, "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet(namespaceResourceName, "metadata.0.uid"),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					nullMetadataMap("annotations"),
					nullMetadataMap("labels"),
				},
			},
			{
				ResourceName:            namespaceResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
		},
	})
}

// Keys containing "/" and "." are the reason escapeJsonPointer exists: "/" separates
// tokens in an RFC 6901 pointer and must be written as "~1".
//
// The SDKv2 suite only creates such keys, and a create sends the whole object rather
// than a patch — so the escaping was never actually exercised. The second step here
// changes one escaped key's value and removes another, producing Replace and Remove
// operations against escaped paths.
func TestAccKubernetesNamespaceV1_withSpecialCharacters(t *testing.T) {
	var conf k8sv1.Namespace
	nsName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceV1Config_specialCharacters(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &conf),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.annotations.myhost.co.uk/any-path", "one"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.annotations.Different", "1234"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.labels.myhost.co.uk/any-path", "one"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.labels.TestLabelThree", "three"),
				),
			},
			{
				Config: testAccKubernetesNamespaceV1Config_specialCharactersUpdated(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &conf),
					// Replace against an escaped path.
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.annotations.myhost.co.uk/any-path", "two"),
					// Remove against an escaped path.
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.annotations.%", "1"),
					resource.TestCheckNoResourceAttr(namespaceResourceName, "metadata.0.annotations.Different"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.labels.myhost.co.uk/any-path", "two"),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.labels.%", "1"),
				),
			},
		},
	})
}

func TestAccKubernetesNamespaceV1_deleteTimeout(t *testing.T) {
	var conf k8sv1.Namespace
	nsName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceV1Config_timeouts(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &conf),
					resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0.name", nsName),
					resource.TestCheckResourceAttr(namespaceResourceName, "timeouts.delete", "10m"),
				),
			},
		},
	})
}

func testAccCheckKubernetesNamespaceV1Destroy(s *terraform.State) error {
	conn, err := mainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_namespace_v1" {
			continue
		}

		resp, err := conn.CoreV1().Namespaces().Get(ctx, rs.Primary.ID, metav1.GetOptions{})
		if err == nil && resp.Name == rs.Primary.ID {
			return fmt.Errorf("namespace still exists: %s", rs.Primary.ID)
		}
	}

	return nil
}

func testAccCheckKubernetesNamespaceV1Exists(n string, obj *k8sv1.Namespace) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		conn, err := mainClientset()
		if err != nil {
			return err
		}

		out, err := conn.CoreV1().Namespaces().Get(context.TODO(), rs.Primary.ID, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccCheckKubernetesDefaultServiceAccountExists(n string, obj *k8sv1.ServiceAccount) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		conn, err := mainClientset()
		if err != nil {
			return err
		}

		// The namespace resource ID is the namespace name; the SDKv2 test routed it
		// through idParts, which is unexported and unnecessary for a cluster-scoped
		// resource.
		out, err := conn.CoreV1().ServiceAccounts(rs.Primary.ID).Get(context.TODO(), "default", metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesNamespaceV1Config_smallerLists(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      Different         = "1234"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
    }

    name = "%s"
  }
}
`, nsName)
}

func testAccKubernetesNamespaceV1Config_specialCharacters(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    annotations = {
      "myhost.co.uk/any-path" = "one"
      "Different"             = "1234"
    }

    labels = {
      "myhost.co.uk/any-path" = "one"
      "TestLabelThree"        = "three"
    }

    name = "%s"
  }
}
`, nsName)
}

// Changes the value of the "/"-containing key and drops its sibling, so the resulting
// patch contains a Replace and a Remove whose paths both require ~1 escaping.
func testAccKubernetesNamespaceV1Config_specialCharactersUpdated(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    annotations = {
      "myhost.co.uk/any-path" = "two"
    }

    labels = {
      "myhost.co.uk/any-path" = "two"
    }

    name = "%s"
  }
}
`, nsName)
}

func testAccKubernetesNamespaceV1Config_waitForDefaultServiceAccount(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = "%s"
  }

  wait_for_default_service_account = true
}
`, nsName)
}
