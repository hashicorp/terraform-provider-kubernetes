// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	jsonpatch "gopkg.in/evanphx/json-patch.v4"
	coreapi "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/apimachinery/pkg/util/wait"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
)

// Live lifecycle acceptance tests.

var serviceAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": func() (tfprotov6.ProviderServer, error) {
		return mux.MuxServer(context.Background(), "test")
	},
}

func TestAccKubernetesServiceV1_basic(t *testing.T) {
	var conf coreapi.Service
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_service_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: serviceAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesServiceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesServiceV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceV1Exists(resourceName, &conf),
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
					testAccCheckServiceV1Ports(&conf, []coreapi.ServicePort{
						{
							Port:       int32(8080),
							Protocol:   coreapi.ProtocolTCP,
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
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesServiceV1Config_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceV1Exists(resourceName, &conf),
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
					testAccCheckServiceV1Ports(&conf, []coreapi.ServicePort{
						{
							Port:       int32(8081),
							Protocol:   coreapi.ProtocolTCP,
							TargetPort: intstr.FromInt(80),
						},
					}),
				),
			},
			{
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesServiceV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceV1Exists(resourceName, &conf),
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
					testAccCheckServiceV1Ports(&conf, []coreapi.ServicePort{
						{
							Port:       int32(8080),
							Protocol:   coreapi.ProtocolTCP,
							TargetPort: intstr.FromInt(80),
						},
					}),
				),
			},
		},
	})
}

func TestAccKubernetesServiceV1_identity(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_service_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: serviceAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		CheckDestroy: testAccCheckKubernetesServiceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesServiceV1Config_identity(name),
				Check:  resource.TestCheckResourceAttr(resourceName, "wait_for_load_balancer", "true"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(
						resourceName, map[string]knownvalue.Check{
							"namespace":   knownvalue.StringExact("default"),
							"name":        knownvalue.StringExact(name),
							"api_version": knownvalue.StringExact("v1"),
							"kind":        knownvalue.StringExact("Service"),
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

func TestAccKubernetesServiceV1_loadBalancer(t *testing.T) {
	var conf coreapi.Service
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_service_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); skipIfNoLoadBalancersAvailable(t) },
		ProtoV6ProviderFactories: serviceAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesServiceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesServiceV1Config_loadBalancer(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceV1Exists(resourceName, &conf),
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
					testAccCheckloadBalancerIngressCheck(resourceName),
					testAccCheckServiceV1Ports(&conf, []coreapi.ServicePort{
						{
							Port:       int32(8888),
							Protocol:   coreapi.ProtocolTCP,
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
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesServiceV1Config_loadBalancer_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceV1Exists(resourceName, &conf),
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
					testAccCheckServiceV1Ports(&conf, []coreapi.ServicePort{
						{
							Port:       int32(9999),
							Protocol:   coreapi.ProtocolTCP,
							TargetPort: intstr.FromInt(81),
						},
					}),
				),
			},
		},
	})
}

func TestAccKubernetesServiceV1_loadBalancer_internal_traffic_policy(t *testing.T) {
	var conf coreapi.Service
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_service_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoLoadBalancersAvailable(t)
			// internalTrafficPolicy is available in Kubernetes 1.22+.
			skipIfClusterVersionLessThan(t, "1.22.0")
		},
		ProtoV6ProviderFactories: serviceAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesServiceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesServiceV1Config_loadBalancer_internal_traffic_policy(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceV1Exists(resourceName, &conf),
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
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesServiceV1Config_loadBalancer_internal_traffic_policy_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_traffic_policy", "Local"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.internal_traffic_policy", "Local"),
				),
			},
		},
	})
}

func TestAccKubernetesServiceV1_loadBalancer_class(t *testing.T) {
	var conf coreapi.Service
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_service_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: serviceAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesServiceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesServiceV1Config_loadBalancer_class(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.type", "LoadBalancer"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.load_balancer_class", "loadbalancer.io/loadbalancer"),
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

func TestAccKubernetesServiceV1_loadBalancer_healthcheck(t *testing.T) {
	var conf coreapi.Service
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_service_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); skipIfNoLoadBalancersAvailable(t) },
		ProtoV6ProviderFactories: serviceAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesServiceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesServiceV1Config_loadBalancer_healthcheck(name, 31111),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceV1Exists(resourceName, &conf),
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
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesServiceV1Config_loadBalancer_healthcheck(name, 31112),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.external_traffic_policy", "Local"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.type", "LoadBalancer"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.health_check_node_port", "31112"),
				),
			},
		},
	})
}

func TestAccKubernetesServiceV1_headless(t *testing.T) {
	var conf coreapi.Service
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_service_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: serviceAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesServiceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesServiceV1Config_headless(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cluster_ip", "None"),
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

func TestAccKubernetesServiceV1_loadBalancer_annotations_aws(t *testing.T) {
	var conf coreapi.Service
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_service_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoLoadBalancersAvailable(t)
			cloud, err := serviceClusterCloud()
			if err != nil {
				t.Fatal(err)
			}
			if cloud != "aws" {
				t.Skip("AWS load balancer annotation behavior requires an AWS cluster")
			}
		},
		ProtoV6ProviderFactories: serviceAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesServiceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesServiceV1Config_loadBalancer_annotations_aws(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceV1Exists(resourceName, &conf),
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
					testAccCheckServiceV1Ports(&conf, []coreapi.ServicePort{
						{
							Port:       int32(8888),
							Protocol:   coreapi.ProtocolTCP,
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
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesServiceV1Config_loadBalancer_annotations_aws_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceV1Exists(resourceName, &conf),
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
					testAccCheckServiceV1Ports(&conf, []coreapi.ServicePort{
						{
							Port:       int32(9999),
							Protocol:   coreapi.ProtocolTCP,
							TargetPort: intstr.FromInt(81),
						},
					}),
				),
			},
		},
	})
}

func TestAccKubernetesServiceV1_nodePort(t *testing.T) {
	var conf coreapi.Service
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_service_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: serviceAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesServiceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesServiceV1Config_nodePort(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceV1Exists(resourceName, &conf),
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
					resource.TestCheckResourceAttr(resourceName, "spec.0.session_affinity_config.client_ip.timeout_seconds", "300"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.type", "NodePort"),
					testAccCheckServiceV1Ports(&conf, []coreapi.ServicePort{
						{
							AppProtocol: ptr.To("ssh"),
							Name:        "first",
							Port:        int32(10222),
							Protocol:    coreapi.ProtocolTCP,
							TargetPort:  intstr.FromInt(22),
						},
						{
							AppProtocol: ptr.To("terraform.io/kubernetes"),
							Name:        "second",
							Port:        int32(10333),
							Protocol:    coreapi.ProtocolTCP,
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
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesServiceV1Config_nodePort_toClusterIP(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceV1Exists(resourceName, &conf),
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
					resource.TestCheckResourceAttr(resourceName, "spec.0.session_affinity_config.client_ip.timeout_seconds", "300"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.type", "ClusterIP"),
				),
			},
		},
	})
}

func TestAccKubernetesServiceV1_noTargetPort(t *testing.T) {
	var conf coreapi.Service
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_service_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); skipIfNoLoadBalancersAvailable(t) },
		ProtoV6ProviderFactories: serviceAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesServiceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesServiceV1Config_noTargetPort(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceV1Exists(resourceName, &conf),
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
					testAccCheckServiceV1Ports(&conf, []coreapi.ServicePort{
						{
							Name:       "http",
							Port:       int32(80),
							Protocol:   coreapi.ProtocolTCP,
							TargetPort: intstr.FromInt(80),
						},
						{
							Name:       "https",
							Port:       int32(443),
							Protocol:   coreapi.ProtocolTCP,
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

func TestAccKubernetesServiceV1_stringTargetPort(t *testing.T) {
	var conf coreapi.Service
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_service_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); skipIfNoLoadBalancersAvailable(t) },
		ProtoV6ProviderFactories: serviceAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesServiceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesServiceV1Config_stringTargetPort(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceV1Exists(resourceName, &conf),
					testAccCheckServiceV1Ports(&conf, []coreapi.ServicePort{
						{
							Port:       int32(8080),
							Protocol:   coreapi.ProtocolTCP,
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

func TestAccKubernetesServiceV1_externalName(t *testing.T) {
	var conf coreapi.Service
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_service_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: serviceAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesServiceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesServiceV1Config_externalName(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceV1Exists(resourceName, &conf),
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

func TestAccKubernetesServiceV1_externalName_toClusterIp(t *testing.T) {
	var conf coreapi.Service
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_service_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: serviceAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesServiceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesServiceV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceV1Exists(resourceName, &conf),
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
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesServiceV1Config_externalName(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.cluster_ip", ""),
					resource.TestCheckResourceAttr(resourceName, "spec.0.type", "ExternalName"),
				),
			},
			{
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesServiceV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.cluster_ip"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.type", "ClusterIP"),
				),
			},
		},
	})
}

func TestAccKubernetesServiceV1_generatedName(t *testing.T) {
	var conf coreapi.Service
	prefix := acctest.RandomWithPrefix("tf-acc-test-gen") + "-"
	resourceName := "kubernetes_service_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: serviceAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesServiceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesServiceV1Config_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceV1Exists(resourceName, &conf),
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
	var conf coreapi.Service
	prefix := acctest.RandomWithPrefix("tf-acc-test-gen") + "-"
	resourceName := "kubernetes_service_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: serviceAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesServiceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesServiceV1ConfigV1_ipFamilies(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceV1Exists(resourceName, &conf),
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

func TestAccKubernetesServiceV1_loadBalancer_ipMode(t *testing.T) {
	var conf coreapi.Service
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_service_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNoLoadBalancersAvailable(t)
		},
		IDRefreshIgnore:          []string{"metadata.0.resource_version"},
		ProtoV6ProviderFactories: serviceAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesServiceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesServiceV1Config_loadBalancer_ipMode(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.type", "LoadBalancer"),
					resource.TestCheckResourceAttr(resourceName, "status.0.load_balancer.0.ingress.0.ip_mode", "VIP"),
				),
			},
		},
	})
}

func testAccKubernetesServiceV1Config_loadBalancer_ipMode(name string) string {
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
`, name)
}

func testAccCheckServiceV1Ports(svc *coreapi.Service, expected []coreapi.ServicePort) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(expected) == 0 && len(svc.Spec.Ports) == 0 {
			return nil
		}

		ports := svc.DeepCopy().Spec.Ports

		// Ignore NodePorts as these are assigned randomly; do not mutate the API snapshot.
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

func testAccCheckloadBalancerIngressCheck(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		lb := "status.0.load_balancer.0.ingress.0"
		cloud, err := serviceClusterCloud()
		if err != nil {
			return err
		}
		if cloud == "gce" {
			ip := fmt.Sprintf("%s.ip", lb)
			if rs.Primary.Attributes[ip] != "" {
				return nil
			}
			return fmt.Errorf("Attribute '%s' expected to be set for GKE cluster", ip)
		} else {
			hostname := fmt.Sprintf("%s.hostname", lb)
			if rs.Primary.Attributes[hostname] != "" {
				return nil
			}
			return fmt.Errorf("Attribute '%s' expected to be set for EKS cluster", hostname)
		}
	}
}

func testAccCheckKubernetesServiceV1Destroy(s *terraform.State) error {
	conn, err := mainClientset()
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_service" && rs.Type != "kubernetes_service_v1" {
			continue
		}
		namespace, name, err := kubernetes.IdParts(rs.Primary.ID)
		if err != nil {
			return err
		}
		_, err = conn.CoreV1().Services(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("checking Service %s destruction: %w", rs.Primary.ID, err)
		}
		return fmt.Errorf("Service still exists: %s", rs.Primary.ID)
	}
	return nil
}

func testAccCheckKubernetesServiceV1Exists(n string, obj *coreapi.Service) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn, err := mainClientset()
		if err != nil {
			return err
		}
		ctx := context.TODO()

		namespace, name, err := kubernetes.IdParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		out, err := conn.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out.DeepCopy()
		return nil
	}
}

// Match the retained SDK tests' cloud availability check. Actual test runs
// must still be scoped to an authorized sandbox.
func skipIfNoLoadBalancersAvailable(t *testing.T) {
	t.Helper()
	cloud, err := serviceClusterCloud()
	if err != nil {
		t.Fatal(err)
	}
	if cloud != "aws" && cloud != "gce" {
		t.Skip("requires EKS or GKE load balancer provisioning")
	}
}

func serviceClusterCloud() (string, error) {
	client, err := mainClientset()
	if err != nil {
		return "", err
	}
	nodes, err := client.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	for _, node := range nodes.Items {
		for _, cloud := range []string{"aws", "gce"} {
			if strings.HasPrefix(node.Spec.ProviderID, cloud+"://") {
				return cloud, nil
			}
		}
	}
	return "", nil
}

func skipIfClusterVersionLessThan(t *testing.T, minimum string) {
	t.Helper()
	client, err := mainClientset()
	if err != nil {
		t.Fatal(err)
	}
	serverVersion, err := client.Discovery().ServerVersion()
	if err != nil {
		t.Fatal(err)
	}
	current, err := version.ParseGeneric(serverVersion.GitVersion)
	if err != nil {
		t.Fatal(err)
	}
	if current.LessThan(version.MustParseGeneric(minimum)) {
		t.Skipf("requires Kubernetes %s or newer", minimum)
	}
}

func testAccKubernetesConfig_ignoreAnnotations() string {
	return `provider "kubernetes" {
  ignore_annotations = [
    "cloud\\.google\\.com\\/neg",
  ]
}
`
}

func testAccKubernetesServiceV1Config_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service_v1" "test" {
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

func testAccKubernetesServiceV1Config_identity(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service_v1" "test" {
  metadata {
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

func testAccKubernetesServiceV1Config_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service_v1" "test" {
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

func testAccKubernetesServiceV1Config_loadBalancer(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service_v1" "test" {
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

func testAccKubernetesServiceV1Config_loadBalancer_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service_v1" "test" {
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

func testAccKubernetesServiceV1Config_loadBalancer_annotations_aws(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service_v1" "test" {
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

func testAccKubernetesServiceV1Config_loadBalancer_annotations_aws_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service_v1" "test" {
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

func testAccKubernetesServiceV1Config_headless(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    cluster_ip = "None"
    selector = {
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

func testAccKubernetesServiceV1Config_loadBalancer_healthcheck(name string, nodePort int) string {
	return fmt.Sprintf(`resource "kubernetes_service_v1" "test" {
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

func testAccKubernetesServiceV1Config_loadBalancer_internal_traffic_policy(name string) string {
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

func testAccKubernetesServiceV1Config_loadBalancer_internal_traffic_policy_modified(name string) string {
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

func testAccKubernetesServiceV1Config_loadBalancer_class(name string) string {
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

func testAccKubernetesServiceV1Config_nodePort(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service_v1" "test" {
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
    session_affinity_config = {
      client_ip = {
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

func testAccKubernetesServiceV1Config_nodePort_toClusterIP(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service_v1" "test" {
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
    session_affinity_config = {
      client_ip = {
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

func testAccKubernetesServiceV1Config_stringTargetPort(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service_v1" "test" {
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

func testAccKubernetesServiceV1Config_noTargetPort(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service_v1" "test" {
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

func testAccKubernetesServiceV1Config_externalName(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service_v1" "test" {
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

func testAccKubernetesServiceV1Config_generatedName(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_service_v1" "test" {
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

func testAccKubernetesServiceV1ConfigV1_ipFamilies(prefix string) string {
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

func TestAccKubernetesServiceV1_defaultsAndRemoval(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-service-defaults")
	address := "kubernetes_service_v1.test"
	var baseline *coreapi.Service
	check := func(populated bool, targetPort intstr.IntOrString) resource.TestCheckFunc {
		return resource.ComposeAggregateTestCheckFunc(serviceCheckStableAPI(address, &baseline, func(actual *coreapi.Service) error {
			if actual.Spec.Type != coreapi.ServiceTypeClusterIP || actual.Spec.SessionAffinity != coreapi.ServiceAffinityNone {
				return fmt.Errorf("unexpected Service defaults: type=%s affinity=%s", actual.Spec.Type, actual.Spec.SessionAffinity)
			}
			if populated {
				if actual.Annotations["example.com/managed"] != "yes" || actual.Labels["example.com/managed"] != "yes" ||
					!reflect.DeepEqual(actual.Spec.Selector, map[string]string{"app": name}) ||
					!reflect.DeepEqual(actual.Spec.ExternalIPs, []string{"192.0.2.23"}) ||
					!actual.Spec.PublishNotReadyAddresses || len(actual.Spec.Ports) != 2 ||
					actual.Spec.Ports[0].TargetPort != targetPort ||
					actual.Spec.Ports[0].AppProtocol == nil || *actual.Spec.Ports[0].AppProtocol != "http" ||
					actual.Spec.Ports[1].Protocol != coreapi.ProtocolUDP {
					return fmt.Errorf("optional fields were not applied: metadata=%#v spec=%#v", actual.ObjectMeta, actual.Spec)
				}
			} else if len(actual.Annotations) != 0 || len(actual.Labels) != 0 ||
				len(actual.Spec.Selector) != 0 || len(actual.Spec.ExternalIPs) != 0 ||
				actual.Spec.PublishNotReadyAddresses || len(actual.Spec.Ports) != 1 ||
				actual.Spec.Ports[0].Name != "" || actual.Spec.Ports[0].AppProtocol != nil ||
				actual.Spec.Ports[0].TargetPort != targetPort ||
				actual.Spec.Ports[0].Protocol != coreapi.ProtocolTCP {
				return fmt.Errorf("omitted fields were not defaulted/removed: metadata=%#v spec=%#v", actual.ObjectMeta, actual.Spec)
			}
			return nil
		}), resource.TestCheckResourceAttr(address, "spec.0.port.0.target_port", targetPort.String()))
	}
	minimal := serviceLifecycleConfig(name, false)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: serviceAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesServiceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: minimal,
				Check: resource.ComposeAggregateTestCheckFunc(
					check(false, intstr.FromInt(80)),
					resource.TestCheckResourceAttr(address, "spec.0.port.0.target_port", "80"),
					resource.TestCheckResourceAttr(address, "spec.0.port.0.protocol", "TCP"),
					resource.TestCheckResourceAttr(address, "spec.0.session_affinity", "None"),
					resource.TestCheckResourceAttr(address, "wait_for_load_balancer", "true"),
				),
				ConfigPlanChecks: serviceSettledPlanChecks(),
			},
			{
				Config: minimal, Check: check(false, intstr.FromInt(80)),
				ConfigPlanChecks: serviceSettledPlanChecks(plancheck.ExpectEmptyPlan()),
			},
			{
				ResourceName: address, ImportState: true, ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "wait_for_load_balancer"},
			},
			{
				Config: serviceLifecycleConfig(name, true), Check: check(true, intstr.FromString("web")),
				ConfigPlanChecks: serviceSettledPlanChecks(plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)),
			},
			{
				// Omitting Optional+Computed target_port retains its prior value.
				Config: minimal, Check: check(false, intstr.FromString("web")),
				ConfigPlanChecks: serviceSettledPlanChecks(plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)),
			},
			{
				Config: minimal, Check: check(false, intstr.FromString("web")),
				ConfigPlanChecks: serviceSettledPlanChecks(plancheck.ExpectEmptyPlan()),
			},
		},
	})
}

func TestAccKubernetesServiceV1_mutableTrafficPolicies(t *testing.T) {
	for _, serviceType := range []string{"NodePort", "LoadBalancer"} {
		t.Run(serviceType, func(t *testing.T) {
			name := acctest.RandomWithPrefix("tf-service-traffic")
			address := "kubernetes_service_v1.test"
			var baseline *coreapi.Service
			var steps []resource.TestStep
			for index, policy := range []string{"Cluster", "Local", "Cluster"} {
				step := resource.TestStep{
					Config: serviceTrafficConfig(name, serviceType, policy),
					Check: serviceCheckStableAPI(address, &baseline, func(actual *coreapi.Service) error {
						if actual.Spec.Type != coreapi.ServiceType(serviceType) ||
							actual.Spec.ExternalTrafficPolicy != coreapi.ServiceExternalTrafficPolicy(policy) ||
							actual.Spec.InternalTrafficPolicy == nil ||
							string(*actual.Spec.InternalTrafficPolicy) != policy {
							return fmt.Errorf("traffic policy mismatch: %#v", actual.Spec)
						}
						for _, port := range actual.Spec.Ports {
							if port.NodePort == 0 {
								return fmt.Errorf("%s did not allocate a NodePort", serviceType)
							}
						}
						if serviceType == "LoadBalancer" && policy == "Local" {
							if actual.Spec.HealthCheckNodePort == 0 {
								return fmt.Errorf("Local LoadBalancer did not allocate its health check NodePort")
							}
						} else if actual.Spec.HealthCheckNodePort != 0 {
							return fmt.Errorf("health check NodePort was not removed: %d", actual.Spec.HealthCheckNodePort)
						}
						return nil
					}),
					ConfigPlanChecks: serviceSettledPlanChecks(),
				}
				if index != 0 {
					step.ConfigPlanChecks = serviceSettledPlanChecks(plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate))
				}
				steps = append(steps, step)
			}
			steps = append(steps, resource.TestStep{
				ResourceName: address, ImportState: true, ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "wait_for_load_balancer"},
			})
			if serviceType == "NodePort" {
				steps = append(steps, resource.TestStep{
					Config: serviceTrafficConfig(name, "ClusterIP", "Cluster"),
					ConfigPlanChecks: serviceSettledPlanChecks(
						plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)),
					Check: serviceCheckStableAPI(address, &baseline, func(actual *coreapi.Service) error {
						if actual.Spec.Type != coreapi.ServiceTypeClusterIP || actual.Spec.ExternalTrafficPolicy != "" {
							return fmt.Errorf("NodePort-to-ClusterIP did not clear external traffic policy: %#v", actual.Spec)
						}
						for _, port := range actual.Spec.Ports {
							if port.NodePort != 0 {
								return fmt.Errorf("NodePort-to-ClusterIP retained NodePort %d", port.NodePort)
							}
						}
						return nil
					}),
				})
			}
			resource.ParallelTest(t, resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(t)
					skipIfClusterVersionLessThan(t, "1.22.0")
				},
				ProtoV6ProviderFactories: serviceAccProtoV6ProviderFactories,
				CheckDestroy:             testAccCheckKubernetesServiceV1Destroy,
				Steps:                    steps,
			})
		})
	}
}

// ClientIP defaults come from the API. This does not claim that traffic routing or
// session stickiness works on a cluster without client pods and a network probe.
func TestAccKubernetesServiceV1_clientIPDefault(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-service-affinity")
	address := "kubernetes_service_v1.test"
	var baseline *coreapi.Service
	var steps []resource.TestStep
	for index, affinity := range []string{"None", "ClientIP", "None"} {
		config := strings.Replace(serviceLifecycleConfig(name, false), "  spec {\n",
			fmt.Sprintf("  spec {\n    session_affinity = %q\n", affinity), 1)
		step := resource.TestStep{
			Config:           config,
			ConfigPlanChecks: serviceSettledPlanChecks(),
			Check: serviceCheckStableAPI(address, &baseline, func(actual *coreapi.Service) error {
				if string(actual.Spec.SessionAffinity) != affinity {
					return fmt.Errorf("session affinity = %s, want %s", actual.Spec.SessionAffinity, affinity)
				}
				if affinity == "ClientIP" {
					if actual.Spec.SessionAffinityConfig == nil ||
						actual.Spec.SessionAffinityConfig.ClientIP == nil ||
						actual.Spec.SessionAffinityConfig.ClientIP.TimeoutSeconds == nil ||
						*actual.Spec.SessionAffinityConfig.ClientIP.TimeoutSeconds != 10800 {
						return fmt.Errorf("ClientIP did not retain the API's 10800 second default: %#v", actual.Spec.SessionAffinityConfig)
					}
				} else if actual.Spec.SessionAffinityConfig != nil {
					return fmt.Errorf("removing ClientIP retained session affinity configuration: %#v", actual.Spec.SessionAffinityConfig)
				}
				return nil
			}),
		}
		if index != 0 {
			step.ConfigPlanChecks = serviceSettledPlanChecks(plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate))
		}
		if affinity == "ClientIP" {
			step.Check = resource.ComposeAggregateTestCheckFunc(step.Check,
				resource.TestCheckResourceAttr(address, "spec.0.session_affinity_config.client_ip.timeout_seconds", "10800"))
		}
		steps = append(steps, step)
	}
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: serviceAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesServiceV1Destroy,
		Steps:                    steps,
	})
}

func TestAccKubernetesServiceV1_dualStack(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-service-dualstack")
	address := "kubernetes_service_v1.test"
	config := strings.Replace(serviceLifecycleConfig(name, false), "  spec {\n", `  spec {
    ip_families      = ["IPv4", "IPv6"]
    ip_family_policy = "RequireDualStack"
`, 1)
	var baseline *coreapi.Service
	check := serviceCheckStableAPI(address, &baseline, func(actual *coreapi.Service) error {
		if !reflect.DeepEqual(actual.Spec.IPFamilies, []coreapi.IPFamily{coreapi.IPv4Protocol, coreapi.IPv6Protocol}) ||
			len(actual.Spec.ClusterIPs) != 2 || actual.Spec.IPFamilyPolicy == nil ||
			*actual.Spec.IPFamilyPolicy != coreapi.IPFamilyPolicyRequireDualStack {
			return fmt.Errorf("API did not allocate the required dual-stack Service: %#v", actual.Spec)
		}
		return nil
	})
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			if os.Getenv("KUBE_SERVICE_DUAL_STACK") != "1" {
				t.Skip("requires a dual-stack Service CIDR cluster; set KUBE_SERVICE_DUAL_STACK=1")
			}
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: serviceAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesServiceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: config, Check: check,
				ConfigPlanChecks: serviceSettledPlanChecks(),
			},
			{
				Config: config, Check: check,
				ConfigPlanChecks: serviceSettledPlanChecks(plancheck.ExpectEmptyPlan()),
			},
			{
				ResourceName: address, ImportState: true, ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "wait_for_load_balancer"},
			},
		},
	})
}

func TestAccKubernetesServiceV1_disappears(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-service-disappears")
	address := "kubernetes_service_v1.test"
	config := serviceLifecycleConfig(name, false)
	var original coreapi.Service
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: serviceAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesServiceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceV1Exists(address, &original),
					func(_ *terraform.State) error {
						client, err := mainClientset()
						if err != nil {
							return err
						}
						// UID preconditions prevent deleting a same-named object
						// that was replaced by another actor.
						return client.CoreV1().Services(original.Namespace).Delete(context.Background(), original.Name,
							metav1.DeleteOptions{Preconditions: &metav1.Preconditions{UID: &original.UID}})
					},
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: config,
				ConfigPlanChecks: serviceSettledPlanChecks(
					plancheck.ExpectResourceAction(address, plancheck.ResourceActionCreate)),
				Check: func(state *terraform.State) error {
					var recreated coreapi.Service
					if err := testAccCheckKubernetesServiceV1Exists(address, &recreated)(state); err != nil {
						return err
					}
					if recreated.UID == original.UID || recreated.UID == "" {
						return fmt.Errorf("disappeared Service was not recreated with a new UID")
					}
					return nil
				},
			},
		},
	})
}

func serviceSettledPlanChecks(preApply ...plancheck.PlanCheck) resource.ConfigPlanChecks {
	return resource.ConfigPlanChecks{
		PreApply:             preApply,
		PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
	}
}

func serviceCheckStableAPI(address string, baseline **coreapi.Service, check func(*coreapi.Service) error) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		var actual coreapi.Service
		if err := testAccCheckKubernetesServiceV1Exists(address, &actual)(state); err != nil {
			return err
		}
		if actual.UID == "" || len(actual.Spec.ClusterIPs) == 0 {
			return fmt.Errorf("Service identity or allocated ClusterIPs are missing")
		}
		if *baseline == nil {
			*baseline = actual.DeepCopy()
		} else if actual.UID != (*baseline).UID ||
			!reflect.DeepEqual(actual.Spec.ClusterIPs, (*baseline).Spec.ClusterIPs) {
			return fmt.Errorf("mutable update changed UID/ClusterIPs: %s/%v -> %s/%v",
				(*baseline).UID, (*baseline).Spec.ClusterIPs, actual.UID, actual.Spec.ClusterIPs)
		}
		if err := resource.TestCheckResourceAttr(address, "metadata.0.uid", string(actual.UID))(state); err != nil {
			return err
		}
		return check(&actual)
	}
}

func serviceLifecycleConfig(name string, populated bool) string {
	metadata, spec, ports := "", "", `
    port {
      port = 80
    }
`
	if populated {
		metadata = `
    annotations = {
      "example.com/managed" = "yes"
    }
    labels = {
      "example.com/managed" = "yes"
    }
`
		spec = fmt.Sprintf(`
    selector = {
      app = %q
    }
    external_ips                = ["192.0.2.23"]
    publish_not_ready_addresses = true
`, name)
		ports = `
    port {
      name         = "http"
      port         = 80
      target_port  = "web"
      app_protocol = "http"
    }
    port {
      name        = "dns"
      port        = 5353
      protocol    = "UDP"
      target_port = 53
    }
`
	}
	return fmt.Sprintf(`resource "kubernetes_service_v1" "test" {
  metadata {
    name = %q
%s
  }
  spec {
%s%s
  }
}
`, name, metadata, spec, ports)
}

func serviceTrafficConfig(name, serviceType, policy string) string {
	externalPolicy := fmt.Sprintf("    external_traffic_policy = %q\n", policy)
	class := ""
	if serviceType == "ClusterIP" {
		externalPolicy = ""
	}
	if serviceType == "LoadBalancer" {
		class = "    load_balancer_class = \"terraform-acceptance.invalid/no-controller\"\n"
	}
	return fmt.Sprintf(`resource "kubernetes_service_v1" "test" {
  metadata {
    name = %q
  }
  spec {
    type                    = %q
    internal_traffic_policy = %q
%s%s
    port {
      name        = "http"
      port        = 80
      target_port = 8080
    }
  }
  wait_for_load_balancer = false
}
`, name, serviceType, policy, externalPolicy, class)
}

func TestAccKubernetesServiceV1_nodePortCombinedEditPreservesAllocation(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-service-port-edit")
	address := "kubernetes_service_v1.test"
	var baseline *coreapi.Service
	check := func(edited bool) resource.TestCheckFunc {
		portName, portNumber := "", int32(80)
		if edited {
			portName, portNumber = "http", 81
		}
		return resource.ComposeAggregateTestCheckFunc(
			serviceCheckStableAPI(address, &baseline, func(actual *coreapi.Service) error {
				if len(baseline.Spec.Ports) != 1 || baseline.Spec.Ports[0].NodePort == 0 {
					return fmt.Errorf("baseline Service did not allocate exactly one NodePort: %#v", baseline.Spec.Ports)
				}
				expected := baseline.DeepCopy()
				expected.Spec.Ports[0].Name = portName
				expected.Spec.Ports[0].Port = portNumber
				if !reflect.DeepEqual(expected.Spec, actual.Spec) {
					return fmt.Errorf("combined port edit changed more than name/port: expected %#v, got %#v", expected.Spec, actual.Spec)
				}
				if actual.Spec.Ports[0].TargetPort != intstr.FromInt(80) {
					return fmt.Errorf("omitted target_port changed from its original API default: %v", actual.Spec.Ports[0].TargetPort)
				}
				t.Logf("Service %s/%s UID=%s ClusterIPs=%v nodePort=%d targetPort=%s name=%q port=%d",
					actual.Namespace, actual.Name, actual.UID, actual.Spec.ClusterIPs, actual.Spec.Ports[0].NodePort,
					actual.Spec.Ports[0].TargetPort.String(), actual.Spec.Ports[0].Name, actual.Spec.Ports[0].Port)
				return nil
			}),
			resource.TestCheckResourceAttr(address, "spec.0.port.0.name", portName),
			resource.TestCheckResourceAttr(address, "spec.0.port.0.port", fmt.Sprint(portNumber)),
			resource.TestCheckResourceAttr(address, "spec.0.port.0.target_port", "80"),
			func(state *terraform.State) error {
				if baseline == nil || len(baseline.Spec.Ports) != 1 {
					return fmt.Errorf("missing allocated NodePort snapshot")
				}
				return resource.TestCheckResourceAttr(address, "spec.0.port.0.node_port",
					fmt.Sprint(baseline.Spec.Ports[0].NodePort))(state)
			},
		)
	}
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: serviceAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesServiceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: serviceNodePortCombinedEditConfig(name, false), Check: check(false),
				ConfigPlanChecks: serviceSettledPlanChecks(),
			},
			{
				Config: serviceNodePortCombinedEditConfig(name, true), Check: check(true),
				ConfigPlanChecks: serviceSettledPlanChecks(plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)),
			},
			{
				Config: serviceNodePortCombinedEditConfig(name, true), Check: check(true),
				ConfigPlanChecks: serviceSettledPlanChecks(plancheck.ExpectEmptyPlan()),
			},
		},
	})
}

