// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	batchapi "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestAccBatchV1RetryDefaultsUpgrade(t *testing.T) {
	for _, kind := range []struct {
		name         string
		address      string
		specPath     string
		checkDestroy resource.TestCheckFunc
		config       func(string, string) string
	}{
		{
			name:         "Job",
			address:      "kubernetes_job_v1.test",
			specPath:     "spec.0.",
			checkDestroy: testAccCheckKubernetesJobV1Destroy,
			config:       testAccJobV1RetryDefaultsConfig,
		},
		{
			name:         "CronJob",
			address:      "kubernetes_cron_job_v1.test",
			specPath:     "spec.0.job_template.0.spec.0.",
			checkDestroy: testAccCheckKubernetesCronJobV1Destroy,
			config:       testAccCronJobV1RetryDefaultsConfig,
		},
	} {
		for _, variant := range []struct {
			name    string
			fields  string
			backoff int32
		}{
			{name: "both_omitted"},
			{name: "max_failed_omitted", fields: "backoff_limit_per_index = 2", backoff: 2},
		} {
			t.Run(kind.name+"/"+variant.name, func(t *testing.T) {
				name := "tf-acc-retry-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
				config := kind.config(name, variant.fields)
				zeroConfig := kind.config(name, fmt.Sprintf("backoff_limit_per_index = %d\nmax_failed_indexes = 0", variant.backoff))
				nonzeroConfig := kind.config(name, variant.fields+"\nmax_failed_indexes = 1")
				var before testAccBatchRetrySnapshot
				check := func(capture, unchanged bool, maxFailed int32) resource.TestCheckFunc {
					return resource.ComposeAggregateTestCheckFunc(
						testAccCheckBatchRetryDefaults(t, kind.address, &before, capture, unchanged, variant.backoff, maxFailed),
						resource.TestCheckResourceAttr(kind.address, kind.specPath+"backoff_limit_per_index", fmt.Sprint(variant.backoff)),
						resource.TestCheckResourceAttr(kind.address, kind.specPath+"max_failed_indexes", fmt.Sprint(maxFailed)),
					)
				}
				settled := func(preApply plancheck.PlanCheck) resource.ConfigPlanChecks {
					return resource.ConfigPlanChecks{
						PreApply:             []plancheck.PlanCheck{preApply},
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					}
				}
				resource.ParallelTest(t, resource.TestCase{
					PreCheck: func() {
						testAccPreCheck(t)
						skipIfClusterVersionLessThan(t, "1.33.0")
					},
					CheckDestroy: kind.checkDestroy,
					Steps: []resource.TestStep{
						{
							ExternalProviders: map[string]resource.ExternalProvider{
								"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.2.1"},
							},
							Config: config,
							Check:  check(true, false, 0),
						},
						{
							ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
							Config:                   config,
							// Other omitted SDK scalars normalize in a one-time state-only
							// update; neither retry defaults nor the full API spec may change.
							ConfigPlanChecks: settled(plancheck.ExpectResourceAction(kind.address, plancheck.ResourceActionUpdate)),
							Check:            check(false, true, 0),
						},
						{
							ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
							Config:                   config,
							ConfigPlanChecks:         settled(plancheck.ExpectEmptyPlan()),
							Check:                    check(false, true, 0),
						},
						{
							ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
							Config:                   zeroConfig,
							ConfigPlanChecks:         settled(plancheck.ExpectEmptyPlan()),
							Check:                    check(false, true, 0),
						},
						{
							ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
							Config:                   config,
							ConfigPlanChecks:         settled(plancheck.ExpectEmptyPlan()),
							Check:                    check(false, true, 0),
						},
						{
							ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
							Config:                   nonzeroConfig,
							ConfigPlanChecks:         settled(plancheck.ExpectResourceAction(kind.address, plancheck.ResourceActionUpdate)),
							Check:                    check(false, false, 1),
						},
						{
							ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
							Config:                   config,
							ConfigPlanChecks:         settled(plancheck.ExpectResourceAction(kind.address, plancheck.ResourceActionUpdate)),
							Check:                    check(false, true, 0),
						},
						{
							ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
							Config:                   config,
							ConfigPlanChecks:         settled(plancheck.ExpectEmptyPlan()),
							Check:                    check(false, true, 0),
						},
					},
				})
			})
		}
	}
}

