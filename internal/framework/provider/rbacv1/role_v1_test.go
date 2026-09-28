// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1_test

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccRole_basic(t *testing.T) {
	// Colons are deliberate: RBAC names are path segments, not DNS subdomains, and SDKv2
	// validates them with validateRBACNameFunc (schema_rbac.go). "system:controller:foo" is
	// a real ClusterRole name, so this fixture guards that the RBAC override is in place.
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_role_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccRoleCheckExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.namespace", "default"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttr(resourceName, "rule.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.api_groups.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resource_names.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.api_groups.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.verbs.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.resource_names.#", "0"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.api_groups.*", "core"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.resources.*", "pods"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.verbs.*", "get"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.verbs.*", "list"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.verbs.*", "watch"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.resource_names.*", "foo"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.1.api_groups.*", "apps"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.1.resources.*", "deployments"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.1.verbs.*", "get"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.1.verbs.*", "list"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccRoleConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccRoleCheckExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.Different", "1234"),
					resource.TestCheckNoResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationTwo"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "2"),
					resource.TestCheckNoResourceAttr(resourceName, "metadata.0.labels.TestLabelTwo"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "rule.#", "1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.api_groups.*", "batch"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.resources.*", "jobs"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.api_groups.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resource_names.#", "0"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.verbs.*", "watch"),
				),
			},
		},
	})
}

func TestAccRole_generatedName(t *testing.T) {
	prefix := "tf-acc-test-gen:" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "kubernetes_role_v1.test"
	var uid string

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleConfig_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccRoleCheckExists(resourceName, &uid),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr(resourceName, "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
				),
			},
			{
				Config: strings.Replace(testAccRoleConfig_generatedName(prefix), "generate_name =", "labels = { added = \"yes\" }\n    generate_name =", 1),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)},
				},
				Check: testAccRoleCheckExists(resourceName, &uid),
			},
		},
	})
}

func TestAccRole_metadataUpdate(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_role_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleConfig_labels(name, "acceptance"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.test", "acceptance"),
				),
			},
			{
				Config: testAccRoleConfig_labels(name, "updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.test", "updated"),
				),
			},
		},
	})
}

func TestAccRole_resourceNames(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_role_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleConfig_noResourceNames(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "rule.0.resource_names.#", "0"),
				),
			},
		},
	})
}

func TestAccRole_ruleTransitions(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_role_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleConfig_ruleTransitionsStep0(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "rule.#", "3"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.resources.*", "pods"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.verbs.*", "get"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.1.resources.*", "deployments"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.1.verbs.*", "list"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.2.resources.*", "cronjobs"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.2.verbs.*", "list"),
				),
			},
			{
				Config: testAccRoleConfig_ruleTransitionsStep1(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "rule.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.resources.*", "deployments"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.verbs.*", "get"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.verbs.*", "list"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.1.resources.*", "jobs"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.1.verbs.*", "get"),
				),
			},
			{
				Config: testAccRoleConfig_ruleTransitionsStep2(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "rule.#", "4"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.resources.*", "pods"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.verbs.*", "list"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.1.resources.*", "deployments"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.1.verbs.*", "list"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.2.resources.*", "cronjobs"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.2.verbs.*", "list"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.3.resources.*", "jobs"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.3.verbs.*", "get"),
				),
			},
		},
	})
}

func TestAccRole_identity(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_role_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccRoleConfig_noResourceNames(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(
						resourceName,
						map[string]knownvalue.Check{
							"namespace":   knownvalue.StringExact("default"),
							"name":        knownvalue.StringExact(name),
							"api_version": knownvalue.StringExact("rbac.authorization.k8s.io/v1"),
							"kind":        knownvalue.StringExact("Role"),
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

func TestAccRole_nameAndGenerateName(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	// Setting both is rejected. SDKv2's namespaced schema declared the conflict against an
	// unresolvable path so it never fired, but the combination was never meaningful — the
	// API ignores generate_name whenever name is present. See the note on
	// TestAccRole_movedFromAlias_nameAndGenerateName for the upgrade impact.
	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRoleConfig_nameAndGenerateName(name),
				ExpectError: regexp.MustCompile("(?s)Invalid Attribute Combination"),
			},
		},
	})
}
func TestAccRole_nullMetadata(t *testing.T) {
	name := "tf-acc-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	// A null map value is rejected at plan, matching SDKv2's validateLabels and
	// validateAnnotations. Role previously accepted it and silently dropped the key, so
	// every later plan showed the key being added back.
	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRoleConfig_nullMetadata(name, "null"),
				ExpectError: regexp.MustCompile("(?s)Invalid Attribute Value.*value must be a string"),
			},
		},
	})
}