func TestAccKubernetesServiceV1_externalIPsRetainLocalTrafficPolicy(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-service-external-ip")
	address := "kubernetes_service_v1.test"
	var baseline *coreapi.Service
	check := func(serviceType coreapi.ServiceType) resource.TestCheckFunc {
		return resource.ComposeAggregateTestCheckFunc(
			serviceCheckStableAPI(address, &baseline, func(actual *coreapi.Service) error {
				if len(baseline.Spec.Ports) != 1 || baseline.Spec.Ports[0].NodePort == 0 {
					return fmt.Errorf("baseline NodePort was not allocated: %#v", baseline.Spec.Ports)
				}
				expected := baseline.DeepCopy()
				expected.Spec.Type = serviceType
				if serviceType == coreapi.ServiceTypeClusterIP {
					expected.Spec.Ports[0].NodePort = 0
				}
				if !reflect.DeepEqual(expected.Spec, actual.Spec) {
					return fmt.Errorf("type change did not preserve external IP traffic settings: expected %#v, got %#v", expected.Spec, actual.Spec)
				}
				if actual.Spec.ExternalTrafficPolicy != coreapi.ServiceExternalTrafficPolicyLocal ||
					!reflect.DeepEqual(actual.Spec.ExternalIPs, []string{"192.0.2.1"}) {
					return fmt.Errorf("external IP Service lost Local traffic policy: %#v", actual.Spec)
				}
				t.Logf("Service %s/%s UID=%s ClusterIPs=%v type=%s externalIPs=%v externalTrafficPolicy=%s nodePort=%d",
					actual.Namespace, actual.Name, actual.UID, actual.Spec.ClusterIPs, actual.Spec.Type,
					actual.Spec.ExternalIPs, actual.Spec.ExternalTrafficPolicy, actual.Spec.Ports[0].NodePort)
				return nil
			}),
			resource.TestCheckResourceAttr(address, "spec.0.type", string(serviceType)),
			resource.TestCheckResourceAttr(address, "spec.0.external_ips.0", "192.0.2.1"),
			resource.TestCheckResourceAttr(address, "spec.0.external_traffic_policy", "Local"),
		)
	}
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: serviceAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesServiceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: serviceExternalIPTrafficConfig(name, "NodePort"), Check: check(coreapi.ServiceTypeNodePort),
				ConfigPlanChecks: serviceSettledPlanChecks(),
			},
			{
				Config: serviceExternalIPTrafficConfig(name, "ClusterIP"), Check: check(coreapi.ServiceTypeClusterIP),
				ConfigPlanChecks: serviceSettledPlanChecks(plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)),
			},
			{
				Config: serviceExternalIPTrafficConfig(name, "ClusterIP"), Check: check(coreapi.ServiceTypeClusterIP),
				ConfigPlanChecks: serviceSettledPlanChecks(plancheck.ExpectEmptyPlan()),
			},
		},
	})
}

func serviceNodePortCombinedEditConfig(name string, edited bool) string {
	port := "      port = 80"
	if edited {
		port = "      name = \"http\"\n      port = 81"
	}
	return fmt.Sprintf(`resource "kubernetes_service_v1" "test" {
  metadata {
    name = %q
  }
  spec {
    type = "NodePort"
    port {
%s
    }
  }
}
`, name, port)
}

func serviceExternalIPTrafficConfig(name, serviceType string) string {
	return fmt.Sprintf(`resource "kubernetes_service_v1" "test" {
  metadata {
    name = %q
  }
  spec {
    type                    = %q
    external_ips            = ["192.0.2.1"]
    external_traffic_policy = "Local"
    port {
      name        = "http"
      port        = 80
      target_port = 8080
    }
  }
}
`, name, serviceType)
}

// Live migration tests and migration guards.

const (
	serviceSDKRelease             = "3.2.1"
	serviceSDKPreIdentityRelease  = "2.37.1"
	serviceRegistryProviderSource = "registry.terraform.io/hashicorp/kubernetes"
)

// Each source is created by the exact registry release, not the local SDK alias.
// Configuration bodies are identical except for the approved affinity syntax
// adjustment; the stored list shape and every Service setting must stay unchanged.
func TestAccKubernetesServiceV1_migration(t *testing.T) {
	shapes := []struct {
		name   string
		config func(string) string
	}{
		{"ClusterIP", testAccKubernetesServiceV1Config_basic},
		{"NodePort", func(name string) string {
			return strings.Replace(testAccKubernetesServiceV1Config_basic(name),
				"  spec {\n", "  spec {\n    type = \"NodePort\"\n", 1)
		}},
		{"Headless", testAccKubernetesServiceV1Config_headless},
		{"ExternalName", testAccKubernetesServiceV1Config_externalName},
		// An explicit class keeps the default cloud controller from provisioning
		// infrastructure. This proves wait=false API/state behavior, not LB readiness.
		{"LoadBalancerWaitFalse", testAccKubernetesServiceV1Config_loadBalancer_class},
	}
	routes := []struct {
		name       string
		sourceType string
		targetName string
	}{
		{"AliasToV1", "kubernetes_service", "test"},
		{"V1Unchanged", "kubernetes_service_v1", "test"},
		{"V1AddressMove", "kubernetes_service_v1", "renamed"},
	}
	for _, release := range []string{serviceSDKRelease, serviceSDKPreIdentityRelease} {
		for _, route := range routes {
			for _, shape := range shapes {
				t.Run(release+"/"+route.name+"/"+shape.name, func(t *testing.T) {
					name := acctest.RandomWithPrefix("tf-service-migrate")
					serviceAcceptanceMigration(t, release, route.sourceType, route.targetName, name, shape.config(name))
				})
			}
		}
	}
}

// The registry release must receive its original block HCL. Only the Framework
// target uses the approved session_affinity_config/client_ip object assignments.
// The same complete-state, API-spec and pre-apply no-op assertions still apply.
func TestAccKubernetesServiceV1_migrationClientIP(t *testing.T) {
	for _, sourceType := range []string{"kubernetes_service", "kubernetes_service_v1"} {
		t.Run(sourceType, func(t *testing.T) {
			name := acctest.RandomWithPrefix("tf-service-affinity")
			serviceAcceptanceMigration(t, serviceSDKRelease, sourceType, "test", name,
				testAccKubernetesServiceV1Config_nodePort(name))
		})
	}
}

func serviceAcceptanceMigration(t *testing.T, release, sourceType, targetName, name, config string) {
	t.Helper()
	sourceAddress := sourceType + ".test"
	targetAddress := "kubernetes_service_v1." + targetName
	sourceConfig := serviceSDKAffinityConfig(strings.ReplaceAll(config, `"kubernetes_service_v1"`, `"`+sourceType+`"`))
	targetConfig := strings.Replace(config, `"kubernetes_service_v1" "test"`,
		`"kubernetes_service_v1" "`+targetName+`"`, 1)
	from := ""
	if sourceAddress != targetAddress {
		from = sourceAddress
		targetConfig += fmt.Sprintf(`
moved {
  from = %s
  to   = %s
}
`, sourceAddress, targetAddress)
	}
	var snapshot serviceMigrationSnapshot
	identityChecks := []statecheck.StateCheck{
		statecheck.ExpectIdentity(targetAddress, map[string]knownvalue.Check{
			"namespace":   knownvalue.StringExact("default"),
			"name":        knownvalue.StringExact(name),
			"api_version": knownvalue.StringExact("v1"),
			"kind":        knownvalue.StringExact("Service"),
		}),
		snapshot.stateCheck(targetAddress, false, release == serviceSDKPreIdentityRelease),
	}
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { serviceMigrationPreCheck(t) },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		CheckDestroy: testAccCheckKubernetesServiceV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						Source:            "hashicorp/kubernetes",
						VersionConstraint: "= " + release,
					},
				},
				Config: testAccKubernetesConfig_ignoreAnnotations() + sourceConfig,
				Check:  snapshot.checkRemote(sourceAddress, true),
				ConfigStateChecks: []statecheck.StateCheck{
					snapshot.stateCheck(sourceAddress, true, release == serviceSDKPreIdentityRelease),
				},
			},
			{
				ProtoV6ProviderFactories: serviceAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesConfig_ignoreAnnotations() + targetConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
						serviceMigrationNoOpPlan{address: targetAddress, previousAddress: from},
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check:             snapshot.checkRemote(targetAddress, false),
				ConfigStateChecks: identityChecks,
			},
			{
				ProtoV6ProviderFactories: serviceAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesConfig_ignoreAnnotations() + targetConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
						serviceMigrationNoOpPlan{address: targetAddress},
					},
				},
				Check:             snapshot.checkRemote(targetAddress, false),
				ConfigStateChecks: identityChecks,
			},
			{
				ProtoV6ProviderFactories: serviceAccProtoV6ProviderFactories,
				ResourceName:             targetAddress,
				ImportState:              true,
				ImportStateVerify:        true,
				// Preserve the original test's ignores: resourceVersion can race
				// controllers; wait is provider-only and not reconstructible by Read.
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "wait_for_load_balancer"},
			},
			{
				ProtoV6ProviderFactories: serviceAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesConfig_ignoreAnnotations() + targetConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
						serviceMigrationNoOpPlan{address: targetAddress},
					},
				},
				Check:             snapshot.checkRemote(targetAddress, false),
				ConfigStateChecks: identityChecks,
			},
		},
	})
}

const serviceFrameworkAffinityConfig = `    session_affinity_config = {
      client_ip = {
        timeout_seconds = 300
      }
    }`

const serviceSDKAffinityBlock = `    session_affinity_config {
      client_ip {
        timeout_seconds = 300
      }
    }`

func serviceSDKAffinityConfig(config string) string {
	return strings.ReplaceAll(config, serviceFrameworkAffinityConfig, serviceSDKAffinityBlock)
}

func TestServiceMigrationAffinitySyntaxOnly(t *testing.T) {
	for _, config := range []string{
		testAccKubernetesServiceV1Config_nodePort("test"),
		testAccKubernetesServiceV1Config_nodePort_toClusterIP("test"),
	} {
		if strings.Count(config, serviceFrameworkAffinityConfig) != 1 {
			t.Fatal("Framework fixture must contain exactly the approved affinity list assignment")
		}
		source := serviceSDKAffinityConfig(config)
		if strings.Count(source, serviceSDKAffinityBlock) != 1 {
			t.Fatal("SDK fixture must retain its original affinity blocks")
		}
		if restored := strings.ReplaceAll(source, serviceSDKAffinityBlock, serviceFrameworkAffinityConfig); restored != config {
			t.Fatal("affinity syntax adjustment changed other Service configuration")
		}
	}
}

func serviceMigrationPreCheck(t *testing.T) {
	t.Helper()
	testAccPreCheck(t)
	if os.Getenv("TF_REATTACH_PROVIDERS") != "" {
		t.Fatal("released-provider migration requires TF_REATTACH_PROVIDERS to be unset")
	}
	cliConfig := os.Getenv("TF_CLI_CONFIG_FILE")
	if cliConfig == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatal(err)
		}
		cliConfig = filepath.Join(home, ".terraformrc")
	}
	content, err := os.ReadFile(cliConfig)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "dev_overrides") {
		t.Fatal("released-provider migration requires a CLI configuration without dev_overrides")
	}
}

// This guard runs before apply, including for moved blocks. Checking only the
// post-apply UID would discover a replacement after the original Service was lost.
type serviceMigrationNoOpPlan struct {
	address         string
	previousAddress string
}

func (check serviceMigrationNoOpPlan) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	if req.Plan == nil {
		resp.Error = fmt.Errorf("missing migration plan")
		return
	}
	found := false
	for _, change := range req.Plan.ResourceChanges {
		if change.Mode == tfjson.DataResourceMode {
			continue
		}
		if found || change.Address != check.address || change.PreviousAddress != check.previousAddress ||
			change.Change == nil || !reflect.DeepEqual(change.Change.Actions, tfjson.Actions{tfjson.ActionNoop}) {
			resp.Error = fmt.Errorf("migration must only no-op %s (previous address %q), got %#v",
				check.address, check.previousAddress, change)
			return
		}
		if !reflect.DeepEqual(change.Change.Before, change.Change.After) {
			resp.Error = fmt.Errorf("migration changed planned Service values (-before +after):\n%s",
				cmp.Diff(change.Change.Before, change.Change.After))
			return
		}
		found = true
	}
	if !found {
		resp.Error = fmt.Errorf("migration plan omitted %s", check.address)
	}
}

type serviceMigrationSnapshot struct {
	object   *coreapi.Service
	values   map[string]interface{}
	identity map[string]interface{}
	provider string
}

func (snapshot *serviceMigrationSnapshot) checkRemote(address string, capture bool) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		var actual coreapi.Service
		if err := testAccCheckKubernetesServiceV1Exists(address, &actual)(state); err != nil {
			return err
		}
		if capture {
			snapshot.object = actual.DeepCopy()
		} else {
			if snapshot.object == nil {
				return fmt.Errorf("missing released-provider API snapshot")
			}
			if actual.UID != snapshot.object.UID {
				return fmt.Errorf("Service was replaced: UID %s -> %s", snapshot.object.UID, actual.UID)
			}
			if !reflect.DeepEqual(snapshot.object.Spec.ClusterIPs, actual.Spec.ClusterIPs) {
				return fmt.Errorf("Service ClusterIPs changed: %v -> %v", snapshot.object.Spec.ClusterIPs, actual.Spec.ClusterIPs)
			}
			if !reflect.DeepEqual(serviceNodePorts(snapshot.object), serviceNodePorts(&actual)) {
				return fmt.Errorf("Service NodePorts changed: %v -> %v", serviceNodePorts(snapshot.object), serviceNodePorts(&actual))
			}
			if diff := cmp.Diff(snapshot.object.Spec, actual.Spec); diff != "" {
				return fmt.Errorf("complete Service.Spec changed during migration (-before +after):\n%s", diff)
			}
			if !reflect.DeepEqual(snapshot.object.Labels, actual.Labels) ||
				!reflect.DeepEqual(snapshot.object.Annotations, actual.Annotations) ||
				snapshot.object.ResourceVersion != actual.ResourceVersion {
				return fmt.Errorf("migration wrote or changed the remote Service metadata")
			}
		}
		attributes := map[string]string{
			"id":                          actual.Namespace + "/" + actual.Name,
			"metadata.0.uid":              string(actual.UID),
			"spec.0.cluster_ip":           actual.Spec.ClusterIP,
			"spec.0.cluster_ips.#":        strconv.Itoa(len(actual.Spec.ClusterIPs)),
			"spec.0.port.#":               strconv.Itoa(len(actual.Spec.Ports)),
			"spec.0.type":                 string(actual.Spec.Type),
			"metadata.0.namespace":        actual.Namespace,
			"metadata.0.name":             actual.Name,
			"metadata.0.generation":       strconv.FormatInt(actual.Generation, 10),
			"metadata.0.resource_version": actual.ResourceVersion,
		}
		for index, ip := range actual.Spec.ClusterIPs {
			attributes[fmt.Sprintf("spec.0.cluster_ips.%d", index)] = ip
		}
		for index, port := range actual.Spec.Ports {
			attributes[fmt.Sprintf("spec.0.port.%d.node_port", index)] = strconv.Itoa(int(port.NodePort))
		}
		for key, value := range attributes {
			if err := resource.TestCheckResourceAttr(address, key, value)(state); err != nil {
				return err
			}
		}
		return nil
	}
}

func serviceNodePorts(service *coreapi.Service) []int32 {
	result := make([]int32, len(service.Spec.Ports))
	for index, port := range service.Spec.Ports {
		result[index] = port.NodePort
	}
	return result
}

type serviceStateCheckFunc func(context.Context, statecheck.CheckStateRequest, *statecheck.CheckStateResponse)

func (check serviceStateCheckFunc) CheckState(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	check(ctx, req, resp)
}

func (snapshot *serviceMigrationSnapshot) stateCheck(address string, capture, preIdentity bool) statecheck.StateCheck {
	return serviceStateCheckFunc(func(_ context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
		if req.State == nil || req.State.Values == nil || req.State.Values.RootModule == nil {
			resp.Error = fmt.Errorf("missing Terraform state")
			return
		}
		for _, actual := range req.State.Values.RootModule.Resources {
			if actual.Address != address {
				continue
			}
			if actual.ProviderName != serviceRegistryProviderSource {
				resp.Error = fmt.Errorf("provider source = %q, want %q", actual.ProviderName, serviceRegistryProviderSource)
				return
			}
			if capture {
				data, err := json.Marshal(actual)
				if err != nil {
					resp.Error = err
					return
				}
				var copy tfjson.StateResource
				decoder := json.NewDecoder(bytes.NewReader(data))
				decoder.UseNumber()
				if err := decoder.Decode(&copy); err != nil {
					resp.Error = err
					return
				}
				if err := serviceExpectedAffinityObjects(copy.AttributeValues); err != nil {
					resp.Error = err
					return
				}
				snapshot.values, snapshot.identity, snapshot.provider = copy.AttributeValues, copy.IdentityValues, copy.ProviderName
				if preIdentity && len(snapshot.identity) != 0 {
					resp.Error = fmt.Errorf("pre-identity release unexpectedly wrote identity: %v", snapshot.identity)
				}
				if !preIdentity && len(snapshot.identity) == 0 {
					resp.Error = fmt.Errorf("released provider did not write Service identity")
				}
				return
			}
			if snapshot.values == nil {
				resp.Error = fmt.Errorf("missing complete released-provider state snapshot")
				return
			}
			if actual.SchemaVersion != 2 {
				resp.Error = fmt.Errorf("local Service schema version = %d, want Framework version 2", actual.SchemaVersion)
				return
			}
			if !reflect.DeepEqual(snapshot.values, actual.AttributeValues) {
				resp.Error = fmt.Errorf("complete Terraform Service state changed (-released +local):\n%s",
					cmp.Diff(snapshot.values, actual.AttributeValues))
				return
			}
			if snapshot.provider != actual.ProviderName {
				resp.Error = fmt.Errorf("provider source changed during migration")
				return
			}
			if !preIdentity && !reflect.DeepEqual(snapshot.identity, actual.IdentityValues) {
				resp.Error = fmt.Errorf("resource identity changed (-released +local):\n%s",
					cmp.Diff(snapshot.identity, actual.IdentityValues))
			}
			return
		}
		resp.Error = fmt.Errorf("missing Service state at %s", address)
	})
}

func TestServiceMigrationNoOpPlanGuard(t *testing.T) {
	address := "kubernetes_service_v1.test"
	from := "kubernetes_service.test"
	for _, testCase := range []struct {
		name    string
		address string
		from    string
		actions tfjson.Actions
		after   interface{}
		wantErr bool
	}{
		{"no-op move", address, from, tfjson.Actions{tfjson.ActionNoop}, "same", false},
		{"replacement", address, from, tfjson.Actions{tfjson.ActionDelete, tfjson.ActionCreate}, "same", true},
		{"create-before-destroy replacement", address, from, tfjson.Actions{tfjson.ActionCreate, tfjson.ActionDelete}, "same", true},
		{"deletion", address, from, tfjson.Actions{tfjson.ActionDelete}, "same", true},
		{"update", address, from, tfjson.Actions{tfjson.ActionUpdate}, "same", true},
		{"missing move", address, "", tfjson.Actions{tfjson.ActionNoop}, "same", true},
		{"wrong address", "kubernetes_service_v1.other", from, tfjson.Actions{tfjson.ActionNoop}, "same", true},
		{"changed values", address, from, tfjson.Actions{tfjson.ActionNoop}, "changed", true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			check := serviceMigrationNoOpPlan{address: address, previousAddress: from}
			var response plancheck.CheckPlanResponse
			check.CheckPlan(context.Background(), plancheck.CheckPlanRequest{
				Plan: &tfjson.Plan{ResourceChanges: []*tfjson.ResourceChange{{
					Address: testCase.address, PreviousAddress: testCase.from, Mode: tfjson.ManagedResourceMode,
					Change: &tfjson.Change{Actions: testCase.actions, Before: "same", After: testCase.after},
				}}},
			}, &response)
			if (response.Error != nil) != testCase.wantErr {
				t.Fatalf("guard error = %v, want error %t", response.Error, testCase.wantErr)
			}
		})
	}
}

// Normalize only the approved historical list-to-object paths in the expected
// snapshot. Every other stored attribute and every identity field remains exact.
func serviceExpectedAffinityObjects(values map[string]interface{}) error {
	specs, _ := values["spec"].([]interface{})
	for _, raw := range specs {
		spec, ok := raw.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid snapshot spec")
		}
		for _, field := range []string{"session_affinity_config", "client_ip"} {
			value, exists := spec[field]
			if !exists {
				break
			}
			if list, ok := value.([]interface{}); ok {
				switch len(list) {
				case 0:
					value = nil
				case 1:
					value = list[0]
				default:
					return fmt.Errorf("invalid snapshot %s cardinality", field)
				}
				spec[field] = value
			}
			if value == nil {
				break
			}
			spec, ok = value.(map[string]interface{})
			if !ok {
				return fmt.Errorf("invalid snapshot %s object", field)
			}
		}
	}
	return nil
}

func TestServiceMigrationCompleteStateSnapshot(t *testing.T) {
	const address = "kubernetes_service_v1.test"
	makeState := func() *tfjson.State {
		return &tfjson.State{Values: &tfjson.StateValues{RootModule: &tfjson.StateModule{
			Resources: []*tfjson.StateResource{{
				Address: address, ProviderName: serviceRegistryProviderSource, SchemaVersion: 2,
				AttributeValues: map[string]interface{}{
					"id": "default/test",
					"spec": []interface{}{map[string]interface{}{
						"cluster_ips": []interface{}{"10.96.0.2"},
						"port": []interface{}{map[string]interface{}{
							"port": json.Number("80"), "node_port": json.Number("32000"),
						}},
						"selector": nil,
					}},
				},
				IdentityValues: map[string]interface{}{
					"namespace": "default", "name": "test", "kind": "Service", "api_version": "v1",
				},
			}},
		}}}
	}
	for _, testCase := range []struct {
		name    string
		change  func(*tfjson.StateResource)
		wantErr bool
	}{
		{"unchanged", func(_ *tfjson.StateResource) {}, false},
		{"other attribute lost", func(state *tfjson.StateResource) { delete(state.AttributeValues, "spec") }, true},
		{"identity changed", func(state *tfjson.StateResource) { state.IdentityValues["name"] = "other" }, true},
		{"provider changed", func(state *tfjson.StateResource) { state.ProviderName = "example.com/hashicorp/kubernetes" }, true},
		{"still routed to SDK", func(state *tfjson.StateResource) { state.SchemaVersion = 0 }, true},
		{"null changed to empty", func(state *tfjson.StateResource) {
			state.AttributeValues["spec"].([]interface{})[0].(map[string]interface{})["selector"] = map[string]interface{}{}
		}, true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var snapshot serviceMigrationSnapshot
			captured := makeState()
			var response statecheck.CheckStateResponse
			snapshot.stateCheck(address, true, false).CheckState(context.Background(),
				statecheck.CheckStateRequest{State: captured}, &response)
			if response.Error != nil {
				t.Fatal(response.Error)
			}
			// The snapshot must own its maps rather than referring to mutable input.
			captured.Values.RootModule.Resources[0].AttributeValues["id"] = "mutated"
			actual := makeState()
			testCase.change(actual.Values.RootModule.Resources[0])
			response = statecheck.CheckStateResponse{}
			snapshot.stateCheck(address, false, false).CheckState(context.Background(),
				statecheck.CheckStateRequest{State: actual}, &response)
			if (response.Error != nil) != testCase.wantErr {
				t.Fatalf("state check error = %v, want error %t", response.Error, testCase.wantErr)
			}
		})
	}
}

// Live traffic-continuity tests and safety guards.

func TestAccKubernetesServiceV1_trafficContinuity(t *testing.T) {
	for _, route := range []struct {
		name       string
		sourceType string
	}{
		{"V1Unchanged", "kubernetes_service_v1"},
		{"AliasToV1", "kubernetes_service"},
	} {
		t.Run(route.name, func(t *testing.T) {
			name := acctest.RandomWithPrefix("tf-service-flow")
			sourceAddress := route.sourceType + ".test"
			targetAddress := "kubernetes_service_v1.test"
			source := serviceConnectivityConfig(name, route.sourceType, true)
			target := serviceConnectivityConfig(name, "kubernetes_service_v1", true)
			previousAddress := ""
			if sourceAddress != targetAddress {
				previousAddress = sourceAddress
				target += `
moved {
  from = kubernetes_service.test
  to   = kubernetes_service_v1.test
}
`
			}
			snapshot := &serviceConnectivitySnapshot{name: name, t: t}
			released := map[string]resource.ExternalProvider{
				"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= " + serviceSDKRelease},
			}
			resource.ParallelTest(t, resource.TestCase{
				PreCheck: func() { serviceMigrationPreCheck(t) },
				TerraformVersionChecks: []tfversion.TerraformVersionCheck{
					tfversion.SkipBelow(tfversion.Version1_12_0),
				},
				CheckDestroy: serviceConnectivityCheckDestroy,
				Steps: []resource.TestStep{
					{
						ExternalProviders: released,
						Config:            serviceConnectivityConfig(name, route.sourceType, false),
						// Pod Create waits for Running, not Ready. Establish a ready
						// endpoint before Terraform creates the fail-fast probe Pod.
						Check: snapshot.prepare(sourceAddress),
					},
					{
						ExternalProviders: released,
						Config:            source,
						Check:             snapshot.checkTraffic(sourceAddress),
					},
					{
						ProtoV6ProviderFactories: serviceAccProtoV6ProviderFactories,
						Config:                   target,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{
								plancheck.ExpectEmptyPlan(),
								serviceConnectivityNoOpPlan{address: targetAddress, previousAddress: previousAddress},
								serviceConnectivityLivePlanCheck{snapshot: snapshot},
							},
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: snapshot.checkTraffic(targetAddress),
					},
				},
			})
		})
	}
}

type serviceConnectivitySnapshot struct {
	name           string
	t              *testing.T
	service        *coreapi.Service
	backendUID     k8stypes.UID
	probeUID       k8stypes.UID
	latestRequests int64
}

func (snapshot *serviceConnectivitySnapshot) prepare(address string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		client, err := mainClientset()
		if err != nil {
			return err
		}
		err = wait.PollUntilContextTimeout(context.Background(), time.Second, 2*time.Minute, true,
			func(ctx context.Context) (bool, error) {
				backend, err := client.CoreV1().Pods("default").Get(ctx, snapshot.name+"-backend", metav1.GetOptions{})
				if err != nil {
					return false, err
				}
				ready, err := serviceConnectivityPodReady(backend, "backend")
				if err != nil || !ready {
					return false, err
				}
				service, err := client.CoreV1().Services("default").Get(ctx, snapshot.name, metav1.GetOptions{})
				if err != nil {
					return false, err
				}
				if service.Spec.ClusterIP == "" || service.Spec.ClusterIP == "None" {
					return false, fmt.Errorf("traffic test requires an allocated ClusterIP")
				}
				slices, err := client.DiscoveryV1().EndpointSlices("default").List(ctx, metav1.ListOptions{
					LabelSelector: "kubernetes.io/service-name=" + snapshot.name,
				})
				if err != nil {
					return false, err
				}
				for _, slice := range slices.Items {
					for _, endpoint := range slice.Endpoints {
						if endpoint.Conditions.Ready != nil && *endpoint.Conditions.Ready &&
							endpoint.TargetRef != nil && endpoint.TargetRef.UID == backend.UID &&
							len(endpoint.Addresses) > 0 {
							snapshot.service = service.DeepCopy()
							snapshot.backendUID = backend.UID
							return true, nil
						}
					}
				}
				return false, nil
			})
		if err != nil {
			return fmt.Errorf("waiting for backend readiness and Service endpoint: %w", err)
		}
		return resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr(address, "metadata.0.uid", string(snapshot.service.UID)),
			resource.TestCheckResourceAttr(address, "spec.0.cluster_ip", snapshot.service.Spec.ClusterIP),
			resource.TestCheckResourceAttr("kubernetes_pod_v1.backend", "metadata.0.uid", string(snapshot.backendUID)),
		)(state)
	}
}

func (snapshot *serviceConnectivitySnapshot) observeTraffic(ctx context.Context) error {
	if snapshot.service == nil || snapshot.backendUID == "" {
		return fmt.Errorf("missing pre-migration Service/backend snapshot")
	}
	client, err := mainClientset()
	if err != nil {
		return err
	}
	minimum := snapshot.latestRequests + 5
	return wait.PollUntilContextTimeout(ctx, time.Second, 2*time.Minute, true,
		func(ctx context.Context) (bool, error) {
			service, err := client.CoreV1().Services("default").Get(ctx, snapshot.name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			if service.UID != snapshot.service.UID || service.Spec.ClusterIP != snapshot.service.Spec.ClusterIP {
				return false, fmt.Errorf("Service UID/ClusterIP changed: %s/%s -> %s/%s",
					snapshot.service.UID, snapshot.service.Spec.ClusterIP, service.UID, service.Spec.ClusterIP)
			}
			if diff := cmp.Diff(snapshot.service.Spec, service.Spec); diff != "" {
				return false, fmt.Errorf("Service.Spec changed during traffic probe (-before +after):\n%s", diff)
			}
			for _, role := range []string{"backend", "probe"} {
				pod, err := client.CoreV1().Pods("default").Get(ctx, snapshot.name+"-"+role, metav1.GetOptions{})
				if err != nil {
					return false, err
				}
				expectedUID := snapshot.backendUID
				if role == "probe" {
					if snapshot.probeUID == "" {
						snapshot.probeUID = pod.UID
					}
					expectedUID = snapshot.probeUID
				}
				if pod.UID != expectedUID {
					return false, fmt.Errorf("%s Pod was replaced: %s -> %s", role, expectedUID, pod.UID)
				}
				ready, err := serviceConnectivityPodReady(pod, role)
				if err != nil || !ready {
					return false, err
				}
			}
			logs, err := client.CoreV1().Pods("default").GetLogs(snapshot.name+"-probe", &coreapi.PodLogOptions{
				Container: "probe", LimitBytes: ptr.To[int64](1024 * 1024),
			}).DoRaw(ctx)
			if err != nil {
				return false, err
			}
			count, err := serviceConnectivityRequestCount(string(logs), snapshot.service.Spec.ClusterIP)
			if err != nil {
				return false, err
			}
			if count < minimum {
				return false, nil
			}
			snapshot.t.Logf("ClusterIP %s: %d consecutive successful requests; Service/backend/probe UIDs %s/%s/%s",
				snapshot.service.Spec.ClusterIP, count, snapshot.service.UID, snapshot.backendUID, snapshot.probeUID)
			snapshot.latestRequests = count
			return true, nil
		})
}

func (snapshot *serviceConnectivitySnapshot) checkTraffic(address string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		if err := snapshot.observeTraffic(context.Background()); err != nil {
			return err
		}
		return resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr(address, "metadata.0.uid", string(snapshot.service.UID)),
			resource.TestCheckResourceAttr(address, "spec.0.cluster_ip", snapshot.service.Spec.ClusterIP),
			resource.TestCheckResourceAttr("kubernetes_pod_v1.backend", "metadata.0.uid", string(snapshot.backendUID)),
			resource.TestCheckResourceAttr("kubernetes_pod_v1.probe", "metadata.0.uid", string(snapshot.probeUID)),
		)(state)
	}
}

func serviceConnectivityPodReady(pod *coreapi.Pod, container string) (bool, error) {
	if pod.Spec.RestartPolicy != coreapi.RestartPolicyNever || pod.UID == "" {
		return false, fmt.Errorf("%s Pod must have a UID and restartPolicy Never", container)
	}
	if pod.DeletionTimestamp != nil || pod.Status.Phase == coreapi.PodFailed || pod.Status.Phase == coreapi.PodSucceeded {
		return false, fmt.Errorf("%s Pod is exiting: phase=%s reason=%s", container, pod.Status.Phase, pod.Status.Reason)
	}
	for _, status := range pod.Status.ContainerStatuses {
		if status.Name != container {
			continue
		}
		if status.RestartCount != 0 || status.State.Terminated != nil || status.LastTerminationState.Terminated != nil {
			return false, fmt.Errorf("%s Pod exited or restarted: %#v", container, status)
		}
		if status.State.Running == nil || !status.Ready {
			return false, nil
		}
		for _, condition := range pod.Status.Conditions {
			if condition.Type == coreapi.PodReady && condition.Status == coreapi.ConditionTrue {
				return pod.Status.Phase == coreapi.PodRunning, nil
			}
		}
	}
	return false, nil
}

