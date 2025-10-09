package admissionregistrationv1_test

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	sdkv2 "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func sdkv2providerMeta() func() any {
	p := kubernetes.Provider()
	p.Configure(context.Background(), sdkv2.NewResourceConfigRaw(nil))
	return p.Meta
}

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": providerserver.NewProtocol6WithError(provider.New("test", sdkv2providerMeta())),
}
