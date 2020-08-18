package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccKubernetesDataSourcePVC_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourcePVCConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubernetes_persistent_volume_claim.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("data.kubernetes_persistent_volume_claim.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("data.kubernetes_persistent_volume_claim.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("data.kubernetes_persistent_volume_claim.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("data.kubernetes_persistent_volume_claim.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("data.kubernetes_persistent_volume_claim.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_persistent_volume_claim.test", "spec.0.access_modes.1245328686", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("data.kubernetes_persistent_volume_claim.test", "spec.0.resources.#", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.%", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.storage", "5Gi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.access_modes.1245328686", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.storage", "5Gi"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourcePVCConfig_basic(name string) string {
	return fmt.Sprintf(`

	resource "kubernetes_persistent_volume_claim" "test" {
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
		lifecycle {
			ignore_changes = [metadata]
		}
	  }
	  
	  
	  
	  data "kubernetes_persistent_volume_claim" "test" {
		  metadata {
			  name = "${kubernetes_persistent_volume_claim.test.metadata.0.name}"
		  }
	  }
`, name)
}
