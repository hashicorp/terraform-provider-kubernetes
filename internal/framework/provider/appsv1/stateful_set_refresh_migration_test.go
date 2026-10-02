// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1_test

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccKubernetesStatefulSetV1_migrationAfterSDKRefresh(t *testing.T) {
	for _, scenario := range []string{"minimal", "full", "alias", "explicit-empty-template-namespace"} {
		t.Run(scenario, func(t *testing.T) {
			name := acctest.RandomWithPrefix("tf-sts-refresh")
			resourceName := "kubernetes_stateful_set_v1.test"
			legacyResourceName := resourceName
			config := testAccKubernetesStatefulSetV1ConfigMinimal(name, busyboxImage)
			legacy := config
			switch scenario {
			case "explicit-empty-template-namespace":
				config = withEmptyTemplateNamespace(t, config)
				legacy = config
			case "full":
				config = statefulSetFullMigrationConfig(name, false)
				legacy = statefulSetFullMigrationConfig(name, true)
			case "alias":
				legacyResourceName = "kubernetes_stateful_set.test"
				legacy = strings.ReplaceAll(legacy, `"kubernetes_stateful_set_v1"`, `"kubernetes_stateful_set"`)
				config += `
moved {
  from = kubernetes_stateful_set.test
  to   = kubernetes_stateful_set_v1.test
}
`
			}
			external := map[string]resource.ExternalProvider{"kubernetes": {
				Source: "hashicorp/kubernetes", VersionConstraint: statefulSetSDKv2ProviderVersion,
			}}
			var snapshot statefulSetMigrationSnapshot
			resource.ParallelTest(t, resource.TestCase{
				PreCheck:     func() { testAccPreCheck(t) },
				CheckDestroy: testAccCheckKubernetesStatefulSetV1Destroy,
				Steps: []resource.TestStep{
					{
						ExternalProviders: external,
						Config:            legacy,
						Check:             statefulSetCaptureMigrationSnapshot(legacyResourceName, &snapshot),
					},
					{
						ExternalProviders: external,
						RefreshState:      true,
						// Per-step provider switching clears configuration for refresh.
						// The follow-up destroy plan is checked but never applied.
						ExpectNonEmptyPlan: true,
						RefreshPlanChecks: resource.RefreshPlanChecks{PostRefresh: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(legacyResourceName, plancheck.ResourceActionDestroy),
						}},
						Check: statefulSetCheckMigrationSnapshot(legacyResourceName, &snapshot),
					},
					{
						ProtoV6ProviderFactories: testAccProviderFactories,
						Config:                   config,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check:             statefulSetCheckMigrationSnapshot(resourceName, &snapshot),
						ConfigStateChecks: statefulSetIdentityChecks(resourceName, name),
					},
				},
			})
		})
	}
}
