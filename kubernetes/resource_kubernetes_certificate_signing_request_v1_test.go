package kubernetes

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesCertificateSigningRequestV1_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	usages := []string{"client auth"}
	signerName := "kubernetes.io/kube-apiserver-client"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.22.0")
		},
		IDRefreshName:     "kubernetes_certificate_signing_request_v1.test",
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesCertificateSigningRequestV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesCertificateSigningRequestV1Config_basic(name, signerName, usages, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesCertificateSigningRequestV1Valid,
					resource.TestCheckResourceAttrSet("kubernetes_certificate_signing_request_v1.test", "certificate"),
					resource.TestCheckResourceAttr("kubernetes_certificate_signing_request_v1.test", "spec.0.signer_name", signerName),
					resource.TestCheckResourceAttr("kubernetes_certificate_signing_request_v1.test", "spec.0.usages.0", usages[0]),
				),
			},
		},
	})
}

func TestAccKubernetesCertificateSigningRequestV1_generateName(t *testing.T) {
	generateName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.22.0")
		},
		IDRefreshName:     "kubernetes_certificate_signing_request_v1.test",
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesCertificateSigningRequestV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesCertificateSigningRequestV1Config_generateName(generateName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesCertificateSigningRequestV1Valid,
					resource.TestCheckResourceAttrSet("kubernetes_certificate_signing_request_v1.test", "certificate"),
				),
			},
		},
	})
}

// testAccCheckKubernetesCertificateSigningRequestV1Valid checks to see that the locally-stored certificate
// contains a valid PEM preamble. It also checks that the CSR resource has been deleted from Kubernetes, since
// the CSR is only supposed to exist momentarily as the certificate is generated. (CSR resources are ephemeral
// in Kubernetes and therefore are only used temporarily).
func testAccCheckKubernetesCertificateSigningRequestV1Valid(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type == "kubernetes_certificate_signing_request_v1" {
			if !strings.HasPrefix(rs.Primary.Attributes["certificate"], "-----BEGIN CERTIFICATE----") {
				return fmt.Errorf("certificate is missing cert PEM preamble from resource: %s", rs.Primary.ID)
			}
		}
	}
	return testAccCheckKubernetesCertificateSigningRequestV1RemoteResourceDeleted(s)
}

func testAccCheckKubernetesCertificateSigningRequestV1RemoteResourceDeleted(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.Background()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_certificate_signing_request_v1" {
			continue
		}
		out, err := conn.CertificatesV1().CertificateSigningRequests().Get(ctx, rs.Primary.ID, metav1.GetOptions{})
		if err == nil {
			if out.Name == rs.Primary.ID {
				return fmt.Errorf("CertificateSigningRequest still exists in Kubernetes: %s", rs.Primary.ID)
			}
		}
	}
	return nil
}

func testAccCheckKubernetesCertificateSigningRequestV1Destroy(s *terraform.State) error {
	return testAccCheckKubernetesCertificateSigningRequestV1RemoteResourceDeleted(s)
}

func testAccKubernetesCertificateSigningRequestV1Config_basic(name, signerName string, usages []string, autoApprove bool) string {
	return fmt.Sprintf(`resource "kubernetes_certificate_signing_request_v1" "test" {
  metadata {
    name = %q
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
    signer_name = %q
    usages      = %q
  }
}
`, name, autoApprove, signerName, usages)
}

func testAccKubernetesCertificateSigningRequestV1Config_generateName(generateName string) string {
	return fmt.Sprintf(`resource "kubernetes_certificate_signing_request_v1" "test" {
  metadata {
    generate_name = %q
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
    signer_name = "kubernetes.io/kube-apiserver-client"
    usages      = ["client auth"]
  }
}
`, generateName)
}
