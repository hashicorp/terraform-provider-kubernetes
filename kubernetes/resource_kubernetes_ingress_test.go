package kubernetes

import (
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	api "k8s.io/kubernetes/pkg/api/v1"
	kubernetes "k8s.io/kubernetes/pkg/client/clientset_generated/clientset"
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

func TestAccKubernetesIngress_loadBalancer(t *testing.T) {
	var conf api.Ingress
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t); skipIfNoLoadBalancersAvailable(t) },
		IDRefreshName: "kubernetes_ingress.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesIngressDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressConfig_loadBalancer(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressExists("kubernetes_ingress.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.port.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.port.#", "1"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress.test", "spec.0.port.0.node_port"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.port.0.port", "8888"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.port.0.protocol", "TCP"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.port.0.target_port", "80"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress.test", "spec.0.cluster_ip"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.external_ips.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.external_ips.1452553500", "10.0.0.4"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.external_ips.3371212991", "10.0.0.3"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.external_name", "ext-name-"+name),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.load_balancer_source_ranges.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.load_balancer_source_ranges.138364083", "10.0.0.5/32"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.load_balancer_source_ranges.445311837", "10.0.0.6/32"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.selector.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.selector.App", "MyApp"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.session_affinity", "ClientIP"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.type", "LoadBalancer"),
					testAccCheckIngressPorts(&conf, []api.IngressPort{
						{
							Port:       int32(8888),
							Protocol:   api.ProtocolTCP,
							TargetPort: intstr.FromInt(80),
						},
					}),
				),
			},
			{
				Config: testAccKubernetesIngressConfig_loadBalancer_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressExists("kubernetes_ingress.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.#", "1"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress.test", "spec.0.cluster_ip"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.external_ips.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.external_ips.1452553500", "10.0.0.4"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.external_ips.563283338", "10.0.0.5"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.external_name", "ext-name-modified-"+name),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.load_balancer_source_ranges.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.load_balancer_source_ranges.2271073252", "10.0.0.1/32"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.load_balancer_source_ranges.2515041290", "10.0.0.2/32"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.port.#", "1"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress.test", "spec.0.port.0.node_port"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.port.0.port", "9999"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.port.0.protocol", "TCP"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.port.0.target_port", "81"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.selector.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.selector.App", "MyModifiedApp"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.selector.NewSelector", "NewValue"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.session_affinity", "ClientIP"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.type", "LoadBalancer"),
					testAccCheckIngressPorts(&conf, []api.IngressPort{
						{
							Port:       int32(9999),
							Protocol:   api.ProtocolTCP,
							TargetPort: intstr.FromInt(81),
						},
					}),
				),
			},
		},
	})
}

func TestAccKubernetesIngress_nodePort(t *testing.T) {
	var conf api.Ingress
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_ingress.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesIngressDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressConfig_nodePort(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressExists("kubernetes_ingress.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.#", "1"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress.test", "spec.0.cluster_ip"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.external_ips.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.external_ips.1452553500", "10.0.0.4"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.external_ips.563283338", "10.0.0.5"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.external_name", "ext-name-"+name),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.load_balancer_ip", "12.0.0.125"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.port.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.port.0.name", "first"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress.test", "spec.0.port.0.node_port"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.port.0.port", "10222"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.port.0.protocol", "TCP"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.port.0.target_port", "22"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.port.1.name", "second"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress.test", "spec.0.port.1.node_port"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.port.1.port", "10333"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.port.1.protocol", "TCP"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.port.1.target_port", "33"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.selector.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.selector.App", "MyApp"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.session_affinity", "ClientIP"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.type", "NodePort"),
					testAccCheckIngressPorts(&conf, []api.IngressPort{
						{
							Name:       "first",
							Port:       int32(10222),
							Protocol:   api.ProtocolTCP,
							TargetPort: intstr.FromInt(22),
						},
						{
							Name:       "second",
							Port:       int32(10333),
							Protocol:   api.ProtocolTCP,
							TargetPort: intstr.FromInt(33),
						},
					}),
				),
			},
		},
	})
}

func TestAccKubernetesIngress_externalName(t *testing.T) {
	var conf api.Ingress
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_ingress.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesIngressDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressConfig_externalName(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressExists("kubernetes_ingress.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.cluster_ip", ""),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.external_ips.#", "0"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.external_name", "terraform.io"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.load_balancer_ip", ""),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.load_balancer_source_ranges.#", "0"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.port.#", "0"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.selector.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.session_affinity", "None"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.type", "ExternalName"),
				),
			},
		},
	})
}

func TestAccKubernetesIngress_importBasic(t *testing.T) {
	resourceName := "kubernetes_ingress.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesIngressDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressConfig_basic(name),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccKubernetesIngress_generatedName(t *testing.T) {
	var conf api.Ingress
	prefix := "tf-acc-test-gen-"

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_ingress.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesIngressDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressConfig_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressExists("kubernetes_ingress.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr("kubernetes_ingress.test", "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet("kubernetes_ingress.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress.test", "metadata.0.uid"),
				),
			},
		},
	})
}

func TestAccKubernetesIngress_importGeneratedName(t *testing.T) {
	resourceName := "kubernetes_ingress.test"
	prefix := "tf-acc-test-gen-import-"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesIngressDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressConfig_generatedName(prefix),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckIngressPorts(svc *api.Ingress, expected []api.IngressPort) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(expected) == 0 && len(svc.Spec.Ports) == 0 {
			return nil
		}

		ports := svc.Spec.Ports

		// Ignore NodePorts as these are assigned randomly
		for k, _ := range ports {
			ports[k].NodePort = 0
		}

		if !reflect.DeepEqual(ports, expected) {
			return fmt.Errorf("Ingress ports don't match.\nExpected: %#v\nGiven: %#v",
				expected, svc.Spec.Ports)
		}

		return nil
	}
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

		resp, err := conn.CoreV1().Ingresss(namespace).Get(name, meta_v1.GetOptions{})
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

		out, err := conn.CoreV1().Ingresss(namespace).Get(name, meta_v1.GetOptions{})
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
		port {
			port = 8080
			target_port = 80
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
