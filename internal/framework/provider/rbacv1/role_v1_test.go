// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccRole_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_role_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.namespace", "default"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttr(resourceName, "rule.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.api_groups.*", ""),
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
				ImportStateVerifyIgnore: []string{
					"metadata.0.resource_version",
				},
			},
			{
				Config: testAccRoleConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "rule.#", "1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.api_groups.*", "batch"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.resources.*", "jobs"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.verbs.*", "get"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.verbs.*", "list"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.verbs.*", "watch"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.verbs.*", "create"),
				),
			},
		},
	})
}

func TestAccRole_generatedName(t *testing.T) {
	prefix := "tf-acc-test-gen-"
	resourceName := "kubernetes_role_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleConfig_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr(resourceName, "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
				),
			},
		},
	})
}

func TestAccRole_metadataUpdate(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_role_v1.test"

	resource.ParallelTest(t, resource.TestCase{
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

// TestAccRole_nameAndGenerateNameConflict exercises the
// stringvalidator.ConflictsWith validators on metadata.name/generate_name —
// previously untested even though the validators have existed since the
// initial migration.
func TestAccRole_nameAndGenerateNameConflict(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRoleConfig_nameAndGenerateNameConflict(name),
				ExpectError: regexp.MustCompile(`(?i)cannot be specified when`),
			},
		},
	})
}

// TestAccRole_missingRule exercises Role.ValidateConfig's "rule block
// missing entirely" check (role_v1.go) — no rule blocks at all should be
// rejected at plan time, matching the SDKv2 schema's Required rule field.
// This specifically is NOT caught by listvalidator.SizeAtLeast(1) on its
// own: that validator skips null values, and a completely omitted
// ListNestedBlock is null, not an empty list — see the comment on
// ValidateConfig for how this was found.
func TestAccRole_missingRule(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRoleConfig_noRule(name),
				ExpectError: regexp.MustCompile(`(?i)at least one "rule" block is required`),
			},
		},
	})
}

// TestAccRole_missingMetadata is the metadata-block counterpart of
// TestAccRole_missingRule: a config with no metadata block at all should be
// rejected at plan time by ValidateConfig (role_v1.go), the same gap
// SizeBetween(1, 1) alone doesn't cover.
func TestAccRole_missingMetadata(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRoleConfig_noMetadata(),
				ExpectError: regexp.MustCompile(`(?i)at least one "metadata" block is required`),
			},
		},
	})
}

// TestAccRole_duplicateMetadata exercises the metadata ListNestedBlock's
// listvalidator.SizeBetween(1, 1) — declaring the block twice should be
// rejected at plan time, the same way SDKv2's MaxItems: 1 rejected it.
func TestAccRole_duplicateMetadata(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRoleConfig_duplicateMetadata(name),
				ExpectError: regexp.MustCompile(`(?i)at most 1`),
			},
		},
	})
}

// TestAccRole_disappears confirms Terraform detects and recreates a Role
// that was deleted out-of-band (directly via the Kubernetes API, bypassing
// Terraform), rather than erroring or silently drifting.
func TestAccRole_disappears(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_role_v1.test"

	resource.ParallelTest(t, resource.TestCase{
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

// testAccRoleDeleteOutOfBand deletes the named Role directly via the
// Kubernetes clientset (bypassing Terraform), for TestAccRole_disappears.
func testAccRoleDeleteOutOfBand(name string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		conn, err := sdkv2providerMeta()().(kubernetes.KubeClientsets).MainClientset()
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
  }

  rule {
    api_groups     = [""]
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
  }

  rule {
    api_groups = ["batch"]
    resources  = ["jobs"]
    verbs      = ["get", "list", "watch", "create"]
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

func testAccRoleConfig_nameAndGenerateNameConflict(name string) string {
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
