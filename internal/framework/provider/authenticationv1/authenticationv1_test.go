// Copyright (c) HashiCorp, Inc.

package authenticationv1_test

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/hashicorp/terraform-plugin-testing/echoprovider"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	sdkv2 "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// NOTE this is a shim back to the SDKv2 so we don't have to duplicate
// the client initialization code.
func sdkv2providerMeta() func() any {
	p := kubernetes.Provider()
	p.Configure(context.Background(), sdkv2.NewResourceConfigRaw(nil))
	return p.Meta
}

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": providerserver.NewProtocol6WithError(provider.New("test", sdkv2providerMeta())),
	"echo":       echoprovider.NewProviderServer(),
}
