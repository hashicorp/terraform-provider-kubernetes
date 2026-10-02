// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1_test

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	sdkterraform "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

const (
	busyboxImage = "busybox:1.36"
	agnhostImage = "registry.k8s.io/e2e-test-images/agnhost:2.43"
)

var (
	testAccProvider                 = kubernetes.Provider()
	testAccConfigure                sync.Once
	testAccConfigureError           string
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"kubernetes": func() (tfprotov6.ProviderServer, error) {
			return mux.MuxServer(context.Background(), "test")
		},
	}
)

func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("KUBE_CONFIG_PATH") == "" && os.Getenv("KUBE_CONFIG_PATHS") == "" &&
		os.Getenv("KUBE_HOST") == "" && os.Getenv("KUBE_CTX") == "" {
		t.Fatal("Set an explicit Kubernetes test configuration before running batch acceptance tests.")
	}
	testAccConfigure.Do(func() {
		diagnostics := testAccProvider.Configure(context.Background(), sdkterraform.NewResourceConfigRaw(nil))
		for _, diagnostic := range diagnostics {
			if diagnostic.Summary != "" && diagnostics.HasError() {
				testAccConfigureError += diagnostic.Summary + ": " + diagnostic.Detail + "\n"
			}
		}
	})
	if testAccConfigureError != "" {
		t.Fatal(testAccConfigureError)
	}
}

func skipIfClusterVersionLessThan(t *testing.T, minimum string) {
	t.Helper()
	clients, ok := testAccProvider.Meta().(kubernetes.KubeClientsets)
	if !ok {
		t.Fatal("Kubernetes acceptance client is not configured")
	}
	client, err := clients.MainClientset()
	if err != nil {
		t.Fatal(err)
	}
	server, err := client.ServerVersion()
	if err != nil {
		t.Fatal(err)
	}
	actual, err := version.NewVersion(server.String())
	if err != nil {
		t.Fatal(err)
	}
	required, err := version.NewVersion(minimum)
	if err != nil {
		t.Fatal(err)
	}
	if actual.LessThan(required) {
		t.Skipf("Requires Kubernetes %s or later; found %s", minimum, actual)
	}
}
