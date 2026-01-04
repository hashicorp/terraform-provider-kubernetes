// Copyright IBM Corp. 2017, 2025
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	corev1 "k8s.io/api/core/v1"
)

func TestAccKubernetesTokenRequestV1_basic(t *testing.T) {
	var conf corev1.ServiceAccount
	saName := "kubernetes_service_account_v1.test"
	resourceName := "kubernetes_token_request_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesServiceAccountV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesTokenRequestV1Config_basic(`["api"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceAccountV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(saName, "metadata.0.name", "tokentest"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", "tokentest"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.audiences.0", "api"),
					resource.TestCheckResourceAttrSet(resourceName, "token"),
				),
			},
			{
				Config: testAccKubernetesTokenRequestV1Config_basic(`["api", "vault"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceAccountV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(saName, "metadata.0.name", "tokentest"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", "tokentest"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.audiences.0", "api"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.audiences.1", "vault"),
					resource.TestCheckResourceAttrSet(resourceName, "token"),
				),
			},
		},
	})
}

func testAccKubernetesTokenRequestV1Config_basic(audiences string) string {
	return fmt.Sprintf(`resource "kubernetes_service_account_v1" "test" {
  metadata {
    name = "tokentest"
  }
}

resource "kubernetes_token_request_v1" "test" {
  metadata {
    name = kubernetes_service_account_v1.test.metadata.0.name
  }
  spec {
    audiences = %s
  }
}
`, audiences)
}
