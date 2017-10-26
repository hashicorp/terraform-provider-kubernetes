package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetes "k8s.io/client-go/kubernetes"
	// api "k8s.io/client-go/pkg/api/v1"
	api "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

func TestAccKubernetesIngress_basic(t *testing.T) {
	var conf api.Ingress
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_ingress.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesIngressDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressExists("kubernetes_ingress.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_ingress.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.backend.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.backend.0.service_name", ""),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.backend.0.service_port", "0"),
					// testAccCheckIngressPorts(&conf, []api.IngressPort{
					// 	{
					// 		Port:       int32(8080),
					// 		Protocol:   api.ProtocolTCP,
					// 		TargetPort: intstr.FromInt(80),
					// 	},
					// }),
				),
			},
			{
				Config: testAccKubernetesIngressConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressExists("kubernetes_ingress.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_ingress.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.backend.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.backend.0.service_name", "https"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.backend.0.service_port", "443"),
					// testAccCheckIngressPorts(&conf, []api.IngressPort{
					// 	{
					// 		Port:       int32(443),
					// 		Protocol:   api.ProtocolTCP,
					// 		TargetPort: intstr.FromInt(80),
					// 	},
					// }),
				),
			},
		},
	})
}

func testAccCheckKubernetesIngressDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*kubernetes.Clientset)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_ingress" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.ExtensionsV1beta1().Ingresses(namespace).Get(name, meta_v1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Ingress still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesIngressExists(n string, obj *api.Ingress) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*kubernetes.Clientset)

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		out, err := conn.ExtensionsV1beta1().Ingresses(namespace).Get(name, meta_v1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesIngressConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_ingress" "test" {
	metadata {
		annotations {
			TestAnnotationOne = "one"
			TestAnnotationTwo = "two"
		}
		labels {
			TestLabelOne = "one"
			TestLabelTwo = "two"
			TestLabelThree = "three"
		}
		name = "%s"
	}
	spec {
		backend {
			service_name = "app1"
			service_port = 443
		}
	}
}`, name)
}

func testAccKubernetesIngressConfig_modified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_ingress" "test" {
	metadata {
		annotations {
			TestAnnotationOne = "one"
			Different = "1234"
		}
		labels {
			TestLabelOne = "one"
			TestLabelThree = "three"
		}
		name = "%s"
	}
	spec {
		port {
			port = 8081
			target_port = 80
		}
	}
}`, name)
}

func testAccKubernetesIngressConfig_loadBalancer(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_ingress" "test" {
	metadata {
		name = "%s"
	}
	spec {
		external_name = "ext-name-%s"
		external_ips = ["10.0.0.3", "10.0.0.4"]
		load_balancer_source_ranges = ["10.0.0.5/32", "10.0.0.6/32"]
		selector {
			App = "MyApp"
		}
		session_affinity = "ClientIP"
		port {
			port = 8888
			target_port = 80
		}
		type = "LoadBalancer"
	}
}`, name, name)
}

func testAccKubernetesIngressConfig_loadBalancer_modified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_ingress" "test" {
	metadata {
		name = "%s"
	}
	spec {
		external_name = "ext-name-modified-%s"
		external_ips = ["10.0.0.4", "10.0.0.5"]
		load_balancer_source_ranges = ["10.0.0.1/32", "10.0.0.2/32"]
		selector {
			App = "MyModifiedApp"
			NewSelector = "NewValue"
		}
		session_affinity = "ClientIP"
		port {
			port = 9999
			target_port = 81
		}
		type = "LoadBalancer"
	}
}`, name, name)
}

func testAccKubernetesIngressConfig_nodePort(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_ingress" "test" {
	metadata {
		name = "%s"
	}
	spec {
		external_name = "ext-name-%s"
		external_ips = ["10.0.0.4", "10.0.0.5"]
		load_balancer_ip = "12.0.0.125"
		selector {
			App = "MyApp"
		}
		session_affinity = "ClientIP"
		port {
			name = "first"
			port = 10222
			target_port = 22
		}
		port {
			name = "second"
			port = 10333
			target_port = 33
		}
		type = "NodePort"
	}
}`, name, name)
}

func testAccKubernetesIngressConfig_externalName(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_ingress" "test" {
	metadata {
		name = "%s"
	}
	spec {
		type = "ExternalName"
		external_name = "terraform.io"
	}
}
`, name)
}

func testAccKubernetesIngressConfig_generatedName(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_ingress" "test" {
	metadata {
		generate_name = "%s"
	}
	spec {
		port {
			port = 8080
			target_port = 80
		}
	}
}`, prefix)
}
