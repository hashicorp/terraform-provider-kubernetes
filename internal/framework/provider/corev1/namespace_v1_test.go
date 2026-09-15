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
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const resourceName = "kubernetes_namespace_v1.test"

// metadataNull asserts that a metadata attribute is null rather than a known
// empty value.
//
// The SDKv2 tests this file replaces asserted `metadata.0.annotations.% == "0"`,
// because SDKv2 zero-filled every leaf of a block it wrote. Under the Framework
// an omitted Optional attribute is null, and that difference is the substance of
// this migration rather than a slackened assertion — so it is asserted
// explicitly here, against the state JSON, in both directions
// (see also TestAccKubernetesNamespaceV1_explicitEmptyMaps).
func metadataNull(attr string) statecheck.StateCheck {
	return statecheck.ExpectKnownValue(
		resourceName,
		tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey(attr),
		knownvalue.Null(),
	)
}

func metadataEmptyMap(attr string) statecheck.StateCheck {
	return statecheck.ExpectKnownValue(
		resourceName,
		tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey(attr),
		knownvalue.MapSizeExact(0),
	)
}

func TestAccKubernetesNamespaceV1_basic(t *testing.T) {
	var conf corev1.Namespace
	nsName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceV1Config_basic(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", nsName),
					resource.TestCheckResourceAttr(resourceName, "id", nsName),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					metadataNull("annotations"),
					metadataNull("labels"),
					metadataNull("generate_name"),
				},
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "wait_for_default_service_account"},
			},
			{
				Config: testAccKubernetesNamespaceV1Config_addAnnotations(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", nsName),
				),
				ConfigStateChecks: []statecheck.StateCheck{metadataNull("labels")},
			},
			{
				Config: testAccKubernetesNamespaceV1Config_addLabels(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", nsName),
				),
			},
			{
				Config: testAccKubernetesNamespaceV1Config_smallerLists(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.Different", "1234"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", nsName),
				),
			},
			{
				// Removal. This is the step that proves annotations and labels
				// stayed Optional WITHOUT Computed: making them Optional+Computed
				// to dodge the null/empty problem would make this a no-op and
				// leave the keys on the live object forever.
				Config: testAccKubernetesNamespaceV1Config_noLists(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(resourceName, &conf),
					testAccCheckNamespaceMetadataKeysAbsent(&conf,
						[]string{"TestAnnotationOne", "Different"},
						[]string{"TestLabelOne", "TestLabelThree"}),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", nsName),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					metadataNull("annotations"),
					metadataNull("labels"),
				},
			},
		},
	})
}

// TestAccKubernetesNamespaceV1_explicitEmptyMaps pins the other side of the
// null/empty contract: a practitioner who writes `annotations = {}` configured a
// known empty map, and it must stay a known empty map rather than being
// normalised to null. Only the one-time v0 state upgrader nulls empties, because
// only there are the two cases indistinguishable.
func TestAccKubernetesNamespaceV1_explicitEmptyMaps(t *testing.T) {
	nsName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceV1Config_explicitEmptyMaps(nsName),
				ConfigStateChecks: []statecheck.StateCheck{
					metadataEmptyMap("annotations"),
					metadataEmptyMap("labels"),
				},
			},
			{
				// A second plan on the same configuration must be empty; if the
				// writers normalised the empty maps to null this would loop.
				Config: testAccKubernetesNamespaceV1Config_explicitEmptyMaps(nsName),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestAccKubernetesNamespaceV1_identity(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		CheckDestroy: testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceV1Config_basic(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(
						resourceName, map[string]knownvalue.Check{
							"name":        knownvalue.StringExact(name),
							"api_version": knownvalue.StringExact("v1"),
							"kind":        knownvalue.StringExact("Namespace"),
						},
					),
				},
			},
			{
				ResourceName:    resourceName,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func TestAccKubernetesNamespaceV1_default_service_account(t *testing.T) {
	var nsConf corev1.Namespace
	var saConf corev1.ServiceAccount
	nsName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceV1Config_wait_for_default_service_acccount(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(resourceName, &nsConf),
					testAccCheckKubernetesDefaultServiceAccountExists(resourceName, &saConf),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "wait_for_default_service_account"},
			},
		},
	})
}

