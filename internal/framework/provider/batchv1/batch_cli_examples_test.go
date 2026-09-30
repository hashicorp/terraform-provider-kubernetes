// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestBatchCLI_DocumentedExamples(t *testing.T) {
	batchCLIEnvironment(t)
	for _, example := range []struct {
		path    string
		address string
	}{
		{path: "job_v1/example_1.tf", address: "kubernetes_job_v1.demo"},
		{path: "job_v1/example_2.tf", address: "kubernetes_job_v1.demo"},
		{path: "cron_job_v1/example_1.tf", address: "kubernetes_cron_job_v1.demo"},
	} {
		t.Run(example.path, func(t *testing.T) {
			api := newBatchCLIAPI(t)
			source, err := os.ReadFile(filepath.Join("../../../../examples/resources", example.path))
			if err != nil {
				t.Fatal(err)
			}
			// Exercise the checked-in documentation unchanged, including its
			// ordinary blocks, implicit image policy, and numeric TTL coercion.
			config := fmt.Sprintf("provider \"kubernetes\" {\n host = %q\n}\n%s", api.server.URL, source)
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				CheckDestroy:             api.checkDestroy,
				Steps: []resource.TestStep{{
					Config: config, PlanOnly: true, ExpectNonEmptyPlan: true,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PostApplyPostRefresh: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(example.address, plancheck.ResourceActionCreate),
						},
					},
				}},
			})
			api.mu.Lock()
			defer api.mu.Unlock()
			if len(api.calls) != 0 {
				t.Errorf("plan-only documentation validation unexpectedly reached the API: %v", api.calls)
			}
		})
	}
}
