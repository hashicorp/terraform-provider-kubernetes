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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesAPIServiceV1_basic(t *testing.T) {
	group := fmt.Sprintf("tf-acc-test-%s.k8s.io", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	version := "v1"
	name := fmt.Sprintf("%s.%s", version, group)
	resourceName := "kubernetes_api_service_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesAPIServiceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesAPIServiceV1Config_basic(name, group, version),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesAPIServiceV1Exists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service.0.name", "metrics-server"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service.0.namespace", "kube-system"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service.0.port", "443"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.group", group),
					resource.TestCheckResourceAttr(resourceName, "spec.0.group_priority_minimum", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.version", version),
					resource.TestCheckResourceAttr(resourceName, "spec.0.version_priority", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ca_bundle", ""),
					resource.TestCheckResourceAttr(resourceName, "spec.0.insecure_skip_tls_verify", "true"),
				),
			},
			{
				Config: testAccKubernetesAPIServiceV1Config_modified(name, group, version),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesAPIServiceV1Exists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service.0.name", "metrics-server"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service.0.namespace", "kube-system"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service.0.port", "8443"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.group", group),
					resource.TestCheckResourceAttr(resourceName, "spec.0.group_priority_minimum", "100"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.version", version),
					resource.TestCheckResourceAttr(resourceName, "spec.0.version_priority", "100"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ca_bundle", "data"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.insecure_skip_tls_verify", "false"),
				),
			},
			{
				Config: testAccKubernetesAPIServiceV1Config_modified_local_service(name, group, version),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesAPIServiceV1Exists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.group", group),
					resource.TestCheckResourceAttr(resourceName, "spec.0.group_priority_minimum", "100"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.version", version),
					resource.TestCheckResourceAttr(resourceName, "spec.0.version_priority", "100"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ca_bundle", ""),
					resource.TestCheckResourceAttr(resourceName, "spec.0.insecure_skip_tls_verify", "false"),
				),
			},
			{
				Config: testAccKubernetesAPIServiceV1Config_basic(name, group, version),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesAPIServiceV1Exists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service.0.name", "metrics-server"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service.0.namespace", "kube-system"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service.0.port", "443"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.group", group),
					resource.TestCheckResourceAttr(resourceName, "spec.0.group_priority_minimum", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.version", version),
					resource.TestCheckResourceAttr(resourceName, "spec.0.version_priority", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ca_bundle", ""),
					resource.TestCheckResourceAttr(resourceName, "spec.0.insecure_skip_tls_verify", "true"),
				),
			},
		},
	})
}

func testAccCheckKubernetesAPIServiceV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).AggregatorClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_api_service_v1" {
			continue
		}

		name := rs.Primary.ID

		resp, err := conn.ApiregistrationV1().APIServices().Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Service still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesAPIServiceV1Exists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn, err := testAccProvider.Meta().(KubeClientsets).AggregatorClientset()
		if err != nil {
			return err
		}
		ctx := context.TODO()

		name := rs.Primary.ID

		_, err = conn.ApiregistrationV1().APIServices().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccKubernetesAPIServiceV1Config_basic(name, group, version string) string {
	return fmt.Sprintf(`resource "kubernetes_api_service_v1" "test" {
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
      name      = "metrics-server"
      namespace = "kube-system"
    }

    group                  = "%s"
    group_priority_minimum = 1

    version          = "%s"
    version_priority = 1

    insecure_skip_tls_verify = true
  }
}
`, name, group, version)
}

func testAccKubernetesAPIServiceV1Config_modified(name, group, version string) string {
	return fmt.Sprintf(`resource "kubernetes_api_service_v1" "test" {
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
      name      = "metrics-server"
      namespace = "kube-system"
      port      = 8443
    }

    group                  = "%s"
    group_priority_minimum = 100

    version          = "%s"
    version_priority = 100

    ca_bundle                = "data"
    insecure_skip_tls_verify = false
  }
}
`, name, group, version)
}

func testAccKubernetesAPIServiceV1Config_modified_local_service(name, group, version string) string {
	return fmt.Sprintf(`resource "kubernetes_api_service_v1" "test" {
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
