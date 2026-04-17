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

func TestAccKubernetesHTTPRouteV1_apiGatewayPattern(t *testing.T) {
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
				Config: testAccHTTPRouteV1APIGatewayPatternConfig(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckHTTPRouteV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.#", "3"),
					// Rule 0: HTTP → HTTPS redirect
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.filters.0.type", "RequestRedirect"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.filters.0.request_redirect.0.scheme", "https"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.filters.0.request_redirect.0.status_code", "301"),
					// Rule 1: /api prefix → backend with retry + timeout + header modifier
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.1.matches.0.path.0.value", "/api"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.1.timeouts.0.request", "30s"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.1.timeouts.0.backend_request", "10s"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.1.retry.0.codes.#", "4"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.1.retry.0.attempts", "3"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.1.retry.0.backoff", "500ms"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.1.filters.0.type", "RequestHeaderModifier"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.1.filters.0.request_header_modifier.0.add.0.name", "X-Forwarded-Proto"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.1.filters.0.request_header_modifier.0.add.0.value", "https"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.1.filters.0.request_header_modifier.0.remove.0", "X-Internal-Debug"),
					// Rule 2: /static prefix → URL rewrite + response header
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.2.filters.0.type", "URLRewrite"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.2.filters.0.url_rewrite.0.path.0.type", "ReplacePrefixMatch"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.2.filters.0.url_rewrite.0.path.0.replace_prefix_match", "/assets"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.2.filters.1.type", "ResponseHeaderModifier"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.2.filters.1.response_header_modifier.0.set.0.name", "Cache-Control"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.2.filters.1.response_header_modifier.0.set.0.value", "public, max-age=31536000"),
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

func smGatewayStack(rName, gcName, protocol string, extraServices ...string) string {
	extra := ""
	for _, s := range extraServices {
		extra += s
	}
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

resource "kubernetes_service_v1" "backend" {
  metadata {
    name = %[1]q
  }
  spec {
    selector = { app = "backend" }
    port {
      port        = 8080
      target_port = 80
    }
  }
}
%[4]s
`, rName, gcName, protocol, extra)
}

func testAccHTTPRouteV1CorsAndCookieSessionConfig(rName, gcName string) string {
	return smGatewayStack(rName, gcName, "HTTP") + fmt.Sprintf(`
resource "kubernetes_http_route_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    parent_refs {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }
    hostnames = ["api.example.com"]

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
          allow_origins    = ["https://app.example.com"]
          allow_methods    = ["GET", "POST"]
          allow_headers    = ["Authorization"]
          expose_headers   = ["X-Request-Id"]
          allow_credentials = true
          max_age          = 3600
        }
      }

      backend_refs {
        name = kubernetes_service_v1.backend.metadata.0.name
        port = 8080
      }

      session_persistence {
        type             = "Cookie"
        session_name     = "JSESSIONID"
        absolute_timeout = "1h"
        idle_timeout     = "30m"
        cookie_config {
          lifetime_type = "Permanent"
        }
      }
    }
  }
}
`, rName)
}

func testAccHTTPRouteV1MirrorAndCanaryConfig(rName, gcName string) string {
	extraSvcs := fmt.Sprintf(`
resource "kubernetes_service_v1" "stable" {
  metadata { name = "%[1]s-stable" }
  spec {
    selector = { app = "stable" }
    port {
      port        = 8080
      target_port = 80
    }
  }
}
resource "kubernetes_service_v1" "canary" {
  metadata { name = "%[1]s-canary" }
  spec {
    selector = { app = "canary" }
    port {
      port        = 8080
      target_port = 80
    }
  }
}
resource "kubernetes_service_v1" "shadow" {
  metadata { name = "%[1]s-shadow" }
  spec {
    selector = { app = "shadow" }
    port {
      port        = 8080
      target_port = 80
    }
  }
}
`, rName)

	return smGatewayStack(rName, gcName, "HTTP", extraSvcs) + fmt.Sprintf(`
