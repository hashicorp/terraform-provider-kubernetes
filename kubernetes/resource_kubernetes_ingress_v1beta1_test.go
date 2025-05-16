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
	api "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesIngressV1Beta1_basic(t *testing.T) {
	var conf api.Ingress
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_ingress.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionGreaterThanOrEqual(t, "1.22.0")
		},

		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesIngressV1Beta1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressV1Beta1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressV1Beta1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress_class_name", "ingress-class"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend.0.service_name", "app1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend.0.service_port", "443"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.host", "server.domain.com"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.0.path.0.path", "/.*"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.0.path.0.backend.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.0.path.0.backend.0.service_name", "app2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.0.path.0.backend.0.service_port", "80"),
				),
			},
			{
				Config: testAccKubernetesIngressV1Beta1Config_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressV1Beta1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress_class_name", "other-ingress-class"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend.0.service_name", "svc"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backend.0.service_port", "8443"),
				),
			},
		},
	})
}

func TestAccKubernetesIngressV1Beta1_TLS(t *testing.T) {
	var conf api.Ingress
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_ingress.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionGreaterThanOrEqual(t, "1.22.0")
		},

		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesIngressV1Beta1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressV1Beta1Config_TLS(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressV1Beta1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.tls.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.tls.0.hosts.0", "host1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.tls.0.secret_name", "super-sekret"),
				),
			},
			{
				Config: testAccKubernetesIngressV1Beta1Config_TLS_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressV1Beta1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.tls.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.tls.0.hosts.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.tls.0.hosts.1", "host2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.tls.0.secret_name", "super-sekret"),
				),
			},
		},
	})
}

func TestAccKubernetesIngressV1Beta1_emptyTLS(t *testing.T) {
	var conf api.Ingress
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_ingress.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionGreaterThanOrEqual(t, "1.22.0")
		},

		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesIngressV1Beta1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressV1Beta1Config_TLS(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressV1Beta1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.tls.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.tls.0.hosts.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.tls.0.secret_name", ""),
				),
			},
		},
	})
}

func TestAccKubernetesIngressV1Beta1_InternalKey(t *testing.T) {
	var conf api.Ingress
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_ingress.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionGreaterThanOrEqual(t, "1.22.0")
		},

		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesIngressV1Beta1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressV1Beta1Config_internalKey(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressV1Beta1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.kubernetes.io/ingress-anno", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.kubernetes.io/ingress-label", "one"),
				),
			},
			{
				Config: testAccKubernetesIngressV1Beta1Config_internalKey_removed(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressV1Beta1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckNoResourceAttr(resourceName, "metadata.0.annotations.kubernetes.io/ingress-anno"),
					resource.TestCheckNoResourceAttr(resourceName, "metadata.0.labels.kubernetes.io/ingress-label"),
				),
			},
			{
				Config: testAccKubernetesIngressV1Beta1Config_internalKey(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressV1Beta1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.kubernetes.io/ingress-anno", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.kubernetes.io/ingress-label", "one"),
				),
			},
		},
	})
}

func TestAccKubernetesIngressV1Beta1_WaitForLoadBalancerGoogleCloud(t *testing.T) {
	var conf api.Ingress
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_ingress.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionGreaterThanOrEqual(t, "1.22.0")
			skipIfNotRunningInGke(t)
		},

		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesIngressV1Beta1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressV1Beta1Config_waitForLoadBalancer(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressV1Beta1Exists(resourceName, &conf),
					resource.TestCheckResourceAttrSet(resourceName, "status.0.load_balancer.0.ingress.0.ip"),
				),
			},
		},
	})
}

func testAccCheckKubernetesIngressV1Beta1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_ingress" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.ExtensionsV1beta1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Ingress still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesIngressV1Beta1Exists(n string, obj *api.Ingress) resource.TestCheckFunc {
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

		out, err := conn.ExtensionsV1beta1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesIngressV1Beta1Config_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_ingress" "test" {
  metadata {
    name = "%s"
  }
  spec {
    ingress_class_name = "ingress-class"
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
          path = "/.*"
        }
      }
    }
  }
}`, name)
}

func testAccKubernetesIngressV1Beta1Config_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_ingress" "test" {
  metadata {
    name = "%s"
  }
  spec {
    ingress_class_name = "other-ingress-class"
    backend {
      service_name = "svc"
      service_port = 8443
    }
  }
}`, name)
}

func testAccKubernetesIngressV1Beta1Config_TLS(name string) string {
	return fmt.Sprintf(`resource "kubernetes_ingress" "test" {
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

func testAccKubernetesIngressV1Beta1Config_emptyTLS(name string) string {
	return fmt.Sprintf(`resource "kubernetes_ingress" "test" {
  metadata {
    name = "%s"
  }
  spec {
    backend {
      service_name = "app1"
      service_port = 443
    }
    tls {
    }
  }
}`, name)
}

func testAccKubernetesIngressV1Beta1Config_TLS_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_ingress" "test" {
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

func testAccKubernetesIngressV1Beta1Config_internalKey(name string) string {
	return fmt.Sprintf(`resource "kubernetes_ingress" "test" {
  metadata {
    name = "%s"
    annotations = {
      "kubernetes.io/ingress-anno" = "one"
      TestAnnotationTwo            = "two"
    }
    labels = {
      "kubernetes.io/ingress-label" = "one"
      TestLabelTwo                  = "two"
      TestLabelThree                = "three"
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

func testAccKubernetesIngressV1Beta1Config_internalKey_removed(name string) string {
	return fmt.Sprintf(`resource "kubernetes_ingress" "test" {
  metadata {
    name = "%s"
    annotations = {
      TestAnnotationTwo = "two"
    }
    labels = {
      TestLabelTwo   = "two"
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

func testAccKubernetesIngressV1Beta1Config_waitForLoadBalancer(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service_v1" "test" {
  metadata {
    name = %q
  }
  spec {
    type = "NodePort"
    selector = {
      app = %q
    }
    port {
      port        = 8000
      target_port = 80
      protocol    = "TCP"
    }
  }
}

resource "kubernetes_deployment_v1" "test" {
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
          name  = "test"
          image = "gcr.io/google-samples/hello-app:2.0"
          env {
            name  = "PORT"
            value = "80"
          }
        }
      }
    }
  }
}

resource "kubernetes_ingress" "test" {
  depends_on = [
    kubernetes_service_v1.test,
    kubernetes_deployment_v1.test
  ]
  metadata {
    name = %q
  }
  spec {
    backend {
      service_name = %q
      service_port = 8000
    }
  }
  wait_for_load_balancer = true
}`, name, name, name, name, name, name, name)
}
