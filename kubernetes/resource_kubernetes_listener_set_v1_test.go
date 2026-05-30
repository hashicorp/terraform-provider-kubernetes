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

func TestAccKubernetesListenerSetV1_multipleListeners(t *testing.T) {
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
				Config: testAccListenerSetV1ConfigMultipleListeners(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckListenerSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.port", "8080"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.1.port", "8443"),
				),
			},
		},
	})
}

func TestAccKubernetesListenerSetV1_withTLS(t *testing.T) {
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
				Config: testAccListenerSetV1ConfigWithTLS(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckListenerSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.protocol", "HTTPS"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.tls.0.mode", "Terminate"),
				),
			},
		},
	})
}

func TestAccKubernetesListenerSetV1_updateListenerPort(t *testing.T) {
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
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.port", "8080"),
				),
			},
			{
				Config: testAccListenerSetV1ConfigUpdatedPort(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckListenerSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.listeners.0.port", "9090"),
				),
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
  metadata { name = %[2]q }
  spec { controller_name = "example.com/gateway-controller" }
}

resource "kubernetes_gateway_v1" "test" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.test.metadata.0.name
    listeners { name = "http"; port = 80; protocol = "HTTP" }
  }
}

resource "kubernetes_listener_set_v1" "test" {
  metadata { name = %[1]q }
  spec {
    parent_ref { name = kubernetes_gateway_v1.test.metadata.0.name }
    listeners { name = "http-extra"; port = 8080; protocol = "HTTP" }
  }
}
`, rName, gcName)
}

func testAccListenerSetV1ConfigMultipleListeners(rName, gcName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "test" {
  metadata { name = %[2]q }
  spec { controller_name = "example.com/gateway-controller" }
}

resource "kubernetes_gateway_v1" "test" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.test.metadata.0.name
    listeners { name = "http"; port = 80; protocol = "HTTP" }
  }
}

resource "kubernetes_listener_set_v1" "test" {
  metadata { name = %[1]q }
  spec {
    parent_ref { name = kubernetes_gateway_v1.test.metadata.0.name }
    listeners { name = "http-alt"; port = 8080; protocol = "HTTP" }
    listeners { name = "https"; port = 8443; protocol = "HTTPS" }
  }
}
`, rName, gcName)
}

func testAccListenerSetV1ConfigWithTLS(rName, gcName string) string {
	return fmt.Sprintf(`
resource "kubernetes_secret_v1" "test" {
  metadata { name = "%[1]s-tls" }
  data {
    tls.crt = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURCekNDQWMrZ0F3SUJBZ0lSQUp2ZjJ2TnBkYnJmYnFMd0t0K2JFQXdNQTVHQTFVZERnNVY05DUXF3R0FZRApWUVFLRXd4ME1CNHdNQ0lHQTFVRUNnd0xYMEV4RGpBUUJnTlZCQU1NR0VwaklGb1hEVEk0TURreE1qQTVPVEF4Ck1CTUdCeXFHU000OUFnRUdDQ3FHU000OUF3RUhBMElBQkJqWVN5YkI2dXJkVjB1NnlLWUJxZjIwQlNjY0ZmYQpGdWJxQ0V3bVd6T2hJb1FjYjVxM0d0QWJ3cGdFZlJnS3FwNk9jUjM3Q0R1VUdKc3JkVXhKdEh5M29hMGRJWjcKZ0JmREI1c29rZ3N1eXF0T09sR0d0V2RlR3B3aEJlR0h6Nmh5VXJhNmdJZnB5b0tQZVdKbEhJNWdZcGJkYkJVCnlrV3NvN0p6cVh1dG1sWnNpT2hZbW1JYk9vVWd4cW1xN1VJMXNkRzRrWkN1T2RqNGx2VjB5V0NqT1l5YkpjSQpGQW5oQWdNQkFBRXdEUVlKS29aSWh2Y05BUUVMQlFBRGdnRUJBTjQ3TlJhVUJZb3JkV3M3c3R4MmJ1N1p6aGQwCmMwVnF2WjdGZEVhN3ZQd0FqS3J5aVdGdXZsQWp3eXpJWU5hR0pZcWxjNzR2NlN4dU1vQW93VzJvMXhLWlBvT0MKM1h6N3dJNlJxYjJ3bUJ3N2R5VjN1V01xM2lWd3N1aDdJc2VJdXJrYjFyZmJyNjNqV2VZMXp3T250S3ZzV2NxCm5vUXR3cDdIb0lPbWVYQnFZMnhXcWpRbVZ5b3F6U3N5WnVZcXp5QXVJN3V5eU5lZnBJNlBnN1ZlVHh2b3VZaQo4MjJrZGZsUjJ3V0l6Y3F4cDh3b253UjN3aWJ3Z2hLcHJrZ1p5cXVhM3FhR2VlQ2ZyZ3F5N2R3b1F3PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg=="
    tls.key = "LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb3dJQkFBS0NBUUVBMmZ5b3l2Q1JvZkZ3Z2ZQWmZ4TjJhMm90UWJZVXlKMHl4N1h3aE5YQzR4MmZJNmsKMzVhVnlqVjRyVW1JNGZxVjJhQW93R3F3V0N0M1BtQkxZMXZqY2JjT0x0b2RlZU9oRnNpQWp3eXpJWU5hR0pZCnFxY3N5N3F3b0lPbWVYQnFZMnhXcWpRbVZ5b3F6U3N5WnVZcXp5QXVJN3V5eU5lZnBJNlBnN1ZlVHh2b3VZaQo0MmRrZGZsUjJ3V0l6Y3F4cDh3b253UjN3aWJ3Z2hLcHJrZ1p5cXVhM3FhR2VlQ2ZyZ3F5N2R3b1F3PQotLS0tLUVORCBSU0EgUFJJVkFURSBLRVktLS0tLQo="
  }
  type = "kubernetes.io/tls"
}

resource "kubernetes_gateway_class_v1" "test" {
  metadata { name = %[2]q }
  spec { controller_name = "example.com/gateway-controller" }
}

resource "kubernetes_gateway_v1" "test" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.test.metadata.0.name
    listeners { name = "http"; port = 80; protocol = "HTTP" }
  }
}

resource "kubernetes_listener_set_v1" "test" {
  metadata { name = %[1]q }
  spec {
    parent_ref { name = kubernetes_gateway_v1.test.metadata.0.name }
    listeners {
      name     = "https"
      port     = 8443
      protocol = "HTTPS"
      tls {
        mode = "Terminate"
      }
    }
  }
}
`, rName, gcName)
}

func testAccListenerSetV1ConfigUpdatedPort(rName, gcName string) string {
	return fmt.Sprintf(`
resource "kubernetes_gateway_class_v1" "test" {
  metadata { name = %[2]q }
  spec { controller_name = "example.com/gateway-controller" }
}

resource "kubernetes_gateway_v1" "test" {
  metadata { name = "%[1]s-gw" }
  spec {
    gateway_class_name = kubernetes_gateway_class_v1.test.metadata.0.name
    listeners { name = "http"; port = 80; protocol = "HTTP" }
  }
}

resource "kubernetes_listener_set_v1" "test" {
  metadata { name = %[1]q }
  spec {
    parent_ref { name = kubernetes_gateway_v1.test.metadata.0.name }
    listeners { name = "http-extra"; port = 9090; protocol = "HTTP" }
  }
}
`, rName, gcName)
}
