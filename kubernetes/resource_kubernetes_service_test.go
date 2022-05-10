package kubernetes

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestAccKubernetesService_basic(t *testing.T) {
	var conf api.Service
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_service.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesServiceConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.allocate_load_balancer_node_ports"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cluster_ip"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cluster_ips.#"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.name", ""),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.node_port", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.port", "8080"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.target_port", "80"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.session_affinity", "None"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.type", "ClusterIP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.publish_not_ready_addresses", "false"),
					testAccCheckServicePorts(&conf, []api.ServicePort{
						{
							Port:       int32(8080),
							Protocol:   api.ProtocolTCP,
							TargetPort: intstr.FromInt(80),
						},
					}),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "wait_for_load_balancer"},
			},
			{
				Config: testAccKubernetesServiceConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cluster_ip"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.name", ""),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.node_port", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.port", "8081"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.target_port", "80"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.session_affinity", "None"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.type", "ClusterIP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.publish_not_ready_addresses", "true"),
					testAccCheckServicePorts(&conf, []api.ServicePort{
						{
							Port:       int32(8081),
							Protocol:   api.ProtocolTCP,
							TargetPort: intstr.FromInt(80),
						},
					}),
				),
			},
			{
				Config: testAccKubernetesServiceConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cluster_ip"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.name", ""),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.node_port", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.port", "8080"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.target_port", "80"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.session_affinity", "None"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.type", "ClusterIP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.publish_not_ready_addresses", "false"),
					testAccCheckServicePorts(&conf, []api.ServicePort{
						{
							Port:       int32(8080),
							Protocol:   api.ProtocolTCP,
							TargetPort: intstr.FromInt(80),
						},
					}),
				),
			},
		},
	})
}

func TestAccKubernetesService_loadBalancer(t *testing.T) {
	var conf api.Service
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_service.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNoLoadBalancersAvailable(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesServiceConfig_loadBalancer(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.port.0.node_port"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.port", "8888"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.target_port", "80"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cluster_ip"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_ips.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_ips.1", "10.0.0.4"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_ips.0", "10.0.0.3"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_name", "ext-name-"+name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_traffic_policy", "Cluster"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.load_balancer_source_ranges.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.load_balancer_source_ranges.0", "10.0.0.5/32"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.load_balancer_source_ranges.1", "10.0.0.6/32"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.App", "MyApp"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.type", "LoadBalancer"),
					resource.TestCheckResourceAttrSet(resourceName, "status.0.load_balancer.0.ingress.0.hostname"),
					testAccCheckServicePorts(&conf, []api.ServicePort{
						{
							Port:       int32(8888),
							Protocol:   api.ProtocolTCP,
							TargetPort: intstr.FromInt(80),
						},
					}),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "wait_for_load_balancer"},
			},
			{
				Config: testAccKubernetesServiceConfig_loadBalancer_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cluster_ip"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_ips.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_ips.0", "10.0.0.4"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_ips.1", "10.0.0.5"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_name", "ext-name-modified-"+name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_traffic_policy", "Local"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.load_balancer_source_ranges.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.load_balancer_source_ranges.0", "10.0.0.1/32"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.load_balancer_source_ranges.1", "10.0.0.2/32"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.port.0.node_port"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.port", "9999"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.target_port", "81"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.App", "MyModifiedApp"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.NewSelector", "NewValue"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.type", "LoadBalancer"),
					testAccCheckServicePorts(&conf, []api.ServicePort{
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

func TestAccKubernetesService_loadBalancer_internal_traffic_policy(t *testing.T) {
	var conf api.Service
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_service_v1.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoLoadBalancersAvailable(t)
			// internalTrafficPolicy is availabe in version 1.22+
			skipIfClusterVersionLessThan(t, "1.21.0")
		},
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesServiceConfig_loadBalancer_internal_traffic_policy(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_traffic_policy", "Cluster"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.internal_traffic_policy", "Cluster"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "wait_for_load_balancer"},
			},
			{
				Config: testAccKubernetesServiceConfig_loadBalancer_internal_traffic_policy_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_traffic_policy", "Local"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.internal_traffic_policy", "Local"),
				),
			},
		},
	})
}

