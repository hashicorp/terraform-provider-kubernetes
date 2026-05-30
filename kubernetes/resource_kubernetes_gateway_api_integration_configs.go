// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import "fmt"

func testAccGatewayAPIIntegrationFullStack(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_namespace_v1" "route" {
  metadata { name = "%[1]s-route" }
}
resource "kubernetes_namespace_v1" "backend" {
  metadata { name = "%[1]s-backend" }
}
resource "kubernetes_service_v1" "backend" {
  metadata {
    name      = "%[1]s-backend-svc"
    namespace = kubernetes_namespace_v1.backend.metadata[0].name
  }
  spec {
    selector = { app = "backend" }
    port {
      name        = "http"
      port        = 80
      target_port = 8080
    }
    port {
      name        = "https"
      port        = 443
      target_port = 8443
    }
  }
}
resource "kubernetes_gateway_class_v1" "int" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "int" {
  metadata {
    name      = "%[1]s-gw"
    namespace = kubernetes_namespace_v1.route.metadata[0].name
  }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.int.metadata[0].name
    listeners {
      name     = "http"
      hostname = "*.example.com"
      port     = 80
      protocol = "HTTP"
      allowed_routes {
        kinds { kind = "HTTPRoute" }
      }
    }
    listeners {
      name     = "grpc"
      hostname = "*.grpc.example.com"
      port     = 8443
      protocol = "GRPC"
      allowed_routes {
        kinds { kind = "GRPCRoute" }
      }
    }
  }
}
resource "kubernetes_http_route_v1" "int" {
  metadata {
    name      = "%[1]s-route"
    namespace = kubernetes_namespace_v1.route.metadata[0].name
  }
  spec {
    parent_refs {
      name = kubernetes_gateway_v1.int.metadata[0].name
    }
    hostnames = ["app.example.com"]
    rules {
      matches {
        path {
          type  = "PathPrefix"
          value = "/api"
        }
      }
      backend_refs {
        name      = kubernetes_service_v1.backend.metadata[0].name
        namespace = kubernetes_namespace_v1.backend.metadata[0].name
        port      = 80
      }
    }
  }
}
resource "kubernetes_reference_grant_v1" "int" {
  metadata {
    name      = "%[1]s-rg"
    namespace = kubernetes_namespace_v1.backend.metadata[0].name
  }
  spec {
    from {
      group     = "gateway.networking.k8s.io"
      kind      = "HTTPRoute"
      namespace = kubernetes_namespace_v1.route.metadata[0].name
    }
    to {
      group = ""
      kind  = "Service"
    }
  }
}
resource "kubernetes_backend_tls_policy_v1" "int" {
  metadata {
    name      = "%[1]s-btls"
    namespace = kubernetes_namespace_v1.backend.metadata[0].name
  }
  spec {
    target_refs {
      group = ""
      kind  = "Service"
      name  = kubernetes_service_v1.backend.metadata[0].name
    }
    validation {
      hostname                   = "backend.example.com"
      well_known_ca_certificates = "System"
    }
    options = {
      min_version = "VersionTLS12"
      max_version = "VersionTLS13"
    }
  }
}
`, rName)
}

func testAccGatewayAPIIntegrationGatewayFullOptions(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "full" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "full" {
  metadata { name = "%[1]s" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.full.metadata[0].name
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
    addresses {
      type  = "IPAddress"
      value = "10.0.0.1"
    }
    infrastructure {
      labels      = { foo = "bar" }
      annotations = { "test-key" = "test-value" }
    }
  }
}
`, rName)
}

func testAccGatewayAPIIntegrationGatewayBefore(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "update" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "update_test" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.update.metadata[0].name
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
`, rName)
}

func testAccGatewayAPIIntegrationGatewayAfter(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "update" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "update_test" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.update.metadata[0].name
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
    addresses {
      type  = "IPAddress"
      value = "10.0.0.2"
    }
    infrastructure {
      labels = { env = "test", updated = "true" }
    }
  }
}
`, rName)
}

