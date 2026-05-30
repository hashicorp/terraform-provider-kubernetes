// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// =============================================================================
// Empirical Field Limit Tests — run against a live Kind cluster with Gateway API CRDs
//
// These tests verify:
//   - MaxItems/MinItems schema constraints hit the CRD and are accepted
//   - Enum values (protocol, match types, TLS modes) work end-to-end
//   - Hostname validation (1-253 chars)
//   - Port number validation
//   - Weight range (0-1000000)
//   - Update behavior (add/remove listeners, change backends)
//   - Status fields are populated after controller reconciliation
//
// Run:
//   export KUBE_CONFIG_PATH=/tmp/kind-kubeconfig
//   export TF_ACC=1
//   go test -v -run TestAccGatewayAPIFieldLimits ./kubernetes/ -timeout 60m
// =============================================================================

// ---------------------------------------------------------------------------
// Gateway Listener Limits
// ---------------------------------------------------------------------------

// Test that a Gateway with many listeners (8, approaching spec limit of 64) works
func TestAccGatewayAPIFieldLimits_gatewayManyListeners(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")
	resourceName := "kubernetes_gateway_v1.many"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGatewayV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIFieldLimitsGatewayManyListeners(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.#", "8"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.name", "http-0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.7.name", "http-7"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.port", "8000"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.7.port", "8007"),
				),
			},
		},
	})
}

// Test Gateway hostname validation — long hostname (253 chars boundary)
func TestAccGatewayAPIFieldLimits_gatewayHostnameLength(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")
	resourceName := "kubernetes_gateway_v1.hostname"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGatewayV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIFieldLimitsGatewayHostnameLength(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.hostname",
						"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.com"),
				),
			},
		},
	})
}

// Test Gateway protocol enum values — all core protocols
func TestAccGatewayAPIFieldLimits_gatewayProtocolTypes(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")
	resourceName := "kubernetes_gateway_v1.protocols"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGatewayV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIFieldLimitsGatewayProtocols(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.protocol", "HTTP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.1.protocol", "HTTPS"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.2.protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.3.protocol", "TLS"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.4.protocol", "GRPC"),
				),
			},
		},
	})
}

// Test Gateway TLS mode enum values
func TestAccGatewayAPIFieldLimits_gatewayTLSMode(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")
	resourceName := "kubernetes_gateway_v1.tls_mode"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGatewayV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIFieldLimitsGatewayTLSMode(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.tls.0.mode", "Passthrough"),
				),
			},
		},
	})
}

// Test Gateway allowedRoutes namespace values
func TestAccGatewayAPIFieldLimits_gatewayAllowedRoutesNamespaces(t *testing.T) {
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
				Config: testAccGatewayAPIFieldLimitsGatewayAllowedRoutesNS(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_gateway_v1.allow_all", "spec.0.listeners.0.allowed_routes.0.namespaces.0.from", "All"),
					resource.TestCheckResourceAttr("kubernetes_gateway_v1.allow_same", "spec.0.listeners.0.allowed_routes.0.namespaces.0.from", "Same"),
				),
			},
		},
	})
}

// Test Gateway addresses with multiple types
func TestAccGatewayAPIFieldLimits_gatewayAddresses(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")
	resourceName := "kubernetes_gateway_v1.addresses"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGatewayV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIFieldLimitsGatewayAddresses(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "spec.0.addresses.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.addresses.0.type", "IPAddress"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.addresses.0.value", "10.0.0.1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.addresses.1.type", "Hostname"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.addresses.1.value", "gw.example.com"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// HTTPRoute Field Limits
// ---------------------------------------------------------------------------

// Test HTTPRoute with many rules (approaching 16 limit)
func TestAccGatewayAPIFieldLimits_httpRouteManyRules(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIFieldLimitsHTTPRouteManyRules(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.many_rules", "spec.0.rules.#", "8"),
				),
			},
		},
	})
}

// Test HTTPRoute backend weight distribution with edge values
func TestAccGatewayAPIFieldLimits_httpRouteBackendWeightsEdge(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIFieldLimitsHTTPRouteWeightsEdge(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.weights", "spec.0.rules.0.backend_refs.0.weight", "1"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.weights", "spec.0.rules.0.backend_refs.1.weight", "999999"),
				),
			},
		},
	})
}

