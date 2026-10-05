// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package nodev1_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	sdkv2 "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	nodev1 "k8s.io/api/node/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
)

func sdkv2providerMeta() func() any {
	p := kubernetes.Provider()
	p.Configure(context.Background(), sdkv2.NewResourceConfigRaw(nil))
	return p.Meta
}

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": providerserver.NewProtocol6WithError(provider.New("test", sdkv2providerMeta())),
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("KUBE_CONFIG_PATH") == "" && os.Getenv("KUBE_CTX") == "" &&
		(os.Getenv("KUBE_HOST") == "" || os.Getenv("KUBE_CLUSTER_CA_CERT_DATA") == "") {
		t.Fatal("KUBE_CONFIG_PATH, KUBE_CTX, or KUBE_HOST with KUBE_CLUSTER_CA_CERT_DATA must be set for acceptance tests")
	}
	if diags := kubernetes.Provider().Configure(context.Background(), sdkv2.NewResourceConfigRaw(nil)); diags.HasError() {
		t.Fatal(diags[0].Summary)
	}
}

// testAccClientset builds a clientset from the same environment the provider under test
// uses, so checks observe the live object rather than Terraform state.
func testAccClientset() (k8sclient.Interface, error) {
	meta := sdkv2providerMeta()()
	clients, ok := meta.(kubernetes.KubeClientsets)
	if !ok {
		return nil, fmt.Errorf("unexpected provider meta %T", meta)
	}
	return clients.MainClientset()
}

func testAccCheckRuntimeClassV1Destroy(s *terraform.State) error {
	conn, err := testAccClientset()
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_runtime_class_v1" {
			continue
		}
		_, err := conn.NodeV1().RuntimeClasses().Get(context.Background(), rs.Primary.ID, metav1.GetOptions{})
		if err == nil {
			return fmt.Errorf("runtime class still exists: %s", rs.Primary.ID)
		}
		if !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func testAccCheckRuntimeClassV1Exists(addr string, obj *nodev1.RuntimeClass) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[addr]
		if !ok {
			return fmt.Errorf("not found in state: %s", addr)
		}
		conn, err := testAccClientset()
		if err != nil {
			return err
		}
		out, err := conn.NodeV1().RuntimeClasses().Get(context.Background(), rs.Primary.ID, metav1.GetOptions{})
		if err != nil {
			return err
		}
		*obj = *out
		return nil
	}
}
