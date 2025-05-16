// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesIngressClassV1_basic(t *testing.T) {
	var conf networking.IngressClass
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_ingress_class_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesIngressClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressClassV1ConfigBasic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressClassV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", rName),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.controller", "example.com/ingress-controller"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parameters.#", "0"),
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
					"spec.0.parameters.0.scope",
				},
			},
		},
	})
}

func TestAccKubernetesIngressClassV1_parameters(t *testing.T) {
	var conf networking.IngressClass
	rName := acctest.RandomWithPrefix("tf-acc-test")
	rNameUpdated := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_ingress_class_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesIngressClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressClassV1ConfigParameters(rName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressClassV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", rName),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.controller", "example.com/ingress-controller"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parameters.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parameters.0.kind", "IngressParameters"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parameters.0.name", rName),
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
					"spec.0.parameters.0.scope",
				},
			},
			{
				Config: testAccKubernetesIngressClassV1ConfigParameters(rName, rNameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressClassV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", rName),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.controller", "example.com/ingress-controller"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parameters.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parameters.0.kind", "IngressParameters"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parameters.0.name", rNameUpdated),
				),
			},
		},
	})
}

func TestAccKubernetesIngressClassV1_parameters_apiGroup(t *testing.T) {
	var conf networking.IngressClass
	rName := acctest.RandomWithPrefix("tf-acc-test")
	rNameUpdated := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_ingress_class_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesIngressClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressClassV1ConfigParametersApiGroup(rName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressClassV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", rName),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.controller", "example.com/ingress-controller"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parameters.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parameters.0.kind", "IngressParameters"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parameters.0.name", rName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parameters.0.api_group", rName),
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
					"spec.0.parameters.0.scope",
				},
			},
			{
				Config: testAccKubernetesIngressClassV1ConfigParametersApiGroup(rName, rNameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressClassV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", rName),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.controller", "example.com/ingress-controller"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parameters.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parameters.0.kind", "IngressParameters"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parameters.0.name", rNameUpdated),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parameters.0.api_group", rNameUpdated),
				),
			},
		},
	})
}

func testAccCheckKubernetesIngressClassV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_ingress_class_v1" {
			continue
		}

		ctx := context.Background()
		name := rs.Primary.ID
		resp, err := conn.NetworkingV1().IngressClasses().Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Ingress still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesIngressClassV1Exists(n string, obj *networking.IngressClass) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
		if err != nil {
			return err
		}

		ctx := context.Background()
		name := rs.Primary.ID
		out, err := conn.NetworkingV1().IngressClasses().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesIngressClassV1ConfigBasic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_ingress_class_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    controller = "example.com/ingress-controller"
  }
}
`, name)
}

func testAccKubernetesIngressClassV1ConfigParameters(name, paramName string) string {
	return fmt.Sprintf(`resource "kubernetes_ingress_class_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    controller = "example.com/ingress-controller"
    parameters {
      kind = "IngressParameters"
      name = %[2]q
    }
  }
}
`, name, paramName)
}

func testAccKubernetesIngressClassV1ConfigParametersApiGroup(name, paramName string) string {
	return fmt.Sprintf(`resource "kubernetes_ingress_class_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    controller = "example.com/ingress-controller"
    parameters {
      api_group = %[2]q
      kind      = "IngressParameters"
      name      = %[2]q
    }
  }
}
`, name, paramName)
}
