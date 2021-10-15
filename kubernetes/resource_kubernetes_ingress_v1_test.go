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

func TestAccKubernetesIngressV1_basic(t *testing.T) {
	var conf networking.Ingress
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_ingress_v1.test",
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesIngressV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressV1Exists("kubernetes_ingress_v1.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_ingress_v1.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress_v1.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress_v1.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.ingress_class_name", "ingress-class"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.default_backend.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.default_backend.0.service.0.name", "app1"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.default_backend.0.service.0.port.0.number", "443"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.rule.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.rule.0.host", "server.domain.com"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.rule.0.http.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.rule.0.http.0.path.0.path", "/.*"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.rule.0.http.0.path.0.backend.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.rule.0.http.0.path.0.backend.0.service.0.name", "app2"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.rule.0.http.0.path.0.backend.0.service.0.port.0.number", "80"),
				),
			},
			{
				Config: testAccKubernetesIngressV1Config_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressV1Exists("kubernetes_ingress_v1.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_ingress_v1.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress_v1.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress_v1.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.ingress_class_name", "other-ingress-class"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.default_backend.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.default_backend.0.service.0.name", "svc"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.default_backend.0.service.0.port.0.number", "8443"),
				),
			},
		},
	})
}

func TestAccKubernetesIngressV1_TLS(t *testing.T) {
	var conf networking.Ingress
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_ingress_v1.test",
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesIngressV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressV1Config_TLS(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressV1Exists("kubernetes_ingress_v1.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_ingress_v1.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress_v1.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress_v1.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.tls.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.tls.0.hosts.0", "host1"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.tls.0.secret_name", "super-sekret"),
				),
			},
			{
				Config: testAccKubernetesIngressV1Config_TLS_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressV1Exists("kubernetes_ingress_v1.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_ingress_v1.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress_v1.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_ingress_v1.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.tls.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.tls.0.hosts.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.tls.0.hosts.1", "host2"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "spec.0.tls.0.secret_name", "super-sekret"),
				),
			},
		},
	})
}

func TestAccKubernetesIngressV1_InternalKey(t *testing.T) {
	var conf networking.Ingress
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_ingress_v1.test",
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesIngressV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressV1Config_internalKey(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressV1Exists("kubernetes_ingress_v1.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "metadata.0.annotations.kubernetes.io/ingress-anno", "one"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "metadata.0.labels.kubernetes.io/ingress-label", "one"),
				),
			},
			{
				Config: testAccKubernetesIngressV1Config_internalKey_removed(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressV1Exists("kubernetes_ingress_v1.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "metadata.0.name", name),
					resource.TestCheckNoResourceAttr("kubernetes_ingress_v1.test", "metadata.0.annotations.kubernetes.io/ingress-anno"),
					resource.TestCheckNoResourceAttr("kubernetes_ingress_v1.test", "metadata.0.labels.kubernetes.io/ingress-label"),
				),
			},
			{
				Config: testAccKubernetesIngressV1Config_internalKey(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressV1Exists("kubernetes_ingress_v1.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "metadata.0.annotations.kubernetes.io/ingress-anno", "one"),
					resource.TestCheckResourceAttr("kubernetes_ingress_v1.test", "metadata.0.labels.kubernetes.io/ingress-label", "one"),
				),
			},
		},
	})
}

func TestAccKubernetesIngressV1_WaitForLoadBalancerGoogleCloud(t *testing.T) {
	var conf networking.Ingress
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotRunningInGke(t) },
		IDRefreshName:     "kubernetes_ingress_v1.test",
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesIngressV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressV1Config_waitForLoadBalancer(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressV1Exists("kubernetes_ingress_v1.test", &conf),
					resource.TestCheckResourceAttrSet("kubernetes_ingress_v1.test", "status.0.load_balancer.0.ingress.0.ip"),
				),
			},
		},
	})
}

func testAccCheckKubernetesIngressV1ForceNew(old, new *networking.Ingress, wantNew bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if wantNew {
			if old.ObjectMeta.UID == new.ObjectMeta.UID {
				return fmt.Errorf("Expecting new resource for Ingress %s", old.ObjectMeta.UID)
			}
		} else {
			if old.ObjectMeta.UID != new.ObjectMeta.UID {
				return fmt.Errorf("Expecting Ingress UIDs to be the same: expected %s got %s", old.ObjectMeta.UID, new.ObjectMeta.UID)
			}
		}
		return nil
	}
}

