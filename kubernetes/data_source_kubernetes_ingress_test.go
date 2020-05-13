package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccKubernetesDataSourceIngress_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceIngressConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubernetes_ingress.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("data.kubernetes_ingress.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("data.kubernetes_ingress.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("data.kubernetes_ingress.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("data.kubernetes_ingress.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("data.kubernetes_ingress.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_ingress.test", "spec.0.backend.#", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_ingress.test", "spec.0.backend.0.service_name", "app1"),
					resource.TestCheckResourceAttr("data.kubernetes_ingress.test", "spec.0.backend.0.service_port", "443"),
					resource.TestCheckResourceAttr("data.kubernetes_ingress.test", "spec.0.rule.#", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_ingress.test", "spec.0.rule.0.host", "server.domain.com"),
					resource.TestCheckResourceAttr("data.kubernetes_ingress.test", "spec.0.rule.0.http.#", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_ingress.test", "spec.0.rule.0.http.0.path.0.path", "/.*"),
					resource.TestCheckResourceAttr("data.kubernetes_ingress.test", "spec.0.rule.0.http.0.path.0.backend.#", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_ingress.test", "spec.0.rule.0.http.0.path.0.backend.0.service_name", "app2"),
					resource.TestCheckResourceAttr("data.kubernetes_ingress.test", "spec.0.rule.0.http.0.path.0.backend.0.service_port", "80"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceIngressConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_ingress" "test" {
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

data "kubernetes_ingress" "test" {
	metadata {
		name = "${kubernetes_ingress.test.metadata.0.name}"
	}
}`, name)
}
