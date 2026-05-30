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

// =============================================================================
// Integration tests that validate complex multi-resource Gateway API
// configurations on a Kind cluster with Gateway API CRDs installed.
//
// Run with:
//   export KUBE_CONFIG_PATH=/tmp/kind-kubeconfig
//   export TF_ACC=1
//   go test -v -run TestAccGatewayAPIIntegration ./kubernetes/ -timeout 60m
//
// Prerequisites:
//   - Kind cluster with Gateway API experimental CRDs
//   - cloud-provider-kind deployed (for LoadBalancer services)
//   - Backend services in backend-ns and gateway-api-tests namespaces
// =============================================================================

// skipIfNoGatewayAPI skips tests when Gateway API CRDs are not installed
func skipIfNoGatewayAPI(t *testing.T) {
	conn, err := testAccProvider.Meta().(KubeClientsets).GatewayClientset()
	if err != nil {
		t.Skipf("Gateway API client not available: %s", err)
		return
	}
	_, err = conn.Gateways("default").List(context.Background(), metav1.ListOptions{Limit: 1})
	if err != nil {
		t.Skipf("Gateway API CRDs not available: %s", err)
	}
}

func skipIfNoListenerSet(t *testing.T) {
	skipIfNoGatewayAPI(t)
	conn, err := testAccProvider.Meta().(KubeClientsets).GatewayClientset()
	if err != nil {
		t.Skipf("Gateway API client not available: %s", err)
		return
	}
	_, err = conn.ListenerSets("default").List(context.Background(), metav1.ListOptions{Limit: 1})
	if err != nil {
		t.Skipf("ListenerSet CRD not available: %s", err)
	}
}

func skipIfNoTLSRoute(t *testing.T) {
	skipIfNoGatewayAPI(t)
	conn, err := testAccProvider.Meta().(KubeClientsets).GatewayClientset()
	if err != nil {
		t.Skipf("Gateway API client not available: %s", err)
		return
	}
	_, err = conn.TLSRoutes("default").List(context.Background(), metav1.ListOptions{Limit: 1})
	if err != nil {
		t.Skipf("TLSRoute v1 CRD not available: %s", err)
	}
}

func skipIfNoReferenceGrantV1(t *testing.T) {
	skipIfNoGatewayAPI(t)
	conn, err := testAccProvider.Meta().(KubeClientsets).GatewayClientset()
	if err != nil {
		t.Skipf("Gateway API client not available: %s", err)
		return
	}
	_, err = conn.ReferenceGrants("default").List(context.Background(), metav1.ListOptions{Limit: 1})
	if err != nil {
		t.Skipf("ReferenceGrant v1 CRD not available: %s", err)
	}
}

// =============================================================================
// 1. Full Stack - GatewayClass + Gateway + HTTPRoute + ReferenceGrant + BackendTLSPolicy
// =============================================================================

func TestAccGatewayAPIIntegration_fullStack(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")
	resourceName := "kubernetes_gateway_v1.int"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoReferenceGrantV1(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGatewayV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIIntegrationFullStack(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckGatewayV1Exists(resourceName, &gatewayv1.Gateway{}),
					resource.TestCheckResourceAttr("kubernetes_gateway_class_v1.int", "metadata.0.name", rName+"-gc"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", rName+"-gw"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.int", "metadata.0.name", rName+"-route"),
					resource.TestCheckResourceAttr("kubernetes_reference_grant_v1.int", "metadata.0.name", rName+"-rg"),
					resource.TestCheckResourceAttr("kubernetes_backend_tls_policy_v1.int", "metadata.0.name", rName+"-btls"),
				),
			},
		},
	})
}

// =============================================================================
// 2. Gateway - All Spec Options (addresses, infrastructure, allowedRoutes)
// =============================================================================

func TestAccGatewayAPIIntegration_gatewayWithAllOptions(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")
	resourceName := "kubernetes_gateway_v1.full"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGatewayV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIIntegrationGatewayFullOptions(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", rName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.gateway_class_name", rName+"-gc"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.name", "http"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.port", "80"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.protocol", "HTTP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.hostname", "*.example.com"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.allowed_routes.0.namespaces.0.from", "All"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.allowed_routes.0.kinds.0.kind", "HTTPRoute"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.addresses.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.addresses.0.value", "10.0.0.1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.infrastructure.0.labels.foo", "bar"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.infrastructure.0.annotations.test-key", "test-value"),
				),
			},
		},
	})
}

// =============================================================================
// 3. Gateway - Update InPlace (add listener, addresses, infrastructure)
// =============================================================================

func TestAccGatewayAPIIntegration_gatewayUpdateInPlace(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")
	resourceName := "kubernetes_gateway_v1.update_test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGatewayV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIIntegrationGatewayBefore(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.port", "80"),
				),
			},
			{
				Config: testAccGatewayAPIIntegrationGatewayAfter(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.port", "80"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.1.port", "443"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.addresses.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.infrastructure.0.labels.env", "test"),
				),
			},
		},
	})
}

