package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccKubernetesDataSourceService_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceServiceConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubernetes_service.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("data.kubernetes_service.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("data.kubernetes_service.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("data.kubernetes_service.test", "metadata.0.self_link"),
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
				),
			},
		},
	})
}

func testAccKubernetesDataSourceServiceConfig_basic(name string) string {
	return testAccKubernetesServiceConfig_basic(name) + `
data "kubernetes_service" "test" {
	metadata {
		name = "${kubernetes_service.test.metadata.0.name}"
	}
}
`
}
