// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1_test

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	sdkv2 "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	client "k8s.io/client-go/kubernetes"
)

func testAccRoleClient() (*client.Clientset, error) {
	p := kubernetes.Provider()
	if diags := p.Configure(context.Background(), sdkv2.NewResourceConfigRaw(nil)); diags.HasError() {
		return nil, fmt.Errorf("configuring Kubernetes provider: %v", diags)
	}
	return p.Meta().(kubernetes.KubeClientsets).MainClientset()
}

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": func() (tfprotov6.ProviderServer, error) {
		return mux.MuxServer(context.Background(), "test")
	},
}
