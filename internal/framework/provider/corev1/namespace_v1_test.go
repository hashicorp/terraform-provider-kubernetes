// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// testNamespaceConfig returns the shared HCL config used by both the
// acceptance tests and the migration test. Using kube-system keeps it
// self-contained — it is guaranteed to exist on every Kubernetes cluster.
func testNamespaceConfig() string {
	return `
data "kubernetes_namespace_v1" "test" {
  metadata {
    name = "kube-system"
  }
}

resource "terraform_data" "test" {
  input = {
    name             = data.kubernetes_namespace_v1.test.metadata[0].name
    uid              = data.kubernetes_namespace_v1.test.metadata[0].uid
    resource_version = data.kubernetes_namespace_v1.test.metadata[0].resource_version
    generation       = data.kubernetes_namespace_v1.test.metadata[0].generation
    labels           = data.kubernetes_namespace_v1.test.metadata[0].labels
    annotations      = data.kubernetes_namespace_v1.test.metadata[0].annotations
    spec             = data.kubernetes_namespace_v1.test.spec
  }
}
`
}

// TestAccKubernetesDataSourceNamespaceV1_basic verifies that the framework
// implementation reads all expected attributes from a live namespace.
func TestAccKubernetesDataSourceNamespaceV1_basic(t *testing.T) {
	dataSourceName := "data.kubernetes_namespace_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testNamespaceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", "kube-system"),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.finalizers.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.finalizers.0", "kubernetes"),
				),
			},
		},
	})
}

// TestAccKubernetesDataSourceNamespaceV1_not_found verifies the silent
// not-found behaviour is preserved in the framework implementation.
func TestAccKubernetesDataSourceNamespaceV1_not_found(t *testing.T) {
	dataSourceName := "data.kubernetes_namespace_v1.test"
	name := "ceci-n-est-pas-un-namespace"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "kubernetes_namespace_v1" "test" {
  metadata {
    name = "%s"
  }
}`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(dataSourceName, "spec.#", "0"),
				),
			},
		},
	})
}

// TestAccKubernetesDataSourceNamespaceV1_no_metadata verifies that omitting
// the metadata block entirely is caught at plan time by the
// listvalidator.SizeBetween(1,1) validator on the metadata block — the user
// gets a clear validation error before any API call is made.
func TestAccKubernetesDataSourceNamespaceV1_no_metadata(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "kubernetes_namespace_v1" "test" {
}`,
				ExpectError: regexp.MustCompile(`missing metadata block`),
			},
		},
	})
}
