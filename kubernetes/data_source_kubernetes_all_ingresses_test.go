// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccKubernetesDataSourceAllIngresses_basic(t *testing.T) {
	dataSourceName := "data.kubernetes_all_ingresses.test"
	rxPosNum := regexp.MustCompile("^[0-9]*$")
	ingName := regexp.MustCompile(`^[a-zA-Z][-\w]*$`)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceAllIngressesConfig_basic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(dataSourceName, "ingresses.#", rxPosNum),
					resource.TestCheckResourceAttrSet(dataSourceName, "ingresses.0.name"),
					resource.TestMatchResourceAttr(dataSourceName, "ingresses.0.name", ingName),
					resource.TestCheckResourceAttrSet(dataSourceName, "id"),
				),
			},
		},
	})
}

func TestAccKubernetesDataSourceAllIngresses_withSelectors(t *testing.T) {
	dataSourceName := "data.kubernetes_all_ingresses.test"
	rxPosNum := regexp.MustCompile("^[0-9]*$")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceAllIngressesConfig_withSelectors(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(dataSourceName, "ingresses.#", rxPosNum),
					resource.TestCheckResourceAttrSet(dataSourceName, "id"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceAllIngressesConfig_basic() string {
	return `data "kubernetes_all_ingresses" "test" {}`
}

func testAccKubernetesDataSourceAllIngressesConfig_withSelectors() string {
	return `
data "kubernetes_all_ingresses" "test" {
  label_selector = "app=web"
  field_selector = "metadata.namespace=default"
}
`
}
