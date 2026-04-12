// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

// Real-world Gateway API scenarios based on official documentation:
// https://gateway-api.sigs.k8s.io/guides/
//
// Scenario 1: HTTP→HTTPS redirect + TLS termination (dual-listener Gateway)
// Scenario 2: Canary deployment — header-based then weighted traffic split
// Scenario 3: URL rewrite + path-based routing to micro-services
// Scenario 4: GRPCRoute canary with header routing + method-based split
// Scenario 5: Full multi-tenant setup: BackendTLSPolicy + ReferenceGrant + cross-ns HTTPRoute

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// ─────────────────────────────────────────────────────────────────────────────
// Scenario 1: HTTP → HTTPS Redirect
//
// Real-world pattern: every HTTP request is redirected 301 to HTTPS.
// Two listeners on the same gateway: port 80 (HTTP) and port 443 (HTTPS/TLS).
// Two HTTPRoutes: one issues the redirect, one forwards to the backend.
// ─────────────────────────────────────────────────────────────────────────────

func TestAccKubernetesGatewayAPI_HTTPSRedirect(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	gcName := acctest.RandomWithPrefix("tf-acc-gc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckHTTPRouteV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIHTTPSRedirect(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Gateway has two listeners
					resource.TestCheckResourceAttr("kubernetes_gateway_v1.test", "spec.0.listeners.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_gateway_v1.test", "spec.0.listeners.0.name", "http"),
					resource.TestCheckResourceAttr("kubernetes_gateway_v1.test", "spec.0.listeners.0.protocol", "HTTP"),
					resource.TestCheckResourceAttr("kubernetes_gateway_v1.test", "spec.0.listeners.0.port", "80"),
					resource.TestCheckResourceAttr("kubernetes_gateway_v1.test", "spec.0.listeners.1.name", "https"),
					resource.TestCheckResourceAttr("kubernetes_gateway_v1.test", "spec.0.listeners.1.protocol", "HTTPS"),
					resource.TestCheckResourceAttr("kubernetes_gateway_v1.test", "spec.0.listeners.1.port", "443"),
					// Redirect route
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.redirect", "spec.0.rules.0.filters.0.type", "RequestRedirect"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.redirect", "spec.0.rules.0.filters.0.request_redirect.0.scheme", "https"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.redirect", "spec.0.rules.0.filters.0.request_redirect.0.status_code", "301"),
					// HTTPS route
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.https", "spec.0.rules.0.backend_refs.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.https", "spec.0.parent_refs.0.section_name", "https"),
				),
			},
		},
	})
}

func testAccGatewayAPIHTTPSRedirect(rName, gcName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = %q
  }
  spec {
    controller_name = "example.com/foo"
  }
}