type testAccBatchRetrySnapshot struct {
	uid  types.UID
	spec any
}

func testAccCheckBatchRetryDefaults(t *testing.T, address string, before *testAccBatchRetrySnapshot, capture, unchanged bool, backoff, maxFailed int32) resource.TestCheckFunc {
	t.Helper()
	return func(state *terraform.State) error {
		var snapshot testAccBatchRetrySnapshot
		var spec batchapi.JobSpec
		switch address {
		case "kubernetes_job_v1.test":
			var job batchapi.Job
			if err := testAccCheckKubernetesJobV1Exists(address, &job)(state); err != nil {
				return err
			}
			snapshot = testAccBatchRetrySnapshot{uid: job.UID, spec: job.Spec}
			spec = job.Spec
		case "kubernetes_cron_job_v1.test":
			var cronJob batchapi.CronJob
			if err := testAccCheckKubernetesCronJobV1Exists(address, &cronJob)(state); err != nil {
				return err
			}
			snapshot = testAccBatchRetrySnapshot{uid: cronJob.UID, spec: cronJob.Spec}
			spec = cronJob.Spec.JobTemplate.Spec
		default:
			return fmt.Errorf("unsupported retry-default test address %q", address)
		}
		if spec.CompletionMode == nil || *spec.CompletionMode != batchapi.IndexedCompletion {
			return fmt.Errorf("%s lost Indexed completion mode", address)
		}
		for _, field := range []struct {
			name string
			got  *int32
			want int32
		}{
			{"backoffLimitPerIndex", spec.BackoffLimitPerIndex, backoff},
			{"maxFailedIndexes", spec.MaxFailedIndexes, maxFailed},
		} {
			if field.got == nil {
				return fmt.Errorf("%s API spec.%s is absent, want pointer to %d", address, field.name, field.want)
			}
			if *field.got != field.want {
				return fmt.Errorf("%s API spec.%s = %d, want %d", address, field.name, *field.got, field.want)
			}
		}
		if snapshot.uid == "" {
			return fmt.Errorf("%s has an empty API UID", address)
		}
		if capture {
			*before = snapshot
		} else {
			if snapshot.uid != before.uid {
				return fmt.Errorf("%s UID changed: before %q, after %q", address, before.uid, snapshot.uid)
			}
			if unchanged && !reflect.DeepEqual(before.spec, snapshot.spec) {
				return fmt.Errorf("%s API spec changed:\nbefore: %#v\nafter: %#v", address, before.spec, snapshot.spec)
			}
		}
		t.Logf("%s: UID=%s, API retry pointers=%d/%d, captured=%t, full spec unchanged=%t",
			address, snapshot.uid, backoff, maxFailed, capture, unchanged)
		return nil
	}
}

func testAccJobV1RetryDefaultsConfig(name, retryFields string) string {
	config := fmt.Sprintf(`resource "kubernetes_job_v1" "test" {
  metadata {
    name      = %q
    namespace = "default"
  }
  spec {
    completion_mode = "Indexed"
    completions     = 2
    parallelism     = 1
    template {
      metadata {}
      spec {
        restart_policy = "Never"
        container {
          name    = "test"
          image   = %q
          command = ["sh", "-c", "true"]
        }
      }
    }
  }
  wait_for_completion = false
}
`, name, busyboxImage)
	return strings.Replace(config, `completion_mode = "Indexed"`, `completion_mode = "Indexed"`+"\n"+retryFields, 1)
}

func testAccCronJobV1RetryDefaultsConfig(name, retryFields string) string {
	config := fmt.Sprintf(`resource "kubernetes_cron_job_v1" "test" {
  metadata {
    name      = %q
    namespace = "default"
  }
  spec {
    schedule = "0 0 * * *"
    suspend  = true
    job_template {
      metadata {}
      spec {
        completion_mode = "Indexed"
        completions     = 2
        parallelism     = 1
        template {
          metadata {}
          spec {
            restart_policy = "Never"
            container {
              name    = "test"
              image   = %q
              command = ["sh", "-c", "true"]
            }
          }
        }
      }
    }
  }
}
`, name, busyboxImage)
	return strings.Replace(config, `completion_mode = "Indexed"`, `completion_mode = "Indexed"`+"\n"+retryFields, 1)
}
