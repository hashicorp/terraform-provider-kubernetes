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

func TestAccKubernetesReferenceGrantV1_basic(t *testing.T) {
	var conf gatewayv1.ReferenceGrant
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_reference_grant_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckReferenceGrantV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccReferenceGrantV1ConfigBasic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckReferenceGrantV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", rName),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.from.0.kind", "HTTPRoute"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.to.0.kind", "Service"),
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

func testAccCheckReferenceGrantV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).GatewayClientset()
	if err != nil {
		return err
	}
	ctx := context.Background()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_reference_grant_v1" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.ReferenceGrants(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("ReferenceGrant still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckReferenceGrantV1Exists(n string, obj *gatewayv1.ReferenceGrant) resource.TestCheckFunc {
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

		out, err := conn.ReferenceGrants(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccReferenceGrantV1ConfigBasic(rName string) string {
	return fmt.Sprintf(`
resource "kubernetes_reference_grant_v1" "test" {
  metadata {
    name      = %q
    namespace = "default"
  }
  spec {
    from {
      group     = "gateway.networking.k8s.io"
      kind      = "HTTPRoute"
      namespace = "default"
    }
    to {
      group = ""
      kind  = "Service"
    }
  }
}
`, rName)
}