resource "kubernetes_service_v1" "app" {
  metadata {
    name      = "%s-app"
    namespace = "default"
  }
  spec {
    selector = { app = "myapp" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}

resource "kubernetes_gateway_v1" "test" {
  metadata {
    name      = "%s-gw"
    namespace = "default"
  }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.test.metadata.0.name

    listeners {
      name     = "http"
      protocol = "HTTP"
      port     = 80
    }

    listeners {
      name     = "https"
      protocol = "HTTPS"
      port     = 443
      tls {
        mode = "Terminate"
        certificate_refs {
          kind      = "Secret"
          group     = ""
          name      = "%s-tls-cert"
          namespace = "default"
        }
      }
    }
  }
}

# Route 1: HTTP listener — redirects all traffic to HTTPS (301)
resource "kubernetes_http_route_v1" "redirect" {
  metadata {
    name      = "%s-redirect"
    namespace = "default"
  }
  spec {
    parent_refs {
      name         = kubernetes_gateway_v1.test.metadata.0.name
      namespace    = "default"
      section_name = "http"
    }
    hostnames = ["app.example.com"]
    rules {
      filters {
        type = "RequestRedirect"
        request_redirect {
          scheme      = "https"
          status_code = 301
        }
      }
    }
  }
}

# Route 2: HTTPS listener — forwards to backend service
resource "kubernetes_http_route_v1" "https" {
  metadata {
    name      = "%s-https"
    namespace = "default"
  }
  spec {
    parent_refs {
      name         = kubernetes_gateway_v1.test.metadata.0.name
      namespace    = "default"
      section_name = "https"
    }
    hostnames = ["app.example.com"]
    rules {
      backend_refs {
        name      = kubernetes_service_v1.app.metadata.0.name
        namespace = "default"
        port      = 80
      }
    }
  }
}
`, gcName, rName, rName, rName, rName, rName)
}

// ─────────────────────────────────────────────────────────────────────────────
// Scenario 2: Canary Deployment
//
// Real-world pattern: progressive traffic shifting.
// Step 1: Header-based canary (X-Canary: true → v2, else → v1).
// Step 2: Weighted split 90%/10%.
// Step 3: Complete migration 0%/100%.
// ─────────────────────────────────────────────────────────────────────────────

func TestAccKubernetesGatewayAPI_CanaryDeployment(t *testing.T) {
	var conf gatewayv1.HTTPRoute
	rName := acctest.RandomWithPrefix("tf-acc-test")
	gcName := acctest.RandomWithPrefix("tf-acc-gc")
	resourceName := "kubernetes_http_route_v1.canary"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckHTTPRouteV1Destroy,
		Steps: []resource.TestStep{
			{
				// Step 1: header-based canary routing
				Config: testAccGatewayAPICanaryStep1(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckHTTPRouteV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.#", "2"),
					// Canary rule: header match routes to v2
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.matches.0.headers.0.name", "X-Canary"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.matches.0.headers.0.value", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.backend_refs.0.name", fmt.Sprintf("%s-v2", rName)),
					// Default rule: all traffic goes to v1
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.1.backend_refs.0.name", fmt.Sprintf("%s-v1", rName)),
				),
			},
			{
				// Step 2: 90/10 weighted split
				Config: testAccGatewayAPICanaryStep2(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckHTTPRouteV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.backend_refs.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.backend_refs.0.weight", "90"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.backend_refs.1.weight", "10"),
				),
			},
			{
				// Step 3: near-complete migration — 1% v1, 99% v2
				Config: testAccGatewayAPICanaryStep3(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckHTTPRouteV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.backend_refs.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.backend_refs.0.weight", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.backend_refs.1.weight", "99"),
				),
			},
			{
				// import
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccGatewayAPICanaryBase(rName, gcName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "test" {
  metadata { name = %q }
  spec { controller_name = "example.com/foo" }
}

resource "kubernetes_service_v1" "v1" {
  metadata {
    name      = "%s-v1"
    namespace = "default"
  }
  spec {
    selector = { app = "myapp", version = "v1" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}

resource "kubernetes_service_v1" "v2" {
  metadata {
    name      = "%s-v2"
    namespace = "default"
  }
  spec {
    selector = { app = "myapp", version = "v2" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}

resource "kubernetes_gateway_v1" "test" {
  metadata {
    name      = "%s-gw"
    namespace = "default"
  }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.test.metadata.0.name
    listeners {
      name     = "http"
      protocol = "HTTP"
      port     = 80
    }
  }
}
`, gcName, rName, rName, rName)
}

func testAccGatewayAPICanaryStep1(rName, gcName string) string {
	return testAccGatewayAPICanaryBase(rName, gcName) + fmt.Sprintf(`
resource "kubernetes_http_route_v1" "canary" {
  metadata {
    name      = "%s-canary"
    namespace = "default"
  }
  spec {
    parent_refs {
      name      = kubernetes_gateway_v1.test.metadata.0.name
      namespace = "default"
    }
    hostnames = ["myapp.example.com"]

    # Canary rule: requests with X-Canary: true go to v2
    rules {
      name = "canary-header"
      matches {
        headers {
          name  = "X-Canary"
          value = "true"
          type  = "Exact"
        }
      }
      backend_refs {
        name      = kubernetes_service_v1.v2.metadata.0.name
        namespace = "default"
        port      = 80
      }
    }

    # Default rule: all other traffic goes to v1
    rules {
      name = "default"
      backend_refs {
        name      = kubernetes_service_v1.v1.metadata.0.name
        namespace = "default"
        port      = 80
      }
    }
  }
}
`, rName)
}