func TestAccKubernetesService_loadBalancer_class(t *testing.T) {
	var conf api.Service
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_service_v1.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesServiceConfig_loadBalancer_class(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.type", "LoadBalancer"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.load_balancer_class", "loadbalancer.io/loadbalancer"),
				),
			},
		},
	})
}

func TestAccKubernetesService_loadBalancer_healthcheck(t *testing.T) {
	var conf api.Service
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_service.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNoLoadBalancersAvailable(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesServiceConfig_loadBalancer_healthcheck(name, 31111),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_traffic_policy", "Local"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.type", "LoadBalancer"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.health_check_node_port", "31111"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "wait_for_load_balancer"},
			},
			{
				Config: testAccKubernetesServiceConfig_loadBalancer_healthcheck(name, 31112),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_traffic_policy", "Local"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.type", "LoadBalancer"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.health_check_node_port", "31112"),
				),
			},
		},
	})
}

func TestAccKubernetesService_headless(t *testing.T) {
	var conf api.Service
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_service.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesServiceConfig_headless(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cluster_ip", "None"),
				),
			},
		},
	})
}

func TestAccKubernetesService_loadBalancer_annotations_aws(t *testing.T) {
	var conf api.Service
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_service.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNoLoadBalancersAvailable(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesServiceConfig_loadBalancer_annotations_aws(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.service.beta.kubernetes.io/aws-load-balancer-backend-protocol", "http"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.service.beta.kubernetes.io/aws-load-balancer-connection-idle-timeout", "300"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.service.beta.kubernetes.io/aws-load-balancer-ssl-ports", "*"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.port.0.node_port"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.port", "8888"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.target_port", "80"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cluster_ip"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_ips.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_ips.1", "10.0.0.4"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_ips.0", "10.0.0.3"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_name", "ext-name-"+name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.load_balancer_source_ranges.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.load_balancer_source_ranges.0", "10.0.0.5/32"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.load_balancer_source_ranges.1", "10.0.0.6/32"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.App", "MyApp"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.type", "LoadBalancer"),
					testAccCheckServicePorts(&conf, []api.ServicePort{
						{
							Port:       int32(8888),
							Protocol:   api.ProtocolTCP,
							TargetPort: intstr.FromInt(80),
						},
					}),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "wait_for_load_balancer"},
			},
			{
				Config: testAccKubernetesServiceConfig_loadBalancer_annotations_aws_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "4"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.service.beta.kubernetes.io/aws-load-balancer-backend-protocol", "http"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.service.beta.kubernetes.io/aws-load-balancer-connection-idle-timeout", "60"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.service.beta.kubernetes.io/aws-load-balancer-ssl-ports", "*"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.service.beta.kubernetes.io/aws-load-balancer-cross-zone-load-balancing-enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cluster_ip"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_ips.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_ips.0", "10.0.0.4"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_ips.1", "10.0.0.5"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_name", "ext-name-modified-"+name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.load_balancer_source_ranges.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.load_balancer_source_ranges.0", "10.0.0.1/32"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.load_balancer_source_ranges.1", "10.0.0.2/32"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.port.0.node_port"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.port", "9999"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.target_port", "81"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.App", "MyModifiedApp"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.NewSelector", "NewValue"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.type", "LoadBalancer"),
					testAccCheckServicePorts(&conf, []api.ServicePort{
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

func TestAccKubernetesService_nodePort(t *testing.T) {
	var conf api.Service
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_service.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesServiceConfig_nodePort(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cluster_ip"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_ips.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_ips.0", "10.0.0.4"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_ips.1", "10.0.0.5"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_name", "ext-name-"+name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.load_balancer_ip", "12.0.0.125"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.name", "first"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.port.0.node_port"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.port", "10222"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.target_port", "22"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.app_protocol", "ssh"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.1.name", "second"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.port.1.node_port"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.1.port", "10333"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.1.protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.1.target_port", "33"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.1.app_protocol", "terraform.io/kubernetes"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.App", "MyApp"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.session_affinity", "ClientIP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.session_affinity_config.0.client_ip.0.timeout_seconds", "300"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.type", "NodePort"),
					testAccCheckServicePorts(&conf, []api.ServicePort{
						{
							AppProtocol: ptrToString("ssh"),
							Name:        "first",
							Port:        int32(10222),
							Protocol:    api.ProtocolTCP,
							TargetPort:  intstr.FromInt(22),
						},
						{
							AppProtocol: ptrToString("terraform.io/kubernetes"),
							Name:        "second",
							Port:        int32(10333),
							Protocol:    api.ProtocolTCP,
							TargetPort:  intstr.FromInt(33),
						},
					}),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "wait_for_load_balancer"},
			},
			{
				Config: testAccKubernetesServiceConfig_nodePort_toClusterIP(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cluster_ip"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_ips.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_ips.0", "10.0.0.4"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_ips.1", "10.0.0.5"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_name", "ext-name-"+name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.load_balancer_ip", "12.0.0.125"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.name", "first"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.node_port", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.port", "10222"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.target_port", "22"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.1.name", "second"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.1.node_port", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.1.port", "10334"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.1.protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.1.target_port", "33"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.App", "MyApp"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.session_affinity", "ClientIP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.session_affinity_config.0.client_ip.0.timeout_seconds", "300"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.type", "ClusterIP"),
				),
			},
		},
	})
}

