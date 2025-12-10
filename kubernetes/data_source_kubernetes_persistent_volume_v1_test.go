// Copyright IBM Corp. 2017, 2025
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccKubernetesDataSourcePersistentVolumeV1_basic(t *testing.T) {
	resourceName := "kubernetes_persistent_volume_v1.test"
	dataSourceName := "data.kubernetes_persistent_volume_v1.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfRunningInGke(t)
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourcePersistentVolumeV1_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.access_modes.0", "ReadWriteOnce"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.capacity.storage", "5Gi"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.0", "ReadWriteOnce"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.storage", "5Gi"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourcePersistentVolumeV1_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume_v1" "test" {
  metadata {
    name = "%s"
  }

  spec {

    capacity = {
      storage = "5Gi"
    }

    access_modes = ["ReadWriteOnce"]
    persistent_volume_source {
      vsphere_volume {
        volume_path = "/absolute/path"
      }
    }

  }
}

data "kubernetes_persistent_volume_v1" "test" {
  metadata {
    name = "${kubernetes_persistent_volume_v1.test.metadata.0.name}"
  }
}
`, name)
}
