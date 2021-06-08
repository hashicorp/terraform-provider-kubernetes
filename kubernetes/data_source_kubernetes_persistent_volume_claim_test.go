package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccKubernetesDataSourcePVC_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{ // The first apply creates the resource. The second apply reads the resource using the data source.
				Config: testAccKubernetesDataSourcePVCConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.access_modes.0", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.storage", "5Gi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.name", name),
				),
			},
			{
				Config: testAccKubernetesDataSourcePVCConfig_basic(name) +
					testAccKubernetesDataSourcePVCConfig_read(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubernetes_persistent_volume_claim.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("data.kubernetes_persistent_volume_claim.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("data.kubernetes_persistent_volume_claim.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("data.kubernetes_persistent_volume_claim.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("data.kubernetes_persistent_volume_claim.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_persistent_volume_claim.test", "spec.0.access_modes.0", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("data.kubernetes_persistent_volume_claim.test", "spec.0.resources.#", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.%", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.storage", "5Gi"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourcePVCConfig_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume_claim" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    name = "%s"
  }

  spec {
    access_modes = ["ReadWriteOnce"]

    resources {
      requests = {
        storage = "5Gi"
      }
    }

    selector {
      match_expressions {
        key      = "environment"
        operator = "In"
        values   = ["non-exists-12345"]
      }
    }
  }

  wait_until_bound = false
}
`, name)
}

func testAccKubernetesDataSourcePVCConfig_read() string {
	return fmt.Sprintf(`data "kubernetes_persistent_volume_claim" "test" {
  metadata {
    name = "${kubernetes_persistent_volume_claim.test.metadata.0.name}"
  }
}
`)
}