func TestAccKubernetesService_noTargetPort(t *testing.T) {
	var conf api.Service
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_service.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNoLoadBalancersAvailable(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesServiceConfig_noTargetPort(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cluster_ip"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_ips.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.name", "http"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.port.0.node_port"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.port", "80"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.0.target_port", "80"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.1.name", "https"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.port.1.node_port"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.1.port", "443"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.1.protocol", "TCP"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.1.target_port", "443"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.App", "MyOtherApp"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.session_affinity", "None"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.type", "LoadBalancer"),
					testAccCheckServicePorts(&conf, []api.ServicePort{
						{
							Name:       "http",
							Port:       int32(80),
							Protocol:   api.ProtocolTCP,
							TargetPort: intstr.FromInt(80),
						},
						{
							Name:       "https",
							Port:       int32(443),
							Protocol:   api.ProtocolTCP,
							TargetPort: intstr.FromInt(443),
						},
					}),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "wait_for_load_balancer"},
			},
		},
	})
}

func TestAccKubernetesService_stringTargetPort(t *testing.T) {
	var conf api.Service
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_service.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNoLoadBalancersAvailable(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesServiceConfig_stringTargetPort(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists(resourceName, &conf),
					testAccCheckServicePorts(&conf, []api.ServicePort{
						{
							Port:       int32(8080),
							Protocol:   api.ProtocolTCP,
							TargetPort: intstr.FromString("http-server"),
						},
					}),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "wait_for_load_balancer"},
			},
		},
	})
}

func TestAccKubernetesService_externalName(t *testing.T) {
	var conf api.Service
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_service.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesServiceConfig_externalName(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cluster_ip", ""),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_ips.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_name", "terraform.io"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.load_balancer_ip", ""),
					resource.TestCheckResourceAttr(resourceName, "spec.0.load_balancer_source_ranges.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.port.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.session_affinity", "None"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.type", "ExternalName"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "wait_for_load_balancer"},
			},
		},
	})
}

