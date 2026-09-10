// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1_test

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	sdkv2 "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// testAccProtoV6ProviderFactories serves the full muxed provider (SDKv2 +
// Plugin Framework + manifest). Because kubernetes_cluster_role_binding_v1 is
// now registered on the Framework provider, these factories exercise the
// migrated Framework implementation.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": func() (tfprotov6.ProviderServer, error) {
		return mux.MuxServer(context.Background(), "test")
	},
}

// sdkv2providerMeta configures an SDKv2 provider from the environment and
// returns its meta accessor, used only to obtain a Kubernetes client for the
// acceptance-test destroy/exists checks.
func sdkv2providerMeta() func() any {
	p := kubernetes.Provider()
	p.Configure(context.Background(), sdkv2.NewResourceConfigRaw(nil))
	return p.Meta
}

func testAccKubernetesClusterRoleBindingV1Destroy(s *terraform.State) error {
	conn, err := sdkv2providerMeta()().(kubernetes.KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_cluster_role_binding_v1" {
			continue
		}
		name := rs.Primary.ID
		resp, err := conn.RbacV1().ClusterRoleBindings().Get(ctx, name, metav1.GetOptions{})
		if err == nil && resp.Name == rs.Primary.ID {
			return fmt.Errorf("ClusterRoleBinding still exists: %s", rs.Primary.ID)
		}
	}
	return nil
}
