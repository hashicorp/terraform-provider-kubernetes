// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestAccKubernetesBackendTLSPolicyV1_basic(t *testing.T) {
	var conf gatewayv1.BackendTLSPolicy
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_backend_tls_policy_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckBackendTLSPolicyV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBackendTLSPolicyV1ConfigBasic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckBackendTLSPolicyV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", rName),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.target_refs.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.target_refs.0.kind", "Service"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.validation.0.hostname", "example.com"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "metadata.0.uid"},
			},
		},
	})
}

func testAccCheckBackendTLSPolicyV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).GatewayClientset()
	if err != nil {
		return err
	}
	ctx := context.Background()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_backend_tls_policy_v1" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.BackendTLSPolicies(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("BackendTLSPolicy still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckBackendTLSPolicyV1Exists(n string, obj *gatewayv1.BackendTLSPolicy) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn, err := testAccProvider.Meta().(KubeClientsets).GatewayClientset()
		if err != nil {
			return err
		}

		ctx := context.Background()
		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		out, err := conn.BackendTLSPolicies(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccBackendTLSPolicyV1ConfigBasic(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_service_v1" "test" {
  metadata {
    name = "%s-svc"
  }
  spec {
    selector = {
      app = "test"
    }
    port {
      port        = 443
      target_port = 443
    }
  }
}

resource "kubernetes_backend_tls_policy_v1" "test" {
  metadata {
    name = %q
  }
  spec {
    target_refs {
      group = ""
      kind  = "Service"
      name  = kubernetes_service_v1.test.metadata.0.name
    }
    validation {
      hostname                   = "example.com"
      well_known_ca_certificates = "System"
    }
  }
}
`, rName, rName)
}