func TestAccKubernetesService_externalName_toClusterIp(t *testing.T) {
	var conf api.Service
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_service.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesServiceConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cluster_ip"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.type", "ClusterIP"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "wait_for_load_balancer"},
			},
			{
				Config: testAccKubernetesServiceConfig_externalName(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cluster_ip", ""),
					resource.TestCheckResourceAttr(resourceName, "spec.0.type", "ExternalName"),
				),
			},
			{
				Config: testAccKubernetesServiceConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cluster_ip"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.type", "ClusterIP"),
				),
			},
		},
	})
}

func TestAccKubernetesService_generatedName(t *testing.T) {
	var conf api.Service
	prefix := "tf-acc-test-gen-"
	resourceName := "kubernetes_service.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesServiceConfig_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists(resourceName, &conf),
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
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "wait_for_load_balancer"},
			},
		},
	})
}

func TestAccKubernetesServiceV1_ipFamilies(t *testing.T) {
	var conf api.Service
	prefix := "tf-acc-test-gen-"
	resourceName := "kubernetes_service_v1.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesServiceConfigV1_ipFamilies(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr(resourceName, "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
				),
			},
		},
	})
}

func testAccCheckKubernetesServiceForceNew(old, new *api.Service, wantNew bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if wantNew {
			if old.ObjectMeta.UID == new.ObjectMeta.UID {
				return fmt.Errorf("Expecting new resource for Service %s", old.ObjectMeta.UID)
			}
		} else {
			if old.ObjectMeta.UID != new.ObjectMeta.UID {
				return fmt.Errorf("Expecting Service UIDs to be the same: expected %s got %s", old.ObjectMeta.UID, new.ObjectMeta.UID)
			}
		}
		return nil
	}
}

func testAccCheckServicePorts(svc *api.Service, expected []api.ServicePort) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(expected) == 0 && len(svc.Spec.Ports) == 0 {
			return nil
		}

		ports := svc.Spec.Ports

		// Ignore NodePorts as these are assigned randomly
		for k := range ports {
			ports[k].NodePort = 0
		}

		if !reflect.DeepEqual(ports, expected) {
			return fmt.Errorf("Service ports don't match.\nExpected: %#v\nGiven: %#v",
				expected, svc.Spec.Ports)
		}

		return nil
	}
}

func testAccCheckKubernetesServiceDestroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_service" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Service still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesServiceExists(n string, obj *api.Service) resource.TestCheckFunc {
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

		out, err := conn.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesServiceConfig_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service" "test" {
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
    port {
      port        = 8080
      target_port = 80
    }
  }
}
`, name)
}

func testAccKubernetesServiceConfig_stateUpgradev0(provider, name string) string {
	return fmt.Sprintf(`resource "kubernetes_service" "test" {
  provider = "%s"
  metadata {
    name = "%s"
  }

  spec {
    type = "LoadBalancer"
    port {
      port        = 8080
      target_port = 80
    }
  }
}
`, provider, name)
}

func testAccKubernetesServiceConfig_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      Different         = "1234"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
    }

    name = "%s"
  }

  spec {
    port {
      port        = 8081
      target_port = 80
    }

    publish_not_ready_addresses = "true"
  }
}
`, name)
}

func testAccKubernetesServiceConfig_loadBalancer(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service" "test" {
  metadata {
    name = "%[1]s"
  }

  spec {
    external_name               = "ext-name-%[1]s"
    external_ips                = ["10.0.0.3", "10.0.0.4"]
    load_balancer_source_ranges = ["10.0.0.5/32", "10.0.0.6/32"]

    selector = {
      App = "MyApp"
    }

    port {
      port        = 8888
      target_port = 80
    }

    type = "LoadBalancer"
  }
}
`, name)
}

func testAccKubernetesServiceConfig_loadBalancer_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service" "test" {
  metadata {
    name = "%[1]s"
  }

  spec {
    external_name               = "ext-name-modified-%[1]s"
    external_ips                = ["10.0.0.4", "10.0.0.5"]
    load_balancer_source_ranges = ["10.0.0.1/32", "10.0.0.2/32"]
    external_traffic_policy     = "Local"

    selector = {
      App         = "MyModifiedApp"
      NewSelector = "NewValue"
    }

    port {
      port        = 9999
      target_port = 81
    }

    type = "LoadBalancer"
  }
}
`, name)
}

