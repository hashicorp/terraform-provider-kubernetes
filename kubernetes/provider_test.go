package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	gversion "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/hashicorp/terraform-provider-google/google"
	"github.com/terraform-providers/terraform-provider-aws/aws"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm"
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"kubernetes": testAccProvider,
		"google":     google.Provider(),
		"aws":        aws.Provider(),
		"azurerm":    azurerm.Provider(),
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ terraform.ResourceProvider = Provider()
}

func TestProvider_configure(t *testing.T) {
	resetEnv := unsetEnv(t)
	defer resetEnv()

	os.Setenv("KUBECONFIG", "test-fixtures/kube-config.yaml")
	os.Setenv("KUBE_CTX", "gcp")

	rc := terraform.NewResourceConfigRaw(map[string]interface{}{})
	p := Provider()
	err := p.Configure(rc)
	if err != nil {
		t.Fatal(err)
	}
}

func unsetEnv(t *testing.T) func() {
	e := getEnv()

	if err := os.Unsetenv("KUBECONFIG"); err != nil {
		t.Fatalf("Error unsetting env var KUBECONFIG: %s", err)
	}
	if err := os.Unsetenv("KUBE_CONFIG"); err != nil {
		t.Fatalf("Error unsetting env var KUBE_CONFIG: %s", err)
	}
	if err := os.Unsetenv("KUBE_CTX"); err != nil {
		t.Fatalf("Error unsetting env var KUBE_CTX: %s", err)
	}
	if err := os.Unsetenv("KUBE_CTX_AUTH_INFO"); err != nil {
		t.Fatalf("Error unsetting env var KUBE_CTX_AUTH_INFO: %s", err)
	}
	if err := os.Unsetenv("KUBE_CTX_CLUSTER"); err != nil {
		t.Fatalf("Error unsetting env var KUBE_CTX_CLUSTER: %s", err)
	}
	if err := os.Unsetenv("KUBE_HOST"); err != nil {
		t.Fatalf("Error unsetting env var KUBE_HOST: %s", err)
	}
	if err := os.Unsetenv("KUBE_USER"); err != nil {
		t.Fatalf("Error unsetting env var KUBE_USER: %s", err)
	}
	if err := os.Unsetenv("KUBE_PASSWORD"); err != nil {
		t.Fatalf("Error unsetting env var KUBE_PASSWORD: %s", err)
	}
	if err := os.Unsetenv("KUBE_CLIENT_CERT_DATA"); err != nil {
		t.Fatalf("Error unsetting env var KUBE_CLIENT_CERT_DATA: %s", err)
	}
	if err := os.Unsetenv("KUBE_CLIENT_KEY_DATA"); err != nil {
		t.Fatalf("Error unsetting env var KUBE_CLIENT_KEY_DATA: %s", err)
	}
	if err := os.Unsetenv("KUBE_CLUSTER_CA_CERT_DATA"); err != nil {
		t.Fatalf("Error unsetting env var KUBE_CLUSTER_CA_CERT_DATA: %s", err)
	}
	if err := os.Unsetenv("KUBE_INSECURE"); err != nil {
		t.Fatalf("Error unsetting env var KUBE_INSECURE: %s", err)
	}
	if err := os.Unsetenv("KUBE_LOAD_CONFIG_FILE"); err != nil {
		t.Fatalf("Error unsetting env var KUBE_LOAD_CONFIG_FILE: %s", err)
	}
	if err := os.Unsetenv("KUBE_TOKEN"); err != nil {
		t.Fatalf("Error unsetting env var KUBE_TOKEN: %s", err)
	}

	return func() {
		if err := os.Setenv("KUBE_CONFIG", e.Config); err != nil {
			t.Fatalf("Error resetting env var KUBE_CONFIG: %s", err)
		}
		if err := os.Setenv("KUBECONFIG", e.Config); err != nil {
			t.Fatalf("Error resetting env var KUBECONFIG: %s", err)
		}
		if err := os.Setenv("KUBE_CTX", e.Ctx); err != nil {
			t.Fatalf("Error resetting env var KUBE_CTX: %s", err)
		}
		if err := os.Setenv("KUBE_CTX_AUTH_INFO", e.CtxAuthInfo); err != nil {
			t.Fatalf("Error resetting env var KUBE_CTX_AUTH_INFO: %s", err)
		}
		if err := os.Setenv("KUBE_CTX_CLUSTER", e.CtxCluster); err != nil {
			t.Fatalf("Error resetting env var KUBE_CTX_CLUSTER: %s", err)
		}
		if err := os.Setenv("KUBE_HOST", e.Host); err != nil {
			t.Fatalf("Error resetting env var KUBE_HOST: %s", err)
		}
		if err := os.Setenv("KUBE_USER", e.User); err != nil {
			t.Fatalf("Error resetting env var KUBE_USER: %s", err)
		}
		if err := os.Setenv("KUBE_PASSWORD", e.Password); err != nil {
			t.Fatalf("Error resetting env var KUBE_PASSWORD: %s", err)
		}
		if err := os.Setenv("KUBE_CLIENT_CERT_DATA", e.ClientCertData); err != nil {
			t.Fatalf("Error resetting env var KUBE_CLIENT_CERT_DATA: %s", err)
		}
		if err := os.Setenv("KUBE_CLIENT_KEY_DATA", e.ClientKeyData); err != nil {
			t.Fatalf("Error resetting env var KUBE_CLIENT_KEY_DATA: %s", err)
		}
		if err := os.Setenv("KUBE_CLUSTER_CA_CERT_DATA", e.ClusterCACertData); err != nil {
			t.Fatalf("Error resetting env var KUBE_CLUSTER_CA_CERT_DATA: %s", err)
		}
		if err := os.Setenv("KUBE_INSECURE", e.Insecure); err != nil {
			t.Fatalf("Error resetting env var KUBE_INSECURE: %s", err)
		}
		if err := os.Setenv("KUBE_LOAD_CONFIG_FILE", e.LoadConfigFile); err != nil {
			t.Fatalf("Error resetting env var KUBE_LOAD_CONFIG_FILE: %s", err)
		}
		if err := os.Setenv("KUBE_TOKEN", e.Token); err != nil {
			t.Fatalf("Error resetting env var KUBE_TOKEN: %s", err)
		}
	}
}

