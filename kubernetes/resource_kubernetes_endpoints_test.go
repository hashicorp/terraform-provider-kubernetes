package kubernetes

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	api "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesEndpoints_basic(t *testing.T) {
	var conf api.Endpoints
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_endpoints.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesEndpointDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesEndpointsConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesEndpointExists("kubernetes_endpoints.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_endpoints.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoints.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoints.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoints.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.1675725831.address.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.1675725831.address.4043229419.ip", "10.0.0.4"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.1675725831.port.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.1675725831.port.3819788156.name", "httptransport"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.1675725831.port.3819788156.port", "80"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.1675725831.port.3819788156.protocol", "TCP"),
					testAccCheckEndpointSubsets(&conf, []api.EndpointSubset{
						{
							Addresses: []api.EndpointAddress{
								{
									IP: "10.0.0.4",
								},
							},
							Ports: []api.EndpointPort{
								{
									Name:     "httptransport",
									Port:     80,
									Protocol: api.ProtocolTCP,
								},
							},
						},
					}),
				),
			},
			{
				Config: testAccKubernetesEndpointsConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesEndpointExists("kubernetes_endpoints.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_endpoints.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoints.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoints.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoints.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.1936320382.address.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.1936320382.address.761794926.ip", "10.0.0.5"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.1936320382.port.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.1936320382.port.1261168365.name", "httptransport"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.1936320382.port.1261168365.port", "82"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.1936320382.port.1261168365.protocol", "TCP"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.271063890.address.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.271063890.address.3580306450.ip", "10.0.0.6"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.271063890.address.3580306450.hostname", "test-hostname"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.271063890.address.3580306450.node_name", "test-nodename"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.271063890.address.1295295525.ip", "10.0.0.7"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.271063890.not_ready_address.678871443.ip", "10.0.0.10"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.271063890.not_ready_address.4125104150.ip", "10.0.0.11"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.271063890.port.#", "3"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.271063890.port.3846540438.name", "httpstransport"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.271063890.port.3846540438.port", "443"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.271063890.port.3846540438.protocol", "TCP"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.271063890.port.2795404772.name", "httpstransport2"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.271063890.port.2795404772.port", "444"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.271063890.port.2795404772.protocol", "TCP"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.271063890.port.2264813282.name", "aaaa"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.271063890.port.2264813282.port", "442"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.271063890.port.2264813282.protocol", "TCP"),
					testAccCheckEndpointSubsets(&conf, []api.EndpointSubset{
						{
							Addresses: []api.EndpointAddress{
								{
									IP: "10.0.0.5",
								},
							},
							Ports: []api.EndpointPort{
								{
									Name:     "httptransport",
									Port:     82,
									Protocol: api.ProtocolTCP,
								},
							},
						},
						{
							Addresses: []api.EndpointAddress{
								{
									IP:       "10.0.0.6",
									Hostname: "test-hostname",
									NodeName: ptrToString("test-nodename"),
								},
								{
									IP: "10.0.0.7",
								},
							},
							NotReadyAddresses: []api.EndpointAddress{
								{
									IP: "10.0.0.10",
								},
								{
									IP: "10.0.0.11",
								},
							},
							Ports: []api.EndpointPort{
								{
									Name:     "httpstransport",
									Port:     443,
									Protocol: api.ProtocolTCP,
								},
								{
									Name:     "aaaa",
									Port:     442,
									Protocol: api.ProtocolTCP,
								},
								{
									Name:     "httpstransport2",
									Port:     444,
									Protocol: api.ProtocolTCP,
								},
							},
						},
					}),
				),
			},
			{
				Config: testAccKubernetesEndpointsConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesEndpointExists("kubernetes_endpoints.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_endpoints.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoints.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoints.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoints.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.1675725831.address.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.1675725831.address.4043229419.ip", "10.0.0.4"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.1675725831.port.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.1675725831.port.3819788156.name", "httptransport"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.1675725831.port.3819788156.port", "80"),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "subset.1675725831.port.3819788156.protocol", "TCP"),
					testAccCheckEndpointSubsets(&conf, []api.EndpointSubset{
						{
							Addresses: []api.EndpointAddress{
								{
									IP: "10.0.0.4",
								},
							},
							Ports: []api.EndpointPort{
								{
									Name:     "httptransport",
									Port:     80,
									Protocol: api.ProtocolTCP,
								},
							},
						},
					}),
				),
			},
		},
	})
}

func TestAccKubernetesEndpoints_importBasic(t *testing.T) {
	resourceName := "kubernetes_endpoints.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesEndpointDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesEndpointsConfig_basic(name),
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

func TestAccKubernetesEndpoints_generatedName(t *testing.T) {
	var conf api.Endpoints
	prefix := "tf-acc-test-gen-"

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_endpoints.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesEndpointDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesEndpointsConfig_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesEndpointExists("kubernetes_endpoints.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_endpoints.test", "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr("kubernetes_endpoints.test", "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet("kubernetes_endpoints.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoints.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoints.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoints.test", "metadata.0.uid"),
				),
			},
		},
	})
}

func TestAccKubernetesEndpoints_importGeneratedName(t *testing.T) {
	resourceName := "kubernetes_endpoints.test"
	prefix := "tf-acc-test-gen-import-"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesEndpointDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesEndpointsConfig_generatedName(prefix),
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

func testAccCheckEndpointSubsets(svc *api.Endpoints, expected []api.EndpointSubset) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(expected) == 0 && len(svc.Subsets) == 0 {
			return nil
		}

		subsets := svc.Subsets
		if diff := cmp.Diff(expected, subsets); diff != "" {
			return fmt.Errorf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}

		return nil
	}
}

func testAccCheckKubernetesEndpointDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*KubeClientsets).MainClientset

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_endpoints" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.CoreV1().Endpoints(namespace).Get(name, meta_v1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Endpoint still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesEndpointExists(n string, obj *api.Endpoints) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*KubeClientsets).MainClientset

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		out, err := conn.CoreV1().Endpoints(namespace).Get(name, meta_v1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesEndpointsConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_endpoints" "test" {
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

func testAccKubernetesEndpointsConfig_modified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_endpoints" "test" {
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

func testAccKubernetesEndpointsConfig_generatedName(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_endpoints" "test" {
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