func testAccKubernetesServiceConfig_loadBalancer_annotations_aws(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service" "test" {
  metadata {
    name = "%[1]s"
    annotations = {
      "service.beta.kubernetes.io/aws-load-balancer-backend-protocol"        = "http"
      "service.beta.kubernetes.io/aws-load-balancer-connection-idle-timeout" = "300"
      "service.beta.kubernetes.io/aws-load-balancer-ssl-ports"               = "*"
    }
  }

  spec {
    external_name               = "ext-name-%[1]s"
    external_ips                = ["10.0.0.3", "10.0.0.4"]
    load_balancer_source_ranges = ["10.0.0.5/32", "10.0.0.6/32"]

    selector = {
      App = "MyApp"
    }

    port {
      port        = 8888
      target_port = 80
    }

    type = "LoadBalancer"
  }
}
`, name)
}

func testAccKubernetesServiceConfig_loadBalancer_annotations_aws_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service" "test" {
  metadata {
    name = "%[1]s"
    annotations = {
      "service.beta.kubernetes.io/aws-load-balancer-backend-protocol"                  = "http"
      "service.beta.kubernetes.io/aws-load-balancer-connection-idle-timeout"           = "60"
    "service.beta.kubernetes.io/aws-load-balancer-ssl-ports"                         = "*"
    "service.beta.kubernetes.io/aws-load-balancer-cross-zone-load-balancing-enabled" = "true"
    }
  }

  spec {
    external_name               = "ext-name-modified-%[1]s"
    external_ips                = ["10.0.0.4", "10.0.0.5"]
    load_balancer_source_ranges = ["10.0.0.1/32", "10.0.0.2/32"]

    selector = {
      App         = "MyModifiedApp"
      NewSelector = "NewValue"
    }

    port {
      port        = 9999
      target_port = 81
    }

    type = "LoadBalancer"
  }
}
`, name)
}

func testAccKubernetesServiceConfig_headless(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service" "test" {
  metadata {
    name = "%s"
  }
  spec {
    cluster_ip = "None"
    selector   = {
      App = "MyApp"
    }
    port {
      port        = 8888
      target_port = 80
    }
  }
}
`, name)
}

func testAccKubernetesServiceConfig_loadBalancer_healthcheck(name string, nodePort int) string {
	return fmt.Sprintf(`resource "kubernetes_service" "test" {
  metadata {
    name = "%[1]s"
  }

  spec {
    external_name               = "ext-name-%[1]s"
    external_ips                = ["10.0.0.3", "10.0.0.4"]
    load_balancer_source_ranges = ["10.0.0.5/32", "10.0.0.6/32"]
    external_traffic_policy     = "Local"
    health_check_node_port      = %[2]d

    selector = {
      App = "MyApp"
    }

    port {
      port        = 8888
      target_port = 80
    }

    type = "LoadBalancer"
  }
}
`, name, nodePort)
}

func testAccKubernetesServiceConfig_loadBalancer_internal_traffic_policy(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service_v1" "test" {
  metadata {
    name = "%[1]s"
  }

  spec {
    external_name               = "ext-name-%[1]s"
    external_ips                = ["10.0.0.3", "10.0.0.4"]
    load_balancer_source_ranges = ["10.0.0.5/32", "10.0.0.6/32"]

    external_traffic_policy = "Cluster"
    internal_traffic_policy = "Cluster"

    selector = {
      App = "MyApp"
    }

    port {
      port        = 8888
      target_port = 80
    }

    type = "LoadBalancer"
  }
}
`, name)
}

