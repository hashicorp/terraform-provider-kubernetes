package kubernetes

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesNamespace_basic(t *testing.T) {
	var conf api.Namespace
	nsName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_namespace.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_namespace.test",
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceConfig_basic(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceExists(resourceName, &conf),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.name", nsName),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.uid"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesNamespaceConfig_addAnnotations(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceExists("kubernetes_namespace.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.name", nsName),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.uid"),
				),
			},
			{
				Config: testAccKubernetesNamespaceConfig_addLabels(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceExists("kubernetes_namespace.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.name", nsName),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.uid"),
				),
			},
			{
				Config: testAccKubernetesNamespaceConfig_smallerLists(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceExists("kubernetes_namespace.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.annotations.Different", "1234"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.name", nsName),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.uid"),
				),
			},
			{
				Config: testAccKubernetesNamespaceConfig_noLists(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceExists("kubernetes_namespace.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.name", nsName),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.uid"),
				),
			},
		},
	})
}

func TestAccKubernetesNamespace_generatedName(t *testing.T) {
	var conf api.Namespace
	prefix := "tf-acc-test-gen-"
	resourceName := "kubernetes_namespace.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_namespace.test",
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceConfig_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceExists("kubernetes_namespace.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr("kubernetes_namespace.test", "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.uid"),
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

func TestAccKubernetesNamespace_withSpecialCharacters(t *testing.T) {
	var conf api.Namespace
	nsName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_namespace.test",
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceConfig_specialCharacters(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceExists("kubernetes_namespace.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.annotations.%", "2"),
					//  "myhost.co.uk/any-path": "one",
					//  "Different":             "1234",
					//}),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.annotations.myhost.co.uk/any-path", "one"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.annotations.Different", "1234"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.labels.%", "2"),
					//  "myhost.co.uk/any-path": "one",
					//  "TestLabelThree":        "three",
					//}),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.labels.myhost.co.uk/any-path", "one"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.name", nsName),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.uid"),
				),
			},
		},
	})
}

func TestAccKubernetesNamespace_deleteTimeout(t *testing.T) {
	var conf api.Namespace
	nsName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_namespace.test",
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceConfig_deleteTimeout(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceExists("kubernetes_namespace.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_namespace.test", "metadata.0.name", nsName),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_namespace.test", "metadata.0.uid"),
				),
			},
		},
	})
}

func testAccCheckKubernetesNamespaceDestroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_namespace" {
			continue
		}

		resp, err := conn.CoreV1().Namespaces().Get(ctx, rs.Primary.ID, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Namespace still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesNamespaceExists(n string, obj *api.Namespace) resource.TestCheckFunc {
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

		out, err := conn.CoreV1().Namespaces().Get(ctx, rs.Primary.ID, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesNamespaceConfig_basic(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace" "test" {
  metadata {
    name = "%s"
  }
}
`, nsName)
}

func testAccKubernetesNamespaceConfig_addAnnotations(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }
    name = "%s"
  }
}
`, nsName)
}
func testAccKubernetesNamespaceConfig_addLabels(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace" "test" {
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
}
`, nsName)
}

func testAccKubernetesNamespaceConfig_smallerLists(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace" "test" {
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
}
`, nsName)
}

func testAccKubernetesNamespaceConfig_noLists(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace" "test" {
  metadata {
    name = "%s"
  }
}
`, nsName)
}

func testAccKubernetesNamespaceConfig_generatedName(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace" "test" {
  metadata {
    generate_name = "%s"
  }
}
`, prefix)
}

func testAccKubernetesNamespaceConfig_specialCharacters(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace" "test" {
  metadata {
    annotations = {
      "myhost.co.uk/any-path" = "one"
      "Different"             = "1234"
    }

    labels = {
      "myhost.co.uk/any-path" = "one"
      "TestLabelThree"        = "three"
    }

    name = "%s"
  }
}
`, nsName)
}

func testAccKubernetesNamespaceConfig_deleteTimeout(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace" "test" {
  metadata {
    name = "%s"
  }
  timeouts {
    delete = "30m"
  }
}
`, nsName)
}
