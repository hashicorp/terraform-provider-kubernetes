package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccKubernetesDataSourceIngressV1_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{ // Create the ingress resource in the first apply. Then check it in the second apply.
				Config: testAccKubernetesDataSourceIngressV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_ingress_v1.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress_v1.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress_v1.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.default_backend.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.default_backend.0.service.0.name", "app1"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.default_backend.0.service.0.port.0.number", "443"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.rule.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.rule.0.host", "server.domain.com"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.rule.0.http.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.rule.0.http.0.path.0.path", "/.*"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.rule.0.http.0.path.0.path_type", "Prefix"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.rule.0.http.0.path.0.backend.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.rule.0.http.0.path.0.backend.0.service.0.name", "app2"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.rule.0.http.0.path.0.backend.0.service.0.port.0.number", "80"),
				),
			},
			{
				Config: testAccKubernetesDataSourceIngressV1Config_basic(name) +
					testAccKubernetesDataSourceIngressV1Config_read(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubernetes_ingress_v1.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("data.kubernetes_ingress_v1.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("data.kubernetes_ingress_v1.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("data.kubernetes_ingress_v1.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("data.kubernetes_ingress_v1.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_ingress_v1.test", "spec.0.default_backend.#", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_ingress_v1.test", "spec.0.default_backend.0.service.0.name", "app1"),
					resource.TestCheckResourceAttr("data.kubernetes_ingress_v1.test", "spec.0.default_backend.0.service.0.port.0.number", "443"),
					resource.TestCheckResourceAttr("data.kubernetes_ingress_v1.test", "spec.0.rule.#", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_ingress_v1.test", "spec.0.rule.0.host", "server.domain.com"),
					resource.TestCheckResourceAttr("data.kubernetes_ingress_v1.test", "spec.0.rule.0.http.#", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_ingress_v1.test", "spec.0.rule.0.http.0.path.0.path", "/.*"),
					resource.TestCheckResourceAttr("data.kubernetes_ingress_v1.test", "spec.0.rule.0.http.0.path.0.path_type", "Prefix"),
					resource.TestCheckResourceAttr("data.kubernetes_ingress_v1.test", "spec.0.rule.0.http.0.path.0.backend.#", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_ingress_v1.test", "spec.0.rule.0.http.0.path.0.backend.0.service.0.name", "app2"),
					resource.TestCheckResourceAttr("data.kubernetes_ingress_v1.test", "spec.0.rule.0.http.0.path.0.backend.0.service.0.port.0.number", "80"),
				),
			},
		},
	})
}

func TestAccKubernetesDataSourceIngressV1_regression(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotRunningInEks(t) },
		IDRefreshName:     "kubernetes_ingress_v1.test",
		ExternalProviders: testAccExternalProviders,
		CheckDestroy:      testAccCheckKubernetesIngressDestroy,
		Steps: []resource.TestStep{
			{ // Create resource and data source using schema v0.
				Config: requiredProviders() + testAccKubernetesDataSourceIngressV1Config_regression("kubernetes-released", name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("data.kubernetes_ingress_v1.test", "metadata.0.name", name),
				),
			},
			{ // Apply StateUpgrade to resource. This will cause data source to re-read the data.
				Config: requiredProviders() + testAccKubernetesDataSourceIngressV1Config_regression("kubernetes-local", name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("kubernetes_ingress_v1.test", "status.0.load_balancer.0.ingress.0.hostname"),
					resource.TestCheckNoResourceAttr("kubernetes_ingress_v1.test", "load_balancer_ingress.0.hostname"),
					resource.TestCheckNoResourceAttr("data.kubernetes_ingress_v1.test", "load_balancer_ingress.0.hostname"),
					resource.TestCheckResourceAttrSet("data.kubernetes_ingress_v1.test", "status.0.load_balancer.0.ingress.0.hostname"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceIngressV1Config_basic(name string) string {
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
          path = "/.*"
		  path_type = "Prefix"
        }
      }
    }
  }
}
`, name)
}

func testAccKubernetesDataSourceIngressV1Config_read() string {
	return fmt.Sprintf(`data "kubernetes_ingress_v1" "test" {
  metadata {
    name = "${kubernetes_ingress_v1.test.metadata.0.name}"
    namespace = "${kubernetes_ingress_v1.test.metadata.0.namespace}"
  }
}
`)
}

// Note: this test uses a unique namespace in order to avoid name collisions in AWS.
// This ensures a unique TargetGroup for each test run.
func testAccKubernetesDataSourceIngressV1Config_regression(provider, name string) string {
	return fmt.Sprintf(`data "kubernetes_ingress" "test" {
  provider = %s
  metadata {
    name      = kubernetes_ingress_v1.test.metadata.0.name
    namespace = kubernetes_ingress_v1.test.metadata.0.namespace
  }
}

resource "kubernetes_namespace" "test" {
  provider = %s
  metadata {
    name = "%s"
  }
}

resource "kubernetes_service" "test" {
  provider = %s
  metadata {
    name = "%s"
    namespace = kubernetes_namespace.test.metadata.0.name
  }
  spec {
    port {
      port = 80
      target_port = 80
      protocol = "TCP"
    }
    type = "NodePort"
  }
}

resource "kubernetes_ingress_v1" "test" {
  provider = %s
  wait_for_load_balancer = true
  metadata {
    name = "%s"
    namespace = kubernetes_namespace.test.metadata.0.name
    annotations = {
      "kubernetes.io/ingress.class" = "alb"
      "alb.ingress.kubernetes.io/scheme" = "internet-facing"
      "alb.ingress.kubernetes.io/target-type" = "ip"
    }
  }
  spec {
    rule {
      http {
        path {
          path = "/*"
		  path_type = "Prefix"
          backend {
            service {
			  name = kubernetes_service.test.metadata.0.name
              port {
                number = 80
			  } 
			}
          }
        }
      }
    }
  }
}
`, provider, provider, name, provider, name, provider, name)
}
