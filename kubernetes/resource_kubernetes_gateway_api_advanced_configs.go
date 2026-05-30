// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import "fmt"

// ---------------------------------------------------------------------------
// Canary deployment configs
// ---------------------------------------------------------------------------

func testAccGatewayAPIAdvancedCanaryBefore(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "canary" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "canary" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.canary.metadata[0].name
    listeners {
      name     = "http"
      hostname = "*.example.com"
      port     = 80
      protocol = "HTTP"
      allowed_routes {
        namespaces { from = "All" }
        kinds { kind = "HTTPRoute" }
      }
    }
  }
}
resource "kubernetes_service_v1" "stable" {
  metadata { name = "%[1]s-stable" }
  spec {
    selector = { app = "stable" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}
resource "kubernetes_service_v1" "canary" {
  metadata { name = "%[1]s-canary" }
  spec {
    selector = { app = "canary" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}
resource "kubernetes_http_route_v1" "canary" {
  metadata { name = "%[1]s-route" }
  spec {
    parent_refs { name = kubernetes_gateway_v1.canary.metadata[0].name }
    rules {
      matches {
        path {
          type  = "PathPrefix"
          value = "/"
        }
      }
      backend_refs {
        name   = kubernetes_service_v1.stable.metadata[0].name
        port   = 80
        weight = 90
      }
      backend_refs {
        name   = kubernetes_service_v1.canary.metadata[0].name
        port   = 80
        weight = 10
      }
    }
  }
}
`, rName)
}

func testAccGatewayAPIAdvancedCanaryAfter(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "canary" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "canary" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.canary.metadata[0].name
    listeners {
      name     = "http"
      hostname = "*.example.com"
      port     = 80
      protocol = "HTTP"
      allowed_routes {
        namespaces { from = "All" }
        kinds { kind = "HTTPRoute" }
      }
    }
  }
}
resource "kubernetes_service_v1" "stable" {
  metadata { name = "%[1]s-stable" }
  spec {
    selector = { app = "stable" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}
resource "kubernetes_service_v1" "canary" {
  metadata { name = "%[1]s-canary" }
  spec {
    selector = { app = "canary" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}
resource "kubernetes_http_route_v1" "canary" {
  metadata { name = "%[1]s-route" }
  spec {
    parent_refs { name = kubernetes_gateway_v1.canary.metadata[0].name }
    rules {
      matches {
        path {
          type  = "PathPrefix"
          value = "/"
        }
      }
      backend_refs {
        name   = kubernetes_service_v1.stable.metadata[0].name
        port   = 80
        weight = 50
      }
      backend_refs {
        name   = kubernetes_service_v1.canary.metadata[0].name
        port   = 80
        weight = 50
      }
    }
  }
}
`, rName)
}

// ---------------------------------------------------------------------------
// Multi-tenant namespace isolation config
// ---------------------------------------------------------------------------

func testAccGatewayAPIAdvancedMultiTenant(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "tenant" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "tenant" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.tenant.metadata[0].name
    listeners {
      name     = "http"
      hostname = "*.tenant.example.com"
      port     = 80
      protocol = "HTTP"
      allowed_routes {
        namespaces {
          from     = "Selector"
          selector {
            match_labels = { tenant = "a" }
          }
        }
        kinds { kind = "HTTPRoute" }
      }
    }
  }
}
`, rName)
}

// ---------------------------------------------------------------------------
// HTTPRoute ResponseHeaderModifier config
// ---------------------------------------------------------------------------

func testAccGatewayAPIAdvancedResponseHeaders(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "rh" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "rh" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.rh.metadata[0].name
    listeners {
      name     = "http"
      hostname = "*.example.com"
      port     = 80
      protocol = "HTTP"
      allowed_routes {
        namespaces { from = "All" }
        kinds { kind = "HTTPRoute" }
      }
    }
  }
}
resource "kubernetes_service_v1" "rh" {
  metadata { name = "%[1]s-svc" }
  spec {
    selector = { app = "test" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}
resource "kubernetes_http_route_v1" "resp_headers" {
  metadata { name = "%[1]s-route" }
  spec {
    parent_refs { name = kubernetes_gateway_v1.rh.metadata[0].name }
    rules {
      matches {
        path {
          type  = "PathPrefix"
          value = "/"
        }
      }
      filters {
        type = "ResponseHeaderModifier"
        response_header_modifier {
          add {
            name  = "X-Custom-Response"
            value = "added"
          }
          set {
            name  = "X-Set-Response"
            value = "set"
          }
          remove = ["X-Remove-Response"]
        }
      }
      filters {
        type = "RequestHeaderModifier"
        request_header_modifier {
          add {
            name  = "X-Request-ID"
            value = "test"
          }
        }
      }
      backend_refs {
        name = kubernetes_service_v1.rh.metadata[0].name
        port = 80
      }
    }
  }
}
`, rName)
}