// =============================================================================
// 4. HTTPRoute - All Match Types (PathPrefix, Exact, header, query_param, method)
// =============================================================================

func TestAccGatewayAPIIntegration_httpRouteAllMatchTypes(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIIntegrationHTTPRouteMatchTypes(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.matches", "spec.0.rules.0.matches.0.path.0.type", "PathPrefix"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.matches", "spec.0.rules.0.matches.0.path.0.value", "/api"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.matches", "spec.0.rules.1.matches.0.path.0.type", "Exact"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.matches", "spec.0.rules.1.matches.0.path.0.value", "/health"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.matches", "spec.0.rules.2.matches.0.headers.0.name", "X-Custom"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.matches", "spec.0.rules.2.matches.0.headers.0.value", "test-value"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.matches", "spec.0.rules.3.matches.0.query_params.0.name", "foo"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.matches", "spec.0.rules.3.matches.0.query_params.0.value", "bar"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.matches", "spec.0.rules.4.matches.0.method", "POST"),
				),
			},
		},
	})
}

// =============================================================================
// 5. HTTPRoute - All Filter Types (header, redirect, rewrite, mirror, CORS)
// =============================================================================

func TestAccGatewayAPIIntegration_httpRouteFilters(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIIntegrationHTTPRouteFilters(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.filters", "spec.0.rules.0.filters.0.type", "RequestHeaderModifier"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.filters", "spec.0.rules.1.filters.0.type", "RequestRedirect"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.filters", "spec.0.rules.2.filters.0.type", "URLRewrite"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.filters", "spec.0.rules.3.filters.0.type", "RequestMirror"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.filters", "spec.0.rules.4.filters.0.type", "CORS"),
				),
			},
		},
	})
}

// =============================================================================
// 6. HTTPRoute - Backend Weight Distribution
// =============================================================================

func TestAccGatewayAPIIntegration_httpRouteBackendWeights(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIIntegrationBackendWeights(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.weights", "spec.0.rules.0.backend_refs.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.weights", "spec.0.rules.0.backend_refs.0.weight", "80"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.weights", "spec.0.rules.0.backend_refs.1.weight", "20"),
				),
			},
		},
	})
}

// =============================================================================
// 7. HTTPRoute - Timeouts and Session Persistence
// =============================================================================

func TestAccGatewayAPIIntegration_httpRouteTimeoutsAndSessionPersistence(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIIntegrationHTTPRouteAdvanced(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.advanced", "spec.0.rules.0.timeouts.0.request", "30s"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.advanced", "spec.0.rules.0.timeouts.0.backend_request", "10s"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.advanced", "spec.0.rules.0.session_persistence.0.type", "Cookie"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.advanced", "spec.0.rules.0.session_persistence.0.absolute_timeout", "300s"),
				),
			},
		},
	})
}

// =============================================================================
// 8. GRPCRoute - Method Matching
// =============================================================================

func TestAccGatewayAPIIntegration_grpcRouteMethodMatching(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIIntegrationGRPCRoute(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_grpc_route_v1.int", "metadata.0.name", rName+"-grpc"),
					resource.TestCheckResourceAttr("kubernetes_grpc_route_v1.int", "spec.0.rules.0.matches.0.method.0.service", "example.Service"),
					resource.TestCheckResourceAttr("kubernetes_grpc_route_v1.int", "spec.0.rules.0.matches.0.method.0.method", "Method"),
				),
			},
		},
	})
}

// =============================================================================
// 9. TLSRoute - Basic Routing
// =============================================================================

func TestAccGatewayAPIIntegration_tlsRoute(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoTLSRoute(t)
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIIntegrationTLSRoute(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_tls_route_v1.int", "metadata.0.name", rName+"-tls"),
					resource.TestCheckResourceAttr("kubernetes_tls_route_v1.int", "spec.0.rules.0.backend_refs.0.name", rName+"-svc"),
					resource.TestCheckResourceAttr("kubernetes_tls_route_v1.int", "spec.0.rules.0.backend_refs.0.port", "443"),
				),
			},
		},
	})
}

// =============================================================================
// 10. ListenerSet - Multiple Listeners with Gateway
// =============================================================================

func TestAccGatewayAPIIntegration_listenerSetWithGateway(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoListenerSet(t)
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIIntegrationListenerSet(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_listener_set_v1.int", "metadata.0.name", rName+"-ls"),
					resource.TestCheckResourceAttr("kubernetes_listener_set_v1.int", "spec.0.parent_ref.0.name", rName+"-gw"),
					resource.TestCheckResourceAttr("kubernetes_listener_set_v1.int", "spec.0.listeners.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_listener_set_v1.int", "spec.0.listeners.0.name", "http"),
					resource.TestCheckResourceAttr("kubernetes_listener_set_v1.int", "spec.0.listeners.0.port", "80"),
					resource.TestCheckResourceAttr("kubernetes_listener_set_v1.int", "spec.0.listeners.0.protocol", "HTTP"),
					resource.TestCheckResourceAttr("kubernetes_listener_set_v1.int", "spec.0.listeners.1.name", "tls"),
					resource.TestCheckResourceAttr("kubernetes_listener_set_v1.int", "spec.0.listeners.1.port", "443"),
					resource.TestCheckResourceAttr("kubernetes_listener_set_v1.int", "spec.0.listeners.1.protocol", "TLS"),
				),
			},
		},
	})
}

