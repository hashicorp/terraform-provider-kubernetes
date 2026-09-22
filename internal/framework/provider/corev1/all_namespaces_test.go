// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccKubernetesDataSourceAllNamespaces_basic verifies that the framework
// implementation reads all expected attributes from a live cluster.
// Mirrors TestAccKubernetesDataSourceAllNamespaces_basic in
// kubernetes/data_source_kubernetes_all_namespaces_test.go.
func TestAccKubernetesDataSourceAllNamespaces_basic(t *testing.T) {
	dataSourceName := "data.kubernetes_all_namespaces.test"
	rxPosNum := regexp.MustCompile("^[1-9][0-9]*$")
	rxNsName := regexp.MustCompile(`^[a-zA-Z][-\w]*$`)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAllNamespacesConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// id is computed — just assert it is set to something non-empty.
					resource.TestCheckResourceAttrSet(dataSourceName, "id"),
					// The cluster always has at least one namespace (default).
					resource.TestMatchResourceAttr(dataSourceName, "namespaces.#", rxPosNum),
					// The first namespace must be set and match the naming regex.
					resource.TestCheckResourceAttrSet(dataSourceName, "namespaces.0"),
					resource.TestMatchResourceAttr(dataSourceName, "namespaces.0", rxNsName),
				),
			},
		},
	})
}

func testAllNamespacesConfig() string {
	return `data "kubernetes_all_namespaces" "test" {}`
}
