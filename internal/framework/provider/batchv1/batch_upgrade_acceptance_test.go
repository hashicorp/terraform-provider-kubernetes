// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1_test

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	batchv1 "k8s.io/api/batch/v1"
)

// Explicit empty values, "" for API-defaulted strings, a template label the
// Job controller also sets, and respelled quantities, all written by 3.3.0,
// must not replace anything after the upgrade.
func TestAccKubernetesJobV1_upgradeExplicitEmpty(t *testing.T) {
	name := "tf-acc-job-empty-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	const address = "kubernetes_job_v1.test"
	var before, after batchv1.Job
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckKubernetesJobV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.3.0"},
				},
				Config: testAccBatchExplicitEmptyJob(name, sdkResources("0.5", "64Mi")),
				Check:  testAccCheckKubernetesJobV1Exists(address, &before),
				// The SDKv2 provider never settles on these values.
				ExpectNonEmptyPlan: true,
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccBatchExplicitEmptyJob(name, frameworkResources("0.5", "64Mi")),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{expectNoReplacement(address)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesJobV1Exists(address, &after),
					testAccCheckKubernetesJobV1ForceNew(&before, &after, false),
				),
			},
			{
				// The template is immutable: a real change replaces the Job.
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccBatchExplicitEmptyJob(name, frameworkResources("0.5", "128Mi")),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionDestroyBeforeCreate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

// 3.3.0 sent an explicit pod-level runAsNonRoot false. The template is
// immutable, so removing the block replaces the Job.
func TestAccKubernetesJobV1_upgradeRemoveZeroSecurityContext(t *testing.T) {
	name := "tf-acc-job-sc-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	const address = "kubernetes_job_v1.test"
	const securityContext = `security_context {
          run_as_non_root = false
        }`
	var created, upgraded, replaced batchv1.Job
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckKubernetesJobV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.3.0"},
				},
				Config: testAccBatchSecurityContextJob(name, securityContext),
				Check:  testAccCheckKubernetesJobV1Exists(address, &created),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccBatchSecurityContextJob(name, securityContext),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesJobV1Exists(address, &upgraded),
					testAccCheckKubernetesJobV1ForceNew(&created, &upgraded, false),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccBatchSecurityContextJob(name, ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionDestroyBeforeCreate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesJobV1Exists(address, &replaced),
					testAccCheckKubernetesJobV1ForceNew(&upgraded, &replaced, true),
				),
			},
		},
	})
}

// 3.3.0 state cannot tell a configured template label from one admission
// added, so until an apply records the configured labels, a label removed from
// the configuration is dropped from state only. Afterwards removing one
// replaces the Job.
func TestAccKubernetesJobV1_upgradeRemoveTemplateLabel(t *testing.T) {
	for _, tc := range []struct {
		name, created, upgraded     string
		upgradeAction, removeAction plancheck.ResourceActionType
	}{
		{"before an apply", `app = "a", extra = "b"`, `app = "a", extra = "b"`, plancheck.ResourceActionNoop, plancheck.ResourceActionUpdate},
		{"after an apply", `app = "a", extra = "b", x = "c"`, `app = "a", x = "c"`, plancheck.ResourceActionUpdate, plancheck.ResourceActionDestroyBeforeCreate},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name := "tf-acc-job-label-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
			const address = "kubernetes_job_v1.test"
			labels := func(labels string) string {
				return strings.Replace(testAccBatchSecurityContextJob(name, ""), "metadata {}", "metadata {\n        labels = {"+labels+"}\n      }", 1)
			}
			var created, upgraded, removed batchv1.Job
			replaced := tc.removeAction == plancheck.ResourceActionDestroyBeforeCreate
			resource.ParallelTest(t, resource.TestCase{
				PreCheck:     func() { testAccPreCheck(t) },
				CheckDestroy: testAccCheckKubernetesJobV1Destroy,
				Steps: []resource.TestStep{
					{
						ExternalProviders: map[string]resource.ExternalProvider{
							"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.3.0"},
						},
						Config: labels(tc.created),
						Check:  testAccCheckKubernetesJobV1Exists(address, &created),
					},
					{
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Config:                   labels(tc.upgraded),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, tc.upgradeAction)},
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							testAccCheckKubernetesJobV1Exists(address, &upgraded),
							testAccCheckKubernetesJobV1ForceNew(&created, &upgraded, false),
						),
					},
					{
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Config:                   labels(`app = "a"`),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, tc.removeAction)},
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							testAccCheckKubernetesJobV1Exists(address, &removed),
							testAccCheckKubernetesJobV1ForceNew(&upgraded, &removed, replaced),
							func(*terraform.State) error {
								for _, key := range []string{"extra", "x"} {
									if _, ok := removed.Spec.Template.Labels[key]; ok && replaced {
										return fmt.Errorf("live pod template labels = %v, want no %s", removed.Spec.Template.Labels, key)
									}
								}
								return nil
							},
						),
					},
				},
			})
		})
	}
}

