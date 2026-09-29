// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	corev1api "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

// createTestNamespace creates a namespace directly via the Kubernetes API,
// bypassing Terraform. Used for out-of-band setup in disappearance tests.
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

// deleteTestNamespace deletes a namespace and waits until the Kubernetes API
// returns a 404 for it. A plain Delete() call returns as soon as the API
// accepts the request; the namespace stays in Terminating phase (and still
// returns a 200 with spec.finalizers intact) until all finalizers are cleared.
// Without waiting, step 2 of the disappearance test would read a Terminating
// namespace and observe spec.# = 1 instead of the expected 0.
func deleteTestNamespace(t *testing.T, name string) {
	t.Helper()
	conn, err := sdkv2providerMeta()().(kubernetes.KubeClientsets).MainClientset()
	if err != nil {
		t.Fatalf("deleteTestNamespace %q: client: %v", name, err)
	}
	if err := conn.CoreV1().Namespaces().Delete(context.Background(), name, metav1.DeleteOptions{}); err != nil {
		t.Fatalf("deleteTestNamespace %q: delete: %v", name, err)
	}
	// Poll until the namespace is fully gone (404) before returning.
	for i := 0; i < 60; i++ {
		_, err := conn.CoreV1().Namespaces().Get(context.Background(), name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("deleteTestNamespace %q: namespace still present after 30s", name)
}

// TestAccKubernetesDataSourceNamespaceV1_MigrateFromSDKv2 is the migration
// test described in the Hashicorp migration guide for data sources.
//
// Step 1 — runs against the last published SDKv2 version of the provider
// (3.2.1) via ExternalProviders. The terraform_data resource captures all
// data source attribute values into state, acting as an anchor for comparison.
//
// Step 2 — runs the identical config against the local framework implementation
// via ProtoV6ProviderFactories. ExpectEmptyPlan() asserts that no planned
// changes are produced, confirming the framework returns byte-for-byte
// identical attribute values and existing state remains fully compatible.
func TestAccKubernetesDataSourceNamespaceV1_MigrateFromSDKv2(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				// Step 1: establish state with the last published SDKv2 provider.
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						VersionConstraint: "3.2.1",
						Source:            "hashicorp/kubernetes",
					},
				},
				Config: testNamespaceConfig(),
			},
			{
				// Step 2: switch to the local framework implementation and
				// verify no planned changes are produced.
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testNamespaceConfig(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// TestAccKubernetesDataSourceNamespaceV1_explicit_metadata_inputs covers
// Fix 1: annotations and labels must be accepted in the data source config
// (they are Optional+Computed, not read-only). In the original PR they were
// declared Computed-only, causing a "Cannot set value for this attribute as
// the provider has marked it as read-only" error for any existing config
// that set them — matching the SDKv2 schema where both fields were Optional.
func TestAccKubernetesDataSourceNamespaceV1_explicit_metadata_inputs(t *testing.T) {
	dataSourceName := "data.kubernetes_namespace_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Explicitly setting annotations and labels must not be rejected.
				// The API response will overwrite them in state on every read —
				// that is the correct data-source behaviour and matches SDKv2.
				Config: `
data "kubernetes_namespace_v1" "test" {
  metadata {
    name        = "kube-system"
    annotations = {}
    labels      = {}
  }
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "id", "kube-system"),
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", "kube-system"),
					// API-populated fields are present despite the empty-map config.
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.resource_version"),
				),
			},
		},
	})
}

// TestAccKubernetesDataSourceNamespaceV1_deferred_read covers Fix 2: when
// metadata.name is unknown at plan time (it comes from another resource's
// output), Terraform defers the data source read to apply time. With the
// original ListNestedBlock, spec was planned as a known empty list; Read
// then populated it with finalizers, causing Terraform Core to raise
// "Provider produced inconsistent final plan / new element 0 has appeared".
// With ListNestedAttribute (Computed:true), spec is planned as unknown and
// resolves cleanly at apply time. The second step confirms convergence.
func TestAccKubernetesDataSourceNamespaceV1_deferred_read(t *testing.T) {
	dataSourceName := "data.kubernetes_namespace_v1.test"

	// Config with an unknown name input — forces a deferred data source read.
	deferredConfig := `
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

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: deferredConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "id", "kube-system"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.finalizers.0", "kubernetes"),
				),
			},
			{
				// Convergence check: a second apply must produce an empty plan.
				Config: deferredConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// TestAccKubernetesDataSourceNamespaceV1_disappearance covers Fix 3: when a
// namespace that was previously in state is deleted externally, the next read
// must be silent (no error), id must equal the requested name (not null), and
// spec must be empty. In the original PR the 404 path returned without writing
// state, leaving id as null — breaking any downstream reference to .id.
func TestAccKubernetesDataSourceNamespaceV1_disappearance(t *testing.T) {
	name := fmt.Sprintf("tf-acc-ns-disappear-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	dataSourceName := "data.kubernetes_namespace_v1.test"

	cfg := fmt.Sprintf(`
data "kubernetes_namespace_v1" "test" {
  metadata {
    name = "%s"
  }
}`, name)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Step 1: create the namespace out-of-band, then read it via the
				// data source to establish state.
				PreConfig: func() { createTestNamespace(t, name) },
				Config:    cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "id", name),
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(dataSourceName, "spec.#", "1"),
				),
			},
			{
				// Step 2: delete the namespace out-of-band, then re-apply the same
				// config. Read encounters a 404 and must NOT return an error; id must
				// still equal name (not null) and spec must be empty.
				PreConfig: func() { deleteTestNamespace(t, name) },
				Config:    cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "id", name),
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(dataSourceName, "spec.#", "0"),
				),
			},
		},
	})
}
