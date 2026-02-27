// Copyright IBM Corp. 2017, 2026

package admissionregistrationv1_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccValidatingAdmissionPolicy_basic(t *testing.T) {
	name := "test-policy"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testValidatingAdmissionPolicyConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_validating_admission_policy_v1.test", "metadata.name", name),
					resource.TestCheckResourceAttr("kubernetes_validating_admission_policy_v1.test", "spec.validations.0.expression", "object.spec.replicas <= 5"),
					resource.TestCheckResourceAttr("kubernetes_validating_admission_policy_v1.test", "spec.validations.0.message", "Replica count must not exceed 5"),
				),
			},
			{
				ResourceName:      "kubernetes_validating_admission_policy_v1.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"timeouts",
					"metadata.resource_version",
					"spec.match_constraints.match_policy",
				},
			},
		},
	})
}

func TestAccValidatingAdmissionPolicy_withMatchConstraints(t *testing.T) {
	name := "test-policy-constraints"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testValidatingAdmissionPolicyConfig_withMatchConstraints(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_validating_admission_policy_v1.test", "metadata.name", name),
					resource.TestCheckResourceAttr("kubernetes_validating_admission_policy_v1.test", "spec.match_constraints.resource_rules.0.api_groups.0", "apps"),
					resource.TestCheckResourceAttr("kubernetes_validating_admission_policy_v1.test", "spec.match_constraints.resource_rules.0.resources.0", "deployments"),
				),
			},
		},
	})
}

func TestAccValidatingAdmissionPolicy_update(t *testing.T) {
	name := "test-policy-update"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testValidatingAdmissionPolicyConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_validating_admission_policy_v1.test", "spec.validations.0.message", "Replica count must not exceed 5"),
				),
			},
			{
				Config: testValidatingAdmissionPolicyConfig_updated(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_validating_admission_policy_v1.test", "spec.validations.0.message", "Replica count must not exceed 10"),
				),
			},
		},
	})
}

func testValidatingAdmissionPolicyConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_validating_admission_policy_v1" "test" {
  metadata = {
    name = %q
  }

  spec = {
    failure_policy = "Fail"

    match_constraints = {
      resource_rules = [{
        api_groups   = ["apps"]
        api_versions = ["v1"]
        operations   = ["CREATE", "UPDATE"]
        resources    = ["deployments"]
      }]
    }

    audit_annotations = [{
      key              = "example"
      value_expression = "'ok'"
    }]

    validations = [{
      expression = "object.spec.replicas <= 5"
      message    = "Replica count must not exceed 5"
    }]
  }
}
`, name)
}

func testValidatingAdmissionPolicyConfig_withMatchConstraints(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_validating_admission_policy_v1" "test" {
  metadata = {
    name = %q
  }

  spec = {
    failure_policy = "Fail"

    match_constraints = {
      resource_rules = [{
        api_groups   = ["apps"]
        api_versions = ["v1"]
        operations   = ["CREATE", "UPDATE"]
        resources    = ["deployments"]
      }]
    }

    audit_annotations = [{
      key              = "example"
      value_expression = "'ok'"
    }]

    validations = [{
      expression = "object.spec.replicas <= 5"
      message    = "Replica count must not exceed 5"
    }]
  }
}
`, name)
}

func testValidatingAdmissionPolicyConfig_updated(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_validating_admission_policy_v1" "test" {
  metadata = {
    name = %q
  }

  spec = {
    failure_policy = "Fail"

    match_constraints = {
      resource_rules = [{
        api_groups   = ["apps"]
        api_versions = ["v1"]
        operations   = ["CREATE", "UPDATE"]
        resources    = ["deployments"]
      }]
    }

    audit_annotations = [{
      key              = "example"
      value_expression = "'ok'"
    }]

    validations = [{
      expression = "object.spec.replicas <= 10"
      message    = "Replica count must not exceed 10"
    }]
  }
}
`, name)
}
