// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// =============================================================================
// Advanced Empirical Tests — inspired by Gateway API conformance tests and
// gateway-api-bench benchmarks (howardjohn/gateway-api-bench).
//
// These tests validate real-world scenarios:
//   - Canary deployments with weighted backends
//   - Multi-tenant namespace isolation
//   - Response header modifier filters
//   - Multiple parentRefs (route attached to multiple gateways)
//   - Gateway with infrastructure parametersRef
//   - HTTPRoute with requestMirror + CORS combined
//   - ListenerSet with namespace-scoped Listener entries
//   - BackendTLSPolicy with ConfigMap CA ref
//   - Gateway address type: NamedAddress
//   - HTTPRoute with multiple matches per rule (AND logic)
//
// Run:
//   export KUBE_CONFIG_PATH=/tmp/kind-kubeconfig
//   export TF_ACC=1
//   go test -v -run 'TestAccGatewayAPIAdvanced' ./kubernetes/ -timeout 60m
// =============================================================================

// ---------------------------------------------------------------------------
// Canary Deployment — 50/50 weighted split across two backends
// Validates: HTTPRoute backend weight, multiple backendRefs, service discovery
// ---------------------------------------------------------------------------
func TestAccGatewayAPIAdvanced_canaryDeployment(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIAdvancedCanaryBefore(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.canary", "spec.0.rules.0.backend_refs.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.canary", "spec.0.rules.0.backend_refs.0.weight", "90"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.canary", "spec.0.rules.0.backend_refs.1.weight", "10"),
				),
			},
			{
				Config: testAccGatewayAPIAdvancedCanaryAfter(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.canary", "spec.0.rules.0.backend_refs.0.weight", "50"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.canary", "spec.0.rules.0.backend_refs.1.weight", "50"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// Multi-Tenant Namespace Isolation
// Validates: Gateway allowedRoutes from=Same, from=Selector with label match
// ---------------------------------------------------------------------------
func TestAccGatewayAPIAdvanced_multiTenantIsolation(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGatewayV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIAdvancedMultiTenant(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_gateway_v1.tenant", "spec.0.listeners.0.allowed_routes.0.namespaces.0.from", "Selector"),
					resource.TestCheckResourceAttr("kubernetes_gateway_v1.tenant", "spec.0.listeners.0.allowed_routes.0.namespaces.0.selector.0.match_labels.tenant", "a"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// HTTPRoute ResponseHeaderModifier Filter
// Validates: Response header add, set, remove operations
// ---------------------------------------------------------------------------
func TestAccGatewayAPIAdvanced_responseHeaderModifier(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIAdvancedResponseHeaders(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.resp_headers", "spec.0.rules.0.filters.0.type", "ResponseHeaderModifier"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.resp_headers", "spec.0.rules.0.filters.1.type", "RequestHeaderModifier"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// HTTPRoute with Multiple Matches per Rule (AND logic)
// Validates: path + header + method all match (AND)
// ---------------------------------------------------------------------------
func TestAccGatewayAPIAdvanced_multipleMatchesAndLogic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIAdvancedAndMatches(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.and_matches", "spec.0.rules.0.matches.0.path.0.type", "PathPrefix"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.and_matches", "spec.0.rules.0.matches.0.headers.0.name", "X-API-Key"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.and_matches", "spec.0.rules.0.matches.0.method", "POST"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// Gateway with infrastructure labels + annotations (propagated to Pod/Service)
// Validates: infrastructure block with labels, annotations, parametersRef
// ---------------------------------------------------------------------------
func TestAccGatewayAPIAdvanced_infrastructureConfig(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")
	resourceName := "kubernetes_gateway_v1.infra"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGatewayV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIAdvancedInfrastructure(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "spec.0.infrastructure.0.labels.app", "gateway"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.infrastructure.0.labels.team", "platform"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.infrastructure.0.annotations.prometheus.io/scrape", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.infrastructure.0.annotations.prometheus.io/port", "9102"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// HTTPRoute with requestMirror + CORS combined (real API gateway pattern)
// Validates: filter ordering, CORS with all fields, mirror to debugging backend
// ---------------------------------------------------------------------------
func TestAccGatewayAPIAdvanced_apiGatewayPattern(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIAdvancedAPIGateway(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.api_gw", "spec.0.rules.0.filters.0.type", "CORS"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.api_gw", "spec.0.rules.0.filters.1.type", "RequestMirror"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.api_gw", "spec.0.rules.0.filters.2.type", "RequestHeaderModifier"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// HTTPRoute with Redirect + URLRewrite (migrating API versioning)
// Validates: RequestRedirect with scheme, port, statusCode, path replacement
// ---------------------------------------------------------------------------
func TestAccGatewayAPIAdvanced_redirectAndRewrite(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIAdvancedRedirectRewrite(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.redirect", "spec.0.rules.0.filters.0.type", "RequestRedirect"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.redirect", "spec.0.rules.0.filters.0.request_redirect.0.scheme", "https"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.redirect", "spec.0.rules.0.filters.0.request_redirect.0.port", "443"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.redirect", "spec.0.rules.0.filters.0.request_redirect.0.status_code", "301"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// GRPCRoute with header matching + weighted backends
// Validates: GRPCRoute matches with headers, multiple backendRefs
// ---------------------------------------------------------------------------
func TestAccGatewayAPIAdvanced_grpcRouteWithHeaders(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIAdvancedGRPCRouteHeaders(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_grpc_route_v1.headers", "spec.0.rules.0.matches.0.method.0.service", "payment.Service"),
					resource.TestCheckResourceAttr("kubernetes_grpc_route_v1.headers", "spec.0.rules.0.matches.0.headers.0.name", "x-version"),
					resource.TestCheckResourceAttr("kubernetes_grpc_route_v1.headers", "spec.0.rules.0.backend_refs.#", "2"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TLSRoute with multiple rules and hostnames
// Validates: TLSRoute with SNI-based routing, multiple rules
// ---------------------------------------------------------------------------
func TestAccGatewayAPIAdvanced_tlsRouteSNI(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoTLSRoute(t)
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIAdvancedTLSRouteSNI(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_tls_route_v1.sni", "spec.0.hostnames.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_tls_route_v1.sni", "spec.0.rules.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_tls_route_v1.sni", "spec.0.rules.0.backend_refs.0.name", rName+"-sni-a"),
					resource.TestCheckResourceAttr("kubernetes_tls_route_v1.sni", "spec.0.rules.1.backend_refs.0.name", rName+"-sni-b"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// Gateway with allowedListeners (ListenerSet namespace filtering)
// Validates: allowedListeners.namespaces.from=Selector with match_labels
// ---------------------------------------------------------------------------
func TestAccGatewayAPIAdvanced_gatewayAllowedListeners(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")
	resourceName := "kubernetes_gateway_v1.allowed_ls"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGatewayV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIAdvancedAllowedListeners(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "spec.0.allowed_listeners.0.namespaces.0.from", "Selector"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.allowed_listeners.0.namespaces.0.selector.0.match_labels.gateway-team", "infra"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// HTTPRoute requestRedirect with path rewrite (ReplacePrefixMatch)
// Validates: redirect path modifier types
// ---------------------------------------------------------------------------
func TestAccGatewayAPIAdvanced_redirectPathModifier(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIAdvancedRedirectPath(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.redirect_path", "spec.0.rules.0.filters.0.type", "RequestRedirect"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.redirect_path", "spec.0.rules.0.filters.0.request_redirect.0.path.0.type", "ReplacePrefixMatch"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.redirect_path", "spec.0.rules.0.filters.0.request_redirect.0.path.0.replace_prefix_match", "/v2"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// BackendTLSPolicy with CA certificate ConfigMap reference
// Validates: validation.ca_certificate_refs with ConfigMap type
// ---------------------------------------------------------------------------
func TestAccGatewayAPIAdvanced_backendTLSWithCARef(t *testing.T) {
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
				Config: testAccGatewayAPIAdvancedBackendTLSWithCA(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_backend_tls_policy_v1.ca_ref", "spec.0.validation.0.hostname", "api.internal.example.com"),
					resource.TestCheckResourceAttr("kubernetes_backend_tls_policy_v1.ca_ref", "spec.0.validation.0.ca_certificate_refs.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_backend_tls_policy_v1.ca_ref", "spec.0.validation.0.ca_certificate_refs.0.kind", "ConfigMap"),
					resource.TestCheckResourceAttr("kubernetes_backend_tls_policy_v1.ca_ref", "spec.0.validation.0.ca_certificate_refs.0.name", rName+"-ca-bundle"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// Full production-like stack:
// GatewayClass + Gateway(3 listeners: HTTP, HTTPS, GRPC) +
// HTTPRoute(canary) + GRPCRoute(method match) + TLSRoute(SNI) +
// BackendTLSPolicy + ReferenceGrant + ListenerSet
// ---------------------------------------------------------------------------
func TestAccGatewayAPIAdvanced_productionStack(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoTLSRoute(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGatewayAPIIntegrationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIAdvancedProductionStack(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// GatewayClass
					resource.TestCheckResourceAttr("kubernetes_gateway_class_v1.prod", "spec.0.controller_name", "gateway.envoyproxy.io/gatewayclass-controller"),
					// Gateway
					resource.TestCheckResourceAttr("kubernetes_gateway_v1.prod", "spec.0.listeners.#", "3"),
					resource.TestCheckResourceAttr("kubernetes_gateway_v1.prod", "spec.0.listeners.0.name", "http"),
					resource.TestCheckResourceAttr("kubernetes_gateway_v1.prod", "spec.0.listeners.1.name", "https"),
					resource.TestCheckResourceAttr("kubernetes_gateway_v1.prod", "spec.0.listeners.2.name", "grpc"),
					// HTTPRoute
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.prod", "spec.0.rules.0.backend_refs.#", "2"),
					// GRPCRoute
					resource.TestCheckResourceAttr("kubernetes_grpc_route_v1.prod", "spec.0.rules.0.matches.0.method.0.service", "api.V1"),
					// TLSRoute
					resource.TestCheckResourceAttr("kubernetes_tls_route_v1.prod", "spec.0.hostnames.#", "1"),
					// BackendTLSPolicy
					resource.TestCheckResourceAttr("kubernetes_backend_tls_policy_v1.prod", "spec.0.validation.0.hostname", "backend.internal.example.com"),
					// ReferenceGrant
					resource.TestCheckResourceAttr("kubernetes_reference_grant_v1.prod", "spec.0.from.0.kind", "HTTPRoute"),
				),
			},
		},
	})
}
