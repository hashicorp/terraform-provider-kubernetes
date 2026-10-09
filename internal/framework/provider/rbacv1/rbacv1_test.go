// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1_test

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	sdkv2 "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

// sdkv2providerMeta returns the SDKv2 provider meta (KubeClientsets) for use
// by the Framework provider in tests. The SDKv2 provider schema DefaultFuncs
// automatically read KUBE_CONFIG_PATH and KUBE_CTX from the environment, so
// no explicit config map is needed.
func sdkv2providerMeta() func() any {
	p := kubernetes.Provider()
	p.Configure(context.Background(), sdkv2.NewResourceConfigRaw(nil))
	return p.Meta
}

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": providerserver.NewProtocol6WithError(provider.New("test", sdkv2providerMeta())),
}
