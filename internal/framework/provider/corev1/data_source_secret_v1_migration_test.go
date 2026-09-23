// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// providerVersion is the last released version of hashicorp/kubernetes that served
// kubernetes_secret_v1 as an SDKv2 data source. Pinned to an exact version (no ~> or >=)
// so the migration baseline is deterministic across CI runs (K8S-MIGRATE-017).
const providerVersion = "3.2.1"

// TestAccKubernetesDataSourceSecretV1_migration is a two-step migration test that proves
// identical HCL works across the last released SDKv2-backed provider and the local
// Framework build:
//
//   - Step 1: Apply with the released provider pinned to providerVersion (3.2.1).
//     This is the exact released state real users hold before upgrading.
//   - Step 2: Apply the byte-for-byte identical HCL using the local mux provider.
//     The Framework data source reads the same Secret; the plan must be empty,
//     proving zero breaking change for users upgrading from 3.2.1.
func TestAccKubernetesDataSourceSecretV1_migration(t *testing.T) {
	name := fmt.Sprintf("tf-acc-mig-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	datasourceName := "data.kubernetes_secret_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				// Step 1: apply with the exact released provider (K8S-MIGRATE-017).
				// providerVersion is the last release that served kubernetes_secret_v1 via SDKv2.
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						Source:            "hashicorp/kubernetes",
						VersionConstraint: providerVersion,
					},
				},
				Config: testAccKubernetesDataSourceSecretV1MigrationConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(datasourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(datasourceName, "data.one", "first"),
					resource.TestCheckResourceAttr(datasourceName, "type", "Opaque"),
				),
			},
			{
				// Step 2: same config, served by the local Framework/mux provider.
				// Data sources carry no persisted state, so the plan must be empty.
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesDataSourceSecretV1MigrationConfig(name),
				ExpectNonEmptyPlan:       false,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(datasourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(datasourceName, "data.one", "first"),
					resource.TestCheckResourceAttr(datasourceName, "type", "Opaque"),
				),
			},
		},
	})
}

// testAccKubernetesDataSourceSecretV1MigrationConfig is shared verbatim between both
// test steps — it is the proof that no HCL change is required when upgrading providers.
func testAccKubernetesDataSourceSecretV1MigrationConfig(name string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
  metadata {
    name = %q
  }
  data = {
    one = "first"
    two = "second"
  }
  type = "Opaque"
}

data "kubernetes_secret_v1" "test" {
  metadata {
    name = kubernetes_secret_v1.test.metadata.0.name
  }
}
`, name)
}
