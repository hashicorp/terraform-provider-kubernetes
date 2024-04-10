// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccKubernetesDataSourceIngressV1_basic(t *testing.T) {
	resourceName := "kubernetes_ingress_v1.test"
	dataSourceName := "data.kubernetes_ingress_v1.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.22.0")
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{ // Create the ingress resource in the first apply. Then check it in the second apply.
				Config: testAccKubernetesDataSourceIngressV1_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.default_backend.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.default_backend.0.service.0.name", "app1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.default_backend.0.service.0.port.0.number", "443"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.host", "server.domain.com"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.0.path.0.path", "/.*"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.0.path.0.path_type", "Prefix"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.0.path.0.backend.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.0.path.0.backend.0.service.0.name", "app2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.0.path.0.backend.0.service.0.port.0.number", "80"),
				),
			},
			{
				Config: testAccKubernetesDataSourceIngressV1_basic(name) +
					testAccKubernetesDataSourceIngressV1_read(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.default_backend.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.default_backend.0.service.0.name", "app1"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.default_backend.0.service.0.port.0.number", "443"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.rule.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.rule.0.host", "server.domain.com"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.rule.0.http.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.rule.0.http.0.path.0.path", "/.*"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.rule.0.http.0.path.0.path_type", "Prefix"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.rule.0.http.0.path.0.backend.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.rule.0.http.0.path.0.backend.0.service.0.name", "app2"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.rule.0.http.0.path.0.backend.0.service.0.port.0.number", "80"),
				),
			},
		},
	})
}

func TestAccKubernetesDataSourceIngressV1_not_found(t *testing.T) {
	dataSourceName := "data.kubernetes_ingress_v1.test"
	name := fmt.Sprintf("ceci-n.est-pas-une-ingress-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.22.0")
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceIngressV1_nonexistent(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(dataSourceName, "spec.#", "0"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceIngressV1_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_ingress_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    default_backend {
      service {
        name = "app1"
        port {
          number = 443
        }
      }
    }
    rule {
      host = "server.domain.com"
      http {
        path {
          backend {
            service {
              name = "app2"
              port {
                number = 80
              }
            }
          }
          path      = "/.*"
          path_type = "Prefix"
        }
      }
    }
  }
}
`, name)
}

func testAccKubernetesDataSourceIngressV1_read() string {
	return `data "kubernetes_ingress_v1" "test" {
  metadata {
    name      = "${kubernetes_ingress_v1.test.metadata.0.name}"
    namespace = "${kubernetes_ingress_v1.test.metadata.0.namespace}"
  }
}
`
}

func testAccKubernetesDataSourceIngressV1_nonexistent(name string) string {
	return fmt.Sprintf(`data "kubernetes_ingress_v1" "test" {
  metadata {
    name      = "%s"
  }
}
`, name)
}
