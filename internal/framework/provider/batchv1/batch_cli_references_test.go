// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestBatchCLI_DependentReferences(t *testing.T) {
	batchCLIEnvironment(t)
	for _, kind := range []string{"jobs", "cronjobs"} {
		t.Run(kind, func(t *testing.T) {
			api := newBatchCLIAPI(t)
			address := batchCLIAddress(kind)
			spec := address + ".spec[0]"
			if kind == "cronjobs" {
				spec += ".job_template[0].spec[0]"
			}
			dependent := fmt.Sprintf(`
resource "terraform_data" "dependent" {
  input = {
    name = %[1]s.metadata[0].name
    cpu = %[2]s.template[0].spec[0].container[0].resources[0].limits["cpu"]
    init_cpu = %[2]s.template[0].spec[0].init_container[0].resources[0].requests["cpu"]
  }
}
output "batch_name" { value = terraform_data.dependent.output.name }
output "batch_cpu" { value = terraform_data.dependent.output.cpu }
output "batch_init_cpu" { value = terraform_data.dependent.output.init_cpu }
`, address, spec)
			options := batchCLIOptions{rich: true}
			initial := batchCLIConfig(api, kind, options) + dependent
			updated := batchCLIConfig(api, kind, batchCLIUpdated(kind, options)) + dependent
			var uid string
			outputs := resource.ComposeTestCheckFunc(
				resource.TestCheckOutput("batch_name", "batch-local-qa"),
				resource.TestCheckOutput("batch_cpu", "100m"),
				resource.TestCheckOutput("batch_init_cpu", "100m"),
				resource.TestCheckResourceAttr("terraform_data.dependent", "output.name", "batch-local-qa"),
			)
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				CheckDestroy:             api.checkDestroy,
				Steps: []resource.TestStep{
					{
						Config: initial,
						Check: resource.ComposeTestCheckFunc(
							api.check(kind, &uid, 1, 0, batchCLIValidateRich), outputs,
						),
					},
					{
						Config: updated,
						ConfigPlanChecks: batchCLISettled(
							plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate),
							plancheck.ExpectResourceAction("terraform_data.dependent", plancheck.ResourceActionNoop),
						),
						Check: resource.ComposeTestCheckFunc(
							api.check(kind, &uid, 1, 1, batchCLIValidateUpdated), outputs,
						),
					},
					{
						Config: updated, ConfigPlanChecks: batchCLISettled(plancheck.ExpectEmptyPlan()),
						Check: resource.ComposeTestCheckFunc(api.check(kind, &uid, 1, 1, nil), outputs),
					},
				},
			})
		})
	}
}
