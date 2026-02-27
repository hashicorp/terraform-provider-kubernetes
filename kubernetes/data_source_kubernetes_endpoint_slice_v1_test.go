// Copyright (c) IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

//nolint:all
package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccKubernetesDataSourceEndpointSliceV1_basic(t *testing.T) {
	resourceName := "kubernetes_endpoint_slice_v1.test"
	dataSourceName := "data.kubernetes_endpoint_slice_v1.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesEndpointSliceV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "endpoint.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "endpoint.0.condition.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "endpoint.0.addresses.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "endpoint.0.addresses.0", "129.144.50.56"),
					resource.TestCheckResourceAttr(resourceName, "port.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "port.0.port", "90"),
					resource.TestCheckResourceAttr(resourceName, "port.0.name", "first"),
					resource.TestCheckResourceAttr(resourceName, "port.0.app_protocol", "test"),
				),
			},
			{
				Config: testAccKubernetesEndpointSliceV1Config_modified(name) + testAccKubernetesDataSourceEndpointSliceV1_read(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(dataSourceName, "endpoint.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "endpoint.0.condition.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "endpoint.0.condition.0.ready", "true"),
					resource.TestCheckResourceAttr(dataSourceName, "endpoint.0.condition.0.serving", "false"),
					resource.TestCheckResourceAttr(dataSourceName, "endpoint.0.condition.0.terminating", "false"),
					resource.TestCheckResourceAttr(dataSourceName, "endpoint.0.hostname", "test"),
					resource.TestCheckResourceAttr(dataSourceName, "endpoint.0.node_name", "test"),
					resource.TestCheckResourceAttr(dataSourceName, "endpoint.0.addresses.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "endpoint.0.target_ref.0.name", "test"),
					resource.TestCheckResourceAttr(dataSourceName, "endpoint.0.addresses.0", "2001:db8:3333:4444:5555:6666:7777:8888"),
					resource.TestCheckResourceAttr(dataSourceName, "endpoint.0.addresses.1", "2002:db8:3333:4444:5555:6666:7777:8888"),
					resource.TestCheckResourceAttr(dataSourceName, "port.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "port.0.port", "90"),
					resource.TestCheckResourceAttr(dataSourceName, "port.0.name", "first"),
					resource.TestCheckResourceAttr(dataSourceName, "port.0.app_protocol", "test"),
					resource.TestCheckResourceAttr(dataSourceName, "port.1.port", "900"),
					resource.TestCheckResourceAttr(dataSourceName, "port.1.name", "second"),
					resource.TestCheckResourceAttr(dataSourceName, "port.1.app_protocol", "test"),
					resource.TestCheckResourceAttr(dataSourceName, "address_type", "IPv6"),
				),
			},
		},
	})
}

func TestAccKubernetesDataSourceEndpointSliceV1_not_found(t *testing.T) {
	dataSourceName := "data.kubernetes_endpoint_slice_v1.test"
	name := fmt.Sprintf("ceci-n.est-pas-une-endpoint-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceEndpointSliceV1_nonexistent(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(dataSourceName, "endpoint.#", "0"),
					resource.TestCheckResourceAttr(dataSourceName, "port.#", "0"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceEndpointSliceV1_read() string {
	return `data "kubernetes_endpoint_slice_v1" "test" {
  metadata {
    name = "${kubernetes_endpoint_slice_v1.test.metadata.0.name}"
  }
}
`
}

func testAccKubernetesDataSourceEndpointSliceV1_nonexistent(name string) string {
	return fmt.Sprintf(`data "kubernetes_endpoint_slice_v1" "test" {
  metadata {
    name = "%s"
  }
}
`, name)
}
