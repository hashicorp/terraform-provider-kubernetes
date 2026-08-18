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

// TestAccRuntimeClassV1_generateName verifies that when generate_name is used,
// the server-assigned name is correctly read back into state and the plan
// settles to empty after create.
func TestAccRuntimeClassV1_generateName(t *testing.T) {
	prefix := "tf-acc-gen-"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRuntimeClassV1Config_generateName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					// name must be non-empty (server assigned)
					resource.TestCheckResourceAttrSet(
						"kubernetes_runtime_class_v1.test",
						"metadata.0.name",
					),
					resource.TestCheckResourceAttr(
						"kubernetes_runtime_class_v1.test",
						"metadata.0.generate_name",
						prefix,
					),
					resource.TestCheckResourceAttrSet(
						"kubernetes_runtime_class_v1.test",
						"metadata.0.uid",
					),
					resource.TestCheckResourceAttr(
						"kubernetes_runtime_class_v1.test",
						"handler",
						"runc",
					),
				),
			},
			// Re-apply — plan must be empty (server-assigned name must not cause drift).
			{
				Config:             testAccRuntimeClassV1Config_generateName(prefix),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// TestAccRuntimeClassV1_annotations verifies create, update, and idempotency
// for the annotations metadata field.
func TestAccRuntimeClassV1_annotations(t *testing.T) {
	name := "tf-acc-test-annotations"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRuntimeClassV1Config_withAnnotation(name, "owner", "team-a"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"kubernetes_runtime_class_v1.test",
						"metadata.0.name",
						name,
					),
					resource.TestCheckResourceAttr(
						"kubernetes_runtime_class_v1.test",
						"metadata.0.annotations.owner",
						"team-a",
					),
				),
			},
			{
				Config: testAccRuntimeClassV1Config_withAnnotation(name, "owner", "team-b"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"kubernetes_runtime_class_v1.test",
						"metadata.0.annotations.owner",
						"team-b",
					),
				),
			},
			// Re-apply same config — plan must be empty.
			{
				Config:             testAccRuntimeClassV1Config_withAnnotation(name, "owner", "team-b"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// TestAccRuntimeClassV1_labelsAndAnnotations verifies that both labels and
// annotations can be set together and the plan is idempotent after create.
func TestAccRuntimeClassV1_labelsAndAnnotations(t *testing.T) {
	name := "tf-acc-test-meta-full"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRuntimeClassV1Config_labelsAndAnnotations(name),
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
					resource.TestCheckResourceAttr(
						"kubernetes_runtime_class_v1.test",
						"metadata.0.annotations.owner",
						"team-a",
					),
				),
			},
			// Idempotency check — re-apply must produce an empty plan.
			{
				Config:             testAccRuntimeClassV1Config_labelsAndAnnotations(name),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// TestAccRuntimeClassV1_importWithIdentity exercises Terraform 1.12+ structured
// import using an identity block (api_version + kind + name) instead of a bare
// string ID.
func TestAccRuntimeClassV1_importWithIdentity(t *testing.T) {
	name := "tf-acc-test-import-identity"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRuntimeClassV1Config_basic(name),
			},
			{
				ResourceName:    "kubernetes_runtime_class_v1.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
				// ImportStateVerify is not supported with plannable import blocks
				// (ImportBlockWithResourceIdentity). The import itself succeeding
				// without error is the assertion.
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

func testAccRuntimeClassV1Config_generateName(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_runtime_class_v1" "test" {
  metadata {
    generate_name = %q
  }
  handler = "runc"
}
`, prefix)
}

func testAccRuntimeClassV1Config_withAnnotation(name, annotationKey, annotationValue string) string {
	return fmt.Sprintf(`
resource "kubernetes_runtime_class_v1" "test" {
  metadata {
    name = %q
    annotations = {
      %q = %q
    }
  }
  handler = "runc"
}
`, name, annotationKey, annotationValue)
}

func testAccRuntimeClassV1Config_labelsAndAnnotations(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_runtime_class_v1" "test" {
  metadata {
    name = %q
    labels = {
      env = "staging"
    }
    annotations = {
      owner = "team-a"
    }
  }
  handler = "runc"
}
`, name)
}
