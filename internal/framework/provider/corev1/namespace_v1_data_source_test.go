// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	corev1api "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

const namespaceDataSourceName = "data.kubernetes_namespace_v1.test"

// TestAccKubernetesDataSourceNamespaceV1_basic reads a live namespace.
func TestAccKubernetesDataSourceNamespaceV1_basic(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNamespaceDataSourceConfig("kube-system"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(namespaceDataSourceName, "id", "kube-system"),
					resource.TestCheckResourceAttr(namespaceDataSourceName, "metadata.0.name", "kube-system"),
					resource.TestCheckResourceAttrSet(namespaceDataSourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttrSet(namespaceDataSourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttr(namespaceDataSourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(namespaceDataSourceName, "spec.0.finalizers.#", "1"),
					resource.TestCheckResourceAttr(namespaceDataSourceName, "spec.0.finalizers.0", "kubernetes"),
				),
			},
		},
	})
}

// TestAccKubernetesDataSourceNamespaceV1_not_found verifies the 404 path: no error,
// id still set to the requested name, and unset metadata recorded as SDKv2 zero values.
func TestAccKubernetesDataSourceNamespaceV1_not_found(t *testing.T) {
	name := fmt.Sprintf("tf-acc-ns-absent-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNamespaceDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(namespaceDataSourceName, "id", name),
					resource.TestCheckResourceAttr(namespaceDataSourceName, "metadata.0.name", name),
				),
				// State checks, not TestCheckResourceAttr: that helper accepts an absent
				// key whenever a ".#" is expected to be "0", so it cannot tell a null
				// collection from an empty one — the distinction being asserted here.
				ConfigStateChecks: namespaceNotFoundStateChecks(),
			},
		},
	})
}

// TestAccKubernetesDataSourceNamespaceV1_no_metadata verifies that omitting the
// metadata block is caught at plan time by listvalidator.IsRequired().
func TestAccKubernetesDataSourceNamespaceV1_no_metadata(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccNamespaceDataSourceNoMetadataConfig(),
				ExpectError: regexp.MustCompile(`(?s)Invalid Block.*metadata must have a configuration value`),
			},
		},
	})
}

// TestAccKubernetesDataSourceNamespaceV1_no_name verifies that metadata.name is
// Required, so omitting it fails at plan time rather than during the read.
func TestAccKubernetesDataSourceNamespaceV1_no_name(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccNamespaceDataSourceNoNameConfig(),
				ExpectError: regexp.MustCompile(`(?s)(Missing required argument|Incorrect attribute value type|argument "name" is required|Missing Configuration for Required Attribute)`),
			},
		},
	})
}

// TestAccKubernetesDataSourceNamespaceV1_deferred_read covers an unknown
// metadata.name, which defers the read to apply time.
//
// spec must be a Computed ListNestedAttribute for this to work. As a ListNestedBlock it
// was planned as a known empty list, and Read then populating finalizers raised
// "Provider produced inconsistent final plan / new element 0 has appeared". Step 2
// confirms convergence.
func TestAccKubernetesDataSourceNamespaceV1_deferred_read(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNamespaceDataSourceDeferredReadConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(namespaceDataSourceName, "id", "kube-system"),
					resource.TestCheckResourceAttr(namespaceDataSourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(namespaceDataSourceName, "spec.0.finalizers.0", "kubernetes"),
				),
			},
			{
				Config: testAccNamespaceDataSourceDeferredReadConfig(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

// TestAccKubernetesDataSourceNamespaceV1_disappearance covers a namespace that is in
// state and then deleted externally. The next read must be silent and must still set
// id, which an earlier revision got wrong by returning without writing state at all.
func TestAccKubernetesDataSourceNamespaceV1_disappearance(t *testing.T) {
	name := fmt.Sprintf("tf-acc-ns-disappear-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() { createTestNamespace(t, name) },
				Config:    testAccNamespaceDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(namespaceDataSourceName, "id", name),
					resource.TestCheckResourceAttr(namespaceDataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(namespaceDataSourceName, "spec.#", "1"),
				),
			},
			{
				PreConfig: func() { deleteTestNamespace(t, name) },
				Config:    testAccNamespaceDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(namespaceDataSourceName, "id", name),
					resource.TestCheckResourceAttr(namespaceDataSourceName, "metadata.0.name", name),
				),
				ConfigStateChecks: namespaceNotFoundStateChecks(),
			},
		},
	})
}

