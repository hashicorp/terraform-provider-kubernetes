// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1_test

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	sdkv2 "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func sdkv2providerMeta() (func() any, error) {
	p := kubernetes.Provider()
	if diags := p.Configure(context.Background(), sdkv2.NewResourceConfigRaw(nil)); diags.HasError() {
		return nil, fmt.Errorf("configuring Kubernetes provider: %v", diags)
	}
	return p.Meta, nil
}

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": func() (tfprotov6.ProviderServer, error) {
		meta, err := sdkv2providerMeta()
		if err != nil {
			return nil, err
		}
		return providerserver.NewProtocol6WithError(provider.New("test", meta))()
	},
}

var testAccMuxProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": func() (tfprotov6.ProviderServer, error) {
		return mux.MuxServer(context.Background(), "test")
	},
}

func testAccKubernetesClusterRoleBindingV1Destroy(s *terraform.State) error {
	meta, err := sdkv2providerMeta()
	if err != nil {
		return err
	}
	conn, err := meta().(kubernetes.KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_cluster_role_binding_v1" {
			continue
		}
		_, err := conn.RbacV1().ClusterRoleBindings().Get(ctx, rs.Primary.ID, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return err
		}
		return fmt.Errorf("ClusterRoleBinding still exists: %s", rs.Primary.ID)
	}
	return nil
}