func testAccRoleConfig_nullMetadata(name, value string) string {
	config := fmt.Sprintf(`
resource "kubernetes_role_v1" "test" {
  metadata {
    name        = %q
    labels      = { keep = "retained", optional = null }
    annotations = { keep = "retained", optional = null }
  }
  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["get"]
  }
}
`, name)
	return strings.ReplaceAll(config, "optional = null", "optional = "+value)
}

func TestAccRole_missingRule(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRoleConfig_noRule(name),
				ExpectError: regexp.MustCompile(`(?s)rule.*(required|at least 1)`),
			},
		},
	})
}

func TestAccRole_missingMetadata(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRoleConfig_noMetadata(),
				ExpectError: regexp.MustCompile(`(?s)metadata.*(required|at least 1)`),
			},
		},
	})
}

func TestAccRole_duplicateMetadata(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRoleConfig_duplicateMetadata(name),
				ExpectError: regexp.MustCompile(`(?i)at most 1`),
			},
		},
	})
}

func TestAccRole_disappears(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_role_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleConfig_noResourceNames(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					testAccRoleDeleteOutOfBand(name),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccRoleDeleteOutOfBand(name string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		conn, err := testAccRoleClient()
		if err != nil {
			return err
		}
		return conn.RbacV1().Roles("default").Delete(context.Background(), name, metav1.DeleteOptions{})
	}
}

func testAccRoleConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_v1" "test" {
  metadata {
    name = %[1]q
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }
    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }
  }

  rule {
    api_groups     = ["core"]
    resources      = ["pods"]
    resource_names = ["foo"]
    verbs          = ["get", "list", "watch"]
  }

  rule {
    api_groups = ["apps"]
    resources  = ["deployments"]
    verbs      = ["get", "list"]
  }
}
`, name)
}

func testAccRoleConfig_modified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_v1" "test" {
  metadata {
    name = %[1]q
    annotations = {
      TestAnnotationOne = "one"
      Different         = "1234"
    }
    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
    }
  }

  rule {
    api_groups = ["batch"]
    resources  = ["jobs"]
    verbs      = ["watch"]
  }
}
`, name)
}

func testAccRoleConfig_generatedName(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_v1" "test" {
  metadata {
    generate_name = %[1]q
  }

  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["get"]
  }
}
`, prefix)
}

func testAccRoleConfig_labels(name, value string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_v1" "test" {
  metadata {
    name = %[1]q
    labels = {
      test = %[2]q
    }
  }

  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["get"]
  }
}
`, name, value)
}

func testAccRoleConfig_noResourceNames(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_v1" "test" {
  metadata {
    name = %[1]q
  }

  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["list"]
  }
}
`, name)
}

func testAccRoleConfig_ruleTransitionsStep0(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_v1" "test" {
  metadata {
    name = %[1]q
  }

  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["get"]
  }

  rule {
    api_groups = [""]
    resources  = ["deployments"]
    verbs      = ["list"]
  }

  rule {
    api_groups = [""]
    resources  = ["cronjobs"]
    verbs      = ["list"]
  }
}
`, name)
}

func testAccRoleConfig_ruleTransitionsStep1(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_v1" "test" {
  metadata {
    name = %[1]q
  }

  rule {
    api_groups = [""]
    resources  = ["deployments"]
    verbs      = ["get", "list"]
  }

  rule {
    api_groups = [""]
    resources  = ["jobs"]
    verbs      = ["get"]
  }
}
`, name)
}

func testAccRoleConfig_ruleTransitionsStep2(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_v1" "test" {
  metadata {
    name = %[1]q
  }

  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["list"]
  }

  rule {
    api_groups = [""]
    resources  = ["deployments"]
    verbs      = ["list"]
  }

  rule {
    api_groups = [""]
    resources  = ["cronjobs"]
    verbs      = ["list"]
  }

  rule {
    api_groups = [""]
    resources  = ["jobs"]
    verbs      = ["get"]
  }
}
`, name)
}

