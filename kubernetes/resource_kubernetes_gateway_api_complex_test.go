// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestAccKubernetesHTTPRouteV1_complexFilters(t *testing.T) {
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
				Config: testAccHTTPRouteV1ConfigComplex(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckHTTPRouteV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", rName),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.namespace", "default"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.hostnames.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.hostnames.0", "api.example.com"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.hostnames.1", "www.example.com"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.matches.0.path.0.type", "PathPrefix"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.matches.0.path.0.value", "/api"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.matches.0.method", "GET"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.matches.0.headers.0.name", "X-Version"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.matches.0.headers.0.value", "v2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.backend_refs.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.backend_refs.0.weight", "80"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.backend_refs.1.weight", "20"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.filters.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.filters.0.type", "RequestHeaderModifier"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.filters.0.request_header_modifier.0.add.0.name", "X-Request-Id"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.filters.0.request_header_modifier.0.remove.0", "X-Internal-Token"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.1.filters.0.type", "URLRewrite"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.1.filters.0.url_rewrite.0.path.0.type", "ReplacePrefixMatch"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.1.filters.0.url_rewrite.0.path.0.replace_prefix_match", "/assets"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.2.filters.0.type", "ResponseHeaderModifier"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.2.filters.0.response_header_modifier.0.set.0.name", "Cache-Control"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.2.filters.0.response_header_modifier.0.set.0.value", "no-cache"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.2.timeouts.0.request", "30s"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.2.retry.0.attempts", "3"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.2.retry.0.backoff", "100ms"),
				),
			},
			// Update: change weights (80/20 → 60/40), add 4th rule
			{
				Config: testAccHTTPRouteV1ConfigComplexUpdated(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckHTTPRouteV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.#", "4"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.backend_refs.0.weight", "60"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.backend_refs.1.weight", "40"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.3.matches.0.path.0.type", "Exact"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.3.matches.0.path.0.value", "/health"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.3.matches.0.query_params.0.name", "check"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.3.matches.0.query_params.0.value", "true"),
				),
			},
			// Import verification
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "metadata.0.uid"},
			},
		},
	})
}

// TestAccKubernetesGatewayAPIStack_crossNamespace tests a full cross-namespace Gateway API stack:
// - GatewayClass + Gateway in namespace A
// - Services in namespace A and B
// - ReferenceGrant allowing namespace A gateway to reach namespace B services
// - HTTPRoute in namespace A referencing backend in namespace B
func TestAccKubernetesGatewayAPIStack_crossNamespace(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	gcName := acctest.RandomWithPrefix("tf-acc-test-gc")
	routeResource := "kubernetes_http_route_v1.cross"
	grantResource := "kubernetes_reference_grant_v1.allow"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckHTTPRouteV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIStackCrossNamespace(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(routeResource, "spec.0.rules.0.backend_refs.0.namespace", fmt.Sprintf("%s-b", rName)),
					resource.TestCheckResourceAttr(routeResource, "spec.0.rules.0.backend_refs.0.name", fmt.Sprintf("%s-svc-b", rName)),
					resource.TestCheckResourceAttr(grantResource, "metadata.0.namespace", fmt.Sprintf("%s-b", rName)),
					resource.TestCheckResourceAttr(grantResource, "spec.0.from.0.namespace", "default"),
					resource.TestCheckResourceAttr(grantResource, "spec.0.to.0.kind", "Service"),
				),
			},
		},
	})
}

