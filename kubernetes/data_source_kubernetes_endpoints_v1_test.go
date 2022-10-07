package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	corev1 "k8s.io/api/core/v1"
)

func TestAccKubernetesDataSourceEndpointsV1_basic(t *testing.T) {
	var conf corev1.Endpoints
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_endpoints.test"
	dataSourceName := "data.kubernetes_endpoints_v1.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesEndpointDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesEndpointsConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesEndpointExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "subset.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.address.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.address.0.ip", "10.0.0.4"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.port.0.name", "httptransport"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.port.0.port", "80"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.port.0.protocol", "TCP"),
				),
			},
			{
				Config: testAccKubernetesEndpointsConfig_basic(name) + testAccKubernetesDataSourceEndpointsV1Config_read(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesEndpointExists(dataSourceName, &conf),
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(dataSourceName, "subset.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "subset.0.address.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "subset.0.address.0.ip", "10.0.0.4"),
					resource.TestCheckResourceAttr(dataSourceName, "subset.0.port.0.name", "httptransport"),
					resource.TestCheckResourceAttr(dataSourceName, "subset.0.port.0.port", "80"),
					resource.TestCheckResourceAttr(dataSourceName, "subset.0.port.0.protocol", "TCP"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceEndpointsV1Config_read() string {
	return fmt.Sprintf(`data "kubernetes_endpoints_v1" "test" {
  metadata {
    name = "${kubernetes_endpoints.test.metadata.0.name}"
  }
}
`)
}
