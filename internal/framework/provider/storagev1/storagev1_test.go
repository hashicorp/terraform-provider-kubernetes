// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package storagev1_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	gversion "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	sdkv2 "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	k8sclient "k8s.io/client-go/kubernetes"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

// The muxed server is the provider exactly as users run it, so these tests also catch a
// type registered on both the SDKv2 and the framework server.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": func() (tfprotov6.ProviderServer, error) {
		return mux.MuxServer(context.Background(), "test")
	},
}

// testAccPreCheck fails fast when no cluster configuration is set, and checks the SDKv2
// provider can be configured, as in the corev1 and rbacv1 packages.
func testAccPreCheck(t *testing.T) {
	hasFileCfg := (os.Getenv("KUBE_CTX_AUTH_INFO") != "" && os.Getenv("KUBE_CTX_CLUSTER") != "") ||
		os.Getenv("KUBE_CTX") != "" ||
		os.Getenv("KUBE_CONFIG_PATH") != ""
	hasUserCredentials := os.Getenv("KUBE_USER") != "" && os.Getenv("KUBE_PASSWORD") != ""
	hasClientCert := os.Getenv("KUBE_CLIENT_CERT_DATA") != "" && os.Getenv("KUBE_CLIENT_KEY_DATA") != ""
	hasStaticCfg := (os.Getenv("KUBE_HOST") != "" &&
		os.Getenv("KUBE_CLUSTER_CA_CERT_DATA") != "") &&
		(hasUserCredentials || hasClientCert || os.Getenv("KUBE_TOKEN") != "")

	if !hasFileCfg && !hasStaticCfg && !hasUserCredentials {
		t.Fatalf("File config (KUBE_CTX_AUTH_INFO and KUBE_CTX_CLUSTER) or static configuration"+
			"(%s) or (%s) must be set for acceptance tests",
			strings.Join([]string{"KUBE_HOST", "KUBE_USER", "KUBE_PASSWORD", "KUBE_CLUSTER_CA_CERT_DATA"}, ", "),
			strings.Join([]string{"KUBE_HOST", "KUBE_CLIENT_CERT_DATA", "KUBE_CLIENT_KEY_DATA", "KUBE_CLUSTER_CA_CERT_DATA"}, ", "),
		)
	}

	if _, err := testAccClientset(); err != nil {
		t.Fatal(err)
	}
}

// testAccClientset returns a client configured the same way as the provider under test,
// for checking the live object independently of Terraform state.
func testAccClientset() (*k8sclient.Clientset, error) {
	p := kubernetes.Provider()
	if diags := p.Configure(context.Background(), sdkv2.NewResourceConfigRaw(nil)); diags.HasError() {
		return nil, fmt.Errorf("configuring Kubernetes provider: %v", diags)
	}
	return p.Meta().(kubernetes.KubeClientsets).MainClientset()
}

// testAccClusterVersionAtLeast reports whether the test cluster runs at least version.
func testAccClusterVersionAtLeast(t *testing.T, version string) bool {
	t.Helper()
	conn, err := testAccClientset()
	if err != nil {
		t.Fatal(err)
	}
	serverVersion, err := conn.ServerVersion()
	if err != nil {
		t.Fatal(err)
	}
	clusterVersion, err := gversion.NewVersion(serverVersion.GitVersion)
	if err != nil {
		t.Fatal(err)
	}
	return clusterVersion.GreaterThanOrEqual(gversion.Must(gversion.NewVersion(version)))
}
