// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import "fmt"

// ---------------------------------------------------------------------------
// Gateway field limit configs
// ---------------------------------------------------------------------------

func testAccGatewayAPIFieldLimitsGatewayManyListeners(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "many" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "many" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.many.metadata[0].name
    listeners {
      name     = "http-0"
      hostname = "a0.example.com"
      port     = 8000
      protocol = "HTTP"
    }
    listeners {
      name     = "http-1"
      hostname = "a1.example.com"
      port     = 8001
      protocol = "HTTP"
    }
    listeners {
      name     = "http-2"
      hostname = "a2.example.com"
      port     = 8002
      protocol = "HTTP"
    }
    listeners {
      name     = "http-3"
      hostname = "a3.example.com"
      port     = 8003
      protocol = "HTTP"
    }
    listeners {
      name     = "http-4"
      hostname = "a4.example.com"
      port     = 8004
      protocol = "HTTP"
    }
    listeners {
      name     = "http-5"
      hostname = "a5.example.com"
      port     = 8005
      protocol = "HTTP"
    }
    listeners {
      name     = "http-6"
      hostname = "a6.example.com"
      port     = 8006
      protocol = "HTTP"
    }
    listeners {
      name     = "http-7"
      hostname = "a7.example.com"
      port     = 8007
      protocol = "HTTP"
    }
  }
}
`, rName)
}

func testAccGatewayAPIFieldLimitsGatewayHostnameLength(rName string) string {
	longHostname := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.com"
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "hn" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "hostname" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.hn.metadata[0].name
    listeners {
      name     = "http"
      hostname = "%[2]s"
      port     = 80
      protocol = "HTTP"
    }
  }
}
`, rName, longHostname)
}

func testAccGatewayAPIFieldLimitsGatewayProtocols(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "proto" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "protocols" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.proto.metadata[0].name
    listeners {
      name     = "http"
      hostname = "http.example.com"
      port     = 80
      protocol = "HTTP"
    }
    listeners {
      name     = "https"
      hostname = "https.example.com"
      port     = 443
      protocol = "HTTPS"
    }
    listeners {
      name     = "tcp"
      port     = 9000
      protocol = "TCP"
    }
    listeners {
      name     = "tls"
      hostname = "tls.example.com"
      port     = 8443
      protocol = "TLS"
      tls {
        mode = "Passthrough"
      }
    }
    listeners {
      name     = "grpc"
      hostname = "grpc.example.com"
      port     = 8444
      protocol = "GRPC"
    }
  }
}
`, rName)
}

func testAccGatewayAPIFieldLimitsGatewayTLSMode(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "tls" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "tls_mode" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.tls.metadata[0].name
    listeners {
      name     = "passthrough"
      hostname = "pt.example.com"
      port     = 8443
      protocol = "TLS"
      tls {
        mode = "Passthrough"
      }
    }
  }
}
`, rName)
}

