// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestCronJobCLIComputedContainersFollowName(t *testing.T) {
	batchCLIEnvironment(t)
	for _, field := range []string{"container", "init_container"} {
		for _, operation := range []string{"remove first", "reorder", "new name"} {
			t.Run(field+"/"+operation, func(t *testing.T) {
				api := newBatchCLIAPI(t)
				names := []string{"second"}
				if operation == "reorder" {
					names = append(names, "first")
				} else if operation == "new name" {
					names = []string{"added", "second"}
				}
				before := cronJobCLIContainerConfig(api, field, []string{"first", "second"})
				after := cronJobCLIContainerConfig(api, field, names)
				var uid string
				validate := func(object runtime.Object) error {
					pod := object.(*batch.CronJob).Spec.JobTemplate.Spec.Template.Spec
					containers := pod.Containers
					if field == "init_container" {
						containers = pod.InitContainers
					}
					if len(containers) != len(names) {
						return fmt.Errorf("containers=%d, want %d", len(containers), len(names))
					}
					for i, container := range containers {
						cpu, policy := int64(200), core.PullIfNotPresent
						if names[i] == "first" {
							cpu, policy = 100, core.PullAlways
						} else if names[i] == "added" {
							cpu = 0
						}
						if container.Name != names[i] || container.ImagePullPolicy != policy ||
							container.Resources.Requests.Cpu().MilliValue() != cpu {
							return fmt.Errorf("container %q inherited or lost computed values: policy=%s requests=%v; want name=%s policy=%s cpu=%dm",
								container.Name, container.ImagePullPolicy, container.Resources.Requests, names[i], policy, cpu)
						}
					}
					return nil
				}
				checks := []resource.TestCheckFunc{api.check("cronjobs", &uid, 1, 1, validate)}
				for i, name := range names {
					cpu, policy := "200m", "IfNotPresent"
					if name == "first" {
						cpu, policy = "100m", "Always"
					}
					containerPath := fmt.Sprintf("spec.0.job_template.0.spec.0.template.0.spec.0.%s.%d.", field, i)
					checks = append(checks,
						resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", containerPath+"name", name),
						resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", containerPath+"image_pull_policy", policy),
					)
					if name == "added" {
						checks = append(checks, resource.TestCheckNoResourceAttr("kubernetes_cron_job_v1.test", containerPath+"resources.0.requests.cpu"))
					} else {
						checks = append(checks, resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", containerPath+"resources.0.requests.cpu", cpu))
					}
				}
				resource.UnitTest(t, resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					CheckDestroy:             api.checkDestroy,
					Steps: []resource.TestStep{
						{
							Config: before, ConfigPlanChecks: batchCLISettled(),
							Check: api.check("cronjobs", &uid, 1, 0, nil),
						},
						{
							// Simulate different observed admission values while resources
							// and image_pull_policy remain omitted from configuration.
							PreConfig: func() {
								api.mu.Lock()
								defer api.mu.Unlock()
								object := api.crons["batch-local-qa"]
								if object == nil {
									t.Fatal("initial CronJob missing from local API")
								}
								containers := object.Spec.JobTemplate.Spec.Template.Spec.Containers
								if field == "init_container" {
									containers = object.Spec.JobTemplate.Spec.Template.Spec.InitContainers
								}
								for i := range containers {
									cpu := "200m"
									if containers[i].Name == "first" {
										cpu = "100m"
									}
									containers[i].Resources.Requests = core.ResourceList{core.ResourceCPU: apiresource.MustParse(cpu)}
								}
							},
							Config: after,
							ConfigPlanChecks: batchCLISettled(
								plancheck.ExpectResourceAction("kubernetes_cron_job_v1.test", plancheck.ResourceActionUpdate),
							),
							Check: resource.ComposeTestCheckFunc(checks...),
						},
						{
							Config: after, ConfigPlanChecks: batchCLISettled(plancheck.ExpectEmptyPlan()),
							Check: resource.ComposeTestCheckFunc(checks...),
						},
					},
				})
			})
		}
	}
}

func cronJobCLIContainerConfig(api *batchCLIAPI, field string, names []string) string {
	var blocks strings.Builder
	if field == "init_container" {
		blocks.WriteString(`
            container {
              name = "main"
              image = "busybox:1.36"
            }
`)
	}
	for _, name := range names {
		image := "busybox:1.36"
		if name == "first" {
			image = "busybox:latest"
		}
		fmt.Fprintf(&blocks, `
            %s {
              name = %q
              image = %q
            }
`, field, name, image)
	}
	return fmt.Sprintf(`
provider "kubernetes" {
  host = %q
}
resource "kubernetes_cron_job_v1" "test" {
  metadata {
    name = "batch-local-qa"
  }
  spec {
    schedule = "@hourly"
    job_template {
      metadata {}
      spec {
        template {
          metadata {}
          spec {
            restart_policy = "Never"
            %s
          }
        }
      }
    }
  }
}
`, api.server.URL, blocks.String())
}
