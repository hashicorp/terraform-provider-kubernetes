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

func TestAccKubernetesStatefulSetV1_claimMetadataRemoval(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-sts-claim-metadata")
	resourceName := "kubernetes_stateful_set_v1.test"
	omitted := testAccKubernetesStatefulSetV1ConfigBasic(name, agnhostImage)
	configured := strings.Replace(omitted, `    volume_claim_template {
      metadata {`, `    volume_claim_template {
      metadata {
        labels = { managed = "true" }
        annotations = { managed = "true" }`, 1)
	var snapshot statefulSetMigrationSnapshot
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckKubernetesStatefulSetV1Destroy,
		Steps: []resource.TestStep{
			{Config: configured, Check: statefulSetCaptureMigrationSnapshot(resourceName, &snapshot)},
			{
				Config: omitted, PlanOnly: true, ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPreRefresh: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionDestroyBeforeCreate),
				}},
			},
			{
				Config:           configured,
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
				Check:            statefulSetCheckMigrationSnapshot(resourceName, &snapshot),
			},
		},
	})
}
