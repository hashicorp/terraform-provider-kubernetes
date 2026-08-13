// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1_test

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	sdkv2 "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

// testAccProtoV6ProviderFactories muxes the SDKv2 provider (which owns
// resources like kubernetes_deployment_v1) together with the framework
// provider (which owns the restart actions), since these tests exercise a
// resource and an action, wired together with a lifecycle action_trigger,
// in the same configuration.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": func() (tfprotov6.ProviderServer, error) {
		return mux.MuxServer(context.Background(), "test")
	},
}

// testAccAppsV1Clientset returns a real Kubernetes clientset, configured
// the same way the provider itself configures one, so tests can inspect
// cluster state directly to confirm a restart action actually ran.
func testAccAppsV1Clientset() (kubernetes.KubeClientsets, error) {
	p := kubernetes.Provider()
	diags := p.Configure(context.Background(), sdkv2.NewResourceConfigRaw(nil))
	if diags.HasError() {
		return nil, fmt.Errorf("failed to configure kubernetes provider: %v", diags)
	}
	return p.Meta().(kubernetes.KubeClientsets), nil
}
