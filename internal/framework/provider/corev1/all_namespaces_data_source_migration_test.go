// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccKubernetesDataSourceAllNamespaces_MigrateFromSDKv2(t *testing.T) {
	//  Every other acceptance test in this package creates and deletes namespaces — tf-acc-ns-*, tf-migration-test-*.
	//  If one lands between the two steps, the hash changes, the plan is legitimately non-empty, and the test fails for a reason unrelated to the code under test.
	//  Go defers every t.Parallel() test until the sequential ones have finished, so a resource.Test runs alone relative to its ParallelTest siblings in the same package.
	resource.Test(t, resource.TestCase{
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
