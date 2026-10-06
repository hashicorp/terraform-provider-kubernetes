// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	sdkv2 "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

func sdkv2providerMeta() func() any {
	p := kubernetes.Provider()
	p.Configure(context.Background(), sdkv2.NewResourceConfigRaw(nil))
	return p.Meta
}

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": func() (tfprotov6.ProviderServer, error) {
		p := kubernetes.Provider()
		diags := p.Configure(context.Background(), sdkv2.NewResourceConfigRaw(nil))
		if diags.HasError() {
			return nil, fmt.Errorf("provider configuration error: %v", diags)
		}
		return mux.MuxServerWithProvider(context.Background(), "test", p)
	},
}
