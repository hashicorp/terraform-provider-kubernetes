// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package schedulingv1_test

import (
	"fmt"
	"os/exec"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

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

// TestAccPriorityClassV1_identity verifies that resource identity is populated
// correctly and import using an import block with resource identity succeeds (TF >= 1.12).
func TestAccPriorityClassV1_identity(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-pc")
	resourceName := "kubernetes_priority_class_v1.test"

	tfresource.ParallelTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		Steps: []tfresource.TestStep{
			{
				Config: testAccPriorityClassV1Config_basic(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(resourceName, map[string]knownvalue.Check{
						"name":        knownvalue.StringExact(name),
						"api_version": knownvalue.StringExact("scheduling.k8s.io/v1"),
						"kind":        knownvalue.StringExact("PriorityClass"),
					}),
				},
			},
			{
				ResourceName:    resourceName,
				ImportState:     true,
				ImportStateKind: tfresource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

// TestAccPriorityClassV1_update verifies that mutable fields — description,
// global_default, labels, and annotations — can all be changed in-place
// without a destroy/recreate. ImportState and ImportStateVerify are run after
// each update to confirm the changes are reflected on the remote Kubernetes object.
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
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "metadata.0.generation"},
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
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "metadata.0.generation"},
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
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "metadata.0.generation"},
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

// TestAccPriorityClassV1_valueRequiresReplace verifies that changing the
// immutable 'value' field destroys the old resource and creates a new one
// (RequiresReplace plan modifier).
func TestAccPriorityClassV1_valueRequiresReplace(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-pc")
	resourceName := "kubernetes_priority_class_v1.test"

	tfresource.ParallelTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: testAccPriorityClassV1Config_basic(name),
				Check:  tfresource.TestCheckResourceAttr(resourceName, "value", "100"),
			},
			// Changing value must trigger destroy + create, not an in-place update.
			{
				Config: testAccPriorityClassV1Config_withValue(name, 200),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: tfresource.TestCheckResourceAttr(resourceName, "value", "200"),
			},
		},
	})
}

// TestAccPriorityClassV1_preemptionPolicyRequiresReplace verifies that changing
// preemption_policy also triggers a destroy + create.
func TestAccPriorityClassV1_preemptionPolicyRequiresReplace(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-pc")
	resourceName := "kubernetes_priority_class_v1.test"

	tfresource.ParallelTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: testAccPriorityClassV1Config_basic(name), // preemption_policy = "Never"
				Check:  tfresource.TestCheckResourceAttr(resourceName, "preemption_policy", "Never"),
			},
			{
				Config: testAccPriorityClassV1Config_withPreemptionPolicy(name, "PreemptLowerPriority"),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: tfresource.TestCheckResourceAttr(resourceName, "preemption_policy", "PreemptLowerPriority"),
			},
		},
	})
}

// TestAccPriorityClassV1_globalDefault verifies that global_default can be
// toggled true/false in-place without a destroy/recreate.
//
// NOTE: this test is intentionally non-parallel. Setting global_default = true
// changes the default priority for ALL pods on the cluster for the duration of
// the test, which can corrupt scheduling behaviour in concurrently running tests.
// The original SDKv2 test was also serial for the same reason.
func TestAccPriorityClassV1_globalDefault(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-pc")
	resourceName := "kubernetes_priority_class_v1.test"

	tfresource.Test(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: testAccPriorityClassV1Config_basic(name),
				Check:  tfresource.TestCheckResourceAttr(resourceName, "global_default", "false"),
			},
			// Enable global_default — must be an in-place update, not a replace.
			{
				Config: testAccPriorityClassV1Config_globalDefault(name, true),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: tfresource.TestCheckResourceAttr(resourceName, "global_default", "true"),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "metadata.0.generation"},
			},
			// Disable it again.
			{
				Config: testAccPriorityClassV1Config_basic(name),
				Check:  tfresource.TestCheckResourceAttr(resourceName, "global_default", "false"),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "metadata.0.generation"},
			},
		},
	})
}

// TestAccPriorityClassV1_disappears verifies that if the PriorityClass is
// deleted outside Terraform (e.g. kubectl delete), the next plan detects it
// is gone and proposes to recreate it.
func TestAccPriorityClassV1_disappears(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-pc")
	resourceName := "kubernetes_priority_class_v1.test"

	tfresource.ParallelTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			// Step 1: create the resource.
			{
				Config: testAccPriorityClassV1Config_basic(name),
				Check:  tfresource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
			},
			// Step 2: delete out-of-band, then refresh state.
			// RefreshState: true triggers a Read without applying config.
			// Read returns 404 → provider calls RemoveResource → state cleared.
			// The subsequent plan (ExpectNonEmptyPlan) then shows a create.
			{
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				PreConfig: func() {
					// Delete the PriorityClass outside Terraform to simulate an
					// out-of-band deletion. Errors are intentionally ignored —
					// the object may already be gone.
					_ = exec.Command("kubectl", "delete", "priorityclass", name, "--ignore-not-found").Run()
				},
			},
		},
	})
}

// TestAccPriorityClassV1_invalidValue verifies that value > 1,000,000,000
// is rejected at plan time by the int64validator, not at apply time by the API.
func TestAccPriorityClassV1_invalidValue(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-pc")

	tfresource.ParallelTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config:      testAccPriorityClassV1Config_withValue(name, 1_000_000_001),
				ExpectError: regexp.MustCompile(`1000000001`),
			},
		},
	})
}

// TestAccPriorityClassV1_nameAndGenerateNameConflict verifies that setting
// both name and generate_name is rejected at plan time by the ConflictsWith validator.
func TestAccPriorityClassV1_nameAndGenerateNameConflict(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-pc")

	tfresource.ParallelTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config:      testAccPriorityClassV1Config_nameAndGenerateName(name, "tf-acc-pc-"),
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
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

func testAccPriorityClassV1Config_withValue(name string, value int) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata {
    name = %[1]q
  }

  value             = %[2]d
  preemption_policy = "Never"
}
`, name, value)
}

func testAccPriorityClassV1Config_withPreemptionPolicy(name, policy string) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata {
    name = %[1]q
  }

  value             = 100
  preemption_policy = %[2]q
}
`, name, policy)
}

func testAccPriorityClassV1Config_globalDefault(name string, globalDefault bool) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata {
    name = %[1]q
  }

  value             = 100
  preemption_policy = "Never"
  global_default    = %[2]t
}
`, name, globalDefault)
}

func testAccPriorityClassV1Config_nameAndGenerateName(name, prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata {
    name          = %[1]q
    generate_name = %[2]q
  }

  value = 100
}
`, name, prefix)
}