// =============================================================================
// 11. Cross-Namespace Reference with ReferenceGrant
// =============================================================================

func TestAccGatewayAPIIntegration_crossNamespaceWithGrant(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoReferenceGrantV1(t)
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIIntegrationCrossNamespace(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.cross_ns", "spec.0.rules.0.backend_refs.0.namespace", rName+"-backend"),
					resource.TestCheckResourceAttr("kubernetes_reference_grant_v1.cross_ns", "spec.0.from.0.kind", "HTTPRoute"),
					resource.TestCheckResourceAttr("kubernetes_reference_grant_v1.cross_ns", "spec.0.from.0.namespace", rName+"-route"),
					resource.TestCheckResourceAttr("kubernetes_reference_grant_v1.cross_ns", "spec.0.to.0.kind", "Service"),
				),
			},
		},
	})
}

// =============================================================================
// 12. BackendTLSPolicy - Validation + Options
// =============================================================================

func TestAccGatewayAPIIntegration_backendTLSPolicyWithValidation(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckBackendTLSPolicyV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIIntegrationBackendTLS(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_backend_tls_policy_v1.int", "metadata.0.name", rName+"-btls"),
					resource.TestCheckResourceAttr("kubernetes_backend_tls_policy_v1.int", "spec.0.validation.0.hostname", "backend.example.com"),
					resource.TestCheckResourceAttr("kubernetes_backend_tls_policy_v1.int", "spec.0.validation.0.well_known_ca_certificates", "System"),
					resource.TestCheckResourceAttr("kubernetes_backend_tls_policy_v1.int", "spec.0.options.min_version", "VersionTLS12"),
					resource.TestCheckResourceAttr("kubernetes_backend_tls_policy_v1.int", "spec.0.options.max_version", "VersionTLS13"),
				),
			},
		},
	})
}

// =============================================================================
// 13. Gateway - TLS Listener with CertificateRefs
// =============================================================================

func TestAccGatewayAPIIntegration_gatewayTLSListener(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")
	resourceName := "kubernetes_gateway_v1.tls"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGatewayV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIIntegrationGatewayTLSListener(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.name", "https"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.port", "443"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.protocol", "HTTPS"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.tls.0.mode", "Terminate"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.tls.0.certificate_refs.#", "1"),
				),
			},
		},
	})
}

// =============================================================================
// Helper: destroy check for integration test resources
// =============================================================================

func testAccCheckGatewayAPIIntegrationDestroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).GatewayClientset()
	if err != nil {
		return err
	}
	ctx := context.Background()

	for _, rs := range s.RootModule().Resources {
		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			continue
		}

		switch rs.Type {
		case "kubernetes_gateway_v1":
			_, err := conn.Gateways(namespace).Get(ctx, name, metav1.GetOptions{})
			if err == nil {
				return fmt.Errorf("Gateway %s still exists", rs.Primary.ID)
			}
		case "kubernetes_http_route_v1":
			_, err := conn.HTTPRoutes(namespace).Get(ctx, name, metav1.GetOptions{})
			if err == nil {
				return fmt.Errorf("HTTPRoute %s still exists", rs.Primary.ID)
			}
		case "kubernetes_grpc_route_v1":
			_, err := conn.GRPCRoutes(namespace).Get(ctx, name, metav1.GetOptions{})
			if err == nil {
				return fmt.Errorf("GRPCRoute %s still exists", rs.Primary.ID)
			}
		case "kubernetes_tls_route_v1":
			_, err := conn.TLSRoutes(namespace).Get(ctx, name, metav1.GetOptions{})
			if err == nil {
				return fmt.Errorf("TLSRoute %s still exists", rs.Primary.ID)
			}
		case "kubernetes_listener_set_v1":
			_, err := conn.ListenerSets(namespace).Get(ctx, name, metav1.GetOptions{})
			if err == nil {
				return fmt.Errorf("ListenerSet %s still exists", rs.Primary.ID)
			}
		case "kubernetes_reference_grant_v1":
			_, err := conn.ReferenceGrants(namespace).Get(ctx, name, metav1.GetOptions{})
			if err == nil {
				return fmt.Errorf("ReferenceGrant %s still exists", rs.Primary.ID)
			}
		case "kubernetes_backend_tls_policy_v1":
			_, err := conn.BackendTLSPolicies(namespace).Get(ctx, name, metav1.GetOptions{})
			if err == nil {
				return fmt.Errorf("BackendTLSPolicy %s still exists", rs.Primary.ID)
			}
		}
	}
	return nil
}