resource "kubernetes_http_route_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    parent_refs {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }
    hostnames = ["canary.example.com"]

    rules {
      matches {
        path {
          type  = "PathPrefix"
          value = "/"
        }
      }

      filters {
        type = "RequestMirror"
        request_mirror {
          backend_ref {
            name = kubernetes_service_v1.shadow.metadata.0.name
            port = 8080
          }
          percent = 42
        }
      }

      backend_refs {
        name   = kubernetes_service_v1.stable.metadata.0.name
        port   = 8080
        weight = 90
      }
      backend_refs {
        name   = kubernetes_service_v1.canary.metadata.0.name
        port   = 8080
        weight = 10
      }
    }
  }
}
`, rName)
}

func testAccBackendTLSPolicyV1SubjectAltNamesConfig(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_service_v1" "auth" {
  metadata {
    name = "%[1]s-auth"
  }
  spec {
    selector = { app = "auth" }
    port {
      name        = "https"
      port        = 443
      target_port = 8443
    }
  }
}

resource "kubernetes_config_map_v1" "ca" {
  metadata {
    name = "%[1]s-ca"
  }
  data = {
    "ca.crt" = "-----BEGIN CERTIFICATE-----\nMIIBmTCCAQKgAwIBAgIJAKx1/A==\n-----END CERTIFICATE-----\n"
  }
}

resource "kubernetes_backend_tls_policy_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    target_refs {
      group        = ""
      kind         = "Service"
      name         = kubernetes_service_v1.auth.metadata.0.name
      section_name = "https"
    }
    validation {
      ca_certificate_refs {
        group = ""
        kind  = "ConfigMap"
        name  = kubernetes_config_map_v1.ca.metadata.0.name
      }
      hostname = "auth.example.com"
      subject_alt_names {
        type     = "Hostname"
        hostname = "auth.example.com"
      }
      subject_alt_names {
        type = "URI"
        uri  = "spiffe://cluster.local/ns/default/sa/auth-service"
      }
    }
  }
}
`, rName)
}

func testAccTLSRouteV1SNIPassthroughConfig(rName, gcName string) string {
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
      name     = "tls-passthrough"
      port     = 443
      protocol = "TLS"
      tls {
        mode = "Passthrough"
      }
    }
  }
}

resource "kubernetes_service_v1" "db" {
  metadata { name = "%[1]s-db" }
  spec {
    selector = { app = "postgres" }
    port {
      port        = 5432
      target_port = 5432
    }
  }
}

resource "kubernetes_service_v1" "kafka" {
  metadata { name = "%[1]s-kafka" }
  spec {
    selector = { app = "kafka" }
    port {
      port        = 9093
      target_port = 9093
    }
  }
}

resource "kubernetes_tls_route_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    parent_refs {
      name         = kubernetes_gateway_v1.test.metadata.0.name
      section_name = "tls-passthrough"
    }
    hostnames = ["db.example.com", "kafka.example.com"]

    rules {
      name = "passthrough-rule"
      backend_refs {
        name = kubernetes_service_v1.db.metadata.0.name
        port = 5432
      }
      backend_refs {
        name = kubernetes_service_v1.kafka.metadata.0.name
        port = 9093
      }
    }
  }
}
`, rName, gcName)
}

func testAccHTTPRouteV1HeaderSessionConfig(rName, gcName string) string {
	return smGatewayStack(rName, gcName, "HTTP") + fmt.Sprintf(`
resource "kubernetes_http_route_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    parent_refs {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }
    hostnames = ["tenant.example.com"]

    rules {
      matches {
        path {
          type  = "PathPrefix"
          value = "/api/v2"
        }
        headers {
          name  = "X-Tenant-ID"
          value = "acme"
          type  = "Exact"
        }
        method = "POST"
      }

      backend_refs {
        name = kubernetes_service_v1.backend.metadata.0.name
        port = 8080
      }

      session_persistence {
        type             = "Header"
        session_name     = "X-Session-Token"
        absolute_timeout = "2h"
        idle_timeout     = "15m"
      }
    }
  }
}
`, rName)
}

func testAccGRPCRouteV1ServiceMeshFullConfig(rName, gcName string) string {
	extraSvcs := fmt.Sprintf(`
