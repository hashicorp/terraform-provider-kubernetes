package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccKubernetesDataSourcePod_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := "hashicorp/http-echo:latest"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourcePodConfig_basic(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubernetes_pod.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("data.kubernetes_pod.test", "spec.0.container.0.image", imageName),
				),
			},
		},
	})
}

func testAccKubernetesDataSourcePodConfig_basic(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod" "test" {
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
data "kubernetes_pod" "test" {
  metadata {
    name = "${kubernetes_pod.test.metadata.0.name}"
  }
}
`, name, imageName)
}
