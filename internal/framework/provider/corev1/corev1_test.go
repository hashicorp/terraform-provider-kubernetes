// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	sdkv2 "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

// sdkv2providerMeta configures and returns the SDKv2 provider's Meta function.
// The framework provider delegates Kubernetes client initialisation to the
// SDKv2 codebase during the mux migration period, so every acceptance test
// must configure both sides.
func sdkv2providerMeta() func() any {
	p := kubernetes.Provider()
	p.Configure(context.Background(), sdkv2.NewResourceConfigRaw(nil))
	return p.Meta
}

// testAccProtoV6ProviderFactories is the framework equivalent of the SDKv2
// testAccProviderFactories in kubernetes/provider_test.go. Use this in any
// resource.TestCase that exercises framework-migrated types.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": providerserver.NewProtocol6WithError(provider.New("test", sdkv2providerMeta())),
}
