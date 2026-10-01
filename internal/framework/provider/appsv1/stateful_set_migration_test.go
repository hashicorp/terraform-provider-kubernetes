// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	appsv1 "k8s.io/api/apps/v1"
)

const statefulSetSDKv2ProviderVersion = "3.2.1"

func TestAccKubernetesStatefulSetV1_migrationFromSDKv2(t *testing.T) {
	var before, after appsv1.StatefulSet
	var snapshot statefulSetMigrationSnapshot
	name := acctest.RandomWithPrefix("tf-apps-sts-migrate")
	config := testAccKubernetesStatefulSetV1ConfigMinimal(name, busyboxImage)
	resourceName := "kubernetes_stateful_set_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckKubernetesStatefulSetV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						Source:            "hashicorp/kubernetes",
						VersionConstraint: statefulSetSDKv2ProviderVersion,
					},
				},
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &before),
					statefulSetCaptureMigrationSnapshot(resourceName, &snapshot),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &after),
					testAccCheckKubernetesStatefulSetForceNew(&before, &after, false),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					statefulSetCheckMigrationSnapshot(resourceName, &snapshot),
				),
			},
		},
	})
}

func TestAccKubernetesStatefulSetV1_aliasMove(t *testing.T) {
	var before, after appsv1.StatefulSet
	var snapshot statefulSetMigrationSnapshot
	name := acctest.RandomWithPrefix("tf-apps-sts-move")
	targetConfig := testAccKubernetesStatefulSetV1ConfigMinimal(name, busyboxImage)
	sourceConfig := strings.Replace(targetConfig, `resource "kubernetes_stateful_set_v1" "test"`, `resource "kubernetes_stateful_set" "test"`, 1)
	movedConfig := fmt.Sprintf(`%s
moved {
  from = kubernetes_stateful_set.test
  to   = kubernetes_stateful_set_v1.test
}
`, targetConfig)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckKubernetesStatefulSetV1Destroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						Source:            "hashicorp/kubernetes",
						VersionConstraint: statefulSetSDKv2ProviderVersion,
					},
				},
				Config: sourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists("kubernetes_stateful_set.test", &before),
					statefulSetCaptureMigrationSnapshot("kubernetes_stateful_set.test", &snapshot),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   movedConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists("kubernetes_stateful_set_v1.test", &after),
					testAccCheckKubernetesStatefulSetForceNew(&before, &after, false),
					statefulSetCheckMigrationSnapshot("kubernetes_stateful_set_v1.test", &snapshot),
				),
			},
		},
	})
}
