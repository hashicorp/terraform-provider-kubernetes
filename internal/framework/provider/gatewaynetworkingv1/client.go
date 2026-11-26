// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package gatewaynetworkingv1

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	"k8s.io/client-go/discovery"
	gatewaytyped "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/typed/apis/v1"
)

// apiVersion is the group/version this package serves. Identity, import
// validation and the discovery check all read it, so they cannot disagree.
const apiVersion = "gateway.networking.k8s.io/v1"

type clientProvider struct {
	meta func() any
}

// configure is the shared body of Configure. It stores the callback without
// calling it. The framework also calls Configure before the provider has been
// configured, with nil ProviderData, so nil is not an error; any other
// unexpected type is a diagnostic rather than a panic.
func (p *clientProvider) configure(data any) diag.Diagnostics {
	var diags diag.Diagnostics
	if data == nil {
		return diags
	}
	callback, ok := data.(func() any)
	if !ok {
		diags.AddError("Unexpected provider data", fmt.Sprintf("Expected func() any, got %T.", data))
		return diags
	}
	p.meta = callback
	return diags
}

// clients resolves what an operation needs from the SDKv2 meta: the
// GatewayClass client, a discovery client for gatewayClassEndpoint, and the
// provider's ignore_annotations and ignore_labels lists. It runs per request
// rather than in Configure because the mux configures the SDKv2 provider
// separately, so its meta is only reliable once an operation is running.
//
// Each capability is a separate interface, asserted with a check, so an
// unexpected meta is a diagnostic rather than a panic that ends the plugin
// process. Clients are nil-checked as well: DiscoveryClient returns nil without
// an error when the provider has no REST config.
func (p *clientProvider) clients() (gatewaytyped.GatewayClassInterface, discovery.DiscoveryInterface, kubernetes.MetadataFilters, diag.Diagnostics) {
	var diags diag.Diagnostics
	if p.meta == nil {
		diags.AddError("Provider not configured", "Kubernetes provider metadata is unavailable.")
		return nil, nil, nil, diags
	}
	meta := p.meta()
	gateway, ok := meta.(kubernetes.GatewayClientsets)
	if !ok {
		diags.AddError("Unexpected provider data", fmt.Sprintf("Expected Gateway client metadata, got %T.", meta))
		return nil, nil, nil, diags
	}
	clients, ok := meta.(kubernetes.KubeClientsets)
	if !ok {
		diags.AddError("Unexpected provider data", "Kubernetes discovery client metadata is unavailable.")
		return nil, nil, nil, diags
	}
	filters, ok := meta.(kubernetes.MetadataFilters)
	if !ok {
		diags.AddError("Unexpected provider data", "Kubernetes metadata filters are unavailable.")
		return nil, nil, nil, diags
	}
	conn, err := gateway.GatewayClientset()
	if err != nil {
		diags.AddError("Gateway client error", err.Error())
		return nil, nil, nil, diags
	}
	if conn == nil {
		diags.AddError("Gateway client error", "The configured Gateway client is nil.")
		return nil, nil, nil, diags
	}
	discover, err := clients.DiscoveryClient()
	if err != nil {
		diags.AddError("Kubernetes discovery error", err.Error())
		return nil, nil, nil, diags
	}
	if discover == nil {
		diags.AddError("Kubernetes discovery error", "The configured discovery client is nil.")
		return nil, nil, nil, diags
	}
	return conn.GatewayV1().GatewayClasses(), discover, filters, diags
}

// gatewayClassEndpoint uses discovery to confirm that the cluster serves
// gatewayclasses in apiVersion. Callers run it after a NotFound, because
// apierrors.IsNotFound is true both for a missing object and for a cluster
// without the Gateway API CRDs. A discovery failure is returned as an error,
// so callers fail instead of guessing. A 404 from an unavailable CRD must not
// remove an existing object from state.
func gatewayClassEndpoint(_ context.Context, discover discovery.DiscoveryInterface) error {
	resources, err := discover.ServerResourcesForGroupVersion(apiVersion)
	if err != nil {
		return fmt.Errorf("cannot verify the GatewayClass API endpoint; install compatible Gateway API CRDs and check discovery permissions: %w", err)
	}
	for _, resource := range resources.APIResources {
		if resource.Name == "gatewayclasses" {
			return nil
		}
	}
	return fmt.Errorf("GatewayClass %s is not served; install compatible Gateway API CRDs", apiVersion)
}