func testAccGatewayAPIFieldLimitsGatewayAllowedRoutesNS(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "ns" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "allow_all" {
  metadata { name = "%[1]s-allow-all" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.ns.metadata[0].name
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
resource "kubernetes_gateway_v1" "allow_same" {
  metadata { name = "%[1]s-allow-same" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.ns.metadata[0].name
    listeners {
      name     = "http"
      hostname = "*.same.example.com"
      port     = 80
      protocol = "HTTP"
      allowed_routes {
        namespaces { from = "Same" }
        kinds { kind = "HTTPRoute" }
      }
    }
  }
}
`, rName)
}

func testAccGatewayAPIFieldLimitsGatewayAddresses(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "addr" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "addresses" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.addr.metadata[0].name
    listeners {
      name     = "http"
      hostname = "*.example.com"
      port     = 80
      protocol = "HTTP"
    }
    addresses {
      type  = "IPAddress"
      value = "10.0.0.1"
    }
    addresses {
      type  = "Hostname"
      value = "gw.example.com"
    }
  }
}
`, rName)
}

// ---------------------------------------------------------------------------
// HTTPRoute field limit configs
// ---------------------------------------------------------------------------

func testAccGatewayAPIFieldLimitsHTTPRouteManyRules(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "mr" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "mr" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.mr.metadata[0].name
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
resource "kubernetes_service_v1" "mr" {
  metadata { name = "%[1]s-svc" }
  spec {
    selector = { app = "test" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}
resource "kubernetes_http_route_v1" "many_rules" {
  metadata { name = "%[1]s-route" }
  spec {
    parent_refs { name = kubernetes_gateway_v1.mr.metadata[0].name }
    rules {
      name = "rule-0"
      matches {
        path {
          type  = "PathPrefix"
          value = "/r0"
        }
      }
      backend_refs {
        name = kubernetes_service_v1.mr.metadata[0].name
        port = 80
      }
    }
    rules {
      name = "rule-1"
      matches {
        path {
          type  = "PathPrefix"
          value = "/r1"
        }
      }
      backend_refs {
        name = kubernetes_service_v1.mr.metadata[0].name
        port = 80
      }
    }
    rules {
      name = "rule-2"
      matches {
        path {
          type  = "PathPrefix"
          value = "/r2"
        }
      }
      backend_refs {
        name = kubernetes_service_v1.mr.metadata[0].name
        port = 80
      }
    }
    rules {
      name = "rule-3"
      matches {
        path {
          type  = "PathPrefix"
          value = "/r3"
        }
      }
      backend_refs {
        name = kubernetes_service_v1.mr.metadata[0].name
        port = 80
      }
    }
    rules {
      name = "rule-4"
      matches {
        path {
          type  = "PathPrefix"
          value = "/r4"
        }
      }
      backend_refs {
        name = kubernetes_service_v1.mr.metadata[0].name
        port = 80
      }
    }
    rules {
      name = "rule-5"
      matches {
        path {
          type  = "PathPrefix"
          value = "/r5"
        }
      }
      backend_refs {
        name = kubernetes_service_v1.mr.metadata[0].name
        port = 80
      }
    }
    rules {
      name = "rule-6"
      matches {
        path {
          type  = "PathPrefix"
          value = "/r6"
        }
      }
      backend_refs {
        name = kubernetes_service_v1.mr.metadata[0].name
        port = 80
      }
    }
    rules {
      name = "rule-7"
      matches {
        path {
          type  = "PathPrefix"
          value = "/r7"
        }
      }
      backend_refs {
        name = kubernetes_service_v1.mr.metadata[0].name
        port = 80
      }
    }
  }
}
`, rName)
}

func testAccGatewayAPIFieldLimitsHTTPRouteWeightsEdge(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "we" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "we" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.we.metadata[0].name
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
resource "kubernetes_service_v1" "a" {
  metadata { name = "%[1]s-svc-a" }
  spec {
    selector = { app = "a" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}
resource "kubernetes_service_v1" "b" {
  metadata { name = "%[1]s-svc-b" }
  spec {
    selector = { app = "b" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}
resource "kubernetes_http_route_v1" "weights" {
  metadata { name = "%[1]s-weights" }
  spec {
    parent_refs { name = kubernetes_gateway_v1.we.metadata[0].name }
    rules {
      matches {
        path {
          type  = "PathPrefix"
          value = "/"
        }
      }
      backend_refs {
        name   = kubernetes_service_v1.a.metadata[0].name
        port   = 80
        weight = 1
      }
      backend_refs {
        name   = kubernetes_service_v1.b.metadata[0].name
        port   = 80
        weight = 999999
      }
    }
  }
}
`, rName)
}

func testAccGatewayAPIFieldLimitsHTTPRoutePathTypes(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "pt" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "pt" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.pt.metadata[0].name
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
resource "kubernetes_service_v1" "pt" {
  metadata { name = "%[1]s-svc" }
  spec {
    selector = { app = "test" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}
resource "kubernetes_http_route_v1" "paths" {
  metadata { name = "%[1]s-paths" }
  spec {
    parent_refs { name = kubernetes_gateway_v1.pt.metadata[0].name }
    rules {
      name = "prefix"
      matches {
        path {
          type  = "PathPrefix"
          value = "/api"
        }
      }
      backend_refs {
        name = kubernetes_service_v1.pt.metadata[0].name
        port = 80
      }
    }
    rules {
      name = "exact"
      matches {
        path {
          type  = "Exact"
          value = "/health"
        }
      }
      backend_refs {
        name = kubernetes_service_v1.pt.metadata[0].name
        port = 80
      }
    }
    rules {
      name = "regex"
      matches {
        path {
          type  = "RegularExpression"
          value = "/api/[^/]+/v[0-9]+"
        }
      }
      backend_refs {
        name = kubernetes_service_v1.pt.metadata[0].name
        port = 80
      }
    }
  }
}
`, rName)
}

func testAccGatewayAPIFieldLimitsHTTPRouteMatchTypes(rName string) string {
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
resource "kubernetes_http_route_v1" "match_types" {
  metadata { name = "%[1]s-mt" }
  spec {
    parent_refs { name = kubernetes_gateway_v1.mt.metadata[0].name }
    rules {
      name = "header-exact"
      matches {
        path {
          type  = "PathPrefix"
          value = "/"
        }
        headers {
          name  = "X-Exact"
          value = "test"
          type  = "Exact"
        }
      }
      backend_refs {
        name = kubernetes_service_v1.mt.metadata[0].name
        port = 80
      }
    }
    rules {
      name = "header-regex"
      matches {
        path {
          type  = "PathPrefix"
          value = "/"
        }
        headers {
          name  = "X-Regex"
          value = "t.*"
          type  = "RegularExpression"
        }
      }
      backend_refs {
        name = kubernetes_service_v1.mt.metadata[0].name
        port = 80
      }
    }
    rules {
      name = "qp-exact"
      matches {
        path {
          type  = "PathPrefix"
          value = "/"
        }
        query_params {
          name  = "foo"
          value = "bar"
          type  = "Exact"
        }
      }
      backend_refs {
        name = kubernetes_service_v1.mt.metadata[0].name
        port = 80
      }
    }
    rules {
      name = "qp-regex"
      matches {
        path {
          type  = "PathPrefix"
          value = "/"
        }
        query_params {
          name  = "id"
          value = "[0-9]+"
          type  = "RegularExpression"
        }
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

// ---------------------------------------------------------------------------
// GRPCRoute field limit configs
// ---------------------------------------------------------------------------

func testAccGatewayAPIFieldLimitsGRPCRouteMethodTypes(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "gm" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "gm" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.gm.metadata[0].name
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
resource "kubernetes_service_v1" "gm" {
  metadata { name = "%[1]s-svc" }
  spec {
    selector = { app = "grpc" }
    port {
      port        = 443
      target_port = 8443
    }
  }
}
resource "kubernetes_grpc_route_v1" "methods" {
  metadata { name = "%[1]s-methods" }
  spec {
    parent_refs { name = kubernetes_gateway_v1.gm.metadata[0].name }
    rules {
      matches {
        method {
          type    = "Exact"
          service = "example.Service"
          method  = "Method"
        }
      }
      backend_refs {
        name = kubernetes_service_v1.gm.metadata[0].name
        port = 443
      }
    }
    rules {
      matches {
        method {
          type    = "RegularExpression"
          service = "example.Service"
          method  = "*"
        }
      }
      backend_refs {
        name = kubernetes_service_v1.gm.metadata[0].name
        port = 443
      }
    }
  }
}
`, rName)
}

// ---------------------------------------------------------------------------
// TLSRoute field limit configs
// ---------------------------------------------------------------------------

func testAccGatewayAPIFieldLimitsTLSRouteBackends(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "tb" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "tb" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.tb.metadata[0].name
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
resource "kubernetes_service_v1" "a" {
  metadata { name = "%[1]s-svc-a" }
  spec {
    selector = { app = "a" }
    port {
      port        = 443
      target_port = 8443
    }
  }
}
resource "kubernetes_service_v1" "b" {
  metadata { name = "%[1]s-svc-b" }
  spec {
    selector = { app = "b" }
    port {
      port        = 443
      target_port = 8443
    }
  }
}
resource "kubernetes_service_v1" "c" {
  metadata { name = "%[1]s-svc-c" }
  spec {
    selector = { app = "c" }
    port {
      port        = 443
      target_port = 8443
    }
  }
}
resource "kubernetes_tls_route_v1" "backends" {
  metadata { name = "%[1]s-tls" }
  spec {
    parent_refs { name = kubernetes_gateway_v1.tb.metadata[0].name }
    hostnames = ["tls.example.com"]
    rules {
      backend_refs {
        name   = kubernetes_service_v1.a.metadata[0].name
        port   = 443
        weight = 50
      }
      backend_refs {
        name   = kubernetes_service_v1.b.metadata[0].name
        port   = 443
        weight = 30
      }
      backend_refs {
        name   = kubernetes_service_v1.c.metadata[0].name
        port   = 443
        weight = 20
      }
    }
  }
}
`, rName)
}

// ---------------------------------------------------------------------------
// ListenerSet field limit configs
// ---------------------------------------------------------------------------

func testAccGatewayAPIFieldLimitsListenerSetManyListeners(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "lm" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "lm" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.lm.metadata[0].name
    listeners {
      name     = "placeholder"
      hostname = "*.example.com"
      port     = 80
      protocol = "HTTP"
    }
  }
}
resource "kubernetes_listener_set_v1" "many" {
  metadata { name = "%[1]s-ls" }
  spec {
    parent_ref {
      name = kubernetes_gateway_v1.lm.metadata[0].name
    }
    listeners {
      name     = "http-0"
      port     = 8080
      protocol = "HTTP"
      hostname = "a0.example.com"
      allowed_routes {
        namespaces { from = "All" }
        kinds { kind = "HTTPRoute" }
      }
    }
    listeners {
      name     = "http-1"
      port     = 8081
      protocol = "HTTP"
      hostname = "a1.example.com"
      allowed_routes {
        namespaces { from = "All" }
        kinds { kind = "HTTPRoute" }
      }
    }
    listeners {
      name     = "http-2"
      port     = 8082
      protocol = "HTTP"
      hostname = "a2.example.com"
      allowed_routes {
        namespaces { from = "All" }
        kinds { kind = "HTTPRoute" }
      }
    }
    listeners {
      name     = "http-3"
      port     = 8083
      protocol = "HTTP"
      hostname = "a3.example.com"
      allowed_routes {
        namespaces { from = "All" }
        kinds { kind = "HTTPRoute" }
      }
    }
    listeners {
      name     = "http-4"
      port     = 8084
      protocol = "HTTP"
      hostname = "a4.example.com"
      allowed_routes {
        namespaces { from = "All" }
        kinds { kind = "HTTPRoute" }
      }
    }
    listeners {
      name     = "http-5"
      port     = 8085
      protocol = "HTTP"
      hostname = "a5.example.com"
      allowed_routes {
        namespaces { from = "All" }
        kinds { kind = "HTTPRoute" }
      }
    }
  }
}
`, rName)
}

func testAccGatewayAPIFieldLimitsListenerSetProtocols(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "lp" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "lp" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.lp.metadata[0].name
    listeners {
      name     = "placeholder"
      hostname = "*.example.com"
      port     = 80
      protocol = "HTTP"
    }
  }
}
resource "kubernetes_listener_set_v1" "protocols" {
  metadata { name = "%[1]s-ls" }
  spec {
    parent_ref {
      name = kubernetes_gateway_v1.lp.metadata[0].name
    }
    listeners {
      name     = "http"
      port     = 80
      protocol = "HTTP"
      hostname = "http.example.com"
      allowed_routes {
        namespaces { from = "All" }
        kinds { kind = "HTTPRoute" }
      }
    }
    listeners {
      name     = "https"
      port     = 443
      protocol = "HTTPS"
      hostname = "https.example.com"
      allowed_routes {
        namespaces { from = "All" }
        kinds { kind = "HTTPRoute" }
      }
    }
    listeners {
      name     = "tcp"
      port     = 9000
      protocol = "TCP"
    }
    listeners {
      name     = "tls"
      port     = 8443
      protocol = "TLS"
      hostname = "tls.example.com"
      tls {
        mode = "Passthrough"
      }
    }
    listeners {
      name     = "grpc"
      port     = 8444
      protocol = "GRPC"
      hostname = "grpc.example.com"
      allowed_routes {
        namespaces { from = "All" }
        kinds { kind = "GRPCRoute" }
      }
    }
  }
}
`, rName)
}

// ---------------------------------------------------------------------------
// BackendTLSPolicy field limit configs
// ---------------------------------------------------------------------------

func testAccGatewayAPIFieldLimitsBackendTLSSAN(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_service_v1" "san" {
  metadata { name = "%[1]s-svc" }
  spec {
    selector = { app = "backend" }
    port {
      port        = 443
      target_port = 8443
    }
  }
}
resource "kubernetes_backend_tls_policy_v1" "san" {
  metadata { name = "%[1]s-btls" }
  spec {
    target_refs {
      group = ""
      kind  = "Service"
      name  = kubernetes_service_v1.san.metadata[0].name
    }
    validation {
      hostname                   = "backend.example.com"
      well_known_ca_certificates = "System"
      subject_alt_names {
        type     = "Hostname"
        hostname = "*.backend.example.com"
      }
      subject_alt_names {
        type = "URI"
        uri  = "spiffe://backend.example.com/svc"
      }
    }
  }
}
`, rName)
}

