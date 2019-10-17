package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesAPIService_basic(t *testing.T) {
	group := fmt.Sprintf("tf-acc-test-%s.k8s.io", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	version := "v1beta1"
	name := fmt.Sprintf("%s.%s", version, group)

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_api_service.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesAPIServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesAPIServiceConfig_basic(name, group, version),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesAPIServiceExists("kubernetes_api_service.test"),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_api_service.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_api_service.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_api_service.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_api_service.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.service.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.service.0.name", "metrics-server"),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.service.0.namespace", "kube-system"),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.group", group),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.group_priority_minimum", "1"),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.version", version),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.version_priority", "1"),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.ca_bundle", ""),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.insecure_skip_tls_verify", "true"),
				),
			},
			{
				Config: testAccKubernetesAPIServiceConfig_modified(name, group, version),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesAPIServiceExists("kubernetes_api_service.test"),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_api_service.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_api_service.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_api_service.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_api_service.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.service.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.service.0.name", "metrics-server"),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.service.0.namespace", "kube-system"),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.group", group),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.group_priority_minimum", "100"),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.version", version),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.version_priority", "100"),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.ca_bundle", "ZGF0YQ=="),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.insecure_skip_tls_verify", "false"),
				),
			},
			{
				Config: testAccKubernetesAPIServiceConfig_modified_local_service(name, group, version),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesAPIServiceExists("kubernetes_api_service.test"),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_api_service.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_api_service.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_api_service.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_api_service.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.service.#", "0"),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.group", group),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.group_priority_minimum", "100"),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.version", version),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.version_priority", "100"),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.ca_bundle", ""),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.insecure_skip_tls_verify", "false"),
				),
			},
			{
				Config: testAccKubernetesAPIServiceConfig_basic(name, group, version),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesAPIServiceExists("kubernetes_api_service.test"),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_api_service.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_api_service.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_api_service.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_api_service.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.service.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.service.0.name", "metrics-server"),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.service.0.namespace", "kube-system"),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.group", group),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.group_priority_minimum", "1"),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.version", version),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.version_priority", "1"),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.ca_bundle", ""),
					resource.TestCheckResourceAttr("kubernetes_api_service.test", "spec.0.insecure_skip_tls_verify", "true"),
				),
			},
		},
	})
}

func TestAccKubernetesAPIService_importBasic(t *testing.T) {
	resourceName := "kubernetes_api_service.test"
	group := fmt.Sprintf("tf-acc-test-%s.k8s.io", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	version := "v1beta1"
	name := fmt.Sprintf("%s.%s", version, group)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesAPIServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesAPIServiceConfig_basic(name, group, version),
			},

			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
		},
	})
}

func testAccCheckKubernetesAPIServiceDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*KubeClientsets).AggregatorClientset

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_api_service" {
			continue
		}

		name := rs.Primary.ID

		resp, err := conn.ApiregistrationV1().APIServices().Get(name, meta_v1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Service still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesAPIServiceExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*KubeClientsets).AggregatorClientset

		name := rs.Primary.ID

		_, err := conn.ApiregistrationV1().APIServices().Get(name, meta_v1.GetOptions{})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccKubernetesAPIServiceConfig_basic(name, group, version string) string {
	return fmt.Sprintf(`
resource "kubernetes_api_service" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    name = "%s"
  }

  spec {
    service {
      name        = "metrics-server"
      namespace   = "kube-system"
    }

    group                  = "%s"
    group_priority_minimum  = 1

    version          = "%s"
    version_priority = 1

    insecure_skip_tls_verify = true
  }
}
`, name, group, version)
}

func testAccKubernetesAPIServiceConfig_modified(name, group, version string) string {
	return fmt.Sprintf(`
  resource "kubernetes_api_service" "test" {
    metadata {
      annotations = {
        TestAnnotationOne = "one"
      }

      labels = {
        TestLabelOne = "one"
        TestLabelTwo = "two"
      }

      name = "%s"
    }

    spec {
      service {
        name        = "metrics-server"
        namespace   = "kube-system"
      }

      group                  = "%s"
      group_priority_minimum = 100

      version          = "%s"
      version_priority = 100

      ca_bundle = "${base64encode("data")}"
      insecure_skip_tls_verify = false
    }
  }
`, name, group, version)
}

func testAccKubernetesAPIServiceConfig_modified_local_service(name, group, version string) string {
	return fmt.Sprintf(`
  resource "kubernetes_api_service" "test" {
    metadata {
      annotations = {
        TestAnnotationOne = "one"
      }

      labels = {
        TestLabelOne = "one"
        TestLabelTwo = "two"
      }

      name = "%s"
    }

    spec {
      group                  = "%s"
      group_priority_minimum = 100

      version          = "%s"
      version_priority = 100

      insecure_skip_tls_verify = false
    }
  }
`, name, group, version)
}
