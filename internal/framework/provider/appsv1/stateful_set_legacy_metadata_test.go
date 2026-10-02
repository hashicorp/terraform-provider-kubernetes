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

func TestAccKubernetesStatefulSetV1_migrationEmptyClaimMetadata(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-sts-empty-metadata")
	resourceName := "kubernetes_stateful_set_v1.test"
	config := statefulSetFullMigrationConfig(name, false)
	legacy := strings.Replace(statefulSetFullMigrationConfig(name, true), `    volume_claim_template {
      metadata {`, `    volume_claim_template {
      metadata {
        labels = {}
        annotations = {}`, 1)
	var snapshot statefulSetMigrationSnapshot

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckKubernetesStatefulSetV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{"kubernetes": {
					Source: "hashicorp/kubernetes", VersionConstraint: statefulSetSDKv2ProviderVersion,
				}},
				Config: legacy,
				Check:  statefulSetCaptureMigrationSnapshot(resourceName, &snapshot),
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: statefulSetCheckMigrationSnapshot(resourceName, &snapshot),
			},
		},
	})
}
