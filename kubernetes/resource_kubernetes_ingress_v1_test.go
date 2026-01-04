// Copyright IBM Corp. 2017, 2025
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

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccKubernetesIngressV1_serviceBackend(t *testing.T) {
	var conf networking.Ingress
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_ingress_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.22.0")
		},

		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesIngressV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressV1Config_serviceBackend(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress_class_name", "ingress-class"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.default_backend.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.default_backend.0.service.0.name", "app1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.default_backend.0.service.0.port.0.number", "443"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.host", "server.domain.com"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.0.path.0.path", "/.*"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.0.path.0.backend.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.0.path.0.backend.0.service.0.name", "app2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.0.path.0.backend.0.service.0.port.0.name", "http"),
				),
			},
			{
				Config: testAccKubernetesIngressV1Config_serviceBackend_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress_class_name", "other-ingress-class"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.default_backend.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.default_backend.0.service.0.name", "svc"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.default_backend.0.service.0.port.0.number", "8443"),
				),
			},
		},
	})
}

func TestAccKubernetesIngressV1_resourceBackend(t *testing.T) {
	var conf networking.Ingress
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_ingress_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.22.0")
		},

		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesIngressV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressV1Config_resourceBackend(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress_class_name", "ingress-class"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.default_backend.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.default_backend.0.resource.0.api_group", "k8s.example.com"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.default_backend.0.resource.0.kind", "StorageBucket"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.default_backend.0.resource.0.name", "static-assets"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.host", "server.domain.com"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.0.path.0.path", "/icons"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.0.path.0.path_type", "ImplementationSpecific"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.0.path.0.backend.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.0.path.0.backend.0.resource.0.api_group", "k8s.example.com"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.0.path.0.backend.0.resource.0.kind", "StorageBucket"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.0.path.0.backend.0.resource.0.name", "icon-assets"),
				),
			},
		},
	})
}

func TestAccKubernetesIngressV1_TLS(t *testing.T) {
	var conf networking.Ingress
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_ingress_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.22.0")
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesIngressV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressV1Config_TLS(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressV1Exists(resourceName, &conf),
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
				Config: testAccKubernetesIngressV1Config_TLS_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressV1Exists(resourceName, &conf),
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

func TestAccKubernetesIngressV1_emptyTLS(t *testing.T) {
	var conf networking.Ingress
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_ingress_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.22.0")
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesIngressV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressV1Config_emptyTLS(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressV1Exists(resourceName, &conf),
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

func TestAccKubernetesIngressV1_InternalKey(t *testing.T) {
	var conf networking.Ingress
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_ingress_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.22.0")
		},

		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesIngressV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressV1Config_internalKey(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.kubernetes.io/ingress-anno", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.kubernetes.io/ingress-label", "one"),
				),
			},
			{
				Config: testAccKubernetesIngressV1Config_internalKey_removed(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckNoResourceAttr(resourceName, "metadata.0.annotations.kubernetes.io/ingress-anno"),
					resource.TestCheckNoResourceAttr(resourceName, "metadata.0.labels.kubernetes.io/ingress-label"),
				),
			},
			{
				Config: testAccKubernetesIngressV1Config_internalKey(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.kubernetes.io/ingress-anno", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.kubernetes.io/ingress-label", "one"),
				),
			},
		},
	})
}

func TestAccKubernetesIngressV1_WaitForLoadBalancerGoogleCloud(t *testing.T) {
	var conf networking.Ingress
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_ingress_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.22.0")
			skipIfNotRunningInGke(t)
		},

		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesIngressV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressV1Config_waitForLoadBalancer(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttrSet(resourceName, "status.0.load_balancer.0.ingress.0.ip"),
				),
			},
		},
	})
}

func TestAccKubernetesIngressV1_hostOnlyRule(t *testing.T) {
	var conf networking.Ingress
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_ingress_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.22.0")
		},

		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesIngressV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressV1Config_ruleHostOnly(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress_class_name", "ingress-class"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.default_backend.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.host", "server.domain.com"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.#", "0"),
				),
			},
		},
	})
}

