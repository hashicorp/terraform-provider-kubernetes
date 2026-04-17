// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

// Acceptance tests for Gateway API data sources.
// Each test follows the two-step pattern:
//   Step 1 - create the underlying resource and verify it exists.
//   Step 2 - add the data source, then check that it returns the same values.

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// -----------------------------------------------------------------------------
// kubernetes_gateway_v1 data source
// -----------------------------------------------------------------------------

func TestAccKubernetesGatewayV1DataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	gcName := acctest.RandomWithPrefix("tf-acc-gc")
	resourceName := "kubernetes_gateway_v1.test"
	dataSourceName := "data.kubernetes_gateway_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayV1DataSourceConfig(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.name", "http"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.protocol", "HTTP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.port", "80"),
				),
			},
			{
				Config: testAccGatewayV1DataSourceConfig(rName, gcName) + testAccGatewayV1DataSourceReadConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", rName+"-gw"),
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.namespace", "default"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.gateway_class_name", gcName),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.listeners.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.listeners.0.name", "http"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.listeners.0.protocol", "HTTP"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.listeners.0.port", "80"),
				),
			},
		},
	})
}

func testAccGatewayV1DataSourceConfig(rName, gcName string) string {
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
    name      = "%[1]s-gw"
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
`, rName, gcName)
}

func testAccGatewayV1DataSourceReadConfig() string {
	return `
data "kubernetes_gateway_v1" "test" {
  metadata {
    name      = kubernetes_gateway_v1.test.metadata.0.name
    namespace = kubernetes_gateway_v1.test.metadata.0.namespace
  }
}
`
}

// -----------------------------------------------------------------------------
// kubernetes_http_route_v1 data source
// -----------------------------------------------------------------------------

func TestAccKubernetesHTTPRouteV1DataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	gcName := acctest.RandomWithPrefix("tf-acc-gc")
	resourceName := "kubernetes_http_route_v1.test"
	dataSourceName := "data.kubernetes_http_route_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckHTTPRouteV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccHTTPRouteV1DataSourceConfig(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "spec.0.hostnames.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.filters.0.type", "RequestHeaderModifier"),
				),
			},
			{
				Config: testAccHTTPRouteV1DataSourceConfig(rName, gcName) + testAccHTTPRouteV1DataSourceReadConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", rName),
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.namespace", "default"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.hostnames.0", "ds.example.com"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.parent_refs.0.name", rName+"-gw"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.rules.0.filters.0.type", "RequestHeaderModifier"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.rules.0.filters.0.request_header_modifier.0.add.0.name", "X-DS-Test"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.rules.0.backend_refs.0.name", rName+"-svc"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.rules.0.backend_refs.0.port", "80"),
				),
			},
		},
	})
}

func testAccHTTPRouteV1DataSourceConfig(rName, gcName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = %[2]q
  }
  spec {
    controller_name = "example.com/gateway-controller"
  }
}

resource "kubernetes_service_v1" "test" {
  metadata {
    name      = "%[1]s-svc"
    namespace = "default"
  }
  spec {
    selector = { app = "test" }
    port {
      port        = 80
      target_port = 8080
    }
  }
}

resource "kubernetes_gateway_v1" "test" {
  metadata {
    name      = "%[1]s-gw"
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

resource "kubernetes_http_route_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = "default"
  }
  spec {
    parent_refs {
      name      = kubernetes_gateway_v1.test.metadata.0.name
      namespace = "default"
    }
    hostnames = ["ds.example.com"]
    rules {
      filters {
        type = "RequestHeaderModifier"
        request_header_modifier {
          add {
            name  = "X-DS-Test"
            value = "true"
          }
        }
      }
      backend_refs {
        name      = kubernetes_service_v1.test.metadata.0.name
        namespace = "default"
        port      = 80
      }
    }
  }
}
`, rName, gcName)
}

func testAccHTTPRouteV1DataSourceReadConfig() string {
	return `
data "kubernetes_http_route_v1" "test" {
  metadata {
    name      = kubernetes_http_route_v1.test.metadata.0.name
    namespace = kubernetes_http_route_v1.test.metadata.0.namespace
  }
}
`
}

// -----------------------------------------------------------------------------
// kubernetes_grpc_route_v1 data source
// -----------------------------------------------------------------------------

func TestAccKubernetesGRPCRouteV1DataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	gcName := acctest.RandomWithPrefix("tf-acc-gc")
	resourceName := "kubernetes_grpc_route_v1.test"
	dataSourceName := "data.kubernetes_grpc_route_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGRPCRouteV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGRPCRouteV1DataSourceConfig(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.matches.0.method.0.service", "com.example.EchoService"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.matches.0.method.0.method", "Echo"),
				),
			},
			{
				Config: testAccGRPCRouteV1DataSourceConfig(rName, gcName) + testAccGRPCRouteV1DataSourceReadConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", rName),
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.namespace", "default"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.hostnames.0", "grpc-ds.example.com"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.rules.0.matches.0.method.0.service", "com.example.EchoService"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.rules.0.matches.0.method.0.method", "Echo"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.rules.0.backend_refs.0.name", rName+"-svc"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.rules.0.backend_refs.0.port", "50051"),
				),
			},
		},
	})
}

