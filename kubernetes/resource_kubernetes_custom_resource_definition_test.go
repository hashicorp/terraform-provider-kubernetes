package kubernetes

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	api "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesCustomResourceDefinition_minimal(t *testing.T) {
	var conf api.CustomResourceDefinition
	name := fmt.Sprintf("tfacctest%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlpha))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_custom_resource_definition.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesCustomResourceDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesCustomResourceDefinitionConfig_minimal(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesCustomResourceDefinitionExists("kubernetes_custom_resource_definition.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "metadata.0.name", name+"s.crdtest.example.com"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.scope", "Cluster"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.group", "crdtest.example.com"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.version", "v1"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.names.0.kind", strings.Title(name)),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.names.0.plural", name+"s"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.names.0.singular", name),
				),
			},
		},
	})
}

func TestAccKubernetesCustomResourceDefinition_basic(t *testing.T) {
	var conf api.CustomResourceDefinition
	name := fmt.Sprintf("tfacctest%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlpha))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_custom_resource_definition.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesCustomResourceDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesCustomResourceDefinitionConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesCustomResourceDefinitionExists("kubernetes_custom_resource_definition.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "TestAnnotationTwo": "two"}),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelTwo": "two", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "metadata.0.name", name+"s.crdtest.example.com"),
					resource.TestCheckResourceAttrSet("kubernetes_custom_resource_definition.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_custom_resource_definition.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_custom_resource_definition.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_custom_resource_definition.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.scope", "Cluster"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.group", "crdtest.example.com"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.version", "v1"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.names.0.kind", strings.Title(name)),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.names.0.plural", name+"s"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.names.0.singular", name),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.names.0.short_names.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.names.0.short_names.0", name[0:15]),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.names.0.list_kind", name+"list"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.names.0.categories.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.names.0.categories.0", "crdtest"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.names.0.categories.1", "test"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.subresources.0.scale.0.spec_replicas_path", ".replicas"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.subresources.0.scale.0.status_replicas_path", ".status"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.subresources.0.scale.0.label_selector_path", ".labels"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.additional_printer_column.#", "1"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.additional_printer_column.0.name", "Spec"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.additional_printer_column.0.type", "string"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.additional_printer_column.0.description", "The spec of the basiccrd"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.additional_printer_column.0.json_path", ".spec"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.additional_printer_column.0.priority", "5"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.additional_printer_column.0.format", "password"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.conversion.0.strategy", "None"),
				),
			},
			{
				Config: testAccKubernetesCustomResourceDefinitionConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesCustomResourceDefinitionExists("kubernetes_custom_resource_definition.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "metadata.0.annotations.Different", "1234"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "Different": "1234"}),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "metadata.0.name", name+"s.modcrdtest.example.com"),
					resource.TestCheckResourceAttrSet("kubernetes_custom_resource_definition.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_custom_resource_definition.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_custom_resource_definition.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_custom_resource_definition.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.scope", "Namespaced"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.group", "modcrdtest.example.com"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.version", "v2"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.names.0.kind", strings.Title(name)),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.names.0.plural", name+"s"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.names.0.singular", name),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.names.0.short_names.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.names.0.short_names.0", name[0:15]),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.names.0.list_kind", name+"list"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.names.0.categories.#", "3"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.names.0.categories.0", "crdtest"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.names.0.categories.1", "test"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.names.0.categories.2", "mod"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.0.name", "v2"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.0.served", "true"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.0.storage", "true"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.0.subresources.0.scale.0.spec_replicas_path", ".v2.replicas"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.0.subresources.0.scale.0.status_replicas_path", ".v2.status"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.0.subresources.0.scale.0.label_selector_path", ".v2.labels"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.0.additional_printer_column.#", "2"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.0.additional_printer_column.0.name", "Creation Date"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.0.additional_printer_column.0.type", "string"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.0.additional_printer_column.0.description", "The creation date of the basiccrd"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.0.additional_printer_column.0.json_path", ".create_date"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.0.additional_printer_column.0.priority", "10"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.0.additional_printer_column.0.format", "dqteTime"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.0.additional_printer_column.1.name", "Spec"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.0.additional_printer_column.1.type", "string"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.0.additional_printer_column.1.description", "The spec of ther basiccrd"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.0.additional_printer_column.1.json_path", ".spec"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.0.additional_printer_column.1.priority", "5"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.0.additional_printer_column.1.format", "password"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.1.name", "v1"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.1.served", "true"),
					resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.1.storage", "false"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.1.subresources.0.scale.0.spec_replicas_path", ".replicas"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.1.subresources.0.scale.0.status_replicas_path", ".status"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.1.subresources.0.scale.0.label_selector_path", ".labels"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.1.additional_printer_column.#", "1"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.1.additional_printer_column.0.name", "Spec"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.1.additional_printer_column.0.type", "string"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.1.additional_printer_column.0.description", "The spec of ther basiccrd"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.1.additional_printer_column.0.json_path", ".spec"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.1.additional_printer_column.0.priority", "5"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.versions.1.additional_printer_column.0.format", "password"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.conversion.0.strategy", "Webhook"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.conversion.0.webhook_client_config.0.url", "http://example.com/webhook"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.conversion.0.webhook_client_config.0.service.0.name", "ConvertService"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.conversion.0.webhook_client_config.0.service.0.namespace", "crdtest.example.com"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.conversion.0.webhook_client_config.0.service.0.path", "http://example.com/service"),
					// resource.TestCheckResourceAttr("kubernetes_custom_resource_definition.test", "spec.0.conversion.0.webhook_client_config.0.ca_bundle", "abcdefhghijk"),
				),
			},
		},
	})
}