func TestAccKubernetesNamespaceV1_generatedName(t *testing.T) {
	var conf corev1.Namespace
	prefix := "tf-acc-test-gen-"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceV1Config_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr(resourceName, "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					metadataNull("annotations"),
					metadataNull("labels"),
				},
			},
			{
				// REGRESSION GUARD. metadata.name is Optional+Computed and
				// ForceNew, and under generate_name it is absent from the
				// configuration, so it plans unknown. If RequiresReplace ran
				// before UseStateForUnknown it would compare that unknown against
				// the stored generated name, never match, and propose destroying
				// and recreating the namespace on every plan — taking every
				// object inside it. The SDKv2 test applied once and could not
				// have caught this.
				Config: testAccKubernetesNamespaceV1Config_generatedName(prefix),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "wait_for_default_service_account"},
			},
		},
	})
}

func TestAccKubernetesNamespaceV1_withSpecialCharacters(t *testing.T) {
	var conf corev1.Namespace
	nsName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceV1Config_specialCharacters(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.myhost.co.uk/any-path", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.Different", "1234"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.myhost.co.uk/any-path", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", nsName),
				),
			},
		},
	})
}

func TestAccKubernetesNamespaceV1_deleteTimeout(t *testing.T) {
	var conf corev1.Namespace
	nsName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceV1Config_deleteTimeout(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "timeouts.delete", "30m"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", nsName),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					metadataNull("annotations"),
					metadataNull("labels"),
				},
			},
		},
	})
}

// TestAccKubernetesNamespaceV1_validation covers every metadata ValidateFunc the
// SDKv2 schema carried (rule K8S-MIGRATE-003). These must fail OFFLINE, during
// validate and plan, with no API call — dropping one would move the failure to a
// mid-apply Kubernetes 422. The test harness plans before applying, so a plain
// ExpectError step does pin plan-time failure.
func TestAccKubernetesNamespaceV1_validation(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// validateName -> NameIsDNSSubdomain
				Config:      testAccKubernetesNamespaceV1Config_basic("Invalid_Name"),
				ExpectError: regexp.MustCompile(`Invalid value`),
				PlanOnly:    true,
			},
			{
				// validateGenerateName -> NameIsDNSLabel
				Config:      testAccKubernetesNamespaceV1Config_generatedName("Invalid_Prefix"),
				ExpectError: regexp.MustCompile(`Invalid value`),
				PlanOnly:    true,
			},
			{
				// validateAnnotations -> IsQualifiedName on every key
				Config:      testAccKubernetesNamespaceV1Config_badAnnotationKey("tf-acc-test-validation"),
				ExpectError: regexp.MustCompile(`Invalid value`),
				PlanOnly:    true,
			},
			{
				// validateLabels -> IsQualifiedName on keys
				Config:      testAccKubernetesNamespaceV1Config_badLabelKey("tf-acc-test-validation"),
				ExpectError: regexp.MustCompile(`Invalid value`),
				PlanOnly:    true,
			},
			{
				// validateLabels -> IsValidLabelValue on values
				Config:      testAccKubernetesNamespaceV1Config_badLabelValue("tf-acc-test-validation"),
				ExpectError: regexp.MustCompile(`Invalid value`),
				PlanOnly:    true,
			},
			{
				// metadata is Required with MaxItems 1; omitting it must still
				// fail at plan rather than panicking in Create.
				Config:      testAccKubernetesNamespaceV1Config_noMetadata(),
				ExpectError: regexp.MustCompile(`(?s)Block.*metadata.*required|Missing.*metadata`),
				PlanOnly:    true,
			},
			{
				// ConflictsWith between name and generate_name.
				Config:      testAccKubernetesNamespaceV1Config_nameAndGenerateName("tf-acc-test-conflict"),
				ExpectError: regexp.MustCompile(`(?s)Invalid Attribute Combination|cannot be specified when`),
				PlanOnly:    true,
			},
		},
	})
}

