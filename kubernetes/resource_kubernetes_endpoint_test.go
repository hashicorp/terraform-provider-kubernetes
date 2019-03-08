package kubernetes

import (
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	api "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func TestAccKubernetesEndpoint_basic(t *testing.T) {
	var conf api.Endpoints
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_endpoint.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesEndpointDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesEndpointConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesEndpointExists("kubernetes_endpoint.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_endpoint.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoint.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoint.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoint.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.0.addresses.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.0.addresses.0.ip", "10.0.0.4"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.0.ports.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.0.ports.0.name", "httptransport"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.0.ports.0.port", "80"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.0.ports.0.protocol", "TCP"),
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
				Config: testAccKubernetesEndpointConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesEndpointExists("kubernetes_endpoint.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_endpoint.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoint.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoint.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoint.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.0.addresses.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.0.addresses.0.ip", "10.0.0.5"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.0.ports.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.0.ports.0.name", "httptransport"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.0.ports.0.port", "82"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.0.ports.0.protocol", "TCP"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.1.addresses.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.1.addresses.0.ip", "10.0.0.6"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.1.addresses.0.hostname", "test-hostname"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.1.addresses.0.node_name", "test-nodename"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.1.addresses.1.ip", "10.0.0.7"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.1.ports.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.1.ports.0.name", "httpstransport"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.1.ports.0.port", "443"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.1.ports.0.protocol", "TCP"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.1.ports.1.name", "httpstransport2"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.1.ports.1.port", "444"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.1.ports.1.protocol", "TCP"),
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
							Ports: []api.EndpointPort{
								{
									Name:     "httpstransport",
									Port:     443,
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
				Config: testAccKubernetesEndpointConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesEndpointExists("kubernetes_endpoint.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_endpoint.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoint.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoint.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoint.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.0.addresses.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.0.addresses.0.ip", "10.0.0.4"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.0.ports.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.0.ports.0.name", "httptransport"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.0.ports.0.port", "80"),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "subsets.0.ports.0.protocol", "TCP"),
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

func TestAccKubernetesEndpoint_importBasic(t *testing.T) {
	resourceName := "kubernetes_endpoint.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesEndpointDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesEndpointConfig_basic(name),
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

func TestAccKubernetesEndpoint_generatedName(t *testing.T) {
	var conf api.Endpoints
	prefix := "tf-acc-test-gen-"

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_endpoint.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesEndpointDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesEndpointConfig_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesEndpointExists("kubernetes_endpoint.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_endpoint.test", "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr("kubernetes_endpoint.test", "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet("kubernetes_endpoint.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoint.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoint.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_endpoint.test", "metadata.0.uid"),
				),
			},
		},
	})
}

func TestAccKubernetesEndpoint_importGeneratedName(t *testing.T) {
	resourceName := "kubernetes_endpoint.test"
	prefix := "tf-acc-test-gen-import-"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesEndpointDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesEndpointConfig_generatedName(prefix),
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

		if !reflect.DeepEqual(subsets, expected) {
			return fmt.Errorf("Endpoint subsets don't match.\nExpected: %#v\nGiven: %#v",
				expected, svc.Subsets)
		}

		return nil
	}
}

func testAccCheckKubernetesEndpointDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*kubernetes.Clientset)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_endpoint" {
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

		conn := testAccProvider.Meta().(*kubernetes.Clientset)

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

func testAccKubernetesEndpointConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_endpoint" "test" {
  metadata {
    annotations {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }

    labels {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    name = "%s"
  }

  subsets {
    addresses {
      ip = "10.0.0.4"
    }

    ports {
      name     = "httptransport"
      port     = 80
      protocol = "TCP"
    }
  }
}
`, name)
}

func testAccKubernetesEndpointConfig_modified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_endpoint" "test" {
  metadata {
    annotations {
      TestAnnotationOne = "one"
      Different         = "1234"
    }

    labels {
      TestLabelOne   = "one"
      TestLabelThree = "three"
    }

    name = "%s"
  }

  subsets {
    addresses {
      ip = "10.0.0.5"
    }

    ports {
      name     = "httptransport"
      port     = 82
      protocol = "TCP"
    }
  }

  subsets {
    addresses {
      ip        = "10.0.0.6"
      hostname  = "test-hostname"
      node_name = "test-nodename"
    }

    addresses {
      ip = "10.0.0.7"
    }

    ports {
      name     = "httpstransport"
      port     = 443
      protocol = "TCP"
    }

    ports {
      name     = "httpstransport2"
      port     = 444
      protocol = "TCP"
    }
  }
}
`, name)
}

func testAccKubernetesEndpointConfig_generatedName(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_endpoint" "test" {
  metadata {
    generate_name = "%s"
  }

  subsets {
    addresses {
      ip = "10.0.0.4"
    }

    ports {
      name     = "transport"
      port     = 80
      protocol = "TCP"
    }
  }
}
`, prefix)
}
