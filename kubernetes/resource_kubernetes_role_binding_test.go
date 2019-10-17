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

func TestAccKubernetesRoleBinding_basic(t *testing.T) {
	var conf api.RoleBinding
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_role_binding.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesRoleBindingDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesRoleBindingConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRoleBindingExists("kubernetes_role_binding.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_role_binding.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_role_binding.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_role_binding.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_role_binding.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "role_ref.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "role_ref.0.kind", "Role"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "role_ref.0.name", "admin"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "subject.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "subject.0.name", "notauser"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "subject.0.kind", "User"),
				),
			},
			{
				Config: testAccKubernetesRoleBindingConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRoleBindingExists("kubernetes_role_binding.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_role_binding.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_role_binding.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_role_binding.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_role_binding.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "role_ref.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "role_ref.0.kind", "Role"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "role_ref.0.name", "admin"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "subject.#", "3"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "subject.0.name", "notauser"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "subject.0.kind", "User"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "subject.1.namespace", "kube-system"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "subject.1.name", "default"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "subject.1.kind", "ServiceAccount"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "subject.1.api_group", ""),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "subject.2.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "subject.2.name", "system:masters"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "subject.2.kind", "Group"),
				),
			},
		},
	})
}

func TestAccKubernetesRoleBinding_importBasic(t *testing.T) {
	resourceName := "kubernetes_role_binding.test"
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesRoleBindingDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesRoleBindingConfig_basic(name),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccKubernetesRoleBinding_sa_subject(t *testing.T) {
	var conf api.RoleBinding
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_role_binding.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesRoleBindingDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesRoleBindingConfig_sa_subject(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRoleBindingExists("kubernetes_role_binding.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_role_binding.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_role_binding.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_role_binding.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_role_binding.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "role_ref.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "role_ref.0.kind", "Role"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "role_ref.0.name", "admin"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "subject.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "subject.0.api_group", ""),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "subject.0.name", "someservice"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "subject.0.kind", "ServiceAccount"),
				),
			},
		},
	})
}

func TestAccKubernetesRoleBinding_group_subject(t *testing.T) {
	var conf api.RoleBinding
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_role_binding.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesRoleBindingDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesRoleBindingConfig_group_subject(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRoleBindingExists("kubernetes_role_binding.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_role_binding.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_role_binding.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_role_binding.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_role_binding.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "role_ref.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "role_ref.0.kind", "Role"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "role_ref.0.name", "admin"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "subject.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "subject.0.name", "somegroup"),
					resource.TestCheckResourceAttr("kubernetes_role_binding.test", "subject.0.kind", "Group"),
				),
			},
		},
	})
}

func testAccCheckKubernetesRoleBindingDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*KubeClientsets).MainClientset

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_role_binding" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.RbacV1().RoleBindings(namespace).Get(name, meta_v1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("RoleBinding still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesRoleBindingExists(n string, obj *api.RoleBinding) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*KubeClientsets).MainClientset

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.RbacV1().RoleBindings(namespace).Get(name, meta_v1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *resp
		return nil
	}
}

func testAccKubernetesRoleBindingConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding" "test" {
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

func testAccKubernetesRoleBindingConfig_modified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding" "test" {
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

func testAccKubernetesRoleBindingConfig_sa_subject(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding" "test" {
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

func testAccKubernetesRoleBindingConfig_group_subject(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding" "test" {
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
