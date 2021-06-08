package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccKubernetesDataSourcePod_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(8))
	imageName := "hashicorp/http-echo:latest"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourcePodConfig_basic(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_pod.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.image", imageName),
				),
			},
			{
				Config: testAccKubernetesDataSourcePodConfig_basic(name, imageName) +
					testAccKubernetesDataSourcePodConfig_read(),
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
`, name, imageName)
}

func testAccKubernetesDataSourcePodConfig_read() string {
	return fmt.Sprintf(`data "kubernetes_pod" "test" {
  metadata {
    name = "${kubernetes_pod.test.metadata.0.name}"
  }
}
`)
}