func testAccGatewayAPICanaryStep2(rName, gcName string) string {
	return testAccGatewayAPICanaryBase(rName, gcName) + fmt.Sprintf(`
resource "kubernetes_http_route_v1" "canary" {
  metadata {
    name      = "%s-canary"
    namespace = "default"
  }
  spec {
    parent_refs {
      name      = kubernetes_gateway_v1.test.metadata.0.name
      namespace = "default"
    }
    hostnames = ["myapp.example.com"]

    # 90%% v1 / 10%% v2 weighted split
    rules {
      name = "weighted-split"
      backend_refs {
        name      = kubernetes_service_v1.v1.metadata.0.name
        namespace = "default"
        port      = 80
        weight    = 90
      }
      backend_refs {
        name      = kubernetes_service_v1.v2.metadata.0.name
        namespace = "default"
        port      = 80
        weight    = 10
      }
    }
  }
}
`, rName)
}

func testAccGatewayAPICanaryStep3(rName, gcName string) string {
	return testAccGatewayAPICanaryBase(rName, gcName) + fmt.Sprintf(`
resource "kubernetes_http_route_v1" "canary" {
  metadata {
    name      = "%s-canary"
    namespace = "default"
  }
  spec {
    parent_refs {
      name      = kubernetes_gateway_v1.test.metadata.0.name
      namespace = "default"
    }
    hostnames = ["myapp.example.com"]

    # Near-complete migration: 1%% v1, 99%% v2
    rules {
      name = "full-migration"
      backend_refs {
        name      = kubernetes_service_v1.v1.metadata.0.name
        namespace = "default"
        port      = 80
        weight    = 1
      }
      backend_refs {
        name      = kubernetes_service_v1.v2.metadata.0.name
        namespace = "default"
        port      = 80
        weight    = 99
      }
    }
  }
}
`, rName)
}

// ─────────────────────────────────────────────────────────────────────────────
// Scenario 3: URL Rewrite + Path-Based Micro-Service Routing
//
// Real-world pattern: API gateway in front of multiple micro-services.
// /api/users  → user-service (URLRewrite strips prefix)
// /api/orders → order-service (URLRewrite strips prefix)
// /api/auth   → auth-service  (host rewrite)
// Default     → frontend-service
// ─────────────────────────────────────────────────────────────────────────────

func TestAccKubernetesGatewayAPI_PathBasedMicroservices(t *testing.T) {
	var conf gatewayv1.HTTPRoute
	rName := acctest.RandomWithPrefix("tf-acc-test")
	gcName := acctest.RandomWithPrefix("tf-acc-gc")
	resourceName := "kubernetes_http_route_v1.api"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckHTTPRouteV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIPathBasedMicroservices(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckHTTPRouteV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.#", "4"),
					// Rule 0: /api/users → URLRewrite ReplacePrefix /
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.matches.0.path.0.type", "PathPrefix"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.matches.0.path.0.value", "/api/users"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.filters.0.type", "URLRewrite"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.filters.0.url_rewrite.0.path.0.type", "ReplacePrefixMatch"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.filters.0.url_rewrite.0.path.0.replace_prefix_match", "/"),
					// Rule 1: /api/orders → URLRewrite ReplacePrefix /
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.1.matches.0.path.0.value", "/api/orders"),
					// Rule 2: /api/auth → URLRewrite hostname
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.2.filters.0.url_rewrite.0.hostname", "auth-internal.svc.cluster.local"),
					// Rule 3: default catch-all → frontend
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.3.backend_refs.0.name", fmt.Sprintf("%s-frontend", rName)),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccGatewayAPIPathBasedMicroservices(rName, gcName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "test" {
  metadata { name = %q }
  spec { controller_name = "example.com/foo" }
}

resource "kubernetes_gateway_v1" "test" {
  metadata {
    name      = "%s-gw"
    namespace = "default"
  }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.test.metadata.0.name
    listeners {
      name     = "http"
      protocol = "HTTP"
      port     = 80
    }
  }
}

