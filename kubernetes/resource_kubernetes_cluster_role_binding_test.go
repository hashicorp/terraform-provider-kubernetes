package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	api "k8s.io/api/rbac/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesClusterRoleBinding(t *testing.T) {
	var conf api.ClusterRoleBinding
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_cluster_role_binding.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesClusterRoleBindingDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleBindingExists("kubernetes_cluster_role_binding.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.kind", "ClusterRole"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.name", "cluster-admin"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.name", "notauser"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.kind", "User"),
				),
			},
			{
				Config: testAccKubernetesClusterRoleBindingConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleBindingExists("kubernetes_cluster_role_binding.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.kind", "ClusterRole"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.name", "cluster-admin"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.#", "3"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.name", "notauser"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.kind", "User"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.1.namespace", "kube-system"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.1.name", "default"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.1.kind", "ServiceAccount"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.1.api_group", ""),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.2.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.2.name", "system:masters"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.2.kind", "Group"),
				),
			},
		},
	})
}

func TestAccKubernetesClusterRoleBinding_serviceaccount_subject(t *testing.T) {
	var conf api.ClusterRoleBinding
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_cluster_role_binding.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesClusterRoleBindingDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingConfig_serviceaccount_subject(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleBindingExists("kubernetes_cluster_role_binding.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.kind", "ClusterRole"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.name", "cluster-admin"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.api_group", ""),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.name", "someservice"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.kind", "ServiceAccount"),
				),
			},
		},
	})
}

func TestAccKubernetesClusterRoleBinding_group_subject(t *testing.T) {
	var conf api.ClusterRoleBinding
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_cluster_role_binding.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesClusterRoleBindingDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingConfig_group_subject(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleBindingExists("kubernetes_cluster_role_binding.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_cluster_role_binding.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.kind", "ClusterRole"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "role_ref.0.name", "cluster-admin"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.name", "somegroup"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role_binding.test", "subject.0.kind", "Group"),
				),
			},
		},
	})
}

func TestAccKubernetesClusterRoleBinding_importBasic(t *testing.T) {
	resourceName := "kubernetes_cluster_role_binding.test"
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesClusterRoleBindingDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingConfig_basic(name),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckKubernetesClusterRoleBindingDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*KubeClientsets).MainClientset

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_cluster_role_binding" {
			continue
		}
		name := rs.Primary.ID
		resp, err := conn.RbacV1().ClusterRoleBindings().Get(name, meta_v1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("ClusterRoleBinding still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesClusterRoleBindingExists(n string, obj *api.ClusterRoleBinding) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*KubeClientsets).MainClientset
		name := rs.Primary.ID
		resp, err := conn.RbacV1().ClusterRoleBindings().Get(name, meta_v1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *resp
		return nil
	}
}

func testAccKubernetesClusterRoleBindingConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_cluster_role_binding" "test" {
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

func testAccKubernetesClusterRoleBindingConfig_modified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_cluster_role_binding" "test" {
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

func testAccKubernetesClusterRoleBindingConfig_serviceaccount_subject(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_cluster_role_binding" "test" {
  metadata {
    name = "%s"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = "cluster-admin"
  }

  subject {
    kind      = "ServiceAccount"
    name      = "someservice"
  }
}
`, name)
}

func testAccKubernetesClusterRoleBindingConfig_group_subject(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_cluster_role_binding" "test" {
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
