// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1_test

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"

	gversion "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	sdkschema "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	sdkv2 "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
)

const (
	busyboxImage = "busybox:1.36"
	agnhostImage = "registry.k8s.io/e2e-test-images/agnhost:2.43"
)

var testAccProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": func() (tfprotov6.ProviderServer, error) {
		return mux.MuxServerWithProvider(context.Background(), "test", kubernetes.Provider())
	},
}

func TestWorkloadMuxSchemas(t *testing.T) {
	ctx := context.Background()
	legacy := kubernetes.Provider()
	legacySchemas, err := sdkschema.NewGRPCProviderServer(legacy).GetProviderSchema(ctx, &tfprotov5.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	server, err := testAccProviderFactories["kubernetes"]()
	if err != nil {
		t.Fatal(err)
	}
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range schemas.Diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("%s: %s", diagnostic.Summary, diagnostic.Detail)
		}
	}
	for target, alias := range map[string]string{
		"kubernetes_deployment_v1":   "kubernetes_deployment",
		"kubernetes_daemon_set_v1":   "kubernetes_daemonset",
		"kubernetes_stateful_set_v1": "kubernetes_stateful_set",
	} {
		t.Run(target, func(t *testing.T) {
			if _, registered := legacy.ResourcesMap[target]; registered {
				t.Fatal("versioned resource is still registered in SDKv2")
			}
			source := legacySchemas.ResourceSchemas[alias]
			if source == nil || schemas.ResourceSchemas[alias] == nil {
				t.Fatal("deprecated SDKv2 alias is missing")
				return
			}
			destination := schemas.ResourceSchemas[target]
			if destination == nil {
				t.Fatal("Framework resource is missing from the production mux")
				return
			}
			if destination.Version != source.Version {
				t.Errorf("schema version = %d, want %d", destination.Version, source.Version)
			}
			if !destination.ValueType().Equal(source.ValueType()) {
				t.Error("Framework persisted state type differs from the original SDKv2 resource")
			}
		})
	}
}

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
			strings.Join([]string{"KUBE_HOST", "KUBE_CLIENT_CERT_DATA", "KUBE_CLIENT_KEY_DATA", "KUBE_CLUSTER_CA_CERT_DATA"}, ", "))
	}
	if _, err := testAccWorkloadClient(); err != nil {
		t.Fatal(err)
	}
}

func testAccWorkloadClient() (*k8sclient.Clientset, error) {
	p := kubernetes.Provider()
	diagnostics := p.Configure(context.Background(), sdkv2.NewResourceConfigRaw(nil))
	if diagnostics.HasError() {
		return nil, fmt.Errorf("configuring acceptance test provider: %s", diagnostics[0].Summary)
	}
	clients, ok := p.Meta().(kubernetes.KubeClientsets)
	if !ok {
		return nil, fmt.Errorf("unexpected acceptance test provider metadata %T", p.Meta())
	}
	return clients.MainClientset()
}

func IdParts(id string) (string, string, error) {
	return kubernetes.IdParts(id)
}

func skipIfClusterVersionLessThan(t *testing.T, minimum string) {
	t.Helper()
	client, err := testAccWorkloadClient()
	if err != nil {
		t.Fatal(err)
	}
	server, err := client.ServerVersion()
	if err != nil {
		t.Fatal(err)
	}
	currentVersion, err := gversion.NewVersion(server.String())
	if err != nil {
		t.Fatal(err)
	}
	minimumVersion, err := gversion.NewVersion(minimum)
	if err != nil {
		t.Fatal(err)
	}
	if currentVersion.LessThan(minimumVersion) {
		t.Skipf("This test does not run on cluster versions below %v", minimum)
	}
}

func skipIfUnsupportedSecurityContextRunAsGroup(t *testing.T) {
	t.Helper()
	skipIfClusterVersionLessThan(t, "1.14.0")
}

func skipIfNotRunningInKind(t *testing.T) {
	t.Helper()
	client, err := testAccWorkloadClient()
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := client.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes.Items) == 0 {
		t.Fatal("No nodes found in the cluster")
	}
	providerID, err := url.Parse(nodes.Items[0].Spec.ProviderID)
	if err != nil {
		t.Fatal(err)
	}
	if providerID.Scheme != "kind" {
		t.Skip("The Kubernetes endpoint must come from Kind for this test to run - skipping")
	}
}

func isRunningInEks() (bool, error) {
	client, err := testAccWorkloadClient()
	if err != nil {
		return false, err
	}
	_, err = client.CoreV1().ConfigMaps("kube-system").Get(context.Background(), "aws-auth", metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return false, nil
	}
	return err == nil, err
}

func skipIfRunningInEks(t *testing.T) {
	t.Helper()
	inEKS, err := isRunningInEks()
	if err != nil {
		t.Fatal(err)
	}
	if inEKS {
		t.Skip("This test cannot be run in EKS cluster")
	}
}

func skipIfNotRunningInEks(t *testing.T) {
	t.Helper()
	inEKS, err := isRunningInEks()
	if err != nil {
		t.Fatal(err)
	}
	if !inEKS {
		t.Skip("The Kubernetes endpoint must come from EKS for this test to run - skipping")
	}
	if os.Getenv("AWS_DEFAULT_REGION") == "" || os.Getenv("AWS_ACCESS_KEY_ID") == "" || os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		t.Fatal("AWS_DEFAULT_REGION, AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY must be set for AWS tests")
	}
}
