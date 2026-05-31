// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// ============================================================
// Gateway API Schema Validation Tests
// These tests verify schema constraints (enum, MaxItems, MinItems)
// are enforced at the Terraform provider layer before reaching K8s.
// They run fast without a cluster and catch field-level bugs.
// ============================================================

// Gateway protocol enum validation
func TestAccGatewayV1_invalidProtocol(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccGatewayV1ConfigInvalidProtocol(rName),
				ExpectError: regexp.MustCompile(`protocol to be one of`),
			},
		},
	})
}

func TestAccGatewayV1_validProtocols(t *testing.T) {
	resourceName := "kubernetes_gateway_v1.test"
	// TLS is validated separately: the Gateway API CEL rules require a tls.mode
	// for protocol TLS, which this generic single-listener config does not set.
	protocols := []string{"HTTP", "HTTPS", "TCP", "UDP"}

	for _, protocol := range protocols {
		protocol := protocol
		t.Run(protocol, func(t *testing.T) {
			rName := acctest.RandomWithPrefix("tf-acc-test")
			gcName := acctest.RandomWithPrefix("tf-acc-test-gc")
			resource.ParallelTest(t, resource.TestCase{
				PreCheck:          func() { testAccPreCheck(t) },
				ProviderFactories: testAccProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: testAccGatewayV1ConfigProtocol(rName, gcName, protocol),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.protocol", protocol),
						),
					},
				},
			})
		})
	}
}

func TestAccGatewayV1_listenerTLSModeEnum(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccGatewayV1ConfigInvalidTLSMode(rName),
				ExpectError: regexp.MustCompile(`expected mode to be one of`),
			},
		},
	})
}

func TestAccGatewayV1_listenerNameMaxLength(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	// Generate a listener name longer than 253 characters
	longName := strings.Repeat("a", 254)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccGatewayV1ConfigLongListenerName(rName, longName),
				ExpectError: regexp.MustCompile(`expected length of name to be in the range`),
			},
		},
	})
}

// HTTPRoute validation tests
func TestAccHTTPRouteV1_invalidFilterType(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccHTTPRouteV1ConfigInvalidFilterType(rName),
				ExpectError: regexp.MustCompile(`expected type to be one of`),
			},
		},
	})
}

func TestAccHTTPRouteV1_invalidStatusCode(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccHTTPRouteV1ConfigInvalidStatusCode(rName),
				ExpectError: regexp.MustCompile(`expected status_code to be one of`),
			},
		},
	})
}

func TestAccHTTPRouteV1_invalidPercent(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccHTTPRouteV1ConfigInvalidPercent(rName),
				ExpectError: regexp.MustCompile(`expected percent to be in the range`),
			},
		},
	})
}

func TestAccHTTPRouteV1_invalidPathType(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccHTTPRouteV1ConfigInvalidPathType(rName),
				ExpectError: regexp.MustCompile(`expected type to be one of`),
			},
		},
	})
}

func TestAccHTTPRouteV1_useDefaultGatewaysEnum(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccHTTPRouteV1ConfigInvalidUseDefaultGateways(rName),
				ExpectError: regexp.MustCompile(`expected use_default_gateways to be one of`),
			},
		},
	})
}

// GRPCRoute validation tests
func TestAccGRPCRouteV1_invalidFilterType(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccGRPCRouteV1ConfigInvalidFilterType(rName),
				ExpectError: regexp.MustCompile(`expected type to be one of`),
			},
		},
	})
}

func TestAccGRPCRouteV1_invalidMethodType(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccGRPCRouteV1ConfigInvalidMethodType(rName),
				ExpectError: regexp.MustCompile(`expected type to be one of`),
			},
		},
	})
}

// TLSRoute validation tests
func TestAccTLSRouteV1_maxItemsRules(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccTLSRouteV1ConfigTwoRules(rName),
				ExpectError: regexp.MustCompile(`Too many rules blocks`),
			},
		},
	})
}

// BackendTLSPolicy validation tests
func TestAccBackendTLSPolicyV1_hostnameRequired(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccBackendTLSPolicyV1ConfigMissingHostname(rName),
				ExpectError: regexp.MustCompile(`(?i)(Missing required argument|must specify either)`),
			},
		},
	})
}

func TestAccBackendTLSPolicyV1_subjectAltNamesTypeEnum(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccBackendTLSPolicyV1ConfigInvalidSubjectAltNameType(rName),
				ExpectError: regexp.MustCompile(`expected type to be one of`),
			},
		},
	})
}

// GatewayClass validation tests
func TestAccGatewayClassV1_invalidControllerNamePattern(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test-gc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccGatewayClassV1ConfigInvalidControllerName(rName),
				ExpectError: regexp.MustCompile(`ControllerName must be a domain-prefixed path`),
			},
		},
	})
}

