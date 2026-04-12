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

func TestAccKubernetesHTTPRouteV1_basic(t *testing.T) {
	var conf gatewayv1.HTTPRoute
	rName := acctest.RandomWithPrefix("tf-acc-test")
	gcName := acctest.RandomWithPrefix("tf-acc-test-gc")
	resourceName := "kubernetes_http_route_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckHTTPRouteV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccHTTPRouteV1ConfigBasic(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckHTTPRouteV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", rName),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.backend_refs.#", "1"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "metadata.0.uid"},
			},
		},
	})
}

func TestAccKubernetesHTTPRouteV1_withMatch(t *testing.T) {
	var conf gatewayv1.HTTPRoute
	rName := acctest.RandomWithPrefix("tf-acc-test")
	gcName := acctest.RandomWithPrefix("tf-acc-test-gc")
	resourceName := "kubernetes_http_route_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckHTTPRouteV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccHTTPRouteV1ConfigWithMatch(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckHTTPRouteV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", rName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.matches.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.matches.0.path.0.type", "PathPrefix"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.matches.0.path.0.value", "/api"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.matches.0.method", "GET"),
				),
			},
		},
	})
}

func TestAccKubernetesHTTPRouteV1_withFilters(t *testing.T) {
	var conf gatewayv1.HTTPRoute
	rName := acctest.RandomWithPrefix("tf-acc-test")
	gcName := acctest.RandomWithPrefix("tf-acc-test-gc")
	resourceName := "kubernetes_http_route_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckHTTPRouteV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccHTTPRouteV1ConfigWithFilters(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckHTTPRouteV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", rName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.filters.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.filters.0.type", "RequestHeaderModifier"),
				),
			},
		},
	})
}

func TestAccKubernetesHTTPRouteV1_withHostnames(t *testing.T) {
	var conf gatewayv1.HTTPRoute
	rName := acctest.RandomWithPrefix("tf-acc-test")
	gcName := acctest.RandomWithPrefix("tf-acc-test-gc")
	resourceName := "kubernetes_http_route_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckHTTPRouteV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccHTTPRouteV1ConfigWithHostnames(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckHTTPRouteV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", rName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.hostnames.#", "2"),
				),
			},
		},
	})
}