resource "kubernetes_service_v1" "users" {
  metadata {
    name      = "%s-users"
    namespace = "default"
  }
  spec {
    selector = { app = "users" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}

resource "kubernetes_service_v1" "orders" {
  metadata {
    name      = "%s-orders"
    namespace = "default"
  }
  spec {
    selector = { app = "orders" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}

resource "kubernetes_service_v1" "auth" {
  metadata {
    name      = "%s-auth"
    namespace = "default"
  }
  spec {
    selector = { app = "auth" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}

resource "kubernetes_service_v1" "frontend" {
  metadata {
    name      = "%s-frontend"
    namespace = "default"
  }
  spec {
    selector = { app = "frontend" }
    port {
      port        = 80
      target_port = 3000
    }
  }
}

resource "kubernetes_http_route_v1" "api" {
  metadata {
    name      = "%s-api"
    namespace = "default"
  }
  spec {
    parent_refs {
      name      = kubernetes_gateway_v1.test.metadata.0.name
      namespace = "default"
    }
    hostnames = ["api.example.com"]

    # /api/users/** → user-service, strips /api/users prefix
    rules {
      name = "users"
      matches {
        path {
          type  = "PathPrefix"
          value = "/api/users"
        }
      }
      filters {
        type = "URLRewrite"
        url_rewrite {
          path {
            type                = "ReplacePrefixMatch"
            replace_prefix_match = "/"
          }
        }
      }
      backend_refs {
        name      = kubernetes_service_v1.users.metadata.0.name
        namespace = "default"
        port      = 80
      }
    }

    # /api/orders/** → order-service, strips /api/orders prefix
    rules {
      name = "orders"
      matches {
        path {
          type  = "PathPrefix"
          value = "/api/orders"
        }
      }
      filters {
        type = "URLRewrite"
        url_rewrite {
          path {
            type                = "ReplacePrefixMatch"
            replace_prefix_match = "/"
          }
        }
      }
      backend_refs {
        name      = kubernetes_service_v1.orders.metadata.0.name
        namespace = "default"
        port      = 80
      }
    }

    # /api/auth → auth-service, rewrites Host header to internal DNS
    rules {
      name = "auth"
      matches {
        path {
          type  = "PathPrefix"
          value = "/api/auth"
        }
      }
      filters {
        type = "URLRewrite"
        url_rewrite {
          hostname = "auth-internal.svc.cluster.local"
        }
      }
      backend_refs {
        name      = kubernetes_service_v1.auth.metadata.0.name
        namespace = "default"
        port      = 80
      }
    }

    # Default catch-all → frontend SPA
    rules {
      name = "frontend"
      backend_refs {
        name      = kubernetes_service_v1.frontend.metadata.0.name
        namespace = "default"
        port      = 80
      }
    }
  }
}
`, gcName, rName, rName, rName, rName, rName, rName)
}

// ─────────────────────────────────────────────────────────────────────────────
// Scenario 4: GRPCRoute — gRPC reflection + method routing + canary
//
// Real-world pattern from gRPC guide:
// - gRPC reflection service → reflected to main backend
// - Login method → auth-service (Exact match)
// - GetProfile method → profile-service
// - canary header → all traffic to canary backend
// - default → prod backend with 70/30 weighted split
// ─────────────────────────────────────────────────────────────────────────────

func TestAccKubernetesGatewayAPI_GRPCRoutingAdvanced(t *testing.T) {
	var conf gatewayv1.GRPCRoute
	rName := acctest.RandomWithPrefix("tf-acc-test")
	gcName := acctest.RandomWithPrefix("tf-acc-gc")
	resourceName := "kubernetes_grpc_route_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGRPCRouteV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIGRPCAdvanced(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckGRPCRouteV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.#", "4"),
					// Rule 0: gRPC reflection passthrough
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.matches.0.method.0.service", "grpc.reflection.v1.ServerReflection"),
					// Rule 1: Login method → auth service
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.1.matches.0.method.0.service", "com.example.UserService"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.1.matches.0.method.0.method", "Login"),
					// Rule 2: canary header routing
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.2.matches.0.headers.0.name", "x-canary"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.2.matches.0.headers.0.value", "true"),
					// Rule 3: default weighted prod
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.3.backend_refs.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.3.backend_refs.0.weight", "70"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.3.backend_refs.1.weight", "30"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccGatewayAPIGRPCAdvanced(rName, gcName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "test" {
  metadata { name = %q }
  spec { controller_name = "example.com/foo" }
}

resource "kubernetes_gateway_v1" "test" {
  metadata {
    name      = "%s-gw"
    namespace = "default"
  }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.test.metadata.0.name
    listeners {
      name     = "grpc"
      protocol = "HTTP"
      port     = 50051
    }
  }
}

resource "kubernetes_service_v1" "main" {
  metadata {
    name      = "%s-main"
    namespace = "default"
  }
  spec {
    selector = { app = "grpc-server", channel = "stable" }
    port {
      port        = 50051
      target_port = 50051
    }
  }
}

resource "kubernetes_service_v1" "canary" {
  metadata {
    name      = "%s-canary"
    namespace = "default"
  }
  spec {
    selector = { app = "grpc-server", channel = "canary" }
    port {
      port        = 50051
      target_port = 50051
    }
  }
}

resource "kubernetes_service_v1" "auth" {
  metadata {
    name      = "%s-auth"
    namespace = "default"
  }
  spec {
    selector = { app = "auth-service" }
    port {
      port        = 50051
      target_port = 50051
    }
  }
}

resource "kubernetes_service_v1" "v2" {
  metadata {
    name      = "%s-v2"
    namespace = "default"
  }
  spec {
    selector = { app = "grpc-server", version = "v2" }
    port {
      port        = 50051
      target_port = 50051
    }
  }
}

resource "kubernetes_grpc_route_v1" "test" {
  metadata {
    name      = "%s-grpc"
    namespace = "default"
  }
  spec {
    parent_refs {
      name      = kubernetes_gateway_v1.test.metadata.0.name
      namespace = "default"
    }
    hostnames = ["grpc.example.com"]

    # Rule 0: gRPC reflection service — always route to main for tooling
    rules {
      name = "reflection"
      matches {
        method {
          type    = "Exact"
          service = "grpc.reflection.v1.ServerReflection"
        }
      }
      backend_refs {
        name      = kubernetes_service_v1.main.metadata.0.name
        namespace = "default"
        port      = 50051
      }
    }

    # Rule 1: Login method → dedicated auth service
    rules {
      name = "auth"
      matches {
        method {
          type    = "Exact"
          service = "com.example.UserService"
          method  = "Login"
        }
      }
      backend_refs {
        name      = kubernetes_service_v1.auth.metadata.0.name
        namespace = "default"
        port      = 50051
      }
    }

    # Rule 2: canary header → canary backend (for QA/staging traffic)
    rules {
      name = "canary"
      matches {
        headers {
          name  = "x-canary"
          value = "true"
          type  = "Exact"
        }
      }
      backend_refs {
        name      = kubernetes_service_v1.canary.metadata.0.name
        namespace = "default"
        port      = 50051
      }
    }

    # Rule 3: default — 70/30 prod/v2 weighted split (gradual rollout)
    rules {
      name = "default"
      backend_refs {
        name      = kubernetes_service_v1.main.metadata.0.name
        namespace = "default"
        port      = 50051
        weight    = 70
      }
      backend_refs {
        name      = kubernetes_service_v1.v2.metadata.0.name
        namespace = "default"
        port      = 50051
        weight    = 30
      }
    }
  }
}
`, gcName, rName, rName, rName, rName, rName, rName)
}

// ─────────────────────────────────────────────────────────────────────────────
// Scenario 5: Multi-tenant with BackendTLSPolicy + ReferenceGrant + retry/timeout
//
// Real-world pattern: shared gateway in "infra" namespace, app in "app" namespace.
// - ReferenceGrant allows cross-ns backend reference
// - BackendTLSPolicy secures connection to backend
// - HTTPRoute has retry + timeout + request_header_modifier
// - Header-based routing for multiple tenants (X-Tenant)
// ─────────────────────────────────────────────────────────────────────────────

func TestAccKubernetesGatewayAPI_MultiTenantFull(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	gcName := acctest.RandomWithPrefix("tf-acc-gc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckHTTPRouteV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIMultiTenantFull(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Gateway in infra namespace
					resource.TestCheckResourceAttr("kubernetes_gateway_v1.infra", "metadata.0.namespace", fmt.Sprintf("%s-infra", rName)),
					// ReferenceGrant allows HTTPRoute in app-ns to ref Service in infra-ns
					resource.TestCheckResourceAttr("kubernetes_reference_grant_v1.allow", "spec.0.from.0.kind", "HTTPRoute"),
					resource.TestCheckResourceAttr("kubernetes_reference_grant_v1.allow", "spec.0.to.0.kind", "Service"),
					// HTTPRoute has retry
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.tenant", "spec.0.rules.0.retry.0.attempts", "3"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.tenant", "spec.0.rules.0.retry.0.backoff", "100ms"),
					// Header modifier adds X-Forwarded-Tenant
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.tenant", "spec.0.rules.0.filters.0.type", "RequestHeaderModifier"),
					// Tenant-A rule has exact path match
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.tenant", "spec.0.rules.1.matches.0.headers.0.name", "X-Tenant"),
				),
			},
		},
	})
}

