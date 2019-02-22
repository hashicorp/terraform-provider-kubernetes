package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	api "k8s.io/api/extensions/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.backend.0.service_name", "app1"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.backend.0.service_port", "443"),
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
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.backend.0.service_name", "svc"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.backend.0.service_port", "8443"),
				),
			},
		},
	})
}

func TestAccKubernetesIngress_TLS(t *testing.T) {
	var conf api.Ingress
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_ingress.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesIngressDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressConfig_TLS(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressExists("kubernetes_ingress.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_ingress.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.tls.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.tls.0.hosts.0", "host1"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.tls.0.secret_name", "super-sekret"),
				),
			},
			{
				Config: testAccKubernetesIngressConfig_TLS_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressExists("kubernetes_ingress.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_ingress.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.tls.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.tls.0.hosts.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.tls.0.hosts.1", "host2"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "spec.0.tls.0.secret_name", "super-sekret"),
				),
			},
		},
	})
}

func TestAccKubernetesIngress_InternalKey(t *testing.T) {
	var conf api.Ingress
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_ingress.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesIngressDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressConfig_internalKey(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressExists("kubernetes_ingress.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "metadata.0.annotations.kubernetes.io/ingress-anno", "one"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "metadata.0.labels.kubernetes.io/ingress-label", "one"),
				),
			},
			{
				Config: testAccKubernetesIngressConfig_internalKey_removed(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressExists("kubernetes_ingress.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "metadata.0.name", name),
					resource.TestCheckNoResourceAttr("kubernetes_ingress.test", "metadata.0.annotations.kubernetes.io/ingress-anno"),
					resource.TestCheckNoResourceAttr("kubernetes_ingress.test", "metadata.0.labels.kubernetes.io/ingress-label"),
				),
			},
			{
				Config: testAccKubernetesIngressConfig_internalKey(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressExists("kubernetes_ingress.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "metadata.0.annotations.kubernetes.io/ingress-anno", "one"),
					resource.TestCheckResourceAttr("kubernetes_ingress.test", "metadata.0.labels.kubernetes.io/ingress-label", "one"),
				),
			},
		},
	})
}

func testAccCheckKubernetesIngressDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*kubernetesProvider).conn

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

		conn := testAccProvider.Meta().(*kubernetesProvider).conn

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
		name = "%s"
	}
	spec {
		backend {
			service_name = "app1"
			service_port = 443
		}
		rule {
			host = "server.domain.com"
			http {
				path {
					backend {
						service_name = "app2"
						service_port = 80
					}
					path_regex = "/.*"
				}
			}
		}
	}
}`, name)
}

func testAccKubernetesIngressConfig_modified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_ingress" "test" {
	metadata {
		name = "%s"
	}
	spec {
		backend {
			service_name = "svc"
			service_port = 8443
		}
	}
}`, name)
}

func testAccKubernetesIngressConfig_TLS(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_ingress" "test" {
	metadata {
		name = "%s"
	}
	spec {
		backend {
			service_name = "app1"
			service_port = 443
		}
		tls {
			hosts       = ["host1"]
			secret_name = "super-sekret"
		}
	}
}`, name)
}

func testAccKubernetesIngressConfig_TLS_modified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_ingress" "test" {
	metadata {
		name = "%s"
	}
	spec {
		backend {
			service_name = "app1"
			service_port = 443
		}
		tls {
			hosts       = ["host1", "host2"]
			secret_name = "super-sekret"
		}
	}
}`, name)
}

func testAccKubernetesIngressConfig_internalKey(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_ingress" "test" {
	metadata {
		name = "%s"
		
		annotations {
			"kubernetes.io/ingress-anno" = "one"
			TestAnnotationTwo = "two"
		}
		labels {
			"kubernetes.io/ingress-label" = "one"
			TestLabelTwo = "two"
			TestLabelThree = "three"
		}
	}
	spec {
		backend {
			service_name = "app1"
			service_port = 443
		}
		tls {
			hosts       = ["host1", "host2"]
			secret_name = "super-sekret"
		}
	}
}`, name)
}

func testAccKubernetesIngressConfig_internalKey_removed(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_ingress" "test" {
	metadata {
		name = "%s"
		
		annotations {
			TestAnnotationTwo = "two"
		}
		labels {
			TestLabelTwo = "two"
			TestLabelThree = "three"
		}
	}
	spec {
		backend {
			service_name = "app1"
			service_port = 443
		}
		tls {
			hosts       = ["host1", "host2"]
			secret_name = "super-sekret"
		}
	}
}`, name)
}
