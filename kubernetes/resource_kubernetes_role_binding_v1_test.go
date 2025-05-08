// Copyright (c) HashiCorp, Inc.
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
)

func TestAccKubernetesRoleBindingV1_basic(t *testing.T) {
	var conf rbacv1.RoleBinding
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_role_binding_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesRoleBindingConfigV1_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRoleBindingV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.kind", "Role"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.name", "admin"),
					resource.TestCheckResourceAttr(resourceName, "subject.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.name", "notauser"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.kind", "User"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesRoleBindingConfigV1_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRoleBindingV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.kind", "Role"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.name", "admin"),
					resource.TestCheckResourceAttr(resourceName, "subject.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.name", "notauser"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.kind", "User"),
					resource.TestCheckResourceAttr(resourceName, "subject.1.namespace", "kube-system"),
					resource.TestCheckResourceAttr(resourceName, "subject.1.name", "default"),
					resource.TestCheckResourceAttr(resourceName, "subject.1.kind", "ServiceAccount"),
					resource.TestCheckResourceAttr(resourceName, "subject.1.api_group", ""),
					resource.TestCheckResourceAttr(resourceName, "subject.2.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "subject.2.name", "system:masters"),
					resource.TestCheckResourceAttr(resourceName, "subject.2.kind", "Group"),
				),
			},
			{
				Config: testAccKubernetesRoleBindingConfigV1_modified_role_ref(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRoleBindingV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.kind", "ClusterRole"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.name", "cluster-admin"),
					resource.TestCheckResourceAttr(resourceName, "subject.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.name", "notauser"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.kind", "User"),
					resource.TestCheckResourceAttr(resourceName, "subject.1.namespace", "kube-system"),
					resource.TestCheckResourceAttr(resourceName, "subject.1.name", "default"),
					resource.TestCheckResourceAttr(resourceName, "subject.1.kind", "ServiceAccount"),
					resource.TestCheckResourceAttr(resourceName, "subject.1.api_group", ""),
					resource.TestCheckResourceAttr(resourceName, "subject.2.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "subject.2.name", "system:masters"),
					resource.TestCheckResourceAttr(resourceName, "subject.2.kind", "Group"),
				),
			},
			{
				Config: testAccKubernetesRoleBindingConfigV1_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRoleBindingV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.kind", "Role"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.name", "admin"),
					resource.TestCheckResourceAttr(resourceName, "subject.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.name", "notauser"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.kind", "User"),
					resource.TestCheckResourceAttr(resourceName, "subject.1.namespace", "kube-system"),
					resource.TestCheckResourceAttr(resourceName, "subject.1.name", "default"),
					resource.TestCheckResourceAttr(resourceName, "subject.1.kind", "ServiceAccount"),
					resource.TestCheckResourceAttr(resourceName, "subject.1.api_group", ""),
					resource.TestCheckResourceAttr(resourceName, "subject.2.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "subject.2.name", "system:masters"),
					resource.TestCheckResourceAttr(resourceName, "subject.2.kind", "Group"),
				),
			},
		},
	})
}

func TestAccKubernetesRoleBindingV1_generatedName(t *testing.T) {
	var conf rbacv1.RoleBinding
	prefix := "tf-acc-test-gen:"
	resourceName := "kubernetes_role_binding_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesRoleBindingConfigV1_generateName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRoleBindingV1Exists(resourceName, &conf),
					resource.TestMatchResourceAttr(resourceName, "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.kind", "Role"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.name", "admin"),
					resource.TestCheckResourceAttr(resourceName, "subject.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.name", "notauser"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.kind", "User"),
				),
			},
		},
	})
}

func TestAccKubernetesRoleBindingV1_sa_subject(t *testing.T) {
	var conf rbacv1.RoleBinding
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_role_binding_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesRoleBindingConfigV1_sa_subject(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRoleBindingV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.kind", "Role"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.name", "admin"),
					resource.TestCheckResourceAttr(resourceName, "subject.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.api_group", ""),
					resource.TestCheckResourceAttr(resourceName, "subject.0.name", "someservice"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.kind", "ServiceAccount"),
				),
			},
		},
	})
}

func TestAccKubernetesRoleBindingV1_group_subject(t *testing.T) {
	var conf rbacv1.RoleBinding
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_role_binding_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesRoleBindingConfigV1_group_subject(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRoleBindingV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.kind", "Role"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.name", "admin"),
					resource.TestCheckResourceAttr(resourceName, "subject.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.name", "somegroup"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.kind", "Group"),
				),
			},
		},
	})
}

