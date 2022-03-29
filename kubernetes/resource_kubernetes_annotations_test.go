package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccKubernetesAnnotations_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	namespace := "default"
	resourceName := "kubernetes_annotations.test"

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
				Config: testAccKubernetesAnnotations_empty(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "ConfigMap"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "annotations.%", "0"),
				),
			},
			{
				Config: testAccKubernetesAnnotations_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "ConfigMap"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "annotations.test1", "one"),
					resource.TestCheckResourceAttr(resourceName, "annotations.test2", "two"),
				),
			},
			{
				Config: testAccKubernetesAnnotations_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "ConfigMap"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "annotations.test1", "one"),
					resource.TestCheckResourceAttr(resourceName, "annotations.test3", "three"),
				),
			},
			{
				Config: testAccKubernetesAnnotations_empty(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "ConfigMap"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "annotations.%", "0"),
				),
			},
		},
	})
}

func testAccKubernetesAnnotations_empty(name string) string {
	return fmt.Sprintf(`resource "kubernetes_annotations" "test" {
    api_version = "v1"
    kind        = "ConfigMap"
    metadata {
      name = %q
    }
    annotations = {}
  }
`, name)
}

func testAccKubernetesAnnotations_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_annotations" "test" {
    api_version = "v1"
    kind        = "ConfigMap"
    metadata {
      name = %q
    }
    annotations = {
      "test1" = "one"
      "test2" = "two"
    }
  }
`, name)
}

func testAccKubernetesAnnotations_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_annotations" "test" {
    api_version = "v1"
    kind        = "ConfigMap"
    metadata {
      name = %q
    }
    annotations = {
      "test1" = "one"
      "test3" = "three"
    }
  }
`, name)
}