func serviceConnectivityRequestCount(logs, clusterIP string) (int64, error) {
	var count int64
	for _, line := range strings.Split(strings.TrimSpace(logs), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 || fields[0] != "OK" || fields[2] != clusterIP {
			return 0, fmt.Errorf("traffic probe failed or emitted an unexpected record: %q", line)
		}
		next, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil || next != count+1 {
			return 0, fmt.Errorf("traffic probe request sequence interrupted at %q", line)
		}
		count = next
	}
	return count, nil
}

type serviceConnectivityLivePlanCheck struct{ snapshot *serviceConnectivitySnapshot }

func (check serviceConnectivityLivePlanCheck) CheckPlan(ctx context.Context, _ plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	resp.Error = check.snapshot.observeTraffic(ctx)
}

type serviceConnectivityNoOpPlan struct {
	address         string
	previousAddress string
}

func (check serviceConnectivityNoOpPlan) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	expected := map[string]string{
		check.address:               check.previousAddress,
		"kubernetes_pod_v1.backend": "",
		"kubernetes_pod_v1.probe":   "",
	}
	if req.Plan == nil {
		resp.Error = fmt.Errorf("missing traffic-continuity migration plan")
		return
	}
	// A Service update can make the dependent probe's environment unknown and
	// force its replacement. Report the Service cause before the downstream symptom.
	for _, change := range req.Plan.ResourceChanges {
		if change.Mode != tfjson.ManagedResourceMode || change.Address != check.address {
			continue
		}
		if change.Change == nil {
			resp.Error = fmt.Errorf("Service %s has no change details", change.Address)
			return
		}
		if change.PreviousAddress != check.previousAddress ||
			!reflect.DeepEqual(change.Change.Actions, tfjson.Actions{tfjson.ActionNoop}) ||
			!reflect.DeepEqual(change.Change.Before, change.Change.After) {
			unknown, err := json.MarshalIndent(change.Change.AfterUnknown, "", "  ")
			if err != nil {
				resp.Error = fmt.Errorf("encoding Service plan unknown values: %w", err)
				return
			}
			resp.Error = fmt.Errorf("Service %s must be no-op (previous address %q); actions=%v previous_address=%q\nBefore/After (-before +after):\n%s\nAfterUnknown:\n%s",
				change.Address, check.previousAddress, change.Change.Actions, change.PreviousAddress,
				cmp.Diff(change.Change.Before, change.Change.After), unknown)
			return
		}
	}
	for _, change := range req.Plan.ResourceChanges {
		if change.Mode != tfjson.ManagedResourceMode {
			continue
		}
		previous, found := expected[change.Address]
		if !found || change.PreviousAddress != previous || change.Change == nil ||
			!reflect.DeepEqual(change.Change.Actions, tfjson.Actions{tfjson.ActionNoop}) ||
			!reflect.DeepEqual(change.Change.Before, change.Change.After) {
			resp.Error = fmt.Errorf("traffic migration requires unchanged Service and both Pods, got %#v", change)
			return
		}
		delete(expected, change.Address)
	}
	if len(expected) != 0 {
		resp.Error = fmt.Errorf("traffic migration omitted managed resources: %v", expected)
	}
}

func serviceConnectivityCheckDestroy(state *terraform.State) error {
	client, err := mainClientset()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	for _, instance := range state.RootModule().Resources {
		if instance.Type != "kubernetes_pod_v1" && instance.Type != "kubernetes_service" &&
			instance.Type != "kubernetes_service_v1" {
			continue
		}
		namespace, name, err := kubernetes.IdParts(instance.Primary.ID)
		if err != nil {
			return err
		}
		if instance.Type == "kubernetes_pod_v1" {
			_, err = client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		} else {
			_, err = client.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
		}
		if apierrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("checking %s %s destruction: %w", instance.Type, instance.Primary.ID, err)
		}
		return fmt.Errorf("%s still exists: %s", instance.Type, instance.Primary.ID)
	}
	return nil
}

func serviceConnectivityConfig(name, serviceType string, includeProbe bool) string {
	config := fmt.Sprintf(`
resource "kubernetes_pod_v1" "backend" {
  metadata {
    name      = "%[1]s-backend"
    namespace = "default"
    labels = {
      "test-service" = "%[1]s"
    }
  }
  spec {
    restart_policy                   = "Never"
    termination_grace_period_seconds = 1
    container {
      name              = "backend"
      image             = "busybox:1.36"
      image_pull_policy = "IfNotPresent"
      command           = ["/bin/sh", "-ec"]
      args = [<<-SCRIPT
        mkdir -p /www
        printf '%%s\n' "$RESPONSE_TOKEN" > /www/index.html
        exec httpd -f -p 8080 -h /www
      SCRIPT
      ]
      env {
        name  = "RESPONSE_TOKEN"
        value = "%[1]s"
      }
      readiness_probe {
        http_get {
          path = "/"
          port = 8080
        }
        period_seconds  = 1
        timeout_seconds = 1
      }
    }
  }
  timeouts {
    create = "3m"
    delete = "90s"
  }
}

resource "%[2]s" "test" {
  metadata {
    name      = "%[1]s"
    namespace = "default"
  }
  spec {
    selector = kubernetes_pod_v1.backend.metadata[0].labels
    port {
      port        = 80
      target_port = 8080
    }
  }
}
`, name, serviceType)
	if !includeProbe {
		return config
	}
	return config + fmt.Sprintf(`
resource "kubernetes_pod_v1" "probe" {
  metadata {
    name      = "%[1]s-probe"
    namespace = "default"
  }
  spec {
    restart_policy                   = "Never"
    termination_grace_period_seconds = 1
    container {
      name              = "probe"
      image             = "busybox:1.36"
      image_pull_policy = "IfNotPresent"
      command           = ["/bin/sh", "-ec"]
      args = [<<-SCRIPT
        n=0
        host="$SERVICE_IP"
        case "$host" in *:*) host="[$host]" ;; esac
        while true; do
          if ! body="$(wget -Y off -T 2 -q -O - "http://$host:80/")"; then
            echo "FAIL request"
            exit 1
          fi
          if [ "$body" != "$EXPECTED_TOKEN" ]; then
            echo "FAIL response"
            exit 1
          fi
          n=$((n + 1))
          printf 'OK %%s %%s\n' "$n" "$SERVICE_IP"
          sleep 0.2
        done
      SCRIPT
      ]
      env {
        name  = "SERVICE_IP"
        value = %[2]s.test.spec[0].cluster_ip
      }
      env {
        name  = "EXPECTED_TOKEN"
        value = "%[1]s"
      }
    }
  }
  timeouts {
    create = "3m"
    delete = "90s"
  }
}
`, name, serviceType)
}

func TestServiceConnectivityNoOpPlanGuard(t *testing.T) {
	for _, changed := range []string{"", "kubernetes_service_v1.test", "kubernetes_pod_v1.backend", "kubernetes_pod_v1.probe"} {
		t.Run("changed="+changed, func(t *testing.T) {
			plan := &tfjson.Plan{}
			for _, address := range []string{"kubernetes_service_v1.test", "kubernetes_pod_v1.backend", "kubernetes_pod_v1.probe"} {
				change := &tfjson.ResourceChange{
					Address: address, Mode: tfjson.ManagedResourceMode,
					Change: &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionNoop}, Before: "unchanged", After: "unchanged"},
				}
				if address == "kubernetes_service_v1.test" {
					change.PreviousAddress = "kubernetes_service.test"
				}
				if address == changed {
					change.Change.Actions = tfjson.Actions{tfjson.ActionDelete, tfjson.ActionCreate}
				}
				plan.ResourceChanges = append(plan.ResourceChanges, change)
			}
			check := serviceConnectivityNoOpPlan{address: "kubernetes_service_v1.test", previousAddress: "kubernetes_service.test"}
			var response plancheck.CheckPlanResponse
			check.CheckPlan(context.Background(), plancheck.CheckPlanRequest{Plan: plan}, &response)
			if (response.Error != nil) != (changed != "") {
				t.Fatalf("guard error = %v, changed resource = %q", response.Error, changed)
			}
		})
	}
}

func TestServiceConnectivityProbeLogSequence(t *testing.T) {
	for _, testCase := range []struct {
		logs    string
		want    int64
		wantErr bool
	}{
		{"", 0, false},
		{"OK 1 10.96.0.2\nOK 2 10.96.0.2\n", 2, false},
		{"OK 1 10.96.0.2\nFAIL request\n", 0, true},
		{"OK 1 10.96.0.2\nOK 3 10.96.0.2\n", 0, true},
		{"OK 1 10.96.0.3\n", 0, true},
	} {
		count, err := serviceConnectivityRequestCount(testCase.logs, "10.96.0.2")
		if count != testCase.want || (err != nil) != testCase.wantErr {
			t.Fatalf("logs %q: count=%d error=%v, want count=%d error=%t", testCase.logs, count, err, testCase.want, testCase.wantErr)
		}
	}
}

func TestServiceConnectivityServiceChangeDiagnostic(t *testing.T) {
	address := "kubernetes_service_v1.test"
	request := plancheck.CheckPlanRequest{Plan: &tfjson.Plan{ResourceChanges: []*tfjson.ResourceChange{
		{
			Address: "kubernetes_pod_v1.probe", Mode: tfjson.ManagedResourceMode,
			Change: &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionDelete, tfjson.ActionCreate}},
		},
		{
			Address: address, Mode: tfjson.ManagedResourceMode,
			Change: &tfjson.Change{
				Actions:      tfjson.Actions{tfjson.ActionUpdate},
				Before:       map[string]interface{}{"cluster_ip": "10.96.0.2"},
				After:        map[string]interface{}{"cluster_ip": nil},
				AfterUnknown: map[string]interface{}{"cluster_ip": true},
			},
		},
	}}}
	var response plancheck.CheckPlanResponse
	serviceConnectivityNoOpPlan{address: address}.CheckPlan(context.Background(), request, &response)
	if response.Error == nil {
		t.Fatal("Service update and dependent probe replacement were accepted")
	}
	for _, want := range []string{"Service " + address, "actions=[update]", "Before/After", "10.96.0.2", "AfterUnknown", `"cluster_ip": true`} {
		if !strings.Contains(response.Error.Error(), want) {
			t.Errorf("diagnostic lacks %q: %s", want, response.Error)
		}
	}
}

func TestServiceConnectivityConfigSyntax(t *testing.T) {
	for _, serviceType := range []string{"kubernetes_service", "kubernetes_service_v1"} {
		for _, includeProbe := range []bool{false, true} {
			config := serviceConnectivityConfig("offline-probe", serviceType, includeProbe)
			if _, diags := hclsyntax.ParseConfig([]byte(config), "traffic.tf", hcl.InitialPos); diags.HasErrors() {
				t.Fatalf("%s (probe=%t): %s", serviceType, includeProbe, diags.Error())
			}
		}
	}
	source := serviceConnectivityConfig("offline-probe", "kubernetes_service", true)
	target := serviceConnectivityConfig("offline-probe", "kubernetes_service_v1", true)
	changedType := strings.ReplaceAll(source, `"kubernetes_service"`, `"kubernetes_service_v1"`)
	changedReference := strings.ReplaceAll(changedType, "kubernetes_service.test", "kubernetes_service_v1.test")
	if changedReference != target {
		t.Fatal("alias move changed configuration beyond the Service type and reference")
	}
}

func TestServiceConnectivityPodExitGuard(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		mutate func(*coreapi.Pod)
	}{
		{"terminated", func(pod *coreapi.Pod) {
			pod.Status.ContainerStatuses[0].State.Terminated = &coreapi.ContainerStateTerminated{ExitCode: 1}
		}},
		{"restarted", func(pod *coreapi.Pod) { pod.Status.ContainerStatuses[0].RestartCount = 1 }},
		{"previous exit", func(pod *coreapi.Pod) {
			pod.Status.ContainerStatuses[0].LastTerminationState.Terminated = &coreapi.ContainerStateTerminated{}
		}},
		{"completed", func(pod *coreapi.Pod) { pod.Status.Phase = coreapi.PodSucceeded }},
		{"failed", func(pod *coreapi.Pod) { pod.Status.Phase = coreapi.PodFailed }},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			pod := &coreapi.Pod{
				ObjectMeta: metav1.ObjectMeta{UID: "probe-uid"},
				Spec:       coreapi.PodSpec{RestartPolicy: coreapi.RestartPolicyNever},
				Status: coreapi.PodStatus{
					Phase:      coreapi.PodRunning,
					Conditions: []coreapi.PodCondition{{Type: coreapi.PodReady, Status: coreapi.ConditionTrue}},
					ContainerStatuses: []coreapi.ContainerStatus{{
						Name: "probe", Ready: true, State: coreapi.ContainerState{Running: &coreapi.ContainerStateRunning{}},
					}},
				},
			}
			if ready, err := serviceConnectivityPodReady(pod, "probe"); !ready || err != nil {
				t.Fatalf("healthy probe was not ready: %v", err)
			}
			testCase.mutate(pod)
			if ready, err := serviceConnectivityPodReady(pod, "probe"); ready || err == nil {
				t.Fatalf("exited/restarted probe accepted: ready=%t error=%v", ready, err)
			}
		})
	}
}

// Registration, state migration, and import unit tests.

func serviceRegisteredFrameworkResource(t *testing.T) fwresource.Resource {
	t.Helper()
	for _, constructor := range provider.New("test", nil).Resources(context.Background()) {
		r := constructor()
		var metadata fwresource.MetadataResponse
		r.Metadata(context.Background(), fwresource.MetadataRequest{ProviderTypeName: "kubernetes"}, &metadata)
		if metadata.TypeName == "kubernetes_service_v1" {
			return r
		}
	}
	t.Fatal("kubernetes_service_v1 is not registered on the Framework server")
	return nil
}

func TestServiceFrameworkRegistration(t *testing.T) {
	ctx := context.Background()
	sdk := kubernetes.Provider()
	if _, ok := sdk.ResourcesMap["kubernetes_service_v1"]; ok {
		t.Error("versioned Service must not remain registered on SDKv2")
	}
	if _, ok := sdk.ResourcesMap["kubernetes_service"]; !ok {
		t.Error("deprecated Service alias must remain registered on SDKv2")
	}
	for _, name := range []string{"kubernetes_service", "kubernetes_service_v1"} {
		if _, ok := sdk.DataSourcesMap[name]; !ok {
			t.Errorf("Service data source %q must remain registered", name)
		}
	}
	r := serviceRegisteredFrameworkResource(t)
	var schema fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schema)
	if schema.Diagnostics.HasError() {
		t.Fatal(schema.Diagnostics)
	}
	if schema.Schema.Version != 2 {
		t.Fatalf("Service schema version = %d, want version 2", schema.Schema.Version)
	}

	server, err := mux.MuxServer(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	response, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range response.Diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("mux schema error: %s: %s", diagnostic.Summary, diagnostic.Detail)
		}
	}
	legacy, migrated := response.ResourceSchemas["kubernetes_service"], response.ResourceSchemas["kubernetes_service_v1"]
	if legacy == nil || migrated == nil {
		t.Fatal("production mux must expose both Service resource types")
	}
	if legacy.Version != 1 || migrated.Version != 2 {
		t.Fatalf("legacy/current schema versions = %d/%d", legacy.Version, migrated.Version)
	}

}

func TestServiceUnknownConfiguration(t *testing.T) {
	ctx := context.Background()
	server, err := mux.MuxServer(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	resourceSchema := schemas.ResourceSchemas["kubernetes_service_v1"]
	if resourceSchema == nil {
		t.Fatal("missing Service schema")
	}
	objectType, ok := resourceSchema.ValueType().(tftypes.Object)
	if !ok {
		t.Fatal("Service schema is not an object")
	}
	values := make(map[string]tftypes.Value, len(objectType.AttributeTypes))
	for name, valueType := range objectType.AttributeTypes {
		values[name] = tftypes.NewValue(valueType, nil)
	}
	for _, name := range []string{"metadata", "spec"} {
		values[name] = tftypes.NewValue(objectType.AttributeTypes[name], tftypes.UnknownValue)
	}
	config, err := tfprotov6.NewDynamicValue(objectType, tftypes.NewValue(objectType, values))
	if err != nil {
		t.Fatal(err)
	}
	response, err := server.ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{
		TypeName: "kubernetes_service_v1", Config: &config,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range response.Diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			t.Errorf("unknown configuration must defer validation: %s: %s", diagnostic.Summary, diagnostic.Detail)
		}
	}
}

func TestServiceMoveStatePreservesNetworkIdentity(t *testing.T) {
	ctx := context.Background()
	r := serviceRegisteredFrameworkResource(t)
	var schema fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schema)
	var identitySchema fwresource.IdentitySchemaResponse
	r.(fwresource.ResourceWithIdentity).IdentitySchema(ctx, fwresource.IdentitySchemaRequest{}, &identitySchema)
	if identitySchema.IdentitySchema.Version != common.IdentitySchemaVersion {
		t.Fatalf("identity version = %d, want %d", identitySchema.IdentitySchema.Version, common.IdentitySchemaVersion)
	}
	movers := r.(fwresource.ResourceWithMoveState).MoveState(ctx)
	for _, version := range []int64{0, 1} {
		for _, host := range []string{"registry.terraform.io", "mirror.example.com"} {
			for _, withIdentity := range []bool{false, true} {
				t.Run(fmt.Sprintf("v%d/%s/identity-%t", version, host, withIdentity), func(t *testing.T) {
					raw := serviceStoredFixture(t, version)
					var response fwresource.MoveStateResponse
					for _, mover := range movers {
						response = fwresource.MoveStateResponse{TargetState: tfsdk.State{Schema: schema.Schema}}
						if withIdentity {
							response.TargetIdentity = &tfsdk.ResourceIdentity{Schema: identitySchema.IdentitySchema}
						}
						mover.StateMover(ctx, fwresource.MoveStateRequest{
							SourceTypeName:        "kubernetes_service",
							SourceSchemaVersion:   version,
							SourceProviderAddress: host + "/hashicorp/kubernetes",
							SourceRawState:        raw,
						}, &response)
						if response.Diagnostics.HasError() {
							t.Fatal(response.Diagnostics)
						}
						if !response.TargetState.Raw.IsNull() {
							break
						}
					}
					if response.TargetState.Raw.IsNull() {
						t.Fatal("supported Service source did not produce target state")
					}
					serviceCheckConvertedState(t, response.TargetState, version)
					if withIdentity {
						var identity common.NamespacedResourceIdentity
						if diags := response.TargetIdentity.Get(ctx, &identity); diags.HasError() {
							t.Fatal(diags)
						}
						if identity.Name.ValueString() != "svc-abc" || identity.Namespace.ValueString() != "apps" ||
							identity.Kind.ValueString() != "Service" || identity.APIVersion.ValueString() != "v1" {
							t.Fatalf("moved identity is incorrect: %#v", identity)
						}
					}
				})
			}
		}
	}
}

func TestServiceMoveStateRejectsInvalidSources(t *testing.T) {
	ctx := context.Background()
	r := serviceRegisteredFrameworkResource(t)
	var schema fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schema)
	for _, tc := range []struct {
		name, resourceType, provider string
		version                      int64
		raw                          *tfprotov6.RawState
		wantError                    bool
	}{
		{"wrong type", "kubernetes_pod", "registry.terraform.io/hashicorp/kubernetes", 1, serviceStoredFixture(t, 1), false},
		{"same type uses upgrade", "kubernetes_service_v1", "registry.terraform.io/hashicorp/kubernetes", 1, serviceStoredFixture(t, 1), false},
		{"future version", "kubernetes_service", "registry.terraform.io/hashicorp/kubernetes", 2, serviceStoredFixture(t, 1), false},
		{"negative version", "kubernetes_service", "registry.terraform.io/hashicorp/kubernetes", -1, serviceStoredFixture(t, 1), false},
		{"other provider", "kubernetes_service", "registry.terraform.io/other/kubernetes", 1, serviceStoredFixture(t, 1), false},
		{"provider suffix lookalike", "kubernetes_service", "registry.terraform.io/nothashicorp/kubernetes", 1, serviceStoredFixture(t, 1), false},
		{"missing raw", "kubernetes_service", "registry.terraform.io/hashicorp/kubernetes", 1, nil, true},
		{"malformed JSON", "kubernetes_service", "registry.terraform.io/hashicorp/kubernetes", 1, &tfprotov6.RawState{JSON: []byte("{")}, true},
		{"null JSON", "kubernetes_service", "registry.terraform.io/hashicorp/kubernetes", 1, &tfprotov6.RawState{JSON: []byte("null")}, true},
		{"missing ID", "kubernetes_service", "registry.terraform.io/hashicorp/kubernetes", 1, &tfprotov6.RawState{JSON: []byte(`{"metadata":[{"name":"svc","namespace":"apps"}]}`)}, true},
		{"ID without namespace", "kubernetes_service", "registry.terraform.io/hashicorp/kubernetes", 1, &tfprotov6.RawState{JSON: []byte(`{"id":"svc","metadata":[{"name":"svc","namespace":"apps"}]}`)}, true},
		{"mismatched identity", "kubernetes_service", "registry.terraform.io/hashicorp/kubernetes", 1, &tfprotov6.RawState{JSON: []byte(`{"id":"apps/svc","metadata":[{"name":"different","namespace":"apps"}]}`)}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hadError := false
			for _, mover := range r.(fwresource.ResourceWithMoveState).MoveState(ctx) {
				response := fwresource.MoveStateResponse{TargetState: tfsdk.State{Schema: schema.Schema}}
				mover.StateMover(ctx, fwresource.MoveStateRequest{
					SourceTypeName: tc.resourceType, SourceSchemaVersion: tc.version,
					SourceProviderAddress: tc.provider, SourceRawState: tc.raw,
				}, &response)
				hadError = hadError || response.Diagnostics.HasError()
				if !response.TargetState.Raw.IsNull() {
					t.Fatal("invalid or unsupported source produced target state")
				}
			}
			if hadError != tc.wantError {
				t.Fatalf("error = %t, want %t", hadError, tc.wantError)
			}
		})
	}
}

func TestServiceHistoricalStateUpgrade(t *testing.T) {
	ctx := context.Background()
	r := serviceRegisteredFrameworkResource(t)
	var schema fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schema)
	upgraders := r.(fwresource.ResourceWithUpgradeState).UpgradeState(ctx)
	upgrader, ok := upgraders[0]
	if !ok {
		t.Fatal("missing direct historical schema v0 upgrader")
	}
	response := fwresource.UpgradeStateResponse{State: tfsdk.State{Schema: schema.Schema}}
	upgrader.StateUpgrader(ctx, fwresource.UpgradeStateRequest{RawState: serviceStoredFixture(t, 0)}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	if response.DynamicValue != nil {
		raw, err := response.DynamicValue.Unmarshal(schema.Schema.Type().TerraformType(ctx))
		if err != nil {
			t.Fatal(err)
		}
		response.State.Raw = raw
	}
	serviceCheckConvertedState(t, response.State, 0)
}

func TestServiceImportIdentifiers(t *testing.T) {
	ctx := context.Background()
	r := serviceRegisteredFrameworkResource(t)
	var schema fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schema)
	var identitySchema fwresource.IdentitySchemaResponse
	r.(fwresource.ResourceWithIdentity).IdentitySchema(ctx, fwresource.IdentitySchemaRequest{}, &identitySchema)
	for _, tc := range []struct {
		name, id, namespace, wantID string
		byIdentity, wantError       bool
	}{
		{name: "ID", id: "apps/svc", wantID: "apps/svc"},
		{name: "identity", namespace: "apps", wantID: "apps/svc", byIdentity: true},
		{name: "default namespace identity", wantID: "default/svc", byIdentity: true},
		{name: "missing namespace separator", id: "svc", wantError: true},
		{name: "extra separator", id: "apps/svc/extra", wantError: true},
		{name: "missing identifier", wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := fwresource.ImportStateRequest{ID: tc.id}
			response := fwresource.ImportStateResponse{
				State: tfsdk.State{
					Schema: schema.Schema,
					Raw:    tftypes.NewValue(schema.Schema.Type().TerraformType(ctx), nil),
				},
				Identity: &tfsdk.ResourceIdentity{
					Schema: identitySchema.IdentitySchema,
				},
			}
			if tc.byIdentity {
				request.Identity = &tfsdk.ResourceIdentity{Schema: identitySchema.IdentitySchema}
				namespace := types.StringNull()
				if tc.namespace != "" {
					namespace = types.StringValue(tc.namespace)
				}
				if diags := request.Identity.Set(ctx, common.NamespacedResourceIdentity{
					ResourceIdentity: common.ResourceIdentity{
						Name: types.StringValue("svc"), Kind: types.StringValue("Service"),
						APIVersion: types.StringValue("v1"),
					},
					Namespace: namespace,
				}); diags.HasError() {
					t.Fatal(diags)
				}
				response.Identity = request.Identity
			}
			r.(fwresource.ResourceWithImportState).ImportState(ctx, request, &response)
			if response.Diagnostics.HasError() != tc.wantError {
				t.Fatalf("unexpected import diagnostics: %v", response.Diagnostics)
			}
			if tc.wantError {
				return
			}
			var id types.String
			if diags := response.State.GetAttribute(ctx, path.Root("id"), &id); diags.HasError() {
				t.Fatal(diags)
			}
			if id.IsNull() || id.IsUnknown() || id.ValueString() != tc.wantID {
				t.Fatalf("import ID = %s, want %q", id, tc.wantID)
			}
			var identity common.NamespacedResourceIdentity
			if diags := response.Identity.Get(ctx, &identity); diags.HasError() {
				t.Fatal(diags)
			}
			if identity.Name.ValueString() != "svc" || identity.Kind.ValueString() != "Service" ||
				identity.APIVersion.ValueString() != "v1" || identity.Namespace.IsNull() {
				t.Fatalf("import did not populate complete Service identity: %#v", identity)
			}
		})
	}
}

func TestServicePreIdentityUpgrade(t *testing.T) {
	ctx := context.Background()
	r := serviceRegisteredFrameworkResource(t)
	var identitySchema fwresource.IdentitySchemaResponse
	r.(fwresource.ResourceWithIdentity).IdentitySchema(ctx, fwresource.IdentitySchemaRequest{}, &identitySchema)
	upgraders := r.(fwresource.ResourceWithUpgradeIdentity).UpgradeIdentity(ctx)
	upgrader, ok := upgraders[0]
	if !ok {
		t.Fatal("missing pre-identity upgrader")
	}
	response := fwresource.UpgradeIdentityResponse{
		Identity: &tfsdk.ResourceIdentity{Schema: identitySchema.IdentitySchema},
	}
	upgrader.IdentityUpgrader(ctx, fwresource.UpgradeIdentityRequest{}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	var identity common.NamespacedResourceIdentity
	if diags := response.Identity.Get(ctx, &identity); diags.HasError() {
		t.Fatal(diags)
	}
	if !identity.Name.IsNull() || !identity.Namespace.IsNull() ||
		!identity.Kind.IsNull() || !identity.APIVersion.IsNull() {
		t.Fatalf("absent historical identity must remain unresolved until Read: %#v", identity)
	}
}

func serviceCheckConvertedState(t *testing.T, state tfsdk.State, version int64) {
	t.Helper()
	ctx := context.Background()
	for _, check := range []struct {
		path path.Path
		want string
	}{
		{path.Root("id"), "apps/svc-abc"},
		{path.Root("metadata").AtListIndex(0).AtName("uid"), "original-service-uid"},
		{path.Root("metadata").AtListIndex(0).AtName("namespace"), "apps"},
		{path.Root("metadata").AtListIndex(0).AtName("name"), "svc-abc"},
		{path.Root("metadata").AtListIndex(0).AtName("generate_name"), "svc-"},
		{path.Root("spec").AtListIndex(0).AtName("cluster_ip"), "10.96.0.42"},
		{path.Root("spec").AtListIndex(0).AtName("type"), "LoadBalancer"},
		{path.Root("spec").AtListIndex(0).AtName("external_traffic_policy"), "Local"},
		{path.Root("spec").AtListIndex(0).AtName("port").AtListIndex(0).AtName("target_port"), "http"},
		{path.Root("timeouts").AtName("create"), "12m"},
	} {
		var value types.String
		if diags := state.GetAttribute(ctx, check.path, &value); diags.HasError() {
			t.Fatal(diags)
		}
		if value.IsNull() || value.IsUnknown() || value.ValueString() != check.want {
			t.Errorf("%s = %s, want %q", check.path, value, check.want)
		}
	}
	for _, check := range []struct {
		path path.Path
		want int64
	}{
		{path.Root("spec").AtListIndex(0).AtName("port").AtListIndex(0).AtName("node_port"), 32000},
		{path.Root("spec").AtListIndex(0).AtName("health_check_node_port"), 32001},
		{path.Root("metadata").AtListIndex(0).AtName("generation"), 7},
	} {
		var value types.Int64
		if diags := state.GetAttribute(ctx, check.path, &value); diags.HasError() {
			t.Fatal(diags)
		}
		if value.IsNull() || value.IsUnknown() || value.ValueInt64() != check.want {
			t.Errorf("%s = %s, want %d", check.path, value, check.want)
		}
	}
	var wait types.Bool
	if diags := state.GetAttribute(ctx, path.Root("wait_for_load_balancer"), &wait); diags.HasError() {
		t.Fatal(diags)
	}
	if wait.IsNull() || wait.IsUnknown() || wait.ValueBool() {
		t.Fatalf("provider-only wait setting was lost: %s", wait)
	}
	if version == 1 {
		var timeout types.Int64
		timeoutPath := path.Root("spec").AtListIndex(0).AtName("session_affinity_config").
			AtName("client_ip").AtName("timeout_seconds")
		if diags := state.GetAttribute(ctx, timeoutPath, &timeout); diags.HasError() {
			t.Fatal(diags)
		}
		if timeout.ValueInt64() != 300 {
			t.Fatalf("affinity timeout lost: %s", timeout)
		}
		var clusterIPs types.List
		if diags := state.GetAttribute(ctx, path.Root("spec").AtListIndex(0).AtName("cluster_ips"), &clusterIPs); diags.HasError() {
			t.Fatal(diags)
		}
		if len(clusterIPs.Elements()) != 2 {
			t.Fatalf("dual-stack identity lost: %s", clusterIPs)
		}
	}
}

func serviceStoredFixture(t *testing.T, version int64) *tfprotov6.RawState {
	t.Helper()
	port := map[string]any{
		"app_protocol": "example.com/http", "name": "http", "node_port": 32000,
		"port": 80, "protocol": "TCP", "target_port": "http",
	}
	spec := map[string]any{
		"allocate_load_balancer_node_ports": false,
		"cluster_ip":                        "10.96.0.42", "cluster_ips": []string{"10.96.0.42", "fd00::42"},
		"external_ips": []string{"192.0.2.2"}, "external_name": "",
		"external_traffic_policy": "Local", "health_check_node_port": 32001,
		"internal_traffic_policy": "Local", "ip_families": []string{"IPv4", "IPv6"},
		"ip_family_policy": "RequireDualStack", "load_balancer_class": "example.com/controller",
		"load_balancer_ip": "192.0.2.3", "load_balancer_source_ranges": []string{"192.0.2.0/24"},
		"port": []map[string]any{port}, "publish_not_ready_addresses": true,
		"selector": map[string]string{"app": "web"}, "session_affinity": "ClientIP",
		"session_affinity_config": []map[string]any{{
			"client_ip": []map[string]any{{"timeout_seconds": 300}},
		}},
		"type": "LoadBalancer",
	}
	source := map[string]any{
		"id": "apps/svc-abc", "wait_for_load_balancer": false,
		"metadata": []map[string]any{{
			"name": "svc-abc", "namespace": "apps", "generate_name": "svc-",
			"annotations": map[string]string{"example.com/owner": "retained"},
			"labels":      map[string]string{"app": "web"}, "generation": 7,
			"resource_version": "1234", "uid": "original-service-uid",
		}},
		"spec": []map[string]any{spec}, "timeouts": map[string]any{"create": "12m"},
		"status": []map[string]any{{
			"load_balancer": []map[string]any{{
				"ingress": []map[string]any{{
					"ip": "192.0.2.3", "ip_mode": "VIP", "hostname": "lb.example.com",
				}},
			}},
		}},
	}
	if version == 0 {
		delete(source, "status")
		source["load_balancer_ingress"] = []map[string]any{{"ip": "192.0.2.3", "hostname": "lb.example.com"}}
		delete(port, "app_protocol")
		for _, field := range []string{
			"allocate_load_balancer_node_ports", "cluster_ips", "internal_traffic_policy",
			"ip_families", "ip_family_policy", "load_balancer_class", "session_affinity_config",
		} {
			delete(spec, field)
		}
	}
	data, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	return &tfprotov6.RawState{JSON: data}
}

// Loopback CLI migration regressions.

func serviceCLIMigration(t *testing.T, version string, variant serviceCLIVariant, alias bool) {
	t.Helper()
	serviceCLIMigrationWithConfig(t, version, variant, alias, nil)
}

func serviceCLIMigrationWithConfig(t *testing.T, version string, variant serviceCLIVariant, alias bool, transform func(string) string, migrationChecks ...plancheck.PlanCheck) {
	t.Helper()
	serviceCLIMigrationWithOptions(t, version, variant, alias, transform, false, false, migrationChecks...)
}

func serviceCLIMigrationWithOptions(t *testing.T, version string, variant serviceCLIVariant, alias bool, transform func(string) string, noRefresh, repeatSDKApply bool, migrationChecks ...plancheck.PlanCheck) {
	t.Helper()
	workspaces := serviceCLIEnvironment(t)
	api := newServiceCLIAPI(t)
	legacy := serviceCLIConfig(api, variant, "kubernetes_service_v1", "test")
	if transform != nil {
		legacy = transform(legacy)
	}
	current := serviceCLIAffinityAssignments(t, legacy)
	previous := ""
	if alias {
		legacy = strings.ReplaceAll(legacy, "kubernetes_service_v1", "kubernetes_service")
		previous = "kubernetes_service.test"
		current += `
moved {
  from = kubernetes_service.test
  to = kubernetes_service_v1.test
}
`
	}
	baselinePhase := "baseline"
	initial := resource.TestStep{Config: legacy}
	if version == "local" {
		initial.ProtoV6ProviderFactories = serviceCLIProtoV6ProviderFactories
		baselinePhase = "create"
	} else {
		initial.ExternalProviders = map[string]resource.ExternalProvider{
			"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= " + version},
		}
	}
	initial.PreConfig = func() { api.setPhase(baselinePhase, false) }
	address := "kubernetes_service_v1.test"
	oldAddress := address
	if alias {
		oldAddress = previous
	}
	var baseline *coreapi.Service
	initial.Check = api.checkVersion(oldAddress, &baseline, false, version)
	if version != "local" {
		initial.Check = resource.ComposeTestCheckFunc(serviceCLIReleasedVersion(t, workspaces, version), initial.Check)
	}
	checks := serviceCLISettled(address, previous, "no-op")
	if len(migrationChecks) != 0 {
		checks.PreApply = migrationChecks
	}
	options := &resource.AdditionalCLIOptions{}
	if noRefresh {
		checks.PreApply = append(checks.PreApply, serviceCLINoRefreshPlan{api: api})
	}
	testCase := resource.TestCase{
		AdditionalCLIOptions: options,
		IsUnitTest:           true,
		CheckDestroy:         api.checkDestroy,
		ErrorCheck:           api.errorCheck,
		Steps: []resource.TestStep{
			initial,
			{
				PreConfig: func() {
					options.Plan.NoRefresh = noRefresh
					api.setPhase("migration", true)
				},
				ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
				Config:                   current,
				ConfigPlanChecks:         checks,
				Check:                    api.check(address, &baseline, true),
			},
			{
				PreConfig: func() {
					options.Plan.NoRefresh = false
					api.setPhase("follow-up", true)
				},
				ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
				Config:                   current,
				ConfigPlanChecks:         serviceCLISettled(address, "", "no-op"),
				Check:                    api.check(address, &baseline, true),
			},
			{
				PreConfig:                func() { api.setPhase("cleanup", false) },
				ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
				Config:                   current, Destroy: true,
			},
		},
	}
	if repeatSDKApply {
		repeated := initial
		repeated.PreConfig = func() { api.setPhase("sdk-second-apply", true) }
		repeated.ConfigPlanChecks = serviceCLISettled(oldAddress, "", "no-op")
		repeated.Check = api.checkVersion(oldAddress, &baseline, true, version)
		testCase.Steps = append(testCase.Steps[:1], append([]resource.TestStep{repeated}, testCase.Steps[1:]...)...)
	}
	resource.Test(t, testCase)
}

