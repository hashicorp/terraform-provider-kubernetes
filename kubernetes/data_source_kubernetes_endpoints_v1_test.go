// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	corev1 "k8s.io/api/core/v1"
)

func TestAccKubernetesDataSourceEndpointsV1_basic(t *testing.T) {
	resourceName := "kubernetes_endpoints_v1.test"
	dataSourceName := "data.kubernetes_endpoints_v1.test"
	var conf corev1.Endpoints
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesEndpointV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesEndpointsV1_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesEndpointV1Exists(resourceName, &conf),
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
				Config: testAccKubernetesEndpointsV1_basic(name) + testAccKubernetesDataSourceEndpointsV1_read(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesEndpointV1Exists(dataSourceName, &conf),
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

func TestAccKubernetesDataSourceEndpointsV1_not_found(t *testing.T) {
	dataSourceName := "data.kubernetes_endpoints_v1.test"
	name := fmt.Sprintf("ceci-n.est-pas-une-endpoint-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesEndpointV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceEndpointsV1_nonexistent(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(dataSourceName, "subset.#", "0"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceEndpointsV1_read() string {
	return `data "kubernetes_endpoints_v1" "test" {
  metadata {
    name = "${kubernetes_endpoints_v1.test.metadata.0.name}"
  }
}
`
}

func testAccKubernetesDataSourceEndpointsV1_nonexistent(name string) string {
	return fmt.Sprintf(`data "kubernetes_endpoints_v1" "test" {
  metadata {
    name = "%s"
  }
}
`, name)
}
