package kubernetes

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	api "k8s.io/api/core/v1"
)

func TestAccKubernetesKubeSystemNamespace_basic(t *testing.T) {
	var conf api.Namespace

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesKubeSystemNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesKubeSystemNamespaceConfigBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKubernetesNamespaceExists("kubernetes_namespace.kube-system", &conf),
					resource.TestCheckResourceAttr("kubernetes_namespace.kube-system", "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_namespace.kube-system", "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_namespace.kube-system", "metadata.0.name", "kube-system"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.kube-system", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.kube-system", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.kube-system", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.kube-system", "metadata.0.uid"),
				),
			},
			{
				Config: testAccKubernetesKubeSystemNamespaceConfigAnnotations,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceExists("kubernetes_namespace.kube-system", &conf),
					resource.TestCheckResourceAttr("kubernetes_namespace.kube-system", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_namespace.kube-system", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_namespace.kube-system", "metadata.0.annotations.TestAnnotationTwo", "two"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "TestAnnotationTwo": "two"}),
					resource.TestCheckResourceAttr("kubernetes_namespace.kube-system", "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_namespace.kube-system", "metadata.0.name", "kube-system"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.kube-system", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.kube-system", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.kube-system", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.kube-system", "metadata.0.uid"),
				),
			},
		},
	})
}

func testAccCheckKubernetesKubeSystemNamespaceDestroy(s *terraform.State) error {
	// We expect the kube-system namespace to still exist
	return nil
}

const testAccKubernetesKubeSystemNamespaceConfigBasic = `
provider "kubernetes" {
  config_context_auth_info = "ops"
  config_context_cluster   = "mycluster"
}

resource "kubernetes_kube_system_namespace" "kube-system" {
  metadata {
    name = "kube-system"
  }
}
`

const testAccKubernetesKubeSystemNamespaceConfigAnnotations = `
provider "kubernetes" {
  config_context_auth_info = "ops"
  config_context_cluster   = "mycluster"
}

resource "kubernetes_kube_system_namespace" "kube-system" {
  metadata {
    name = "kube-system"
  }

  annotations {
    TestAnnotationOne = "one"
    TestAnnotationTwo = "two"
  }
}
`