// TestAccKubernetesGRPCRouteV1_complex tests GRPCRoute with:
// - Method matching (service + method name)
// - Header matching
// - request_header_modifier filter
// - Multiple backend_refs with weights
// - Update: change method, add headers
func TestAccKubernetesGRPCRouteV1_complex(t *testing.T) {
	var conf gatewayv1.GRPCRoute
	rName := acctest.RandomWithPrefix("tf-acc-test")
	gcName := acctest.RandomWithPrefix("tf-acc-test-gc")
	resourceName := "kubernetes_grpc_route_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGRPCRouteV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGRPCRouteV1ConfigComplex(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckGRPCRouteV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.matches.0.method.0.type", "Exact"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.matches.0.method.0.service", "mypackage.MyService"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.matches.0.method.0.method", "GetUser"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.matches.0.headers.0.name", "x-tenant-id"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.matches.0.headers.0.value", "acme"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.backend_refs.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.backend_refs.0.weight", "90"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.backend_refs.1.weight", "10"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.filters.0.type", "RequestHeaderModifier"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.filters.0.request_header_modifier.0.set.0.name", "x-grpc-version"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.filters.0.request_header_modifier.0.set.0.value", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.1.matches.0.method.0.service", "mypackage.HealthService"),
				),
			},
			// Update: change weights, add header to filter
			{
				Config: testAccGRPCRouteV1ConfigComplexUpdated(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.backend_refs.0.weight", "70"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.backend_refs.1.weight", "30"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.filters.0.request_header_modifier.0.add.0.name", "x-canary"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.filters.0.request_header_modifier.0.add.0.value", "true"),
				),
			},
		},
	})
}

// TestAccKubernetesDataSourceHTTPRouteV1_complex verifies that a data source
// correctly reads back all fields of a complex HTTPRoute
func TestAccKubernetesDataSourceHTTPRouteV1_complex(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	gcName := acctest.RandomWithPrefix("tf-acc-test-gc")
	dsName := "data.kubernetes_http_route_v1.read"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckHTTPRouteV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccHTTPRouteV1ConfigComplexWithDataSource(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dsName, "metadata.0.name", rName),
					resource.TestCheckResourceAttr(dsName, "spec.0.rules.#", "2"),
					resource.TestCheckResourceAttr(dsName, "spec.0.rules.0.backend_refs.0.weight", "75"),
					resource.TestCheckResourceAttr(dsName, "spec.0.rules.0.filters.0.type", "RequestHeaderModifier"),
				),
			},
		},
	})
}

func gatewayStackConfig(rName, gcName, protocol string) string {
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
    name = "%[1]s-gw"
  }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.test.metadata.0.name
    listeners {
      name     = "http"
      port     = 80
      protocol = %[3]q
    }
  }
}

resource "kubernetes_service_v1" "backend1" {
  metadata {
    name = "%[1]s-svc-1"
  }
  spec {
    selector = { app = "backend1" }
    port {
      port        = 8080
      target_port = 80
    }
  }
}

resource "kubernetes_service_v1" "backend2" {
  metadata {
    name = "%[1]s-svc-2"
  }
  spec {
    selector = { app = "backend2" }
    port {
      port        = 8080
      target_port = 80
    }
  }
}
`, rName, gcName, protocol)
}

func testAccHTTPRouteV1ConfigComplex(rName, gcName string) string {
	return gatewayStackConfig(rName, gcName, "HTTP") + fmt.Sprintf(`
