// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	batchapi "k8s.io/api/batch/v1"
)

func TestAccKubernetesJobV1_pod_failure_policy_update(t *testing.T) {
	name := fmt.Sprintf("tf-acc-policy-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	initial := strings.Replace(testAccKubernetesJobV1Config_basic(name, busyboxImage),
		`container_name = "hello"`, "", 1)
	updated := strings.Replace(initial, `action = "FailJob"`, `action = "Ignore"`, 1)
	var before, after batchapi.Job
	const address = "kubernetes_job_v1.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.26.0")
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesJobV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: initial,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesJobV1Exists(address, &before),
					resource.TestCheckResourceAttr(address, "spec.0.pod_failure_policy.0.rule.0.on_exit_codes.0.container_name", ""),
				),
			},
			{
				Config: updated,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(address, plancheck.ResourceActionDestroyBeforeCreate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesJobV1Exists(address, &after),
					testAccCheckKubernetesJobV1ForceNew(&before, &after, true),
					resource.TestCheckResourceAttr(address, "spec.0.pod_failure_policy.0.rule.0.action", "Ignore"),
					resource.TestCheckResourceAttr(address, "spec.0.pod_failure_policy.0.rule.0.on_exit_codes.0.container_name", ""),
				),
			},
			{
				Config: updated,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}
