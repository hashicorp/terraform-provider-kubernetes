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

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccGatewayClassV1_basic(t *testing.T) {
	var conf gatewayv1.GatewayClass
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_gateway_class_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGatewayClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayClassV1ConfigBasic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckGatewayClassV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", rName),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.controller_name", "example.com/gateway-controller"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.description", ""),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parameters_ref.#", "0"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"metadata.0.resource_version",
					"metadata.0.uid",
					"metadata.0.generation",
					"status",
				},
			},
		},
	})
}

func TestAccGatewayClassV1_description(t *testing.T) {
	var conf gatewayv1.GatewayClass
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_gateway_class_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGatewayClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayClassV1ConfigDescription(rName, "Initial description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckGatewayClassV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", rName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.controller_name", "example.com/gateway-controller"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.description", "Initial description"),
				),
			},
			{
				Config: testAccGatewayClassV1ConfigDescription(rName, "Updated description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckGatewayClassV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", rName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.controller_name", "example.com/gateway-controller"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.description", "Updated description"),
				),
			},
		},
	})
}

func TestAccGatewayClassV1_identity(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_gateway_class_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGatewayClassV1Destroy,

		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayClassV1ConfigBasic(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(
						resourceName, map[string]knownvalue.Check{
							"name":        knownvalue.StringExact(name),
							"api_version": knownvalue.StringExact("gateway.networking.k8s.io/v1"),
							"kind":        knownvalue.StringExact("GatewayClass"),
						},
					),
				},
			},
			{
				ResourceName:    resourceName,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func TestAccGatewayClassV1_parametersRef(t *testing.T) {
	var conf gatewayv1.GatewayClass
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_gateway_class_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGatewayClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayClassV1ConfigParametersRef(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckGatewayClassV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", rName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.controller_name", "example.com/gateway-controller"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parameters_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parameters_ref.0.group", "example.com"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parameters_ref.0.kind", "GatewayParameters"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parameters_ref.0.name", "test-params"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"metadata.0.resource_version",
					"metadata.0.uid",
					"metadata.0.generation",
					"status",
				},
			},
		},
	})
}

func TestAccGatewayClassV1_parametersRefWithNamespace(t *testing.T) {
	var conf gatewayv1.GatewayClass
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_gateway_class_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGatewayClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayClassV1ConfigParametersRefNamespace(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckGatewayClassV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", rName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.controller_name", "example.com/gateway-controller"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parameters_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parameters_ref.0.group", ""),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parameters_ref.0.kind", "ConfigMap"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parameters_ref.0.name", "gateway-config"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parameters_ref.0.namespace", "gateway-system"),
				),
			},
		},
	})
}

func testAccCheckGatewayClassV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).GatewayClientset()
	if err != nil {
		return err
	}
	ctx := context.Background()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_gateway_class_v1" {
			continue
		}

		name := rs.Primary.ID
		resp, err := conn.GatewayClasses().Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("GatewayClass still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckGatewayClassV1Exists(n string, obj *gatewayv1.GatewayClass) resource.TestCheckFunc {
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
		name := rs.Primary.ID
		out, err := conn.GatewayClasses().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccGatewayClassV1ConfigBasic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    controller_name = "example.com/gateway-controller"
  }
}
`, name)
}

func testAccGatewayClassV1ConfigDescription(name, description string) string {
	return fmt.Sprintf(`resource "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    controller_name = "example.com/gateway-controller"
    description     = %[2]q
  }
}
`, name, description)
}

func testAccGatewayClassV1ConfigParametersRef(name string) string {
	return fmt.Sprintf(`resource "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    controller_name = "example.com/gateway-controller"
    parameters_ref {
      group = "example.com"
      kind  = "GatewayParameters"
      name  = "test-params"
    }
  }
}
`, name)
}

func testAccGatewayClassV1ConfigParametersRefNamespace(name string) string {
	return fmt.Sprintf(`resource "kubernetes_gateway_class_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    controller_name = "example.com/gateway-controller"
    parameters_ref {
      group     = ""
      kind      = "ConfigMap"
      name      = "gateway-config"
      namespace = "gateway-system"
    }
  }
}
`, name)
}
