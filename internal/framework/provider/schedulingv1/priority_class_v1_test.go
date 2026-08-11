// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package schedulingv1_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPriorityClassV1_basic(t *testing.T) {
	name := "tf-acc-test-pc-basic"
	resourceName := "kubernetes_priority_class_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPriorityClassV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.name", name),
					resource.TestCheckResourceAttr(resourceName, "value", "100"),
					resource.TestCheckResourceAttr(resourceName, "preemption_policy", "Never"),
					resource.TestCheckResourceAttr(resourceName, "description", ""),
					resource.TestCheckResourceAttr(resourceName, "global_default", "false"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.uid"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.resource_version"),
				),
			},
			// Import by name
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"metadata.resource_version",
					"metadata.generation",
				},
			},
		},
	})
}

func TestAccPriorityClassV1_update(t *testing.T) {
	name := "tf-acc-test-pc-update"
	resourceName := "kubernetes_priority_class_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPriorityClassV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", ""),
					resource.TestCheckResourceAttr(resourceName, "global_default", "false"),
				),
			},
			{
				Config: testAccPriorityClassV1Config_updated(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", "High priority workloads"),
					resource.TestCheckResourceAttr(resourceName, "global_default", "false"),
					resource.TestCheckResourceAttr(resourceName, "metadata.labels.team", "platform"),
				),
			},
		},
	})
}

func TestAccPriorityClassV1_upgradeFromSDKv2(t *testing.T) {
	// This test downloads provider v3.0.1 from the Terraform registry.
	// Skip it in environments without public registry access.
	if testing.Short() {
		t.Skip("skipping registry-dependent upgrade test in -short mode")
	}

	name := "tf-acc-test-pc-upgrade"
	resourceName := "kubernetes_priority_class_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			// Step 1: provision with the last SDK v2 release (v3.0.1).
			// The provider will write schema-version 0 (TypeList metadata) state.
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						Source:            "hashicorp/kubernetes",
						VersionConstraint: "3.0.1",
					},
				},
				Config: testAccPriorityClassV1Config_sdkv2(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "value", "100"),
				),
			},
			// Step 2: apply the local (framework) provider — state upgrader converts v0 → v1.
			// ExpectEmptyPlan asserts no diff after upgrade, proving the upgrader is correct.
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccPriorityClassV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.name", name),
					resource.TestCheckResourceAttr(resourceName, "value", "100"),
				),
			},
		},
	})
}

// testAccPriorityClassV1Config_sdkv2 uses block-style metadata required by the
// SDK v2 provider (v3.0.1 and earlier). Used only in the upgrade test Step 1.
func testAccPriorityClassV1Config_sdkv2(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata {
    name = %q
  }

  value             = 100
  preemption_policy = "Never"
}
`, name)
}

func testAccPriorityClassV1Config_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata = {
    name = %q
  }

  value             = 100
  preemption_policy = "Never"
}
`, name)
}

func testAccPriorityClassV1Config_updated(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata = {
    name = %q
    labels = {
      team = "platform"
    }
  }

  value             = 100
  preemption_policy = "Never"
  description       = "High priority workloads"
}
`, name)
}