func testAccCheckHTTPRouteV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).GatewayClientset()
	if err != nil {
		return err
	}
	ctx := context.Background()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_http_route_v1" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.HTTPRoutes(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("HTTPRoute still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckHTTPRouteV1Exists(n string, obj *gatewayv1.HTTPRoute) resource.TestCheckFunc {
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

		out, err := conn.HTTPRoutes(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccHTTPRouteV1ConfigBasic(rName, gcName string) string {
	return fmt.Sprintf(`
resource "kubernetes_service_v1" "test" {
  metadata {
    name = "%[1]s-svc"
  }
  spec {
    selector = {
      app = "test"
    }
    port {
      port        = 8080
      target_port = 80
    }
  }
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
    name = "%[1]s-gw"
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

resource "kubernetes_http_route_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    parent_refs {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }
    rules {
      backend_refs {
        name = kubernetes_service_v1.test.metadata.0.name
        port = 8080
      }
    }
  }
}
`, rName, gcName)
}

func testAccHTTPRouteV1ConfigWithMatch(rName, gcName string) string {
	return fmt.Sprintf(`
resource "kubernetes_service_v1" "test" {
  metadata {
    name = "%[1]s-svc"
  }
  spec {
    selector = {
      app = "test"
    }
    port {
      port        = 8080
      target_port = 80
    }
  }
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
    name = "%[1]s-gw"
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

resource "kubernetes_http_route_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    parent_refs {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }
    rules {
      matches {
        path {
          type  = "PathPrefix"
          value = "/api"
        }
        method = "GET"
        headers {
          name  = "X-Custom-Header"
          value = "test-value"
        }
      }
      backend_refs {
        name = kubernetes_service_v1.test.metadata.0.name
        port = 8080
      }
    }
  }
}
`, rName, gcName)
}

func testAccHTTPRouteV1ConfigWithFilters(rName, gcName string) string {
	return fmt.Sprintf(`
resource "kubernetes_service_v1" "test" {
  metadata {
    name = "%[1]s-svc"
  }
  spec {
    selector = {
      app = "test"
    }
    port {
      port        = 8080
      target_port = 80
    }
  }
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
    name = "%[1]s-gw"
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

resource "kubernetes_http_route_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    parent_refs {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }
    rules {
      filters {
        type = "RequestHeaderModifier"
        request_header_modifier {
          add {
            name  = "X-Custom-Header"
            value = "custom-value"
          }
        }
      }
      backend_refs {
        name = kubernetes_service_v1.test.metadata.0.name
        port = 8080
      }
    }
  }
}
`, rName, gcName)
}

func testAccHTTPRouteV1ConfigWithHostnames(rName, gcName string) string {
	return fmt.Sprintf(`
resource "kubernetes_service_v1" "test" {
  metadata {
    name = "%[1]s-svc"
  }
  spec {
    selector = {
      app = "test"
    }
    port {
      port        = 8080
      target_port = 80
    }
  }
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
    name = "%[1]s-gw"
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

resource "kubernetes_http_route_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    hostnames = ["example.com", "api.example.com"]
    parent_refs {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }
    rules {
      backend_refs {
        name = kubernetes_service_v1.test.metadata.0.name
        port = 8080
      }
    }
  }
}
`, rName, gcName)
}

func TestAccKubernetesHTTPRouteV1_withSessionPersistence(t *testing.T) {
	var conf gatewayv1.HTTPRoute
	rName := acctest.RandomWithPrefix("tf-acc-test")
	gcName := acctest.RandomWithPrefix("tf-acc-test-gc")
	resourceName := "kubernetes_http_route_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckHTTPRouteV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccHTTPRouteV1ConfigWithSessionPersistence(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckHTTPRouteV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.session_persistence.0.session_name", "sticky"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.session_persistence.0.type", "Cookie"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.session_persistence.0.absolute_timeout", "3600s"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.session_persistence.0.idle_timeout", "300s"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.session_persistence.0.cookie_config.0.lifetime_type", "Permanent"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "metadata.0.uid"},
			},
		},
	})
}

func TestAccKubernetesHTTPRouteV1_withRetry(t *testing.T) {
	var conf gatewayv1.HTTPRoute
	rName := acctest.RandomWithPrefix("tf-acc-test")
	gcName := acctest.RandomWithPrefix("tf-acc-test-gc")
	resourceName := "kubernetes_http_route_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckHTTPRouteV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccHTTPRouteV1ConfigWithRetry(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckHTTPRouteV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.retry.0.codes.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.retry.0.codes.0", "503"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.retry.0.codes.1", "504"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.retry.0.attempts", "3"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.retry.0.backoff", "250ms"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.timeouts.0.request", "30s"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.timeouts.0.backend_request", "10s"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "metadata.0.uid"},
			},
		},
	})
}

func TestAccKubernetesHTTPRouteV1_withBackendRefFilters(t *testing.T) {
	var conf gatewayv1.HTTPRoute
	rName := acctest.RandomWithPrefix("tf-acc-test")
	gcName := acctest.RandomWithPrefix("tf-acc-test-gc")
	resourceName := "kubernetes_http_route_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckHTTPRouteV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccHTTPRouteV1ConfigWithBackendRefFilters(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckHTTPRouteV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.backend_refs.0.filters.0.type", "RequestHeaderModifier"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.backend_refs.0.filters.0.request_header_modifier.0.set.0.name", "X-Backend-Version"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.backend_refs.0.filters.0.request_header_modifier.0.set.0.value", "v2"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "metadata.0.uid"},
			},
		},
	})
}

