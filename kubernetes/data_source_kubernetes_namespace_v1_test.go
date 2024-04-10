// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccKubernetesDataSourceNamespaceV1_basic(t *testing.T) {
	dataSourceName := "data.kubernetes_namespace_v1.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceNamespaceV1_basic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", "kube-system"),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.finalizers.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.finalizers.0", "kubernetes"),
				),
			},
		},
	})
}

func TestAccKubernetesDataSourceNamespaceV1_not_found(t *testing.T) {
	dataSourceName := "data.kubernetes_namespace_v1.test"
	name := fmt.Sprintf("ceci-n.est-pas-une-namespace-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceNamespaceV1_nonexistent(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(dataSourceName, "spec.#", "0"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceNamespaceV1_basic() string {
	return `data "kubernetes_namespace_v1" "test" {
  metadata {
    name = "kube-system"
  }
}
`
}

func testAccKubernetesDataSourceNamespaceV1_nonexistent(name string) string {
	return fmt.Sprintf(`data "kubernetes_namespace_v1" "test" {
  metadata {
    name = "%s"
  }
}
`, name)
}