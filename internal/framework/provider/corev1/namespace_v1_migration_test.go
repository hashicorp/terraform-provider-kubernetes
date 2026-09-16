// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

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