func testAccKubernetesServiceConfig_loadBalancer_internal_traffic_policy_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service_v1" "test" {
  metadata {
    name = "%[1]s"
  }

  spec {
    external_name               = "ext-name-%[1]s"
    external_ips                = ["10.0.0.3", "10.0.0.4"]
    load_balancer_source_ranges = ["10.0.0.5/32", "10.0.0.6/32"]

    external_traffic_policy = "Local"
    internal_traffic_policy = "Local"

    selector = {
      App = "MyApp"
    }

    port {
      port        = 8888
      target_port = 80
    }

    type = "LoadBalancer"
  }
}
`, name)
}

func testAccKubernetesServiceConfig_loadBalancer_class(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service_v1" "test" {
  metadata {
    name = "%s"
  }

  spec {
    type                = "LoadBalancer"
    load_balancer_class = "loadbalancer.io/loadbalancer"
    port {
      port        = 80
      target_port = 8080
    }
  }

  wait_for_load_balancer = false
}
`, name)
}

func testAccKubernetesServiceConfig_nodePort(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service" "test" {
  metadata {
    name = "%[1]s"
  }

  spec {
    external_name    = "ext-name-%[1]s"
    external_ips     = ["10.0.0.4", "10.0.0.5"]
    load_balancer_ip = "12.0.0.125"

    selector = {
      App = "MyApp"
    }

    session_affinity = "ClientIP"
    session_affinity_config {
      client_ip {
        timeout_seconds = 300
      }
    }

    port {
      name         = "first"
      port         = 10222
      target_port  = 22
      app_protocol = "ssh"
    }

    port {
      name         = "second"
      port         = 10333
      target_port  = 33
      app_protocol = "terraform.io/kubernetes"
    }

    type = "NodePort"
  }
}
`, name)
}

func testAccKubernetesServiceConfig_nodePort_toClusterIP(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service" "test" {
  metadata {
    name = "%[1]s"
  }

  spec {
    external_name    = "ext-name-%[1]s"
    external_ips     = ["10.0.0.4", "10.0.0.5"]
    load_balancer_ip = "12.0.0.125"

    selector = {
      App = "MyApp"
    }

    session_affinity = "ClientIP"
    session_affinity_config {
      client_ip {
        timeout_seconds = 300
      }
    }

    port {
      name        = "first"
      port        = 10222
      target_port = 22
    }

    port {
      name        = "second"
      port        = 10334
      target_port = 33
    }

    type = "ClusterIP"
  }
}
`, name)
}

func testAccKubernetesServiceConfig_stringTargetPort(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service" "test" {
  metadata {
    name = "%s"

    labels = {
      app  = "helloweb"
      tier = "frontend"
    }
  }

  spec {
    type = "LoadBalancer"

    selector = {
      app  = "helloweb"
      tier = "frontend"
    }

    port {
      port        = 8080
      target_port = "http-server"
    }
  }
}
`, name)
}

func testAccKubernetesServiceConfig_noTargetPort(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service" "test" {
  metadata {
    name = "%s"
  }

  spec {
    selector = {
      App = "MyOtherApp"
    }

    port {
      name = "http"
      port = 80
    }

    port {
      name = "https"
      port = 443
    }

    type = "LoadBalancer"
  }
}
`, name)
}

func testAccKubernetesServiceConfig_externalName(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service" "test" {
  metadata {
    name = "%s"
  }

  spec {
    type          = "ExternalName"
    external_name = "terraform.io"
  }
}
`, name)
}

func testAccKubernetesServiceConfig_generatedName(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_service" "test" {
  metadata {
    generate_name = "%s"
  }

  spec {
    port {
      port        = 8080
      target_port = 80
    }
  }
}
`, prefix)
}

func testAccKubernetesServiceConfigV1_ipFamilies(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_service_v1" "test" {
  metadata {
    generate_name = "%s"
  }

  spec {
    port {
      port        = 8080
      target_port = 80
    }

    ip_families = [
      "IPv4",
    ]
    ip_family_policy = "SingleStack"
  }
}
`, prefix)
}
