package kubernetes

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	api "k8s.io/api/rbac/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesRole_basic(t *testing.T) {
	var conf api.Role
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_role.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesRoleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesRoleConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRoleExists("kubernetes_role.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_role.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_role.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_role.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_role.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_role.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.0.api_groups.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.0.api_groups.1804436815", "core"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.0.resources.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.0.resources.3245178296", "pods"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.0.verbs.#", "3"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.0.verbs.4248514160", "get"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.0.verbs.1154021400", "list"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.0.verbs.1342917158", "watch"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.0.resource_names.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.0.resource_names.2356372769", "foo"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.1.api_groups.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.1.api_groups.270302810", "apps"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.1.resources.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.1.resources.926696405", "deployments"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.1.verbs.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.1.verbs.4248514160", "get"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.1.verbs.1154021400", "list"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.1.resource_names.#", "0"),
				),
			},
			{
				Config: testAccKubernetesRoleConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRoleExists("kubernetes_role.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_role.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_role.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_role.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_role.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_role.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.0.api_groups.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.0.api_groups.4161491668", "batch"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.0.resources.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.0.resources.2828234181", "jobs"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.0.verbs.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.0.verbs.1342917158", "watch"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.0.resource_names.#", "0"),
				),
			},
		},
	})
}

func TestAccKubernetesRole_importBasic(t *testing.T) {
	resourceName := "kubernetes_role.test"
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesRoleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesRoleConfig_basic(name),
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

func TestAccKubernetesRole_generatedName(t *testing.T) {
	var conf api.Role
	prefix := "tf-acc-test-gen:"

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_role.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesRoleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesRoleConfig_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRoleExists("kubernetes_role.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_role.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_role.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_role.test", "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr("kubernetes_role.test", "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet("kubernetes_role.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_role.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_role.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_role.test", "metadata.0.uid"),
				),
			},
		},
	})
}

func TestAccKubernetesRoleBug(t *testing.T) {
	var conf api.Role
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_role.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesRoleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesRoleConfigBug_step_0(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRoleExists("kubernetes_role.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.#", "3"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.0.resources.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.0.resources.3245178296", "pods"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.0.verbs.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.0.verbs.4248514160", "get"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.1.resources.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.1.resources.926696405", "deployments"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.1.verbs.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.1.verbs.1154021400", "list"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.2.resources.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.2.resources.382107465", "cronjobs"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.2.verbs.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.2.verbs.1154021400", "list"),
				),
			},
			{
				Config: testAccKubernetesRoleConfigBug_step_1(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRoleExists("kubernetes_role.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.0.resources.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.0.resources.926696405", "deployments"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.0.verbs.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.0.verbs.4248514160", "get"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.0.verbs.1154021400", "list"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.1.api_groups.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.1.resources.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.1.resources.2828234181", "jobs"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.1.verbs.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.1.verbs.4248514160", "get"),
				),
			},
			{
				Config: testAccKubernetesRoleConfigBug_step_2(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRoleExists("kubernetes_role.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.#", "4"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.0.resources.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.0.resources.3245178296", "pods"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.0.verbs.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.0.verbs.1154021400", "list"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.1.resources.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.1.resources.926696405", "deployments"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.1.verbs.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.1.verbs.1154021400", "list"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.2.resources.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.2.resources.382107465", "cronjobs"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.2.verbs.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.2.verbs.1154021400", "list"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.3.api_groups.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.3.resources.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.3.resources.2828234181", "jobs"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.3.verbs.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_role.test", "rule.3.verbs.4248514160", "get"),
				),
			},
		},
	})
}

func testAccKubernetesRoleConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role" "test" {
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

func testAccKubernetesRoleConfig_modified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role" "test" {
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
func testAccKubernetesRoleConfig_generatedName(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role" "test" {
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

func testAccCheckKubernetesRoleExists(n string, obj *api.Role) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
		if err != nil {
			return err
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		out, err := conn.RbacV1().Roles(namespace).Get(name, meta_v1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccCheckKubernetesRoleDestroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_role" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.RbacV1().Roles(namespace).Get(name, meta_v1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Service still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccKubernetesRoleConfigBug_step_0(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role" "test" {
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

func testAccKubernetesRoleConfigBug_step_1(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role" "test" {
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

func testAccKubernetesRoleConfigBug_step_2(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role" "test" {
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