func testAccGRPCRouteV1DataSourceConfig(rName, gcName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = %[2]q
  }
  spec {
    controller_name = "example.com/gateway-controller"
  }
}

resource "kubernetes_service_v1" "test" {
  metadata {
    name      = "%[1]s-svc"
    namespace = "default"
  }
  spec {
    selector = { app = "grpc-test" }
    port {
      port        = 50051
      target_port = 50051
    }
  }
}

resource "kubernetes_gateway_v1" "test" {
  metadata {
    name      = "%[1]s-gw"
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

resource "kubernetes_grpc_route_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = "default"
  }
  spec {
    parent_refs {
      name      = kubernetes_gateway_v1.test.metadata.0.name
      namespace = "default"
    }
    hostnames = ["grpc-ds.example.com"]
    rules {
      matches {
        method {
          type    = "Exact"
          service = "com.example.EchoService"
          method  = "Echo"
        }
      }
      backend_refs {
        name      = kubernetes_service_v1.test.metadata.0.name
        namespace = "default"
        port      = 50051
      }
    }
  }
}
`, rName, gcName)
}

func testAccGRPCRouteV1DataSourceReadConfig() string {
	return `
data "kubernetes_grpc_route_v1" "test" {
  metadata {
    name      = kubernetes_grpc_route_v1.test.metadata.0.name
    namespace = kubernetes_grpc_route_v1.test.metadata.0.namespace
  }
}
`
}

// -----------------------------------------------------------------------------
// kubernetes_tls_route_v1 data source
// -----------------------------------------------------------------------------

func TestAccKubernetesTLSRouteV1DataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	gcName := acctest.RandomWithPrefix("tf-acc-gc")
	resourceName := "kubernetes_tls_route_v1.test"
	dataSourceName := "data.kubernetes_tls_route_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckTLSRouteV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTLSRouteV1DataSourceConfig(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "spec.0.hostnames.0", "tls-ds.example.com"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.backend_refs.0.name", rName+"-svc"),
				),
			},
			{
				Config: testAccTLSRouteV1DataSourceConfig(rName, gcName) + testAccTLSRouteV1DataSourceReadConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", rName),
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.namespace", "default"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.hostnames.0", "tls-ds.example.com"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.rules.0.backend_refs.0.name", rName+"-svc"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.rules.0.backend_refs.0.port", "443"),
				),
			},
		},
	})
}

func testAccTLSRouteV1DataSourceConfig(rName, gcName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = %[2]q
  }
  spec {
    controller_name = "example.com/gateway-controller"
  }
}

resource "kubernetes_service_v1" "test" {
  metadata {
    name      = "%[1]s-svc"
    namespace = "default"
  }
  spec {
    selector = { app = "tls-test" }
    port {
      port        = 443
      target_port = 8443
    }
  }
}

resource "kubernetes_gateway_v1" "test" {
  metadata {
    name      = "%[1]s-gw"
    namespace = "default"
  }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.test.metadata.0.name
    listeners {
      name     = "tls"
      protocol = "TLS"
      port     = 443
      tls {
        mode = "Passthrough"
      }
    }
  }
}

resource "kubernetes_tls_route_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = "default"
  }
  spec {
    parent_refs {
      name      = kubernetes_gateway_v1.test.metadata.0.name
      namespace = "default"
    }
    hostnames = ["tls-ds.example.com"]
    rules {
      backend_refs {
        name      = kubernetes_service_v1.test.metadata.0.name
        namespace = "default"
        port      = 443
      }
    }
  }
}
`, rName, gcName)
}

func testAccTLSRouteV1DataSourceReadConfig() string {
	return `
data "kubernetes_tls_route_v1" "test" {
  metadata {
    name      = kubernetes_tls_route_v1.test.metadata.0.name
    namespace = kubernetes_tls_route_v1.test.metadata.0.namespace
  }
}
`
}

// -----------------------------------------------------------------------------
// kubernetes_reference_grant_v1 data source
// -----------------------------------------------------------------------------

