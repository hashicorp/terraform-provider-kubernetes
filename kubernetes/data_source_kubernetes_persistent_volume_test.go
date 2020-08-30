package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccKubernetesDataSourcePV_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourcePVConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubernetes_persistent_volume.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("data.kubernetes_persistent_volume.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("data.kubernetes_persistent_volume.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("data.kubernetes_persistent_volume.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("data.kubernetes_persistent_volume.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("data.kubernetes_persistent_volume.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_persistent_volume.test", "spec.0.access_modes.1245328686", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("data.kubernetes_persistent_volume.test", "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_persistent_volume.test", "spec.0.capacity.storage", "5Gi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.1245328686", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.storage", "5Gi"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourcePVConfig_basic(name string) string {
	return fmt.Sprintf(`

	resource "kubernetes_persistent_volume" "test" {
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
	  
	  data "kubernetes_persistent_volume" "test" {
		  metadata {
			  name = "${kubernetes_persistent_volume.test.metadata.0.name}"
		  }
	  }
`, name)
}
