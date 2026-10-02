// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1_test

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestBatchCLI_IndexedComputedModeUpdate(t *testing.T) {
	batchCLIEnvironment(t)
	for _, kind := range []string{"jobs", "cronjobs"} {
		t.Run(kind, func(t *testing.T) {
			api := newBatchCLIAPI(t)
			initial := batchCLIRetryConfig(api, kind, "backoff_limit_per_index = 3\nmax_failed_indexes = 1")
			updated := strings.Replace(initial, `completion_mode = "Indexed"`, "", 1)
			updated = strings.Replace(updated, "max_failed_indexes = 1", "max_failed_indexes = 2", 1)
			zero := strings.Replace(updated, "max_failed_indexes = 2", "max_failed_indexes = 0", 1)
			removed := strings.Replace(zero, "max_failed_indexes = 0", "", 1)
			var uid string
			check := func(updates int, expected int32) resource.TestCheckFunc {
				return api.check(kind, &uid, 1, updates, batchCLIRetryValues(3, expected))
			}
			updatePlan := batchCLISettled(plancheck.ExpectResourceAction(batchCLIAddress(kind), plancheck.ResourceActionUpdate))
			emptyPlan := batchCLISettled(plancheck.ExpectEmptyPlan())
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				CheckDestroy:             api.checkDestroy,
				Steps: []resource.TestStep{
					{Config: initial, Check: check(0, 1)},
					{Config: updated, ConfigPlanChecks: updatePlan, Check: check(1, 2)},
					{Config: zero, ConfigPlanChecks: updatePlan, Check: check(2, 0)},
					{Config: removed, ConfigPlanChecks: emptyPlan, Check: check(2, 0)},
					{Config: updated, ConfigPlanChecks: updatePlan, Check: check(3, 2)},
					{Config: removed, ConfigPlanChecks: updatePlan, Check: check(4, 0)},
					{Config: removed, ConfigPlanChecks: emptyPlan, Check: check(4, 0)},
				},
			})
		})
	}
}