resource "kubernetes_http_route_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    hostnames = ["api.example.com", "www.example.com"]
    parent_refs {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }

    # Rule 0: path+header+method match, weighted backends, header modifier
    rules {
      matches {
        path {
          type  = "PathPrefix"
          value = "/api"
        }
        method = "GET"
        headers {
          name  = "X-Version"
          value = "v2"
          type  = "Exact"
        }
      }
      filters {
        type = "RequestHeaderModifier"
        request_header_modifier {
          add {
            name  = "X-Request-Id"
            value = "generated"
          }
          set {
            name  = "X-Forwarded-Proto"
            value = "https"
          }
          remove = ["X-Internal-Token"]
        }
      }
      backend_refs {
        name   = kubernetes_service_v1.backend1.metadata.0.name
        port   = 8080
        weight = 80
      }
      backend_refs {
        name   = kubernetes_service_v1.backend2.metadata.0.name
        port   = 8080
        weight = 20
      }
    }

    # Rule 1: url_rewrite filter
    rules {
      matches {
        path {
          type  = "PathPrefix"
          value = "/static"
        }
      }
      filters {
        type = "URLRewrite"
        url_rewrite {
          path {
            type                 = "ReplacePrefixMatch"
            replace_prefix_match = "/assets"
          }
        }
      }
      backend_refs {
        name   = kubernetes_service_v1.backend1.metadata.0.name
        port   = 8080
        weight = 1
      }
    }

    # Rule 2: response_header_modifier + timeouts + retry
    rules {
      matches {
        path {
          type  = "PathPrefix"
          value = "/slow"
        }
      }
      filters {
        type = "ResponseHeaderModifier"
        response_header_modifier {
          set {
            name  = "Cache-Control"
            value = "no-cache"
          }
          remove = ["X-Powered-By"]
        }
      }
      timeouts {
        request         = "30s"
        backend_request = "25s"
      }
      retry {
        codes    = [503, 504]
        attempts = 3
        backoff  = "100ms"
      }
      backend_refs {
        name   = kubernetes_service_v1.backend2.metadata.0.name
        port   = 8080
        weight = 1
      }
    }
  }
}
`, rName)
}

func testAccHTTPRouteV1ConfigComplexUpdated(rName, gcName string) string {
	return gatewayStackConfig(rName, gcName, "HTTP") + fmt.Sprintf(`
resource "kubernetes_http_route_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    hostnames = ["api.example.com", "www.example.com"]
    parent_refs {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }

    # Rule 0: weights changed 80/20 → 60/40
    rules {
      matches {
        path {
          type  = "PathPrefix"
          value = "/api"
        }
        method = "GET"
        headers {
          name  = "X-Version"
          value = "v2"
          type  = "Exact"
        }
      }
      filters {
        type = "RequestHeaderModifier"
        request_header_modifier {
          add {
            name  = "X-Request-Id"
            value = "generated"
          }
          set {
            name  = "X-Forwarded-Proto"
            value = "https"
          }
          remove = ["X-Internal-Token"]
        }
      }
      backend_refs {
        name   = kubernetes_service_v1.backend1.metadata.0.name
        port   = 8080
        weight = 60
      }
      backend_refs {
        name   = kubernetes_service_v1.backend2.metadata.0.name
        port   = 8080
        weight = 40
      }
    }

    # Rule 1: url_rewrite (unchanged)
    rules {
      matches {
        path {
          type  = "PathPrefix"
          value = "/static"
        }
      }
      filters {
        type = "URLRewrite"
        url_rewrite {
          path {
            type                 = "ReplacePrefixMatch"
            replace_prefix_match = "/assets"
          }
        }
      }
      backend_refs {
        name   = kubernetes_service_v1.backend1.metadata.0.name
        port   = 8080
        weight = 1
      }
    }

    # Rule 2: response_header_modifier + timeouts + retry (unchanged)
    rules {
      matches {
        path {
          type  = "PathPrefix"
          value = "/slow"
        }
      }
      filters {
        type = "ResponseHeaderModifier"
        response_header_modifier {
          set {
            name  = "Cache-Control"
            value = "no-cache"
          }
          remove = ["X-Powered-By"]
        }
      }
      timeouts {
        request         = "30s"
        backend_request = "25s"
      }
      retry {
        codes    = [503, 504]
        attempts = 3
        backoff  = "100ms"
      }
      backend_refs {
        name   = kubernetes_service_v1.backend2.metadata.0.name
        port   = 8080
        weight = 1
      }
    }

    # Rule 3: NEW — exact path + query_param match
    rules {
      matches {
        path {
          type  = "Exact"
          value = "/health"
        }
        query_params {
          name  = "check"
          value = "true"
          type  = "Exact"
        }
      }
      backend_refs {
        name   = kubernetes_service_v1.backend1.metadata.0.name
        port   = 8080
        weight = 1
      }
    }
  }
}
`, rName)
}

func testAccHTTPRouteV1ConfigComplexWithDataSource(rName, gcName string) string {
	return gatewayStackConfig(rName, gcName, "HTTP") + fmt.Sprintf(`
