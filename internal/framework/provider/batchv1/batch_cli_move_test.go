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
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestBatchCLI_JobAliasMove(t *testing.T) {
	batchCLIEnvironment(t)
	api := newBatchCLIAPI(t)
	current := batchCLIConfig(api, "jobs", batchCLIOptions{})
	legacy := strings.Replace(current, `"kubernetes_job_v1"`, `"kubernetes_job"`, 1)
	moved := current + `
moved {
  from = kubernetes_job.test
  to = kubernetes_job_v1.test
}
`
	var original *batchapi.Job
	var uid string
	unchanged := func(object runtime.Object) error {
		job, ok := object.(*batchapi.Job)
		if !ok || original == nil || !apiequality.Semantic.DeepEqual(original.Spec, job.Spec) {
			return fmt.Errorf("alias move changed the Job's desired specification")
		}
		return nil
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             api.checkDestroy,
		Steps: []resource.TestStep{
			{
				Config: legacy,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_job.test", "metadata.0.name", "batch-local-qa"),
					func(_ *terraform.State) error {
						api.mu.Lock()
						defer api.mu.Unlock()
						job := api.jobs["batch-local-qa"]
						if job == nil || job.UID == "" {
							return fmt.Errorf("legacy Job does not exist in the local API")
						}
						original, uid = job.DeepCopy(), string(job.UID)
						return nil
					},
				),
			},
			{
				Config: moved,
				ConfigPlanChecks: batchCLISettled(
					plancheck.ExpectResourceAction("kubernetes_job_v1.test", plancheck.ResourceActionUpdate),
				),
				Check: api.check("jobs", &uid, 1, 0, unchanged),
			},
			{
				Config: moved, ConfigPlanChecks: batchCLISettled(plancheck.ExpectEmptyPlan()),
				Check: api.check("jobs", &uid, 1, 0, unchanged),
			},
		},
	})
}
