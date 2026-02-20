// Copyright (c) HashiCorp, Inc.

package admissionregistrationv1_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var policyName = "test-policy"

func TestAccValidatingAdmissionPolicyBinding_basic(t *testing.T) {
	name := "test-policy-binding"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testValidatingAdmissionPolicyBindingConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_validating_admission_policy_binding_v1.test", "metadata.name", name),
					resource.TestCheckResourceAttr("kubernetes_validating_admission_policy_binding_v1.test", "spec.validation_actions.0", "Deny"),
				),
			},
			{
				ResourceName:      "kubernetes_validating_admission_policy_binding_v1.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"timeouts",
					"metadata.resource_version",
				},
			},
		},
	})
}

func testValidatingAdmissionPolicyBindingConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_validating_admission_policy_binding_v1" "test" {
  metadata = {
    name = %q
  }

  spec = {
    policy_name = %[1]q

    validation_actions = ["Deny"]

    param_ref = {
      name                        = "test-policy-binding"
      namespace                   = "test-namespace"
      parameter_not_found_action  = "Deny"
    }
  }
}
`, name, policyName)
}

func TestAccValidatingAdmissionPolicyBinding_withMatchResources(t *testing.T) {
	name := "test-policy-binding-resources"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testValidatingAdmissionPolicyBindingConfig_withMatchResources(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_validating_admission_policy_binding_v1.test", "metadata.name", name),
					resource.TestCheckResourceAttr("kubernetes_validating_admission_policy_binding_v1.test", "spec.match_resources.resource_rules.0.api_groups.0", "apps"),
					resource.TestCheckResourceAttr("kubernetes_validating_admission_policy_binding_v1.test", "spec.match_resources.resource_rules.0.resources.0", "deployments"),
				),
			},
		},
	})
}

func testValidatingAdmissionPolicyBindingConfig_withMatchResources(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_validating_admission_policy_binding_v1" "test" {
  metadata = {
    name = %q
  }

  spec = {
    policy_name = %[1]q

    validation_actions = ["Deny"]

    param_ref = {
      name                        = "test-policy-binding"
      namespace                   = "test-namespace"
      parameter_not_found_action  = "Deny"
    }

    match_resources = {
      resource_rules = [{
        api_groups   = ["apps"]
        api_versions = ["v1"]
        operations   = ["CREATE", "UPDATE"]
        resources    = ["deployments"]
      }]
    }
  }
}
`, name, policyName)
}

func TestAccValidatingAdmissionPolicyBinding_update(t *testing.T) {
	name := "test-policy-binding-update"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testValidatingAdmissionPolicyBindingConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_validating_admission_policy_binding_v1.test", "spec.validation_actions.0", "Deny"),
				),
			},
			{
				Config: testValidatingAdmissionPolicyBindingConfig_updated(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_validating_admission_policy_binding_v1.test", "spec.validation_actions.0", "Audit"),
				),
			},
		},
	})
}

func testValidatingAdmissionPolicyBindingConfig_updated(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_validating_admission_policy_binding_v1" "test" {
  metadata = {
    name = %q
  }

  spec = {
    policy_name = %[1]q

    validation_actions = ["Audit"]

    param_ref = {
      name                        = "test-policy-binding"
      namespace                   = "test-namespace"
      parameter_not_found_action  = "Deny"
    }
  }
}
`, name, policyName)
}
