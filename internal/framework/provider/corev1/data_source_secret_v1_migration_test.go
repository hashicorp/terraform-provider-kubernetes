// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
)

// TestAccKubernetesDataSourceSecretV1_migration is a two-step migration test that proves
// identical HCL works across an external SDKv2 release and the local Framework provider:
//
//   - Step 1: Apply with the released SDKv2-based provider (hashicorp/kubernetes ~> 2.38).
//     This is the last pure-SDKv2 major line, confirming the pre-migration baseline.
//   - Step 2: Apply the byte-for-byte identical HCL using the local mux provider.
//     The Framework data source reads the same Secret; the plan must be empty
//     (ExpectNonEmptyPlan: false), proving zero breaking change for users upgrading.
func TestAccKubernetesDataSourceSecretV1_migration(t *testing.T) {
	name := fmt.Sprintf("tf-acc-mig-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	datasourceName := "data.kubernetes_secret_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				// Step 1: apply with the released SDKv2-based provider.
				// hashicorp/kubernetes 2.38.x is the last pure-SDKv2 release line.
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						Source:            "hashicorp/kubernetes",
						VersionConstraint: "~> 2.38",
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
				ProtoV6ProviderFactories: testAccMuxProviderFactories,
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

var testAccMuxProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": func() (tfprotov6.ProviderServer, error) {
		return mux.MuxServer(context.Background(), "test")
	},
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
