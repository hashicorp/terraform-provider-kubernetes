package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	api "k8s.io/api/scheduling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesPriorityClass_basic(t *testing.T) {
	var conf api.PriorityClass
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_priority_class.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPriorityClassDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPriorityClassConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPriorityClassExists(resourceName, &conf),
					resource.TestCheckResourceAttr("kubernetes_priority_class.test", "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_priority_class.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_priority_class.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_priority_class.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_priority_class.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("kubernetes_priority_class.test", "metadata.0.labels.TestLabelFour", "four"),
					resource.TestCheckResourceAttr("kubernetes_priority_class.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_priority_class.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_priority_class.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_priority_class.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_priority_class.test", "value", "100"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesPriorityClassConfig_metaModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPriorityClassExists(resourceName, &conf),
					resource.TestCheckResourceAttr("kubernetes_priority_class.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_priority_class.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_priority_class.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_priority_class.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_priority_class.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_priority_class.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_priority_class.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("kubernetes_priority_class.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_priority_class.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_priority_class.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_priority_class.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_priority_class.test", "value", "100"),
				),
			},
			{
				Config: testAccKubernetesPriorityClassConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPriorityClassExists(resourceName, &conf),
					resource.TestCheckResourceAttr("kubernetes_priority_class.test", "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_priority_class.test", "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_priority_class.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_priority_class.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_priority_class.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_priority_class.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_priority_class.test", "value", "100"),
					resource.TestCheckResourceAttr("kubernetes_priority_class.test", "description", "Foobar"),
					resource.TestCheckResourceAttr("kubernetes_priority_class.test", "global_default", "true"),
				),
			},
		},
	})
}

func TestAccKubernetesPriorityClass_generatedName(t *testing.T) {
	var conf api.PriorityClass
	prefix := "tf-acc-test-"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_priority_class.test",
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPriorityClassDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPriorityClassConfig_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPriorityClassExists("kubernetes_priority_class.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_priority_class.test", "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_priority_class.test", "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_priority_class.test", "metadata.0.generate_name", prefix),
					resource.TestCheckResourceAttrSet("kubernetes_priority_class.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_priority_class.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_priority_class.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_priority_class.test", "value", "999"),
				),
			},
		},
	})
}

func testAccCheckKubernetesPriorityClassDestroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_priority_class" {
			continue
		}

		name := rs.Primary.ID

		resp, err := conn.SchedulingV1().PriorityClasses().Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == name {
				return fmt.Errorf("Resource Quota still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesPriorityClassExists(n string, obj *api.PriorityClass) resource.TestCheckFunc {
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

		name := rs.Primary.ID

		out, err := conn.SchedulingV1().PriorityClasses().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesPriorityClassConfig_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_priority_class" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
      TestLabelFour  = "four"
    }

    name = "%s"
  }

  value = 100
}
`, name)
}

func testAccKubernetesPriorityClassConfig_metaModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_priority_class" "test" {
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

  value = 100
}
`, name)
}

func testAccKubernetesPriorityClassConfig_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_priority_class" "test" {
  metadata {
    name = "%s"
  }

  value          = 100
  description    = "Foobar"
  global_default = true
}
`, name)
}

func testAccKubernetesPriorityClassConfig_generatedName(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_priority_class" "test" {
  metadata {
    generate_name = "%s"
  }

  value = 999
}
`, prefix)
}