func TestAccKubernetesReferenceGrantV1DataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_reference_grant_v1.test"
	dataSourceName := "data.kubernetes_reference_grant_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckReferenceGrantV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccReferenceGrantV1DataSourceConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "spec.0.from.0.kind", "HTTPRoute"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.to.0.kind", "Service"),
				),
			},
			{
				Config: testAccReferenceGrantV1DataSourceConfig(rName) + testAccReferenceGrantV1DataSourceReadConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", rName),
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.namespace", "default"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.from.0.group", "gateway.networking.k8s.io"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.from.0.kind", "HTTPRoute"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.from.0.namespace", "default"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.to.0.group", ""),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.to.0.kind", "Service"),
				),
			},
		},
	})
}

func testAccReferenceGrantV1DataSourceConfig(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_reference_grant_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = "default"
  }
  spec {
    from {
      group     = "gateway.networking.k8s.io"
      kind      = "HTTPRoute"
      namespace = "default"
    }
    to {
      group = ""
      kind  = "Service"
    }
  }
}
`, rName)
}

func testAccReferenceGrantV1DataSourceReadConfig() string {
	return `
data "kubernetes_reference_grant_v1" "test" {
  metadata {
    name      = kubernetes_reference_grant_v1.test.metadata.0.name
    namespace = kubernetes_reference_grant_v1.test.metadata.0.namespace
  }
}
`
}

// -----------------------------------------------------------------------------
// kubernetes_backend_tls_policy_v1 data source
// -----------------------------------------------------------------------------

func TestAccKubernetesBackendTLSPolicyV1DataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_backend_tls_policy_v1.test"
	dataSourceName := "data.kubernetes_backend_tls_policy_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckBackendTLSPolicyV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBackendTLSPolicyV1DataSourceConfig(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "spec.0.target_refs.0.kind", "Service"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.validation.0.well_known_ca_certificates", "System"),
				),
			},
			{
				Config: testAccBackendTLSPolicyV1DataSourceConfig(rName) + testAccBackendTLSPolicyV1DataSourceReadConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", rName),
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.namespace", "default"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.target_refs.0.group", ""),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.target_refs.0.kind", "Service"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.target_refs.0.name", rName+"-svc"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.validation.0.hostname", "svc.example.com"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.validation.0.well_known_ca_certificates", "System"),
				),
			},
		},
	})
}

func testAccBackendTLSPolicyV1DataSourceConfig(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_service_v1" "test" {
  metadata {
    name      = "%[1]s-svc"
    namespace = "default"
  }
  spec {
    selector = { app = "tls-backend" }
    port {
      port        = 443
      target_port = 8443
    }
  }
}

resource "kubernetes_backend_tls_policy_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = "default"
  }
  spec {
    target_refs {
      group = ""
      kind  = "Service"
      name  = kubernetes_service_v1.test.metadata.0.name
    }
    validation {
      hostname                   = "svc.example.com"
      well_known_ca_certificates = "System"
    }
  }
}
`, rName)
}

func testAccBackendTLSPolicyV1DataSourceReadConfig() string {
	return `
data "kubernetes_backend_tls_policy_v1" "test" {
  metadata {
    name      = kubernetes_backend_tls_policy_v1.test.metadata.0.name
    namespace = kubernetes_backend_tls_policy_v1.test.metadata.0.namespace
  }
}
`
}

// -----------------------------------------------------------------------------
// kubernetes_listener_set_v1 data source
// -----------------------------------------------------------------------------

func TestAccKubernetesListenerSetV1DataSource_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc-test")
	gcName := acctest.RandomWithPrefix("tf-acc-gc")
	resourceName := "kubernetes_listener_set_v1.test"
	dataSourceName := "data.kubernetes_listener_set_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckListenerSetV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccListenerSetV1DataSourceConfig(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.name", "extra-http"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.1.name", "extra-alt"),
				),
			},
			{
				Config: testAccListenerSetV1DataSourceConfig(rName, gcName) + testAccListenerSetV1DataSourceReadConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", rName),
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.namespace", "default"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.parent_ref.0.name", rName+"-gw"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.listeners.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.listeners.0.name", "extra-http"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.listeners.0.port", "8080"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.listeners.0.protocol", "HTTP"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.listeners.1.name", "extra-alt"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.listeners.1.port", "8081"),
				),
			},
		},
	})
}

func testAccListenerSetV1DataSourceConfig(rName, gcName string) string {
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
    name      = "%[1]s-gw"
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

resource "kubernetes_listener_set_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = "default"
  }
  spec {
    parent_ref {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }
    listeners {
      name     = "extra-http"
      protocol = "HTTP"
      port     = 8080
    }
    listeners {
      name     = "extra-alt"
      protocol = "HTTP"
      port     = 8081
    }
  }
}
`, rName, gcName)
}

func testAccListenerSetV1DataSourceReadConfig() string {
	return `
data "kubernetes_listener_set_v1" "test" {
  metadata {
    name      = kubernetes_listener_set_v1.test.metadata.0.name
    namespace = kubernetes_listener_set_v1.test.metadata.0.namespace
  }
}
`
}
