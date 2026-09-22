// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// testAllNamespacesMigrationConfig returns HCL that anchors every attribute
// produced by the data source into a terraform_data resource. This gives
// Terraform something concrete to diff between Step 1 (SDKv2 state) and
// Step 2 (framework state), making ExpectEmptyPlan() a meaningful assertion.
func testAllNamespacesMigrationConfig() string {
	return `
data "kubernetes_all_namespaces" "test" {}

resource "terraform_data" "test" {
  input = {
    id         = data.kubernetes_all_namespaces.test.id
    namespaces = data.kubernetes_all_namespaces.test.namespaces
  }
}
`
}

// TestAccKubernetesDataSourceAllNamespaces_MigrateFromSDKv2 is the migration
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
func TestAccKubernetesDataSourceAllNamespaces_MigrateFromSDKv2(t *testing.T) {
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
				Config: testAllNamespacesMigrationConfig(),
			},
			{
				// Step 2: switch to the local framework implementation and
				// verify no planned changes are produced.
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAllNamespacesMigrationConfig(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}