func TestAccKubernetesCronJobV1_upgradeExplicitEmpty(t *testing.T) {
	name := "tf-acc-cron-empty-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	const address = "kubernetes_cron_job_v1.test"
	var before, after batchv1.CronJob
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckKubernetesCronJobV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.3.0"},
				},
				Config: testAccBatchExplicitEmptyCronJob(name, sdkResources("0.5", "64Mi"), "busybox:1.36"),
				Check:  testAccCheckKubernetesCronJobV1Exists(address, &before),
				// The SDKv2 provider never settles on these values.
				ExpectNonEmptyPlan: true,
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccBatchExplicitEmptyCronJob(name, frameworkResources("0.5", "64Mi"), "busybox:1.36"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{expectNoReplacement(address)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesCronJobV1Exists(address, &after),
					testAccCheckKubernetesCronJobV1ForceNew(&before, &after, false),
					func(*terraform.State) error {
						// 3.3.0 sent no pod security context for this block, and
						// the restricted Pod Security Standard rejects runAsNonRoot false.
						if sc := after.Spec.JobTemplate.Spec.Template.Spec.SecurityContext; sc != nil && sc.RunAsNonRoot != nil {
							return fmt.Errorf("live pod securityContext = %+v, want no runAsNonRoot", sc)
						}
						return nil
					},
				),
			},
			{
				// One quantity changes while the other keeps its respelled value.
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccBatchExplicitEmptyCronJob(name, frameworkResources("0.5", "128Mi"), "busybox:1.37"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesCronJobV1Exists(address, &after),
					testAccCheckKubernetesCronJobV1ForceNew(&before, &after, false),
				),
			},
		},
	})
}

// State written by 3.3.0 plans no change before its first refresh, while a
// CronJob schedule edit is still an in-place update.
func TestAccKubernetesBatchV1_upgradeWithoutRefresh(t *testing.T) {
	name := "tf-acc-batch-norefresh-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	config := func(schedule string) string {
		return testAccBatchSecurityContextJob(name, "") + fmt.Sprintf(`
resource "kubernetes_cron_job_v1" "test" {
  metadata {
    name = %q
  }
  spec {
    schedule = %q
    job_template {
      metadata {}
      spec {
        template {
          metadata {}
          spec {
            restart_policy = "Never"
            container {
              name    = "main"
              image   = %q
              command = ["sh", "-c", "true"]
            }
          }
        }
      }
    }
  }
}
`, name, schedule, busyboxImage)
	}
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:             func() { testAccPreCheck(t) },
		AdditionalCLIOptions: &resource.AdditionalCLIOptions{Plan: resource.PlanOptions{NoRefresh: true}},
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccCheckKubernetesJobV1Destroy,
			testAccCheckKubernetesCronJobV1Destroy,
		),
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.3.0"},
				},
				Config: config("0 0 1 1 *"),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config("0 0 1 1 *"),
				ConfigPlanChecks:         resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config("0 0 2 1 *"),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction("kubernetes_cron_job_v1.test", plancheck.ResourceActionUpdate),
				}},
			},
		},
	})
}

