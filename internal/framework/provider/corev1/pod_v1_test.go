// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"

	gversion "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	sdkv2 "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	corev1api "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
)

const (
	busyboxImage = "busybox:1.36"
	agnhostImage = "registry.k8s.io/e2e-test-images/agnhost:2.43"
)

type KubeClientsets = kubernetes.KubeClientsets

var (
	testAccProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"kubernetes": func() (tfprotov6.ProviderServer, error) {
			provider := kubernetes.Provider()
			return mux.MuxServerWithProvider(context.Background(), "test", provider)
		},
	}
)

func testAccPodV1PreCheck(t *testing.T) {
	t.Helper()
	testAccPreCheck(t)
	if _, err := testAccPodV1MainClientset(); err != nil {
		t.Fatalf("failed to configure pod v1 test client: %v", err)
	}
}

func IdParts(id string) (string, string, error) {
	return kubernetes.IdParts(id)
}

func setClusterVersionVar(t *testing.T, varName string) {
	t.Helper()
	cv, err := getClusterVersion()
	if err != nil {
		t.Skipf("Could not get cluster version")
	}
	os.Setenv(varName, fmt.Sprintf("v%s", cv.Core().Original()))
}

func skipIfClusterVersionLessThan(t *testing.T, vs string) {
	t.Helper()
	if clusterVersionLessThan(vs) {
		t.Skipf("This test does not run on cluster versions below %v", vs)
	}
}

func skipIfUnsupportedSecurityContextRunAsGroup(t *testing.T) {
	t.Helper()
	skipIfClusterVersionLessThan(t, "1.14.0")
}

func skipIfRunningInAks(t *testing.T) {
	t.Helper()
	isInAks, err := isRunningInAks()
	if err != nil {
		t.Fatal(err)
	}
	if isInAks {
		t.Skip("This test cannot be run in AKS cluster")
	}
}

func skipIfNotRunningInKind(t *testing.T) {
	t.Helper()
	isInKind, err := isRunningInKind()
	if err != nil {
		t.Fatal(err)
	}
	if !isInKind {
		t.Skip("The Kubernetes endpoint must come from Kind for this test to run - skipping")
	}
}

func skipIfRunningInKind(t *testing.T) {
	t.Helper()
	isInKind, err := isRunningInKind()
	if err != nil {
		t.Fatal(err)
	}
	if isInKind {
		t.Skip("This test can't run in Kind - skipping")
	}
}

func skipIfRunningInMinikube(t *testing.T) {
	t.Helper()
	isInMinikube, err := isRunningInMinikube()
	if err != nil {
		t.Fatal(err)
	}
	if isInMinikube {
		t.Skip("This test requires multiple Kubernetes nodes - skipping")
	}
}

func skipIfNotRunningInGke(t *testing.T) {
	t.Helper()
	isInGke, err := isRunningInGke()
	if err != nil {
		t.Fatal(err)
	}
	if !isInGke {
		t.Skip("The Kubernetes endpoint must come from GKE for this test to run - skipping")
	}
	for _, ev := range []string{"GOOGLE_PROJECT", "GOOGLE_REGION", "GOOGLE_ZONE"} {
		if os.Getenv(ev) == "" {
			t.Skipf("%s must be set for GoogleCloud tests", ev)
		}
	}
}

func skipIfRunningInEks(t *testing.T) {
	t.Helper()
	isInEks, err := isRunningInEks()
	if err != nil {
		t.Fatal(err)
	}
	if isInEks {
		t.Skip("This test cannot be run in EKS cluster")
	}
}

func skipIfNotRunningInMinikube(t *testing.T) {
	t.Helper()
	isInMinikube, err := isRunningInMinikube()
	if err != nil {
		t.Fatal(err)
	}
	if !isInMinikube {
		t.Skip("The Kubernetes endpoint must come from Minikube for this test to run - skipping")
	}
}

func testAccKubernetesConfig_ignoreAnnotations() string {
	return `provider "kubernetes" {
  ignore_annotations = [
    "cloud\\.google\\.com\\/neg",
  ]
}
`
}

func testAccCheckKubernetesServiceAccountV1Exists(n string, obj *corev1api.ServiceAccount) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		conn, err := testAccPodV1MainClientset()
		if err != nil {
			return err
		}

		namespace, name, err := IdParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		out, err := conn.CoreV1().ServiceAccounts(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccCheckKubernetesPersistentVolumeClaimV1IsDestroyed(obj *corev1api.PersistentVolumeClaim) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		meta := obj.GetObjectMeta()
		conn, err := testAccPodV1MainClientset()
		if err != nil {
			return err
		}
		out, err := conn.CoreV1().PersistentVolumeClaims(meta.GetNamespace()).Get(context.Background(), meta.GetName(), metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			return err
		}

		return fmt.Errorf("expected no PVC but still found %q", out.GetObjectMeta().GetName())
	}
}

func getClusterVersion() (*gversion.Version, error) {
	conn, err := testAccPodV1MainClientset()
	if err != nil {
		return nil, err
	}
	serverVersion, err := conn.ServerVersion()
	if err != nil {
		return nil, err
	}

	return gversion.NewVersion(serverVersion.String())
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

func isRunningInKind() (bool, error) {
	node, err := getFirstNode()
	if err != nil {
		return false, err
	}
	u, err := url.Parse(node.Spec.ProviderID)
	if err != nil {
		return false, err
	}
	if u.Scheme == "kind" {
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

func isRunningInEks() (bool, error) {
	conn, err := testAccPodV1MainClientset()
	if err != nil {
		return false, err
	}
	_, err = conn.CoreV1().ConfigMaps("kube-system").Get(context.Background(), "aws-auth", metav1.GetOptions{})
	if err != nil {
		return false, nil
	}
	return true, nil
}

func getFirstNode() (corev1api.Node, error) {
	conn, err := testAccPodV1MainClientset()
	if err != nil {
		return corev1api.Node{}, err
	}

	resp, err := conn.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return corev1api.Node{}, err
	}
	if len(resp.Items) < 1 {
		return corev1api.Node{}, fmt.Errorf("expected at least 1 node, none found")
	}

	return resp.Items[0], nil
}

func testAccPodV1MainClientset() (*k8sclient.Clientset, error) {
	p := kubernetes.Provider()
	diags := p.Configure(context.Background(), sdkv2.NewResourceConfigRaw(nil))
	if diags.HasError() {
		return nil, fmt.Errorf("configuring SDKv2 provider: %s", diags[0].Summary)
	}

	meta := p.Meta()
	if meta == nil {
		return nil, fmt.Errorf("configuring SDKv2 provider: empty provider metadata")
	}

	clients, ok := meta.(KubeClientsets)
	if !ok {
		return nil, fmt.Errorf("unexpected provider metadata type %T", meta)
	}

	return clients.MainClientset()
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
