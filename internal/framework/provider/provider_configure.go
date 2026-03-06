// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/provider"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func (p *KubernetesProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// NOTE for migration purposes we are re-using the client configurations which are initialized at configure time
	// by the SDKv2 codebase. Once all SDKv2 resources have been removed the client initialization code should be
	// migrated here.

	resp.ResourceData = p.SDKv2Meta
	resp.DataSourceData = p.SDKv2Meta
	resp.EphemeralResourceData = p.SDKv2Meta
}
