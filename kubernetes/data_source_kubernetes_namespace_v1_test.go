// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

// NOTE: Despite the _v1 filename, the tests here cover the unversioned
// `kubernetes_namespace` data source, which SDKv2 still serves as a deprecated alias.
// The `kubernetes_namespace_v1` tests moved to internal/framework/provider/corev1 when
// that data source was migrated to the Plugin Framework. The filename is unchanged so
// the SDKv2 data source and its tests stay paired, since both are still backed by
// dataSourceKubernetesNamespaceV1 in data_source_kubernetes_namespace_v1.go.

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccKubernetesDataSourceNamespace_basic(t *testing.T) {
	dataSourceName := "data.kubernetes_namespace.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceNamespace_basic(),
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

func TestAccKubernetesDataSourceNamespace_not_found(t *testing.T) {
	dataSourceName := "data.kubernetes_namespace.test"
	name := fmt.Sprintf("ceci-n.est-pas-une-namespace-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceNamespace_nonexistent(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(dataSourceName, "spec.#", "0"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceNamespace_basic() string {
	return `data "kubernetes_namespace" "test" {
  metadata {
    name = "kube-system"
  }
}
`
}

func testAccKubernetesDataSourceNamespace_nonexistent(name string) string {
	return fmt.Sprintf(`data "kubernetes_namespace" "test" {
  metadata {
    name = "%s"
  }
}
`, name)
}