func getEnv() *currentEnv {
	e := &currentEnv{
		Ctx:               os.Getenv("KUBE_CTX"),
		CtxAuthInfo:       os.Getenv("KUBE_CTX_AUTH_INFO"),
		CtxCluster:        os.Getenv("KUBE_CTX_CLUSTER"),
		Host:              os.Getenv("KUBE_HOST"),
		User:              os.Getenv("KUBE_USER"),
		Password:          os.Getenv("KUBE_PASSWORD"),
		ClientCertData:    os.Getenv("KUBE_CLIENT_CERT_DATA"),
		ClientKeyData:     os.Getenv("KUBE_CLIENT_KEY_DATA"),
		ClusterCACertData: os.Getenv("KUBE_CLUSTER_CA_CERT_DATA"),
		Insecure:          os.Getenv("KUBE_INSECURE"),
		LoadConfigFile:    os.Getenv("KUBE_LOAD_CONFIG_FILE"),
		Token:             os.Getenv("KUBE_TOKEN"),
	}
	if cfg := os.Getenv("KUBE_CONFIG"); cfg != "" {
		e.Config = cfg
	}
	if cfg := os.Getenv("KUBECONFIG"); cfg != "" {
		e.Config = cfg
	}
	return e
}

func testAccPreCheck(t *testing.T) {
	hasFileCfg := (os.Getenv("KUBE_CTX_AUTH_INFO") != "" && os.Getenv("KUBE_CTX_CLUSTER") != "") ||
		os.Getenv("KUBE_CTX") != "" ||
		os.Getenv("KUBECONFIG") != "" ||
		os.Getenv("KUBE_CONFIG") != ""
	hasUserCredentials := os.Getenv("KUBE_USER") != "" && os.Getenv("KUBE_PASSWORD") != ""
	hasClientCert := os.Getenv("KUBE_CLIENT_CERT_DATA") != "" && os.Getenv("KUBE_CLIENT_KEY_DATA") != ""
	hasStaticCfg := (os.Getenv("KUBE_HOST") != "" &&
		os.Getenv("KUBE_CLUSTER_CA_CERT_DATA") != "") &&
		(hasUserCredentials || hasClientCert || os.Getenv("KUBE_TOKEN") != "")

	if !hasFileCfg && !hasStaticCfg {
		t.Fatalf("File config (KUBE_CTX_AUTH_INFO and KUBE_CTX_CLUSTER) or static configuration"+
			" (%s) must be set for acceptance tests",
			strings.Join([]string{
				"KUBE_HOST",
				"KUBE_USER",
				"KUBE_PASSWORD",
				"KUBE_CLIENT_CERT_DATA",
				"KUBE_CLIENT_KEY_DATA",
				"KUBE_CLUSTER_CA_CERT_DATA",
			}, ", "))
	}

	err := testAccProvider.Configure(terraform.NewResourceConfigRaw(nil))
	if err != nil {
		t.Fatal(err)
	}
}

func getClusterVersion() (*gversion.Version, error) {
	meta := testAccProvider.Meta()

	if meta == nil {
		return nil, fmt.Errorf("Provider not initialized, unable to check cluster version")
	}

	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return nil, err
	}
	serverVersion, err := conn.ServerVersion()

	if err != nil {
		return nil, err
	}

	return gversion.NewVersion(serverVersion.String())
}

func skipIfClusterVersionLessThan(t *testing.T, vs string) {
	if clusterVersionLessThan(vs) {
		t.Skip(fmt.Sprintf("This test will only run on cluster versions %v and above", vs))
	}
}

