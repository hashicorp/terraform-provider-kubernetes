// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1_test

import (
	"context"
	"os"
	"strings"
	"testing"

	sdkv2 "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

func testAccPreCheck(t *testing.T) {
	t.Helper()
	hasFileCfg := (os.Getenv("KUBE_CTX_AUTH_INFO") != "" && os.Getenv("KUBE_CTX_CLUSTER") != "") ||
		os.Getenv("KUBE_CTX") != "" || os.Getenv("KUBE_CONFIG_PATH") != ""
	hasUserCredentials := os.Getenv("KUBE_USER") != "" && os.Getenv("KUBE_PASSWORD") != ""
	hasClientCert := os.Getenv("KUBE_CLIENT_CERT_DATA") != "" && os.Getenv("KUBE_CLIENT_KEY_DATA") != ""
	hasStaticCfg := os.Getenv("KUBE_HOST") != "" && os.Getenv("KUBE_CLUSTER_CA_CERT_DATA") != "" &&
		(hasUserCredentials || hasClientCert || os.Getenv("KUBE_TOKEN") != "")
	if !hasFileCfg && !hasStaticCfg && !hasUserCredentials {
		t.Fatalf("File config (KUBE_CTX_AUTH_INFO and KUBE_CTX_CLUSTER) or static configuration"+
			"(%s) or (%s) must be set for acceptance tests",
			strings.Join([]string{"KUBE_HOST", "KUBE_USER", "KUBE_PASSWORD", "KUBE_CLUSTER_CA_CERT_DATA"}, ", "),
			strings.Join([]string{"KUBE_HOST", "KUBE_CLIENT_CERT_DATA", "KUBE_CLIENT_KEY_DATA", "KUBE_CLUSTER_CA_CERT_DATA"}, ", "),
		)
	}
	if diags := kubernetes.Provider().Configure(context.TODO(), sdkv2.NewResourceConfigRaw(nil)); diags.HasError() {
		t.Fatal(diags[0].Summary)
	}
}