func testAccGatewayAPIIntegrationHTTPRouteMatchTypes(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "mt" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "mt" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.mt.metadata[0].name
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
resource "kubernetes_service_v1" "mt" {
  metadata { name = "%[1]s-svc" }
  spec {
    selector = { app = "test" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}
resource "kubernetes_http_route_v1" "matches" {
  metadata { name = "%[1]s-matches" }
  spec {
    parent_refs { name = kubernetes_gateway_v1.mt.metadata[0].name }
    hostnames = ["app.example.com"]
    rules {
      name = "path-prefix"
      matches {
        path {
          type  = "PathPrefix"
          value = "/api"
        }
      }
      backend_refs {
        name = kubernetes_service_v1.mt.metadata[0].name
        port = 80
      }
    }
    rules {
      name = "exact-path"
      matches {
        path {
          type  = "Exact"
          value = "/health"
        }
      }
      backend_refs {
        name = kubernetes_service_v1.mt.metadata[0].name
        port = 80
      }
    }
    rules {
      name = "header-match"
      matches {
        path {
          type  = "PathPrefix"
          value = "/headers"
        }
        headers {
          name  = "X-Custom"
          value = "test-value"
        }
      }
      backend_refs {
        name = kubernetes_service_v1.mt.metadata[0].name
        port = 80
      }
    }
    rules {
      name = "query-param-match"
      matches {
        path {
          type  = "PathPrefix"
          value = "/search"
        }
        query_params {
          name  = "foo"
          value = "bar"
        }
      }
      backend_refs {
        name = kubernetes_service_v1.mt.metadata[0].name
        port = 80
      }
    }
    rules {
      name = "method-match"
      matches {
        path {
          type  = "PathPrefix"
          value = "/submit"
        }
        method = "POST"
      }
      backend_refs {
        name = kubernetes_service_v1.mt.metadata[0].name
        port = 80
      }
    }
  }
}
`, rName)
}

func testAccGatewayAPIIntegrationHTTPRouteFilters(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "fl" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "fl" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.fl.metadata[0].name
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
resource "kubernetes_service_v1" "fl" {
  metadata { name = "%[1]s-svc" }
  spec {
    selector = { app = "test" }
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
resource "kubernetes_http_route_v1" "filters" {
  metadata { name = "%[1]s-filters" }
  spec {
    parent_refs { name = kubernetes_gateway_v1.fl.metadata[0].name }
    rules {
      name = "header-modifier"
      matches {
        path {
          type  = "PathPrefix"
          value = "/api"
        }
      }
      filters {
        type = "RequestHeaderModifier"
        request_header_modifier {
          add {
            name  = "X-Added-Header"
            value = "added-value"
          }
          set {
            name  = "X-Set-Header"
            value = "set-value"
          }
          remove = ["X-Remove-Header"]
        }
      }
      backend_refs {
        name = kubernetes_service_v1.fl.metadata[0].name
        port = 80
      }
    }
    rules {
      name = "redirect"
      matches {
        path {
          type  = "PathPrefix"
          value = "/old"
        }
      }
      filters {
        type = "RequestRedirect"
        request_redirect {
          hostname = "new.example.com"
          port     = 443
          scheme   = "https"
        }
      }
    }
    rules {
      name = "url-rewrite"
      matches {
        path {
          type  = "PathPrefix"
          value = "/api/v1"
        }
      }
      filters {
        type = "URLRewrite"
        url_rewrite {
          path {
            type               = "ReplacePrefixMatch"
            replace_prefix_match = "/v1"
          }
        }
      }
      backend_refs {
        name = kubernetes_service_v1.fl.metadata[0].name
        port = 80
      }
    }
    rules {
      name = "mirror"
      matches {
        path {
          type  = "PathPrefix"
          value = "/mirror"
        }
      }
      filters {
        type = "RequestMirror"
        request_mirror {
          percent = 50
          backend_ref {
            name = kubernetes_service_v1.mirror.metadata[0].name
            port = 80
          }
        }
      }
      backend_refs {
        name = kubernetes_service_v1.fl.metadata[0].name
        port = 80
      }
    }
    rules {
      name = "cors"
      matches {
        path {
          type  = "PathPrefix"
          value = "/cors"
        }
      }
      filters {
        type = "CORS"
        cors {
          allow_origins = ["https://example.com"]
          allow_methods = ["GET", "POST", "OPTIONS"]
          allow_headers = ["Authorization", "Content-Type"]
          max_age       = 3600
        }
      }
      backend_refs {
        name = kubernetes_service_v1.fl.metadata[0].name
        port = 80
      }
    }
  }
}
`, rName)
}

func testAccGatewayAPIIntegrationBackendWeights(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "w" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "w" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.w.metadata[0].name
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
resource "kubernetes_service_v1" "primary" {
  metadata { name = "%[1]s-primary" }
  spec {
    selector = { app = "primary" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}
resource "kubernetes_service_v1" "secondary" {
  metadata { name = "%[1]s-secondary" }
  spec {
    selector = { app = "secondary" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}
resource "kubernetes_http_route_v1" "weights" {
  metadata { name = "%[1]s-weights" }
  spec {
    parent_refs { name = kubernetes_gateway_v1.w.metadata[0].name }
    rules {
      matches {
        path {
          type  = "PathPrefix"
          value = "/"
        }
      }
      backend_refs {
        name   = kubernetes_service_v1.primary.metadata[0].name
        port   = 80
        weight = 80
      }
      backend_refs {
        name   = kubernetes_service_v1.secondary.metadata[0].name
        port   = 80
        weight = 20
      }
    }
  }
}
`, rName)
}

func testAccGatewayAPIIntegrationHTTPRouteAdvanced(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "adv" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "adv" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.adv.metadata[0].name
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
resource "kubernetes_service_v1" "adv" {
  metadata { name = "%[1]s-svc" }
  spec {
    selector = { app = "test" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}
resource "kubernetes_http_route_v1" "advanced" {
  metadata { name = "%[1]s-advanced" }
  spec {
    parent_refs { name = kubernetes_gateway_v1.adv.metadata[0].name }
    rules {
      matches {
        path {
          type  = "PathPrefix"
          value = "/"
        }
      }
      timeouts {
        request         = "30s"
        backend_request = "10s"
      }
      session_persistence {
        type             = "Cookie"
        absolute_timeout = "300s"
      }
      backend_refs {
        name = kubernetes_service_v1.adv.metadata[0].name
        port = 80
      }
    }
  }
}
`, rName)
}

func testAccGatewayAPIIntegrationGRPCRoute(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "grpc" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "grpc" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.grpc.metadata[0].name
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
resource "kubernetes_service_v1" "grpc" {
  metadata { name = "%[1]s-svc" }
  spec {
    selector = { app = "grpc" }
    port {
      port        = 443
      target_port = 8443
    }
  }
}
resource "kubernetes_grpc_route_v1" "int" {
  metadata { name = "%[1]s-grpc" }
  spec {
    parent_refs { name = kubernetes_gateway_v1.grpc.metadata[0].name }
    hostnames = ["grpc.example.com"]
    rules {
      matches {
        method {
          service = "example.Service"
          method  = "Method"
        }
      }
      backend_refs {
        name = kubernetes_service_v1.grpc.metadata[0].name
        port = 443
      }
    }
  }
}
`, rName)
}

func testAccGatewayAPIIntegrationTLSRoute(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "tls" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "tls" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.tls.metadata[0].name
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
resource "kubernetes_service_v1" "tls" {
  metadata { name = "%[1]s-svc" }
  spec {
    selector = { app = "tls" }
    port {
      port        = 443
      target_port = 8443
    }
  }
}
resource "kubernetes_tls_route_v1" "int" {
  metadata { name = "%[1]s-tls" }
  spec {
    parent_refs { name = kubernetes_gateway_v1.tls.metadata[0].name }
    hostnames = ["tls.example.com"]
    rules {
      backend_refs {
        name = kubernetes_service_v1.tls.metadata[0].name
        port = 443
      }
    }
  }
}
`, rName)
}

func testAccGatewayAPIIntegrationListenerSet(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "ls" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "ls" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.ls.metadata[0].name
    listeners {
      name     = "placeholder"
      hostname = "*.example.com"
      port     = 80
      protocol = "HTTP"
    }
  }
}
resource "kubernetes_listener_set_v1" "int" {
  metadata { name = "%[1]s-ls" }
  spec {
    parent_ref {
      name = kubernetes_gateway_v1.ls.metadata[0].name
    }
    listeners {
      name     = "http"
      port     = 80
      protocol = "HTTP"
      hostname = "*.http.example.com"
      allowed_routes {
        namespaces { from = "All" }
        kinds { kind = "HTTPRoute" }
      }
    }
    listeners {
      name     = "tls"
      port     = 443
      protocol = "TLS"
      hostname = "*.tls.example.com"
      tls {
        mode = "Passthrough"
      }
    }
  }
}
`, rName)
}

func testAccGatewayAPIIntegrationCrossNamespace(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_namespace_v1" "route" {
  metadata { name = "%[1]s-route" }
}
resource "kubernetes_namespace_v1" "backend" {
  metadata { name = "%[1]s-backend" }
}
resource "kubernetes_gateway_class_v1" "cn" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "cn" {
  metadata {
    name      = "%[1]s-gw"
    namespace = kubernetes_namespace_v1.route.metadata[0].name
  }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.cn.metadata[0].name
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
resource "kubernetes_service_v1" "cn" {
  metadata {
    name      = "%[1]s-svc"
    namespace = kubernetes_namespace_v1.backend.metadata[0].name
  }
  spec {
    selector = { app = "backend" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}
resource "kubernetes_http_route_v1" "cross_ns" {
  metadata {
    name      = "%[1]s-route"
    namespace = kubernetes_namespace_v1.route.metadata[0].name
  }
  spec {
    parent_refs {
      name = kubernetes_gateway_v1.cn.metadata[0].name
    }
    rules {
      matches {
        path {
          type  = "PathPrefix"
          value = "/api"
        }
      }
      backend_refs {
        name      = kubernetes_service_v1.cn.metadata[0].name
        namespace = kubernetes_namespace_v1.backend.metadata[0].name
        port      = 80
      }
    }
  }
}
resource "kubernetes_reference_grant_v1" "cross_ns" {
  metadata {
    name      = "%[1]s-rg"
    namespace = kubernetes_namespace_v1.backend.metadata[0].name
  }
  spec {
    from {
      group     = "gateway.networking.k8s.io"
      kind      = "HTTPRoute"
      namespace = kubernetes_namespace_v1.route.metadata[0].name
    }
    to {
      group = ""
      kind  = "Service"
      name  = kubernetes_service_v1.cn.metadata[0].name
    }
  }
}
`, rName)
}