func testAccCheckKubernetesNamespaceV1Destroy(s *terraform.State) error {
	conn, err := testAccDestroyClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_namespace_v1" {
			continue
		}

		resp, err := conn.CoreV1().Namespaces().Get(ctx, rs.Primary.ID, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Namespace still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesNamespaceV1Exists(n string, obj *corev1.Namespace) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		// The id attribute is what carries the namespace name into
		// rs.Primary.ID. If the migration had dropped it, the testing shim would
		// substitute "id-attribute-not-set" here and this check — and
		// CheckDestroy — would pass vacuously while leaking namespaces.
		if rs.Primary.ID == "" || rs.Primary.ID == "id-attribute-not-set" {
			return fmt.Errorf("resource %s has no usable ID: %q", n, rs.Primary.ID)
		}

		conn, err := testAccDestroyClientset()
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

// testAccCheckNamespaceMetadataKeysAbsent asserts that removing keys from the
// configuration actually removed them from the live object, not just from state.
func testAccCheckNamespaceMetadataKeysAbsent(obj *corev1.Namespace, annotations, labels []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, k := range annotations {
			if _, ok := obj.Annotations[k]; ok {
				return fmt.Errorf("annotation %q is still present on the live namespace", k)
			}
		}
		for _, k := range labels {
			if _, ok := obj.Labels[k]; ok {
				return fmt.Errorf("label %q is still present on the live namespace", k)
			}
		}
		return nil
	}
}

func testAccCheckKubernetesDefaultServiceAccountExists(n string, obj *corev1.ServiceAccount) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn, err := testAccDestroyClientset()
		if err != nil {
			return err
		}

		out, err := conn.CoreV1().ServiceAccounts(rs.Primary.ID).Get(context.TODO(), "default", metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesNamespaceV1Config_basic(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = "%s"
  }
}
`, nsName)
}

func testAccKubernetesNamespaceV1Config_addAnnotations(nsName string) string {
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

func testAccKubernetesNamespaceV1Config_addLabels(nsName string) string {
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

func testAccKubernetesNamespaceV1Config_noLists(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = "%s"
  }
}
`, nsName)
}

func testAccKubernetesNamespaceV1Config_explicitEmptyMaps(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    annotations = {}
    labels      = {}
    name        = "%s"
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

func testAccKubernetesNamespaceV1Config_deleteTimeout(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = "%s"
  }
  timeouts {
    delete = "30m"
  }
}
`, nsName)
}

func testAccKubernetesNamespaceV1Config_wait_for_default_service_acccount(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = "%s"
  }
  wait_for_default_service_account = "true"
}
`, nsName)
}

func testAccKubernetesNamespaceV1Config_badAnnotationKey(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    annotations = {
      "Not A Valid Key!" = "x"
    }
    name = "%s"
  }
}
`, nsName)
}

func testAccKubernetesNamespaceV1Config_badLabelKey(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    labels = {
      "Not A Valid Key!" = "x"
    }
    name = "%s"
  }
}
`, nsName)
}

func testAccKubernetesNamespaceV1Config_badLabelValue(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    labels = {
      "valid-key" = "not a valid label value!"
    }
    name = "%s"
  }
}
`, nsName)
}

func testAccKubernetesNamespaceV1Config_noMetadata() string {
	return `resource "kubernetes_namespace_v1" "test" {
}
`
}

func testAccKubernetesNamespaceV1Config_nameAndGenerateName(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name          = "%s"
    generate_name = "tf-acc-test-gen-"
  }
}
`, nsName)
}