// The migration plan must stand on upgraded stored state. A following normal
// refresh also proves that the converted state agrees with the API response.
func TestServiceCLI_UpgradeWithoutRefresh(t *testing.T) {
	for _, variant := range serviceCLIVariants() {
		if variant.name != "cluster-ip-omitted" && variant.name != "client-ip-timeout" {
			continue
		}
		for _, alias := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/alias-%t", variant.name, alias), func(t *testing.T) {
				serviceCLIMigrationWithOptions(t, "3.2.1", variant, alias, nil, true, false)
			})
		}
		if variant.name == "cluster-ip-omitted" {
			t.Run("after-second-sdk-apply", func(t *testing.T) {
				serviceCLIMigrationWithOptions(t, "3.2.1", variant, false, nil, true, true)
			})
		}
	}
}

type serviceCLINoRefreshPlan struct{ api *serviceCLIAPI }

func (check serviceCLINoRefreshPlan) CheckPlan(_ context.Context, _ plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	check.api.mu.Lock()
	defer check.api.mu.Unlock()
	for request, count := range check.api.calls["migration"] {
		if request != "GET /version" && count != 0 {
			resp.Error = fmt.Errorf("migration plan refreshed the Service: %d %s", count, request)
			return
		}
	}
}

func TestServiceCLI_UpgradeFrom321(t *testing.T) {
	for _, variant := range serviceCLIVariants() {
		if variant.name == "client-ip-empty-nested" {
			// A fresh create sends invalid timeoutSeconds=0. Existing or
			// imported state can retain a valid timeout; those are separate tests.
			continue
		}
		t.Run(variant.name, func(t *testing.T) { serviceCLIMigration(t, "3.2.1", variant, false) })
	}
}

func TestServiceCLI_AliasMove(t *testing.T) {
	for _, version := range []string{"3.2.1", "local"} {
		for _, variant := range serviceCLIVariants() {
			switch variant.name {
			case "cluster-ip-omitted", "headless", "external-name", "node-port-fixed", "load-balancer-local-fixed", "client-ip-timeout":
				t.Run(version+"/"+variant.name, func(t *testing.T) { serviceCLIMigration(t, version, variant, true) })
			}
		}
	}
}

func TestServiceCLI_UpgradeFrom2371(t *testing.T) {
	for _, variant := range serviceCLIVariants() {
		switch variant.name {
		case "cluster-ip-omitted", "dual-stack", "load-balancer-no-nodeports-class", "client-ip-empty-affinity":
			t.Run(variant.name, func(t *testing.T) { serviceCLIMigration(t, "2.37.1", variant, false) })
		}
	}
}

func TestServiceCLI_ReleasedFreshEmptyClientIPRejected(t *testing.T) {
	for _, version := range []string{"3.2.1", "2.37.1"} {
		t.Run(version, func(t *testing.T) {
			workspaces := serviceCLIEnvironment(t)
			api := newServiceCLIAPI(t)
			var variant serviceCLIVariant
			for _, candidate := range serviceCLIVariants() {
				if candidate.name == "client-ip-empty-nested" {
					variant = candidate
				}
			}
			released := map[string]resource.ExternalProvider{
				"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= " + version},
			}
			resource.Test(t, resource.TestCase{
				IsUnitTest: true, CheckDestroy: api.checkDestroy, ErrorCheck: api.errorCheck,
				Steps: []resource.TestStep{
					{
						PreConfig:         func() { api.setPhase("baseline", false) },
						ExternalProviders: released,
						Config:            serviceCLIConfig(api, variant, "kubernetes_service_v1", "test"),
						ExpectError:       regexp.MustCompile(regexp.QuoteMeta("ClientIP affinity timeout must be 1..86400")),
					},
					{
						ExternalProviders: released,
						Config:            fmt.Sprintf("provider \"kubernetes\" { host = %q }\n", api.server.URL),
						Check: resource.ComposeTestCheckFunc(
							serviceCLIReleasedVersion(t, workspaces, version),
							func(_ *terraform.State) error {
								api.mu.Lock()
								defer api.mu.Unlock()
								if len(api.objects) != 0 || api.calls["baseline"]["POST /api/v1/namespaces/default/services"] != 1 {
									return fmt.Errorf("invalid released affinity shape did not receive exactly one rejected create")
								}
								return nil
							},
						),
					},
				},
			})
		})
	}
}

func TestServiceCLI_SameTypeAddressMove(t *testing.T) {
	for _, resourceType := range []string{"kubernetes_service", "kubernetes_service_v1"} {
		for _, destination := range []string{"resource", "for-each", "module"} {
			t.Run(resourceType+"/"+destination, func(t *testing.T) {
				workspaces := serviceCLIEnvironment(t)
				api := newServiceCLIAPI(t)
				variant := serviceCLIVariants()[5]
				initial := serviceCLIConfig(api, variant, "kubernetes_service_v1", "test")
				fromAddress := "kubernetes_service_v1.test"
				current := strings.ReplaceAll(initial, `"test"`, `"renamed"`)
				current = strings.ReplaceAll(current, "kubernetes_service_v1.test", "kubernetes_service_v1.renamed")
				address := "kubernetes_service_v1.renamed"
				if destination == "for-each" {
					initial = strings.Replace(initial, `resource "kubernetes_service_v1" "test" {`,
						"resource \"kubernetes_service_v1\" \"test\" {\n  for_each = toset([\"old\"])", 1)
					initial = strings.ReplaceAll(initial, "kubernetes_service_v1.test", `kubernetes_service_v1.test["old"]`)
					fromAddress = `kubernetes_service_v1.test["old"]`
					current = strings.Replace(current, `resource "kubernetes_service_v1" "renamed" {`,
						"resource \"kubernetes_service_v1\" \"renamed\" {\n  for_each = toset([\"stable\"])", 1)
					current = strings.ReplaceAll(current, "kubernetes_service_v1.renamed", `kubernetes_service_v1.renamed["stable"]`)
					address = `kubernetes_service_v1.renamed["stable"]`
				}
				var files map[string]string
				if destination == "module" {
					child := strings.Replace(initial, "provider \"kubernetes\" {\n  host = \""+api.server.URL+"\"\n}", "", 1)
					child += `
terraform {
  required_providers {
    kubernetes = { source = "hashicorp/kubernetes" }
  }
}
`
					files = map[string]string{"child/main.tf": child}
					current = `
provider "kubernetes" { host = "` + api.server.URL + `" }
module "child" {
  source = "./child"
}
output "service_uid" { value = module.child.service_uid }
output "service_cluster_ip" { value = module.child.service_cluster_ip }
output "service_nodeports" { value = module.child.service_nodeports }
`
					address = "module.child.kubernetes_service_v1.test"
				}
				current += "\nmoved {\n  from = " + fromAddress + "\n  to = " + address + "\n}\n"
				initial = strings.ReplaceAll(initial, "kubernetes_service_v1", resourceType)
				current = strings.ReplaceAll(current, "kubernetes_service_v1", resourceType)
				fromAddress = strings.ReplaceAll(fromAddress, "kubernetes_service_v1", resourceType)
				address = strings.ReplaceAll(address, "kubernetes_service_v1", resourceType)
				for path, content := range files {
					files[path] = strings.ReplaceAll(content, "kubernetes_service_v1", resourceType)
				}
				moveStep := resource.TestStep{
					PreConfig:        func() { api.setPhase("migration", true) },
					Config:           current,
					ConfigPlanChecks: serviceCLISettled(address, fromAddress, "no-op"),
				}
				followStep := resource.TestStep{
					PreConfig:        func() { api.setPhase("follow-up", true) },
					Config:           current,
					ConfigPlanChecks: serviceCLISettled(address, "", "no-op"),
				}
				if files != nil {
					dir := filepath.Join(workspaces, "module-config")
					// ConfigDirectory copies root .tf files, not module trees.
					// Point to the repository-local child fixture explicitly.
					current = strings.Replace(current, `source = "./child"`,
						fmt.Sprintf("source = %q", filepath.Join(dir, "child")), 1)
					files["main.tf"] = current
					for path, contents := range files {
						path = filepath.Join(dir, path)
						if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
							t.Fatal(err)
						}
						if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
							t.Fatal(err)
						}
					}
					moveStep.Config, followStep.Config = "", ""
					moveStep.ConfigDirectory, followStep.ConfigDirectory = config.StaticDirectory(dir), config.StaticDirectory(dir)
				}
				var baseline *coreapi.Service
				moveStep.Check, followStep.Check = api.check(address, &baseline, true), api.check(address, &baseline, true)
				initialStep := resource.TestStep{
					Config: initial, Check: api.check(resourceType+".test", &baseline, false),
					ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
				}
				moveStep.ProtoV6ProviderFactories, followStep.ProtoV6ProviderFactories = serviceCLIProtoV6ProviderFactories, serviceCLIProtoV6ProviderFactories
				if destination == "for-each" {
					initialStep.Check = nil
					initialStep.ConfigStateChecks = []statecheck.StateCheck{api.checkJSONAddress(fromAddress, &baseline)}
				}
				if destination == "for-each" || destination == "module" {
					moveStep.Check, followStep.Check = nil, nil
					moveStep.ConfigStateChecks = []statecheck.StateCheck{api.checkJSONAddress(address, &baseline)}
					followStep.ConfigStateChecks = []statecheck.StateCheck{api.checkJSONAddress(address, &baseline)}
				}
				steps := []resource.TestStep{initialStep, moveStep, followStep}
				// Explicit teardown keeps all refresh requests under the preceding
				// read-only guard. It also empties for_each state before the
				// harness's deprecated, string-index-incompatible final conversion.
				destroy := resource.TestStep{
					PreConfig: func() { api.setPhase("cleanup", false) },
					Config:    current, Destroy: true, ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
				}
				if destination == "module" {
					destroy.Config = ""
					destroy.ConfigDirectory = moveStep.ConfigDirectory
				}
				steps = append(steps, destroy)
				resource.Test(t, resource.TestCase{
					IsUnitTest:   true,
					CheckDestroy: api.checkDestroy,
					ErrorCheck:   api.errorCheck,
					Steps:        steps,
				})
			})
		}
	}
}

func TestServiceCLI_InverseAliasMoveRejected(t *testing.T) {
	serviceCLIEnvironment(t)
	api := newServiceCLIAPI(t)
	initial := serviceCLIConfig(api, serviceCLIVariants()[4], "kubernetes_service_v1", "test")
	unsupported := strings.ReplaceAll(initial, "kubernetes_service_v1", "kubernetes_service") + `
moved {
  from = kubernetes_service_v1.test
  to = kubernetes_service.test
}
`
	var baseline *coreapi.Service
	resource.Test(t, resource.TestCase{
		IsUnitTest: true, ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
		CheckDestroy: api.checkDestroy, ErrorCheck: api.errorCheck,
		Steps: []resource.TestStep{
			{
				Config: initial,
				Check:  api.check("kubernetes_service_v1.test", &baseline, false),
			},
			{
				PreConfig:   func() { api.setPhase("unsupported-move", true) },
				Config:      unsupported,
				ExpectError: regexp.MustCompile("Move Resource State Not Supported"),
			},
			{
				// Restore valid configuration after the rejected plan, proving
				// the source address, API identity, and allocations survived.
				PreConfig:        func() { api.setPhase("follow-up", true) },
				Config:           initial,
				ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
				Check:            api.check("kubernetes_service_v1.test", &baseline, true),
			},
			{PreConfig: func() { api.setPhase("cleanup", false) }, Config: initial, Destroy: true},
		},
	})
}

type serviceCLIInstance struct {
	key   string
	index string
	name  string
}

func serviceCLIInstances(mode string) []serviceCLIInstance {
	if mode == "count" {
		return []serviceCLIInstance{
			{key: "0", index: "[0]", name: "service-loopback-0"},
			{key: "1", index: "[1]", name: "service-loopback-1"},
		}
	}
	return []serviceCLIInstance{
		{key: "first", index: `["first"]`, name: "service-loopback-first"},
		{key: "second", index: `["second"]`, name: "service-loopback-second"},
	}
}

func serviceCLIWholeResourceConfig(api *serviceCLIAPI, mode, resourceType string) string {
	instances := `count = 2`
	name := `"service-loopback-${count.index}"`
	if mode == "for-each" {
		instances = `for_each = toset(["first", "second"])`
		name = `"service-loopback-${each.key}"`
	}
	return fmt.Sprintf(`
provider "kubernetes" {
  host = %q
}
resource %q "test" {
  %s
  metadata {
    name = %s
    namespace = "default"
    annotations = { managed = "whole-resource-move" }
  }
  wait_for_load_balancer = false
  spec {
    type = "NodePort"
    selector = { app = "whole-resource-move" }
    port {
      name = "web"
      port = 80
      target_port = "http"
    }
  }
}
output "service_uids" {
  value = { for key, service in %s.test : key => service.metadata[0].uid }
}
output "service_cluster_ips" {
  value = { for key, service in %s.test : key => service.spec[0].cluster_ips }
}
output "service_nodeports" {
  value = { for key, service in %s.test : key => [for port in service.spec[0].port : port.node_port] }
}
`, api.server.URL, resourceType, instances, name, resourceType, resourceType, resourceType)
}

type serviceCLIWholeResourcePlan struct {
	instances []serviceCLIInstance
	moved     bool
}

func (check serviceCLIWholeResourcePlan) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	expected := map[string]string{}
	for _, instance := range check.instances {
		previous := ""
		if check.moved {
			previous = "kubernetes_service.test" + instance.index
		}
		expected["kubernetes_service_v1.test"+instance.index] = previous
	}
	for _, change := range req.Plan.ResourceChanges {
		from, ok := expected[change.Address]
		if !ok || change.Change == nil || !change.Change.Actions.NoOp() || change.PreviousAddress != from {
			detail, _ := json.Marshal(change)
			resp.Error = fmt.Errorf("whole-resource move requires exact no-op for every instance; unexpected change: %s", detail)
			return
		}
		delete(expected, change.Address)
	}
	if len(expected) != 0 {
		resp.Error = fmt.Errorf("whole-resource move omitted planned instances: %v", expected)
	}
}

func (api *serviceCLIAPI) checkWholeResource(instances []serviceCLIInstance, resourceType string, baseline map[string]*coreapi.Service) statecheck.StateCheck {
	return serviceCLIStateCheckFunc(func(req statecheck.CheckStateRequest) error {
		api.mu.Lock()
		defer api.mu.Unlock()
		if len(api.objects) != len(instances) {
			return fmt.Errorf("whole-resource move expected %d API objects, got %d", len(instances), len(api.objects))
		}
		if api.phase == "baseline" {
			if api.calls["baseline"]["POST /api/v1/namespaces/default/services"] != len(instances) {
				return fmt.Errorf("whole-resource baseline did not create each instance exactly once")
			}
			for request := range api.calls["baseline"] {
				if !strings.HasPrefix(request, "GET ") && !strings.HasPrefix(request, "POST ") {
					return fmt.Errorf("unexpected whole-resource baseline mutation: %s", request)
				}
			}
		}
		for _, phase := range []string{"migration", "follow-up"} {
			for request := range api.calls[phase] {
				if !strings.HasPrefix(request, "GET ") {
					return fmt.Errorf("whole-resource move wrote during %s: %s", phase, request)
				}
			}
		}
		uids := map[string]bool{}
		nodePorts := map[int32]bool{}
		for _, instance := range instances {
			object := api.objects[instance.name]
			if object == nil || object.UID == "" || uids[string(object.UID)] {
				return fmt.Errorf("instance %s has no unique API identity", instance.name)
			}
			uids[string(object.UID)] = true
			if len(object.Spec.Ports) != 1 || object.Spec.Ports[0].NodePort == 0 || nodePorts[object.Spec.Ports[0].NodePort] {
				return fmt.Errorf("instance %s has no unique allocated nodePort", instance.name)
			}
			nodePorts[object.Spec.Ports[0].NodePort] = true
			if object.Annotations["controller.kubernetes.io/observed"] != "preserve" ||
				object.Spec.TrafficDistribution == nil || *object.Spec.TrafficDistribution != "PreferClose" {
				return fmt.Errorf("instance %s lost controller/unmodeled fields", instance.name)
			}
			if old := baseline[instance.name]; old != nil {
				if old.UID != object.UID || !reflect.DeepEqual(old.Spec, object.Spec) ||
					!reflect.DeepEqual(old.Annotations, object.Annotations) || !reflect.DeepEqual(old.Status, object.Status) {
					return fmt.Errorf("whole-resource move changed UID/spec/annotations/status for %s", instance.name)
				}
			} else {
				baseline[instance.name] = object.DeepCopy()
			}
			address := resourceType + ".test" + instance.index
			found := false
			for _, res := range req.State.Values.RootModule.Resources {
				if res.Address != address {
					continue
				}
				found = true
				metadata, ok := res.AttributeValues["metadata"].([]any)
				if !ok || len(metadata) != 1 {
					return fmt.Errorf("%s has no known metadata", address)
				}
				meta, ok := metadata[0].(map[string]any)
				if !ok || meta["uid"] != string(object.UID) {
					return fmt.Errorf("%s state UID differs from API", address)
				}
				specs, ok := res.AttributeValues["spec"].([]any)
				if !ok || len(specs) != 1 {
					return fmt.Errorf("%s has no known spec", address)
				}
				spec, ok := specs[0].(map[string]any)
				if !ok || spec["cluster_ip"] != object.Spec.ClusterIP {
					return fmt.Errorf("%s state ClusterIP differs from API", address)
				}
				ports, ok := spec["port"].([]any)
				if !ok || len(ports) != 1 {
					return fmt.Errorf("%s state has no known port", address)
				}
				port, ok := ports[0].(map[string]any)
				if !ok || fmt.Sprint(port["node_port"]) != strconv.Itoa(int(object.Spec.Ports[0].NodePort)) ||
					port["target_port"] != object.Spec.Ports[0].TargetPort.String() {
					return fmt.Errorf("%s state lost allocated port identity", address)
				}
			}
			if !found {
				return fmt.Errorf("whole-resource state lacks %s", address)
			}
			for outputName, expected := range map[string]any{
				"service_uids": string(object.UID), "service_cluster_ips": object.Spec.ClusterIPs,
				"service_nodeports": []int32{object.Spec.Ports[0].NodePort},
			} {
				output, ok := req.State.Values.Outputs[outputName]
				if !ok {
					return fmt.Errorf("whole-resource state lacks output %s", outputName)
				}
				values, ok := output.Value.(map[string]any)
				if !ok {
					return fmt.Errorf("output %s is not a known object", outputName)
				}
				actual, _ := json.Marshal(values[instance.key])
				wanted, _ := json.Marshal(expected)
				if string(actual) != string(wanted) {
					return fmt.Errorf("output %s[%q] = %s, expected %s", outputName, instance.key, actual, wanted)
				}
			}
		}
		return nil
	})
}

func TestServiceCLI_WholeResourceAliasMove(t *testing.T) {
	for _, version := range []string{"3.2.1", "local"} {
		for _, mode := range []string{"count", "for-each"} {
			t.Run(version+"/"+mode, func(t *testing.T) {
				workspaces := serviceCLIEnvironment(t)
				api := newServiceCLIAPI(t)
				instances := serviceCLIInstances(mode)
				baseline := map[string]*coreapi.Service{}
				initial := resource.TestStep{
					PreConfig:         func() { api.setPhase("baseline", false) },
					Config:            serviceCLIWholeResourceConfig(api, mode, "kubernetes_service"),
					ConfigStateChecks: []statecheck.StateCheck{api.checkWholeResource(instances, "kubernetes_service", baseline)},
				}
				if version == "local" {
					initial.ProtoV6ProviderFactories = serviceCLIProtoV6ProviderFactories
				} else {
					initial.ExternalProviders = map[string]resource.ExternalProvider{
						"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= " + version},
					}
					initial.ConfigStateChecks = append(initial.ConfigStateChecks, serviceCLIStateCheckFunc(func(_ statecheck.CheckStateRequest) error {
						return serviceCLIReleasedVersion(t, workspaces, version)(nil)
					}))
				}
				current := serviceCLIWholeResourceConfig(api, mode, "kubernetes_service_v1") + `
moved {
  from = kubernetes_service.test
  to = kubernetes_service_v1.test
}
`
				migration := resource.TestStep{
					PreConfig:                func() { api.setPhase("migration", true) },
					ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
					Config:                   current,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							serviceCLIWholeResourcePlan{instances: instances, moved: true}, plancheck.ExpectEmptyPlan(),
						},
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
					ConfigStateChecks: []statecheck.StateCheck{api.checkWholeResource(instances, "kubernetes_service_v1", baseline)},
				}
				follow := migration
				follow.PreConfig = func() { api.setPhase("follow-up", true) }
				follow.ConfigPlanChecks.PreApply = []plancheck.PlanCheck{
					serviceCLIWholeResourcePlan{instances: instances}, plancheck.ExpectEmptyPlan(),
				}
				resource.Test(t, resource.TestCase{
					IsUnitTest: true, CheckDestroy: api.checkDestroy, ErrorCheck: api.errorCheck,
					Steps: []resource.TestStep{
						initial, migration, follow,
						{
							PreConfig:                func() { api.setPhase("cleanup", false) },
							ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
							Config:                   current, Destroy: true,
						},
					},
				})
			})
		}
	}
}

// ImportStatePersist uses the existing workspace rather than the usual isolated
// import workspace. Remove only the managed address with the real CLI first;
// unlike a removed-block apply, state rm preserves the existing known outputs.
func serviceCLIForgetForImport(t *testing.T, workspaces string) {
	t.Helper()
	type rawState struct {
		Outputs   map[string]any
		Resources []struct{ Mode, Type, Name string }
	}
	var candidates []string
	err := filepath.WalkDir(workspaces, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || entry.Name() != "terraform.tfstate" {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var state rawState
		if err := json.Unmarshal(content, &state); err != nil {
			return err
		}
		for _, object := range state.Resources {
			if object.Mode == "managed" && object.Type == "kubernetes_service_v1" && object.Name == "test" {
				candidates = append(candidates, path)
			}
		}
		return nil
	})
	if err != nil || len(candidates) != 1 {
		t.Fatalf("locate unique SDK import workspace: paths=%v error=%v", candidates, err)
	}
	path := candidates[0]
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var prior rawState
	if err := json.Unmarshal(before, &prior); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(os.Getenv("TF_ACC_TERRAFORM_PATH"), "state", "rm", "kubernetes_service_v1.test")
	command.Dir = filepath.Dir(path)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("forget SDK address before CLI import: %s: %v", output, err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var next rawState
	if err := json.Unmarshal(after, &next); err != nil {
		t.Fatal(err)
	}
	if len(next.Resources) != 0 || !reflect.DeepEqual(prior.Outputs, next.Outputs) {
		t.Fatalf("state rm must forget only the Service and preserve all known outputs")
	}
}

func TestServiceCLI_SDKImportRetainsAffinityAndNodePort(t *testing.T) {
	workspaces := serviceCLIEnvironment(t)
	api := newServiceCLIAPI(t)
	variant := serviceCLIVariant{
		fields: "type = \"NodePort\"\n session_affinity = \"ClientIP\"\n session_affinity_config {\n client_ip { timeout_seconds = 300 }\n}",
		ports:  `port { port = 80 }`,
	}
	initial := serviceCLIConfig(api, variant, "kubernetes_service_v1", "test")
	// Use the client-only default consistently; no API-derived attributes
	// are ignored during the imported-state comparison.
	initial = strings.Replace(initial, "wait_for_load_balancer = false", "wait_for_load_balancer = true", 1)
	sdkConfig := strings.Replace(initial, "timeout_seconds = 300", "", 1)
	current := serviceCLIAffinityAssignments(t, sdkConfig)
	released := map[string]resource.ExternalProvider{
		"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.2.1"},
	}
	var baseline *coreapi.Service
	var sdkAttributes map[string]string
	resource.Test(t, resource.TestCase{
		IsUnitTest: true, CheckDestroy: api.checkDestroy, ErrorCheck: api.errorCheck,
		Steps: []resource.TestStep{
			{
				PreConfig:         func() { api.setPhase("baseline", false) },
				ExternalProviders: released, Config: initial,
				Check: resource.ComposeTestCheckFunc(
					serviceCLIReleasedVersion(t, workspaces, "3.2.1"),
					api.checkVersion("kubernetes_service_v1.test", &baseline, false, "3.2.1"),
				),
			},
			{
				PreConfig:         func() { api.setPhase("sdk-affinity-omission", true) },
				ExternalProviders: released, Config: sdkConfig,
				ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
				Check: resource.ComposeTestCheckFunc(
					api.checkVersion("kubernetes_service_v1.test", &baseline, true, "3.2.1"),
					func(state *terraform.State) error {
						sdkAttributes = maps.Clone(state.RootModule().Resources["kubernetes_service_v1.test"].Primary.Attributes)
						return nil
					},
				),
			},
			{
				PreConfig: func() {
					api.setPhase("sdk-import", true)
					serviceCLIForgetForImport(t, workspaces)
				},
				ExternalProviders: released, ResourceName: "kubernetes_service_v1.test",
				ImportState: true, ImportStateId: "default/service-loopback", ImportStatePersist: true,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected one persisted SDK import, got %d", len(states))
					}
					before, after := maps.Clone(sdkAttributes), maps.Clone(states[0].Attributes)
					// Only this client-only setting cannot be recovered
					// from Kubernetes; compare every API-derived attribute.
					delete(before, "wait_for_load_balancer")
					delete(after, "wait_for_load_balancer")
					if !reflect.DeepEqual(before, after) {
						return fmt.Errorf("SDK import changed API-derived state:\nbefore=%#v\nafter=%#v", before, after)
					}
					return nil
				},
			},
			{
				PreConfig:                func() { api.setPhase("migration", true) },
				ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
				Config:                   current, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
				Check: api.check("kubernetes_service_v1.test", &baseline, true),
			},
			{
				PreConfig:                func() { api.setPhase("follow-up", true) },
				ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
				Config:                   current, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
				Check: api.check("kubernetes_service_v1.test", &baseline, true),
			},
			{
				PreConfig:                func() { api.setPhase("cleanup", false) },
				ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
				Config:                   current, Destroy: true,
			},
		},
	})
}

// Loopback CLI lifecycle regressions.

func TestServiceCLI_LifecycleUpdateRemovalReorder(t *testing.T) {
	serviceCLIEnvironment(t)
	api := newServiceCLIAPI(t)
	variant := serviceCLIVariants()[5]
	initial := serviceCLIConfig(api, variant, "kubernetes_service_v1", "test")
	updated := strings.ReplaceAll(initial, `managed = "original"`, `managed = "updated"`)
	updated = strings.ReplaceAll(updated, `target_port = "http"`, `target_port = "http-v2"`)
	updated = strings.ReplaceAll(updated, `app = "service-cli"`, `app = "updated"`)
	removed := strings.ReplaceAll(updated, `annotations = { managed = "updated" }`, `annotations = {}`)
	removed = strings.ReplaceAll(removed, `external_ips = ["192.0.2.11", "192.0.2.12"]`, "")
	removed = strings.ReplaceAll(removed, `app_protocol = "http"`, "")
	ports := strings.Split(variant.ports, "\n    port {")
	reorderedPorts := "port {" + ports[1] + "\n    " + ports[0]
	reordered := strings.Replace(initial, variant.ports, reorderedPorts, 1)
	var baseline *coreapi.Service
	stableAllocations := func(_ *terraform.State) error {
		api.mu.Lock()
		defer api.mu.Unlock()
		object := api.objects["service-loopback"]
		if object == nil || baseline == nil || !reflect.DeepEqual(object.Spec.ClusterIPs, baseline.Spec.ClusterIPs) {
			return fmt.Errorf("update changed Service IP allocation")
		}
		ports := map[string]int32{}
		for _, port := range baseline.Spec.Ports {
			ports[port.Name] = port.NodePort
		}
		for _, port := range object.Spec.Ports {
			if ports[port.Name] != port.NodePort {
				return fmt.Errorf("update/reorder changed nodePort for %s: %d -> %d", port.Name, ports[port.Name], port.NodePort)
			}
		}
		return nil
	}
	steps := []resource.TestStep{{Config: initial, Check: api.check("kubernetes_service_v1.test", &baseline, false)}}
	for _, change := range []struct {
		name   string
		config string
	}{
		{"intentional-update", updated}, {"intentional-removal", removed}, {"intentional-reorder", reordered},
	} {
		steps = append(steps, resource.TestStep{
			PreConfig:        func() { api.setPhase(change.name, false) },
			Config:           change.config,
			ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "update"),
			Check:            resource.ComposeTestCheckFunc(api.check("kubernetes_service_v1.test", &baseline, false), stableAllocations),
		})
	}
	steps = append(steps, resource.TestStep{
		PreConfig: func() { api.setPhase("follow-up", true) }, Config: reordered,
		ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
		Check:            resource.ComposeTestCheckFunc(api.check("kubernetes_service_v1.test", &baseline, false), stableAllocations),
	})
	steps = append(steps, resource.TestStep{
		PreConfig: func() { api.setPhase("cleanup", false) }, Config: reordered, Destroy: true,
	})
	resource.Test(t, resource.TestCase{
		IsUnitTest: true, ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
		CheckDestroy: api.checkDestroy, ErrorCheck: api.errorCheck, Steps: steps,
	})
}

