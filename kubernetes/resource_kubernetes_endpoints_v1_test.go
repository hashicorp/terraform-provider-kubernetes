// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesEndpointsV1_basic(t *testing.T) {
	var conf api.Endpoints
	resourceName := "kubernetes_endpoints_v1.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesEndpointV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesEndpointsV1_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesEndpointV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "subset.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.address.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.address.0.ip", "10.0.0.4"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.port.0.name", "httptransport"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.port.0.port", "80"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.port.0.protocol", "TCP"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesEndpointsV1_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesEndpointV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "subset.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.address.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.address.0.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.address.0.ip", "10.0.0.5"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.port.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.port.0.name", "httptransport"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.port.0.port", "82"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.port.0.protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.port.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.port.1.name", "httpstransport"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.port.1.port", "443"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.port.1.protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.port.2.name", "httpstransport2"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.port.2.port", "444"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.port.2.protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.port.0.name", "aaaa"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.port.0.port", "442"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.port.0.protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.address.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.address.0.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.address.0.ip", "10.0.0.7"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.address.1.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.address.1.ip", "10.0.0.6"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.address.1.hostname", "test-hostname"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.not_ready_address.0.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.not_ready_address.0.ip", "10.0.0.10"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.not_ready_address.1.ip", "10.0.0.11"),
					resource.TestCheckResourceAttr(resourceName, "subset.1.not_ready_address.1.%", "3"),
				),
			},
			{
				Config: testAccKubernetesEndpointsV1_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesEndpointV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "subset.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.address.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.address.0.ip", "10.0.0.4"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.port.0.name", "httptransport"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.port.0.port", "80"),
					resource.TestCheckResourceAttr(resourceName, "subset.0.port.0.protocol", "TCP"),
				),
			},
		},
	})
}

func TestAccKubernetesEndpointsV1_generatedName(t *testing.T) {
	var conf api.Endpoints
	prefix := "tf-acc-test-gen-"
	resourceName := "kubernetes_endpoints_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesEndpointV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesEndpointsV1_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesEndpointV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr(resourceName, "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
				),
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

func testAccCheckKubernetesEndpointV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_endpoints_v1" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.CoreV1().Endpoints(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Endpoint still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesEndpointV1Exists(n string, obj *api.Endpoints) resource.TestCheckFunc {
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

		out, err := conn.CoreV1().Endpoints(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesEndpointsV1_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_endpoints_v1" "test" {
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
  subset {
    address {
      ip = "10.0.0.4"
    }
    port {
      name     = "httptransport"
      port     = 80
      protocol = "TCP"
    }
  }
}
`, name)
}

func testAccKubernetesEndpointsV1_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_endpoints_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      Different         = "4424"
    }
    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
    }
    name = "%s"
  }
  subset {
    address {
      ip = "10.0.0.5"
    }
    port {
      name     = "httptransport"
      port     = 82
      protocol = "TCP"
    }
  }
  subset {
    address {
      ip        = "10.0.0.6"
      hostname  = "test-hostname"
      node_name = "test-nodename"
    }
    address {
      ip = "10.0.0.7"
    }
    not_ready_address {
      ip = "10.0.0.10"
    }
    not_ready_address {
      ip = "10.0.0.11"
    }
    port {
      name     = "httpstransport"
      port     = 443
      protocol = "TCP"
    }
    port {
      name     = "httpstransport2"
      port     = 444
      protocol = "TCP"
    }
    port {
      name     = "aaaa"
      port     = 442
      protocol = "TCP"
    }
  }
}
`, name)
}

func testAccKubernetesEndpointsV1_generatedName(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_endpoints_v1" "test" {
  metadata {
    generate_name = "%s"
  }

  subset {
    address {
      ip = "10.0.0.4"
    }

    port {
      name     = "transport"
      port     = 80
      protocol = "TCP"
    }
  }
}
`, prefix)
}
