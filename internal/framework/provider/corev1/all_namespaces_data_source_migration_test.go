// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

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
