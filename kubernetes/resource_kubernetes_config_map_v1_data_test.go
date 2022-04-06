package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccKubernetesConfigMapV1Data_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	namespace := "default"
	resourceName := "kubernetes_config_map_v1_data.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createConfigMap(name, namespace)
		},
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			return destroyConfigMap(name, namespace)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfigMapV1Data_empty(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "data.%", "0"),
				),
			},
			{
				Config: testAccKubernetesConfigMapV1Data_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "data.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "data.test1", "one"),
					resource.TestCheckResourceAttr(resourceName, "data.test2", "two"),
				),
			},
			{
				Config: testAccKubernetesConfigMapV1Data_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "data.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "data.test1", "one"),
					resource.TestCheckResourceAttr(resourceName, "data.test3", "three"),
				),
			},
			{
				Config: testAccKubernetesConfigMapV1Data_empty(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "data.%", "0"),
				),
			},
		},
	})
}

func testAccKubernetesConfigMapV1Data_empty(name string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1_data" "test" {
    metadata {
      name = %q
    }
    data = {}
  }
`, name)
}

func testAccKubernetesConfigMapV1Data_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1_data" "test" {
    metadata {
      name = %q
    }
    data = {
      "test1" = "one"
      "test2" = "two"
    }
  }
`, name)
}

func testAccKubernetesConfigMapV1Data_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1_data" "test" {
    metadata {
      name = %q
    }
    data = {
      "test1" = "one"
      "test3" = "three"
    }
  }
`, name)
}