func TestServiceCLI_Import(t *testing.T) {
	serviceCLIEnvironment(t)
	api := newServiceCLIAPI(t)
	config := serviceCLIConfig(api, serviceCLIVariants()[7], "kubernetes_service_v1", "test")
	var baseline *coreapi.Service
	resource.Test(t, resource.TestCase{
		IsUnitTest: true, ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
		CheckDestroy: api.checkDestroy, ErrorCheck: api.errorCheck,
		Steps: []resource.TestStep{
			{Config: config, Check: api.check("kubernetes_service_v1.test", &baseline, false)},
			{
				PreConfig:    func() { api.setPhase("import", true) },
				ResourceName: "kubernetes_service_v1.test", ImportState: true, ImportStateVerify: true,
				// The wait choice is client-side only, not retrievable from the
				// Service API. All API-derived fields must roundtrip exactly.
				ImportStateVerifyIgnore: []string{"wait_for_load_balancer"},
			},
			{
				PreConfig: func() { api.setPhase("follow-up", true) },
				Config:    config, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
				Check: api.check("kubernetes_service_v1.test", &baseline, true),
			},
			{PreConfig: func() { api.setPhase("cleanup", false) }, Config: config, Destroy: true},
		},
	})
}

func TestServiceCLI_DriftAndDisappearance(t *testing.T) {
	serviceCLIEnvironment(t)
	api := newServiceCLIAPI(t)
	config := serviceCLIConfig(api, serviceCLIVariants()[4], "kubernetes_service_v1", "test")
	var baseline, recreated *coreapi.Service
	resource.Test(t, resource.TestCase{
		IsUnitTest: true, ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
		CheckDestroy: api.checkDestroy, ErrorCheck: api.errorCheck,
		Steps: []resource.TestStep{
			{Config: config, Check: api.check("kubernetes_service_v1.test", &baseline, false)},
			{
				PreConfig: func() {
					api.setPhase("intentional-drift-repair", false)
					api.mu.Lock()
					defer api.mu.Unlock()
					api.objects["service-loopback"].Spec.Selector = map[string]string{"app": "external-drift"}
				},
				Config: config, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "update"),
				Check: api.check("kubernetes_service_v1.test", &baseline, true),
			},
			{
				PreConfig: func() {
					api.setPhase("intentional-recreate", false)
					api.mu.Lock()
					defer api.mu.Unlock()
					delete(api.objects, "service-loopback")
				},
				Config: config, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "create"),
				Check: resource.ComposeTestCheckFunc(
					api.check("kubernetes_service_v1.test", &recreated, false),
					func(_ *terraform.State) error {
						if baseline.UID == recreated.UID {
							return fmt.Errorf("disappeared Service was not recreated")
						}
						return nil
					},
				),
			},
			{
				PreConfig: func() { api.setPhase("follow-up", true) },
				Config:    config, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
				Check: api.check("kubernetes_service_v1.test", &recreated, true),
			},
			{PreConfig: func() { api.setPhase("cleanup", false) }, Config: config, Destroy: true},
		},
	})
}

func TestServiceCLI_DataSourceAndDependentReference(t *testing.T) {
	serviceCLIEnvironment(t)
	api := newServiceCLIAPI(t)
	config := serviceCLIConfig(api, serviceCLIVariants()[0], "kubernetes_service_v1", "test") + `
data "kubernetes_service_v1" "readback" {
  metadata {
    name      = kubernetes_service_v1.test.metadata[0].name
    namespace = kubernetes_service_v1.test.metadata[0].namespace
  }
}
resource "kubernetes_service_v1" "dependent" {
  metadata {
    name      = "service-dependent"
    namespace = "default"
  }
  wait_for_load_balancer = false
  spec {
    type          = "ExternalName"
    external_name = "${kubernetes_service_v1.test.metadata[0].name}.${kubernetes_service_v1.test.metadata[0].namespace}.svc.cluster.local"
  }
}
output "readback_ip" {
  value = data.kubernetes_service_v1.readback.spec[0].cluster_ip
}
`
	var baseline *coreapi.Service
	check := resource.ComposeTestCheckFunc(
		api.check("kubernetes_service_v1.test", &baseline, false),
		resource.TestCheckResourceAttr("kubernetes_service_v1.dependent", "spec.0.external_name", "service-loopback.default.svc.cluster.local"),
		resource.TestCheckResourceAttrPair("data.kubernetes_service_v1.readback", "spec.0.cluster_ip", "kubernetes_service_v1.test", "spec.0.cluster_ip"),
	)
	resource.Test(t, resource.TestCase{
		IsUnitTest: true, ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
		CheckDestroy: api.checkDestroy, ErrorCheck: api.errorCheck,
		Steps: []resource.TestStep{
			{Config: config, Check: check},
			{PreConfig: func() { api.setPhase("follow-up", true) }, Config: config, Check: check},
			{PreConfig: func() { api.setPhase("cleanup", false) }, Config: config, Destroy: true},
		},
	})
}

func TestServiceCLI_UnnamedPortAndSelectorUpdates(t *testing.T) {
	serviceCLIEnvironment(t)
	api := newServiceCLIAPI(t)
	initial := serviceCLIConfig(api, serviceCLIVariants()[4], "kubernetes_service_v1", "test")
	selector := strings.Replace(initial, `selector = { app = "service-cli" }`, `selector = { app = "selected" }`, 1)
	number := strings.Replace(selector, `port { port = 80 }`, `port { port = 81 }`, 1)
	protocol := strings.Replace(number, `port { port = 81 }`, "port {\n      port = 81\n      protocol = \"UDP\"\n    }", 1)
	var baseline *coreapi.Service
	check := func(phase string, port int32, protocol coreapi.Protocol) resource.TestCheckFunc {
		return resource.ComposeTestCheckFunc(
			api.check("kubernetes_service_v1.test", &baseline, false),
			func(_ *terraform.State) error {
				api.mu.Lock()
				defer api.mu.Unlock()
				object := api.objects["service-loopback"]
				if object == nil || baseline == nil {
					return fmt.Errorf("unnamed Service port baseline is missing")
				}
				expected := baseline.Spec.DeepCopy()
				expected.Selector = map[string]string{"app": "selected"}
				expected.Ports[0].Port, expected.Ports[0].Protocol = port, protocol
				if !reflect.DeepEqual(*expected, object.Spec) {
					return fmt.Errorf("unnamed port or selector edit changed unrelated Service fields/allocations: got %+v, expected %+v", object.Spec, *expected)
				}
				if object.Spec.Ports[0].Name != "" || object.Spec.Ports[0].NodePort == 0 ||
					object.Spec.Ports[0].TargetPort.IntVal != 80 {
					return fmt.Errorf("unnamed port edit lost its allocated nodePort or computed targetPort")
				}
				if len(api.patches[phase]) != 1 {
					return fmt.Errorf("%s expected one intentional PATCH, got %d", phase, len(api.patches[phase]))
				}
				var operations []struct {
					Op   string `json:"op"`
					Path string `json:"path"`
				}
				if err := json.Unmarshal(api.patches[phase][0], &operations); err != nil {
					return err
				}
				for _, operation := range operations {
					if operation.Op == "test" {
						continue
					}
					if phase == "selector-only" && !strings.HasPrefix(operation.Path, "/spec/selector") {
						return fmt.Errorf("selector-only update mutated unrelated path %s", operation.Path)
					}
					if strings.Contains(operation.Path, "nodePort") || strings.Contains(operation.Path, "targetPort") {
						return fmt.Errorf("%s overwrote an unchanged allocated/computed port field: %s", phase, operation.Path)
					}
				}
				return nil
			},
		)
	}
	resource.Test(t, resource.TestCase{
		IsUnitTest: true, ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
		CheckDestroy: api.checkDestroy, ErrorCheck: api.errorCheck,
		Steps: []resource.TestStep{
			{Config: initial, Check: api.check("kubernetes_service_v1.test", &baseline, false)},
			{
				PreConfig: func() { api.setPhase("selector-only", false) }, Config: selector,
				ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "update"),
				Check:            check("selector-only", 80, coreapi.ProtocolTCP),
			},
			{
				PreConfig: func() { api.setPhase("unnamed-port-number", false) }, Config: number,
				ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "update"),
				Check:            check("unnamed-port-number", 81, coreapi.ProtocolTCP),
			},
			{
				PreConfig: func() { api.setPhase("unnamed-port-protocol", false) }, Config: protocol,
				ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "update"),
				Check:            check("unnamed-port-protocol", 81, coreapi.ProtocolUDP),
			},
			{
				PreConfig: func() { api.setPhase("follow-up", true) }, Config: protocol,
				ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
				Check:            check("unnamed-port-protocol", 81, coreapi.ProtocolUDP),
			},
			{PreConfig: func() { api.setPhase("cleanup", false) }, Config: protocol, Destroy: true},
		},
	})
}

type serviceCLISDKTransitionOutputPlan struct {
	baseline **coreapi.Service
}

func (check serviceCLISDKTransitionOutputPlan) CheckPlan(ctx context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	serviceCLIExactPlan{address: "kubernetes_service_v1.test", action: "no-op"}.CheckPlan(ctx, req, resp)
	if resp.Error != nil {
		return
	}
	var beforePorts, afterPorts []int32
	for _, port := range (*check.baseline).Spec.Ports {
		beforePorts = append(beforePorts, port.NodePort)
		afterPorts = append(afterPorts, 0)
	}
	before := map[string]any{
		"service_uid": string((*check.baseline).UID), "service_cluster_ip": (*check.baseline).Spec.ClusterIP,
		"service_nodeports": beforePorts,
	}
	if len(req.Plan.OutputChanges) != len(before) {
		resp.Error = fmt.Errorf("SDK transition must change only the known node-port output")
		return
	}
	for name, value := range before {
		change := req.Plan.OutputChanges[name]
		wantBefore, _ := json.Marshal(value)
		wantAfter, action := wantBefore, "no-op"
		if name == "service_nodeports" {
			wantAfter, _ = json.Marshal(afterPorts)
			action = "update"
		}
		if change == nil || len(change.Actions) != 1 || string(change.Actions[0]) != action || change.AfterUnknown != false {
			resp.Error = fmt.Errorf("unexpected SDK transition output action or unknown value for %s: %+v", name, change)
			return
		}
		gotBefore, _ := json.Marshal(change.Before)
		gotAfter, _ := json.Marshal(change.After)
		if string(gotBefore) != string(wantBefore) || string(gotAfter) != string(wantAfter) {
			resp.Error = fmt.Errorf("unexpected SDK transition output %s: %s -> %s; expected %s -> %s", name, gotBefore, gotAfter, wantBefore, wantAfter)
			return
		}
	}
}

func serviceCLIUpdateRegression(t *testing.T, initial, updated serviceCLIVariant, typeTransition bool, expectedUpdate func(*coreapi.Service)) {
	t.Helper()
	for _, version := range []string{"3.2.1", "local"} {
		t.Run(version, func(t *testing.T) {
			workspaces := serviceCLIEnvironment(t)
			api := newServiceCLIAPI(t)
			initialConfig := serviceCLIConfig(api, initial, "kubernetes_service_v1", "test")
			updatedConfig := serviceCLIConfig(api, updated, "kubernetes_service_v1", "test")
			var baseline, expected *coreapi.Service
			initialCheck := api.checkVersion("kubernetes_service_v1.test", &baseline, false, version)
			if version == "3.2.1" {
				initialCheck = resource.ComposeTestCheckFunc(serviceCLIReleasedVersion(t, workspaces, version), initialCheck)
			}
			updateCheck := resource.ComposeTestCheckFunc(
				func(state *terraform.State) error {
					var nodePortOutput []int32
					// The released SDK initially exposes its old node ports
					// in root outputs after a type transition, even though
					// API and resource state already contain zero.
					if version == "3.2.1" && baseline.Spec.Type != expected.Spec.Type && api.phase == "intentional-update" {
						for _, port := range baseline.Spec.Ports {
							nodePortOutput = append(nodePortOutput, port.NodePort)
						}
					}
					return api.checkVersionOutput("kubernetes_service_v1.test", &expected, true, version, nodePortOutput)(state)
				},
				func(_ *terraform.State) error {
					api.mu.Lock()
					defer api.mu.Unlock()
					calls := api.calls["intentional-update"]
					patch := "PATCH /api/v1/namespaces/default/services/service-loopback"
					if calls[patch] != 1 {
						return fmt.Errorf("expected exactly one update PATCH: %v", calls)
					}
					for request, count := range calls {
						if request == "GET /version" && count == 1 {
							continue
						}
						if request != patch && request != "GET /api/v1/namespaces/default/services/service-loopback" {
							return fmt.Errorf("unexpected update request: %s", request)
						}
					}
					return nil
				},
			)
			testCase := resource.TestCase{
				IsUnitTest: true, CheckDestroy: api.checkDestroy, ErrorCheck: api.errorCheck,
				Steps: []resource.TestStep{
					{
						PreConfig: func() { api.setPhase("baseline", false) },
						Config:    initialConfig, Check: initialCheck,
					},
					{
						PreConfig: func() {
							expected = baseline.DeepCopy()
							expectedUpdate(expected)
							api.setPhase("intentional-update", false)
						},
						Config: updatedConfig, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "update"),
						Check: updateCheck,
					},
					{
						PreConfig: func() { api.setPhase("follow-up", true) },
						Config:    updatedConfig, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
						Check: updateCheck,
					},
					{PreConfig: func() { api.setPhase("cleanup", false) }, Config: updatedConfig, Destroy: true},
				},
			}
			if version == "3.2.1" {
				testCase.ExternalProviders = map[string]resource.ExternalProvider{
					"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.2.1"},
				}
				if typeTransition {
					outputPlan := serviceCLISDKTransitionOutputPlan{baseline: &baseline}
					testCase.Steps[1].ExpectNonEmptyPlan = true
					testCase.Steps[1].ConfigPlanChecks.PostApplyPreRefresh = []plancheck.PlanCheck{outputPlan}
					testCase.Steps[1].ConfigPlanChecks.PostApplyPostRefresh = []plancheck.PlanCheck{outputPlan}
					testCase.Steps = []resource.TestStep{
						testCase.Steps[0], testCase.Steps[1],
						{
							PreConfig: func() { api.setPhase("sdk-output-refresh", true) },
							Config:    updatedConfig, Check: updateCheck,
							ConfigPlanChecks: resource.ConfigPlanChecks{
								PreApply:             []plancheck.PlanCheck{outputPlan},
								PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
							},
						},
						testCase.Steps[2], testCase.Steps[3],
					}
				}
			} else {
				testCase.ProtoV6ProviderFactories = serviceCLIProtoV6ProviderFactories
			}
			resource.Test(t, testCase)
		})
	}
}

func TestServiceCLI_CombinedPortNameAndNumberUpdate(t *testing.T) {
	initial := serviceCLIVariant{fields: `type = "NodePort"`, ports: `port { port = 80 }`}
	updated := serviceCLIVariant{fields: initial.fields, ports: `port {
  name = "http"
  port = 81
}`}
	serviceCLIUpdateRegression(t, initial, updated, false, func(expected *coreapi.Service) {
		expected.Spec.Ports[0].Name = "http"
		expected.Spec.Ports[0].Port = 81
	})
}

func TestServiceCLI_ExternalIPsTypeTransition(t *testing.T) {
	for _, serviceType := range []coreapi.ServiceType{coreapi.ServiceTypeNodePort, coreapi.ServiceTypeLoadBalancer} {
		t.Run(string(serviceType), func(t *testing.T) {
			fields := func(serviceType coreapi.ServiceType) string {
				return fmt.Sprintf(`type = %q
external_ips = ["192.0.2.11"]
external_traffic_policy = "Local"
`, serviceType)
			}
			initial := serviceCLIVariant{fields: fields(serviceType), ports: `port { port = 80 }`}
			updated := serviceCLIVariant{fields: fields(coreapi.ServiceTypeClusterIP), ports: initial.ports}
			serviceCLIUpdateRegression(t, initial, updated, true, func(expected *coreapi.Service) {
				expected.Spec.Type = coreapi.ServiceTypeClusterIP
				expected.Spec.Ports[0].NodePort = 0
				expected.Spec.HealthCheckNodePort = 0
				expected.Spec.AllocateLoadBalancerNodePorts = nil
				expected.Status.LoadBalancer = coreapi.LoadBalancerStatus{}
			})
		})
	}
}

// Loopback CLI affinity regressions.

// The approved syntax migration affects only these two nesting levels in the
// Framework target. Released and current SDK alias configurations never pass
// through this converter.
func serviceCLIAffinityAssignments(t *testing.T, config string) string {
	t.Helper()
	file, diagnostics := hclwrite.ParseConfig([]byte(config), "service-cli.tf", hcl.InitialPos)
	if diagnostics.HasErrors() {
		t.Fatal(diagnostics.Error())
	}
	changed := false
	for _, resourceBlock := range file.Body().Blocks() {
		if resourceBlock.Type() != "resource" || len(resourceBlock.Labels()) != 2 ||
			resourceBlock.Labels()[0] != "kubernetes_service_v1" {
			continue
		}
		for _, spec := range resourceBlock.Body().Blocks() {
			if spec.Type() != "spec" {
				continue
			}
			for _, affinity := range spec.Body().Blocks() {
				if affinity.Type() != "session_affinity_config" {
					continue
				}
				for _, clientIP := range affinity.Body().Blocks() {
					if clientIP.Type() == "client_ip" {
						serviceCLIBlockToObject(affinity.Body(), clientIP)
					}
				}
				serviceCLIBlockToObject(spec.Body(), affinity)
				changed = true
			}
		}
	}
	if !changed {
		return config
	}
	return string(hclwrite.Format(file.Bytes()))
}

func serviceCLIBlockToObject(parent *hclwrite.Body, block *hclwrite.Block) {
	tokens := hclwrite.Tokens{
		&hclwrite.Token{Type: hclsyntax.TokenOBrace, Bytes: []byte("{")},
	}
	tokens = append(tokens, block.Body().BuildTokens(nil)...)
	tokens = append(tokens,
		&hclwrite.Token{Type: hclsyntax.TokenCBrace, Bytes: []byte("}")},
	)
	parent.RemoveBlock(block)
	parent.SetAttributeRaw(block.Type(), tokens)
}

func TestServiceCLI_AffinityOmissionRetainsTimeout(t *testing.T) {
	for _, targetVersion := range []string{"3.2.1", "local"} {
		for _, omission := range []struct {
			name   string
			fields string
		}{
			{"timeout", "session_affinity = \"ClientIP\"\n session_affinity_config {\n client_ip {}\n}"},
			{"client-ip", "session_affinity = \"ClientIP\"\n session_affinity_config {}"},
			{"config", `session_affinity = "ClientIP"`},
		} {
			t.Run(targetVersion+"/"+omission.name, func(t *testing.T) {
				workspaces := serviceCLIEnvironment(t)
				api := newServiceCLIAPI(t)
				explicit := serviceCLIVariant{
					fields: "session_affinity = \"ClientIP\"\n session_affinity_config {\n client_ip { timeout_seconds = 300 }\n}",
					ports:  `port { port = 80 }`,
				}
				initialConfig := serviceCLIConfig(api, explicit, "kubernetes_service_v1", "test")
				omitted := explicit
				omitted.fields = omission.fields
				current := serviceCLIConfig(api, omitted, "kubernetes_service_v1", "test")
				sdkOmitted := current
				if targetVersion == "local" {
					current = serviceCLIAffinityAssignments(t, current)
				}
				released := map[string]resource.ExternalProvider{
					"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.2.1"},
				}
				var baseline *coreapi.Service
				target := resource.TestStep{
					PreConfig: func() { api.setPhase("migration", true) },
					Config:    current, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
					Check: api.check("kubernetes_service_v1.test", &baseline, true),
				}
				if targetVersion == "local" {
					target.ProtoV6ProviderFactories = serviceCLIProtoV6ProviderFactories
				} else {
					target.ExternalProviders = released
				}
				follow := target
				follow.PreConfig = func() { api.setPhase("follow-up", true) }
				destroy := resource.TestStep{
					PreConfig: func() { api.setPhase("cleanup", false) },
					Config:    current, Destroy: true,
					ProtoV6ProviderFactories: target.ProtoV6ProviderFactories,
					ExternalProviders:        target.ExternalProviders,
				}
				// This is omission after a valid timeout=300 baseline, not a
				// fresh client_ip{} create (which the SDK sends as invalid 0).
				steps := []resource.TestStep{{
					PreConfig:         func() { api.setPhase("baseline", false) },
					ExternalProviders: released, Config: initialConfig,
					Check: resource.ComposeTestCheckFunc(
						serviceCLIReleasedVersion(t, workspaces, "3.2.1"),
						api.checkVersion("kubernetes_service_v1.test", &baseline, false, "3.2.1"),
					),
				}}
				if targetVersion == "local" {
					steps = append(steps, resource.TestStep{
						PreConfig:         func() { api.setPhase("sdk-affinity-omission", true) },
						ExternalProviders: released, Config: sdkOmitted,
						ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
						Check:            api.checkVersion("kubernetes_service_v1.test", &baseline, true, "3.2.1"),
					})
				}
				steps = append(steps, target, follow, destroy)
				resource.Test(t, resource.TestCase{
					IsUnitTest: true, CheckDestroy: api.checkDestroy, ErrorCheck: api.errorCheck,
					Steps: steps,
				})
			})
		}
	}
}

func TestServiceCLI_AffinityModeChangeClearsConfig(t *testing.T) {
	serviceCLIEnvironment(t)
	api := newServiceCLIAPI(t)
	variant := serviceCLIVariant{
		fields: "session_affinity = \"ClientIP\"\n session_affinity_config {\n client_ip { timeout_seconds = 300 }\n}",
		ports:  `port { port = 80 }`,
	}
	config := func(variant serviceCLIVariant) string {
		return serviceCLIAffinityAssignments(t, serviceCLIConfig(api, variant, "kubernetes_service_v1", "test")) + `
output "affinity_config_count" {
  value = kubernetes_service_v1.test.spec[0].session_affinity_config == null ? 0 : 1
}
`
	}
	initial := config(variant)
	variant.fields = `session_affinity = "ClientIP"`
	omitted := config(variant)
	variant.fields = `session_affinity = "None"`
	disabled := config(variant)
	var baseline *coreapi.Service
	cleared := resource.ComposeTestCheckFunc(
		api.check("kubernetes_service_v1.test", &baseline, false),
		resource.TestCheckResourceAttr("kubernetes_service_v1.test", "spec.0.session_affinity", "None"),
		resource.TestCheckNoResourceAttr("kubernetes_service_v1.test", "spec.0.session_affinity_config.client_ip.timeout_seconds"),
		func(_ *terraform.State) error {
			api.mu.Lock()
			defer api.mu.Unlock()
			object := api.objects["service-loopback"]
			if object == nil || baseline == nil {
				return fmt.Errorf("affinity mode-change baseline is missing")
			}
			expected := baseline.Spec.DeepCopy()
			expected.SessionAffinity = coreapi.ServiceAffinityNone
			expected.SessionAffinityConfig = nil
			if !reflect.DeepEqual(*expected, object.Spec) {
				return fmt.Errorf("affinity mode change did not clear only affinity configuration: got %+v", object.Spec)
			}
			calls := api.calls["affinity-mode-change"]
			if calls["PATCH /api/v1/namespaces/default/services/service-loopback"] != 1 {
				return fmt.Errorf("affinity mode change must issue exactly one PATCH: %v", calls)
			}
			for request := range calls {
				if request != "PATCH /api/v1/namespaces/default/services/service-loopback" &&
					request != "GET /api/v1/namespaces/default/services/service-loopback" {
					return fmt.Errorf("unexpected API request while changing affinity mode: %s", request)
				}
			}
			return nil
		},
	)
	defaulted := resource.ComposeTestCheckFunc(
		api.check("kubernetes_service_v1.test", &baseline, false),
		resource.TestCheckResourceAttr("kubernetes_service_v1.test", "spec.0.session_affinity", "ClientIP"),
		resource.TestCheckResourceAttr("kubernetes_service_v1.test", "spec.0.session_affinity_config.client_ip.timeout_seconds", "10800"),
		func(_ *terraform.State) error {
			api.mu.Lock()
			defer api.mu.Unlock()
			object := api.objects["service-loopback"]
			if object == nil || baseline == nil {
				return fmt.Errorf("affinity defaulting baseline is missing")
			}
			expected := baseline.Spec.DeepCopy()
			if expected.SessionAffinityConfig == nil || expected.SessionAffinityConfig.ClientIP == nil ||
				expected.SessionAffinityConfig.ClientIP.TimeoutSeconds == nil {
				return fmt.Errorf("initial explicit timeout was not present")
			}
			*expected.SessionAffinityConfig.ClientIP.TimeoutSeconds = 10800
			if !reflect.DeepEqual(*expected, object.Spec) {
				return fmt.Errorf("re-enabling ClientIP from nil config did not default only the timeout to 10800: %+v", object.Spec)
			}
			calls := api.calls["affinity-mode-enable"]
			if calls["PATCH /api/v1/namespaces/default/services/service-loopback"] != 1 {
				return fmt.Errorf("re-enabling ClientIP must issue exactly one PATCH: %v", calls)
			}
			for request := range calls {
				if request != "PATCH /api/v1/namespaces/default/services/service-loopback" &&
					request != "GET /api/v1/namespaces/default/services/service-loopback" {
					return fmt.Errorf("unexpected API request while re-enabling ClientIP: %s", request)
				}
			}
			return nil
		},
	)
	resource.Test(t, resource.TestCase{
		IsUnitTest: true, ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
		CheckDestroy: api.checkDestroy, ErrorCheck: api.errorCheck,
		Steps: []resource.TestStep{
			{
				Config: initial,
				Check:  api.check("kubernetes_service_v1.test", &baseline, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("affinity_config_count", knownvalue.Int64Exact(1)),
				},
			},
			{
				PreConfig: func() { api.setPhase("affinity-omission", true) },
				Config:    omitted, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
				Check: api.check("kubernetes_service_v1.test", &baseline, true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("affinity_config_count", knownvalue.Int64Exact(1)),
				},
			},
			{
				PreConfig: func() { api.setPhase("affinity-mode-change", false) },
				Config:    disabled, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "update"),
				Check: cleared,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("affinity_config_count", knownvalue.Int64Exact(0)),
				},
			},
			{
				PreConfig: func() { api.setPhase("follow-up", true) },
				Config:    disabled, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
				Check: cleared,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("affinity_config_count", knownvalue.Int64Exact(0)),
				},
			},
			{
				PreConfig: func() { api.setPhase("affinity-mode-enable", false) },
				Config:    omitted, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "update"),
				Check: defaulted,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("affinity_config_count", knownvalue.Int64Exact(1)),
				},
			},
			{
				PreConfig: func() { api.setPhase("follow-up", true) },
				Config:    omitted, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
				Check: defaulted,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("affinity_config_count", knownvalue.Int64Exact(1)),
				},
			},
			{PreConfig: func() { api.setPhase("cleanup", false) }, Config: omitted, Destroy: true},
		},
	})
}

func TestServiceCLI_OriginalAffinitySyntaxRejected(t *testing.T) {
	for _, variant := range serviceCLIVariants() {
		if variant.name != "client-ip-empty-affinity" && variant.name != "client-ip-timeout" {
			continue
		}
		t.Run(variant.name, func(t *testing.T) {
			workspaces := serviceCLIEnvironment(t)
			api := newServiceCLIAPI(t)
			legacy := serviceCLIConfig(api, variant, "kubernetes_service_v1", "test")
			current := serviceCLIAffinityAssignments(t, legacy)
			var baseline *coreapi.Service
			resource.Test(t, resource.TestCase{
				IsUnitTest: true, CheckDestroy: api.checkDestroy, ErrorCheck: api.errorCheck,
				Steps: []resource.TestStep{
					{
						PreConfig: func() { api.setPhase("baseline", false) },
						ExternalProviders: map[string]resource.ExternalProvider{
							"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.2.1"},
						},
						Config: legacy,
						Check: resource.ComposeTestCheckFunc(
							serviceCLIReleasedVersion(t, workspaces, "3.2.1"),
							api.check("kubernetes_service_v1.test", &baseline, false),
						),
					},
					{
						PreConfig:                func() { api.setPhase("unsupported-syntax", true) },
						ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
						Config:                   legacy, ExpectError: regexp.MustCompile("Unsupported block type"),
					},
					{
						PreConfig:                func() { api.setPhase("migration", true) },
						ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
						Config:                   current, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
						Check: api.check("kubernetes_service_v1.test", &baseline, true),
					},
					{
						PreConfig:                func() { api.setPhase("follow-up", true) },
						ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
						Config:                   current, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
						Check: api.check("kubernetes_service_v1.test", &baseline, true),
					},
					{
						PreConfig:                func() { api.setPhase("cleanup", false) },
						ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
						Config:                   current, Destroy: true,
					},
				},
			})
		})
	}
}

func TestServiceCLI_EmptyClientIPCollectionsRejected(t *testing.T) {
	for _, invalid := range []struct {
		name   string
		config string
	}{
		{name: "outer", config: "[]"},
		{name: "client-ip", config: "{ client_ip = [] }"},
	} {
		for _, operation := range []string{"create", "update"} {
			t.Run(invalid.name+"/"+operation, func(t *testing.T) {
				serviceCLIEnvironment(t)
				api := newServiceCLIAPI(t)
				config := func(affinity string) string {
					return serviceCLIConfig(api, serviceCLIVariant{
						fields: "session_affinity = \"ClientIP\"\nsession_affinity_config = " + affinity,
						ports:  "port { port = 80 }",
					}, "kubernetes_service_v1", "test")
				}
				current := config("{ client_ip = { timeout_seconds = 300 } }")
				rejected := config(invalid.config)
				if operation == "create" {
					serviceCLIRejectAndRecover(t, api, rejected, current, "object required")
					return
				}

				serviceCLIExpectNoAPICalls(t, api, "invalid-config")
				var baseline *coreapi.Service
				resource.Test(t, resource.TestCase{
					IsUnitTest: true, ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
					CheckDestroy: api.checkDestroy, ErrorCheck: api.errorCheck,
					Steps: []resource.TestStep{
						{
							Config: current,
							Check: resource.ComposeTestCheckFunc(
								api.check("kubernetes_service_v1.test", &baseline, false),
								resource.TestCheckResourceAttr("kubernetes_service_v1.test", "spec.0.session_affinity_config.client_ip.timeout_seconds", "300"),
							),
						},
						{
							PreConfig: func() { api.setPhase("invalid-config", true) },
							Config:    rejected, ExpectError: regexp.MustCompile("object required"),
						},
						{
							PreConfig: func() { api.setPhase("migration", true) },
							Config:    current, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
							Check: api.check("kubernetes_service_v1.test", &baseline, true),
						},
						{
							PreConfig: func() { api.setPhase("follow-up", true) },
							Config:    current, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
							Check: api.check("kubernetes_service_v1.test", &baseline, true),
						},
						{PreConfig: func() { api.setPhase("cleanup", false) }, Config: current, Destroy: true},
					},
				})
			})
		}
	}
}

func TestServiceCLI_NoneNullAffinityObject(t *testing.T) {
	serviceCLIEnvironment(t)
	api := newServiceCLIAPI(t)
	config := serviceCLIConfig(api, serviceCLIVariant{
		fields: "session_affinity = \"None\"\nsession_affinity_config = null",
		ports:  "port { port = 80 }",
	}, "kubernetes_service_v1", "test")
	serviceCLICreateAndNoWritePlan(t, api, config)
}

// Loopback CLI allocation regressions.