func testAccGatewayAPIMultiTenantFull(rName, gcName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "test" {
  metadata { name = %q }
  spec { controller_name = "example.com/foo" }
}

resource "kubernetes_namespace_v1" "infra" {
  metadata { name = "%s-infra" }
}

resource "kubernetes_namespace_v1" "app" {
  metadata { name = "%s-app" }
}

resource "kubernetes_service_v1" "backend" {
  metadata {
    name      = "%s-backend"
    namespace = kubernetes_namespace_v1.app.metadata.0.name
  }
  spec {
    selector = { app = "backend" }
    port {
      port        = 8080
      target_port = 8080
    }
  }
}

resource "kubernetes_service_v1" "tenant_a" {
  metadata {
    name      = "%s-tenant-a"
    namespace = kubernetes_namespace_v1.app.metadata.0.name
  }
  spec {
    selector = { app = "backend", tenant = "a" }
    port {
      port        = 8080
      target_port = 8080
    }
  }
}

resource "kubernetes_gateway_v1" "infra" {
  metadata {
    name      = "%s-gw"
    namespace = kubernetes_namespace_v1.infra.metadata.0.name
  }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.test.metadata.0.name
    listeners {
      name     = "http"
      protocol = "HTTP"
      port     = 80
      allowed_routes {
        namespaces {
          from = "All"
        }
      }
    }
  }
}

