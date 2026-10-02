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

func TestBatchCLI_CronSingleFieldChanges(t *testing.T) {
	workspaces := batchCLIEnvironment(t)
	for _, implementation := range []string{"released-3.2.1", "local"} {
		for _, field := range []string{"image", "container-cpu-limit", "completions"} {
			t.Run(implementation+"/"+field, func(t *testing.T) {
				api := newBatchCLIAPI(t)
				released := implementation == "released-3.2.1"
				initial := batchCLIConfig(api, "cronjobs", batchCLIOptions{
					rich: field == "container-cpu-limit", legacy: released,
				})
				var updated string
				action, creates, updates := plancheck.ResourceActionUpdate, 1, 1
				switch field {
				case "image":
					updated = strings.Replace(initial, `image = "busybox:1.36"`, `image = "busybox:1.37"`, 1)
				case "container-cpu-limit":
					before, container, ok := strings.Cut(initial, "          container {")
					if !ok {
						t.Fatal("main container not found")
					}
					updated = before + "          container {" + strings.Replace(container, `limits = { cpu = "100m"`, `limits = { cpu = "200m"`, 1)
				case "completions":
					updated = strings.Replace(initial, "      template {", "      completions = 2\n      template {", 1)
					action, creates, updates = plancheck.ResourceActionDestroyBeforeCreate, 2, 0
				}
				if initial == updated || updated == "" {
					t.Fatalf("fixture did not change %s", field)
				}
				validate := func(object runtime.Object) error {
					spec := object.(*batchapi.CronJob).Spec.JobTemplate.Spec
					switch field {
					case "image":
						if spec.Template.Spec.Containers[0].Image != "busybox:1.37" {
							return fmt.Errorf("image-only update did not reach API")
						}
					case "container-cpu-limit":
						container := spec.Template.Spec.Containers[0]
						if container.Resources.Limits.Cpu().MilliValue() != 200 ||
							container.Resources.Requests.Cpu().MilliValue() != 100 ||
							spec.Template.Spec.InitContainers[0].Resources.Limits.Cpu().MilliValue() != 100 {
							return fmt.Errorf("single CPU limit change did not preserve all other quantities")
						}
					case "completions":
						if spec.Completions == nil || *spec.Completions != 2 {
							return fmt.Errorf("completions change did not reach API")
						}
					}
					return nil
				}
				var uid, updatedUID string
				testCase := resource.TestCase{
					CheckDestroy: api.checkDestroy,
					Steps: []resource.TestStep{
						{Config: initial, ConfigPlanChecks: batchCLISettled(), Check: api.checkState("cronjobs", &uid, 1, 0, nil, released)},
						{
							Config:           updated,
							ConfigPlanChecks: batchCLISettled(plancheck.ExpectResourceAction(batchCLIAddress("cronjobs"), action)),
							Check: resource.ComposeTestCheckFunc(
								api.checkState("cronjobs", &updatedUID, creates, updates, validate, released),
								func(_ *terraform.State) error {
									if (action == plancheck.ResourceActionUpdate) != (uid == updatedUID) {
										return fmt.Errorf("%s has incorrect UID continuity: %s -> %s", field, uid, updatedUID)
									}
									return nil
								},
							),
						},
						{Config: updated, ConfigPlanChecks: batchCLISettled(plancheck.ExpectEmptyPlan()), Check: api.checkState("cronjobs", &updatedUID, creates, updates, validate, released)},
					},
				}
				if released {
					for index := range testCase.Steps {
						testCase.Steps[index].ExternalProviders = map[string]resource.ExternalProvider{
							"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.2.1"},
						}
					}
					testCase.Steps[0].Check = resource.ComposeTestCheckFunc(testCase.Steps[0].Check, batchCLIReleasedVersion(t, workspaces))
				} else {
					testCase.ProtoV6ProviderFactories = testAccProtoV6ProviderFactories
				}
				resource.UnitTest(t, testCase)
			})
		}
	}
}