func TestServiceCLI_AllocationMarkers(t *testing.T) {
	for _, variant := range []serviceCLIVariant{
		{name: "cluster-ips", fields: "cluster_ips = []", ports: "port { port = 80 }"},
		{name: "ip-families", fields: "ip_families = []", ports: "port { port = 80 }"},
		{name: "target-zero", ports: "port {\n      port = 80\n      target_port = \"0\"\n    }"},
	} {
		for _, version := range []string{"3.2.1", "local"} {
			t.Run(variant.name+"/"+version, func(t *testing.T) {
				workspaces := serviceCLIEnvironment(t)
				api := newServiceCLIAPI(t)
				config := serviceCLIConfig(api, variant, "kubernetes_service_v1", "test")
				current := serviceCLIConfig(api, serviceCLIVariant{ports: "port { port = 80 }"}, "kubernetes_service_v1", "test")
				diagnostic := map[string]string{
					"cluster-ips": "Invalid empty cluster_ips", "ip-families": "Invalid empty ip_families",
					"target-zero": "Invalid zero target_port",
				}[variant.name]
				if version == "local" {
					serviceCLIRejectAndRecover(t, api, config, current, diagnostic)
					return
				}
				released := map[string]resource.ExternalProvider{
					"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= " + version},
				}
				serviceCLIExpectNoAPICalls(t, api, "invalid-config")
				var baseline *coreapi.Service
				test := resource.TestCase{
					IsUnitTest: true, CheckDestroy: api.checkDestroy, ErrorCheck: api.errorCheck,
					Steps: []resource.TestStep{
						{
							ExternalProviders: released, Config: current,
							Check: resource.ComposeTestCheckFunc(
								serviceCLIReleasedVersion(t, workspaces, version),
								api.checkVersion("kubernetes_service_v1.test", &baseline, false, version),
								resource.TestCheckResourceAttr("kubernetes_service_v1.test", "spec.0.ip_families.#", "1"),
								resource.TestCheckResourceAttr("kubernetes_service_v1.test", "spec.0.ip_families.0", "IPv4"),
							),
						},
						{
							PreConfig:         func() { api.setPhase("sdk-marker-plan", true) },
							ExternalProviders: released, Config: config,
							PlanOnly: true, ExpectNonEmptyPlan: true,
							ConfigPlanChecks: resource.ConfigPlanChecks{
								PostApplyPreRefresh:  []plancheck.PlanCheck{serviceCLIAllocationMarkerPlan{variant.name}},
								PostApplyPostRefresh: []plancheck.PlanCheck{serviceCLIAllocationMarkerPlan{variant.name}},
							},
						},
						{
							PreConfig:                func() { api.setPhase("invalid-config", true) },
							ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
							Config:                   config, ExpectError: regexp.MustCompile(regexp.QuoteMeta(diagnostic)),
						},
						{
							PreConfig:                func() { api.setPhase("migration", true) },
							ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
							Config:                   current, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
							Check: api.check("kubernetes_service_v1.test", &baseline, true),
						},
						{
							PreConfig:                func() { api.setPhase("follow-up", true) },
							ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
							Config:                   current, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
							Check: api.check("kubernetes_service_v1.test", &baseline, true),
						},
						{
							PreConfig:                func() { api.setPhase("cleanup", false) },
							ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
							Config:                   current, Destroy: true,
						},
					},
				}
				resource.Test(t, test)
			})
		}
	}
}

func serviceCLIEmptyTargetPort() serviceCLIVariant {
	return serviceCLIVariant{
		name: "empty-target-port",
		ports: `port {
      port = 80
      target_port = ""
    }`,
	}
}

func TestServiceCLI_ReleasedEmptyTargetPort(t *testing.T) {
	workspaces := serviceCLIEnvironment(t)
	api := newServiceCLIAPI(t)
	config := serviceCLIConfig(api, serviceCLIEmptyTargetPort(), "kubernetes_service_v1", "test")
	var baseline *coreapi.Service
	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		ExternalProviders: map[string]resource.ExternalProvider{
			"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.2.1"},
		},
		CheckDestroy: api.checkDestroy, ErrorCheck: api.errorCheck,
		Steps: []resource.TestStep{
			{
				PreConfig: func() { api.setPhase("baseline", false) },
				Config:    config,
				Check: resource.ComposeTestCheckFunc(
					serviceCLIReleasedVersion(t, workspaces, "3.2.1"),
					api.checkVersion("kubernetes_service_v1.test", &baseline, false, "3.2.1"),
				),
			},
			{
				PreConfig: func() { api.setPhase("follow-up", true) },
				Config:    config, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
				Check: api.checkVersion("kubernetes_service_v1.test", &baseline, true, "3.2.1"),
			},
			{PreConfig: func() { api.setPhase("cleanup", false) }, Config: config, Destroy: true},
		},
	})
}

func TestServiceCLI_ExplicitEmptyTargetPortCreate(t *testing.T) {
	serviceCLIEnvironment(t)
	api := newServiceCLIAPI(t)
	config := serviceCLIConfig(api, serviceCLIEmptyTargetPort(), "kubernetes_service_v1", "test")
	current := strings.Replace(config, "      target_port = \"\"\n", "", 1)
	serviceCLIRejectAndRecover(t, api, config, current, "Invalid empty target_port")
}

func TestServiceCLI_ExplicitEmptyTargetPortUpgrade(t *testing.T) {
	workspaces := serviceCLIEnvironment(t)
	api := newServiceCLIAPI(t)
	legacy := serviceCLIConfig(api, serviceCLIEmptyTargetPort(), "kubernetes_service_v1", "test")
	current := strings.Replace(legacy, "      target_port = \"\"\n", "", 1)
	serviceCLIExpectNoAPICalls(t, api, "unsupported-empty-target")
	var baseline *coreapi.Service
	resource.Test(t, resource.TestCase{
		IsUnitTest: true, CheckDestroy: api.checkDestroy, ErrorCheck: api.errorCheck,
		Steps: []resource.TestStep{
			{
				PreConfig: func() { api.setPhase("baseline", false) },
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.2.1"},
				},
				Config: legacy,
				Check: resource.ComposeTestCheckFunc(
					serviceCLIReleasedVersion(t, workspaces, "3.2.1"),
					api.checkVersion("kubernetes_service_v1.test", &baseline, false, "3.2.1"),
				),
			},
			{
				PreConfig:                func() { api.setPhase("unsupported-empty-target", true) },
				ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
				Config:                   legacy, ExpectError: regexp.MustCompile("Invalid empty target_port"),
			},
			{
				PreConfig:                func() { api.setPhase("migration", true) },
				ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
				Config:                   current, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
				Check: api.check("kubernetes_service_v1.test", &baseline, true),
			},
			{
				PreConfig:                func() { api.setPhase("follow-up", true) },
				ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
				Config:                   current, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
				Check: api.check("kubernetes_service_v1.test", &baseline, true),
			},
			{
				PreConfig:                func() { api.setPhase("cleanup", false) },
				ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
				Config:                   current, Destroy: true,
			},
		},
	})
}

func serviceCLIScalarAllocationMarkers() []serviceCLIVariant {
	return []serviceCLIVariant{
		{name: "cluster_ip", fields: `cluster_ip = ""`, ports: `port { port = 80 }`},
		{name: "health_check_node_port", fields: `type = "LoadBalancer"
external_traffic_policy = "Local"
health_check_node_port = 0`, ports: `port { port = 80 }`},
	}
}

func TestServiceCLI_ScalarAllocationMarkersRejected(t *testing.T) {
	for _, marker := range serviceCLIScalarAllocationMarkers() {
		for _, version := range []string{"3.2.1", "local"} {
			t.Run(marker.name+"/"+version, func(t *testing.T) {
				workspaces := serviceCLIEnvironment(t)
				api := newServiceCLIAPI(t)
				invalid := serviceCLIConfig(api, marker, "kubernetes_service_v1", "test")
				current := strings.Replace(invalid, `cluster_ip = ""`, "", 1)
				current = strings.Replace(current, "health_check_node_port = 0", "", 1)
				current += fmt.Sprintf("\noutput \"allocation_marker\" { value = kubernetes_service_v1.test.spec[0].%s }\n", marker.name)
				diagnostic := "expected spec.0.cluster_ip to contain a valid IP"
				if marker.name == "health_check_node_port" {
					diagnostic = `expected "spec.0.health_check_node_port" to be a valid port number, got: 0`
				}
				if version == "local" {
					diagnostic = "must be a valid IP address or None"
					if marker.name == "health_check_node_port" {
						diagnostic = "value must be between 1 and 65535"
					}
				}
				serviceCLIExpectNoAPICalls(t, api, "invalid-config")
				var baseline *coreapi.Service
				check := func(unchanged bool) resource.TestCheckFunc {
					return resource.ComposeTestCheckFunc(
						api.checkVersion("kubernetes_service_v1.test", &baseline, unchanged, version),
						func(state *terraform.State) error {
							value := baseline.Spec.ClusterIP
							if marker.name == "health_check_node_port" {
								value = strconv.Itoa(int(baseline.Spec.HealthCheckNodePort))
							}
							return resource.TestCheckOutput("allocation_marker", value)(state)
						},
					)
				}
				testCase := resource.TestCase{
					IsUnitTest: true, CheckDestroy: api.checkDestroy, ErrorCheck: api.errorCheck,
					Steps: []resource.TestStep{
						{
							PreConfig: func() { api.setPhase("invalid-config", true) },
							Config:    invalid, ExpectError: regexp.MustCompile(regexp.QuoteMeta(diagnostic)),
						},
						{
							PreConfig: func() { api.setPhase("create", false) },
							Config:    current, Check: check(false),
						},
						{
							PreConfig: func() { api.setPhase("follow-up", true) },
							Config:    current, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
							Check: check(true),
						},
						{PreConfig: func() { api.setPhase("cleanup", false) }, Config: current, Destroy: true},
					},
				}
				if version == "3.2.1" {
					testCase.ExternalProviders = map[string]resource.ExternalProvider{
						"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.2.1"},
					}
					testCase.Steps[1].Check = resource.ComposeTestCheckFunc(
						serviceCLIReleasedVersion(t, workspaces, version), check(false),
					)
				} else {
					testCase.ProtoV6ProviderFactories = serviceCLIProtoV6ProviderFactories
				}
				resource.Test(t, testCase)
			})
		}
	}
}

func serviceCLIZeroNodePort(serviceType string) serviceCLIVariant {
	return serviceCLIVariant{
		name:   serviceType,
		fields: fmt.Sprintf("type = %q", serviceType),
		ports: `port {
      name = "web"
      port = 80
      node_port = 0
    }`,
	}
}

func TestServiceCLI_ExplicitZeroNodePortUpgrade(t *testing.T) {
	for _, serviceType := range []string{"NodePort", "LoadBalancer"} {
		t.Run(serviceType, func(t *testing.T) {
			// Zero is intentionally literal, not omitted or changed to an
			// expression. The released provider must first prove its contract.
			workspaces := serviceCLIEnvironment(t)
			api := newServiceCLIAPI(t)
			config := serviceCLIConfig(api, serviceCLIZeroNodePort(serviceType), "kubernetes_service_v1", "test")
			current := strings.Replace(config, "      node_port = 0\n", "", 1)
			serviceCLIExpectNoAPICalls(t, api, "unsupported-zero")
			var baseline *coreapi.Service
			resource.Test(t, resource.TestCase{
				IsUnitTest: true, CheckDestroy: api.checkDestroy, ErrorCheck: api.errorCheck,
				Steps: []resource.TestStep{
					{
						PreConfig: func() { api.setPhase("baseline", false) },
						ExternalProviders: map[string]resource.ExternalProvider{
							"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.2.1"},
						},
						Config: config,
						// The released provider itself perpetually plans the
						// allocated port back to literal zero. Permit only the
						// exact measured SDK diff, never a general nonempty plan.
						ExpectNonEmptyPlan: true,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PostApplyPreRefresh:  []plancheck.PlanCheck{serviceCLIReleasedZeroPlan{}},
							PostApplyPostRefresh: []plancheck.PlanCheck{serviceCLIReleasedZeroPlan{}},
						},
						Check: resource.ComposeTestCheckFunc(
							serviceCLIReleasedVersion(t, workspaces, "3.2.1"),
							// Measured SDK behavior: state/API contain the
							// allocated port, while the first output retains 0.
							api.checkVersionOutput("kubernetes_service_v1.test", &baseline, false, "3.2.1", []int32{0}),
						),
					},
					{
						PreConfig:                func() { api.setPhase("unsupported-zero", true) },
						ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
						Config:                   config,
						ExpectError:              regexp.MustCompile("Invalid zero node_port"),
					},
					{
						PreConfig: func() { api.setPhase("sdk-preparation", true) },
						ExternalProviders: map[string]resource.ExternalProvider{
							"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.2.1"},
						},
						Config: current,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{
								serviceCLIExactPlan{address: "kubernetes_service_v1.test", action: "no-op"},
							},
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: api.checkVersion("kubernetes_service_v1.test", &baseline, true, "3.2.1"),
					},
					{
						PreConfig:                func() { api.setPhase("migration", true) },
						ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
						Config:                   current, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
						Check: api.check("kubernetes_service_v1.test", &baseline, true),
					},
					{
						PreConfig:                func() { api.setPhase("follow-up", true) },
						ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
						Config:                   current, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
						Check: api.check("kubernetes_service_v1.test", &baseline, true),
					},
					{
						PreConfig:                func() { api.setPhase("cleanup", false) },
						ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
						Config:                   current, Destroy: true,
					},
				},
			})
		})
	}
}

func TestServiceCLI_ExplicitZeroNodePortCreate(t *testing.T) {
	for _, serviceType := range []string{"NodePort", "LoadBalancer"} {
		t.Run(serviceType, func(t *testing.T) {
			serviceCLIEnvironment(t)
			api := newServiceCLIAPI(t)
			config := serviceCLIConfig(api, serviceCLIZeroNodePort(serviceType), "kubernetes_service_v1", "test")
			current := strings.Replace(config, "      node_port = 0\n", "", 1)
			serviceCLIRejectAndRecover(t, api, config, current, "Invalid zero node_port")
		})
	}
}

func TestServiceCLI_NonAllocatingZeroNodePort(t *testing.T) {
	for _, variant := range []serviceCLIVariant{
		{name: "cluster-ip", fields: `type = "ClusterIP"`},
		{name: "load-balancer-allocation-false", fields: `type = "LoadBalancer"
allocate_load_balancer_node_ports = false`},
		{name: "external-name", fields: `type = "ExternalName"
external_name = "upstream.example.test"`},
	} {
		variant.ports = `port {
  port = 80
  node_port = 0
}`
		for _, operation := range []string{"create", "upgrade"} {
			t.Run(variant.name+"/"+operation, func(t *testing.T) {
				if operation == "upgrade" {
					serviceCLIMigration(t, "3.2.1", variant, false)
					return
				}
				serviceCLIEnvironment(t)
				api := newServiceCLIAPI(t)
				config := serviceCLIConfig(api, variant, "kubernetes_service_v1", "test")
				serviceCLICreateAndNoWritePlan(t, api, config)
			})
		}
	}
}

// Loopback CLI collection and output regressions.

func serviceCLIEmptyCollections(serviceType string) serviceCLIVariant {
	return serviceCLIVariant{
		name: serviceType,
		fields: fmt.Sprintf(`
    type = %q
    external_ips = []
    load_balancer_source_ranges = []
`, serviceType),
		ports: `port { port = 80 }`,
	}
}

func serviceCLIEmptyMetadataAndSelector(config string) string {
	config = strings.Replace(config, `labels = { app = "service-cli" }`, `labels = {}`, 1)
	config = strings.Replace(config, `annotations = { managed = "original" }`, `annotations = {}`, 1)
	return strings.Replace(config, `selector = { app = "service-cli" }`, `selector = {}`, 1)
}

func TestServiceCLI_ExplicitEmptyCollectionsUpgrade(t *testing.T) {
	for _, serviceType := range []string{"ClusterIP", "LoadBalancer"} {
		t.Run(serviceType, func(t *testing.T) {
			serviceCLIMigrationWithConfig(t, "3.2.1", serviceCLIEmptyCollections(serviceType), false,
				serviceCLIEmptyMetadataAndSelector, serviceCLIEmptyCollectionNormalization{
					allowedPaths: []string{
						"metadata.0.annotations", "metadata.0.labels", "spec.0.selector",
						"spec.0.external_ips", "spec.0.load_balancer_source_ranges",
					},
				})
		})
	}
}

func TestServiceCLI_ExplicitEmptyCollectionsCreate(t *testing.T) {
	for _, serviceType := range []string{"ClusterIP", "LoadBalancer"} {
		t.Run(serviceType, func(t *testing.T) {
			serviceCLIEnvironment(t)
			api := newServiceCLIAPI(t)
			config := serviceCLIEmptyMetadataAndSelector(serviceCLIConfig(api, serviceCLIEmptyCollections(serviceType), "kubernetes_service_v1", "test"))
			serviceCLICreateAndNoWritePlan(t, api, config)
		})
	}
}

func TestServiceCLI_EmptyMetadataUpgrade(t *testing.T) {
	for _, field := range []string{"annotations", "labels", "both"} {
		t.Run(field, func(t *testing.T) {
			transform := func(config string) string {
				if field == "annotations" || field == "both" {
					config = strings.Replace(config, `annotations = { managed = "original" }`, `annotations = {}`, 1)
				}
				if field == "labels" || field == "both" {
					config = strings.Replace(config, `labels = { app = "service-cli" }`, `labels = {}`, 1)
				}
				return config
			}
			paths := []string{"metadata.0." + field}
			if field == "both" {
				paths = []string{"metadata.0.annotations", "metadata.0.labels"}
			}
			serviceCLIMigrationWithConfig(t, "3.2.1", serviceCLIVariants()[0], false, transform,
				serviceCLIEmptyCollectionNormalization{allowedPaths: paths})
		})
	}
}

func TestServiceCLI_EmptySpecCollectionUpgrade(t *testing.T) {
	for _, field := range []string{"selector", "external_ips", "load_balancer_source_ranges"} {
		t.Run(field, func(t *testing.T) {
			variant := serviceCLIVariants()[0]
			var transform func(string) string
			if field == "selector" {
				transform = func(config string) string {
					return strings.Replace(config, `selector = { app = "service-cli" }`, `selector = {}`, 1)
				}
			} else {
				variant.fields = field + " = []"
			}
			serviceCLIMigrationWithConfig(t, "3.2.1", variant, false, transform,
				serviceCLIEmptyCollectionNormalization{allowedPaths: []string{"spec.0." + field}})
		})
	}
}

func TestServiceCLI_RepeatedSDKApplyPreservesOmittedCollections(t *testing.T) {
	workspaces := serviceCLIEnvironment(t)
	api := newServiceCLIAPI(t)
	config := serviceCLIConfig(api, serviceCLIVariant{ports: "port { port = 80 }"}, "kubernetes_service_v1", "test")
	config = strings.Replace(config, "    labels = { app = \"service-cli\" }\n", "", 1)
	config = strings.Replace(config, "    annotations = { managed = \"original\" }\n", "", 1)
	released := map[string]resource.ExternalProvider{
		"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.2.1"},
	}
	var baseline *coreapi.Service
	var stored []byte
	stateSnapshot := serviceCLIStateCheckFunc(func(req statecheck.CheckStateRequest) error {
		raw, err := json.Marshal(req.State.Values)
		if err != nil {
			return err
		}
		if stored == nil {
			var expected tfjson.StateValues
			decoder := json.NewDecoder(bytes.NewReader(raw))
			decoder.UseNumber()
			if err := decoder.Decode(&expected); err != nil {
				return err
			}
			for _, r := range expected.RootModule.Resources {
				if r.Type != "kubernetes_service_v1" {
					continue
				}
				if r.SchemaVersion != 1 {
					return fmt.Errorf("baseline schema = %d, want 1", r.SchemaVersion)
				}
				r.SchemaVersion = 2
				if err := serviceExpectedAffinityObjects(r.AttributeValues); err != nil {
					return err
				}
				// The absent object has no sensitivity descendants in schema 2.
				var sensitivity map[string]interface{}
				if err := json.Unmarshal(r.SensitiveValues, &sensitivity); err != nil {
					return err
				}
				for _, spec := range sensitivity["spec"].([]interface{}) {
					delete(spec.(map[string]interface{}), "session_affinity_config")
				}
				r.SensitiveValues, err = json.Marshal(sensitivity)
				if err != nil {
					return err
				}

			}
			stored, err = json.Marshal(expected)
			return err
		}
		if !bytes.Equal(stored, raw) {
			return fmt.Errorf("migration changed state beyond the affinity object conversion:\nexpected=%s\nactual=%s", stored, raw)
		}
		return nil
	})
	resource.Test(t, resource.TestCase{
		IsUnitTest: true, CheckDestroy: api.checkDestroy, ErrorCheck: api.errorCheck,
		Steps: []resource.TestStep{
			{
				PreConfig:         func() { api.setPhase("baseline", false) },
				ExternalProviders: released, Config: config,
				Check: resource.ComposeTestCheckFunc(
					serviceCLIReleasedVersion(t, workspaces, "3.2.1"),
					api.checkVersion("kubernetes_service_v1.test", &baseline, false, "3.2.1"),
				),
			},
			{
				PreConfig:         func() { api.setPhase("sdk-second-apply", true) },
				ExternalProviders: released, Config: config,
				ConfigPlanChecks:  serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
				Check:             api.checkVersion("kubernetes_service_v1.test", &baseline, true, "3.2.1"),
				ConfigStateChecks: []statecheck.StateCheck{stateSnapshot},
			},
			{
				PreConfig:                func() { api.setPhase("migration", true) },
				ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories, Config: config,
				ConfigPlanChecks:  serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
				Check:             api.check("kubernetes_service_v1.test", &baseline, true),
				ConfigStateChecks: []statecheck.StateCheck{stateSnapshot},
			},
			{
				PreConfig:                func() { api.setPhase("follow-up", true) },
				ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories, Config: config,
				ConfigPlanChecks:  serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
				Check:             api.check("kubernetes_service_v1.test", &baseline, true),
				ConfigStateChecks: []statecheck.StateCheck{stateSnapshot},
			},
			{
				PreConfig:                func() { api.setPhase("cleanup", false) },
				ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
				Config:                   config, Destroy: true,
			},
		},
	})
}

func TestServiceCLI_IsolatedCollectionRemoval(t *testing.T) {
	for _, tc := range []struct {
		name, configured, patchPath string
	}{
		{"annotations", `annotations = { managed = "original" }`, "/metadata/annotations/managed"},
		{"labels", `labels = { app = "service-cli" }`, "/metadata/labels/app"},
		{"selector", `selector = { app = "service-cli" }`, "/spec/selector"},
		{"external-ips", `external_ips = ["192.0.2.11"]`, "/spec/externalIPs"},
		{"source-ranges", `load_balancer_source_ranges = ["192.0.2.0/24"]`, "/spec/loadBalancerSourceRanges"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			serviceCLIEnvironment(t)
			api := newServiceCLIAPI(t)
			variant := serviceCLIVariant{
				fields: "type = \"LoadBalancer\"\n    external_ips = [\"192.0.2.11\"]\n    load_balancer_source_ranges = [\"192.0.2.0/24\"]",
				ports:  "port { port = 80 }",
			}
			initial := serviceCLIConfig(api, variant, "kubernetes_service_v1", "test")
			removed := strings.Replace(initial, tc.configured, "", 1)
			if removed == initial {
				t.Fatal("removal fixture did not remove its configured field")
			}
			var baseline *coreapi.Service
			checkRemoval := func(_ *terraform.State) error {
				api.mu.Lock()
				defer api.mu.Unlock()
				actual := api.objects["service-loopback"]
				if actual == nil || baseline == nil || actual.UID != baseline.UID {
					return fmt.Errorf("collection removal lost Service identity")
				}
				expected := baseline.DeepCopy()
				actual = actual.DeepCopy()
				switch tc.name {
				case "annotations":
					delete(expected.Annotations, "managed")
				case "labels":
					expected.Labels = nil
					if len(actual.Labels) == 0 {
						actual.Labels = nil
					}
				case "selector":
					expected.Spec.Selector = nil
					if len(actual.Spec.Selector) == 0 {
						actual.Spec.Selector = nil
					}
				case "external-ips":
					expected.Spec.ExternalIPs = nil
					if len(actual.Spec.ExternalIPs) == 0 {
						actual.Spec.ExternalIPs = nil
					}
				case "source-ranges":
					expected.Spec.LoadBalancerSourceRanges = nil
					if len(actual.Spec.LoadBalancerSourceRanges) == 0 {
						actual.Spec.LoadBalancerSourceRanges = nil
					}
				}
				if !reflect.DeepEqual(expected.Spec, actual.Spec) ||
					!reflect.DeepEqual(expected.Annotations, actual.Annotations) ||
					!reflect.DeepEqual(expected.Labels, actual.Labels) ||
					!reflect.DeepEqual(expected.Status, actual.Status) {
					return fmt.Errorf("isolated collection removal changed unrelated API fields")
				}
				if actual.ResourceVersion == baseline.ResourceVersion {
					return fmt.Errorf("real collection removal did not change the API resource version")
				}
				if len(api.patches["removal"]) != 1 {
					return fmt.Errorf("collection removal must issue exactly one PATCH, got %d", len(api.patches["removal"]))
				}
				var operations []struct {
					Op   string `json:"op"`
					Path string `json:"path"`
				}
				if err := json.Unmarshal(api.patches["removal"][0], &operations); err != nil {
					return err
				}
				for _, op := range operations {
					if op.Op == "test" && op.Path == "/metadata/resourceVersion" {
						continue
					}
					if op.Path != tc.patchPath {
						return fmt.Errorf("collection removal patched unrelated path %s", op.Path)
					}
				}
				return nil
			}
			resource.Test(t, resource.TestCase{
				IsUnitTest: true, ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
				CheckDestroy: api.checkDestroy, ErrorCheck: api.errorCheck,
				Steps: []resource.TestStep{
					{Config: initial, Check: api.check("kubernetes_service_v1.test", &baseline, false)},
					{
						PreConfig: func() { api.setPhase("removal", false) },
						Config:    removed, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "update"),
						Check: resource.ComposeTestCheckFunc(api.check("kubernetes_service_v1.test", &baseline, false), checkRemoval),
					},
					{
						PreConfig: func() { api.setPhase("follow-up", true) },
						Config:    removed, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
						Check: resource.ComposeTestCheckFunc(api.check("kubernetes_service_v1.test", &baseline, false), checkRemoval),
					},
					{PreConfig: func() { api.setPhase("cleanup", false) }, Config: removed, Destroy: true},
				},
			})
		})
	}
}

func TestServiceCLI_EmptyCollectionOmissionPreservesEmptyState(t *testing.T) {
	for _, field := range []struct {
		name, expression, original, empty string
	}{
		{"annotations", "metadata[0].annotations", `annotations = { managed = "original" }`, "annotations = {}"},
		{"labels", "metadata[0].labels", `labels = { app = "service-cli" }`, "labels = {}"},
		{"selector", "spec[0].selector", `selector = { app = "service-cli" }`, "selector = {}"},
		{"external_ips", "spec[0].external_ips", "", "external_ips = []"},
		{"load_balancer_source_ranges", "spec[0].load_balancer_source_ranges", "", "load_balancer_source_ranges = []"},
	} {
		for _, target := range []string{"omit", "null"} {
			t.Run(field.name+"/"+target, func(t *testing.T) {
				workspaces := serviceCLIEnvironment(t)
				api := newServiceCLIAPI(t)
				variant := serviceCLIVariants()[0]
				if field.original == "" {
					variant.fields = field.empty
				}
				initial := serviceCLIConfig(api, variant, "kubernetes_service_v1", "test")
				if field.original != "" {
					initial = strings.Replace(initial, field.original, field.empty, 1)
				}
				// The second SDK apply establishes empty collection values.
				// Omission in the Framework configuration must preserve them.
				source := initial + fmt.Sprintf(`
output "collection_is_null" {
  value = kubernetes_service_v1.test.%s == null
}
output "collection_json" {
  value = jsonencode(kubernetes_service_v1.test.%s)
}
`, field.expression, field.expression)
				replacement := ""
				if target == "null" {
					replacement = field.name + " = null"
				}
				current := strings.Replace(source, field.empty, replacement, 1)
				released := map[string]resource.ExternalProvider{
					"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.2.1"},
				}
				encodedEmpty := "{}"
				if strings.HasSuffix(field.empty, "[]") {
					encodedEmpty = "[]"
				}
				outputs := []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("collection_is_null", knownvalue.Bool(false)),
					statecheck.ExpectKnownOutputValue("collection_json", knownvalue.StringExact(encodedEmpty)),
				}
				var baseline *coreapi.Service
				resource.Test(t, resource.TestCase{
					IsUnitTest: true, CheckDestroy: api.checkDestroy, ErrorCheck: api.errorCheck,
					Steps: []resource.TestStep{
						{
							PreConfig:         func() { api.setPhase("baseline", false) },
							ExternalProviders: released, Config: initial,
							Check: resource.ComposeTestCheckFunc(
								serviceCLIReleasedVersion(t, workspaces, "3.2.1"),
								api.checkVersion("kubernetes_service_v1.test", &baseline, false, "3.2.1"),
							),
						},
						{
							PreConfig:         func() { api.setPhase("sdk-consumers", true) },
							ExternalProviders: released, Config: source,
							ConfigPlanChecks: resource.ConfigPlanChecks{
								PreApply: []plancheck.PlanCheck{
									serviceCLIExactPlan{address: "kubernetes_service_v1.test", action: "no-op"},
								},
								PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
							},
							Check:             api.checkVersion("kubernetes_service_v1.test", &baseline, true, "3.2.1"),
							ConfigStateChecks: outputs,
						},
						{
							PreConfig:                func() { api.setPhase("migration", true) },
							ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
							Config:                   current, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
							Check: api.check("kubernetes_service_v1.test", &baseline, true), ConfigStateChecks: outputs,
						},
						{
							PreConfig:                func() { api.setPhase("follow-up", true) },
							ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
							Config:                   current, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
							Check: api.check("kubernetes_service_v1.test", &baseline, true), ConfigStateChecks: outputs,
						},
						{
							PreConfig:                func() { api.setPhase("cleanup", false) },
							ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
							Config:                   current, Destroy: true,
						},
					},
				})
			})
		}
	}
}

func TestServiceCLI_OptionalStringConsumersUpgrade(t *testing.T) {
	for _, test := range []struct {
		name        string
		expression  string
		serviceType string
	}{
		{"generate-name", "metadata[0].generate_name", "ClusterIP"},
		{"port-name", "spec[0].port[0].name", "ClusterIP"},
		{"app-protocol", "spec[0].port[0].app_protocol", "ClusterIP"},
		{"external-name", "spec[0].external_name", "ClusterIP"},
		{"load-balancer-ip", "spec[0].load_balancer_ip", "LoadBalancer"},
		{"load-balancer-class", "spec[0].load_balancer_class", "LoadBalancer"},
	} {
		t.Run(test.name, func(t *testing.T) {
			workspaces := serviceCLIEnvironment(t)
			api := newServiceCLIAPI(t)
			variant := serviceCLIVariant{
				fields: fmt.Sprintf("type = %q", test.serviceType),
				ports:  `port { port = 80 }`,
			}
			expression := "kubernetes_service_v1.test." + test.expression
			baseConfig := serviceCLIConfig(api, variant, "kubernetes_service_v1", "test")
			config := baseConfig + fmt.Sprintf(`
output "observed_optional_string" {
  value = %s
}
output "observed_optional_string_length" {
  value = length(%s)
}
`, expression, expression)
			// These checks inspect actual typed Terraform outputs. The legacy
			// flat-state shim alone cannot distinguish a null from "".
			outputChecks := []statecheck.StateCheck{
				statecheck.ExpectKnownOutputValue("observed_optional_string", knownvalue.StringExact("")),
				statecheck.ExpectKnownOutputValue("observed_optional_string_length", knownvalue.Int64Exact(0)),
			}
			var baseline *coreapi.Service
			resource.Test(t, resource.TestCase{
				IsUnitTest: true, CheckDestroy: api.checkDestroy, ErrorCheck: api.errorCheck,
				Steps: []resource.TestStep{
					{
						PreConfig: func() { api.setPhase("baseline", false) },
						ExternalProviders: map[string]resource.ExternalProvider{
							"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.2.1"},
						},
						// SDK omitted attributes are null before the first
						// create. Introduce consumers only after real SDK
						// Read has stored the observed empty strings.
						Config: baseConfig,
						Check: resource.ComposeTestCheckFunc(
							serviceCLIReleasedVersion(t, workspaces, "3.2.1"),
							api.checkVersion("kubernetes_service_v1.test", &baseline, false, "3.2.1"),
						),
					},
					{
						PreConfig: func() { api.setPhase("sdk-consumers", true) },
						ExternalProviders: map[string]resource.ExternalProvider{
							"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.2.1"},
						},
						Config: config,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply:             []plancheck.PlanCheck{serviceCLIExactPlan{address: "kubernetes_service_v1.test", action: "no-op"}},
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check:             api.checkVersion("kubernetes_service_v1.test", &baseline, true, "3.2.1"),
						ConfigStateChecks: outputChecks,
					},
					{
						PreConfig:                func() { api.setPhase("migration", true) },
						ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
						Config:                   config, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
						Check:             api.check("kubernetes_service_v1.test", &baseline, true),
						ConfigStateChecks: outputChecks,
					},
					{
						PreConfig:                func() { api.setPhase("follow-up", true) },
						ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
						Config:                   config, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
						Check:             api.check("kubernetes_service_v1.test", &baseline, true),
						ConfigStateChecks: outputChecks,
					},
					{
						PreConfig:                func() { api.setPhase("cleanup", false) },
						ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
						Config:                   config, Destroy: true,
					},
				},
			})
		})
	}
}

