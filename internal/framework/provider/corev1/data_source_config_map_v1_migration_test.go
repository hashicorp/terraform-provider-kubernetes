// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccKubernetesDataSourceConfigMapV1_migration proves that a user who had
// data "kubernetes_config_map_v1" written against the SDKv2 provider can upgrade
// to the Framework provider without changing a single line of their HCL config.
//
// Step 1: applies with the released SDKv2 hashicorp/kubernetes provider.
// Step 2: applies the IDENTICAL config with the local Framework provider — plan must be empty.
func TestAccKubernetesDataSourceConfigMapV1_migration(t *testing.T) {
	dataSourceName := "data.kubernetes_config_map_v1.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				// Step 1: apply with the released SDKv2 provider (last major series).
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						Source:            "hashicorp/kubernetes",
						VersionConstraint: "~> 2.0",
					},
				},
				Config: testAccKubernetesDataSourceConfigMapV1_migrationConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(dataSourceName, "data.one", "first"),
				),
			},
			{
				// Step 2: same HCL config, now served by the local Framework provider.
				// The plan must be empty — proving the transition is fully transparent.
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesDataSourceConfigMapV1_migrationConfig(name),
				ExpectNonEmptyPlan:       false,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(dataSourceName, "data.one", "first"),
				),
			},
		},
	})
}

// testAccKubernetesDataSourceConfigMapV1_migrationConfig returns the combined resource +
// data source HCL used in both steps of the migration test. The config is byte-for-byte
// identical between the SDKv2 step and the Framework step.
func testAccKubernetesDataSourceConfigMapV1_migrationConfig(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_config_map_v1" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "first"
  }
}

data "kubernetes_config_map_v1" "test" {
  metadata {
    name = kubernetes_config_map_v1.test.metadata.0.name
  }
}
`, name)
}
