// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	api "k8s.io/api/core/v1"
)

func TestAccKubernetesEndpointSlice_basic(t *testing.T) {
	//var conf api.Endpoints
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_endpoint_slice_v1.test",
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesEndpointSliceConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					//testAccCheckKubernetesEndpointExists("kubernetes_endpoint_slice_v1.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_endpoint_slice_v1.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoint_slice_v1.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoint_slice_v1.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "endpoint.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "endpoint.0.condition.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "endpoint.0.addresses.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "endpoint.0.addresses.0", "129.144.50.56"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "port.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "port.0.port", "90"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "port.0.name", "first"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "port.0.app_protocol", "test"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "address_type", "IPv4"),
				),
			},
			{
				Config: testAccKubernetesEndpointSliceConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					//testAccCheckKubernetesEndpointExists("kubernetes_endpoint_slice_v1.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_endpoint_slice_v1.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoint_slice_v1.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoint_slice_v1.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "endpoint.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "endpoint.0.condition.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "endpoint.0.condition.ready", "true"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "endpoint.0.condition.hostname", "test"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "endpoint.0.condition.node_name", "test"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "endpoint.0.condition.target_ref.0.name", "test"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "endpoint.0.addresses.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "endpoint.0.addresses.0", "129.144.50.56"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "endpoint.0.addresses.1", "129.144.50.60"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "port.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "port.0.port", "90"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "port.0.name", "first"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "port.0.app_protocol", "test"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "port.1.port", "900"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "port.1.name", "second"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "port.1.app_protocol", "test"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "address_type", "IPv4"),
				),
			},
			// {
			// 	Config: testAccKubernetesEndpointSliceConfig_basic(name),
			// 	Check: resource.ComposeAggregateTestCheckFunc(
			// 		testAccCheckKubernetesEndpointExists("kubernetes_endpoint_slice_v1.test", &conf),
			// 		resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "metadata.0.name", name),
			// 		resource.TestCheckResourceAttrSet("kubernetes_endpoint_slice_v1.test", "metadata.0.generation"),
			// 		resource.TestCheckResourceAttrSet("kubernetes_endpoint_slice_v1.test", "metadata.0.resource_version"),
			// 		resource.TestCheckResourceAttrSet("kubernetes_endpoint_slice_v1.test", "metadata.0.uid"),
			// 		resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "subset.#", "1"),
			// 		resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "subset.0.address.#", "1"),
			// 		resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "subset.0.address.0.ip", "10.0.0.4"),
			// 		resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "subset.0.port.0.name", "httptransport"),
			// 		resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "subset.0.port.0.port", "80"),
			// 		resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "subset.0.port.0.protocol", "TCP"),
			// 	),
			// },
		},
	})
}

func TestAccKubernetesEndpointSlice_generatedName(t *testing.T) {
	var conf api.Endpoints
	prefix := "tf-acc-test-gen-"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_endpoint_slice_v1.test",
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesEndpointsConfig_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesEndpointExists("kubernetes_endpoint_slice_v1.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr("kubernetes_endpoint_slice_v1.test", "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet("kubernetes_endpoint_slice_v1.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoint_slice_v1.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoint_slice_v1.test", "metadata.0.uid"),
				),
			},
			{
				ResourceName:            "kubernetes_endpoint_slice_v1.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
		},
	})
}

func testAccKubernetesEndpointSliceConfig_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_endpoint_slice_v1" "test" {
  metadata {
    name = "%s"
  }

    endpoint {
      condition {
        
      }
      addresses = ["129.144.50.56"  ]
      
    }

    port {
      port = "90"
      name = "first"
      app_protocol = "test"
    }

  address_type = "IPv4"
}
`, name)
}

func testAccKubernetesEndpointSliceConfig_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_endpoint_slice_v1" "test" {
		metadata {
		  name = "%s"
		}
	  
		  endpoint {
			condition {
			  ready = true
			}
			addresses = ["129.144.50.56", "129.144.50.60"]
			hostname = "test"
			node_name = "test"
			target_ref{
				name = "test"
			}
			zone = "us-west"
		  }
	  
		  port {
			port = "90"
			name = "first"
			app_protocol = "test"
		  }

		  port {
			port = "900"
			name = "second"
			app_protocol = "test"
		  }
	  
		address_type = "IPv4"
	  }
`, name)
}

func testAccKubernetesEndpointSliceConfig_generatedName(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_endpoint_slice_v1" "test" {
		metadata {
		  generate_name = "%s"
		}
	  
		  endpoint {
			condition {
			  
			}
			addresses = ["129.144.50.56"]
			
		  }
	  
		  port {
			port = "90"
			name = "first"
			app_protocol = "test"
		  }
	  
		address_type = "IPv4"
	  }
`, prefix)
}