// namespaceNotFoundStateChecks asserts the 404 state shape: SDKv2 zero values for
// unset metadata, and a null spec. Shared so the plain and migration tests cannot drift.
func namespaceNotFoundStateChecks() []statecheck.StateCheck {
	meta := func(field string) tfjsonpath.Path {
		return tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey(field)
	}
	return []statecheck.StateCheck{
		statecheck.ExpectKnownValue(namespaceDataSourceName, tfjsonpath.New("spec"), knownvalue.Null()),
		statecheck.ExpectKnownValue(namespaceDataSourceName, meta("uid"), knownvalue.StringExact("")),
		statecheck.ExpectKnownValue(namespaceDataSourceName, meta("resource_version"), knownvalue.StringExact("")),
		statecheck.ExpectKnownValue(namespaceDataSourceName, meta("generation"), knownvalue.Int64Exact(0)),
		statecheck.ExpectKnownValue(namespaceDataSourceName, meta("annotations"), knownvalue.MapSizeExact(0)),
		statecheck.ExpectKnownValue(namespaceDataSourceName, meta("labels"), knownvalue.MapSizeExact(0)),
	}
}

// createTestNamespace creates a namespace directly via the Kubernetes API, bypassing
// Terraform, for out-of-band setup.
func createTestNamespace(t *testing.T, name string) {
	t.Helper()
	conn, err := sdkv2providerMeta()().(kubernetes.KubeClientsets).MainClientset()
	if err != nil {
		t.Fatalf("createTestNamespace %q: client: %v", name, err)
	}
	ns := &corev1api.Namespace{}
	ns.SetName(name)
	if _, err := conn.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{}); err != nil {
		t.Fatalf("createTestNamespace %q: %v", name, err)
	}
}

// deleteTestNamespace deletes a namespace and waits for the API to return a 404.
// Delete() returns as soon as the request is accepted; the namespace stays in
// Terminating (still a 200, finalizers intact) until they are cleared, so without the
// wait a following read would see a live namespace rather than a missing one.
func deleteTestNamespace(t *testing.T, name string) {
	t.Helper()
	conn, err := sdkv2providerMeta()().(kubernetes.KubeClientsets).MainClientset()
	if err != nil {
		t.Fatalf("deleteTestNamespace %q: client: %v", name, err)
	}
	if err := conn.CoreV1().Namespaces().Delete(context.Background(), name, metav1.DeleteOptions{}); err != nil {
		t.Fatalf("deleteTestNamespace %q: delete: %v", name, err)
	}
	for i := 0; i < 60; i++ {
		_, err := conn.CoreV1().Namespaces().Get(context.Background(), name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("deleteTestNamespace %q: namespace still present after 30s", name)
}

// testAccNamespaceDataSourceConfig reads the named namespace. Migration tests wrap
// this with a terraform_data anchor; see testAccNamespaceDataSourceAnchoredConfig.
func testAccNamespaceDataSourceConfig(name string) string {
	return fmt.Sprintf(`
data "kubernetes_namespace_v1" "test" {
  metadata {
    name = %q
  }
}`, name)
}

func testAccNamespaceDataSourceNoMetadataConfig() string {
	return `
data "kubernetes_namespace_v1" "test" {}`
}

func testAccNamespaceDataSourceNoNameConfig() string {
	return `
data "kubernetes_namespace_v1" "test" {
  metadata {}
}`
}

// testAccNamespaceDataSourceDeferredReadConfig makes metadata.name unknown at plan
// time by sourcing it from another resource's output.
func testAccNamespaceDataSourceDeferredReadConfig() string {
	return `
resource "terraform_data" "name" {
  input = "kube-system"
}

data "kubernetes_namespace_v1" "test" {
  metadata {
    name = terraform_data.name.output
  }
}

resource "terraform_data" "consumer" {
  input = data.kubernetes_namespace_v1.test.spec
}`
}