// ---------------------------------------------------------------------------
// HTTPRoute multiple matches (AND logic) config
// ---------------------------------------------------------------------------

func testAccGatewayAPIAdvancedAndMatches(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "am" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "am" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.am.metadata[0].name
    listeners {
      name     = "http"
      hostname = "*.example.com"
      port     = 80
      protocol = "HTTP"
      allowed_routes {
        namespaces { from = "All" }
        kinds { kind = "HTTPRoute" }
      }
    }
  }
}
resource "kubernetes_service_v1" "am" {
  metadata { name = "%[1]s-svc" }
  spec {
    selector = { app = "test" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}
resource "kubernetes_http_route_v1" "and_matches" {
  metadata { name = "%[1]s-route" }
  spec {
    parent_refs { name = kubernetes_gateway_v1.am.metadata[0].name }
    rules {
      matches {
        path {
          type  = "PathPrefix"
          value = "/api"
        }
        headers {
          name  = "X-API-Key"
          value = "secret"
        }
        method = "POST"
      }
      backend_refs {
        name = kubernetes_service_v1.am.metadata[0].name
        port = 80
      }
    }
  }
}
`, rName)
}

// ---------------------------------------------------------------------------
// Infrastructure config (labels + annotations)
// ---------------------------------------------------------------------------

func testAccGatewayAPIAdvancedInfrastructure(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "infra" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "infra" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.infra.metadata[0].name
    listeners {
      name     = "http"
      hostname = "*.example.com"
      port     = 80
      protocol = "HTTP"
      allowed_routes {
        namespaces { from = "All" }
        kinds { kind = "HTTPRoute" }
      }
    }
    infrastructure {
      labels = {
        app  = "gateway"
        team = "platform"
      }
      annotations = {
        "prometheus.io/scrape" = "true"
        "prometheus.io/port"   = "9102"
      }
    }
  }
}
`, rName)
}

// ---------------------------------------------------------------------------
// API gateway pattern (CORS + Mirror + HeaderModifier)
// ---------------------------------------------------------------------------

