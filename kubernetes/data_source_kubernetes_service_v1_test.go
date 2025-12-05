// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	corev1 "k8s.io/api/core/v1"
)

func TestAccKubernetesDataSourceServiceV1_basic(t *testing.T) {
	resourceName := "kubernetes_service_v1.test"
	dataSourceName := "data.kubernetes_service_v1.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesDataSourceServiceV1_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.allocate_load_balancer_node_ports"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cluster_ip"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cluster_ips.#"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.internal_traffic_policy", "Cluster"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ip_families.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ip_families.0", "IPv4"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ip_family_policy", "SingleStack"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cluster_ip"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.name", ""),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.node_port", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.port", "8080"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.target_port", "80"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.app_protocol", "http"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.session_affinity", "None"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.type", "ClusterIP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.health_check_node_port", "0"),
				),
			},
			{
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesDataSourceServiceV1_basic(name) +
					testAccKubernetesDataSourceServiceV1_read(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.allocate_load_balancer_node_ports", "true"),
					resource.TestCheckResourceAttrSet(dataSourceName, "spec.0.cluster_ip"),
					resource.TestCheckResourceAttrSet(dataSourceName, "spec.0.cluster_ips.#"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.internal_traffic_policy", "Cluster"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.ip_families.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.ip_families.0", "IPv4"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.ip_family_policy", "SingleStack"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.port.#", "1"),
					resource.TestCheckResourceAttrSet(dataSourceName, "spec.0.cluster_ip"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.port.0.name", ""),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.port.0.node_port", "0"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.port.0.port", "8080"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.port.0.protocol", "TCP"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.port.0.target_port", "80"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.port.0.app_protocol", "http"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.session_affinity", "None"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.type", "ClusterIP"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.health_check_node_port", "0"),
				),
			},
		},
	})
}

func TestAccKubernetesDataSourceServiceV1_not_found(t *testing.T) {
	dataSourceName := "data.kubernetes_service_v1.test"
	name := fmt.Sprintf("ceci-n.est-pas-une-service-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesDataSourceServiceV1_nonexistent(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(dataSourceName, "spec.#", "0"),
				),
			},
		},
	})
}

func TestAccKubernetesDataSourceServiceV1_loadBalancer_ipMode(t *testing.T) {
	var conf corev1.Service
	name := acctest.RandomWithPrefix("tf-acc-test")
	datasourceName := "data.kubernetes_service_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNoLoadBalancersAvailable(t) },
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesServiceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesDataSourceServiceV1Config_loadBalancer_ipMode(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceV1Exists(datasourceName, &conf),
					resource.TestCheckResourceAttr(datasourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(datasourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(datasourceName, "spec.0.type", "LoadBalancer"),
					resource.TestCheckResourceAttr(datasourceName, "status.0.load_balancer.0.ingress.0.ip_mode", "VIP"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceServiceV1Config_loadBalancer_ipMode(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_service_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    type = "LoadBalancer"
    selector = {
      app = "test-app"
    }
    port {
      port        = 80
      target_port = 80
    }
  }
}

data "kubernetes_service_v1" "test" {
  metadata {
    name = "${kubernetes_service_v1.test.metadata.0.name}"
  }
}	
`, name)
}

func testAccKubernetesDataSourceServiceV1_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service_v1" "test" {
  metadata {
    name = "%s"
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }
    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }
  }
  spec {
    ip_families      = ["IPv4"]
    ip_family_policy = "SingleStack"
    port {
      port         = 8080
      target_port  = 80
      app_protocol = "http"
    }
  }
}
`, name)
}

func testAccKubernetesDataSourceServiceV1_read() string {
	return `data "kubernetes_service_v1" "test" {
  metadata {
    name = "${kubernetes_service_v1.test.metadata.0.name}"
  }
}
`
}

func testAccKubernetesDataSourceServiceV1_nonexistent(name string) string {
	return fmt.Sprintf(`data "kubernetes_service_v1" "test" {
  metadata {
    name = "%s"
  }
}
`, name)
}