// ---------------------------------------------------------------------------
// ReferenceGrant field limit configs
// ---------------------------------------------------------------------------

func testAccGatewayAPIFieldLimitsReferenceGrantMultiple(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_reference_grant_v1" "multi" {
  metadata { name = "%[1]s-rg" }
  spec {
    from {
      group     = "gateway.networking.k8s.io"
      kind      = "HTTPRoute"
      namespace = "%[1]s-ns-a"
    }
    from {
      group     = "gateway.networking.k8s.io"
      kind      = "GRPCRoute"
      namespace = "%[1]s-ns-b"
    }
    from {
      group     = "gateway.networking.k8s.io"
      kind      = "TLSRoute"
      namespace = "%[1]s-ns-c"
    }
    to {
      group = ""
      kind  = "Service"
    }
    to {
      group = ""
      kind  = "Service"
      name  = "%[1]s-svc-a"
    }
    to {
      group = ""
      kind  = "Service"
      name  = "%[1]s-svc-b"
    }
  }
}
`, rName)
}

// ---------------------------------------------------------------------------
// Update in-place configs
// ---------------------------------------------------------------------------

func testAccGatewayAPIFieldLimitsHTTPRouteUpdateBefore(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "up" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "up" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.up.metadata[0].name
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
resource "kubernetes_service_v1" "a" {
  metadata { name = "%[1]s-svc-a" }
  spec {
    selector = { app = "a" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}
resource "kubernetes_service_v1" "b" {
  metadata { name = "%[1]s-svc-b" }
  spec {
    selector = { app = "b" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}
resource "kubernetes_http_route_v1" "update" {
  metadata { name = "%[1]s-route" }
  spec {
    parent_refs { name = kubernetes_gateway_v1.up.metadata[0].name }
    rules {
      matches {
        path {
          type  = "PathPrefix"
          value = "/api"
        }
      }
      backend_refs {
        name = kubernetes_service_v1.a.metadata[0].name
        port = 80
      }
    }
  }
}
`, rName)
}

