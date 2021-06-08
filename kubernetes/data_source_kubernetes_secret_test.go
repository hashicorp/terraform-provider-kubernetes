package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccKubernetesDataSourceSecret_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceSecretConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_secret.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_secret.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_secret.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "data.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "data.one", "first"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "data.two", "second"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "type", "Opaque"),
					resource.TestCheckResourceAttr("kubernetes_secret.test", "binary_data.raw", "UmF3IGRhdGEgc2hvdWxkIGNvbWUgYmFjayBhcyBpcyBpbiB0aGUgcG9k"),
				),
			},
			{
				Config: testAccKubernetesDataSourceSecretConfig_basic(name) +
					testAccKubernetesDataSourceSecretConfig_read(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubernetes_secret.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("data.kubernetes_secret.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("data.kubernetes_secret.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("data.kubernetes_secret.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("data.kubernetes_secret.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("data.kubernetes_secret.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("data.kubernetes_secret.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr("data.kubernetes_secret.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("data.kubernetes_secret.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("data.kubernetes_secret.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("data.kubernetes_secret.test", "data.%", "2"),
					resource.TestCheckResourceAttr("data.kubernetes_secret.test", "data.one", "first"),
					resource.TestCheckResourceAttr("data.kubernetes_secret.test", "data.two", "second"),
					resource.TestCheckResourceAttr("data.kubernetes_secret.test", "type", "Opaque"),
					resource.TestCheckResourceAttr("data.kubernetes_secret.test", "binary_data.raw", "UmF3IGRhdGEgc2hvdWxkIGNvbWUgYmFjayBhcyBpcyBpbiB0aGUgcG9k"),
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
    name = "${kubernetes_secret.test.metadata.0.name}"
  }

  binary_data = {
    raw = ""
  }
}
`)
}
