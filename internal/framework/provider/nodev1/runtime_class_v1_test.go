// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package nodev1_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRuntimeClassV1_basic(t *testing.T) {
	name := "tf-acc-test-basic"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRuntimeClassV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"kubernetes_runtime_class_v1.test",
						"metadata.0.name",
						name,
					),
					resource.TestCheckResourceAttr(
						"kubernetes_runtime_class_v1.test",
						"handler",
						"runc",
					),
					resource.TestCheckResourceAttrSet(
						"kubernetes_runtime_class_v1.test",
						"metadata.0.uid",
					),
					resource.TestCheckResourceAttrSet(
						"kubernetes_runtime_class_v1.test",
						"metadata.0.resource_version",
					),
				),
			},
			{
				ResourceName:      "kubernetes_runtime_class_v1.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"metadata.0.resource_version",
					"timeouts",
				},
			},
		},
	})
}

func TestAccRuntimeClassV1_labels(t *testing.T) {
	name := "tf-acc-test-labels"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRuntimeClassV1Config_withLabel(name, "env", "staging"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"kubernetes_runtime_class_v1.test",
						"metadata.0.name",
						name,
					),
					resource.TestCheckResourceAttr(
						"kubernetes_runtime_class_v1.test",
						"metadata.0.labels.env",
						"staging",
					),
				),
			},
			{
				Config: testAccRuntimeClassV1Config_withLabel(name, "env", "production"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"kubernetes_runtime_class_v1.test",
						"metadata.0.labels.env",
						"production",
					),
				),
			},
			// Re-apply same config — plan must be empty (proves isInternalKey works).
			{
				Config:             testAccRuntimeClassV1Config_withLabel(name, "env", "production"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// ── Terraform config helpers ──────────────────────────────────────────────────

func testAccRuntimeClassV1Config_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_runtime_class_v1" "test" {
  metadata {
    name = %q
  }
  handler = "runc"
}
`, name)
}

func testAccRuntimeClassV1Config_withLabel(name, labelKey, labelValue string) string {
	return fmt.Sprintf(`
resource "kubernetes_runtime_class_v1" "test" {
  metadata {
    name = %q
    labels = {
      %q = %q
    }
  }
  handler = "runc"
}
`, name, labelKey, labelValue)
}