func testAccGatewayAPIIntegrationBackendTLS(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_service_v1" "btls" {
  metadata { name = "%[1]s-svc" }
  spec {
    selector = { app = "backend" }
    port {
      port        = 443
      target_port = 8443
    }
  }
}
resource "kubernetes_backend_tls_policy_v1" "int" {
  metadata { name = "%[1]s-btls" }
  spec {
    target_refs {
      group = ""
      kind  = "Service"
      name  = kubernetes_service_v1.btls.metadata[0].name
    }
    validation {
      hostname                   = "backend.example.com"
      well_known_ca_certificates = "System"
      subject_alt_names {
        type     = "Hostname"
        hostname = "*.backend.example.com"
      }
    }
    options = {
      min_version = "VersionTLS12"
      max_version = "VersionTLS13"
    }
  }
}
`, rName)
}

func testAccGatewayAPIIntegrationGatewayTLSListener(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "tls" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "tls" {
  metadata { name = "%[1]s" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.tls.metadata[0].name
    listeners {
      name     = "https"
      hostname = "*.example.com"
      port     = 443
      protocol = "HTTPS"
      tls {
        mode = "Terminate"
        certificate_refs {
          kind = "Secret"
          name = "test-tls-cert"
        }
        options = {}
      }
      allowed_routes {
        namespaces { from = "All" }
        kinds { kind = "HTTPRoute" }
      }
    }
  }
}
`, rName)
}
