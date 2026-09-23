// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"

	framework "github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	manifest "github.com/hashicorp/terraform-provider-kubernetes/manifest/provider"

	sdkv2 "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// testAccProtoV6ProviderFactories is the provider factory map passed to every
// resource.TestCase. It builds the full mux stack (SDKv2 + manifest + Framework)
// with the SDKv2 provider pre-configured from environment variables
// (e.g. KUBE_CONFIG_PATH) so that tests can reach the cluster.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": func() (tfprotov6.ProviderServer, error) {
		ctx := context.Background()

		p := kubernetes.Provider()
		p.Configure(ctx, sdkv2.NewResourceConfigRaw(nil))

		upgradedSdk, err := tf5to6server.UpgradeServer(ctx, p.GRPCProvider)
		if err != nil {
			return nil, err
		}

		upgradedManifest, err := tf5to6server.UpgradeServer(ctx, manifest.Provider())
		if err != nil {
			return nil, err
		}

		return tf6muxserver.NewMuxServer(ctx,
			func() tfprotov6.ProviderServer { return upgradedSdk },
			func() tfprotov6.ProviderServer { return upgradedManifest },
			providerserver.NewProtocol6(framework.New("test", p.Meta)),
		)
	},
}