func testAccGatewayAPIFieldLimitsHTTPRouteUpdateAfter(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "up" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "up" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.up.metadata[0].name
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
resource "kubernetes_service_v1" "a" {
  metadata { name = "%[1]s-svc-a" }
  spec {
    selector = { app = "a" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}
resource "kubernetes_service_v1" "b" {
  metadata { name = "%[1]s-svc-b" }
  spec {
    selector = { app = "b" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}
resource "kubernetes_http_route_v1" "update" {
  metadata { name = "%[1]s-route" }
  spec {
    parent_refs { name = kubernetes_gateway_v1.up.metadata[0].name }
    rules {
      matches {
        path {
          type  = "PathPrefix"
          value = "/api/v2"
        }
      }
      filters {
        type = "RequestHeaderModifier"
        request_header_modifier {
          set {
            name  = "X-Updated"
            value = "true"
          }
        }
      }
      backend_refs {
        name = kubernetes_service_v1.b.metadata[0].name
        port = 80
      }
    }
  }
}
`, rName)
}

func testAccGatewayAPIFieldLimitsGatewayClassUpdateBefore(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "update" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
`, rName)
}

func testAccGatewayAPIFieldLimitsGatewayClassUpdateAfter(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "update" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
    description     = "Updated by test"
  }
}
`, rName)
}

func testAccGatewayAPIFieldLimitsGatewayImport(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "imp" {
  metadata { name = "%[1]s-gc" }
  spec {
    controller_name = "gateway.envoyproxy.io/gatewayclass-controller"
  }
}
resource "kubernetes_gateway_v1" "import_test" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.imp.metadata[0].name
    listeners {
      name     = "http"
      hostname = "*.example.com"
      port     = 80
      protocol = "HTTP"
    }
  }
}
`, rName)
}

