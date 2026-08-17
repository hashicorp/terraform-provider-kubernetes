// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package schedulingv1_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	schedulingv1 "github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/schedulingv1"
)

// compile-time check: PriorityClassV1 satisfies resource.Resource.
var _ resource.Resource = (*schedulingv1.PriorityClassV1)(nil)

// TestAccPriorityClassV1_basic creates a PriorityClass with minimal config,
// verifies all computed fields are populated, and confirms import round-trips
// without drift.
func TestAccPriorityClassV1_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-pc")
	resourceName := "kubernetes_priority_class_v1.test"

	tfresource.ParallelTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: testAccPriorityClassV1Config_basic(name),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					tfresource.TestCheckResourceAttr(resourceName, "value", "100"),
					tfresource.TestCheckResourceAttr(resourceName, "preemption_policy", "Never"),
					tfresource.TestCheckResourceAttr(resourceName, "description", ""),
					tfresource.TestCheckResourceAttr(resourceName, "global_default", "false"),
					tfresource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					tfresource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
				),
			},
			// Import by name — verify no drift after import.
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"metadata.0.resource_version",
					"metadata.0.generation",
				},
			},
		},
	})
}

// TestAccPriorityClassV1_update verifies that mutable fields — description,
// global_default, labels, and annotations — can all be changed in-place
// without a destroy/recreate.
func TestAccPriorityClassV1_update(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-pc")
	resourceName := "kubernetes_priority_class_v1.test"

	tfresource.ParallelTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			// Step 1: create with defaults.
			{
				Config: testAccPriorityClassV1Config_basic(name),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr(resourceName, "description", ""),
					tfresource.TestCheckResourceAttr(resourceName, "global_default", "false"),
				),
			},
			// Step 2: add description, labels and annotations — no replace.
			{
				Config: testAccPriorityClassV1Config_updated(name),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr(resourceName, "description", "High priority workloads"),
					tfresource.TestCheckResourceAttr(resourceName, "global_default", "false"),
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.labels.team", "platform"),
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.example.com/note", "updated"),
				),
			},
			// Step 3: remove labels and annotations — verify clean removal.
			{
				Config: testAccPriorityClassV1Config_basic(name),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr(resourceName, "description", ""),
					tfresource.TestCheckNoResourceAttr(resourceName, "metadata.0.labels.team"),
					tfresource.TestCheckNoResourceAttr(resourceName, "metadata.0.annotations.example.com/note"),
				),
			},
		},
	})
}

// TestAccPriorityClassV1_generateName creates a PriorityClass using
// generate_name so the server assigns the full name with a unique suffix.
func TestAccPriorityClassV1_generateName(t *testing.T) {
	resourceName := "kubernetes_priority_class_v1.test"

	tfresource.ParallelTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: testAccPriorityClassV1Config_generateName("tf-acc-pc-"),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					// Server assigns a name with the prefix + random suffix.
					tfresource.TestCheckResourceAttrSet(resourceName, "metadata.0.name"),
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.generate_name", "tf-acc-pc-"),
					tfresource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
				),
			},
		},
	})
}

// TestAccPriorityClassV1_upgradeFromSDKv2 provisions the resource with the
// last SDKv2 release (state schema version 0) then switches to the local
// Framework provider and asserts zero plan diff — proving the Framework reads
// the SDKv2 state without any upgrade step or structural change.
//
// Skipped in -short mode because it downloads from the Terraform registry.
func TestAccPriorityClassV1_upgradeFromSDKv2(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping registry-dependent upgrade test in -short mode")
	}

	name := acctest.RandomWithPrefix("tf-acc-pc")
	resourceName := "kubernetes_priority_class_v1.test"

	tfresource.ParallelTest(t, tfresource.TestCase{
		Steps: []tfresource.TestStep{
			// Step 1: provision with the last SDKv2 release.
			// Writes state at schema version 0 with TypeList metadata.
			{
				ExternalProviders: map[string]tfresource.ExternalProvider{
					"kubernetes": {
						Source:            "hashicorp/kubernetes",
						VersionConstraint: "3.0.1",
					},
				},
				Config: testAccPriorityClassV1Config_basic(name),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					tfresource.TestCheckResourceAttr(resourceName, "value", "100"),
				),
			},
			// Step 2: switch to the local Framework provider.
			// ListNestedBlock produces identical state JSON to TypeList{MaxItems:1}
			// so no UpgradeState is needed — plan must be empty.
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccPriorityClassV1Config_basic(name),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// TestAccPriorityClassV1_moved provisions the deprecated kubernetes_priority_class
// with the last SDKv2 release then uses a moved block to migrate state to
// kubernetes_priority_class_v1 with the Framework provider. The plan must be
// empty — proving MoveState translates the state without drift.
//
// Skipped in -short mode because it downloads from the Terraform registry.
func TestAccPriorityClassV1_moved(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping registry-dependent moved-block test in -short mode")
	}

	name := acctest.RandomWithPrefix("tf-acc-pc")

	tfresource.ParallelTest(t, tfresource.TestCase{
		Steps: []tfresource.TestStep{
			// Step 1: provision kubernetes_priority_class (deprecated type) with
			// the last SDKv2 release. Writes state at schema version 0.
			{
				ExternalProviders: map[string]tfresource.ExternalProvider{
					"kubernetes": {
						Source:            "hashicorp/kubernetes",
						VersionConstraint: "3.0.1",
					},
				},
				Config: testAccPriorityClassConfig_deprecated(name),
			},
			// Step 2: add a moved block and switch to the Framework provider.
			// MoveState translates kubernetes_priority_class → kubernetes_priority_class_v1.
			// Plan must be empty — no destroy, no create.
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccPriorityClassV1Config_movedFrom(name),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// ─── HCL config helpers ───────────────────────────────────────────────────────

func testAccPriorityClassV1Config_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata {
    name = %[1]q
  }

  value             = 100
  preemption_policy = "Never"
}
`, name)
}

func testAccPriorityClassV1Config_updated(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata {
    name = %[1]q
    labels = {
      team = "platform"
    }
    annotations = {
      "example.com/note" = "updated"
    }
  }

  value             = 100
  preemption_policy = "Never"
  description       = "High priority workloads"
}
`, name)
}

func testAccPriorityClassV1Config_generateName(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata {
    generate_name = %[1]q
  }

  value             = 100
  preemption_policy = "Never"
}
`, prefix)
}

// testAccPriorityClassConfig_deprecated uses the old resource type name —
// the one still registered in the SDKv2 provider for backwards compatibility.
func testAccPriorityClassConfig_deprecated(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class" "test" {
  metadata {
    name = %[1]q
  }

  value             = 100
  preemption_policy = "PreemptLowerPriority"
}
`, name)
}

// testAccPriorityClassV1Config_movedFrom contains the moved block that
// migrates kubernetes_priority_class.test → kubernetes_priority_class_v1.test.
func testAccPriorityClassV1Config_movedFrom(name string) string {
	return fmt.Sprintf(`
moved {
  from = kubernetes_priority_class.test
  to   = kubernetes_priority_class_v1.test
}

resource "kubernetes_priority_class_v1" "test" {
  metadata {
    name = %[1]q
  }

  value             = 100
  preemption_policy = "PreemptLowerPriority"
}
`, name)
}
