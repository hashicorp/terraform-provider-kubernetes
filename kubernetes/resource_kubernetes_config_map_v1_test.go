// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccKubernetesConfigMapV1_basic(t *testing.T) {
	var conf corev1.ConfigMap
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_config_map_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesConfigMapV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfigMapV1Config_nodata(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapV1Exists(resourceName, &conf),
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
					resource.TestCheckResourceAttr(resourceName, "immutable", "false"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesConfigMapV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapV1Exists(resourceName, &conf),
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
					testAccCheckConfigMapV1Data(&conf, map[string]string{"one": "first", "two": "second"}),
				),
			},
			{
				Config: testAccKubernetesConfigMapV1Config_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapV1Exists(resourceName, &conf),
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
					testAccCheckConfigMapV1Data(&conf, map[string]string{"one": "first", "two": "second", "nine": "ninth"}),
				),
			},
			{
				Config: testAccKubernetesConfigMapV1Config_noData(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "data.%", "0"),
					testAccCheckConfigMapV1Data(&conf, map[string]string{}),
				),
			},
		},
	})
}

func TestAccKubernetesConfigMapV1_binaryData(t *testing.T) {
	var conf corev1.ConfigMap
	prefix := "tf-acc-test-gen-"
	resourceName := "kubernetes_config_map_v1.test"
	baseDir := "."
	cwd, _ := os.Getwd()
	if filepath.Base(cwd) != "kubernetes" { // running from test binary
		baseDir = "kubernetes"
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesConfigMapV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfigMapV1Config_binaryData(prefix, baseDir),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "binary_data.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "data.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "data.two", "second"),
				),
			},
			{
				Config: testAccKubernetesConfigMapV1Config_binaryData2(prefix, baseDir),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapV1Exists(resourceName, &conf),
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
	var conf corev1.ConfigMap
	prefix := "tf-acc-test-gen-"
	resourceName := "kubernetes_config_map_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesConfigMapV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfigMapV1Config_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapV1Exists(resourceName, &conf),
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
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
		},
	})
}

func TestAccKubernetesConfigMap_immutable(t *testing.T) {
	var conf corev1.ConfigMap
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_config_map_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesConfigMapV1Destroy,
		Steps: []resource.TestStep{
			// create config_map with immutable set to true
			{
				Config: testAccKubernetesConfigMapV1Config_immutable(name, true, "second"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "data.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "data.one", "first"),
					resource.TestCheckResourceAttr(resourceName, "data.two", "second"),
					resource.TestCheckResourceAttr(resourceName, "immutable", "true"),
				),
			},
			// change the immutable variable to false
			{
				Config: testAccKubernetesConfigMapV1Config_immutable(name, false, "second"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "immutable", "false"),
					resource.TestCheckResourceAttr(resourceName, "data.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "data.one", "first"),
					resource.TestCheckResourceAttr(resourceName, "data.two", "second"),
				),
			},
			// update data while immutable is false
			{
				Config: testAccKubernetesConfigMapV1Config_immutable(name, false, "third"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesConfigMapV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "immutable", "false"),
					resource.TestCheckResourceAttr(resourceName, "data.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "data.one", "first"),
					resource.TestCheckResourceAttr(resourceName, "data.two", "third"),
				),
			},
		},
	})
}

func TestAccKubernetesConfigMapV1_identity(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_config_map_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesConfigMapV1Destroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfigMapV1Config_basic(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(
						resourceName, map[string]knownvalue.Check{
							"namespace":   knownvalue.StringExact("default"),
							"name":        knownvalue.StringExact(name),
							"api_version": knownvalue.StringExact("v1"),
							"kind":        knownvalue.StringExact("ConfigMap"),
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

func testAccCheckConfigMapV1Data(m *corev1.ConfigMap, expected map[string]string) resource.TestCheckFunc {
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

func testAccCheckKubernetesConfigMapV1Destroy(s *terraform.State) error {
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

func testAccCheckKubernetesConfigMapV1Exists(n string, obj *corev1.ConfigMap) resource.TestCheckFunc {
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

func testAccKubernetesConfigMapV1Config_nodata(name string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1" "test" {
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

func testAccKubernetesConfigMapV1Config_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1" "test" {
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

func testAccKubernetesConfigMapV1Config_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1" "test" {
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

func testAccKubernetesConfigMapV1Config_noData(name string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1" "test" {
  metadata {
    name = "%s"
  }
}
`, name)
}

func testAccKubernetesConfigMapV1Config_generatedName(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1" "test" {
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

func testAccKubernetesConfigMapV1Config_binaryData(prefix string, bd string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1" "test" {
  metadata {
    generate_name = "%s"
  }

  binary_data = {
    one = "${filebase64("%s/test-fixtures/binary.data")}"
  }

  data = {
    two = "second"
  }
}
`, prefix, bd)
}

func testAccKubernetesConfigMapV1Config_binaryData2(prefix string, bd string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1" "test" {
  metadata {
    generate_name = "%s"
  }

  binary_data = {
    one = "${filebase64("%s/test-fixtures/binary.data")}"
    two = "${filebase64("%s/test-fixtures/binary2.data")}"
    raw = "${base64encode("Raw data should come back as is in the pod")}"
  }

  data = {
    three = "third"
  }
}
`, prefix, bd, bd)
}

func testAccKubernetesConfigMapV1Config_immutable(name string, immutable bool, data string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1" "test" {
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

  immutable = %t

  data = {
    one = "first"
    two = "%s"
  }
}
`, name, immutable, data)
}