func TestAccKubernetesRoleBindingV1_Bug(t *testing.T) {
	var conf rbacv1.RoleBinding
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_role_binding_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesRoleBindingConfigV1Bug_step_0(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRoleBindingV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.kind", "Role"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.name", "admin"),
					resource.TestCheckResourceAttr(resourceName, "subject.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.name", "notauser1"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.kind", "User"),
					resource.TestCheckResourceAttr(resourceName, "subject.1.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "subject.1.name", "notauser2"),
					resource.TestCheckResourceAttr(resourceName, "subject.1.kind", "User"),
					resource.TestCheckResourceAttr(resourceName, "subject.2.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "subject.2.name", "notauser3"),
					resource.TestCheckResourceAttr(resourceName, "subject.2.kind", "User"),
				),
			},
			{
				Config: testAccKubernetesRoleBindingConfigV1Bug_step_1(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRoleBindingV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.kind", "Role"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.name", "admin"),
					resource.TestCheckResourceAttr(resourceName, "subject.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.name", "notauser2"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.kind", "User"),
					resource.TestCheckResourceAttr(resourceName, "subject.1.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "subject.1.name", "notauser4"),
					resource.TestCheckResourceAttr(resourceName, "subject.1.kind", "User"),
				),
			},
			{
				Config: testAccKubernetesRoleBindingConfigV1Bug_step_2(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRoleBindingV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.kind", "Role"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.name", "admin"),
					resource.TestCheckResourceAttr(resourceName, "subject.#", "4"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.name", "notauser0"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.kind", "User"),
					resource.TestCheckResourceAttr(resourceName, "subject.1.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "subject.1.name", "notauser1"),
					resource.TestCheckResourceAttr(resourceName, "subject.1.kind", "User"),
					resource.TestCheckResourceAttr(resourceName, "subject.2.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "subject.2.name", "notauser2"),
					resource.TestCheckResourceAttr(resourceName, "subject.2.kind", "User"),
					resource.TestCheckResourceAttr(resourceName, "subject.3.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "subject.3.name", "notauser3"),
					resource.TestCheckResourceAttr(resourceName, "subject.3.kind", "User"),
				),
			},
		},
	})
}

func testAccCheckKubernetesRoleBindingV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_role_binding_v1" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.RbacV1().RoleBindings(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("RoleBinding still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesRoleBindingV1Exists(n string, obj *rbacv1.RoleBinding) resource.TestCheckFunc {
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

		resp, err := conn.RbacV1().RoleBindings(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *resp
		return nil
	}
}

func testAccKubernetesRoleBindingConfigV1_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name = "%s"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }

  subject {
    kind      = "User"
    name      = "notauser"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, name)
}

func testAccKubernetesRoleBindingConfigV1_generateName(prefixName string) string {
	return fmt.Sprintf(`resource "kubernetes_role_binding_v1" "test" {
  metadata {
    generate_name = "%s"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }

  subject {
    kind      = "User"
    name      = "notauser"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, prefixName)
}

func testAccKubernetesRoleBindingConfigV1_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name = "%s"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }

  subject {
    kind      = "User"
    name      = "notauser"
    api_group = "rbac.authorization.k8s.io"
  }

  subject {
    kind      = "ServiceAccount"
    name      = "default"
    api_group = ""
    namespace = "kube-system"
  }

  subject {
    kind      = "Group"
    name      = "system:masters"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, name)
}

func testAccKubernetesRoleBindingConfigV1_modified_role_ref(name string) string {
	return fmt.Sprintf(`resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name = "%s"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = "cluster-admin"
  }

  subject {
    kind      = "User"
    name      = "notauser"
    api_group = "rbac.authorization.k8s.io"
  }

  subject {
    kind      = "ServiceAccount"
    name      = "default"
    api_group = ""
    namespace = "kube-system"
  }

  subject {
    kind      = "Group"
    name      = "system:masters"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, name)
}

func testAccKubernetesRoleBindingConfigV1_sa_subject(name string) string {
	return fmt.Sprintf(`resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name = "%s"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }

  subject {
    kind      = "ServiceAccount"
    name      = "someservice"
    api_group = ""
  }
}
`, name)
}

func testAccKubernetesRoleBindingConfigV1_group_subject(name string) string {
	return fmt.Sprintf(`resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name = "%s"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }

  subject {
    kind      = "Group"
    name      = "somegroup"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, name)
}

func testAccKubernetesRoleBindingConfigV1Bug_step_0(name string) string {
	return fmt.Sprintf(`resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name      = "%s"
    namespace = "default"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }

  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "User"
    name      = "notauser1"
  }
  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "User"
    name      = "notauser2"
  }
  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "User"
    name      = "notauser3"
  }
}
`, name)
}

func testAccKubernetesRoleBindingConfigV1Bug_step_1(name string) string {
	return fmt.Sprintf(`resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name      = "%s"
    namespace = "default"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }

  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "User"
    name      = "notauser2"
  }
  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "User"
    name      = "notauser4"
  }
}
`, name)
}

func testAccKubernetesRoleBindingConfigV1Bug_step_2(name string) string {
	return fmt.Sprintf(`resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name      = "%s"
    namespace = "default"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }

  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "User"
    name      = "notauser0"
  }
  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "User"
    name      = "notauser1"
  }
  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "User"
    name      = "notauser2"
  }
  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "User"
    name      = "notauser3"
  }
}
`, name)
}