func testAccGatewayAPIAdvancedAPIGateway(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "apigw" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "apigw" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.apigw.metadata[0].name
    listeners {
      name     = "http"
      hostname = "*.example.com"
      port     = 80
      protocol = "HTTP"
      allowed_routes {
        namespaces { from = "All" }
        kinds { kind = "HTTPRoute" }
      }
    }
  }
}
resource "kubernetes_service_v1" "api" {
  metadata { name = "%[1]s-api" }
  spec {
    selector = { app = "api" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}
resource "kubernetes_service_v1" "mirror" {
  metadata { name = "%[1]s-mirror" }
  spec {
    selector = { app = "mirror" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}
resource "kubernetes_http_route_v1" "api_gw" {
  metadata { name = "%[1]s-route" }
  spec {
    parent_refs { name = kubernetes_gateway_v1.apigw.metadata[0].name }
    rules {
      matches {
        path {
          type  = "PathPrefix"
          value = "/api"
        }
      }
      filters {
        type = "CORS"
        cors {
          allow_origins  = ["https://app.example.com"]
          allow_methods  = ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
          allow_headers  = ["Authorization", "Content-Type"]
          expose_headers = ["X-Request-Id"]
          max_age        = 7200
        }
      }
      filters {
        type = "RequestMirror"
        request_mirror {
          percent = 10
          backend_ref {
            name = kubernetes_service_v1.mirror.metadata[0].name
            port = 80
          }
        }
      }
      filters {
        type = "RequestHeaderModifier"
        request_header_modifier {
          set {
            name  = "X-Forwarded-Proto"
            value = "https"
          }
        }
      }
      backend_refs {
        name = kubernetes_service_v1.api.metadata[0].name
        port = 80
      }
    }
  }
}
`, rName)
}

// ---------------------------------------------------------------------------
// Redirect and rewrite config
// ---------------------------------------------------------------------------

func testAccGatewayAPIAdvancedRedirectRewrite(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "rr" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "rr" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.rr.metadata[0].name
    listeners {
      name     = "http"
      hostname = "*.example.com"
      port     = 80
      protocol = "HTTP"
      allowed_routes {
        namespaces { from = "All" }
        kinds { kind = "HTTPRoute" }
      }
    }
  }
}
resource "kubernetes_service_v1" "rr" {
  metadata { name = "%[1]s-svc" }
  spec {
    selector = { app = "test" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}
resource "kubernetes_http_route_v1" "redirect" {
  metadata { name = "%[1]s-route" }
  spec {
    parent_refs { name = kubernetes_gateway_v1.rr.metadata[0].name }
    rules {
      matches {
        path {
          type  = "PathPrefix"
          value = "/v1"
        }
      }
      filters {
        type = "RequestRedirect"
        request_redirect {
          scheme   = "https"
          port     = 443
          status_code = 301
        }
      }
    }
  }
}
`, rName)
}

// ---------------------------------------------------------------------------
// GRPCRoute with headers + weighted backends
// ---------------------------------------------------------------------------

func testAccGatewayAPIAdvancedGRPCRouteHeaders(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "grh" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "grh" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.grh.metadata[0].name
    listeners {
      name     = "grpc"
      hostname = "*.grpc.example.com"
      port     = 8443
      protocol = "GRPC"
      allowed_routes {
        namespaces { from = "All" }
        kinds { kind = "GRPCRoute" }
      }
    }
  }
}
resource "kubernetes_service_v1" "v1" {
  metadata { name = "%[1]s-svc-v1" }
  spec {
    selector = { app = "v1" }
    port {
      port        = 443
      target_port = 8443
    }
  }
}
resource "kubernetes_service_v1" "v2" {
  metadata { name = "%[1]s-svc-v2" }
  spec {
    selector = { app = "v2" }
    port {
      port        = 443
      target_port = 8443
    }
  }
}
resource "kubernetes_grpc_route_v1" "headers" {
  metadata { name = "%[1]s-grpc" }
  spec {
    parent_refs { name = kubernetes_gateway_v1.grh.metadata[0].name }
    rules {
      matches {
        method {
          service = "payment.Service"
          method  = "Process"
        }
        headers {
          name  = "x-version"
          value = "v2"
        }
      }
      backend_refs {
        name   = kubernetes_service_v1.v1.metadata[0].name
        port   = 443
        weight = 80
      }
      backend_refs {
        name   = kubernetes_service_v1.v2.metadata[0].name
        port   = 443
        weight = 20
      }
    }
  }
}
`, rName)
}

// ---------------------------------------------------------------------------
// TLSRoute SNI-based routing
// ---------------------------------------------------------------------------

func testAccGatewayAPIAdvancedTLSRouteSNI(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "tsni" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "tsni" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.tsni.metadata[0].name
    listeners {
      name     = "tls"
      hostname = "*.tls.example.com"
      port     = 443
      protocol = "TLS"
      allowed_routes {
        namespaces { from = "All" }
        kinds { kind = "TLSRoute" }
      }
    }
  }
}
resource "kubernetes_service_v1" "sni_a" {
  metadata { name = "%[1]s-sni-a" }
  spec {
    selector = { app = "sni-a" }
    port {
      port        = 443
      target_port = 8443
    }
  }
}
resource "kubernetes_service_v1" "sni_b" {
  metadata { name = "%[1]s-sni-b" }
  spec {
    selector = { app = "sni-b" }
    port {
      port        = 443
      target_port = 8443
    }
  }
}
resource "kubernetes_tls_route_v1" "sni" {
  metadata { name = "%[1]s-tls" }
  spec {
    parent_refs { name = kubernetes_gateway_v1.tsni.metadata[0].name }
    hostnames = ["a.tls.example.com", "b.tls.example.com"]
    rules {
      backend_refs {
        name = kubernetes_service_v1.sni_a.metadata[0].name
        port = 443
      }
    }
    rules {
      backend_refs {
        name = kubernetes_service_v1.sni_b.metadata[0].name
        port = 443
      }
    }
  }
}
`, rName)
}

