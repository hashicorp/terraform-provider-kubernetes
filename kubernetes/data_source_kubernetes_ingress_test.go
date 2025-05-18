// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccKubernetesDataSourceIngress_basic(t *testing.T) {
	resourceName := "kubernetes_ingress.test"
	dataSourceName := "data.kubernetes_ingress.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionGreaterThanOrEqual(t, "1.22.0")
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{ // Create the ingress resource in the first apply. Then check it in the second apply.
				Config: testAccKubernetesDataSourceIngress_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend.0.service_name", "app1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend.0.service_port", "443"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.host", "server.domain.com"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.0.path.0.path", "/.*"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.0.path.0.backend.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.0.path.0.backend.0.service_name", "app2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.0.path.0.backend.0.service_port", "80"),
				),
			},
			{
				Config: testAccKubernetesDataSourceIngress_basic(name) +
					testAccKubernetesDataSourceIngress_read(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.backend.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.backend.0.service_name", "app1"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.backend.0.service_port", "443"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.rule.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.rule.0.host", "server.domain.com"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.rule.0.http.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.rule.0.http.0.path.0.path", "/.*"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.rule.0.http.0.path.0.backend.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.rule.0.http.0.path.0.backend.0.service_name", "app2"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.rule.0.http.0.path.0.backend.0.service_port", "80"),
				),
			},
		},
	})
}

func TestAccKubernetesDataSourceIngress_not_found(t *testing.T) {
	dataSourceName := "data.kubernetes_ingress.test"
	name := fmt.Sprintf("ceci-n.est-pas-une-ingress-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionGreaterThanOrEqual(t, "1.22.0")
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceIngress_nonexistent(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(dataSourceName, "spec.#", "0"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceIngress_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_ingress" "test" {
  metadata {
    name = "%s"
  }
  spec {
    backend {
      service_name = "app1"
      service_port = 443
    }
    rule {
      host = "server.domain.com"
      http {
        path {
          backend {
            service_name = "app2"
            service_port = 80
          }
          path = "/.*"
        }
      }
    }
  }
}
`, name)
}

func testAccKubernetesDataSourceIngress_read() string {
	return `data "kubernetes_ingress" "test" {
  metadata {
    name      = "${kubernetes_ingress.test.metadata.0.name}"
    namespace = "${kubernetes_ingress.test.metadata.0.namespace}"
  }
}
`
}

func testAccKubernetesDataSourceIngress_nonexistent(name string) string {
	return fmt.Sprintf(`data "kubernetes_ingress" "test" {
  metadata {
    name = "%s"
  }
}
`, name)
}
