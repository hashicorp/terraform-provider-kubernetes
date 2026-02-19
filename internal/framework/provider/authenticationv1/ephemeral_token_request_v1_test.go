// Copyright IBM Corp. 2017, 2025

package authenticationv1_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccEpehemeralTokenRequest_basic(t *testing.T) {
	name := "default"
	namespace := "default"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testEphemeralTokenRequestV1Config(name, namespace),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("token"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func testEphemeralTokenRequestV1Config(name, namespace string) string {
	return fmt.Sprintf(`
   ephemeral "kubernetes_token_request_v1" "test" {
      metadata {
        name = %q
        namespace = %q
      }
      spec {
        audiences = ["api", "vault"]
      }
    }

    provider "echo" {
      data = ephemeral.kubernetes_token_request_v1.test
    }

    resource "echo" "test" {}`, name, namespace)
}
