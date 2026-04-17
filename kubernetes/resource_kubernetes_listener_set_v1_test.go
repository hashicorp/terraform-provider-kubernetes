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

func TestAccKubernetesListenerSetV1_basic(t *testing.T) {
	var conf gatewayv1.ListenerSet
	rName := acctest.RandomWithPrefix("tf-acc-test")
	gcName := acctest.RandomWithPrefix("tf-acc-test-gc")
	resourceName := "kubernetes_listener_set_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckListenerSetV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccListenerSetV1ConfigBasic(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckListenerSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", rName),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.name", "http-extra"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.port", "8080"),
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

func testAccCheckListenerSetV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).GatewayClientset()
	if err != nil {
		return err
	}
	ctx := context.Background()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_listener_set_v1" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.ListenerSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("ListenerSet still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckListenerSetV1Exists(n string, obj *gatewayv1.ListenerSet) resource.TestCheckFunc {
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

		out, err := conn.ListenerSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccListenerSetV1ConfigBasic(rName, gcName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = %[2]q
  }
  spec {
    controller_name = "example.com/gateway-controller"
  }
}

resource "kubernetes_gateway_v1" "test" {
  metadata {
    name = "%[1]s-gw"
  }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.test.metadata.0.name
    listeners {
      name     = "http"
      port     = 80
      protocol = "HTTP"
    }
  }
}

resource "kubernetes_listener_set_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    parent_ref {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }
    listeners {
      name     = "http-extra"
      port     = 8080
      protocol = "HTTP"
    }
  }
}
`, rName, gcName)
}
