package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	api "k8s.io/api/core/v1"
)

func TestAccKubernetesDataSourceEndpoints_basic(t *testing.T) {
	var conf api.Endpoints
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_endpoints.test",
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesEndpointDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesEndpointsConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesEndpointExists("kubernetes_endpoints.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_endpoints.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoints.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoints.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.0.address.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.0.address.0.ip", "10.0.0.4"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.0.port.0.name", "httptransport"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.0.port.0.port", "80"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.0.port.0.protocol", "TCP"),
				),
			},
			{
				Config: testAccKubernetesEndpointsConfig_basic(name) + testAccKubernetesDataSourceEndpointsConfig_read(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesEndpointExists("kubernetes_endpoints.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_endpoints.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoints.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoints.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.0.address.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.0.address.0.ip", "10.0.0.4"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.0.port.0.name", "httptransport"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.0.port.0.port", "80"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.0.port.0.protocol", "TCP"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceEndpointsConfig_read() string {
	return fmt.Sprintf(`data "kubernetes_endpoints" "test" {
  metadata {
    name = "${kubernetes_endpoints.test.metadata.0.name}"
  }
}
`)
}