func testAccCheckKubernetesIngressV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_ingress_v1" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.NetworkingV1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Ingress still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesIngressV1Exists(n string, obj *networking.Ingress) resource.TestCheckFunc {
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

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		out, err := conn.NetworkingV1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesIngressV1Config_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_ingress_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
	ingress_class_name = "ingress-class"
    default_backend {
	  service {
		name = "app1"
		port {
		  number = 443
		}
	  }
    }
    rule {
      host = "server.domain.com"
      http {
        path {
          backend {
            service {
			  name = "app2"
              port {
				number = 80
			  }
			}
          }
          path = "/.*"
        }
      }
    }
  }
}`, name)
}

func testAccKubernetesIngressV1Config_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_ingress_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
	ingress_class_name = "other-ingress-class"
    default_backend {
      service {
		name = "svc"
        port {
		  number = 8443
		}
	  }
    }
  }
}`, name)
}

func testAccKubernetesIngressV1Config_TLS(name string) string {
	return fmt.Sprintf(`resource "kubernetes_ingress_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    default_backend {
      service {
		name = "app1"
		port {
		  number = 443
		}
	  }
    }
    tls {
      hosts       = ["host1"]
      secret_name = "super-sekret"
    }
  }
}`, name)
}

func testAccKubernetesIngressV1Config_TLS_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_ingress_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    default_backend {
      service {
		name = "app1"
		port {
		  number = 443
		}
	  }
    }
    tls {
      hosts       = ["host1", "host2"]
      secret_name = "super-sekret"
    }
  }
}`, name)
}

func testAccKubernetesIngressV1Config_internalKey(name string) string {
	return fmt.Sprintf(`resource "kubernetes_ingress_v1" "test" {
  metadata {
    name = "%s"
    annotations = {
      "kubernetes.io/ingress-anno" = "one"
      TestAnnotationTwo = "two"
    }
    labels = {
      "kubernetes.io/ingress-label" = "one"
      TestLabelTwo = "two"
      TestLabelThree = "three"
    }
  }
  spec {
    default_backend {
      service {
		name = "app1"
		port {
		  number = 443
		}
	  }
    }
    tls {
      hosts       = ["host1", "host2"]
      secret_name = "super-sekret"
    }
  }
}`, name)
}

func testAccKubernetesIngressV1Config_internalKey_removed(name string) string {
	return fmt.Sprintf(`resource "kubernetes_ingress_v1" "test" {
  metadata {
    name = "%s"
    annotations = {
      TestAnnotationTwo = "two"
    }
    labels = {
      TestLabelTwo = "two"
      TestLabelThree = "three"
    }
  }
  spec {
    default_backend {
      service {
		name = "app1"
		port {
		  number = 443
		}
	  }
    }
    tls {
      hosts       = ["host1", "host2"]
      secret_name = "super-sekret"
    }
  }
}`, name)
}

func testAccKubernetesIngressV1Config_waitForLoadBalancer(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service" "test" {
  metadata {
    name = %q
  }
  spec {
    type = "NodePort"
    selector = {
      app = %q
    }
    port {
      port = 8000
      target_port = 80
      protocol = "TCP"
    }
  }
}

resource "kubernetes_deployment" "test" {
  metadata {
    name = %q
  }
  spec {
    selector {
      match_labels = {
        app = %q
      }
    }
    template {
      metadata {
        labels = {
          app = %q
        }
      }
      spec {
        container {
          name = "test"
          image = "gcr.io/google-samples/hello-app:2.0"
          env {
            name = "PORT"
            value = "80"
          }  
        }
      }
    }
  }
}

resource "kubernetes_ingress_v1" "test" {
  depends_on = [
    kubernetes_service.test, 
    kubernetes_deployment.test
  ]
  metadata {
    name = %q
  }
  spec {
    default_backend {
	  service {
		name = %q
		port {
		  number = 8000
		}
	  }
    }
  }
  wait_for_load_balancer = true
}`, name, name, name, name, name, name, name)
}
