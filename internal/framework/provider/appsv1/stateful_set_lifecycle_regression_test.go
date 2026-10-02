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

func TestAccKubernetesStatefulSetV1_claimQuantityEquivalence(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-sts-quantity")
	resourceName := "kubernetes_stateful_set_v1.test"
	canonical := testAccKubernetesStatefulSetV1ConfigBasic(name, agnhostImage)
	noncanonical := strings.Replace(canonical, `storage = "1Gi"`, `storage = "1024Mi"`, 1)
	larger := strings.Replace(canonical, `storage = "1Gi"`, `storage = "2Gi"`, 1)
	var snapshot statefulSetMigrationSnapshot
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) }, ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy: testAccCheckKubernetesStatefulSetV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: noncanonical,
				Check: resource.ComposeAggregateTestCheckFunc(
					statefulSetCaptureMigrationSnapshot(resourceName, &snapshot),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_claim_template.0.spec.0.resources.0.requests.storage", "1024Mi"),
				),
			},
			{
				Config:           canonical,
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
				Check: resource.ComposeAggregateTestCheckFunc(
					statefulSetCheckMigrationSnapshot(resourceName, &snapshot),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_claim_template.0.spec.0.resources.0.requests.storage", "1024Mi"),
				),
			},
			{
				Config: larger, PlanOnly: true, ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPreRefresh: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionDestroyBeforeCreate),
				}},
			},
			{
				Config:           canonical,
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
				Check:            statefulSetCheckMigrationSnapshot(resourceName, &snapshot),
			},
		},
	})
}

func TestAccKubernetesStatefulSetV1_addTypeOnlyUpdateStrategy(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-sts-strategy")
	resourceName := "kubernetes_stateful_set_v1.test"
	omitted := testAccKubernetesStatefulSetV1ConfigMinimal(name, busyboxImage)
	configured := strings.Replace(omitted, "  spec {", `  spec {
    update_strategy {
      type = "RollingUpdate"
    }`, 1)
	var snapshot statefulSetMigrationSnapshot
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) }, ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy: testAccCheckKubernetesStatefulSetV1Destroy,
		Steps: []resource.TestStep{
			{Config: omitted, Check: statefulSetCaptureMigrationSnapshot(resourceName, &snapshot)},
			{
				Config: configured,
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
				}},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_strategy.0.rolling_update.#", "0"),
					statefulSetCheckMigrationSnapshot(resourceName, &snapshot),
				),
			},
			{
				Config:           configured,
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
				Check:            statefulSetCheckMigrationSnapshot(resourceName, &snapshot),
			},
		},
	})
}
