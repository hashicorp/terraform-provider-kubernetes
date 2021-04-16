package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesIngressClass_basic(t *testing.T) {
	var conf networking.IngressClass
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_ingress_class.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesIngressClassDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressClassConfigBasic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressClassExists(resourceName, &conf),
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
			},
		},
	})
}

func TestAccKubernetesIngressClass_parameters(t *testing.T) {
	var conf networking.IngressClass
	rName := acctest.RandomWithPrefix("tf-acc-test")
	rNameUpdated := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_ingress_class.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesIngressClassDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressClassConfigParameters(rName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressClassExists(resourceName, &conf),
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
			},
			{
				Config: testAccKubernetesIngressClassConfigParameters(rName, rNameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressClassExists(resourceName, &conf),
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

func TestAccKubernetesIngressClass_parameters_apiGroup(t *testing.T) {
	var conf networking.IngressClass
	rName := acctest.RandomWithPrefix("tf-acc-test")
	rNameUpdated := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_ingress_class.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesIngressClassDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressClassConfigParametersApiGroup(rName, rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressClassExists(resourceName, &conf),
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
			},
			{
				Config: testAccKubernetesIngressClassConfigParametersApiGroup(rName, rNameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressClassExists(resourceName, &conf),
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

func testAccCheckKubernetesIngressClassDestroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_ingress_class" {
			continue
		}

		_, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.NetworkingV1().IngressClasses().Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Ingress still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesIngressClassExists(n string, obj *networking.IngressClass) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
		if err != nil {
			return err
		}
		ctx := context.TODO()

		_, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		out, err := conn.NetworkingV1().IngressClasses().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesIngressClassConfigBasic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_ingress_class" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    controller = "example.com/ingress-controller"
  }
}
`, name)
}

func testAccKubernetesIngressClassConfigParameters(name, paramName string) string {
	return fmt.Sprintf(`
resource "kubernetes_ingress_class" "test" {
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

func testAccKubernetesIngressClassConfigParametersApiGroup(name, paramName string) string {
	return fmt.Sprintf(`
resource "kubernetes_ingress_class" "test" {
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
