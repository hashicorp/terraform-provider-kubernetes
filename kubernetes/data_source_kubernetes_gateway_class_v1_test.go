// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGatewayClassV1DataSource_basic(t *testing.T) {
	resourceName := "kubernetes_gateway_class_v1.test"
	dataSourceName := "data.kubernetes_gateway_class_v1.test"
	name := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayClassV1DataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.controller_name", "example.com/gateway-controller"),
				),
			},
			{
				Config: testAccGatewayClassV1DataSourceConfig(name) + testAccGatewayClassV1DataSourceReadConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.controller_name", "example.com/gateway-controller"),
				),
			},
		},
	})
}

func TestAccGatewayClassV1DataSource_description(t *testing.T) {
	resourceName := "kubernetes_gateway_class_v1.test"
	dataSourceName := "data.kubernetes_gateway_class_v1.test"
	name := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayClassV1DataSourceConfigDescription(name, "Test description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.description", "Test description"),
				),
			},
			{
				Config: testAccGatewayClassV1DataSourceConfigDescription(name, "Test description") + testAccGatewayClassV1DataSourceReadConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.controller_name", "example.com/gateway-controller"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.description", "Test description"),
				),
			},
		},
	})
}

func TestAccGatewayClassV1DataSource_parametersRef(t *testing.T) {
	resourceName := "kubernetes_gateway_class_v1.test"
	dataSourceName := "data.kubernetes_gateway_class_v1.test"
	name := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayClassV1DataSourceConfigParametersRef(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parameters_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parameters_ref.0.group", "example.com"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parameters_ref.0.kind", "GatewayParameters"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parameters_ref.0.name", "test-params"),
				),
			},
			{
				Config: testAccGatewayClassV1DataSourceConfigParametersRef(name) + testAccGatewayClassV1DataSourceReadConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.controller_name", "example.com/gateway-controller"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.parameters_ref.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.parameters_ref.0.group", "example.com"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.parameters_ref.0.kind", "GatewayParameters"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.parameters_ref.0.name", "test-params"),
				),
			},
		},
	})
}

func TestAccGatewayClassV1DataSource_notFound(t *testing.T) {
	dataSourceName := "data.kubernetes_gateway_class_v1.test"
	name := acctest.RandomWithPrefix("tf-acc-nonexistent")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`data "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = %[1]q
  }
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
				),
			},
		},
	})
}

func testAccGatewayClassV1DataSourceConfig(name string) string {
	return fmt.Sprintf(`resource "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    controller_name = "example.com/gateway-controller"
  }
}
`, name)
}

func testAccGatewayClassV1DataSourceConfigDescription(name, description string) string {
	return fmt.Sprintf(`resource "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    controller_name = "example.com/gateway-controller"
    description     = %[2]q
  }
}
`, name, description)
}

func testAccGatewayClassV1DataSourceConfigParametersRef(name string) string {
	return fmt.Sprintf(`resource "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    controller_name = "example.com/gateway-controller"
    parameters_ref {
      group = "example.com"
      kind  = "GatewayParameters"
      name  = "test-params"
    }
  }
}
`, name)
}

func testAccGatewayClassV1DataSourceReadConfig() string {
	return `data "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = kubernetes_gateway_class_v1.test.metadata.0.name
  }
}
`
}
