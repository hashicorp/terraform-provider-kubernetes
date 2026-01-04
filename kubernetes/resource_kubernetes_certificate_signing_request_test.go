// Copyright IBM Corp. 2017, 2025
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesCertificateSigningRequest_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	usages := []string{"client auth"}
	signerName := "kubernetes.io/legacy-unknown"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionGreaterThanOrEqual(t, "1.22.0")
		},

		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesCertificateSigningRequestDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesCertificateSigningRequestConfig_basic(name, signerName, usages, true),
				Check:  testAccCheckKubernetesCertificateSigningRequestValid,
			},
		},
	})
}

func TestAccKubernetesCertificateSigningRequest_generateName(t *testing.T) {
	generateName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionGreaterThanOrEqual(t, "1.22.0")
		},

		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesCertificateSigningRequestDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesCertificateSigningRequestConfig_generateName(generateName),
				Check:  testAccCheckKubernetesCertificateSigningRequestValid,
			},
		},
	})
}

// testAccCheckKubernetesCertificateSigningRequestValid checks to see that the locally-stored certificate
// contains a valid PEM preamble. It also checks that the CSR resource has been deleted from Kubernetes, since
// the CSR is only supposed to exist momentarily as the certificate is generated. (CSR resources are ephemeral
// in Kubernetes and therefore are only used temporarily).
func testAccCheckKubernetesCertificateSigningRequestValid(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type == "kubernetes_certificate_signing_request" {
			if !strings.HasPrefix(rs.Primary.Attributes["certificate"], "-----BEGIN CERTIFICATE----") {
				return fmt.Errorf("certificate is missing cert PEM preamble from resource: %s", rs.Primary.ID)
			}
		}
	}
	return testAccCheckKubernetesCertificateSigningRequestRemoteResourceDeleted(s)
}

func testAccCheckKubernetesCertificateSigningRequestRemoteResourceDeleted(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_certificate_signing_request" {
			continue
		}

		out, err := conn.CertificatesV1beta1().CertificateSigningRequests().Get(ctx, rs.Primary.ID, metav1.GetOptions{})
		if err == nil {
			if out.Name == rs.Primary.ID {
				return fmt.Errorf("CertificateSigningRequest still exists in Kubernetes: %s", rs.Primary.ID)
			}
		}
	}
	return nil
}

func testAccCheckKubernetesCertificateSigningRequestDestroy(s *terraform.State) error {
	return testAccCheckKubernetesCertificateSigningRequestRemoteResourceDeleted(s)
}

func testAccKubernetesCertificateSigningRequestConfig_basic(name, signerName string, usages []string, autoApprove bool) string {
	return fmt.Sprintf(`resource "kubernetes_certificate_signing_request" "test" {
  metadata {
    name = "%s"
  }
  auto_approve = %t
  spec {
    request     = <<EOT
-----BEGIN CERTIFICATE REQUEST-----
MIHSMIGBAgEAMCoxGDAWBgNVBAoTD2V4YW1wbGUgY2x1c3RlcjEOMAwGA1UEAxMF
YWRtaW4wTjAQBgcqhkjOPQIBBgUrgQQAIQM6AASSG8S2+hQvfMq5ucngPCzK0m0C
ImigHcF787djpF2QDbz3oQ3QsM/I7ftdjB/HHlG2a5YpqjzT0KAAMAoGCCqGSM49
BAMCA0AAMD0CHQDErNLjX86BVfOsYh/A4zmjmGknZpc2u6/coTHqAhxcR41hEU1I
DpNPvh30e0Js8/DYn2YUfu/pQU19
-----END CERTIFICATE REQUEST-----
EOT
    signer_name = "%s"
    usages      = %q
  }
}
`, name, autoApprove, signerName, usages)
}

func testAccKubernetesCertificateSigningRequestConfig_generateName(generateName string) string {
	return fmt.Sprintf(`resource "kubernetes_certificate_signing_request" "test" {
  metadata {
    generate_name = "%s"
  }
  auto_approve = true
  spec {
    request     = <<EOT
-----BEGIN CERTIFICATE REQUEST-----
MIHSMIGBAgEAMCoxGDAWBgNVBAoTD2V4YW1wbGUgY2x1c3RlcjEOMAwGA1UEAxMF
YWRtaW4wTjAQBgcqhkjOPQIBBgUrgQQAIQM6AASSG8S2+hQvfMq5ucngPCzK0m0C
ImigHcF787djpF2QDbz3oQ3QsM/I7ftdjB/HHlG2a5YpqjzT0KAAMAoGCCqGSM49
BAMCA0AAMD0CHQDErNLjX86BVfOsYh/A4zmjmGknZpc2u6/coTHqAhxcR41hEU1I
DpNPvh30e0Js8/DYn2YUfu/pQU19
-----END CERTIFICATE REQUEST-----
EOT
    signer_name = "kubernetes.io/legacy-unknown"
    usages      = ["client auth"]
  }
}
`, generateName)
}
