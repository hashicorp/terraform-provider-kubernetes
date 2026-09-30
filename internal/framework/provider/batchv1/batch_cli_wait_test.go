// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	batchapi "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestBatchCLI_JobCompletionAndTTL(t *testing.T) {
	batchCLIEnvironment(t)
	t.Run("retained-completed-job", func(t *testing.T) {
		api := newBatchCLIAPI(t)
		config := batchCLIConfig(api, "jobs", batchCLIOptions{ttl: "60"})
		updated := batchCLIConfig(api, "jobs", batchCLIOptions{ttl: "120"})
		var uid string
		resource.UnitTest(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			CheckDestroy:             api.checkDestroy,
			Steps: []resource.TestStep{
				{
					Config: config,
					Check: api.check("jobs", &uid, 1, 0, func(object runtime.Object) error {
						job := object.(*batchapi.Job)
						if job.Spec.TTLSecondsAfterFinished == nil || *job.Spec.TTLSecondsAfterFinished != 60 {
							return fmt.Errorf("TTL did not reach the API: %#v", job.Spec.TTLSecondsAfterFinished)
						}
						if api.calls["GET jobs"] == 0 {
							return fmt.Errorf("default completion waiter never read the completed Job")
						}
						return nil
					}),
				},
				{Config: config, ConfigPlanChecks: batchCLISettled(plancheck.ExpectEmptyPlan()), Check: api.check("jobs", &uid, 1, 0, nil)},
				{
					Config:           updated,
					ConfigPlanChecks: batchCLISettled(plancheck.ExpectResourceAction(batchCLIAddress("jobs"), plancheck.ResourceActionUpdate)),
					Check: api.check("jobs", &uid, 1, 1, func(object runtime.Object) error {
						job := object.(*batchapi.Job)
						if job.Spec.TTLSecondsAfterFinished == nil || *job.Spec.TTLSecondsAfterFinished != 120 {
							return fmt.Errorf("TTL-only update did not reach the API: %#v", job.Spec.TTLSecondsAfterFinished)
						}
						return nil
					}),
				},
				{Config: updated, ConfigPlanChecks: batchCLISettled(plancheck.ExpectEmptyPlan()), Check: api.check("jobs", &uid, 1, 1, nil)},
			},
		})
	})
	t.Run("zero-ttl-disappears-before-wait", func(t *testing.T) {
		api := newBatchCLIAPI(t)
		api.expireZeroTTL = true
		config := batchCLIConfig(api, "jobs", batchCLIOptions{ttl: "0"})
		resource.UnitTest(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			CheckDestroy:             api.checkDestroy,
			Steps: []resource.TestStep{{
				Config: config,
				// An immediately expired managed Job necessarily plans creation
				// again. This asserts the precise expected plan, not convergence.
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(batchCLIAddress("jobs"), plancheck.ResourceActionCreate),
					},
				},
				Check: func(_ *terraform.State) error {
					api.mu.Lock()
					defer api.mu.Unlock()
					if api.calls["POST jobs"] != 1 || api.calls["TTL jobs"] != 1 || len(api.jobs) != 0 {
						return fmt.Errorf("zero-TTL waiter did not tolerate API disappearance: calls=%v remaining=%d", api.calls, len(api.jobs))
					}
					return nil
				},
			}},
		})
	})
	t.Run("failed-job-keeps-cleanup-identity", func(t *testing.T) {
		api := newBatchCLIAPI(t)
		api.jobFails = true
		resource.UnitTest(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			CheckDestroy:             api.checkDestroy,
			Steps: []resource.TestStep{{
				Config:      batchCLIConfig(api, "jobs", batchCLIOptions{}),
				ExpectError: regexp.MustCompile(`is in failed state`),
			}},
		})
		api.mu.Lock()
		defer api.mu.Unlock()
		if api.calls["POST jobs"] != 1 || api.calls["GET jobs"] == 0 || api.calls["DELETE jobs"] != 1 {
			t.Errorf("failed creation must retain enough state to clean up: %v", api.calls)
		}
	})
}