func TestAccKubernetesHTTPRouteV1_withMultipleRules(t *testing.T) {
	var conf gatewayv1.HTTPRoute
	rName := acctest.RandomWithPrefix("tf-acc-test")
	gcName := acctest.RandomWithPrefix("tf-acc-test-gc")
	resourceName := "kubernetes_http_route_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckHTTPRouteV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccHTTPRouteV1ConfigWithMultipleRules(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckHTTPRouteV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.matches.0.path.0.value", "/api"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.1.matches.0.path.0.value", "/admin"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.2.backend_refs.0.name", rName+"-svc"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "metadata.0.uid"},
			},
		},
	})
}

// ─── config helpers for new HTTPRoute tests ───────────────────────────────────

func testAccHTTPRouteV1BaseConfig(rName, gcName string) string {
	return fmt.Sprintf(`
resource "kubernetes_service_v1" "test" {
  metadata {
    name = "%[1]s-svc"
  }
  spec {
    selector = { app = "test" }
    port {
      port        = 8080
      target_port = 80
    }
  }
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
    name = "%[1]s-gw"
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

func testAccHTTPRouteV1ConfigWithSessionPersistence(rName, gcName string) string {
	return testAccHTTPRouteV1BaseConfig(rName, gcName) + fmt.Sprintf(`
resource "kubernetes_http_route_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    parent_refs {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }
    rules {
      session_persistence {
        session_name     = "sticky"
        type             = "Cookie"
        absolute_timeout = "3600s"
        idle_timeout     = "300s"
        cookie_config {
          lifetime_type = "Permanent"
        }
      }
      backend_refs {
        name = kubernetes_service_v1.test.metadata.0.name
        port = 8080
      }
    }
  }
}
`, rName)
}

func testAccHTTPRouteV1ConfigWithRetry(rName, gcName string) string {
	return testAccHTTPRouteV1BaseConfig(rName, gcName) + fmt.Sprintf(`
resource "kubernetes_http_route_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    parent_refs {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }
    rules {
      retry {
        codes    = [503, 504]
        attempts = 3
        backoff  = "250ms"
      }
      timeouts {
        request         = "30s"
        backend_request = "10s"
      }
      backend_refs {
        name = kubernetes_service_v1.test.metadata.0.name
        port = 8080
      }
    }
  }
}
`, rName)
}

func testAccHTTPRouteV1ConfigWithBackendRefFilters(rName, gcName string) string {
	return testAccHTTPRouteV1BaseConfig(rName, gcName) + fmt.Sprintf(`
resource "kubernetes_http_route_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    parent_refs {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }
    rules {
      backend_refs {
        name = kubernetes_service_v1.test.metadata.0.name
        port = 8080
        filters {
          type = "RequestHeaderModifier"
          request_header_modifier {
            set {
              name  = "X-Backend-Version"
              value = "v2"
            }
          }
        }
      }
    }
  }
}
`, rName)
}

func testAccHTTPRouteV1ConfigWithMultipleRules(rName, gcName string) string {
	return testAccHTTPRouteV1BaseConfig(rName, gcName) + fmt.Sprintf(`
resource "kubernetes_http_route_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    parent_refs {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }
    hostnames = ["multi.example.com"]

    rules {
      name = "api"
      matches {
        path {
          type  = "PathPrefix"
          value = "/api"
        }
      }
      backend_refs {
        name = kubernetes_service_v1.test.metadata.0.name
        port = 8080
      }
    }

    rules {
      name = "admin"
      matches {
        path {
          type  = "PathPrefix"
          value = "/admin"
        }
        headers {
          name  = "X-Admin"
          value = "true"
          type  = "Exact"
        }
      }
      backend_refs {
        name = kubernetes_service_v1.test.metadata.0.name
        port = 8080
      }
    }

    rules {
      name = "default"
      backend_refs {
        name = kubernetes_service_v1.test.metadata.0.name
        port = 8080
      }
    }
  }
}
`, rName)
}