// ---------------------------------------------------------------------------
// Gateway with allowedListeners
// ---------------------------------------------------------------------------

func testAccGatewayAPIAdvancedAllowedListeners(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "als" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "allowed_ls" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.als.metadata[0].name
    listeners {
      name     = "http"
      hostname = "*.example.com"
      port     = 80
      protocol = "HTTP"
      allowed_routes {
        namespaces { from = "All" }
        kinds { kind = "HTTPRoute" }
      }
    }
    allowed_listeners {
      namespaces {
        from = "Selector"
        selector {
          match_labels = { "gateway-team" = "infra" }
        }
      }
    }
  }
}
`, rName)
}

// ---------------------------------------------------------------------------
// Redirect path modifier (ReplacePrefixMatch)
// ---------------------------------------------------------------------------

func testAccGatewayAPIAdvancedRedirectPath(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "rp" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "rp" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.rp.metadata[0].name
    listeners {
      name     = "http"
      hostname = "*.example.com"
      port     = 80
      protocol = "HTTP"
      allowed_routes {
        namespaces { from = "All" }
        kinds { kind = "HTTPRoute" }
      }
    }
  }
}
resource "kubernetes_service_v1" "rp" {
  metadata { name = "%[1]s-svc" }
  spec {
    selector = { app = "test" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}
resource "kubernetes_http_route_v1" "redirect_path" {
  metadata { name = "%[1]s-route" }
  spec {
    parent_refs { name = kubernetes_gateway_v1.rp.metadata[0].name }
    rules {
      matches {
        path {
          type  = "PathPrefix"
          value = "/v1"
        }
      }
      filters {
        type = "RequestRedirect"
        request_redirect {
          path {
            type               = "ReplacePrefixMatch"
            replace_prefix_match = "/v2"
          }
        }
      }
    }
  }
}
`, rName)
}

// ---------------------------------------------------------------------------
// BackendTLSPolicy with CA certificate ConfigMap ref
// ---------------------------------------------------------------------------

