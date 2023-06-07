// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccKubernetesEndpointSlice_basic(t *testing.T) {
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
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_endpoint_slice_v1.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoint_slice_v1.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoint_slice_v1.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "endpoint.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "endpoint.0.condition.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "endpoint.0.condition.0.ready", "true"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "endpoint.0.hostname", "test"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "endpoint.0.node_name", "test"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "endpoint.0.addresses.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "endpoint.0.target_ref.0.name", "test"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "endpoint.0.addresses.0", "2001:db8:3333:4444:5555:6666:7777:8888"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "endpoint.0.addresses.1", "2002:db8:3333:4444:5555:6666:7777:8888"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "port.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "port.0.port", "90"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "port.0.name", "first"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "port.0.app_protocol", "test"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "port.1.port", "900"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "port.1.name", "second"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "port.1.app_protocol", "test"),
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "address_type", "IPv6"),
				),
			},
		},
	})
}

func TestAccKubernetesEndpointSlice_generatedName(t *testing.T) {
	prefix := "tf-acc-test-gen-"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_endpoint_slice_v1.test",
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesEndpointSliceConfig_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
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
					resource.TestCheckResourceAttr("kubernetes_endpoint_slice_v1.test", "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr("kubernetes_endpoint_slice_v1.test", "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet("kubernetes_endpoint_slice_v1.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoint_slice_v1.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoint_slice_v1.test", "metadata.0.uid"),
				),
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
      addresses = ["129.144.50.56"]
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
			target_ref{
				name = "test"
			}
			addresses = ["2001:db8:3333:4444:5555:6666:7777:8888", "2002:db8:3333:4444:5555:6666:7777:8888"]
			hostname = "test"
			node_name = "test"
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
	  
		address_type = "IPv6"
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
