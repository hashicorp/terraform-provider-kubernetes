// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"testing"

	gversion "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	sdkv2 "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	frameworkprovider "github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider"
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

func TestPodV1MuxRegistration(t *testing.T) {
	ctx := context.Background()
	legacy := kubernetes.Provider()
	if _, ok := legacy.ResourcesMap["kubernetes_pod_v1"]; ok {
		t.Fatal("managed kubernetes_pod_v1 must not be registered on SDKv2")
	}
	if legacy.ResourcesMap["kubernetes_pod"] == nil ||
		legacy.DataSourcesMap["kubernetes_pod"] == nil ||
		legacy.DataSourcesMap["kubernetes_pod_v1"] == nil {
		t.Fatal("deprecated Pod resource and both Pod data sources must remain registered")
	}
	server, err := mux.MuxServerWithProvider(ctx, "test", legacy)
	if err != nil {
		t.Fatal(err)
	}
	response, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range response.Diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("%s: %s", diagnostic.Summary, diagnostic.Detail)
		}
	}
	native := response.ResourceSchemas["kubernetes_pod_v1"]
	alias := response.ResourceSchemas["kubernetes_pod"]
	if native == nil || alias == nil {
		t.Fatal("the full mux must expose both resource types")
		return
	}
	if native.Version != 1 {
		t.Fatalf("schema version = %d, want 1", native.Version)
	}
	if !native.ValueType().Equal(alias.ValueType()) {
		t.Fatalf("native Pod persisted type differs from SDKv2 alias:\nnative: %s\nalias: %s", native.ValueType(), alias.ValueType())
	}
	framework := providerserver.NewProtocol6(frameworkprovider.New("test", nil))()
	frameworkSchema, err := framework.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil || frameworkSchema.ResourceSchemas["kubernetes_pod_v1"] == nil {
		t.Fatalf("native Pod not served by Framework: %v", err)
	}
}

func podSpecProtocolBlock(t *testing.T, block *tfprotov6.SchemaBlock, name string) *tfprotov6.SchemaBlock {
	t.Helper()
	for _, nested := range block.BlockTypes {
		if nested.TypeName == name {
			return nested.Block
		}
	}
	t.Fatalf("protocol schema has no %s block", name)
	return nil
}

func podSpecProtocolAttribute(t *testing.T, attributes []*tfprotov6.SchemaAttribute, name string) *tfprotov6.SchemaAttribute {
	t.Helper()
	for _, attribute := range attributes {
		if attribute.Name == name {
			return attribute
		}
	}
	t.Fatalf("protocol schema has no %s attribute", name)
	return nil
}

func TestPodV1SpecNativeGetSchemaReferences(t *testing.T) {
	ctx := context.Background()
	fullMux, err := mux.MuxServer(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	for name, server := range map[string]tfprotov6.ProviderServer{
		"framework": providerserver.NewProtocol6(frameworkprovider.New("test", nil))(),
		"full mux":  fullMux,
	} {
		t.Run(name, func(t *testing.T) {
			response, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
			if err != nil {
				t.Fatal(err)
			}
			for _, diagnostic := range response.Diagnostics {
				if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
					t.Fatalf("%s: %s", diagnostic.Summary, diagnostic.Detail)
				}
			}
			pod := response.ResourceSchemas["kubernetes_pod_v1"]
			if pod == nil || pod.Version != 1 {
				t.Fatal("expected native Pod schema version 1")
			}
			spec := podSpecProtocolBlock(t, pod.Block, "spec")
			for _, reference := range []struct{ name, child string }{
				{"readiness_gate", "condition_type"},
				{"image_pull_secrets", "name"},
			} {
				attribute := podSpecProtocolAttribute(t, spec.Attributes, reference.name)
				want := tftypes.List{ElementType: tftypes.Object{
					AttributeTypes: map[string]tftypes.Type{reference.child: tftypes.String},
				}}
				if attribute.NestedType != nil || attribute.Type == nil || !attribute.Type.Equal(want) ||
					!attribute.Optional || !attribute.Computed || attribute.Required {
					t.Fatalf("%s must be a plain optional-computed list(object), got %#v", reference.name, attribute)
				}
			}
			for _, container := range []string{"container", "init_container"} {
				block := podSpecProtocolBlock(t, spec, container)
				resources := podSpecProtocolAttribute(t, block.Attributes, "resources")
				if resources.NestedType == nil || resources.Type != nil ||
					resources.NestedType.Nesting != tfprotov6.SchemaObjectNestingModeList ||
					!resources.Optional || !resources.Computed {
					t.Fatalf("%s.resources must expose a computed nested list, got %#v", container, resources)
				}
				for _, child := range []string{"limits", "requests"} {
					attribute := podSpecProtocolAttribute(t, resources.NestedType.Attributes, child)
					if !attribute.Optional || !attribute.Computed {
						t.Fatalf("%s.resources.%s must be optional-computed", container, child)
					}
				}
			}
		})
	}
}

func TestAccKubernetesDataSourcePodV1_basic(t *testing.T) {
	resourceName := "kubernetes_pod_v1.test"
	dataSourceName := "data.kubernetes_pod_v1.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImage
	oneOrMore := regexp.MustCompile(`^[1-9][0-9]*$`)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPodV1PreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourcePodV1_basic(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.image", imageName),
				),
			},
			{
				Config: testAccKubernetesDataSourcePodV1_basic(name, imageName) +
					testAccKubernetesDataSourcePodV1_read(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.container.0.image", imageName),
					resource.TestMatchResourceAttr(dataSourceName, "spec.0.toleration.#", oneOrMore),
				),
			},
		},
	})
}

func TestAccKubernetesDataSourcePodV1_not_found(t *testing.T) {
	dataSourceName := "data.kubernetes_pod_v1.test"
	name := fmt.Sprintf("ceci-n.est-pas-une-pod-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPodV1PreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourcePodV1_nonexistent(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(dataSourceName, "spec.#", "0"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourcePodV1_basic(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    container {
      image = "%s"
      name  = "containername"
    }
  }
}
`, name, imageName)
}

func testAccKubernetesDataSourcePodV1_read() string {
	return `data "kubernetes_pod_v1" "test" {
  metadata {
    name = "${kubernetes_pod_v1.test.metadata.0.name}"
  }
}
`
}

func testAccKubernetesDataSourcePodV1_nonexistent(name string) string {
	return fmt.Sprintf(`data "kubernetes_pod_v1" "test" {
  metadata {
    name = "%s"
  }
}
`, name)
}
