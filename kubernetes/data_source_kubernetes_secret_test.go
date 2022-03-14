package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccKubernetesDataSourceSecret_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_secret.test"
	datasourceName := "data.kubernetes_secret.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceSecretConfig_basic(name),
			},
			{
				Config: testAccKubernetesDataSourceSecretConfig_basic(name) +
					testAccKubernetesDataSourceSecretConfig_read(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(datasourceName, "metadata.0.name", resourceName, "metadata.0.name"),
					resource.TestCheckResourceAttrPair(datasourceName, "metadata.0.generation", resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrPair(datasourceName, "metadata.0.resource_version", resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrPair(datasourceName, "metadata.0.uid", resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttrPair(datasourceName, "metadata.0.annotations.%", resourceName, "metadata.0.annotations.%"),
					resource.TestCheckResourceAttrPair(datasourceName, "metadata.0.annotations.TestAnnotationOne", resourceName, "metadata.0.annotations.TestAnnotationOne"),
					resource.TestCheckResourceAttrPair(datasourceName, "metadata.0.annotations.TestAnnotationTwo", resourceName, "metadata.0.annotations.TestAnnotationTwo"),
					resource.TestCheckResourceAttrPair(datasourceName, "metadata.0.labels.TestLabelOne", resourceName, "metadata.0.labels.TestLabelOne"),
					resource.TestCheckResourceAttrPair(datasourceName, "metadata.0.labels.TestLabelTwo", resourceName, "metadata.0.labels.TestLabelTwo"),
					resource.TestCheckResourceAttrPair(datasourceName, "metadata.0.labels.TestLabelThree", resourceName, "metadata.0.labels.TestLabelThree"),
					resource.TestCheckResourceAttrPair(datasourceName, "data.%", resourceName, "data.%"),
					resource.TestCheckResourceAttrPair(datasourceName, "data.one", resourceName, "data.one"),
					resource.TestCheckResourceAttrPair(datasourceName, "data.two", resourceName, "data.two"),
					resource.TestCheckResourceAttrPair(datasourceName, "type", resourceName, "type"),
					resource.TestCheckResourceAttrPair(datasourceName, "immutable", resourceName, "immutable"),
					resource.TestCheckResourceAttrPair(datasourceName, "binary_data.raw", resourceName, "binary_data.raw"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceSecretConfig_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_secret" "test" {
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

  data = {
    one = "first"
    two = "second"
  }

  binary_data = {
    raw = "${base64encode("Raw data should come back as is in the pod")}"
  }
}
`, name)
}

func testAccKubernetesDataSourceSecretConfig_read() string {
	return fmt.Sprintf(`data "kubernetes_secret" "test" {
  metadata {
    name = kubernetes_secret.test.metadata.0.name
  }

  binary_data = {
    raw = ""
  }
}
`)
}
