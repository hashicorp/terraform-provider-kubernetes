// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccKubernetesDataSourcePodV1_basic(t *testing.T) {
	resourceName := "kubernetes_pod_v1.test"
	dataSourceName := "data.kubernetes_pod_v1.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourcePodV1_basic(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.image", imageName),
				),
			},
			{
				Config: testAccKubernetesDataSourcePodV1_basic(name, imageName) +
					testAccKubernetesDataSourcePodV1_read(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.container.0.image", imageName),
				),
			},
		},
	})
}

func TestAccKubernetesDataSourcePodV1_not_found(t *testing.T) {
	dataSourceName := "data.kubernetes_pod_v1.test"
	name := fmt.Sprintf("ceci-n.est-pas-une-pod-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourcePodV1_nonexistent(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(dataSourceName, "spec.#", "0"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourcePodV1_basic(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    container {
      image = "%s"
      name  = "containername"
    }
  }
}
`, name, imageName)
}

func testAccKubernetesDataSourcePodV1_read() string {
	return `data "kubernetes_pod_v1" "test" {
  metadata {
    name = "${kubernetes_pod_v1.test.metadata.0.name}"
  }
}
`
}

func testAccKubernetesDataSourcePodV1_nonexistent(name string) string {
	return fmt.Sprintf(`data "kubernetes_pod_v1" "test" {
  metadata {
    name = "%s"
  }
}
`, name)
}