// A generate_name Job that 3.3.0 created but saw fail is tainted without a
// name; the upgrade takes the name from the id and replaces it.
func TestAccKubernetesJobV1_upgradeTaintedGeneratedName(t *testing.T) {
	prefix := "tf-acc-test-" + acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum) + "-"
	config := func(command string) string {
		return fmt.Sprintf(`resource "kubernetes_job_v1" "test" {
  metadata {
    generate_name = %q
  }
  spec {
    backoff_limit = 0
    template {
      metadata {}
      spec {
        restart_policy = "Never"
        container {
          name    = "main"
          image   = %q
          command = ["sh", "-c", %q]
        }
      }
    }
  }
  wait_for_completion = true
}
`, prefix, busyboxImage, command)
	}
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckKubernetesJobV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.3.0"},
				},
				Config:      config("exit 1"),
				ExpectError: regexp.MustCompile("is in failed state"),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config("true"),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction("kubernetes_job_v1.test", plancheck.ResourceActionReplace),
				}},
			},
		},
	})
}

type noReplacement struct{ address string }

func expectNoReplacement(address string) plancheck.PlanCheck { return noReplacement{address} }

func (c noReplacement) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	for _, change := range req.Plan.ResourceChanges {
		if change.Address == c.address && change.Change.Actions.Replace() {
			resp.Error = fmt.Errorf("%s is planned for replacement", c.address)
		}
	}
}

// The SDKv2 provider spells container resources as a block, v4 as a list.
func sdkResources(cpu, memory string) string {
	return fmt.Sprintf(`resources {
              limits = { cpu = %q, memory = %q }
            }`, cpu, memory)
}

func frameworkResources(cpu, memory string) string {
	return fmt.Sprintf(`resources = [{ limits = { cpu = %q, memory = %q } }]`, cpu, memory)
}

const testAccBatchExplicitEmptyPodSpec = `
          restart_policy                   = "Never"
          termination_grace_period_seconds = 1
          scheduler_name                   = ""
          node_selector                    = {}
          security_context {
            supplemental_groups = []
          }
          topology_spread_constraint {
            max_skew           = 1
            topology_key       = "kubernetes.io/hostname"
            when_unsatisfiable = "ScheduleAnyway"
            label_selector {
              match_labels = {}
            }
          }
          container {
            name              = "main"
            image             = "%s"
            command           = ["sh", "-c", "true"]
            image_pull_policy = ""
            %s
          }`

func testAccBatchExplicitEmptyJob(name, resources string) string {
	return fmt.Sprintf(`resource "kubernetes_job_v1" "test" {
  metadata {
    name = %q
  }
  spec {
    template {
      metadata {
        labels = {
          app      = "explicit-empty"
          job-name = %q
        }
        annotations = {}
      }
      spec {`+testAccBatchExplicitEmptyPodSpec+`
      }
    }
  }
  wait_for_completion = false
}
`, name, name, busyboxImage, resources)
}

func testAccBatchExplicitEmptyCronJob(name, resources, image string) string {
	return fmt.Sprintf(`resource "kubernetes_cron_job_v1" "test" {
  metadata {
    name = %q
  }
  spec {
    schedule = "0 0 1 1 *"
    suspend  = true
    job_template {
      metadata {
        annotations = {}
      }
      spec {
        template {
          metadata {
            labels      = {}
            annotations = {}
          }
          spec {`+testAccBatchExplicitEmptyPodSpec+`
          }
        }
      }
    }
  }
}
`, name, image, resources)
}

func testAccBatchSecurityContextJob(name, securityContext string) string {
	return fmt.Sprintf(`resource "kubernetes_job_v1" "test" {
  metadata {
    name = %q
  }
  spec {
    template {
      metadata {}
      spec {
        restart_policy                   = "Never"
        termination_grace_period_seconds = 1
        %s
        container {
          name    = "main"
          image   = %q
          command = ["sh", "-c", "true"]
        }
      }
    }
  }
  wait_for_completion = false
}
`, name, securityContext, busyboxImage)
}
