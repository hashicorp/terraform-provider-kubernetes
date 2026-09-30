// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	batchapi "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestBatchCLI_JobIndexedComputedModeUpdate(t *testing.T) {
	batchCLIEnvironment(t)
	api := newBatchCLIAPI(t)
	initial := strings.Replace(batchCLIConfig(api, "jobs", batchCLIOptions{}), "spec {", `spec {
      completion_mode = "Indexed"
      completions = 4
      backoff_limit_per_index = 3
      max_failed_indexes = 1`, 1)
	updated := strings.Replace(initial, `completion_mode = "Indexed"`, "", 1)
	updated = strings.Replace(updated, "max_failed_indexes = 1", "max_failed_indexes = 2", 1)
	zero := strings.Replace(updated, "max_failed_indexes = 2", "max_failed_indexes = 0", 1)
	removed := strings.Replace(zero, "max_failed_indexes = 0", "", 1)
	var uid string
	check := func(updates int, expected int32) resource.TestCheckFunc {
		return api.check("jobs", &uid, 1, updates, func(object runtime.Object) error {
			job, ok := object.(*batchapi.Job)
			if !ok || job.Spec.CompletionMode == nil || *job.Spec.CompletionMode != batchapi.IndexedCompletion {
				return fmt.Errorf("Indexed completion mode was lost during an update")
			}
			value := job.Spec.MaxFailedIndexes
			if expected < 0 {
				if value != nil {
					return fmt.Errorf("maxFailedIndexes was not cleared: %d", *value)
				}
			} else if value == nil || *value != expected {
				return fmt.Errorf("maxFailedIndexes = %v, want %d", value, expected)
			}
			return nil
		})
	}
	updatePlan := batchCLISettled(plancheck.ExpectResourceAction(batchCLIAddress("jobs"), plancheck.ResourceActionUpdate))
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             api.checkDestroy,
		Steps: []resource.TestStep{
			{Config: initial, Check: check(0, 1)},
			{Config: updated, ConfigPlanChecks: updatePlan, Check: check(1, 2)},
			{Config: zero, ConfigPlanChecks: updatePlan, Check: check(2, 0)},
			{Config: removed, ConfigPlanChecks: updatePlan, Check: check(3, -1)},
			{Config: removed, ConfigPlanChecks: batchCLISettled(plancheck.ExpectEmptyPlan()), Check: check(3, -1)},
		},
	})
}
