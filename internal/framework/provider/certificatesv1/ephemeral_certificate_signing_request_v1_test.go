// Copyright IBM Corp. 2017, 2026

package certificatesv1_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccEpehemeralCertificateSigningRequest_basic(t *testing.T) {
	name := "test"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testEphemeralCertificateSigningRequestRequestV1Config(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("certificate"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func testEphemeralCertificateSigningRequestRequestV1Config(name string) string {
	return fmt.Sprintf(`
   ephemeral "kubernetes_certificate_signing_request_v1" "test" {
      metadata {
        name = %q
      }
      spec {
        request = <<EOT
-----BEGIN CERTIFICATE REQUEST-----
MIHSMIGBAgEAMCoxGDAWBgNVBAoTD2V4YW1wbGUgY2x1c3RlcjEOMAwGA1UEAxMF
YWRtaW4wTjAQBgcqhkjOPQIBBgUrgQQAIQM6AASSG8S2+hQvfMq5ucngPCzK0m0C
ImigHcF787djpF2QDbz3oQ3QsM/I7ftdjB/HHlG2a5YpqjzT0KAAMAoGCCqGSM49
BAMCA0AAMD0CHQDErNLjX86BVfOsYh/A4zmjmGknZpc2u6/coTHqAhxcR41hEU1I
DpNPvh30e0Js8/DYn2YUfu/pQU19
-----END CERTIFICATE REQUEST-----
EOT
        usages = ["client auth"]
        signer_name = "kubernetes.io/kube-apiserver-client"
      }
      auto_approve = true
    }

    provider "echo" {
      data = ephemeral.kubernetes_certificate_signing_request_v1.test
    }

    resource "echo" "test" {}`, name)
}
