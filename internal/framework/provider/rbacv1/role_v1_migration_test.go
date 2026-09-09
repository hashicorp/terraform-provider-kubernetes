// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// TestAccRole_UpgradeFromSDKv2 verifies that state created by the last
// published SDKv2 release of kubernetes_role_v1 is read identically by the
// Framework implementation (empty plan) and that subsequent updates still
// work. There is deliberately no UpgradeState/schema-version machinery
// involved here: metadata is a ListNestedBlock, which is wire-identical to
// SDKv2's TypeList{MaxItems:1}, so this test is exercising that the two
// schemas really do produce byte-identical state, not a reshaping path.
func TestAccRole_UpgradeFromSDKv2(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_role_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						VersionConstraint: "3.2.1",
						Source:            "hashicorp/kubernetes",
					},
				},
				Config: testAccRoleConfig_basic(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "rule.#", "2"),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccRoleConfig_basic(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccRoleConfig_modified(name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "rule.#", "1"),
				),
			},
		},
	})
}
