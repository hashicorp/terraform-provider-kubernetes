// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestpatch_test

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	sdkv2 "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// sdkv2providerMeta configures the SDKv2 provider and returns its Meta func, which the
// Framework server reuses as the shared Kubernetes client.
func sdkv2providerMeta() func() any {
	p := kubernetes.Provider()
	p.Configure(context.Background(), sdkv2.NewResourceConfigRaw(nil))
	return p.Meta
}

// testAccProtoV6ProviderFactories serves the Framework provider (which registers
// kubernetes_manifest_patch) over protocol v6, wired to the shared SDKv2 meta.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": providerserver.NewProtocol6WithError(provider.New("test", sdkv2providerMeta())),
}

// testAccClients returns the shared Kubernetes client set for target setup/assertions.
func testAccClients() kubernetes.KubeClientsets {
	return sdkv2providerMeta()().(kubernetes.KubeClientsets)
}