func testAccGatewayAPIAdvancedBackendTLSWithCA(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_service_v1" "ca" {
  metadata { name = "%[1]s-svc" }
  spec {
    selector = { app = "backend" }
    port {
      port        = 443
      target_port = 8443
    }
  }
}
resource "kubernetes_backend_tls_policy_v1" "ca_ref" {
  metadata { name = "%[1]s-btls" }
  spec {
    target_refs {
      group = ""
      kind  = "Service"
      name  = kubernetes_service_v1.ca.metadata[0].name
    }
    validation {
      hostname = "api.internal.example.com"
      ca_certificate_refs {
        group = ""
        kind  = "ConfigMap"
        name  = "%[1]s-ca-bundle"
      }
    }
  }
}
`, rName)
}

// ---------------------------------------------------------------------------
// Production stack - full real-world deployment
// ---------------------------------------------------------------------------

func testAccGatewayAPIAdvancedProductionStack(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_namespace_v1" "app" {
  metadata { name = "%[1]s-app" }
}
resource "kubernetes_namespace_v1" "backend" {
  metadata { name = "%[1]s-backend" }
}
resource "kubernetes_gateway_class_v1" "prod" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "prod" {
  metadata {
    name      = "%[1]s-gw"
    namespace = kubernetes_namespace_v1.app.metadata[0].name
  }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.prod.metadata[0].name
    listeners {
      name     = "http"
      hostname = "*.example.com"
      port     = 80
      protocol = "HTTP"
      allowed_routes {
        namespaces { from = "All" }
        kinds { kind = "HTTPRoute" }
      }
    }
    listeners {
      name     = "https"
      hostname = "*.example.com"
      port     = 443
      protocol = "HTTPS"
      allowed_routes {
        namespaces { from = "All" }
        kinds { kind = "HTTPRoute" }
      }
    }
    listeners {
      name     = "grpc"
      hostname = "*.grpc.example.com"
      port     = 8443
      protocol = "GRPC"
      allowed_routes {
        namespaces { from = "All" }
        kinds { kind = "GRPCRoute" }
      }
    }
  }
}
resource "kubernetes_service_v1" "stable" {
  metadata {
    name      = "%[1]s-stable"
    namespace = kubernetes_namespace_v1.backend.metadata[0].name
  }
  spec {
    selector = { app = "stable" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}
resource "kubernetes_service_v1" "canary" {
  metadata {
    name      = "%[1]s-canary"
    namespace = kubernetes_namespace_v1.backend.metadata[0].name
  }
  spec {
    selector = { app = "canary" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}
resource "kubernetes_service_v1" "grpc" {
  metadata {
    name      = "%[1]s-grpc"
    namespace = kubernetes_namespace_v1.backend.metadata[0].name
  }
  spec {
    selector = { app = "grpc" }
    port {
      port        = 443
      target_port = 8443
    }
  }
}
resource "kubernetes_service_v1" "tls" {
  metadata {
    name      = "%[1]s-tls"
    namespace = kubernetes_namespace_v1.backend.metadata[0].name
  }
  spec {
    selector = { app = "tls" }
    port {
      port        = 443
      target_port = 8443
    }
  }
}
resource "kubernetes_http_route_v1" "prod" {
  metadata {
    name      = "%[1]s-http"
    namespace = kubernetes_namespace_v1.app.metadata[0].name
  }
  spec {
    parent_refs {
      name = kubernetes_gateway_v1.prod.metadata[0].name
    }
    rules {
      matches {
        path {
          type  = "PathPrefix"
          value = "/"
        }
      }
      backend_refs {
        name      = kubernetes_service_v1.stable.metadata[0].name
        namespace = kubernetes_namespace_v1.backend.metadata[0].name
        port      = 80
        weight    = 70
      }
      backend_refs {
        name      = kubernetes_service_v1.canary.metadata[0].name
        namespace = kubernetes_namespace_v1.backend.metadata[0].name
        port      = 80
        weight    = 30
      }
    }
  }
}
resource "kubernetes_grpc_route_v1" "prod" {
  metadata {
    name      = "%[1]s-grpc"
    namespace = kubernetes_namespace_v1.app.metadata[0].name
  }
  spec {
    parent_refs {
      name = kubernetes_gateway_v1.prod.metadata[0].name
    }
    rules {
      matches {
        method {
          service = "api.V1"
          method  = "HealthCheck"
        }
      }
      backend_refs {
        name      = kubernetes_service_v1.grpc.metadata[0].name
        namespace = kubernetes_namespace_v1.backend.metadata[0].name
        port      = 443
      }
    }
  }
}
resource "kubernetes_tls_route_v1" "prod" {
  metadata {
    name      = "%[1]s-tls"
    namespace = kubernetes_namespace_v1.app.metadata[0].name
  }
  spec {
    parent_refs {
      name = kubernetes_gateway_v1.prod.metadata[0].name
    }
    hostnames = ["tls.example.com"]
    rules {
      backend_refs {
        name      = kubernetes_service_v1.tls.metadata[0].name
        namespace = kubernetes_namespace_v1.backend.metadata[0].name
        port      = 443
      }
    }
  }
}
resource "kubernetes_backend_tls_policy_v1" "prod" {
  metadata {
    name      = "%[1]s-btls"
    namespace = kubernetes_namespace_v1.backend.metadata[0].name
  }
  spec {
    target_refs {
      group = ""
      kind  = "Service"
      name  = kubernetes_service_v1.stable.metadata[0].name
    }
    validation {
      hostname = "backend.internal.example.com"
    }
  }
}
resource "kubernetes_reference_grant_v1" "prod" {
  metadata {
    name      = "%[1]s-rg"
    namespace = kubernetes_namespace_v1.backend.metadata[0].name
  }
  spec {
    from {
      group     = "gateway.networking.k8s.io"
      kind      = "HTTPRoute"
      namespace = kubernetes_namespace_v1.app.metadata[0].name
    }
    to {
      group = ""
      kind  = "Service"
    }
  }
}
`, rName)
}
