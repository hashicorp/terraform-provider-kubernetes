package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	networking "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesIngressClass_basic(t *testing.T) {
	var conf networking.IngressClass
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_ingress_class.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesIngressClassDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressClassConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressClassExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.controller", "example.com/ingress-controller"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parameters.#", "0"),
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

		resp, err := conn.NetworkingV1beta1().IngressClasses().Get(ctx, name, metav1.GetOptions{})
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

		out, err := conn.NetworkingV1beta1().IngressClasses().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesIngressClassConfig_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_ingress_class" "test" {
  metadata {
    name = "%s"
  }
  spec {
    controller = "example.com/ingress-controller"
	

  }
}`, name)
}
