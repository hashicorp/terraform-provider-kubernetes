package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccKubernetesDataSourceService_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceServiceConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_service.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.#", "1"),
					resource.TestCheckResourceAttrSet("kubernetes_service.test", "spec.0.cluster_ip"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.0.name", ""),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.0.node_port", "0"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.0.port", "8080"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.0.protocol", "TCP"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.port.0.target_port", "80"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.session_affinity", "None"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.type", "ClusterIP"),
					resource.TestCheckResourceAttr("kubernetes_service.test", "spec.0.health_check_node_port", "0"),
				),
			},
			{
				Config: testAccKubernetesDataSourceServiceConfig_basic(name) +
					testAccKubernetesDataSourceServiceConfig_read(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubernetes_service.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("data.kubernetes_service.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("data.kubernetes_service.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("data.kubernetes_service.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("data.kubernetes_service.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_service.test", "spec.0.port.#", "1"),
					resource.TestCheckResourceAttrSet("data.kubernetes_service.test", "spec.0.cluster_ip"),
					resource.TestCheckResourceAttr("data.kubernetes_service.test", "spec.0.port.0.name", ""),
					resource.TestCheckResourceAttr("data.kubernetes_service.test", "spec.0.port.0.node_port", "0"),
					resource.TestCheckResourceAttr("data.kubernetes_service.test", "spec.0.port.0.port", "8080"),
					resource.TestCheckResourceAttr("data.kubernetes_service.test", "spec.0.port.0.protocol", "TCP"),
					resource.TestCheckResourceAttr("data.kubernetes_service.test", "spec.0.port.0.target_port", "80"),
					resource.TestCheckResourceAttr("data.kubernetes_service.test", "spec.0.session_affinity", "None"),
					resource.TestCheckResourceAttr("data.kubernetes_service.test", "spec.0.type", "ClusterIP"),
					resource.TestCheckResourceAttr("data.kubernetes_service.test", "spec.0.health_check_node_port", "0"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceServiceConfig_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service" "test" {
  metadata {
    name = "%s"
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
  spec {
    port {
      port        = 8080
      target_port = 80
    }
  }
}
`, name)
}

func testAccKubernetesDataSourceServiceConfig_read() string {
	return fmt.Sprintf(`data "kubernetes_service" "test" {
  metadata {
    name = "${kubernetes_service.test.metadata.0.name}"
  }
}
`)
}
