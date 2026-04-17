// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestAccKubernetesGatewayV1_basic(t *testing.T) {
	var conf gatewayv1.Gateway
	rName := acctest.RandomWithPrefix("tf-acc-test")
	gcName := acctest.RandomWithPrefix("tf-acc-test-gc")
	resourceName := "kubernetes_gateway_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGatewayV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayV1ConfigBasic(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckGatewayV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", rName),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.gateway_class_name", gcName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.name", "http"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.port", "80"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.protocol", "HTTP"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "metadata.0.uid", "metadata.0.generation", "status"},
			},
		},
	})
}

func TestAccKubernetesGatewayV1_listenerTLS(t *testing.T) {
	var conf gatewayv1.Gateway
	rName := acctest.RandomWithPrefix("tf-acc-test")
	gcName := acctest.RandomWithPrefix("tf-acc-test-gc")
	resourceName := "kubernetes_gateway_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGatewayV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayV1ConfigListenerTLS(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckGatewayV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", rName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.tls.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.tls.0.mode", "Terminate"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.tls.0.certificate_refs.#", "1"),
				),
			},
		},
	})
}

func TestAccKubernetesGatewayV1_allowedRoutes(t *testing.T) {
	var conf gatewayv1.Gateway
	rName := acctest.RandomWithPrefix("tf-acc-test")
	gcName := acctest.RandomWithPrefix("tf-acc-test-gc")
	resourceName := "kubernetes_gateway_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGatewayV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayV1ConfigAllowedRoutes(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckGatewayV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", rName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.allowed_routes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.allowed_routes.0.namespaces.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.allowed_routes.0.namespaces.0.from", "All"),
				),
			},
		},
	})
}

func TestAccKubernetesGatewayV1_allowedRoutesWithSelector(t *testing.T) {
	var conf gatewayv1.Gateway
	rName := acctest.RandomWithPrefix("tf-acc-test")
	gcName := acctest.RandomWithPrefix("tf-acc-test-gc")
	resourceName := "kubernetes_gateway_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGatewayV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayV1ConfigAllowedRoutesWithSelector(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckGatewayV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", rName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.allowed_routes.0.namespaces.0.from", "Selector"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.allowed_routes.0.namespaces.0.selector.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.allowed_routes.0.namespaces.0.selector.0.match_labels.env", "prod"),
				),
			},
		},
	})
}

func testAccCheckGatewayV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).GatewayClientset()
	if err != nil {
		return err
	}
	ctx := context.Background()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_gateway_v1" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.Gateways(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Gateway still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckGatewayV1Exists(n string, obj *gatewayv1.Gateway) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn, err := testAccProvider.Meta().(KubeClientsets).GatewayClientset()
		if err != nil {
			return err
		}

		ctx := context.Background()
		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		out, err := conn.Gateways(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccGatewayV1ConfigBasic(rName, gcName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = %[2]q
  }
  spec {
    controller_name = "example.com/gateway-controller"
  }
}

resource "kubernetes_gateway_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.test.metadata.0.name
    listeners {
      name     = "http"
      port     = 80
      protocol = "HTTP"
    }
  }
}
`, rName, gcName)
}

func testAccGatewayV1ConfigListenerTLS(rName, gcName string) string {
	return fmt.Sprintf(`
resource "kubernetes_secret_v1" "test" {
  metadata {
    name = "tls-cert"
  }
  data = {
    "tls.crt" = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURCVENDQWUyZ0F3SUJBZ0lJQTFiZXhwMWNrQ2dwb2tIRUJnTlZYUUtNSW5WdWFhbkxtZXM5Nm1rY2dFZ2hSU0cK"
    "tls.key" = "LS0tLS1CRUdJTiBQUklWQVRFIEtFQUtJTktRRU1CMEdBMVVFQ3d3U0dIaGRYSm5RMjluYjJzd0NnWmpZSEJCMApNREI4Q0RBUXdDdApJRlJsMmdGME1ERXdIUVF1TmpBd0x6RXRNQ3NHQ2djaElIcGdiM0JsY2dGa2MyVXVhR1Y2ZEc5d2IyOXNNQ3dHCk1EQXdPd1l6Q3hJRWxVUXdWdWFTNEtXVXVhVTFwa0c1cnhkRzl2YjJzd0lIWXVNVEF4TVRVMExqRXdOVGMzTVRVCk1UQXdNZ2dJUVdGblFWTlJNVkZUTWdveGJHMWxZMmhsY21GMWIzSmxJRU52YlhCdmMyOXpJRkJsY21Ga2QyVXkK"


  }
  type = "kubernetes.io/tls"
}

resource "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = %[2]q
  }
  spec {
    controller_name = "example.com/gateway-controller"
  }
}

resource "kubernetes_gateway_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.test.metadata.0.name
    listeners {
      name     = "https"
      port     = 443
      protocol = "HTTPS"
      tls {
        mode = "Terminate"
        certificate_refs {
          name = kubernetes_secret_v1.test.metadata.0.name
          kind = "Secret"
        }
      }
    }
  }
}
`, rName, gcName)
}

func testAccGatewayV1ConfigAllowedRoutes(rName, gcName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = %[2]q
  }
  spec {
    controller_name = "example.com/gateway-controller"
  }
}

resource "kubernetes_gateway_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.test.metadata.0.name
    listeners {
      name     = "http"
      port     = 80
      protocol = "HTTP"
      allowed_routes {
        namespaces {
          from = "All"
        }
      }
    }
  }
}
`, rName, gcName)
}

func testAccGatewayV1ConfigAllowedRoutesWithSelector(rName, gcName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = %[2]q
  }
  spec {
    controller_name = "example.com/gateway-controller"
  }
}

resource "kubernetes_gateway_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.test.metadata.0.name
    listeners {
      name     = "http"
      port     = 80
      protocol = "HTTP"
      allowed_routes {
        namespaces {
          from = "Selector"
          selector {
            match_labels = {
              env = "prod"
            }
          }
        }
      }
    }
  }
}
`, rName, gcName)
}