func skipIfNoLoadBalancersAvailable(t *testing.T) {
	isInGke, err := isRunningInGke()
	if err != nil {
		t.Fatal(err)
	}
	isInEks, err := isRunningInEks()
	if err != nil {
		t.Fatal(err)
	}
	if !isInGke && !isInEks {
		t.Skip("The Kubernetes endpoint must come from an environment which supports " +
			"load balancer provisioning for this test to run - skipping")
	}
}

func skipIfNotRunningInGke(t *testing.T) {
	isInGke, err := isRunningInGke()
	if err != nil {
		t.Fatal(err)
	}
	if !isInGke {
		t.Skip("The Kubernetes endpoint must come from GKE for this test to run - skipping")
	}
	if os.Getenv("GOOGLE_PROJECT") == "" || os.Getenv("GOOGLE_REGION") == "" || os.Getenv("GOOGLE_ZONE") == "" {
		t.Fatal("GOOGLE_PROJECT, GOOGLE_REGION, and GOOGLE_ZONE must be set for GoogleCloud tests")
	}
}

func skipIfNotRunningInAks(t *testing.T) {
	isInAks, err := isRunningInAks()
	if err != nil {
		t.Fatal(err)
	}
	if !isInAks {
		t.Skip("The Kubernetes endpoint must come from AKS for this test to run - skipping")
	}
	location := os.Getenv("TF_VAR_location")
	subscription := os.Getenv("ARM_SUBSCRIPTION_ID")
	if location == "" || subscription == "" {
		t.Fatal("TF_VAR_location and ARM_SUBSCRIPTION_ID must be set for Azure tests")
	}
}

func skipIfNotRunningInEks(t *testing.T) {
	isInEks, err := isRunningInEks()
	if err != nil {
		t.Fatal(err)
	}
	if !isInEks {
		t.Skip("The Kubernetes endpoint must come from EKS for this test to run - skipping")
	}
	if os.Getenv("AWS_DEFAULT_REGION") == "" || os.Getenv("AWS_ZONE") == "" || os.Getenv("AWS_ACCESS_KEY_ID") == "" || os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		t.Fatal("AWS_DEFAULT_REGION, AWS_ZONE, AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY must be set for AWS tests")
	}
}

func skipIfNotRunningInMinikube(t *testing.T) {
	isInMinikube, err := isRunningInMinikube()
	if err != nil {
		t.Fatal(err)
	}
	if !isInMinikube {
		t.Skip("The Kubernetes endpoint must come from Minikube for this test to run - skipping")
	}
}

func skipIfUnsupportedSecurityContextRunAsGroup(t *testing.T) {
	skipIfClusterVersionLessThan(t, "1.14.0")
}

func isRunningInMinikube() (bool, error) {
	node, err := getFirstNode()
	if err != nil {
		return false, err
	}

	labels := node.GetLabels()
	if v, ok := labels["kubernetes.io/hostname"]; ok && v == "minikube" {
		return true, nil
	}
	return false, nil
}

func isRunningInGke() (bool, error) {
	node, err := getFirstNode()
	if err != nil {
		return false, err
	}

	labels := node.GetLabels()
	if _, ok := labels["cloud.google.com/gke-nodepool"]; ok {
		return true, nil
	}
	return false, nil
}

func isRunningInEks() (bool, error) {
	// EKS nodes don't have any unique labels, so check for the AWS
	// specific config map created by our test-infra.
	meta := testAccProvider.Meta()
	if meta == nil {
		return false, errors.New("Provider not initialized, unable to fetch provider metadata")
	}
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}
	ctx := context.TODO()
	_, err = conn.CoreV1().ConfigMaps("kube-system").Get(ctx, "aws-auth", metav1.GetOptions{})
	if err != nil {
		return false, nil
	}
	return true, nil
}

func isRunningInAks() (bool, error) {
	node, err := getFirstNode()
	if err != nil {
		return false, err
	}

	labels := node.GetLabels()
	if _, ok := labels["kubernetes.azure.com/cluster"]; ok {
		return true, nil
	}
	return false, nil
}

func getFirstNode() (api.Node, error) {
	meta := testAccProvider.Meta()
	if meta == nil {
		return api.Node{}, errors.New("Provider not initialized, unable to get cluster node")
	}
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return api.Node{}, err
	}
	ctx := context.TODO()

	resp, err := conn.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return api.Node{}, err
	}

	if len(resp.Items) < 1 {
		return api.Node{}, errors.New("Expected at least 1 node, none found")
	}

	return resp.Items[0], nil
}

func clusterVersionLessThan(vs string) bool {
	cv, err := getClusterVersion()

	if err != nil {
		return false
	}

	v, err := gversion.NewVersion(vs)

	if err != nil {
		return false
	}

	return cv.LessThan(v)
}

type currentEnv struct {
	Config            string
	Ctx               string
	CtxAuthInfo       string
	CtxCluster        string
	Host              string
	User              string
	Password          string
	ClientCertData    string
	ClientKeyData     string
	ClusterCACertData string
	Insecure          string
	LoadConfigFile    string
	Token             string
}
