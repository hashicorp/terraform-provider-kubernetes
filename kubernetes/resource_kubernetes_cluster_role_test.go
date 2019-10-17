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

func TestAccKubernetesClusterRole_basic(t *testing.T) {
	var conf api.ClusterRole
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_cluster_role.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesClusterRoleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleExists("kubernetes_cluster_role.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_cluster_role.test", "rule.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role.test", "rule.0.resources.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role.test", "rule.0.resources.0", "pods"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role.test", "rule.0.resources.1", "pods/log"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role.test", "rule.0.verbs.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role.test", "rule.0.verbs.0", "get"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role.test", "rule.0.verbs.1", "list"),
				),
			},
			{
				Config: testAccKubernetesClusterRoleConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleExists("kubernetes_cluster_role.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_cluster_role.test", "rule.#", "3"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role.test", "rule.0.verbs.#", "3"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role.test", "rule.0.verbs.2", "watch"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role.test", "rule.1.api_groups.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role.test", "rule.1.resources.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role.test", "rule.1.resources.0", "deployments"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role.test", "rule.1.verbs.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role.test", "rule.1.verbs.0", "get"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role.test", "rule.1.verbs.1", "list"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role.test", "rule.2.non_resource_urls.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role.test", "rule.2.non_resource_urls.0", "/metrics"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role.test", "rule.2.verbs.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_cluster_role.test", "rule.2.verbs.0", "get"),
				),
			},
		},
	})
}
func TestAccKubernetesClusterRole_importBasic(t *testing.T) {
	resourceName := "kubernetes_cluster_role.test"
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesClusterRoleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleConfig_basic(name),
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
func testAccCheckKubernetesClusterRoleDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*KubeClientsets).MainClientset
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_cluster_role" {
			continue
		}
		resp, err := conn.RbacV1().ClusterRoles().Get(rs.Primary.ID, meta_v1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Cluster Role still exists: %s", rs.Primary.ID)
			}
		}
	}
	return nil
}
func testAccCheckKubernetesClusterRoleExists(n string, obj *api.ClusterRole) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		conn := testAccProvider.Meta().(*KubeClientsets).MainClientset
		out, err := conn.RbacV1().ClusterRoles().Get(rs.Primary.ID, meta_v1.GetOptions{})
		if err != nil {
			return err
		}
		*obj = *out
		return nil
	}
}
func testAccKubernetesClusterRoleConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_cluster_role" "test" {
  metadata {
    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    name = "%s"
  }
  
  rule {
    api_groups = [""]
    resources  = ["pods", "pods/log"]
    verbs      = ["get", "list"]
  }
}
`, name)
}
func testAccKubernetesClusterRoleConfig_modified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_cluster_role" "test" {
  metadata {
    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
    }

    name = "%s"
  }

  rule {
    api_groups = [""]
    resources  = ["pods", "pods/log"]
    verbs      = ["get", "list", "watch"]
  }

  rule {
    api_groups = [""]
    resources  = ["deployments"]
    verbs      = ["get", "list"]
  }

  rule {
    non_resource_urls = ["/metrics"]
    verbs = ["get"]
  }
}
`, name)
}
