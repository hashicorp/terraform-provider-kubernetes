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

func TestAccKubernetesGRPCRouteV1_basic(t *testing.T) {
	var conf gatewayv1.GRPCRoute
	rName := acctest.RandomWithPrefix("tf-acc-test")
	gcName := acctest.RandomWithPrefix("tf-acc-test-gc")
	resourceName := "kubernetes_grpc_route_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGRPCRouteV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGRPCRouteV1ConfigBasic(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckGRPCRouteV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", rName),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.backend_refs.#", "1"),
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

func testAccCheckGRPCRouteV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).GatewayClientset()
	if err != nil {
		return err
	}
	ctx := context.Background()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_grpc_route_v1" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.GRPCRoutes(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("GRPCRoute still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckGRPCRouteV1Exists(n string, obj *gatewayv1.GRPCRoute) resource.TestCheckFunc {
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

		out, err := conn.GRPCRoutes(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccGRPCRouteV1ConfigBasic(rName, gcName string) string {
	return fmt.Sprintf(`
resource "kubernetes_service_v1" "test" {
  metadata {
    name = "%[1]s-svc"
  }
  spec {
    selector = {
      app = "test"
    }
    port {
      port        = 50051
      target_port = 50051
    }
  }
}

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
      name     = "grpc"
      port     = 50051
      protocol = "HTTP"
    }
  }
}

resource "kubernetes_grpc_route_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    parent_refs {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }
    rules {
      backend_refs {
        name = kubernetes_service_v1.test.metadata.0.name
        port = 50051
      }
    }
  }
}
`, rName, gcName)
}

func TestAccKubernetesGRPCRouteV1_withMethodMatching(t *testing.T) {
	var conf gatewayv1.GRPCRoute
	rName := acctest.RandomWithPrefix("tf-acc-test")
	gcName := acctest.RandomWithPrefix("tf-acc-test-gc")
	resourceName := "kubernetes_grpc_route_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGRPCRouteV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGRPCRouteV1ConfigWithMethodMatching(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckGRPCRouteV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.matches.0.method.0.type", "Exact"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.matches.0.method.0.service", "com.example.FooService"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.matches.0.method.0.method", "GetFoo"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.1.matches.0.method.0.service", "com.example.FooService"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.1.matches.0.method.0.method", ""),
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

func TestAccKubernetesGRPCRouteV1_withHeaderMatching(t *testing.T) {
	var conf gatewayv1.GRPCRoute
	rName := acctest.RandomWithPrefix("tf-acc-test")
	gcName := acctest.RandomWithPrefix("tf-acc-test-gc")
	resourceName := "kubernetes_grpc_route_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGRPCRouteV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGRPCRouteV1ConfigWithHeaderMatching(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckGRPCRouteV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.matches.0.headers.0.name", "x-env"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.matches.0.headers.0.value", "staging"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.matches.0.headers.1.name", "x-version"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.matches.0.headers.1.value", "v2"),
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

func TestAccKubernetesGRPCRouteV1_withSessionPersistence(t *testing.T) {
	var conf gatewayv1.GRPCRoute
	rName := acctest.RandomWithPrefix("tf-acc-test")
	gcName := acctest.RandomWithPrefix("tf-acc-test-gc")
	resourceName := "kubernetes_grpc_route_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGRPCRouteV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGRPCRouteV1ConfigWithSessionPersistence(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckGRPCRouteV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.session_persistence.0.session_name", "grpc-sticky"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.session_persistence.0.type", "Cookie"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.session_persistence.0.absolute_timeout", "1800s"),
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

func TestAccKubernetesGRPCRouteV1_withWeightedBackends(t *testing.T) {
	var conf gatewayv1.GRPCRoute
	rName := acctest.RandomWithPrefix("tf-acc-test")
	gcName := acctest.RandomWithPrefix("tf-acc-test-gc")
	resourceName := "kubernetes_grpc_route_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGRPCRouteV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGRPCRouteV1ConfigWithWeightedBackends(rName, gcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckGRPCRouteV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.backend_refs.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.backend_refs.0.name", rName+"-svc-v1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.backend_refs.0.weight", "80"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.backend_refs.1.name", rName+"-svc-v2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rules.0.backend_refs.1.weight", "20"),
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

// --- config helpers for new GRPCRoute tests -----------------------------------

func testAccGRPCRouteV1BaseConfig(rName, gcName string) string {
	return fmt.Sprintf(`
resource "kubernetes_service_v1" "test" {
  metadata {
    name = "%[1]s-svc"
  }
  spec {
    selector = { app = "grpc-test" }
    port {
      port        = 50051
      target_port = 50051
    }
  }
}

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
      name     = "grpc"
      port     = 50051
      protocol = "HTTP"
    }
  }
}
`, rName, gcName)
}

func testAccGRPCRouteV1ConfigWithMethodMatching(rName, gcName string) string {
	return testAccGRPCRouteV1BaseConfig(rName, gcName) + fmt.Sprintf(`
resource "kubernetes_grpc_route_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    parent_refs {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }
    hostnames = ["grpc.example.com"]

    # Rule 0: match exact method
    rules {
      name = "get-foo"
      matches {
        method {
          type    = "Exact"
          service = "com.example.FooService"
          method  = "GetFoo"
        }
      }
      backend_refs {
        name = kubernetes_service_v1.test.metadata.0.name
        port = 50051
      }
    }

    # Rule 1: match entire service (all methods)
    rules {
      name = "foo-service"
      matches {
        method {
          type    = "Exact"
          service = "com.example.FooService"
        }
      }
      backend_refs {
        name = kubernetes_service_v1.test.metadata.0.name
        port = 50051
      }
    }
  }
}
`, rName)
}

func testAccGRPCRouteV1ConfigWithHeaderMatching(rName, gcName string) string {
	return testAccGRPCRouteV1BaseConfig(rName, gcName) + fmt.Sprintf(`
resource "kubernetes_grpc_route_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    parent_refs {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }
    rules {
      matches {
        headers {
          name  = "x-env"
          value = "staging"
          type  = "Exact"
        }
        headers {
          name  = "x-version"
          value = "v2"
          type  = "Exact"
        }
      }
      backend_refs {
        name = kubernetes_service_v1.test.metadata.0.name
        port = 50051
      }
    }
  }
}
`, rName)
}

func testAccGRPCRouteV1ConfigWithSessionPersistence(rName, gcName string) string {
	return testAccGRPCRouteV1BaseConfig(rName, gcName) + fmt.Sprintf(`
resource "kubernetes_grpc_route_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    parent_refs {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }
    rules {
      session_persistence {
        session_name     = "grpc-sticky"
        type             = "Cookie"
        absolute_timeout = "1800s"
      }
      backend_refs {
        name = kubernetes_service_v1.test.metadata.0.name
        port = 50051
      }
    }
  }
}
`, rName)
}

func testAccGRPCRouteV1ConfigWithWeightedBackends(rName, gcName string) string {
	return fmt.Sprintf(`
resource "kubernetes_service_v1" "v1" {
  metadata {
    name = "%[1]s-svc-v1"
  }
  spec {
    selector = { app = "grpc-test", version = "v1" }
    port {
      port        = 50051
      target_port = 50051
    }
  }
}

resource "kubernetes_service_v1" "v2" {
  metadata {
    name = "%[1]s-svc-v2"
  }
  spec {
    selector = { app = "grpc-test", version = "v2" }
    port {
      port        = 50051
      target_port = 50051
    }
  }
}

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
      name     = "grpc"
      port     = 50051
      protocol = "HTTP"
    }
  }
}

resource "kubernetes_grpc_route_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    parent_refs {
      name = kubernetes_gateway_v1.test.metadata.0.name
    }
    rules {
      backend_refs {
        name   = kubernetes_service_v1.v1.metadata.0.name
        port   = 50051
        weight = 80
      }
      backend_refs {
        name   = kubernetes_service_v1.v2.metadata.0.name
        port   = 50051
        weight = 20
      }
    }
  }
}
`, rName, gcName)
}