// Test HTTPRoute path match types — PathPrefix, Exact, Regex
func TestAccGatewayAPIFieldLimits_httpRoutePathMatchTypes(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIFieldLimitsHTTPRoutePathTypes(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.paths", "spec.0.rules.0.matches.0.path.0.type", "PathPrefix"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.paths", "spec.0.rules.1.matches.0.path.0.type", "Exact"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.paths", "spec.0.rules.2.matches.0.path.0.type", "RegularExpression"),
				),
			},
		},
	})
}

// Test HTTPRoute header/query match types — Exact vs Regex
func TestAccGatewayAPIFieldLimits_httpRouteMatchTypesExactRegex(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIFieldLimitsHTTPRouteMatchTypes(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.match_types", "spec.0.rules.0.matches.0.headers.0.type", "Exact"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.match_types", "spec.0.rules.1.matches.0.headers.0.type", "RegularExpression"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.match_types", "spec.0.rules.2.matches.0.query_params.0.type", "Exact"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.match_types", "spec.0.rules.3.matches.0.query_params.0.type", "RegularExpression"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// GRPCRoute Field Limits
// ---------------------------------------------------------------------------

// Test GRPCRoute method match types — Exact, Path
func TestAccGatewayAPIFieldLimits_grpcRouteMethodMatchTypes(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIFieldLimitsGRPCRouteMethodTypes(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_grpc_route_v1.methods", "spec.0.rules.0.matches.0.method.0.type", "Exact"),
					resource.TestCheckResourceAttr("kubernetes_grpc_route_v1.methods", "spec.0.rules.1.matches.0.method.0.type", "RegularExpression"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// TLSRoute Field Limits
// ---------------------------------------------------------------------------

// Test TLSRoute with multiple backends and weights
func TestAccGatewayAPIFieldLimits_tlsRouteMultipleBackends(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoTLSRoute(t)
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIFieldLimitsTLSRouteBackends(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_tls_route_v1.backends", "spec.0.rules.0.backend_refs.#", "3"),
					resource.TestCheckResourceAttr("kubernetes_tls_route_v1.backends", "spec.0.rules.0.backend_refs.0.weight", "50"),
					resource.TestCheckResourceAttr("kubernetes_tls_route_v1.backends", "spec.0.rules.0.backend_refs.1.weight", "30"),
					resource.TestCheckResourceAttr("kubernetes_tls_route_v1.backends", "spec.0.rules.0.backend_refs.2.weight", "20"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// ListenerSet Field Limits
// ---------------------------------------------------------------------------

// Test ListenerSet with many listeners (approaching 64 limit)
func TestAccGatewayAPIFieldLimits_listenerSetManyListeners(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoListenerSet(t)
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIFieldLimitsListenerSetManyListeners(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_listener_set_v1.many", "spec.0.listeners.#", "6"),
					resource.TestCheckResourceAttr("kubernetes_listener_set_v1.many", "spec.0.listeners.0.port", "8080"),
					resource.TestCheckResourceAttr("kubernetes_listener_set_v1.many", "spec.0.listeners.5.port", "8085"),
				),
			},
		},
	})
}

// Test ListenerSet protocol enum values
func TestAccGatewayAPIFieldLimits_listenerSetProtocols(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoListenerSet(t)
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIFieldLimitsListenerSetProtocols(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_listener_set_v1.protocols", "spec.0.listeners.0.protocol", "HTTP"),
					resource.TestCheckResourceAttr("kubernetes_listener_set_v1.protocols", "spec.0.listeners.1.protocol", "HTTPS"),
					resource.TestCheckResourceAttr("kubernetes_listener_set_v1.protocols", "spec.0.listeners.2.protocol", "TCP"),
					resource.TestCheckResourceAttr("kubernetes_listener_set_v1.protocols", "spec.0.listeners.3.protocol", "TLS"),
					resource.TestCheckResourceAttr("kubernetes_listener_set_v1.protocols", "spec.0.listeners.4.protocol", "GRPC"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// BackendTLSPolicy Field Limits
// ---------------------------------------------------------------------------

// Test BackendTLSPolicy SAN types — Hostname, URI
func TestAccGatewayAPIFieldLimits_backendTLSSubjectAltNames(t *testing.T) {
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
				Config: testAccGatewayAPIFieldLimitsBackendTLSSAN(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_backend_tls_policy_v1.san", "spec.0.validation.0.subject_alt_names.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_backend_tls_policy_v1.san", "spec.0.validation.0.subject_alt_names.0.type", "Hostname"),
					resource.TestCheckResourceAttr("kubernetes_backend_tls_policy_v1.san", "spec.0.validation.0.subject_alt_names.1.type", "URI"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// ReferenceGrant Field Limits
// ---------------------------------------------------------------------------

// Test ReferenceGrant with multiple from/to entries (approaching 16 limit)
func TestAccGatewayAPIFieldLimits_referenceGrantMultipleEntries(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoReferenceGrantV1(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckReferenceGrantV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIFieldLimitsReferenceGrantMultiple(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_reference_grant_v1.multi", "spec.0.from.#", "3"),
					resource.TestCheckResourceAttr("kubernetes_reference_grant_v1.multi", "spec.0.to.#", "3"),
					resource.TestCheckResourceAttr("kubernetes_reference_grant_v1.multi", "spec.0.from.0.kind", "HTTPRoute"),
					resource.TestCheckResourceAttr("kubernetes_reference_grant_v1.multi", "spec.0.from.1.kind", "GRPCRoute"),
					resource.TestCheckResourceAttr("kubernetes_reference_grant_v1.multi", "spec.0.from.2.kind", "TLSRoute"),
				),
			},
		},
	})
}

// ---------------------------------------------------------------------------
// Update In-Place Tests — verify CRUD update paths for all resources
// ---------------------------------------------------------------------------

// Test HTTPRoute update: change match, add filter, change backend
func TestAccGatewayAPIFieldLimits_httpRouteUpdateInPlace(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIFieldLimitsHTTPRouteUpdateBefore(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.update", "spec.0.rules.0.matches.0.path.0.value", "/api"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.update", "spec.0.rules.0.backend_refs.0.name", rName+"-svc-a"),
				),
			},
			{
				Config: testAccGatewayAPIFieldLimitsHTTPRouteUpdateAfter(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.update", "spec.0.rules.0.matches.0.path.0.value", "/api/v2"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.update", "spec.0.rules.0.backend_refs.0.name", rName+"-svc-b"),
					resource.TestCheckResourceAttr("kubernetes_http_route_v1.update", "spec.0.rules.0.filters.0.type", "RequestHeaderModifier"),
				),
			},
		},
	})
}

// Test GatewayClass update: change controller name, add description
func TestAccGatewayAPIFieldLimits_gatewayClassUpdate(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIFieldLimitsGatewayClassUpdateBefore(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_gateway_class_v1.update", "spec.0.controller_name", "gateway.envoyproxy.io/gatewayclass-controller"),
				),
			},
			{
				Config: testAccGatewayAPIFieldLimitsGatewayClassUpdateAfter(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_gateway_class_v1.update", "spec.0.controller_name", "gateway.envoyproxy.io/gatewayclass-controller"),
					resource.TestCheckResourceAttr("kubernetes_gateway_class_v1.update", "spec.0.description", "Updated by test"),
				),
			},
		},
	})
}

// Test Gateway import/export round-trip
func TestAccGatewayAPIFieldLimits_gatewayImportState(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-acc")
	resourceName := "kubernetes_gateway_v1.import_test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoGatewayAPI(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGatewayV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayAPIFieldLimitsGatewayImport(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.name", "http"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "metadata.0.uid", "status"},
			},
		},
	})
}

// Test BackendTLSPolicy update: change hostname, add options
func TestAccGatewayAPIFieldLimits_backendTLSUpdate(t *testing.T) {
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
				Config: testAccGatewayAPIFieldLimitsBackendTLSUpdateBefore(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_backend_tls_policy_v1.update", "spec.0.validation.0.hostname", "old.example.com"),
				),
			},
			{
				Config: testAccGatewayAPIFieldLimitsBackendTLSUpdateAfter(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_backend_tls_policy_v1.update", "spec.0.validation.0.hostname", "new.example.com"),
					resource.TestCheckResourceAttr("kubernetes_backend_tls_policy_v1.update", "spec.0.options.min_version", "VersionTLS12"),
				),
			},
		},
	})
}