func testAccRoleConfig_nameAndGenerateName(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_v1" "test" {
  metadata {
    name          = %[1]q
    generate_name = %[1]q
  }

  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["get"]
  }
}
`, name)
}

func testAccRoleConfig_noRule(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_v1" "test" {
  metadata {
    name = %[1]q
  }
}
`, name)
}

func testAccRoleConfig_noMetadata() string {
	return `
resource "kubernetes_role_v1" "test" {
  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["get"]
  }
}
`
}

func testAccRoleConfig_duplicateMetadata(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_v1" "test" {
  metadata {
    name = %[1]q
  }
  metadata {
    name = %[1]q
  }

  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["get"]
  }
}
`, name)
}

func TestAccRole_identityImportDefaultNamespace(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_role_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		Steps: []resource.TestStep{
			// Create the role so it exists in the cluster.
			{
				Config: testAccRoleConfig_noResourceNames(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
				),
			},
			// Import by identity WITH explicit namespace — must succeed.
			{
				ResourceName:    resourceName,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func TestAccRole_invalidAnnotationKey(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRoleConfig_badAnnotationKey(),
				ExpectError: regexp.MustCompile(`(?s)Invalid Attribute Value.*annotations\[`),
			},
		},
	})
}

func TestAccRole_invalidLabelValue(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRoleConfig_badLabelValue(),
				ExpectError: regexp.MustCompile(`(?s)Invalid Attribute Value.*labels\[`),
			},
		},
	})
}

func testAccRoleConfig_badAnnotationKey() string {
	return `
resource "kubernetes_role_v1" "test" {
  metadata {
    name = "bad-annotation-role"
    annotations = {
      "Not A Valid Key!" = "value"
    }
  }

  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["get"]
  }
}
`
}

func testAccRoleConfig_badLabelValue() string {
	return `
resource "kubernetes_role_v1" "test" {
  metadata {
    name = "bad-label-role"
    labels = {
      "env" = "value with spaces and !"
    }
  }

  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["get"]
  }
}
`
}

func testAccRoleCheckDestroy(state *terraform.State) error {
	conn, err := testAccRoleClient()
	if err != nil {
		return err
	}
	for _, module := range state.Modules {
		for _, rs := range module.Resources {
			if rs.Type != "kubernetes_role_v1" && rs.Type != "kubernetes_role" {
				continue
			}
			namespace, name, ok := strings.Cut(rs.Primary.ID, "/")
			if !ok {
				return fmt.Errorf("invalid Role ID %q", rs.Primary.ID)
			}
			_, err := conn.RbacV1().Roles(namespace).Get(context.Background(), name, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				continue
			}
			if err != nil {
				return fmt.Errorf("checking destruction of Role %q: %w", rs.Primary.ID, err)
			}
			return fmt.Errorf("Role %q still exists", rs.Primary.ID)
		}
	}
	return nil
}

func testAccRoleCheckExists(address string, priorUID *string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("resource %q not found in state", address)
		}
		namespace, name, ok := strings.Cut(rs.Primary.ID, "/")
		if !ok {
			return fmt.Errorf("invalid Role ID %q", rs.Primary.ID)
		}
		conn, err := testAccRoleClient()
		if err != nil {
			return err
		}
		role, err := conn.RbacV1().Roles(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		uid := string(role.UID)
		if uid == "" || rs.Primary.Attributes["metadata.0.uid"] != uid {
			return fmt.Errorf("Role %q has mismatched API and state UID", rs.Primary.ID)
		}
		if priorUID != nil {
			if *priorUID != "" && *priorUID != uid {
				return fmt.Errorf("Role %q was replaced: UID changed from %q to %q", rs.Primary.ID, *priorUID, uid)
			}
			*priorUID = uid
		}
		return nil
	}
}

func testAccRoleConfig_names(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_v1" "test" {
  metadata {
    name = %[1]q
  }

  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["get"]
  }
}
`, name)
}