// CLI plan guards and their unit tests.

type serviceCLIExactPlan struct {
	address string
	from    string
	action  string
}

func (check serviceCLIExactPlan) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	found := false
	for _, change := range req.Plan.ResourceChanges {
		if change.Mode == "data" {
			continue
		}
		if change.Address != check.address || change.Change == nil ||
			len(change.Change.Actions) != 1 || string(change.Change.Actions[0]) != check.action ||
			change.PreviousAddress != check.from {
			detail, _ := json.Marshal(change)
			resp.Error = fmt.Errorf("expected only %s %s (moved from %q), got %s", check.action, check.address, check.from, detail)
			return
		}
		found = true
	}
	if !found {
		resp.Error = fmt.Errorf("plan did not contain expected resource %s", check.address)
	}
}

func serviceCLISettled(address, from, action string) resource.ConfigPlanChecks {
	checks := []plancheck.PlanCheck{serviceCLIExactPlan{address: address, from: from, action: action}}
	if action == "no-op" {
		checks = append(checks, plancheck.ExpectEmptyPlan())
	}
	return resource.ConfigPlanChecks{
		PreApply:             checks,
		PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
	}
}

// The SDK marker plan is observed, never applied. Its expected unsafe action
// and configured marker are checked while the API write guard is enabled.
type serviceCLIAllocationMarkerPlan struct{ marker string }

func (check serviceCLIAllocationMarkerPlan) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	if len(req.Plan.ResourceChanges) != 1 {
		resp.Error = fmt.Errorf("SDK marker control requires exactly one Service")
		return
	}
	change := req.Plan.ResourceChanges[0]
	if change.Address != "kubernetes_service_v1.test" || change.PreviousAddress != "" || change.Change == nil {
		resp.Error = fmt.Errorf("unexpected SDK marker resource or move")
		return
	}
	expectedAction := "[update]"
	if check.marker == "cluster-ips" {
		expectedAction = "[delete create]"
	}
	if fmt.Sprint(change.Change.Actions) != expectedAction {
		resp.Error = fmt.Errorf("SDK marker %s: expected %s, got %v", check.marker, expectedAction, change.Change.Actions)
		return
	}
	after, ok := change.Change.After.(map[string]any)
	if !ok {
		resp.Error = fmt.Errorf("SDK marker plan has no object")
		return
	}
	specs, ok := after["spec"].([]any)
	if !ok || len(specs) != 1 {
		resp.Error = fmt.Errorf("SDK marker plan has no singleton spec")
		return
	}
	spec, ok := specs[0].(map[string]any)
	if !ok {
		resp.Error = fmt.Errorf("SDK marker spec is not an object")
		return
	}
	field := map[string]string{"cluster-ips": "cluster_ips", "ip-families": "ip_families"}[check.marker]
	if field != "" {
		if values, ok := spec[field].([]any); !ok || len(values) != 0 {
			resp.Error = fmt.Errorf("SDK marker plan did not retain configured empty %s", field)
		}
		if check.marker == "cluster-ips" && fmt.Sprint(change.Change.ReplacePaths) != "[[spec 0 cluster_ips]]" {
			resp.Error = fmt.Errorf("SDK replacement was not solely caused by cluster_ips: %v", change.Change.ReplacePaths)
		}
		return
	}
	ports, ok := spec["port"].([]any)
	if !ok || len(ports) != 1 {
		resp.Error = fmt.Errorf("SDK zero-target plan has no singleton port")
		return
	}
	port, ok := ports[0].(map[string]any)
	if !ok || !reflect.DeepEqual(port["target_port"], "0") {
		resp.Error = fmt.Errorf("SDK zero-target plan did not retain its configured marker")
	}
}

type serviceCLIReleasedZeroPlan struct{}

func (serviceCLIReleasedZeroPlan) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	if len(req.Plan.ResourceChanges) != 1 {
		resp.Error = fmt.Errorf("literal-zero SDK control requires exactly one resource")
		return
	}
	change := req.Plan.ResourceChanges[0]
	if change.Address != "kubernetes_service_v1.test" || change.PreviousAddress != "" || change.Change == nil ||
		len(change.Change.Actions) != 1 || string(change.Change.Actions[0]) != "update" {
		resp.Error = fmt.Errorf("literal-zero SDK control permits only an in-place Service update")
		return
	}
	if serviceCLIPlanContainsUnknown(change.Change.AfterUnknown) {
		resp.Error = fmt.Errorf("SDK literal-zero control unexpectedly introduced unknown values")
		return
	}
	values := make([]map[string]any, 2)
	for i, value := range []any{change.Change.Before, change.Change.After} {
		data, err := json.Marshal(value)
		if err != nil {
			resp.Error = err
			return
		}
		if err := json.Unmarshal(data, &values[i]); err != nil {
			resp.Error = err
			return
		}
	}

	specs := make([]map[string]any, 2)
	ports := make([]map[string]any, 2)
	for i, value := range values {
		list, ok := value["spec"].([]any)
		if !ok || len(list) != 1 {
			resp.Error = fmt.Errorf("literal-zero SDK control has an unexpected spec shape")
			return
		}
		specs[i], ok = list[0].(map[string]any)
		if !ok {
			resp.Error = fmt.Errorf("literal-zero SDK spec is not an object")
			return
		}
		list, ok = specs[i]["port"].([]any)
		if !ok || len(list) != 1 {
			resp.Error = fmt.Errorf("literal-zero SDK control changed port cardinality")
			return
		}
		ports[i], ok = list[0].(map[string]any)
		if !ok {
			resp.Error = fmt.Errorf("literal-zero SDK port is not an object")
			return
		}
	}
	before, beforeOK := ports[0]["node_port"].(float64)
	after, afterOK := ports[1]["node_port"].(float64)
	if !beforeOK || !afterOK || before <= 0 || after != 0 {
		resp.Error = fmt.Errorf("SDK literal-zero control did not show allocated nodePort -> 0")
		return
	}
	ports[0]["node_port"] = after
	for _, field := range []string{"external_ips", "load_balancer_source_ranges"} {
		if reflect.DeepEqual(specs[0][field], specs[1][field]) {
			continue
		}
		empty, ok := specs[1][field].([]any)
		if specs[0][field] != nil || !ok || len(empty) != 0 {
			resp.Error = fmt.Errorf("unexpected SDK zero-control set change: %s", field)
			return
		}
		specs[0][field] = specs[1][field]
	}
	if !reflect.DeepEqual(values[0], values[1]) {
		resp.Error = fmt.Errorf("SDK literal-zero control changed fields beyond nodePort and empty-set normalization")
		return
	}
	for name, output := range req.Plan.OutputChanges {
		if !output.Actions.NoOp() {
			resp.Error = fmt.Errorf("SDK literal-zero control unexpectedly changed output %s", name)
			return
		}
	}
}

func serviceCLIPlanContainsUnknown(value any) bool {
	switch value := value.(type) {
	case bool:
		return value
	case map[string]any:
		for _, child := range value {
			if serviceCLIPlanContainsUnknown(child) {
				return true
			}
		}
	case []any:
		for _, child := range value {
			if serviceCLIPlanContainsUnknown(child) {
				return true
			}
		}
	}
	return false
}

// This exception is only for explicitly configured empty collections. SDKv2
// reads them as null; Framework must honor the configured {} or []. It permits
// no other known value change and the API's migration write guard remains on.
type serviceCLIEmptyCollectionNormalization struct {
	allowedPaths []string
}

func (check serviceCLIEmptyCollectionNormalization) CheckPlan(ctx context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	if len(req.Plan.ResourceChanges) != 1 {
		resp.Error = fmt.Errorf("empty-collection normalization requires exactly one resource")
		return
	}
	change := req.Plan.ResourceChanges[0]
	if change.Address != "kubernetes_service_v1.test" || change.PreviousAddress != "" || change.Change == nil {
		resp.Error = fmt.Errorf("unexpected resource or move in empty-collection normalization")
		return
	}
	if change.Change.Actions.NoOp() {
		plancheck.ExpectEmptyPlan().CheckPlan(ctx, req, resp)
		return
	}
	if len(change.Change.Actions) != 1 || string(change.Change.Actions[0]) != "update" {
		resp.Error = fmt.Errorf("empty-collection normalization cannot perform %v", change.Change.Actions)
		return
	}
	supported := map[string]bool{
		"metadata.0.annotations": true, "metadata.0.labels": true,
		"spec.0.selector": true, "spec.0.external_ips": true, "spec.0.load_balancer_source_ranges": true,
	}
	collections := map[string]bool{}
	for _, path := range check.allowedPaths {
		if !supported[path] {
			resp.Error = fmt.Errorf("unsupported empty-collection normalization path %s", path)
			return
		}
		collections[path] = true
	}
	computed := map[string]bool{
		"metadata.0.generation": true, "metadata.0.resource_version": true,
		"spec.0.cluster_ip": true, "spec.0.cluster_ips": true,
		"spec.0.external_traffic_policy": true, "spec.0.health_check_node_port": true,
		"spec.0.internal_traffic_policy": true, "spec.0.ip_families": true, "spec.0.ip_family_policy": true,
		"spec.0.port.0.node_port": true, "spec.0.port.0.target_port": true,
		"spec.0.session_affinity_config": true, "status": true,
	}
	normalizations := 0
	var walk func(string, any, any, any) error
	walk = func(path string, before, after, unknown any) error {
		if unknown == true {
			if computed[path] {
				return nil
			}
			return fmt.Errorf("unapproved unknown value at %s", path)
		}
		if reflect.DeepEqual(before, after) && !serviceCLIPlanContainsUnknown(unknown) {
			return nil
		}
		if before == nil && collections[path] {
			switch value := after.(type) {
			case map[string]any:
				if len(value) == 0 {
					normalizations++
					return nil
				}
			case []any:
				if len(value) == 0 {
					normalizations++
					return nil
				}
			}
		}
		if left, ok := before.(map[string]any); ok {
			right, rightOK := after.(map[string]any)
			if !rightOK {
				return fmt.Errorf("unexpected object change at %s", path)
			}
			unknownMap, _ := unknown.(map[string]any)
			keys := map[string]bool{}
			for key := range left {
				keys[key] = true
			}
			for key := range right {
				keys[key] = true
			}
			for key := range unknownMap {
				keys[key] = true
			}
			for key := range keys {
				child := key
				if path != "" {
					child = path + "." + key
				}
				if err := walk(child, left[key], right[key], unknownMap[key]); err != nil {
					return err
				}
			}
			return nil
		}
		if left, ok := before.([]any); ok {
			right, rightOK := after.([]any)
			if !rightOK || len(left) != len(right) {
				return fmt.Errorf("unexpected collection cardinality change at %s", path)
			}
			unknownList, _ := unknown.([]any)
			for i := range left {
				var childUnknown any
				if i < len(unknownList) {
					childUnknown = unknownList[i]
				}
				if err := walk(fmt.Sprintf("%s.%d", path, i), left[i], right[i], childUnknown); err != nil {
					return err
				}
			}
			return nil
		}
		return fmt.Errorf("unapproved known value change at %s: %#v -> %#v", path, before, after)
	}
	if err := walk("", change.Change.Before, change.Change.After, change.Change.AfterUnknown); err != nil {
		resp.Error = err
		return
	}
	if normalizations == 0 {
		resp.Error = fmt.Errorf("state-only update contains no explicit empty-collection normalization")
		return
	}
	for name, output := range req.Plan.OutputChanges {
		if output.Actions.NoOp() {
			continue
		}
		expectedUnknown := name == "service_cluster_ip" && output.AfterUnknown == true
		if name == "service_nodeports" {
			values, ok := output.AfterUnknown.([]any)
			expectedUnknown = ok && len(values) == 1 && values[0] == true
		}
		if len(output.Actions) != 1 || string(output.Actions[0]) != "update" || !expectedUnknown {
			resp.Error = fmt.Errorf("unapproved output change during empty-collection normalization: %s", name)
			return
		}
	}
}

func TestServiceCLIPlan_EmptyNormalizationGuard(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func(*tfjson.Plan)
		fail bool
	}{
		{name: "exact-empty-maps"},
		{name: "nonempty-selector", fail: true, edit: func(plan *tfjson.Plan) {
			plan.ResourceChanges[0].Change.After.(map[string]any)["spec"].([]any)[0].(map[string]any)["selector"] = map[string]any{"app": "changed"}
		}},
		{name: "port-change", fail: true, edit: func(plan *tfjson.Plan) {
			plan.ResourceChanges[0].Change.Before.(map[string]any)["spec"].([]any)[0].(map[string]any)["port"] = []any{map[string]any{"port": 80}}
			plan.ResourceChanges[0].Change.After.(map[string]any)["spec"].([]any)[0].(map[string]any)["port"] = []any{map[string]any{"port": 81}}
		}},
		{name: "replacement", fail: true, edit: func(plan *tfjson.Plan) {
			plan.ResourceChanges[0].Change.Actions = tfjson.Actions{tfjson.ActionDelete, tfjson.ActionCreate}
		}},
		{name: "unknown-uid", fail: true, edit: func(plan *tfjson.Plan) {
			plan.ResourceChanges[0].Change.AfterUnknown = map[string]any{"metadata": []any{map[string]any{"uid": true}}}
		}},
		{name: "unknown-uid-in-otherwise-unchanged-metadata", fail: true, edit: func(plan *tfjson.Plan) {
			plan.ResourceChanges[0].Change.Before.(map[string]any)["metadata"] =
				plan.ResourceChanges[0].Change.After.(map[string]any)["metadata"]
			plan.ResourceChanges[0].Change.AfterUnknown = map[string]any{"metadata": []any{map[string]any{"uid": true}}}
		}},
		{name: "changed-uid-output", fail: true, edit: func(plan *tfjson.Plan) {
			plan.OutputChanges = map[string]*tfjson.Change{
				"service_uid": {Actions: tfjson.Actions{tfjson.ActionUpdate}, Before: "old", After: "new"},
			}
		}},
		{name: "update-without-normalization", fail: true, edit: func(plan *tfjson.Plan) {
			plan.ResourceChanges[0].Change.Before = plan.ResourceChanges[0].Change.After
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			plan := &tfjson.Plan{ResourceChanges: []*tfjson.ResourceChange{{
				Address: "kubernetes_service_v1.test",
				Change: &tfjson.Change{
					Actions: tfjson.Actions{tfjson.ActionUpdate},
					Before: map[string]any{
						"metadata": []any{map[string]any{"labels": nil}},
						"spec":     []any{map[string]any{"selector": nil}},
					},
					After: map[string]any{
						"metadata": []any{map[string]any{"labels": map[string]any{}}},
						"spec":     []any{map[string]any{"selector": map[string]any{}}},
					},
				},
			}}}
			if test.edit != nil {
				test.edit(plan)
			}
			var response plancheck.CheckPlanResponse
			serviceCLIEmptyCollectionNormalization{
				allowedPaths: []string{"metadata.0.labels", "spec.0.selector"},
			}.CheckPlan(context.Background(), plancheck.CheckPlanRequest{Plan: plan}, &response)
			if (response.Error != nil) != test.fail {
				t.Fatalf("guard error=%v, wanted failure=%t", response.Error, test.fail)
			}
		})
	}
}

func TestServiceCLIPlan_EmptyNormalizationExactPaths(t *testing.T) {
	for _, field := range []struct {
		path  string
		empty any
	}{
		{"metadata.0.annotations", map[string]any{}},
		{"metadata.0.labels", map[string]any{}},
		{"spec.0.selector", map[string]any{}},
		{"spec.0.external_ips", []any{}},
		{"spec.0.load_balancer_source_ranges", []any{}},
	} {
		for _, otherPath := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/other-path-%t", field.path, otherPath), func(t *testing.T) {
				parts := strings.Split(field.path, ".")
				plan := &tfjson.Plan{ResourceChanges: []*tfjson.ResourceChange{{
					Address: "kubernetes_service_v1.test",
					Change: &tfjson.Change{
						Actions: tfjson.Actions{tfjson.ActionUpdate},
						Before:  map[string]any{parts[0]: []any{map[string]any{parts[2]: nil}}},
						After:   map[string]any{parts[0]: []any{map[string]any{parts[2]: field.empty}}},
					},
				}}}
				allowed := field.path
				if otherPath {
					allowed = "metadata.0.labels"
					if field.path == allowed {
						allowed = "spec.0.selector"
					}
				}
				var response plancheck.CheckPlanResponse
				serviceCLIEmptyCollectionNormalization{
					allowedPaths: []string{allowed},
				}.CheckPlan(context.Background(), plancheck.CheckPlanRequest{Plan: plan}, &response)
				if (response.Error != nil) != otherPath {
					t.Fatalf("normalizing %s with allowed path %s: %v", field.path, allowed, response.Error)
				}
			})
		}
	}
}

// Shared CLI harness and state assertions.

// These tests execute real Terraform against only a loopback HTTP fixture.
// Process-global CLI configuration/credentials make parallel execution unsafe.
var serviceCLIProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"kubernetes": func() (tfprotov6.ProviderServer, error) {
		return mux.MuxServer(context.Background(), "service-cli-test")
	},
}

func serviceCLIEnvironment(t *testing.T) string {
	t.Helper()
	if os.Getenv("TF_KUBERNETES_SERVICE_CLI_TESTS") != "1" {
		t.Skip("Service CLI integration tests require TF_KUBERNETES_SERVICE_CLI_TESTS=1; select TestServiceCLI with -timeout=30m")
	}
	// A short temp path keeps provider reattachment sockets within macOS's limit,
	// including when the checkout itself is a deeply nested worktree.
	t.Setenv("TMPDIR", "/tmp")
	dir := t.TempDir()
	t.Setenv("TF_ACC_TEMP_DIR", dir)
	t.Setenv("CHECKPOINT_DISABLE", "1")
	t.Setenv("TF_IN_AUTOMATION", "1")
	t.Setenv("TF_ACC", "")
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "KUBE") || strings.HasPrefix(name, "TF_CLI_ARGS") ||
			strings.HasPrefix(name, "TF_VAR_") || name == "TF_REATTACH_PROVIDERS" ||
			name == "TF_PLUGIN_CACHE_DIR" || name == "TF_ACC_TERRAFORM_VERSION" {
			t.Setenv(name, "")
		}
	}
	cli, err := exec.LookPath("terraform")
	if err != nil {
		t.Fatal("Service CLI regression tests require an installed Terraform CLI:", err)
	}
	t.Setenv("TF_ACC_TERRAFORM_PATH", cli)
	config := filepath.Join(dir, "terraformrc")
	if err := os.WriteFile(config, []byte("disable_checkpoint = true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	// Never inherit dev_overrides: they can silently substitute the local
	// binary for an ostensibly pinned ExternalProviders baseline.
	t.Setenv("TF_CLI_CONFIG_FILE", config)
	return dir
}

type serviceCLIVariant struct {
	name   string
	fields string
	ports  string
}

func serviceCLIVariants() []serviceCLIVariant {
	return []serviceCLIVariant{
		{name: "cluster-ip-omitted", ports: `port { port = 80 }`},
		{name: "cluster-ip-explicit", fields: `
    type = "ClusterIP"
    cluster_ip = "10.96.0.42"
    cluster_ips = ["10.96.0.42"]
    publish_not_ready_addresses = false
    internal_traffic_policy = "Local"
    session_affinity = "None"
`, ports: `port {
      port = 80
      target_port = "8080"
    }`},
		{name: "headless", fields: `
    cluster_ip = "None"
    publish_not_ready_addresses = true
`, ports: `port {
      name = "peer"
      port = 2380
      target_port = "peer"
    }`},
		{name: "external-name", fields: `
    type = "ExternalName"
    external_name = "upstream.example.test"
`},
		{name: "node-port-allocated", fields: `type = "NodePort"`, ports: `port { port = 80 }`},
		{name: "node-port-fixed", fields: `
    type = "NodePort"
    external_traffic_policy = "Local"
    internal_traffic_policy = "Cluster"
    external_ips = ["192.0.2.11", "192.0.2.12"]
`, ports: `port {
      name = "web"
      port = 80
      target_port = "http"
      node_port = 30080
      app_protocol = "http"
    }
    port {
      name = "dns"
      port = 53
      protocol = "UDP"
      target_port = "5353"
      node_port = 30053
    }`},
		{name: "load-balancer", fields: `type = "LoadBalancer"`, ports: `port { port = 80 }`},
		{name: "load-balancer-local-fixed", fields: `
    type = "LoadBalancer"
    external_traffic_policy = "Local"
    health_check_node_port = 30123
    allocate_load_balancer_node_ports = true
    load_balancer_ip = "192.0.2.50"
    load_balancer_source_ranges = ["192.0.2.0/24", "198.51.100.0/24"]
`, ports: `port {
      port = 443
      target_port = "8443"
      node_port = 30443
    }`},
		{name: "load-balancer-no-nodeports-class", fields: `
    type = "LoadBalancer"
    allocate_load_balancer_node_ports = false
    load_balancer_class = "example.test/private"
`, ports: `port {
      port = 443
      target_port = "https"
    }`},
		{name: "dual-stack", fields: `
    ip_family_policy = "RequireDualStack"
    ip_families = ["IPv4", "IPv6"]
    cluster_ip = "10.96.0.43"
    cluster_ips = ["10.96.0.43", "fd00::43"]
`, ports: `port { port = 80 }`},
		{name: "client-ip-omitted", fields: `session_affinity = "ClientIP"`, ports: `port { port = 80 }`},
		{name: "client-ip-empty-affinity", fields: `
    session_affinity = "ClientIP"
    session_affinity_config {}
`, ports: `port { port = 80 }`},
		{name: "client-ip-empty-nested", fields: `
    session_affinity = "ClientIP"
    session_affinity_config {
      client_ip {}
    }
`, ports: `port { port = 80 }`},
		{name: "client-ip-timeout", fields: `
    session_affinity = "ClientIP"
    session_affinity_config {
      client_ip {
        timeout_seconds = 60
      }
    }
`, ports: `port { port = 80 }`},
		{name: "dual-stack-allocated", fields: `
    ip_family_policy = "PreferDualStack"
`, ports: `port { port = 80 }`},
		{name: "load-balancer-allocation-false-explicit-nodeport", fields: `
    type = "LoadBalancer"
    allocate_load_balancer_node_ports = false
`, ports: `port {
      port = 443
      target_port = "https"
      node_port = 30443
    }`},
	}
}

func serviceCLIConfig(api *serviceCLIAPI, variant serviceCLIVariant, resourceType, label string) string {
	return fmt.Sprintf(`
provider "kubernetes" {
  host = %q
}
resource %q %q {
  metadata {
    name = "service-loopback"
    namespace = "default"
    labels = { app = "service-cli" }
    annotations = { managed = "original" }
  }
  wait_for_load_balancer = false
  spec {
    selector = { app = "service-cli" }
    %s
    %s
  }
}
output "service_uid" {
  value = %s.%s.metadata[0].uid
}
output "service_cluster_ip" {
  value = %s.%s.spec[0].cluster_ip
}
output "service_nodeports" {
  value = [for port in %s.%s.spec[0].port : port.node_port]
}
`, api.server.URL, resourceType, label, variant.fields, variant.ports, resourceType, label, resourceType, label, resourceType, label)
}

func serviceCLIReleasedVersion(t *testing.T, workspaces, version string) resource.TestCheckFunc {
	t.Helper()
	return func(_ *terraform.State) error {
		found := false
		pinned := regexp.MustCompile(`(?s)provider "registry.terraform.io/hashicorp/kubernetes" \{\s*version\s*=\s*"` + regexp.QuoteMeta(version) + `"`)
		err := filepath.WalkDir(workspaces, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.Name() != ".terraform.lock.hcl" {
				return nil
			}
			lock, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if !pinned.Match(lock) {
				return fmt.Errorf("released-provider lock is not exactly hashicorp/kubernetes %s: %s", version, path)
			}
			found = true
			return nil
		})
		if err != nil {
			return err
		}
		if !found {
			return fmt.Errorf("released provider dependency lock was not found")
		}
		t.Logf("verified registry.terraform.io/hashicorp/kubernetes = %s with clean CLI config", version)
		return nil
	}
}

func (api *serviceCLIAPI) check(address string, baseline **coreapi.Service, unchanged bool) resource.TestCheckFunc {
	return api.checkVersion(address, baseline, unchanged, "local")
}

func (api *serviceCLIAPI) checkVersion(address string, baseline **coreapi.Service, unchanged bool, version string) resource.TestCheckFunc {
	return api.checkVersionOutput(address, baseline, unchanged, version, nil)
}

func (api *serviceCLIAPI) checkVersionOutput(address string, baseline **coreapi.Service, unchanged bool, version string, nodePortOutput []int32) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		api.mu.Lock()
		defer api.mu.Unlock()
		object := api.objects["service-loopback"]
		if object == nil || object.UID == "" {
			return fmt.Errorf("Service missing from loopback API")
		}
		if *baseline == nil {
			if api.phase == "baseline" || api.phase == "create" {
				for request, count := range api.calls[api.phase] {
					if strings.HasPrefix(request, "POST ") {
						if count != len(api.objects) {
							return fmt.Errorf("baseline/create POST count %d differs from object count %d", count, len(api.objects))
						}
					} else if !strings.HasPrefix(request, "GET ") {
						return fmt.Errorf("unexpected baseline/create write: %d %s", count, request)
					}
				}
			}
			*baseline = object.DeepCopy()
		} else if object.UID != (*baseline).UID {
			return fmt.Errorf("Service UID changed: %s -> %s", (*baseline).UID, object.UID)
		}
		if unchanged && (!reflect.DeepEqual((*baseline).Spec, object.Spec) ||
			!reflect.DeepEqual((*baseline).Annotations, object.Annotations) ||
			!reflect.DeepEqual((*baseline).Status, object.Status)) {
			before, _ := json.Marshal(*baseline)
			after, _ := json.Marshal(object)
			return fmt.Errorf("Service migration changed API spec, annotations or status:\nbefore=%s\nafter=%s", before, after)
		}
		if object.Annotations["controller.kubernetes.io/observed"] != "preserve" ||
			object.Spec.TrafficDistribution == nil || *object.Spec.TrafficDistribution != "PreferClose" {
			return fmt.Errorf("controller annotation or unmodeled trafficDistribution was lost")
		}
		for _, phase := range []string{"migration", "follow-up", "import", "unsupported-move", "unsupported-syntax", "sdk-consumers", "affinity-omission", "sdk-affinity-omission", "sdk-import"} {
			for request, count := range api.calls[phase] {
				if !strings.HasPrefix(request, "GET ") && count != 0 {
					return fmt.Errorf("unchanged phase %s performed %d %s requests", phase, count, request)
				}
			}
		}
		attrs := map[string]string{
			"id": "default/service-loopback", "metadata.0.uid": string(object.UID),
			"spec.0.cluster_ip": object.Spec.ClusterIP, "spec.0.type": string(object.Spec.Type),
			"spec.0.cluster_ips.#":          strconv.Itoa(len(object.Spec.ClusterIPs)),
			"spec.0.port.#":                 strconv.Itoa(len(object.Spec.Ports)),
			"spec.0.health_check_node_port": strconv.Itoa(int(object.Spec.HealthCheckNodePort)),
		}
		for i, ip := range object.Spec.ClusterIPs {
			attrs[fmt.Sprintf("spec.0.cluster_ips.%d", i)] = ip
		}
		for i, port := range object.Spec.Ports {
			prefix := fmt.Sprintf("spec.0.port.%d.", i)
			attrs[prefix+"port"] = strconv.Itoa(int(port.Port))
			attrs[prefix+"target_port"] = port.TargetPort.String()
			attrs[prefix+"node_port"] = strconv.Itoa(int(port.NodePort))
			attrs[prefix+"protocol"] = string(port.Protocol)
			attrs[prefix+"name"] = port.Name
		}
		if object.Spec.Type == coreapi.ServiceTypeLoadBalancer {
			attrs["status.0.load_balancer.0.ingress.0.ip"] = "192.0.2.50"
			attrs["status.0.load_balancer.0.ingress.0.hostname"] = "loopback.example.test"
			// ip_mode was not exposed by 2.37.1. The complete API status is
			// still snapshotted, and the migrated/current state must expose it.
			if version != "2.37.1" {
				attrs["status.0.load_balancer.0.ingress.0.ip_mode"] = "VIP"
			}
		}
		if object.Spec.SessionAffinity == coreapi.ServiceAffinityClientIP {
			key := "spec.0.session_affinity_config.client_ip.timeout_seconds"
			if _, exists := state.RootModule().Resources[address].Primary.Attributes[key]; !exists {
				key = "spec.0.session_affinity_config.0.client_ip.0.timeout_seconds"
			}
			attrs[key] = strconv.Itoa(int(*object.Spec.SessionAffinityConfig.ClientIP.TimeoutSeconds))
		}
		for key, expected := range attrs {
			if err := resource.TestCheckResourceAttr(address, key, expected)(state); err != nil {
				return err
			}
		}
		if err := resource.TestCheckOutput("service_uid", string(object.UID))(state); err != nil {
			return err
		}
		if err := resource.TestCheckOutput("service_cluster_ip", object.Spec.ClusterIP)(state); err != nil {
			return err
		}
		nodePorts := make([]int32, len(object.Spec.Ports))
		for i, port := range object.Spec.Ports {
			nodePorts[i] = port.NodePort
		}
		if nodePortOutput != nil {
			nodePorts = nodePortOutput
		}
		output, ok := state.RootModule().Outputs["service_nodeports"]
		if !ok || fmt.Sprint(output.Value) != fmt.Sprint(nodePorts) {
			return fmt.Errorf("node-port output is not known or does not match API allocations: %#v, expected %v", output, nodePorts)
		}
		return nil
	}
}

func (api *serviceCLIAPI) checkDestroy(_ *terraform.State) error {
	api.mu.Lock()
	defer api.mu.Unlock()
	if len(api.objects) != 0 {
		return fmt.Errorf("Service still exists after Terraform destroy")
	}
	return nil
}

func (api *serviceCLIAPI) errorCheck(err error) error {
	// A rejected plan must still be able to destroy the baseline fixture.
	// Preserve the original error; never turn a failed migration into success.
	api.setPhase("cleanup", false)
	return err
}

type serviceCLIStateCheckFunc func(statecheck.CheckStateRequest) error

func (check serviceCLIStateCheckFunc) CheckState(_ context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	resp.Error = check(req)
}