// ============================================================
// HCL Config Templates
// ============================================================

func testAccGatewayV1ConfigInvalidProtocol(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = "%[1]s-gc"
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
      name     = "test"
      port     = 80
      protocol = "INVALID"
    }
  }
}
`, rName)
}

func testAccGatewayV1ConfigProtocol(rName, gcName, protocol string) string {
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
      name     = "test"
      port     = 80
      protocol = "%[3]s"
    }
  }
}
`, rName, gcName, protocol)
}

func testAccGatewayV1ConfigInvalidTLSMode(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = "%[1]s-gc"
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
      name     = "tls"
      port     = 443
      protocol = "HTTPS"
      tls {
        mode = "INVALID_MODE"
      }
    }
  }
}
`, rName)
}

func testAccGatewayV1ConfigLongListenerName(rName, longName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = "%[1]s-gc"
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
      name     = %[2]q
      port     = 80
      protocol = "HTTP"
    }
  }
}
`, rName, longName)
}

func testAccHTTPRouteV1ConfigInvalidFilterType(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_namespace_v1" "test" {
  metadata { name = %[1]q }
}

resource "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = "%[1]s-gc"
  }
  spec {
    controller_name = "example.com/gateway-controller"
  }
}

resource "kubernetes_gateway_v1" "test" {
  metadata {
    name      = "%[1]s-gw"
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.test.metadata.0.name
    listeners {
      name     = "test"
      port     = 80
      protocol = "HTTP"
    }
  }
}

resource "kubernetes_http_route_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
  spec {
    parent_refs {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }
    rules {
      matches {
        path {
          value = "/"
        }
      }
      filters {
        type = "INVALID_FILTER_TYPE"
      }
      backend_refs {
        name = "nonexistent"
        port = 8080
      }
    }
  }
}
`, rName)
}

func testAccHTTPRouteV1ConfigInvalidStatusCode(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_namespace_v1" "test" {
  metadata { name = %[1]q }
}

resource "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = "%[1]s-gc"
  }
  spec {
    controller_name = "example.com/gateway-controller"
  }
}

resource "kubernetes_gateway_v1" "test" {
  metadata {
    name      = "%[1]s-gw"
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.test.metadata.0.name
    listeners {
      name     = "test"
      port     = 80
      protocol = "HTTP"
    }
  }
}

resource "kubernetes_http_route_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
  spec {
    parent_refs {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }
    rules {
      matches {
        path { value = "/" }
      }
      filters {
        type = "RequestRedirect"
        request_redirect {
          status_code = 999
        }
      }
    }
  }
}
`, rName)
}

func testAccHTTPRouteV1ConfigInvalidPercent(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_namespace_v1" "test" {
  metadata { name = %[1]q }
}

resource "kubernetes_service_v1" "mirror" {
  metadata {
    name      = "%[1]s-mirror"
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
  spec {
    selector = { app = "mirror" }
    port {
      port        = 8080
      target_port = 8080
    }
  }
}

resource "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = "%[1]s-gc"
  }
  spec {
    controller_name = "example.com/gateway-controller"
  }
}

resource "kubernetes_gateway_v1" "test" {
  metadata {
    name      = "%[1]s-gw"
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.test.metadata.0.name
    listeners {
      name     = "test"
      port     = 80
      protocol = "HTTP"
    }
  }
}

resource "kubernetes_http_route_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
  spec {
    parent_refs {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }
    rules {
      matches {
        path { value = "/" }
      }
      filters {
        type = "RequestMirror"
        request_mirror {
          backend_ref {
            name = kubernetes_service_v1.mirror.metadata.0.name
            port = 8080
          }
          percent = 101
        }
      }
      backend_refs {
        name = "nonexistent"
        port = 8080
      }
    }
  }
}
`, rName)
}

func testAccHTTPRouteV1ConfigInvalidPathType(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_namespace_v1" "test" {
  metadata { name = %[1]q }
}

resource "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = "%[1]s-gc"
  }
  spec {
    controller_name = "example.com/gateway-controller"
  }
}

resource "kubernetes_gateway_v1" "test" {
  metadata {
    name      = "%[1]s-gw"
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.test.metadata.0.name
    listeners {
      name     = "test"
      port     = 80
      protocol = "HTTP"
    }
  }
}

resource "kubernetes_http_route_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
  spec {
    parent_refs {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }
    rules {
      matches {
        path {
          type  = "INVALID_PATH_TYPE"
          value = "/"
        }
      }
      backend_refs {
        name = "nonexistent"
        port = 8080
      }
    }
  }
}
`, rName)
}

func testAccHTTPRouteV1ConfigInvalidUseDefaultGateways(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_namespace_v1" "test" {
  metadata { name = %[1]q }
}

resource "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = "%[1]s-gc"
  }
  spec {
    controller_name = "example.com/gateway-controller"
  }
}

