// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

// Package kubetest holds the acceptance test helpers shared by the Framework
// resource packages, the counterpart of the SDKv2 package's provider_test.go.
// Import it only from *_test packages; it depends on the mux, which imports the
// resource packages.
package kubetest

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"sync"
	"testing"

	gversion "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	sdkterraform "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
)

const (
	BusyboxImage = "busybox:1.36"
	AgnhostImage = "registry.k8s.io/e2e-test-images/agnhost:2.43"

	// IgnoreAnnotationsConfig is a provider block ignoring the annotation GKE
	// adds to Services.
	IgnoreAnnotationsConfig = `provider "kubernetes" {
  ignore_annotations = [
    "cloud\\.google\\.com\\/neg",
  ]
}
`
)

// ProviderFactories serves the muxed SDKv2 and Framework provider.
var ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": func() (tfprotov6.ProviderServer, error) {
		return mux.MuxServerWithProvider(context.Background(), "test", kubernetes.Provider())
	},
}

// ReleasedProvider runs a test step against a published provider version.
func ReleasedProvider(version string) map[string]resource.ExternalProvider {
	return map[string]resource.ExternalProvider{
		"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= " + version},
	}
}

// PreCheck requires an explicit cluster configuration, so tests never run
// against a default kubeconfig by accident.
func PreCheck(t *testing.T) {
	t.Helper()
	hasFileCfg := (os.Getenv("KUBE_CTX_AUTH_INFO") != "" && os.Getenv("KUBE_CTX_CLUSTER") != "") ||
		os.Getenv("KUBE_CTX") != "" || os.Getenv("KUBE_CONFIG_PATH") != "" || os.Getenv("KUBE_CONFIG_PATHS") != ""
	hasUserCredentials := os.Getenv("KUBE_USER") != "" && os.Getenv("KUBE_PASSWORD") != ""
	hasClientCert := os.Getenv("KUBE_CLIENT_CERT_DATA") != "" && os.Getenv("KUBE_CLIENT_KEY_DATA") != ""
	hasStaticCfg := os.Getenv("KUBE_HOST") != "" && os.Getenv("KUBE_CLUSTER_CA_CERT_DATA") != "" &&
		(hasUserCredentials || hasClientCert || os.Getenv("KUBE_TOKEN") != "")
	if !hasFileCfg && !hasStaticCfg && !hasUserCredentials {
		t.Fatal("File config (KUBE_CTX_AUTH_INFO and KUBE_CTX_CLUSTER, KUBE_CTX, KUBE_CONFIG_PATH or KUBE_CONFIG_PATHS) " +
			"or static configuration (KUBE_HOST, KUBE_CLUSTER_CA_CERT_DATA and credentials) must be set for acceptance tests")
	}
	if _, err := Clientset(); err != nil {
		t.Fatal(err)
	}
}

var (
	clientOnce sync.Once
	client     *k8sclient.Clientset
	clientErr  error
)

// Clientset returns a client for the cluster the provider is configured for.
func Clientset() (*k8sclient.Clientset, error) {
	clientOnce.Do(func() {
		p := kubernetes.Provider()
		if diags := p.Configure(context.Background(), sdkterraform.NewResourceConfigRaw(nil)); diags.HasError() {
			clientErr = fmt.Errorf("configuring the provider: %s: %s", diags[0].Summary, diags[0].Detail)
			return
		}
		clients, ok := p.Meta().(kubernetes.KubeClientsets)
		if !ok {
			clientErr = fmt.Errorf("unexpected provider metadata %T", p.Meta())
			return
		}
		client, clientErr = clients.MainClientset()
	})
	return client, clientErr
}

// ClusterVersion returns the version of the Kubernetes API server.
func ClusterVersion(t *testing.T) *gversion.Version {
	t.Helper()
	conn, err := Clientset()
	if err != nil {
		t.Fatal(err)
	}
	server, err := conn.ServerVersion()
	if err != nil {
		t.Fatal(err)
	}
	version, err := gversion.NewVersion(server.String())
	if err != nil {
		t.Fatal(err)
	}
	return version
}

func SkipIfClusterVersionLessThan(t *testing.T, minimum string) {
	t.Helper()
	if ClusterVersion(t).LessThan(gversion.Must(gversion.NewVersion(minimum))) {
		t.Skipf("This test does not run on cluster versions below %s", minimum)
	}
}

func SkipIfNotRunningInKind(t *testing.T) {
	t.Helper()
	if !isKind(t) {
		t.Skip("The Kubernetes endpoint must come from Kind for this test to run - skipping")
	}
}

func SkipIfRunningInKind(t *testing.T) {
	t.Helper()
	if isKind(t) {
		t.Skip("This test can't run in Kind - skipping")
	}
}

func SkipIfNotRunningInMinikube(t *testing.T) {
	t.Helper()
	if firstNode(t).Labels["kubernetes.io/hostname"] != "minikube" {
		t.Skip("The Kubernetes endpoint must come from Minikube for this test to run - skipping")
	}
}

func SkipIfRunningInMinikube(t *testing.T) {
	t.Helper()
	if firstNode(t).Labels["kubernetes.io/hostname"] == "minikube" {
		t.Skip("This test requires multiple Kubernetes nodes - skipping")
	}
}

func SkipIfRunningInAks(t *testing.T) {
	t.Helper()
	if _, ok := firstNode(t).Labels["kubernetes.azure.com/cluster"]; ok {
		t.Skip("This test cannot be run in AKS cluster")
	}
}

func SkipIfNotRunningInGke(t *testing.T) {
	t.Helper()
	if _, ok := firstNode(t).Labels["cloud.google.com/gke-nodepool"]; !ok {
		t.Skip("The Kubernetes endpoint must come from GKE for this test to run - skipping")
	}
	for _, name := range []string{"GOOGLE_PROJECT", "GOOGLE_REGION", "GOOGLE_ZONE"} {
		if os.Getenv(name) == "" {
			t.Skipf("%s must be set for GoogleCloud tests", name)
		}
	}
}

func SkipIfRunningInEks(t *testing.T) {
	t.Helper()
	if isEks(t) {
		t.Skip("This test cannot be run in EKS cluster")
	}
}

func SkipIfNotRunningInEks(t *testing.T) {
	t.Helper()
	if !isEks(t) {
		t.Skip("The Kubernetes endpoint must come from EKS for this test to run - skipping")
	}
	if os.Getenv("AWS_DEFAULT_REGION") == "" || os.Getenv("AWS_ACCESS_KEY_ID") == "" || os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		t.Fatal("AWS_DEFAULT_REGION, AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY must be set for AWS tests")
	}
}

func isKind(t *testing.T) bool {
	t.Helper()
	providerID, err := url.Parse(firstNode(t).Spec.ProviderID)
	if err != nil {
		t.Fatal(err)
	}
	return providerID.Scheme == "kind"
}

func isEks(t *testing.T) bool {
	t.Helper()
	conn, err := Clientset()
	if err != nil {
		t.Fatal(err)
	}
	_, err = conn.CoreV1().ConfigMaps("kube-system").Get(context.Background(), "aws-auth", metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return false
	}
	if err != nil {
		t.Fatal(err)
	}
	return true
}

func firstNode(t *testing.T) corev1.Node {
	t.Helper()
	conn, err := Clientset()
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := conn.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes.Items) == 0 {
		t.Fatal("No nodes found in the cluster")
	}
	return nodes.Items[0]
}