// The testing library's deprecated terraform.State shim rejects string-indexed
// addresses or modules. Use the real JSON state instead, not rewritten state.
func (api *serviceCLIAPI) checkJSONAddress(address string, baseline **coreapi.Service) statecheck.StateCheck {
	return serviceCLIStateCheckFunc(func(req statecheck.CheckStateRequest) error {
		api.mu.Lock()
		defer api.mu.Unlock()
		object := api.objects["service-loopback"]
		if object == nil {
			return fmt.Errorf("indexed Service missing from loopback API")
		}
		if *baseline == nil {
			*baseline = object.DeepCopy()
		}
		if object.UID != (*baseline).UID || !reflect.DeepEqual(object.Spec, (*baseline).Spec) ||
			!reflect.DeepEqual(object.Annotations, (*baseline).Annotations) || !reflect.DeepEqual(object.Status, (*baseline).Status) {
			return fmt.Errorf("indexed move changed Service UID/spec/controller metadata/status")
		}
		for _, phase := range []string{"migration", "follow-up"} {
			for request := range api.calls[phase] {
				if !strings.HasPrefix(request, "GET ") {
					return fmt.Errorf("indexed move wrote during %s: %s", phase, request)
				}
			}
		}
		var resources []*tfjson.StateResource
		modules := []*tfjson.StateModule{req.State.Values.RootModule}
		for len(modules) != 0 {
			module := modules[0]
			modules = append(modules[1:], module.ChildModules...)
			resources = append(resources, module.Resources...)
		}
		for _, res := range resources {
			if res.Address != address {
				continue
			}
			metadata, ok := res.AttributeValues["metadata"].([]any)
			if !ok || len(metadata) != 1 {
				return fmt.Errorf("indexed state has no known metadata")
			}
			metamap, ok := metadata[0].(map[string]any)
			if !ok || metamap["uid"] != string(object.UID) {
				return fmt.Errorf("indexed state UID differs from API")
			}
			specs, ok := res.AttributeValues["spec"].([]any)
			if !ok || len(specs) != 1 {
				return fmt.Errorf("indexed state has no known spec")
			}
			spec, ok := specs[0].(map[string]any)
			if !ok {
				return fmt.Errorf("indexed state spec is not an object")
			}
			before, _ := json.Marshal(object.Spec.ClusterIPs)
			after, _ := json.Marshal(spec["cluster_ips"])
			if string(before) != string(after) || spec["cluster_ip"] != object.Spec.ClusterIP {
				return fmt.Errorf("indexed state lost allocated cluster IPs")
			}
			ports, ok := spec["port"].([]any)
			if !ok || len(ports) != len(object.Spec.Ports) {
				return fmt.Errorf("indexed state lost Service ports")
			}
			for i, port := range object.Spec.Ports {
				actual, ok := ports[i].(map[string]any)
				if !ok || fmt.Sprint(actual["node_port"]) != strconv.Itoa(int(port.NodePort)) ||
					fmt.Sprint(actual["port"]) != strconv.Itoa(int(port.Port)) || actual["target_port"] != port.TargetPort.String() {
					return fmt.Errorf("indexed state lost port identity at %d", i)
				}
			}
			output, ok := req.State.Values.Outputs["service_uid"]
			if !ok || output.Value != string(object.UID) {
				return fmt.Errorf("indexed UID output is not known and stable")
			}
			return nil
		}
		return fmt.Errorf("indexed state lacks %s", address)
	})
}

func serviceCLIExpectNoAPICalls(t *testing.T, api *serviceCLIAPI, phase string) {
	t.Helper()
	t.Cleanup(func() {
		api.mu.Lock()
		defer api.mu.Unlock()
		if len(api.calls[phase]) != 0 {
			t.Errorf("invalid configuration reached the API during %s: %v", phase, api.calls[phase])
		}
	})
}

func serviceCLIRejectAndRecover(t *testing.T, api *serviceCLIAPI, invalid, current, diagnostic string) {
	t.Helper()
	serviceCLIExpectNoAPICalls(t, api, "invalid-config")
	var baseline *coreapi.Service
	resource.Test(t, resource.TestCase{
		IsUnitTest: true, ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
		CheckDestroy: api.checkDestroy, ErrorCheck: api.errorCheck,
		Steps: []resource.TestStep{
			{
				PreConfig: func() { api.setPhase("invalid-config", true) },
				Config:    invalid, ExpectError: regexp.MustCompile(regexp.QuoteMeta(diagnostic)),
			},
			{
				PreConfig: func() { api.setPhase("create", false) },
				Config:    current, Check: api.check("kubernetes_service_v1.test", &baseline, false),
			},
			{
				PreConfig: func() { api.setPhase("follow-up", true) },
				Config:    current, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
				Check: api.check("kubernetes_service_v1.test", &baseline, true),
			},
			{PreConfig: func() { api.setPhase("cleanup", false) }, Config: current, Destroy: true},
		},
	})
}

func serviceCLICreateAndNoWritePlan(t *testing.T, api *serviceCLIAPI, config string) {
	t.Helper()
	var baseline *coreapi.Service
	resource.Test(t, resource.TestCase{
		IsUnitTest: true, ProtoV6ProviderFactories: serviceCLIProtoV6ProviderFactories,
		CheckDestroy: api.checkDestroy, ErrorCheck: api.errorCheck,
		Steps: []resource.TestStep{
			{Config: config, Check: api.check("kubernetes_service_v1.test", &baseline, false)},
			{
				PreConfig: func() { api.setPhase("follow-up", true) },
				Config:    config, ConfigPlanChecks: serviceCLISettled("kubernetes_service_v1.test", "", "no-op"),
				Check: api.check("kubernetes_service_v1.test", &baseline, true),
			},
			{PreConfig: func() { api.setPhase("cleanup", false) }, Config: config, Destroy: true},
		},
	})
}

// Loopback API fixture and its unit tests.

// A bounded Service wire fixture, not a Kubernetes conformance server. It
// decodes client-go's protobuf as well as JSON, allocates deterministic addresses
// and ports, and rejects invalid conditional fields instead of accepting every
// provider request. It never contacts a cluster.
type serviceCLIAPI struct {
	t        *testing.T
	server   *httptest.Server
	mu       sync.Mutex
	objects  map[string]*coreapi.Service
	calls    map[string]map[string]int
	patches  map[string][]json.RawMessage
	phase    string
	readOnly bool
	serial   int
	rv       int
	nextPort int32
}

func newServiceCLIAPI(t *testing.T) *serviceCLIAPI {
	t.Helper()
	api := &serviceCLIAPI{
		t: t, objects: map[string]*coreapi.Service{},
		calls: map[string]map[string]int{}, patches: map[string][]json.RawMessage{},
		phase: "create", nextPort: 31000,
	}
	api.server = httptest.NewServer(http.HandlerFunc(api.serveHTTP))
	t.Cleanup(func() {
		api.server.Close()
		api.mu.Lock()
		defer api.mu.Unlock()
		t.Logf("loopback Service request audit by phase: %v", api.calls)
		if len(api.objects) != 0 {
			t.Errorf("loopback Service objects leaked: %v", api.objects)
		}
	})
	return api
}

func (api *serviceCLIAPI) setPhase(phase string, readOnly bool) {
	api.mu.Lock()
	defer api.mu.Unlock()
	api.phase, api.readOnly = phase, readOnly
	if api.calls[phase] == nil {
		api.calls[phase] = map[string]int{}
	}
}

func (api *serviceCLIAPI) serveHTTP(w http.ResponseWriter, r *http.Request) {
	api.mu.Lock()
	defer api.mu.Unlock()
	if api.calls[api.phase] == nil {
		api.calls[api.phase] = map[string]int{}
	}
	api.calls[api.phase][r.Method+" "+r.URL.Path]++
	write := r.Method == http.MethodPost || r.Method == http.MethodPut ||
		r.Method == http.MethodPatch || r.Method == http.MethodDelete
	if write && api.readOnly {
		body, _ := io.ReadAll(r.Body)
		api.t.Errorf("forbidden Service API write during %s: %s %s body=%s", api.phase, r.Method, r.URL.Path, body)
		serviceCLIStatus(w, http.StatusForbidden, metav1.StatusReasonForbidden, "unchanged migration must not write")
		return
	}
	if api.phase == "cleanup" && write && r.Method != http.MethodDelete {
		api.t.Errorf("cleanup attempted non-DELETE write: %s %s", r.Method, r.URL.Path)
		serviceCLIStatus(w, http.StatusForbidden, metav1.StatusReasonForbidden, "cleanup may only delete")
		return
	}
	if r.Method == http.MethodGet && r.URL.Path == "/version" {
		serviceCLIRespond(w, http.StatusOK, map[string]string{
			"major": "1", "minor": "33", "gitVersion": "v1.33.6",
		})
		return
	}
	const prefix = "/api/v1/namespaces/default/services"
	if r.URL.Path != prefix && !strings.HasPrefix(r.URL.Path, prefix+"/") {
		api.t.Errorf("unexpected loopback endpoint: %s %s", r.Method, r.URL.Path)
		serviceCLIStatus(w, http.StatusNotFound, metav1.StatusReasonNotFound, "unsupported loopback endpoint")
		return
	}
	name := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, prefix), "/")
	if strings.Contains(name, "/") {
		serviceCLIStatus(w, http.StatusNotFound, metav1.StatusReasonNotFound, "unsupported Service subresource")
		return
	}
	old := api.objects[name]
	switch r.Method {
	case http.MethodGet:
		if old == nil {
			serviceCLIStatus(w, http.StatusNotFound, metav1.StatusReasonNotFound, "service not found")
			return
		}
		serviceCLIRespond(w, http.StatusOK, old)
	case http.MethodDelete:
		if old == nil {
			serviceCLIStatus(w, http.StatusNotFound, metav1.StatusReasonNotFound, "service not found")
			return
		}
		delete(api.objects, name)
		serviceCLIRespond(w, http.StatusOK, &metav1.Status{
			TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Status"}, Status: metav1.StatusSuccess,
		})
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		if r.Method != http.MethodPost && old == nil {
			serviceCLIStatus(w, http.StatusNotFound, metav1.StatusReasonNotFound, "service not found")
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			serviceCLIStatus(w, http.StatusBadRequest, metav1.StatusReasonBadRequest, err.Error())
			return
		}
		if r.Method == http.MethodPatch {
			api.patches[api.phase] = append(api.patches[api.phase], append(json.RawMessage(nil), body...))
			body, err = serviceCLIPatch(old, r.Header.Get("Content-Type"), body)
			if err != nil {
				serviceCLIStatus(w, http.StatusBadRequest, metav1.StatusReasonBadRequest, err.Error())
				return
			}
		}
		object := &coreapi.Service{}
		if _, _, err = clientgoscheme.Codecs.UniversalDeserializer().Decode(body, nil, object); err != nil {
			serviceCLIStatus(w, http.StatusBadRequest, metav1.StatusReasonBadRequest, err.Error())
			return
		}
		if r.Method == http.MethodPost {
			if name != "" || object.Name == "" {
				serviceCLIStatus(w, http.StatusBadRequest, metav1.StatusReasonBadRequest, "named Service requires collection POST")
				return
			}
			if api.objects[object.Name] != nil {
				serviceCLIStatus(w, http.StatusConflict, metav1.StatusReasonAlreadyExists, "service already exists")
				return
			}
			api.serial++
			object.UID = k8stypes.UID(fmt.Sprintf("service-cli-%06d", api.serial))
			object.CreationTimestamp = metav1.NewTime(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
		} else {
			if object.Name != name || object.Namespace != "default" {
				serviceCLIStatus(w, http.StatusUnprocessableEntity, metav1.StatusReasonInvalid, "Service identity is immutable")
				return
			}
			if object.UID != "" && object.UID != old.UID {
				serviceCLIStatus(w, http.StatusUnprocessableEntity, metav1.StatusReasonInvalid, "Service UID is immutable")
				return
			}
			if object.ResourceVersion != "" && object.ResourceVersion != old.ResourceVersion {
				serviceCLIStatus(w, http.StatusConflict, metav1.StatusReasonConflict, "resourceVersion does not match")
				return
			}
			object.UID, object.CreationTimestamp = old.UID, old.CreationTimestamp
			object.Status = old.Status
		}
		object.Namespace = "default"
		object.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "Service"}
		if err := api.defaultAndValidate(object, old); err != nil {
			serviceCLIStatus(w, http.StatusUnprocessableEntity, metav1.StatusReasonInvalid, err.Error())
			return
		}
		if r.Method == http.MethodPost {
			// Controller-owned state deliberately outside the resource's desired
			// fields must survive refresh, migration, and later provider updates.
			if object.Annotations == nil {
				object.Annotations = map[string]string{}
			}
			object.Annotations["controller.kubernetes.io/observed"] = "preserve"
			object.Spec.TrafficDistribution = ptr.To("PreferClose")
			if object.Spec.Type == coreapi.ServiceTypeLoadBalancer {
				object.Status.LoadBalancer.Ingress = []coreapi.LoadBalancerIngress{{
					IP: "192.0.2.50", Hostname: "loopback.example.test",
					IPMode: ptr.To(coreapi.LoadBalancerIPModeVIP),
				}}
			}
		}
		api.rv++
		object.ResourceVersion = strconv.Itoa(api.rv)
		api.objects[object.Name] = object.DeepCopy()
		code := http.StatusOK
		if r.Method == http.MethodPost {
			code = http.StatusCreated
		}
		serviceCLIRespond(w, code, object)
	default:
		serviceCLIStatus(w, http.StatusMethodNotAllowed, metav1.StatusReasonMethodNotAllowed, "unsupported verb")
	}
}

func serviceCLIPatch(old *coreapi.Service, contentType string, body []byte) ([]byte, error) {
	before, err := json.Marshal(old)
	if err != nil {
		return nil, err
	}
	switch contentType {
	case string(k8stypes.JSONPatchType):
		patch, err := jsonpatch.DecodePatch(body)
		if err != nil {
			return nil, err
		}
		return patch.Apply(before)
	case string(k8stypes.MergePatchType):
		return jsonpatch.MergePatch(before, body)
	case string(k8stypes.StrategicMergePatchType):
		return strategicpatch.StrategicMergePatch(before, body, coreapi.Service{})
	default:
		return nil, fmt.Errorf("unsupported Service patch content type %q", contentType)
	}
}

func (api *serviceCLIAPI) defaultAndValidate(object, old *coreapi.Service) error {
	spec := &object.Spec
	if spec.Type == "" {
		spec.Type = coreapi.ServiceTypeClusterIP
	}
	// Match Service storage's patchAllocatedValues: a PUT may resubmit the
	// original allocation-free manifest without changing assigned identities.
	if old != nil && old.Spec.Type != coreapi.ServiceTypeExternalName && spec.Type != coreapi.ServiceTypeExternalName {
		if spec.ClusterIP == "" {
			spec.ClusterIP = old.Spec.ClusterIP
		}
		if len(spec.ClusterIPs) == 0 {
			spec.ClusterIPs = append([]string(nil), old.Spec.ClusterIPs...)
		}
	}
	if old != nil && serviceCLINeedsNodePorts(&old.Spec) && serviceCLINeedsNodePorts(spec) {
		used := map[int32]bool{}
		for _, port := range spec.Ports {
			if port.NodePort != 0 {
				used[port.NodePort] = true
			}
		}
		for i := range spec.Ports {
			for _, previous := range old.Spec.Ports {
				if spec.Ports[i].Name == previous.Name && spec.Ports[i].NodePort == 0 && !used[previous.NodePort] {
					spec.Ports[i].NodePort = previous.NodePort
				}
			}
		}
	}
	if old != nil && old.Spec.Type == coreapi.ServiceTypeLoadBalancer && spec.Type == coreapi.ServiceTypeLoadBalancer &&
		old.Spec.ExternalTrafficPolicy == coreapi.ServiceExternalTrafficPolicyLocal &&
		spec.ExternalTrafficPolicy == coreapi.ServiceExternalTrafficPolicyLocal && spec.HealthCheckNodePort == 0 {
		spec.HealthCheckNodePort = old.Spec.HealthCheckNodePort
	}
	if spec.SessionAffinity == "" {
		spec.SessionAffinity = coreapi.ServiceAffinityNone
	}
	if spec.SessionAffinity == coreapi.ServiceAffinityClientIP {
		if spec.SessionAffinityConfig == nil {
			spec.SessionAffinityConfig = &coreapi.SessionAffinityConfig{}
		}
		if spec.SessionAffinityConfig.ClientIP == nil {
			spec.SessionAffinityConfig.ClientIP = &coreapi.ClientIPConfig{}
		}
		if spec.SessionAffinityConfig.ClientIP.TimeoutSeconds == nil {
			spec.SessionAffinityConfig.ClientIP.TimeoutSeconds = ptr.To(int32(10800))
		}
		timeout := *spec.SessionAffinityConfig.ClientIP.TimeoutSeconds
		if timeout <= 0 || timeout > 86400 {
			return fmt.Errorf("ClientIP affinity timeout must be 1..86400")
		}
	} else if spec.SessionAffinity == coreapi.ServiceAffinityNone {
		spec.SessionAffinityConfig = nil
	} else {
		return fmt.Errorf("invalid sessionAffinity %q", spec.SessionAffinity)
	}
	// Kubernetes' Service update strategy clears now-inapplicable allocations
	// when the submitted values are unchanged from the old type.
	if old != nil && old.Spec.Type != spec.Type {
		if spec.Type == coreapi.ServiceTypeExternalName {
			if spec.ClusterIP == old.Spec.ClusterIP {
				spec.ClusterIP = ""
			}
			if reflect.DeepEqual(spec.ClusterIPs, old.Spec.ClusterIPs) {
				spec.ClusterIPs = nil
			}
			if reflect.DeepEqual(spec.IPFamilies, old.Spec.IPFamilies) {
				spec.IPFamilies = nil
			}
			if reflect.DeepEqual(spec.IPFamilyPolicy, old.Spec.IPFamilyPolicy) {
				spec.IPFamilyPolicy = nil
			}
		}
		if spec.Type != coreapi.ServiceTypeLoadBalancer {
			if reflect.DeepEqual(spec.AllocateLoadBalancerNodePorts, old.Spec.AllocateLoadBalancerNodePorts) {
				spec.AllocateLoadBalancerNodePorts = nil
			}
			if reflect.DeepEqual(spec.LoadBalancerClass, old.Spec.LoadBalancerClass) {
				spec.LoadBalancerClass = nil
			}
		}
		if spec.Type == coreapi.ServiceTypeClusterIP || spec.Type == coreapi.ServiceTypeExternalName {
			for i := range spec.Ports {
				for _, previous := range old.Spec.Ports {
					if spec.Ports[i].NodePort == previous.NodePort {
						spec.Ports[i].NodePort = 0
					}
				}
			}
		}
		if spec.Type != coreapi.ServiceTypeLoadBalancer && spec.HealthCheckNodePort == old.Spec.HealthCheckNodePort {
			spec.HealthCheckNodePort = 0
		}
		if !serviceCLIExternallyAccessible(spec) && spec.ExternalTrafficPolicy == old.Spec.ExternalTrafficPolicy {
			spec.ExternalTrafficPolicy = ""
		}
	}
	if old != nil && old.Spec.Type == coreapi.ServiceTypeLoadBalancer &&
		old.Spec.ExternalTrafficPolicy == coreapi.ServiceExternalTrafficPolicyLocal &&
		(spec.Type != coreapi.ServiceTypeLoadBalancer || spec.ExternalTrafficPolicy != coreapi.ServiceExternalTrafficPolicyLocal) &&
		spec.HealthCheckNodePort == old.Spec.HealthCheckNodePort {
		spec.HealthCheckNodePort = 0
	}
	if spec.Type != coreapi.ServiceTypeLoadBalancer {
		object.Status.LoadBalancer = coreapi.LoadBalancerStatus{}
	}
	switch spec.Type {
	case coreapi.ServiceTypeExternalName:
		if spec.ExternalName == "" {
			return fmt.Errorf("ExternalName requires externalName")
		}
		if spec.ClusterIP != "" || len(spec.ClusterIPs) != 0 || len(spec.IPFamilies) != 0 || spec.IPFamilyPolicy != nil {
			return fmt.Errorf("ExternalName cannot carry cluster IP allocations")
		}
	case coreapi.ServiceTypeClusterIP, coreapi.ServiceTypeNodePort, coreapi.ServiceTypeLoadBalancer:
		if len(spec.Ports) == 0 && spec.ClusterIP != coreapi.ClusterIPNone {
			return fmt.Errorf("non-ExternalName Service requires ports")
		}
		if spec.ClusterIP == coreapi.ClusterIPNone && spec.Type != coreapi.ServiceTypeClusterIP {
			return fmt.Errorf("headless Services require ClusterIP type")
		}
		if spec.IPFamilyPolicy == nil {
			spec.IPFamilyPolicy = ptr.To(coreapi.IPFamilyPolicySingleStack)
		}
		if len(spec.IPFamilies) == 0 {
			spec.IPFamilies = []coreapi.IPFamily{coreapi.IPv4Protocol}
			if *spec.IPFamilyPolicy != coreapi.IPFamilyPolicySingleStack {
				spec.IPFamilies = append(spec.IPFamilies, coreapi.IPv6Protocol)
			}
		}
		if len(spec.ClusterIPs) == 0 && spec.ClusterIP != "" {
			spec.ClusterIPs = []string{spec.ClusterIP}
		}
		if len(spec.ClusterIPs) == 0 {
			spec.ClusterIPs = []string{fmt.Sprintf("10.96.0.%d", api.serial+10)}
		}
		if len(spec.IPFamilies) == 2 && len(spec.ClusterIPs) == 1 && spec.ClusterIPs[0] != coreapi.ClusterIPNone {
			spec.ClusterIPs = append(spec.ClusterIPs, fmt.Sprintf("fd00::%x", api.serial+10))
		}
		if spec.ClusterIP == "" {
			spec.ClusterIP = spec.ClusterIPs[0]
		}
		if spec.ClusterIP != spec.ClusterIPs[0] {
			return fmt.Errorf("clusterIP must equal clusterIPs[0]")
		}
		if spec.InternalTrafficPolicy == nil {
			spec.InternalTrafficPolicy = ptr.To(coreapi.ServiceInternalTrafficPolicyCluster)
		}
	default:
		return fmt.Errorf("unsupported Service type %q", spec.Type)
	}
	if old != nil && old.Spec.Type != coreapi.ServiceTypeExternalName && spec.Type != coreapi.ServiceTypeExternalName &&
		spec.ClusterIP != old.Spec.ClusterIP {
		return fmt.Errorf("spec.clusterIP is immutable")
	}
	if spec.Type == coreapi.ServiceTypeLoadBalancer {
		if spec.AllocateLoadBalancerNodePorts == nil {
			spec.AllocateLoadBalancerNodePorts = ptr.To(true)
		}
	} else if spec.AllocateLoadBalancerNodePorts != nil || spec.LoadBalancerClass != nil {
		return fmt.Errorf("allocateLoadBalancerNodePorts and loadBalancerClass require LoadBalancer")
	}
	if old != nil && old.Spec.Type == coreapi.ServiceTypeLoadBalancer && spec.Type == coreapi.ServiceTypeLoadBalancer &&
		!reflect.DeepEqual(spec.LoadBalancerClass, old.Spec.LoadBalancerClass) {
		return fmt.Errorf("loadBalancerClass is immutable")
	}
	external := serviceCLIExternallyAccessible(spec)
	if external && spec.ExternalTrafficPolicy == "" {
		spec.ExternalTrafficPolicy = coreapi.ServiceExternalTrafficPolicyCluster
	}
	if !external && spec.ExternalTrafficPolicy != "" {
		return fmt.Errorf("externalTrafficPolicy requires externally accessible Service")
	}
	needsHealthCheck := spec.Type == coreapi.ServiceTypeLoadBalancer && spec.ExternalTrafficPolicy == coreapi.ServiceExternalTrafficPolicyLocal
	if needsHealthCheck && spec.HealthCheckNodePort == 0 {
		spec.HealthCheckNodePort = api.allocatePort()
	}
	if !needsHealthCheck && spec.HealthCheckNodePort != 0 {
		return fmt.Errorf("healthCheckNodePort requires LoadBalancer with Local external traffic")
	}
	if spec.HealthCheckNodePort != 0 && (spec.HealthCheckNodePort < 30000 || spec.HealthCheckNodePort > 32767) {
		return fmt.Errorf("healthCheckNodePort is outside allocation range")
	}
	seenNames, seenNodePorts := map[string]bool{}, map[int32]bool{}
	for i := range spec.Ports {
		port := &spec.Ports[i]
		if port.Port <= 0 || port.Port > 65535 || (len(spec.Ports) > 1 && port.Name == "") || (port.Name != "" && seenNames[port.Name]) {
			return fmt.Errorf("invalid or duplicate Service port")
		}
		seenNames[port.Name] = true
		if port.Protocol == "" {
			port.Protocol = coreapi.ProtocolTCP
		}
		if port.TargetPort == intstr.FromInt32(0) || port.TargetPort == intstr.FromString("") {
			port.TargetPort = intstr.FromInt32(port.Port)
		}
		if spec.Type == coreapi.ServiceTypeNodePort ||
			(spec.Type == coreapi.ServiceTypeLoadBalancer && *spec.AllocateLoadBalancerNodePorts) {
			if port.NodePort == 0 {
				port.NodePort = api.allocatePort()
			}
		}
		if port.NodePort != 0 {
			if spec.Type != coreapi.ServiceTypeNodePort && spec.Type != coreapi.ServiceTypeLoadBalancer {
				return fmt.Errorf("nodePort requires NodePort or LoadBalancer")
			}
			if port.NodePort < 30000 || port.NodePort > 32767 || seenNodePorts[port.NodePort] || port.NodePort == spec.HealthCheckNodePort {
				return fmt.Errorf("nodePort is outside range or already allocated")
			}
			seenNodePorts[port.NodePort] = true
		}
	}
	for name, allocated := range api.objects {
		if name == object.Name {
			continue
		}
		for _, previous := range allocated.Spec.Ports {
			if previous.NodePort != 0 && (seenNodePorts[previous.NodePort] || spec.HealthCheckNodePort == previous.NodePort) {
				return fmt.Errorf("nodePort is already allocated to another Service")
			}
		}
		if allocated.Spec.HealthCheckNodePort != 0 &&
			(seenNodePorts[allocated.Spec.HealthCheckNodePort] || spec.HealthCheckNodePort == allocated.Spec.HealthCheckNodePort) {
			return fmt.Errorf("healthCheckNodePort is already allocated to another Service")
		}
	}
	return nil
}

func (api *serviceCLIAPI) allocatePort() int32 {
	api.nextPort++
	return api.nextPort
}

func serviceCLINeedsNodePorts(spec *coreapi.ServiceSpec) bool {
	return spec.Type == coreapi.ServiceTypeNodePort || (spec.Type == coreapi.ServiceTypeLoadBalancer &&
		(spec.AllocateLoadBalancerNodePorts == nil || *spec.AllocateLoadBalancerNodePorts))
}

func serviceCLIExternallyAccessible(spec *coreapi.ServiceSpec) bool {
	return spec.Type == coreapi.ServiceTypeNodePort || spec.Type == coreapi.ServiceTypeLoadBalancer ||
		(spec.Type == coreapi.ServiceTypeClusterIP && len(spec.ExternalIPs) > 0)
}

func serviceCLIRespond(w http.ResponseWriter, code int, object any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(object)
}

func serviceCLIStatus(w http.ResponseWriter, code int, reason metav1.StatusReason, message string) {
	serviceCLIRespond(w, code, &metav1.Status{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Status"},
		Status:   metav1.StatusFailure, Code: int32(code), Reason: reason, Message: message,
	})
}

func TestServiceCLIAPI_ServiceConstraints(t *testing.T) {
	for _, test := range []struct {
		name string
		spec coreapi.ServiceSpec
		fail bool
	}{
		{name: "cluster-defaults", spec: coreapi.ServiceSpec{Ports: []coreapi.ServicePort{{Port: 80}}}},
		{name: "headless-without-ports", spec: coreapi.ServiceSpec{ClusterIP: "None"}},
		{name: "external-name", spec: coreapi.ServiceSpec{Type: coreapi.ServiceTypeExternalName, ExternalName: "example.test"}},
		{name: "allocation-on-cluster-ip", fail: true, spec: coreapi.ServiceSpec{
			Ports: []coreapi.ServicePort{{Port: 80}}, AllocateLoadBalancerNodePorts: ptr.To(false),
		}},
		{name: "nodeport-on-cluster-ip", fail: true, spec: coreapi.ServiceSpec{
			Ports: []coreapi.ServicePort{{Port: 80, NodePort: 30080}},
		}},
		{name: "external-name-with-cluster-ip", fail: true, spec: coreapi.ServiceSpec{
			Type: coreapi.ServiceTypeExternalName, ExternalName: "example.test", ClusterIP: "10.96.0.12",
		}},
		{name: "headless-nodeport", fail: true, spec: coreapi.ServiceSpec{
			Type: coreapi.ServiceTypeNodePort, ClusterIP: "None", Ports: []coreapi.ServicePort{{Port: 80}},
		}},
		{name: "invalid-healthcheck", fail: true, spec: coreapi.ServiceSpec{
			Type: coreapi.ServiceTypeLoadBalancer, HealthCheckNodePort: 30123, Ports: []coreapi.ServicePort{{Port: 80}},
		}},
		{name: "allocation-false-with-explicit-nodeport", spec: coreapi.ServiceSpec{
			Type: coreapi.ServiceTypeLoadBalancer, AllocateLoadBalancerNodePorts: ptr.To(false),
			Ports: []coreapi.ServicePort{{Port: 80, NodePort: 30080}},
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			api := &serviceCLIAPI{serial: 1, nextPort: 31000, objects: map[string]*coreapi.Service{}}
			object := &coreapi.Service{Spec: *test.spec.DeepCopy()}
			err := api.defaultAndValidate(object, nil)
			if (err != nil) != test.fail {
				t.Fatalf("validation error=%v, wanted failure=%t", err, test.fail)
			}
			if err != nil {
				return
			}
			if test.name == "cluster-defaults" && (object.Spec.ClusterIP == "" ||
				object.Spec.AllocateLoadBalancerNodePorts != nil || object.Spec.Ports[0].TargetPort.IntVal != 80) {
				t.Fatalf("incorrect ClusterIP defaults: %+v", object.Spec)
			}
			if test.name == "allocation-false-with-explicit-nodeport" && object.Spec.Ports[0].NodePort != 30080 {
				t.Fatal("allocation=false must preserve an explicitly requested node port")
			}
		})
	}
}

func TestServiceCLIAPI_TypeTransitionAndImmutableClusterIP(t *testing.T) {
	api := &serviceCLIAPI{serial: 1, nextPort: 31000, objects: map[string]*coreapi.Service{}}
	old := &coreapi.Service{Spec: coreapi.ServiceSpec{
		Type: coreapi.ServiceTypeLoadBalancer, ExternalTrafficPolicy: coreapi.ServiceExternalTrafficPolicyLocal,
		Ports: []coreapi.ServicePort{{Port: 80}},
	}}
	if err := api.defaultAndValidate(old, nil); err != nil {
		t.Fatal(err)
	}

	bad := old.DeepCopy()
	bad.Spec.ClusterIP, bad.Spec.ClusterIPs = "10.96.0.99", []string{"10.96.0.99"}
	if err := api.defaultAndValidate(bad, old); err == nil {
		t.Fatal("fixture accepted an immutable primary ClusterIP update")
	}
	current := old.DeepCopy()
	current.Spec.Type = coreapi.ServiceTypeExternalName
	current.Spec.ExternalName = "example.test"
	if err := api.defaultAndValidate(current, old); err != nil {
		t.Fatal(err)
	}
	if current.Spec.ClusterIP != "" || current.Spec.ClusterIPs != nil ||
		current.Spec.AllocateLoadBalancerNodePorts != nil || current.Spec.Ports[0].NodePort != 0 ||
		current.Spec.HealthCheckNodePort != 0 {
		t.Fatalf("type transition retained obsolete allocations: %+v", current.Spec)
	}
}

func TestServiceCLIAPI_ExternalIPsTypeTransition(t *testing.T) {
	for _, serviceType := range []coreapi.ServiceType{coreapi.ServiceTypeNodePort, coreapi.ServiceTypeLoadBalancer} {
		for _, retainExternalIPs := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/retain-external-ips-%t", serviceType, retainExternalIPs), func(t *testing.T) {
				api := &serviceCLIAPI{serial: 1, nextPort: 31000, objects: map[string]*coreapi.Service{}}
				old := &coreapi.Service{Spec: coreapi.ServiceSpec{
					Type: serviceType, ExternalTrafficPolicy: coreapi.ServiceExternalTrafficPolicyLocal,
					ExternalIPs: []string{"192.0.2.11"}, Ports: []coreapi.ServicePort{{Port: 80}},
				}}
				if err := api.defaultAndValidate(old, nil); err != nil {
					t.Fatal(err)
				}
				current := old.DeepCopy()
				current.Spec.Type = coreapi.ServiceTypeClusterIP
				expectedPolicy := coreapi.ServiceExternalTrafficPolicyLocal
				if !retainExternalIPs {
					current.Spec.ExternalIPs = nil
					expectedPolicy = ""
				}
				if err := api.defaultAndValidate(current, old); err != nil {
					t.Fatal(err)
				}
				if current.Spec.ExternalTrafficPolicy != expectedPolicy ||
					current.Spec.ClusterIP != old.Spec.ClusterIP ||
					current.Spec.Ports[0].TargetPort != old.Spec.Ports[0].TargetPort ||
					current.Spec.Ports[0].NodePort != 0 || current.Spec.HealthCheckNodePort != 0 ||
					current.Spec.AllocateLoadBalancerNodePorts != nil {
					t.Fatalf("incorrect ClusterIP transition: %+v", current.Spec)
				}
			})
		}
	}
}
