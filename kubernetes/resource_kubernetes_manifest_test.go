package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// NOTE this is an example of how we can test the muxed manifest provider
// using the same framework as the main provider
func TestAccKubernetesManifest_ConfigMap(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	namespace := "default"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_manifest.test",
		ExternalProviders: testAccExternalProviders,
		CheckDestroy:      testAccCheckKubernetesConfigMapDestroy,
		Steps: []resource.TestStep{
			{
				Config: requiredProviders() + testAccKubernetesManifest_ConfigMap(name, namespace),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_manifest.test", "manifest.metadata.name", name),
					resource.TestCheckResourceAttr("kubernetes_manifest.test", "manifest.metadata.namespace", namespace),
					resource.TestCheckResourceAttr("kubernetes_manifest.test", "manifest.data.TEST", "123"),
					resource.TestCheckResourceAttr("kubernetes_manifest.test", "object.metadata.name", name),
					resource.TestCheckResourceAttr("kubernetes_manifest.test", "object.metadata.namespace", namespace),
					resource.TestCheckResourceAttr("kubernetes_manifest.test", "object.data.TEST", "123"),
				),
			},
			// FIXME uncomment when import is implemented
			// {
			// 	ResourceName:            "kubernetes_manifest.test",
			// 	ImportState:             true,
			// 	ImportStateVerify:       true,
			// },
			{
				Config: requiredProviders() + testAccKubernetesManifest_ConfigMap_modified(name, namespace),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_manifest.test", "manifest.metadata.name", name),
					resource.TestCheckResourceAttr("kubernetes_manifest.test", "manifest.metadata.namespace", namespace),
					resource.TestCheckResourceAttr("kubernetes_manifest.test", "manifest.data.TEST", "456"),
					resource.TestCheckResourceAttr("kubernetes_manifest.test", "object.metadata.name", name),
					resource.TestCheckResourceAttr("kubernetes_manifest.test", "object.metadata.namespace", namespace),
					resource.TestCheckResourceAttr("kubernetes_manifest.test", "object.data.TEST", "456"),
				),
			},
		},
	})
}

func testAccKubernetesManifest_ConfigMap(name, namespace string) string {
	return fmt.Sprintf(`resource "kubernetes_manifest" "test" {
  provider = kubernetes-local
  manifest = {
	apiVersion = "v1"
	kind = "ConfigMap"
	metadata = {
	  name = %q
	  namespace = %q
	}
	data = {
	  TEST = "123"
	}
  }
}
`, name, namespace)
}

func testAccKubernetesManifest_ConfigMap_modified(name, namespace string) string {
	return fmt.Sprintf(`resource "kubernetes_manifest" "test" {
  provider = kubernetes-local
  manifest = {
	apiVersion = "v1"
	kind = "ConfigMap"
	metadata = {
	  name = %q
	  namespace = %q
	}
	data = {
	  TEST = "456"
	}
  }
}
`, name, namespace)
}