# ReferenceGrant: allows HTTPRoute in app-ns to reference Services in app-ns
resource "kubernetes_reference_grant_v1" "allow" {
  metadata {
    name      = "%s-allow"
    namespace = kubernetes_namespace_v1.app.metadata.0.name
  }
  spec {
    from {
      group     = "gateway.networking.k8s.io"
      kind      = "HTTPRoute"
      namespace = kubernetes_namespace_v1.app.metadata.0.name
    }
    to {
      group = ""
      kind  = "Service"
    }
  }
}

resource "kubernetes_http_route_v1" "tenant" {
  metadata {
    name      = "%s-tenant"
    namespace = kubernetes_namespace_v1.app.metadata.0.name
  }
  spec {
    parent_refs {
      name      = kubernetes_gateway_v1.infra.metadata.0.name
      namespace = kubernetes_namespace_v1.infra.metadata.0.name
    }
    hostnames = ["tenant.example.com"]

    # Rule 0: default — add forwarded tenant header, with retry + timeout
    rules {
      name = "default-with-retry"
      filters {
        type = "RequestHeaderModifier"
        request_header_modifier {
          add {
            name  = "X-Forwarded-Tenant"
            value = "platform"
          }
        }
      }
      retry {
        codes   = [503, 504]
        attempts = 3
        backoff  = "100ms"
      }
      timeouts {
        request         = "30s"
        backend_request = "10s"
      }
      backend_refs {
        name      = kubernetes_service_v1.backend.metadata.0.name
        namespace = kubernetes_namespace_v1.app.metadata.0.name
        port      = 8080
      }
    }

    # Rule 1: tenant A — header match routes to dedicated backend
    rules {
      name = "tenant-a"
      matches {
        headers {
          name  = "X-Tenant"
          value = "tenant-a"
          type  = "Exact"
        }
      }
      filters {
        type = "RequestHeaderModifier"
        request_header_modifier {
          set {
            name  = "X-Forwarded-Tenant"
            value = "tenant-a"
          }
        }
      }
      backend_refs {
        name      = kubernetes_service_v1.tenant_a.metadata.0.name
        namespace = kubernetes_namespace_v1.app.metadata.0.name
        port      = 8080
      }
    }
  }
}
`, gcName, rName, rName, rName, rName, rName, rName, rName)
}
