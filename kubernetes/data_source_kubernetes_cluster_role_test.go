package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccKubernetesDataSourceClusterRole_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceClusterRole_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubernetes_cluster_role.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("data.kubernetes_cluster_role.test", "rules.#", "4"),
					resource.TestCheckResourceAttr("data.kubernetes_cluster_role.test", "rules.0.resources.#", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_cluster_role.test", "rules.0.resources.0", "pods"),
					resource.TestCheckResourceAttr("data.kubernetes_cluster_role.test", "rules.0.verbs.#", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_cluster_role.test", "rules.0.verbs.0", "list"),
					resource.TestCheckResourceAttr("data.kubernetes_cluster_role.test", "rules.1.resources.#", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_cluster_role.test", "rules.1.resources.0", "deployments"),
					resource.TestCheckResourceAttr("data.kubernetes_cluster_role.test", "rules.1.verbs.#", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_cluster_role.test", "rules.1.verbs.0", "list"),
					resource.TestCheckResourceAttr("data.kubernetes_cluster_role.test", "rules.2.non_resource_urls.#", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_cluster_role.test", "rules.2.non_resource_urls.0", "/metrics"),
					resource.TestCheckResourceAttr("data.kubernetes_cluster_role.test", "rules.2.verbs.#", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_cluster_role.test", "rules.2.verbs.0", "get"),
					resource.TestCheckResourceAttr("data.kubernetes_cluster_role.test", "rules.3.api_groups.#", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_cluster_role.test", "rules.3.resources.#", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_cluster_role.test", "rules.3.resources.0", "jobs"),
					resource.TestCheckResourceAttr("data.kubernetes_cluster_role.test", "rules.3.verbs.#", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_cluster_role.test", "rules.3.verbs.0", "get"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceClusterRole_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_cluster_role" "test" {
	metadata {
		name = "%s"
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
		non_resource_urls = ["/metrics"]
		verbs             = ["get"]
	}
	
	rule {
		api_groups = [""]
		resources  = ["jobs"]
		verbs      = ["get"]
	}
}

data "kubernetes_cluster_role" "test" {
	metadata {
		name = "${kubernetes_cluster_role.test.metadata.0.name}"
	}
}`, name)
}
