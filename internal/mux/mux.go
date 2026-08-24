// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package mux

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	framework "github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	manifest "github.com/hashicorp/terraform-provider-kubernetes/manifest/provider"
)

func MuxServer(ctx context.Context, v string) (tfprotov6.ProviderServer, error) {
	return MuxServerWithProvider(ctx, v, kubernetes.Provider())
}

// MuxServerWithProvider assembles the mux server using a caller-supplied
// SDKv2 provider. Used by main.go via MuxServer and directly by acceptance
// tests that need to supply their own pre-configured provider instance.
func MuxServerWithProvider(ctx context.Context, v string, kubernetesProvider *schema.Provider) (tfprotov6.ProviderServer, error) {
	upgradedSdkProvider, err := tf5to6server.UpgradeServer(
		ctx,
		kubernetesProvider.GRPCProvider,
	)
	if err != nil {
		return nil, err
	}

	upgradedManifestProvider, err := tf5to6server.UpgradeServer(
		ctx,
		manifest.Provider(),
	)
	if err != nil {
		return nil, err
	}

	providers := []func() tfprotov6.ProviderServer{
		func() tfprotov6.ProviderServer { return upgradedSdkProvider },
		func() tfprotov6.ProviderServer { return upgradedManifestProvider },
		providerserver.NewProtocol6(framework.New(v, kubernetesProvider.Meta)),
	}

	return tf6muxserver.NewMuxServer(ctx, providers...)
}