func testAccGatewayAPIFieldLimitsBackendTLSUpdateBefore(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_service_v1" "up" {
  metadata { name = "%[1]s-svc" }
  spec {
    selector = { app = "backend" }
    port {
      port        = 443
      target_port = 8443
    }
  }
}
resource "kubernetes_backend_tls_policy_v1" "update" {
  metadata { name = "%[1]s-btls" }
  spec {
    target_refs {
      group = ""
      kind  = "Service"
      name  = kubernetes_service_v1.up.metadata[0].name
    }
    validation {
      hostname                   = "old.example.com"
      well_known_ca_certificates = "System"
    }
  }
}
`, rName)
}

func testAccGatewayAPIFieldLimitsBackendTLSUpdateAfter(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_service_v1" "up" {
  metadata { name = "%[1]s-svc" }
  spec {
    selector = { app = "backend" }
    port {
      port        = 443
      target_port = 8443
    }
  }
}
resource "kubernetes_backend_tls_policy_v1" "update" {
  metadata { name = "%[1]s-btls" }
  spec {
    target_refs {
      group = ""
      kind  = "Service"
      name  = kubernetes_service_v1.up.metadata[0].name
    }
    validation {
      hostname                   = "new.example.com"
      well_known_ca_certificates = "System"
    }
    options = {
      min_version = "VersionTLS12"
    }
  }
}
`, rName)
}
