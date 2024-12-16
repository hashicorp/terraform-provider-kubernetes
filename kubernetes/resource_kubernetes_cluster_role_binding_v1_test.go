// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesClusterRoleBindingV1_basic(t *testing.T) {
	var conf rbacv1.ClusterRoleBinding
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_cluster_role_binding_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleBindingV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.kind", "ClusterRole"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.name", "cluster-admin"),
					resource.TestCheckResourceAttr(resourceName, "subject.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.name", "notauser"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.kind", "User"),
				),
			},
			{
				Config: testAccKubernetesClusterRoleBindingV1Config_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleBindingV1Exists(resourceName, &conf),
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
				Config: testAccKubernetesClusterRoleBindingV1Config_modified_role_ref(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleBindingV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.kind", "ClusterRole"),
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
				Config: testAccKubernetesClusterRoleBindingV1Config_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleBindingV1Exists(resourceName, &conf),
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
		},
	})
}

func TestAccKubernetesClusterRoleBindingV1_generatedName(t *testing.T) {
	var conf rbacv1.ClusterRoleBinding
	prefix := "tf-acc-test-gen:"
	resourceName := "kubernetes_cluster_role_binding_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingV1Config_generateName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleBindingV1Exists(resourceName, &conf),
					resource.TestMatchResourceAttr(resourceName, "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.kind", "ClusterRole"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.name", "cluster-admin"),
					resource.TestCheckResourceAttr(resourceName, "subject.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.name", "notauser"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.kind", "User"),
				),
			},
		},
	})
}

func TestAccKubernetesClusterRoleBindingV1_serviceaccount_subject(t *testing.T) {
	var conf rbacv1.ClusterRoleBinding
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_cluster_role_binding_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingV1Config_serviceaccount_subject(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleBindingV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.kind", "ClusterRole"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.name", "cluster-admin"),
					resource.TestCheckResourceAttr(resourceName, "subject.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.api_group", ""),
					resource.TestCheckResourceAttr(resourceName, "subject.0.name", "someservice"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.kind", "ServiceAccount"),
				),
			},
		},
	})
}

func TestAccKubernetesClusterRoleBindingV1_group_subject(t *testing.T) {
	var conf rbacv1.ClusterRoleBinding
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_cluster_role_binding_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingV1Config_group_subject(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleBindingV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.kind", "ClusterRole"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.name", "cluster-admin"),
					resource.TestCheckResourceAttr(resourceName, "subject.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.name", "somegroup"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.kind", "Group"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
		},
	})
}

func TestAccKubernetesClusterRoleBindingV1_UpdatePatchOperationsOrderWithRemovals(t *testing.T) {
	var conf rbacv1.ClusterRoleBinding
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_cluster_role_binding_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingV1ConfigBug_step_0(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleBindingV1Exists(resourceName, &conf),
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
				Config: testAccKubernetesClusterRoleBindingV1ConfigBug_step_1(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleBindingV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.kind", "ClusterRole"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.name", "cluster-admin"),
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
				Config: testAccKubernetesClusterRoleBindingV1ConfigBug_step_2(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleBindingV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.kind", "ClusterRole"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.name", "cluster-admin"),
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

func TestAccKubernetesClusterRoleBindingV1_namespaceHandling(t *testing.T) {
	var conf rbacv1.ClusterRoleBinding
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_cluster_role_binding_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingV1Config_namespaceHandling(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleBindingV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.kind", "ClusterRole"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.name", "cluster-admin"),
					resource.TestCheckResourceAttr(resourceName, "subject.#", "3"),
					// Checking Group subject
					resource.TestCheckResourceAttr(resourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.kind", "Group"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.name", "testgroup"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.namespace", ""),
					// Checking User subject
					resource.TestCheckResourceAttr(resourceName, "subject.1.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "subject.1.kind", "User"),
					resource.TestCheckResourceAttr(resourceName, "subject.1.name", "testuser"),
					resource.TestCheckResourceAttr(resourceName, "subject.1.namespace", ""),
					// Checking ServiceAccount subject
					resource.TestCheckResourceAttr(resourceName, "subject.2.api_group", ""),
					resource.TestCheckResourceAttr(resourceName, "subject.2.kind", "ServiceAccount"),
					resource.TestCheckResourceAttr(resourceName, "subject.2.name", "default"),
					resource.TestCheckResourceAttr(resourceName, "subject.2.namespace", "default"),
				),
			},
		},
	})
}

func testAccKubernetesClusterRoleBindingV1Config_namespaceHandling(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = "%s"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = "cluster-admin"
  }

  # Group subject with namespace explicitly set to ""
  subject {
    kind      = "Group"
    name      = "testgroup"
    api_group = "rbac.authorization.k8s.io"
    namespace = ""
  }

  # User subject with namespace explicitly set to ""
  subject {
    kind      = "User"
    name      = "testuser"
    api_group = "rbac.authorization.k8s.io"
    namespace = ""
  }

  # ServiceAccount subject with no namespace specified
  subject {
    kind      = "ServiceAccount"
    name      = "default"
    api_group = ""
  }
}
`, name)
}

func testAccCheckKubernetesClusterRoleBindingV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_cluster_role_binding_v1" {
			continue
		}
		name := rs.Primary.ID
		resp, err := conn.RbacV1().ClusterRoleBindings().Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("ClusterRoleBinding still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesClusterRoleBindingV1Exists(n string, obj *rbacv1.ClusterRoleBinding) resource.TestCheckFunc {
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

		name := rs.Primary.ID
		resp, err := conn.RbacV1().ClusterRoleBindings().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *resp
		return nil
	}
}

func testAccKubernetesClusterRoleBindingV1Config_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding_v1" "test" {
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
}
`, name)
}

func testAccKubernetesClusterRoleBindingV1Config_generateName(namePrefix string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    generate_name = "%s"
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
}
`, namePrefix)
}

func testAccKubernetesClusterRoleBindingV1Config_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding_v1" "test" {
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

func testAccKubernetesClusterRoleBindingV1Config_modified_role_ref(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = "%s"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    # The kind field only accepts this value, anything else returns an error:
    # roleRef.kind: Unsupported value: "Role": supported values: "ClusterRole"
    kind = "ClusterRole"
    name = "admin"
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

func testAccKubernetesClusterRoleBindingV1Config_serviceaccount_subject(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = "%s"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = "cluster-admin"
  }

  subject {
    kind = "ServiceAccount"
    name = "someservice"
  }
}
`, name)
}

func testAccKubernetesClusterRoleBindingV1Config_group_subject(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = "%s"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = "cluster-admin"
  }

  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Group"
    name      = "somegroup"
  }
}
`, name)
}

func testAccKubernetesClusterRoleBindingV1ConfigBug_step_0(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = "%s"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = "cluster-admin"
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

func testAccKubernetesClusterRoleBindingV1ConfigBug_step_1(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = "%s"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = "cluster-admin"
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

func testAccKubernetesClusterRoleBindingV1ConfigBug_step_2(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = "%s"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = "cluster-admin"
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