resource "kubernetes_service_v1" "grpc_v1" {
  metadata { name = "%[1]s-grpc-v1" }
  spec {
    selector = { app = "grpc-v1" }
    port {
      port        = 50051
      target_port = 50051
    }
  }
}
resource "kubernetes_service_v1" "grpc_v2" {
  metadata { name = "%[1]s-grpc-v2" }
  spec {
    selector = { app = "grpc-v2" }
    port {
      port        = 50051
      target_port = 50051
    }
  }
}
`, rName)

	return smGatewayStack(rName, gcName, "HTTP", extraSvcs) + fmt.Sprintf(`
resource "kubernetes_grpc_route_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    parent_refs {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }
    hostnames = ["grpc.example.com"]

    rules {
      name = "login-rule"

      matches {
        method {
          type    = "Exact"
          service = "com.example.AuthService"
          method  = "Login"
        }
      }

      filters {
        type = "RequestHeaderModifier"
        request_header_modifier {
          add {
            name  = "x-shadow-route"
            value = "true"
          }
        }
      }

      backend_refs {
        name   = kubernetes_service_v1.grpc_v1.metadata.0.name
        port   = 50051
        weight = 80
      }
      backend_refs {
        name   = kubernetes_service_v1.grpc_v2.metadata.0.name
        port   = 50051
        weight = 20
      }

      session_persistence {
        type         = "Cookie"
        session_name = "grpc-session"
        idle_timeout = "10m"
      }
    }

    rules {
      name = "regional-rule"

      matches {
        method {
          type    = "Exact"
          service = "com.example.AuthService"
        }
        headers {
          name  = "x-region"
          value = "eu-west"
          type  = "Exact"
        }
      }

      backend_refs {
        name = kubernetes_service_v1.grpc_v1.metadata.0.name
        port = 50051
      }
    }
  }
}
`, rName)
}

func testAccHTTPRouteV1APIGatewayPatternConfig(rName, gcName string) string {
	extraSvcs := fmt.Sprintf(`
resource "kubernetes_service_v1" "api" {
  metadata { name = "%[1]s-api" }
  spec {
    selector = { app = "api" }
    port {
      port        = 8080
      target_port = 80
    }
  }
}
resource "kubernetes_service_v1" "static" {
  metadata { name = "%[1]s-static" }
  spec {
    selector = { app = "static" }
    port {
      port        = 8080
      target_port = 80
    }
  }
}
`, rName)

	return smGatewayStack(rName, gcName, "HTTP", extraSvcs) + fmt.Sprintf(`
resource "kubernetes_http_route_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    parent_refs {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }
    hostnames = ["gateway.example.com"]

    rules {
      matches {
        path {
          type  = "PathPrefix"
          value = "/"
        }
        headers {
          name  = "X-Forwarded-Proto"
          value = "http"
          type  = "Exact"
        }
      }
      filters {
        type = "RequestRedirect"
        request_redirect {
          scheme      = "https"
          status_code = 301
        }
      }
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
          add {
            name  = "X-Forwarded-Proto"
            value = "https"
          }
          remove = ["X-Internal-Debug"]
        }
      }

      backend_refs {
        name = kubernetes_service_v1.api.metadata.0.name
        port = 8080
      }

      timeouts {
        request         = "30s"
        backend_request = "10s"
      }

      retry {
        codes   = [500, 502, 503, 504]
        attempts = 3
        backoff  = "500ms"
      }
    }

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
            type                = "ReplacePrefixMatch"
            replace_prefix_match = "/assets"
          }
        }
      }

      filters {
        type = "ResponseHeaderModifier"
        response_header_modifier {
          set {
            name  = "Cache-Control"
            value = "public, max-age=31536000"
          }
        }
      }

      backend_refs {
        name = kubernetes_service_v1.static.metadata.0.name
        port = 8080
      }
    }
  }
}
`, rName)
}
