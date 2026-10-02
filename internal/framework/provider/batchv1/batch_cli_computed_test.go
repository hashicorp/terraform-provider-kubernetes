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

func TestBatchCLI_ComputedAdmissionValues(t *testing.T) {
	batchCLIEnvironment(t)
	for _, kind := range []string{"jobs", "cronjobs"} {
		for _, shape := range []string{"omitted-resources", "limits-only"} {
			t.Run(kind+"/"+shape, func(t *testing.T) {
				api := newBatchCLIAPI(t)
				api.admitComputed = true
				config := batchCLIConfig(api, kind, batchCLIOptions{rich: shape == "limits-only"})
				config = strings.ReplaceAll(config, `          image_pull_secrets = [{ name = "private-registry" }]`, "")
				config = strings.ReplaceAll(config, `          readiness_gate = [{ condition_type = "example.com/ready" }]`, "")
				config = strings.ReplaceAll(config, `              requests = { cpu = "100m", memory = "16Mi" }`, "")
				address := batchCLIAddress(kind)
				podPath := "spec.0.template.0.spec.0."
				if kind == "cronjobs" {
					podPath = "spec.0.job_template.0.spec.0.template.0.spec.0."
				}
				var uid string
				checks := []resource.TestCheckFunc{
					api.check(kind, &uid, 1, 0, func(object runtime.Object) error {
						var spec batchapi.JobSpec
						switch object := object.(type) {
						case *batchapi.Job:
							spec = object.Spec
						case *batchapi.CronJob:
							spec = object.Spec.JobTemplate.Spec
						}
						pod := spec.Template.Spec
						if len(pod.ImagePullSecrets) != 1 || pod.ImagePullSecrets[0].Name != "admission-registry" ||
							len(pod.ReadinessGates) != 1 || pod.ReadinessGates[0].ConditionType != "example.com/admitted" ||
							pod.Containers[0].Resources.Requests.Cpu().MilliValue() != 50 {
							return fmt.Errorf("admission fixture fields were lost: %#v", pod)
						}
						return nil
					}),
					resource.TestCheckResourceAttr(address, podPath+"image_pull_secrets.0.name", "admission-registry"),
					resource.TestCheckResourceAttr(address, podPath+"readiness_gate.0.condition_type", "example.com/admitted"),
					resource.TestCheckResourceAttr(address, podPath+"container.0.resources.#", "1"),
					resource.TestCheckResourceAttr(address, podPath+"container.0.resources.0.requests.cpu", "50m"),
					resource.TestCheckResourceAttr(address, podPath+"container.0.resources.0.requests.memory", "8Mi"),
				}
				if shape == "limits-only" {
					checks = append(checks,
						resource.TestCheckResourceAttr(address, podPath+"container.0.resources.0.limits.cpu", "100m"),
						resource.TestCheckResourceAttr(address, podPath+"init_container.0.resources.0.requests.cpu", "50m"),
						resource.TestCheckResourceAttr(address, podPath+"init_container.0.resources.0.limits.cpu", "100m"),
					)
				}
				resource.UnitTest(t, resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					CheckDestroy:             api.checkDestroy,
					Steps: []resource.TestStep{
						{Config: config, ConfigPlanChecks: batchCLISettled(), Check: resource.ComposeTestCheckFunc(checks...)},
						{Config: config, ConfigPlanChecks: batchCLISettled(plancheck.ExpectEmptyPlan()), Check: resource.ComposeTestCheckFunc(checks...)},
					},
				})
			})
		}
	}
}
