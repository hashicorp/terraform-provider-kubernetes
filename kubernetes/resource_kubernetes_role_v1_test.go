// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccKubernetesRoleV1_basic(t *testing.T) {
	var conf rbacv1.Role
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_role_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesRoleV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesRoleV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRoleV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "rule.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.api_groups.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.api_groups.0", "core"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resources.0", "pods"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.0", "get"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.1", "list"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.2", "watch"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resource_names.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resource_names.0", "foo"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.api_groups.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.api_groups.0", "apps"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.resources.0", "deployments"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.verbs.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.verbs.0", "get"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.verbs.1", "list"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.resource_names.#", "0"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesRoleV1Config_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRoleV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "rule.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.api_groups.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.api_groups.0", "batch"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resources.0", "jobs"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.0", "watch"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resource_names.#", "0"),
				),
			},
		},
	})
}

func TestAccKubernetesRoleV1_identity(t *testing.T) {
	resourceName := "kubernetes_role_v1.test"
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesRoleV1Destroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesRoleV1Config_basic(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(
						resourceName, map[string]knownvalue.Check{
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

func TestAccKubernetesRoleV1_generatedName(t *testing.T) {
	var conf rbacv1.Role
	prefix := "tf-acc-test-gen:"
	resourceName := "kubernetes_role_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesRoleV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesRoleV1Config_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRoleV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr(resourceName, "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
				),
			},
		},
	})
}

func TestAccKubernetesRoleV1_Bug(t *testing.T) {
	var conf rbacv1.Role
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_role_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesRoleV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesRoleV1ConfigBug_step_0(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRoleV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "rule.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resources.0", "pods"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.0", "get"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.resources.0", "deployments"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.verbs.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.verbs.0", "list"),
					resource.TestCheckResourceAttr(resourceName, "rule.2.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.2.resources.0", "cronjobs"),
					resource.TestCheckResourceAttr(resourceName, "rule.2.verbs.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.2.verbs.0", "list"),
				),
			},
			{
				Config: testAccKubernetesRoleV1ConfigBug_step_1(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRoleV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "rule.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resources.0", "deployments"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.0", "get"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.1", "list"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.api_groups.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.resources.0", "jobs"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.verbs.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.verbs.0", "get"),
				),
			},
			{
				Config: testAccKubernetesRoleV1ConfigBug_step_2(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRoleV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "rule.#", "4"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resources.0", "pods"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.0", "list"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.resources.0", "deployments"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.verbs.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.verbs.0", "list"),
					resource.TestCheckResourceAttr(resourceName, "rule.2.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.2.resources.0", "cronjobs"),
					resource.TestCheckResourceAttr(resourceName, "rule.2.verbs.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.2.verbs.0", "list"),
					resource.TestCheckResourceAttr(resourceName, "rule.3.api_groups.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.3.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.3.resources.0", "jobs"),
					resource.TestCheckResourceAttr(resourceName, "rule.3.verbs.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.3.verbs.0", "get"),
				),
			},
		},
	})
}

func testAccKubernetesRoleV1Config_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_role_v1" "test" {
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

  rule {
    api_groups     = ["core"]
    resources      = ["pods"]
    verbs          = ["get", "list", "watch"]
    resource_names = ["foo"]
  }

  rule {
    api_groups = ["apps"]
    resources  = ["deployments"]
    verbs      = ["get", "list"]
  }
}
`, name)
}

func testAccKubernetesRoleV1Config_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_role_v1" "test" {
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

  rule {
    api_groups = ["batch"]
    resources  = ["jobs"]
    verbs      = ["watch"]
  }
}
`, name)
}

func testAccKubernetesRoleV1Config_generatedName(name string) string {
	return fmt.Sprintf(`resource "kubernetes_role_v1" "test" {
  metadata {
    generate_name = "%s"
  }

  rule {
    api_groups = ["batch"]
    resources  = ["jobs"]
    verbs      = ["watch"]
  }
}
`, name)
}

func testAccCheckKubernetesRoleV1Exists(n string, obj *rbacv1.Role) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
		if err != nil {
			return err
		}
		ctx := context.TODO()

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		out, err := conn.RbacV1().Roles(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccCheckKubernetesRoleV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_role_v1" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.RbacV1().Roles(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Service still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccKubernetesRoleV1ConfigBug_step_0(name string) string {
	return fmt.Sprintf(`resource "kubernetes_role_v1" "test" {
  metadata {
    name      = "%s"
    namespace = "default"
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

func testAccKubernetesRoleV1ConfigBug_step_1(name string) string {
	return fmt.Sprintf(`resource "kubernetes_role_v1" "test" {
  metadata {
    name      = "%s"
    namespace = "default"
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

func testAccKubernetesRoleV1ConfigBug_step_2(name string) string {
	return fmt.Sprintf(`resource "kubernetes_role_v1" "test" {
  metadata {
    name      = "%s"
    namespace = "default"
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
