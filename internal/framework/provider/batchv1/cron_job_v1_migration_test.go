// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1_test

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	batch "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccCronJobV1UpgradeFromSDKv2(t *testing.T) {
	for _, test := range []struct {
		name   string
		config func(string, string) string
	}{
		{"minimal", testAccKubernetesCronJobV1ConfigMinimal},
		{"configured", testAccKubernetesCronJobV1Config_basic},
	} {
		t.Run(test.name, func(t *testing.T) {
			name := "tf-acc-cron-upgrade-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
			config := test.config(name, busyboxImage)
			var before, after batch.CronJob
			resource.ParallelTest(t, resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(t)
					skipIfClusterVersionLessThan(t, "1.25.0")
				},
				CheckDestroy: testAccCheckKubernetesCronJobV1Destroy,
				Steps: []resource.TestStep{
					{
						ExternalProviders: map[string]resource.ExternalProvider{
							"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.2.1"},
						},
						Config: config,
						Check:  testAccCheckKubernetesCronJobV1Exists("kubernetes_cron_job_v1.test", &before),
					},
					{
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Config:                   config,
						// The SDK stored omitted optional scalars as zero/empty.
						// Framework normalizes them to null in a state-only update.
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{
								plancheck.ExpectResourceAction("kubernetes_cron_job_v1.test", plancheck.ResourceActionUpdate),
							},
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							testAccCheckKubernetesCronJobV1Exists("kubernetes_cron_job_v1.test", &after),
							testAccCheckKubernetesCronJobV1ForceNew(&before, &after, false),
							func(_ *terraform.State) error {
								if !reflect.DeepEqual(before.Spec, after.Spec) {
									return fmt.Errorf("migration changed the CronJob API spec:\nbefore: %#v\nafter: %#v", before.Spec, after.Spec)
								}
								return nil
							},
						),
					},
					{
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Config:                   config,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
					},
					{
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						ResourceName:             "kubernetes_cron_job_v1.test",
						ImportState:              true,
						ImportStateVerify:        true,
					},
				},
			})
		})
	}
}

func TestAccCronJobV1Disappears(t *testing.T) {
	var object batch.CronJob
	name := "tf-acc-cron-gone-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesCronJobV1Destroy,
		Steps: []resource.TestStep{{
			Config: testAccKubernetesCronJobV1ConfigMinimal(name, busyboxImage),
			Check: resource.ComposeAggregateTestCheckFunc(
				testAccCheckKubernetesCronJobV1Exists("kubernetes_cron_job_v1.test", &object),
				func(_ *terraform.State) error {
					client, err := testAccProvider.Meta().(kubernetes.KubeClientsets).MainClientset()
					if err != nil {
						return err
					}
					return client.BatchV1().CronJobs(object.Namespace).Delete(context.Background(), object.Name, metav1.DeleteOptions{})
				},
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestAccCronJobV1DefaultsAfterRemoval(t *testing.T) {
	name := "tf-acc-cron-default-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	minimal := testAccKubernetesCronJobV1ConfigMinimal(name, busyboxImage)
	configured := strings.Replace(minimal, `schedule = "*/1 * * * *"`, `schedule = "*/1 * * * *"
    failed_jobs_history_limit = 0
    successful_jobs_history_limit = 0
    suspend = true
    timezone = "Etc/UTC"
    starting_deadline_seconds = 20`, 1)
	if configured == minimal {
		t.Fatal("default-removal test did not configure overrides")
	}
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.25.0")
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesCronJobV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: configured,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.failed_jobs_history_limit", "0"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.successful_jobs_history_limit", "0"),
				),
			},
			{
				Config: minimal,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.failed_jobs_history_limit", "1"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.successful_jobs_history_limit", "3"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.timezone", ""),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.suspend", "false"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.starting_deadline_seconds", "0"),
				),
			},
			{
				Config: minimal,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}