func TestAccKubernetesIngressV1_multipleRulesDifferentHosts(t *testing.T) {
	var conf networking.Ingress
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_ingress_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.22.0")
		},

		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesIngressV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressV1Config_multipleRulesDifferentHosts(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress_class_name", "ingress-class"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.default_backend.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.host", "server.domain.com"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.0.path.0.path", "/app1/*"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.0.path.0.backend.0.service.0.name", "app1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.0.http.0.path.0.backend.0.service.0.port.0.number", "8080"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.1.http.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.1.http.0.path.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.1.host", "server.example.com"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.1.http.0.path.0.path", "/app1/*"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.1.http.0.path.0.backend.0.service.0.name", "app1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.1.http.0.path.0.backend.0.service.0.port.0.number", "8080"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.1.host", "server.example.com"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.1.http.0.path.1.path", "/app2/*"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.1.http.0.path.1.backend.0.service.0.name", "app2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.rule.1.http.0.path.1.backend.0.service.0.port.0.number", "8080"),
				),
			},
		},
	})
}

func TestAccKubernetesIngressV1_defaultIngressClass(t *testing.T) {
	var conf networking.Ingress
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	ingressClass := "default-ingress-class"
	resourceName := "kubernetes_ingress_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.22.0")
		},

		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesIngressV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressV1Config_defaultIngressClass(ingressClass, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesIngressV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ingress_class_name", "default-ingress-class"),
				),
			},
		},
	})
}

func TestAccKubernetesIngressV1_identity(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	ingressClass := "identity-ingress-class"
	resourceName := "kubernetes_ingress_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.22.0")
		},
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesIngressV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIngressV1Config_identity(ingressClass, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(
						resourceName, map[string]knownvalue.Check{
							"namespace":   knownvalue.StringExact("default"),
							"name":        knownvalue.StringExact(name),
							"api_version": knownvalue.StringExact("networking.k8s.io/v1"),
							"kind":        knownvalue.StringExact("Ingress"),
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

func testAccKubernetesIngressV1Config_serviceBackend(name string) string {
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
                name = "http"
              }
            }
          }
          path = "/.*"
        }
      }
    }
  }
  timeouts {
    create = "45m"
  }
}`, name)
}

func testAccKubernetesIngressV1Config_resourceBackend(name string) string {
	return fmt.Sprintf(`resource "kubernetes_ingress_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    ingress_class_name = "ingress-class"
    default_backend {
      resource {
        api_group = "k8s.example.com"
        kind      = "StorageBucket"
        name      = "static-assets"
      }
    }
    rule {
      host = "server.domain.com"
      http {
        path {
          path      = "/icons"
          path_type = "ImplementationSpecific"
          backend {
            resource {
              api_group = "k8s.example.com"
              kind      = "StorageBucket"
              name      = "icon-assets"
            }
          }
        }
      }
    }
  }
}`, name)
}

func testAccKubernetesIngressV1Config_serviceBackend_modified(name string) string {
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

func testAccKubernetesIngressV1Config_emptyTLS(name string) string {
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
      TestAnnotationTwo            = "two"
    }
    labels = {
      "kubernetes.io/ingress-label" = "one"
      TestLabelTwo                  = "two"
      TestLabelThree                = "three"
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
      TestLabelTwo   = "two"
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
      target_port = 8080
      protocol    = "TCP"
    }
  }
  lifecycle {
    ignore_changes = [
      metadata[0].annotations["cloud.google.com/neg"],
      metadata[0].annotations["cloud.google.com/neg-status"],
    ]
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
            value = "8080"
          }
        }
      }
    }
  }
}

resource "kubernetes_ingress_v1" "test" {
  depends_on = [
    kubernetes_service_v1.test,
    kubernetes_deployment_v1.test
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
  timeouts {
    create = "45m"
  }
}`, name, name, name, name, name, name, name)
}

func testAccKubernetesIngressV1Config_ruleHostOnly(name string) string {
	return fmt.Sprintf(`resource "kubernetes_ingress_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    ingress_class_name = "ingress-class"
    rule {
      host = "server.domain.com"
    }
  }
}`, name)
}

func testAccKubernetesIngressV1Config_multipleRulesDifferentHosts(name string) string {
	return fmt.Sprintf(`resource "kubernetes_ingress_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    ingress_class_name = "ingress-class"
    rule {
      host = "server.domain.com"
      http {
        path {
          backend {
            service {
              name = "app1"
              port {
                number = 8080
              }
            }
          }
          path = "/app1/*"
        }
      }
    }
    rule {
      host = "server.example.com"
      http {
        path {
          backend {
            service {
              name = "app1"
              port {
                number = 8080
              }
            }
          }
          path = "/app1/*"
        }
        path {
          backend {
            service {
              name = "app2"
              port {
                number = 8080
              }
            }
          }
          path = "/app2/*"
        }
      }
    }
  }
}`, name)
}

func testAccKubernetesIngressV1Config_defaultIngressClass(ingressClass, name string) string {
	return fmt.Sprintf(`resource "kubernetes_ingress_class_v1" "test" {
  metadata {
    name = "%s"
    labels = {
      "app.kubernetes.io/component" = "controller"
    }
    annotations = {
      "ingressclass.kubernetes.io/is-default-class" = "true"
    }
  }
  spec {
    controller = "k8s.io/ingress-nginx"
  }
}

resource "kubernetes_ingress_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    rule {
      host = "server.domain.com"
      http {
        path {
          backend {
            service {
              name = "app1"
              port {
                number = 8080
              }
            }
          }
          path = "/app1/*"
        }
      }
    }
  }
  depends_on = ["kubernetes_ingress_class_v1.test"]
}`, ingressClass, name)
}

func testAccKubernetesIngressV1Config_identity(ingressClass, name string) string {
	return fmt.Sprintf(`resource "kubernetes_ingress_class_v1" "test" {
  metadata {
    name = "%s"
    labels = {
      "app.kubernetes.io/component" = "controller"
    }
    annotations = {
      "ingressclass.kubernetes.io/is-default-class" = "true"
    }
  }
  spec {
    controller = "k8s.io/ingress-nginx"
  }
}

resource "kubernetes_ingress_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    rule {
      host = "server.domain.com"
      http {
        path {
          backend {
            service {
              name = "app1"
              port {
                number = 8080
              }
            }
          }
          path = "/app1/*"
        }
      }
    }
  }
  depends_on = ["kubernetes_ingress_class_v1.test"]
}`, ingressClass, name)
}
