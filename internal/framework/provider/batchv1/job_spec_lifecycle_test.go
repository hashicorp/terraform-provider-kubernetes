// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	batchapi "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestJobSpecCLI_PodFailurePolicyUpdate(t *testing.T) {
	workspaces := batchCLIEnvironment(t)
	for _, released := range []bool{true, false} {
		name := "framework"
		if released {
			name = "released-3.2.1"
		}
		t.Run(name, func(t *testing.T) {
			api := newBatchCLIAPI(t)
			base := batchCLIConfig(api, "cronjobs", batchCLIOptions{legacy: released})
			initial := strings.Replace(base, "\n      template {", `
      pod_failure_policy {
        rule {
          action = "Ignore"
          on_exit_codes {
            container_name = "task"
            operator = "In"
            values = [1]
          }
        }
      }
      template {`, 1)
			if initial == base {
				t.Fatal("pod failure policy was not added to the fixture")
			}
			updated := strings.Replace(initial, `action = "Ignore"`, `action = "Count"`, 1)
			validate := func(action batchapi.PodFailurePolicyAction) func(runtime.Object) error {
				return func(object runtime.Object) error {
					cron, ok := object.(*batchapi.CronJob)
					if !ok {
						return fmt.Errorf("expected CronJob, got %T", object)
					}
					policy := cron.Spec.JobTemplate.Spec.PodFailurePolicy
					if policy == nil || len(policy.Rules) != 1 || policy.Rules[0].Action != action {
						return fmt.Errorf("unexpected pod failure policy: %+v", policy)
					}
					return nil
				}
			}
			var uid string
			initialCheck := api.checkState("cronjobs", &uid, 1, 0, validate(batchapi.PodFailurePolicyActionIgnore), released)
			if released {
				initialCheck = resource.ComposeTestCheckFunc(initialCheck, batchCLIReleasedVersion(t, workspaces))
			}
			steps := []resource.TestStep{
				{Config: initial, Check: initialCheck},
				{
					Config: updated,
					ConfigPlanChecks: batchCLISettled(plancheck.ExpectResourceAction(
						batchCLIAddress("cronjobs"), plancheck.ResourceActionUpdate)),
					Check: api.checkState("cronjobs", &uid, 1, 1, validate(batchapi.PodFailurePolicyActionCount), released),
				},
				{
					Config: updated, ConfigPlanChecks: batchCLISettled(plancheck.ExpectEmptyPlan()),
					Check: api.checkState("cronjobs", &uid, 1, 1, validate(batchapi.PodFailurePolicyActionCount), released),
				},
			}
			for i := range steps {
				if released {
					steps[i].ExternalProviders = map[string]resource.ExternalProvider{
						"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.2.1"},
					}
				} else {
					steps[i].ProtoV6ProviderFactories = testAccProtoV6ProviderFactories
				}
			}
			resource.UnitTest(t, resource.TestCase{CheckDestroy: api.checkDestroy, Steps: steps})
		})
	}
}

func TestJobSpecCLI_JobPodFailurePolicyEdit(t *testing.T) {
	workspaces := batchCLIEnvironment(t)
	api := newBatchCLIAPI(t)
	base := batchCLIConfig(api, "jobs", batchCLIOptions{})
	initial := strings.Replace(base, "\n      template {", `
      pod_failure_policy {
        rule {
          action = "Ignore"
          on_exit_codes {
            container_name = "task"
            operator = "In"
            values = [1]
          }
        }
      }
      template {`, 1)
	if initial == base {
		t.Fatal("pod failure policy was not added to the fixture")
	}
	updated := strings.Replace(initial, `action = "Ignore"`, `action = "Count"`, 1)
	validate := func(action batchapi.PodFailurePolicyAction) func(runtime.Object) error {
		return func(object runtime.Object) error {
			job, ok := object.(*batchapi.Job)
			if !ok {
				return fmt.Errorf("expected Job, got %T", object)
			}
			policy := job.Spec.PodFailurePolicy
			if policy == nil || len(policy.Rules) != 1 || policy.Rules[0].Action != action {
				return fmt.Errorf("unexpected pod failure policy: %+v", policy)
			}
			return nil
		}
	}
	var initialUID, replacementUID string
	initialSpec := batchCLISpecContinuity(validate(batchapi.PodFailurePolicyActionIgnore))
	resource.UnitTest(t, resource.TestCase{
		CheckDestroy: api.checkDestroy,
		Steps: []resource.TestStep{
			{
				Config: initial,
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.2.1"},
				},
				Check: resource.ComposeTestCheckFunc(
					api.checkState("jobs", &initialUID, 1, 0, initialSpec, true),
					batchCLIReleasedVersion(t, workspaces),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   initial,
				ConfigPlanChecks: batchCLISettled(plancheck.ExpectResourceAction(
					batchCLIAddress("jobs"), plancheck.ResourceActionUpdate)),
				Check: api.check("jobs", &initialUID, 1, 0, initialSpec),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   updated,
				ConfigPlanChecks: batchCLISettled(plancheck.ExpectResourceAction(
					batchCLIAddress("jobs"), plancheck.ResourceActionDestroyBeforeCreate)),
				Check: resource.ComposeTestCheckFunc(
					api.check("jobs", &replacementUID, 2, 0, validate(batchapi.PodFailurePolicyActionCount)),
					func(_ *terraform.State) error {
						if replacementUID == initialUID {
							return fmt.Errorf("immutable Job pod failure policy edit retained UID %q", initialUID)
						}
						return nil
					},
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   updated, ConfigPlanChecks: batchCLISettled(plancheck.ExpectEmptyPlan()),
				Check: api.check("jobs", &replacementUID, 2, 0, validate(batchapi.PodFailurePolicyActionCount)),
			},
		},
	})
}