resource "kubernetes_http_route_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    hostnames = ["api.example.com"]
    parent_refs {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }
    rules {
      matches {
        path {
          type  = "PathPrefix"
          value = "/api"
        }
      }
      filters {
        type = "RequestHeaderModifier"
        request_header_modifier {
          set {
            name  = "X-Source"
            value = "gateway"
          }
        }
      }
      backend_refs {
        name   = kubernetes_service_v1.backend1.metadata.0.name
        port   = 8080
        weight = 75
      }
      backend_refs {
        name   = kubernetes_service_v1.backend2.metadata.0.name
        port   = 8080
        weight = 25
      }
    }
    rules {
      matches {
        path {
          type  = "PathPrefix"
          value = "/health"
        }
      }
      backend_refs {
        name   = kubernetes_service_v1.backend1.metadata.0.name
        port   = 8080
        weight = 1
      }
    }
  }
}

data "kubernetes_http_route_v1" "read" {
  metadata {
    name      = kubernetes_http_route_v1.test.metadata.0.name
    namespace = kubernetes_http_route_v1.test.metadata.0.namespace
  }
}
`, rName)
}

func testAccGatewayAPIStackCrossNamespace(rName, gcName string) string {
	nsA := "default"
	nsB := fmt.Sprintf("%s-b", rName)
	return fmt.Sprintf(`
resource "kubernetes_namespace_v1" "secondary" {
  metadata {
    name = %[3]q
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
    name      = "%[1]s-gw"
    namespace = %[4]q
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

resource "kubernetes_service_v1" "backend_a" {
  metadata {
    name      = "%[1]s-svc-a"
    namespace = %[4]q
  }
  spec {
    selector = { app = "backend-a" }
    port {
      port        = 8080
      target_port = 80
    }
  }
}

resource "kubernetes_service_v1" "backend_b" {
  metadata {
    name      = "%[1]s-svc-b"
    namespace = kubernetes_namespace_v1.secondary.metadata.0.name
  }
  spec {
    selector = { app = "backend-b" }
    port {
      port        = 8080
      target_port = 80
    }
  }
}

# ReferenceGrant: allow HTTPRoutes in namespace A to reference Services in namespace B
resource "kubernetes_reference_grant_v1" "allow" {
  metadata {
    name      = "%[1]s-grant"
    namespace = kubernetes_namespace_v1.secondary.metadata.0.name
  }
  spec {
    from {
      group     = "gateway.networking.k8s.io"
      kind      = "HTTPRoute"
      namespace = %[4]q
    }
    to {
      group = ""
      kind  = "Service"
    }
  }
}

# HTTPRoute: routes /a → same namespace backend, /b → cross-namespace backend
resource "kubernetes_http_route_v1" "cross" {
  metadata {
    name      = "%[1]s-cross"
    namespace = %[4]q
  }
  spec {
    parent_refs {
      name      = kubernetes_gateway_v1.test.metadata.0.name
      namespace = %[4]q
    }
    rules {
      matches {
        path {
          type  = "PathPrefix"
          value = "/cross"
        }
      }
      backend_refs {
        name      = kubernetes_service_v1.backend_b.metadata.0.name
        namespace = kubernetes_namespace_v1.secondary.metadata.0.name
        port      = 8080
        weight    = 1
      }
    }
  }

  depends_on = [kubernetes_reference_grant_v1.allow]
}
`, rName, gcName, nsB, nsA)
}

func testAccGRPCRouteV1ConfigComplex(rName, gcName string) string {
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
    name = "%[1]s-gw"
  }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.test.metadata.0.name
    listeners {
      name     = "grpc"
      port     = 50051
      protocol = "HTTP"
    }
  }
}

resource "kubernetes_service_v1" "grpc1" {
  metadata {
    name = "%[1]s-grpc-1"
  }
  spec {
    selector = { app = "grpc1" }
    port {
      port        = 50051
      target_port = 50051
    }
  }
}

resource "kubernetes_service_v1" "grpc2" {
  metadata {
    name = "%[1]s-grpc-2"
  }
  spec {
    selector = { app = "grpc2" }
    port {
      port        = 50051
      target_port = 50051
    }
  }
}

resource "kubernetes_grpc_route_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    parent_refs {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }

    # Rule 0: exact method + service match, tenant header, header modifier, weighted backends
    rules {
      matches {
        method {
          type    = "Exact"
          service = "mypackage.MyService"
          method  = "GetUser"
        }
        headers {
          name  = "x-tenant-id"
          value = "acme"
          type  = "Exact"
        }
      }
      filters {
        type = "RequestHeaderModifier"
        request_header_modifier {
          set {
            name  = "x-grpc-version"
            value = "2"
          }
          remove = ["x-debug"]
        }
      }
      backend_refs {
        name   = kubernetes_service_v1.grpc1.metadata.0.name
        port   = 50051
        weight = 90
      }
      backend_refs {
        name   = kubernetes_service_v1.grpc2.metadata.0.name
        port   = 50051
        weight = 10
      }
    }

    # Rule 1: service-only match (health check routing)
    rules {
      matches {
        method {
          type    = "Exact"
          service = "mypackage.HealthService"
        }
      }
      backend_refs {
        name   = kubernetes_service_v1.grpc1.metadata.0.name
        port   = 50051
        weight = 1
      }
    }
  }
}
`, rName, gcName)
}

