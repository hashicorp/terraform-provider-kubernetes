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
)

func TestAccKubernetesConfigMap_basic(t *testing.T) {
	var conf api.ConfigMap
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_config_map.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesConfigMapDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfigMapConfig_nodata(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "data.%", "0"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesConfigMapConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "data.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "data.one", "first"),
					resource.TestCheckResourceAttr(resourceName, "data.two", "second"),
					testAccCheckConfigMapData(&conf, map[string]string{"one": "first", "two": "second"}),
				),
			},
			{
				Config: testAccKubernetesConfigMapConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.Different", "1234"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "data.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "data.one", "first"),
					resource.TestCheckResourceAttr(resourceName, "data.two", "second"),
					resource.TestCheckResourceAttr(resourceName, "data.nine", "ninth"),
					testAccCheckConfigMapData(&conf, map[string]string{"one": "first", "two": "second", "nine": "ninth"}),
				),
			},
			{
				Config: testAccKubernetesConfigMapConfig_noData(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "data.%", "0"),
					testAccCheckConfigMapData(&conf, map[string]string{}),
				),
			},
		},
	})
}
func TestAccKubernetesConfigMap_binaryData(t *testing.T) {
	var conf api.ConfigMap
	prefix := "tf-acc-test-gen-"
	resourceName := "kubernetes_config_map.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesConfigMapDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfigMapConfig_binaryData(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "binary_data.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "data.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "data.two", "second"),
				),
			},
			{
				Config: testAccKubernetesConfigMapConfig_binaryData2(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "binary_data.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "binary_data.raw", "UmF3IGRhdGEgc2hvdWxkIGNvbWUgYmFjayBhcyBpcyBpbiB0aGUgcG9k"),
					resource.TestCheckResourceAttr(resourceName, "data.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "data.three", "third"),
				),
			},
		},
	})
}

func TestAccKubernetesConfigMap_generatedName(t *testing.T) {
	var conf api.ConfigMap
	prefix := "tf-acc-test-gen-"
	resourceName := "kubernetes_config_map.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesConfigMapDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfigMapConfig_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapExists(resourceName, &conf),
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
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckConfigMapData(m *api.ConfigMap, expected map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(expected) == 0 && len(m.Data) == 0 {
			return nil
		}
		if !reflect.DeepEqual(m.Data, expected) {
			return fmt.Errorf("%s data don't match.\nExpected: %q\nGiven: %q",
				m.Name, expected, m.Data)
		}
		return nil
	}
}

func testAccCheckKubernetesConfigMapDestroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_config_map" {
			continue
		}
		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}
		resp, err := conn.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Config Map still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesConfigMapExists(n string, obj *api.ConfigMap) resource.TestCheckFunc {
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
		out, err := conn.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func deleteConfigMap(t *testing.T, obj *api.ConfigMap) {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		t.Error(err)
		return
	}

	ctx := context.TODO()
	err = conn.CoreV1().ConfigMaps(
		obj.ObjectMeta.GetNamespace()).Delete(
		ctx, obj.ObjectMeta.GetName(), metav1.DeleteOptions{})
	if err != nil {
		t.Error(err)
		return
	}
}

func testAccKubernetesConfigMapConfig_nodata(name string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map" "test" {
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

  data = {}
}
`, name)
}

func testAccKubernetesConfigMapConfig_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map" "test" {
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

  data = {
    one = "first"
    two = "second"
  }
}
`, name)
}

func testAccKubernetesConfigMapConfig_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map" "test" {
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

  data = {
    one  = "first"
    two  = "second"
    nine = "ninth"
  }
}
`, name)
}

func testAccKubernetesConfigMapConfig_noData(name string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map" "test" {
  metadata {
    name = "%s"
  }
}
`, name)
}

func testAccKubernetesConfigMapConfig_generatedName(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map" "test" {
  metadata {
    generate_name = "%s"
  }

  data = {
    one = "first"
    two = "second"
  }
}
`, prefix)
}

func testAccKubernetesConfigMapConfig_binaryData(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map" "test" {
  metadata {
    generate_name = "%s"
  }

  binary_data = {
    one = "${filebase64("./test-fixtures/binary.data")}"
  }

  data = {
    two = "second"
  }
}
`, prefix)
}

func testAccKubernetesConfigMapConfig_binaryData2(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map" "test" {
  metadata {
    generate_name = "%s"
  }

  binary_data = {
    one = "${filebase64("./test-fixtures/binary.data")}"
    two = "${filebase64("./test-fixtures/binary2.data")}"
    raw = "${base64encode("Raw data should come back as is in the pod")}"
  }

  data = {
    three = "third"
  }
}
`, prefix)
}
