// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package admissionregistrationv1_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/admissionregistrationv1"
)

func TestValidatingAdmissionPolicySchema(t *testing.T) {
	ctx := context.Background()

	resp := &fwresource.SchemaResponse{}
	admissionregistrationv1.NewValidatingAdmissionPolicy().Schema(ctx, fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("building the schema: %+v", resp.Diagnostics)
	}

	if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Fatalf("validating the schema: %+v", diags)
	}

	if _, ok := resp.Schema.Blocks["bootstrap_connection"]; !ok {
		t.Error(`missing "bootstrap_connection" block`)
	}
}

func TestAccValidatingAdmissionPolicy_bootstrapConnection(t *testing.T) {
	name := "test-policy-bootstrap"
	kubeconfig := os.Getenv("KUBE_CONFIG_PATH")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			if kubeconfig == "" {
				t.Skip("KUBE_CONFIG_PATH must be set for the bootstrap_connection acceptance test")
			}
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			// The block is write-only throughout.
			tfversion.SkipBelow(tfversion.Version1_11_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testValidatingAdmissionPolicyConfig_bootstrapConnection(name,
					fmt.Sprintf("config_path = %q", kubeconfig)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_validating_admission_policy_v1.test", "metadata.name", name),
					resource.TestCheckNoResourceAttr("kubernetes_validating_admission_policy_v1.test", "bootstrap_connection.0.config_path"),
				),
			},
			{
				Config: testValidatingAdmissionPolicyConfig_bootstrapConnection(name,
					fmt.Sprintf("config_paths = [%q]", kubeconfig)),
				PlanOnly: true,
			},
		},
	})
}

func testValidatingAdmissionPolicyConfig_bootstrapConnection(name, connection string) string {
	return fmt.Sprintf(`
resource "kubernetes_validating_admission_policy_v1" "test" {
  bootstrap_connection {
    %s
  }

  metadata = {
    name = %q
  }

  spec = {
    failure_policy    = "Fail"
    audit_annotations = []

    match_constraints = {
      resource_rules = [{
        api_groups   = ["apps"]
        api_versions = ["v1"]
        operations   = ["CREATE", "UPDATE"]
        resources    = ["deployments"]
      }]
    }

    validations = [{
      expression = "object.spec.replicas <= 5"
      message    = "Replica count must not exceed 5"
    }]
  }
}
`, connection, name)
}