func testAccGRPCRouteV1ConfigComplexUpdated(rName, gcName string) string {
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
    name = "%[1]s-gw"
  }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.test.metadata.0.name
    listeners {
      name     = "grpc"
      port     = 50051
      protocol = "HTTP"
    }
  }
}

resource "kubernetes_service_v1" "grpc1" {
  metadata {
    name = "%[1]s-grpc-1"
  }
  spec {
    selector = { app = "grpc1" }
    port {
      port        = 50051
      target_port = 50051
    }
  }
}

resource "kubernetes_service_v1" "grpc2" {
  metadata {
    name = "%[1]s-grpc-2"
  }
  spec {
    selector = { app = "grpc2" }
    port {
      port        = 50051
      target_port = 50051
    }
  }
}

resource "kubernetes_grpc_route_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    parent_refs {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }

    # Rule 0: weights 90/10 → 70/30, add canary header to filter
    rules {
      matches {
        method {
          type    = "Exact"
          service = "mypackage.MyService"
          method  = "GetUser"
        }
        headers {
          name  = "x-tenant-id"
          value = "acme"
          type  = "Exact"
        }
      }
      filters {
        type = "RequestHeaderModifier"
        request_header_modifier {
          set {
            name  = "x-grpc-version"
            value = "2"
          }
          add {
            name  = "x-canary"
            value = "true"
          }
          remove = ["x-debug"]
        }
      }
      backend_refs {
        name   = kubernetes_service_v1.grpc1.metadata.0.name
        port   = 50051
        weight = 70
      }
      backend_refs {
        name   = kubernetes_service_v1.grpc2.metadata.0.name
        port   = 50051
        weight = 30
      }
    }

    # Rule 1: service-only match (unchanged)
    rules {
      matches {
        method {
          type    = "Exact"
          service = "mypackage.HealthService"
        }
      }
      backend_refs {
        name   = kubernetes_service_v1.grpc1.metadata.0.name
        port   = 50051
        weight = 1
      }
    }
  }
}
`, rName, gcName)
}
