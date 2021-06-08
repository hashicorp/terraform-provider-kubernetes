package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAccKubernetesDataSourceConfigMap_basic tests that the data source is able to read
// plaintext data, binary data, annotation, label, and name of the config map resource.
func TestAccKubernetesDataSourceConfigMap_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{ // First, create the resource. Data sources are evaluated before resources, and therefore need to be created in a second apply.
				Config: testAccKubernetesDataSourceConfigMapConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "data.one", "first"),
					resource.TestCheckResourceAttr("kubernetes_config_map.test", "binary_data.raw", "UmF3IGRhdGEgc2hvdWxkIGNvbWUgYmFjayBhcyBpcyBpbiB0aGUgcG9k"),
				),
			},
			{ // Use the data source to read the existing resource.
				Config: testAccKubernetesDataSourceConfigMapConfig_basic(name) +
					testAccKubernetesDataSourceConfigMapConfig_read(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubernetes_config_map.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("data.kubernetes_config_map.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("data.kubernetes_config_map.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("data.kubernetes_config_map.test", "data.one", "first"),
					resource.TestCheckResourceAttr("data.kubernetes_config_map.test", "binary_data.raw", "UmF3IGRhdGEgc2hvdWxkIGNvbWUgYmFjayBhcyBpcyBpbiB0aGUgcG9k"),
				),
			},
		},
	})
}

// testAccKubernetesDataSourceConfigMapConfig_basic provides the terraform config
// used to test basic functionality of the config_map data source.
func testAccKubernetesDataSourceConfigMapConfig_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
    }

    labels = {
      TestLabelOne   = "one"
    }

    name = "%s"
  }

  data = {
    one = "first"
  }

  binary_data = {
    raw = "${base64encode("Raw data should come back as is in the pod")}"
  }
}
`, name)
}

func testAccKubernetesDataSourceConfigMapConfig_read() string {
	return fmt.Sprintf(`data "kubernetes_config_map" "test" {
  metadata {
    name = "${kubernetes_config_map.test.metadata.0.name}"
  }
}
`)
}
