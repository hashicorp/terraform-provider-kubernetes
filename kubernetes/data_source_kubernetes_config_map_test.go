package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

// TestAccKubernetesDataSourceConfigMap_basic tests that the data source is able to read
// plaintext data, binary data, annotation, label, and name of the config map resource.
func TestAccKubernetesDataSourceConfigMap_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceConfigMapConfig_basic(name),
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

data "kubernetes_config_map" "test" {
  metadata {
    name = "${kubernetes_config_map.test.metadata.0.name}"
  }
}`, name)
}