func TestAccKubernetesCustomResourceDefinition_importBasic(t *testing.T) {
	resourceName := "kubernetes_custom_resource_definition.test"
	name := fmt.Sprintf("tfacctest%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlpha))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesCustomResourceDefinitionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesCustomResourceDefinitionConfig_basic(name),
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

func testAccCheckKubernetesCustomResourceDefinitionDestroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).ApiextensionsClientset()
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_custom_resource_definition" {
			continue
		}

		resp, err := conn.ApiextensionsV1beta1().CustomResourceDefinitions().Get(rs.Primary.ID, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Custom resource definition still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesCustomResourceDefinitionExists(n string, obj *api.CustomResourceDefinition) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn, err := testAccProvider.Meta().(KubeClientsets).ApiextensionsClientset()
		if err != nil {
			return err
		}

		out, err := conn.ApiextensionsV1beta1().CustomResourceDefinitions().Get(rs.Primary.ID, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesCustomResourceDefinitionConfig_minimal(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_custom_resource_definition" "test" {
  metadata {
    name = "%[1]ss.crdtest.example.com"
  }

  spec {
    scope = "Cluster"
    group = "crdtest.example.com"
    version = "v1"

    names {
      kind = "%[2]s"
	  plural = "%[1]ss"
	  list_kind = "%[2]sList"
      singular = "%[1]s"
	}
	
	versions {
	  name = "v1"
	}
  }
}
`, name, strings.Title(name))
}

func testAccKubernetesCustomResourceDefinitionConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_custom_resource_definition" "test" {
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

    name = "%ss.crdtest.example.com"
  }

  spec {
    scope = "Cluster"
    group = "crdtest.example.com"
    version = "v1"

    names {
      kind = "%[2]s"
      plural = "%[1]ss"
      singular = "%[1]s"
      short_names = ["%[3]s"]
      list_kind = "%[1]slist"
      categories = ["crdtest", "test"]
    }

    # subresources {
    #   scale {
    #     spec_replicas_path = ".replicas"
    #     status_replicas_path = ".status"
    #     label_selector_path = ".labels"
    #   }
    # }

    # additional_printer_column {
    #   name = "Spec"
    #   type = "string"
    #   description = "The spec of the basiccrd"
    #   json_path = ".spec"
    #   priority = 5
    #   format = "password"
    # }
	
	versions {
	  name = "v1"
	}

    # conversion {
    #   strategy = "None"
    # }
  }
}
`, name, strings.Title(name), name[0:15])
}

func testAccKubernetesCustomResourceDefinitionConfig_modified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_custom_resource_definition" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      Different         = "1234"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
    }

    name = "%ss.modcrdtest.example.com"
  }

  spec {
    scope = "Namespaced"
    group = "modcrdtest.example.com"

    names {
      kind = "%[2]s"
      plural = "%[1]ss"
      singular = "%[1]s"
      short_names = ["%[3]s"]
      list_kind = "%[1]slist"
      categories = ["crdtest", "test", "mod"]
	}
	
	version = "v2"

    versions {
      name = "v2"
      served = "true"
      storage = "true"

      # subresources {
      #   scale {
      #     spec_replicas_path = ".v2.replicas"
      #     status_replicas_path = ".v2.status"
      #     label_selector_path = ".v2.labels"
      #   }
      # }

      # additional_printer_column {
      #   name = "Creation Date"
      #   type = "string"
      #   description = "The creation date of the basiccrd"
      #   json_path = ".create_date"
      #   priority = 10
      #   format = "dateTime"
	  # }

      # additional_printer_column {
      #   name = "Spec"
      #   type = "string"
      #   description = "The spec of the basiccrd"
      #   json_path = ".spec"
      #   priority = 5
      #   format = "password"
      # }
	}
  
    versions {
      name = "v1"
      served = "true"
      storage = "false"

      # subresources {
      #   scale {
      #     spec_replicas_path = ".replicas"
      #     status_replicas_path = ".status"
      #     label_selector_path = ".labels"
      #   }
      # }

      # additional_printer_column {
      #   name = "Spec"
      #   type = "string"
      #   description = "The spec of the minimalcrd"
      #   json_path = ".spec"
      #   priority = 5
      #   format = "password"
      # }
    }

    # conversion {
    #   strategy = "Webhook"
    #   webhook_client_config {
    #     url = "http://example.com/webhook"
    #     service {
    #       name = "ConvertService"
    #       namespace = "crdtest.example.com"
    #       path = "http://example.com/service"
    #     }
    #     ca_bundle = "abcdefghijk"
    #   }
    # }
  }
}
`, name, strings.Title(name), name[0:15])
}
