// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/provider"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func (p *KubernetesProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Deferred actions are handled centrally by the SDKv2 server
	// (kubernetes/provider.go, ConfigureProvider) so that a provider configured
	// from another resource's not-yet-known outputs — the cluster-then-workloads
	// pattern in _examples/deferred-actions — defers instead of planning against
	// an unknown endpoint. No Framework resource needed that until now; moving
	// one here without the same gate would drop it off the deferral path
	// silently, which is exactly the kind of unannounced behaviour change a
	// migration must not smuggle in (rule K8S-MIGRATE-022).
	if req.ClientCapabilities.DeferralAllowed && !req.Config.Raw.IsFullyKnown() {
		resp.Deferred = &provider.Deferred{
			Reason: provider.DeferredReasonProviderConfigUnknown,
		}
	}

	// NOTE for migration purposes we are re-using the client configurations which are initialized at configure time
	// by the SDKv2 codebase. Once all SDKv2 resources have been removed the client initialization code should be
	// migrated here.

	resp.ResourceData = p.SDKv2Meta
	resp.DataSourceData = p.SDKv2Meta
	resp.EphemeralResourceData = p.SDKv2Meta
}
