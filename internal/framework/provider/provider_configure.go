// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/provider"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func (p *KubernetesProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Deferred actions: when the provider configuration still contains unknown values —
	// typically because the cluster it points at is created in the same apply — Terraform
	// asks the provider to defer its resources rather than fail. This mirrors what the
	// SDKv2 provider already does in ConfigureProvider (kubernetes/provider.go); without
	// it every resource served from here reports "No deferred changes found".
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
