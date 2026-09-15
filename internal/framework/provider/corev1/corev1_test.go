// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	sdkv2 "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	clientset "k8s.io/client-go/kubernetes"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
	provkubernetes "github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

// providerVersion is the exact released version the migration tests upgrade
// FROM. It must stay an exact version, never a constraint: an operator lets the
// test resolve whatever the registry serves that day, so what CI proved would
// not be what anyone thinks it proved (rule K8S-MIGRATE-017).
const providerVersion = "3.2.1"

// testAccProtoV6ProviderFactories serves the production mux rather than the
// Framework server alone (rule K8S-MIGRATE-019).
//
// This package's configurations reference types that are still SDKv2 — the
// deprecated kubernetes_namespace alias and the kubernetes_namespace_v1 DATA
// source, neither of which moved — and a Framework-only server cannot serve
// them, so such a step could not even init. Serving the mux also means these
// tests exercise the same server composition that ships.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": func() (tfprotov6.ProviderServer, error) {
		return mux.MuxServer(context.Background(), "test")
	},
}

// testAccPreCheck mirrors the SDKv2 package's precheck. It is duplicated rather
// than shared because that one lives in package kubernetes' internal test
// scope and is not importable from here.
func testAccPreCheck(t *testing.T) {
	t.Helper()

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
}

// testAccDestroyClientset builds the same clientset the provider would, so the
// CheckDestroy and Exists helpers talk to the cluster under test. It returns an
// error rather than taking *testing.T because CheckDestroy has no T.
func testAccDestroyClientset() (*clientset.Clientset, error) {
	p := provkubernetes.Provider()
	if diags := p.Configure(context.Background(), sdkv2.NewResourceConfigRaw(nil)); diags.HasError() {
		return nil, fmt.Errorf("configuring the provider for test checks: %s", diags[0].Summary)
	}
	return p.Meta().(provkubernetes.KubeClientsets).MainClientset()
}