resource "kubernetes_gateway_v1" "test" {
  metadata {
    name      = "%[1]s-gw"
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.test.metadata.0.name
    listeners {
      name     = "test"
      port     = 80
      protocol = "HTTP"
    }
  }
}

resource "kubernetes_http_route_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
  spec {
    use_default_gateways = "INVALID_VALUE"
    rules {
      matches {
        path { value = "/" }
      }
      backend_refs {
        name = "nonexistent"
        port = 8080
      }
    }
  }
}
`, rName)
}

func testAccGRPCRouteV1ConfigInvalidFilterType(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_namespace_v1" "test" {
  metadata { name = %[1]q }
}

resource "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = "%[1]s-gc"
  }
  spec {
    controller_name = "example.com/gateway-controller"
  }
}

resource "kubernetes_gateway_v1" "test" {
  metadata {
    name      = "%[1]s-gw"
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.test.metadata.0.name
    listeners {
      name     = "grpc"
      port     = 443
      protocol = "GRPC"
    }
  }
}

resource "kubernetes_grpc_route_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
  spec {
    parent_refs {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }
    rules {
      matches {
        method {
          service = "service.service"
        }
      }
      filters {
        type = "INVALID_GRPC_FILTER"
      }
      backend_refs {
        name = "nonexistent"
        port = 443
      }
    }
  }
}
`, rName)
}

func testAccGRPCRouteV1ConfigInvalidMethodType(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_namespace_v1" "test" {
  metadata { name = %[1]q }
}

resource "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = "%[1]s-gc"
  }
  spec {
    controller_name = "example.com/gateway-controller"
  }
}

resource "kubernetes_gateway_v1" "test" {
  metadata {
    name      = "%[1]s-gw"
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.test.metadata.0.name
    listeners {
      name     = "grpc"
      port     = 443
      protocol = "GRPC"
    }
  }
}

resource "kubernetes_grpc_route_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
  spec {
    parent_refs {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }
    rules {
      matches {
        method {
          type    = "INVALID_METHOD_TYPE"
          service = "service.service"
        }
      }
      backend_refs {
        name = "nonexistent"
        port = 443
      }
    }
  }
}
`, rName)
}

func testAccTLSRouteV1ConfigTwoRules(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_namespace_v1" "test" {
  metadata { name = %[1]q }
}

resource "kubernetes_service_v1" "test" {
  metadata {
    name      = "%[1]s-svc"
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
  spec {
    selector = { app = "test" }
    port {
      port        = 443
      target_port = 443
    }
  }
}

resource "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = "%[1]s-gc"
  }
  spec {
    controller_name = "example.com/gateway-controller"
  }
}

resource "kubernetes_gateway_v1" "test" {
  metadata {
    name      = "%[1]s-gw"
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.test.metadata.0.name
    listeners {
      name     = "tls"
      port     = 443
      protocol = "TLS"
      tls { mode = "Passthrough" }
    }
  }
}

resource "kubernetes_tls_route_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
  spec {
    hostnames = ["example.com"]
    parent_refs {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }
    rules {
      backend_refs {
        name = kubernetes_service_v1.test.metadata.0.name
        port = 443
      }
    }
    rules {
      backend_refs {
        name = kubernetes_service_v1.test.metadata.0.name
        port = 443
      }
    }
  }
}
`, rName)
}

func testAccBackendTLSPolicyV1ConfigMissingHostname(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_namespace_v1" "test" {
  metadata { name = %[1]q }
}

resource "kubernetes_service_v1" "test" {
  metadata {
    name      = "%[1]s-svc"
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
  spec {
    selector = { app = "test" }
    port {
      port        = 443
      target_port = 443
    }
  }
}

resource "kubernetes_backend_tls_policy_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
  spec {
    target_refs {
      group = ""
      kind  = "Service"
      name  = kubernetes_service_v1.test.metadata.0.name
    }
    validation {
    }
  }
}
`, rName)
}

func testAccBackendTLSPolicyV1ConfigInvalidSubjectAltNameType(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_namespace_v1" "test" {
  metadata { name = %[1]q }
}

resource "kubernetes_service_v1" "test" {
  metadata {
    name      = "%[1]s-svc"
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
  spec {
    selector = { app = "test" }
    port {
      port        = 443
      target_port = 443
    }
  }
}

resource "kubernetes_backend_tls_policy_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
  spec {
    target_refs {
      group = ""
      kind  = "Service"
      name  = kubernetes_service_v1.test.metadata.0.name
    }
    validation {
      hostname = "example.com"
      subject_alt_names {
        type = "INVALID_TYPE"
      }
    }
  }
}
`, rName)
}

func testAccGatewayClassV1ConfigInvalidControllerName(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    controller_name = "invalid-without-domain"
  }
}
`, rName)
}
