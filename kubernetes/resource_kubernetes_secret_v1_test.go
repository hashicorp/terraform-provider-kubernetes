// Copyright (c) HashiCorp, Inc.
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

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesSecretV1_basic(t *testing.T) {
	var conf1, conf2 corev1.Secret
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_secret_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesSecretV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesSecretV1Config_emptyData(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretV1Exists(resourceName, &conf1),
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
					resource.TestCheckResourceAttr(resourceName, "type", "Opaque"),
					resource.TestCheckResourceAttr(resourceName, "immutable", "false"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "wait_for_service_account_token"},
			},
			{
				Config: testAccKubernetesSecretV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretV1Exists(resourceName, &conf2),
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
					resource.TestCheckResourceAttr(resourceName, "type", "Opaque"),
					testAccCheckSecretV1Data(&conf2, map[string]string{"one": "first", "two": "second"}),
					testAccCheckSecretV1NotRecreated(&conf1, &conf2),
				),
			},
			{
				Config: testAccKubernetesSecretV1Config_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretV1Exists(resourceName, &conf1),
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
					resource.TestCheckResourceAttr(resourceName, "type", "Opaque"),
					testAccCheckSecretV1Data(&conf1, map[string]string{"one": "first", "two": "second", "nine": "ninth"}),
				),
			},
			{
				Config: testAccKubernetesSecretV1Config_noData(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretV1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "data.%", "0"),
				),
			},
			{
				Config: testAccKubernetesSecretV1Config_typeSpecified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretV1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "data.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "data.username", "admin"),
					resource.TestCheckResourceAttr(resourceName, "data.password", "password"),
					resource.TestCheckResourceAttr(resourceName, "type", "kubernetes.io/basic-auth"),
					testAccCheckSecretV1Data(&conf1, map[string]string{"username": "admin", "password": "password"}),
				),
			},
		},
	})
}

func TestAccKubernetesSecretV1_immutable(t *testing.T) {
	var conf1, conf2, conf3, conf4 corev1.Secret

	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_secret_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesSecretV1Destroy,
		Steps: []resource.TestStep{
			// create an immutable secret
			{
				Config: testAccKubernetesSecretV1Config_immutable(name, true, "password"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretV1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "immutable", "true"),
				),
			},
			// import the secret
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "wait_for_service_account_token"},
			},
			// changing the data for the immutable secret will force recreate
			{
				Config: testAccKubernetesSecretV1Config_immutable(name, true, "newpassword"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "immutable", "true"),
					testAccCheckSecretV1Recreated(&conf1, &conf2),
				),
			},
			// change immutable back to false will force recreate
			{
				Config: testAccKubernetesSecretV1Config_immutable(name, false, "password"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretV1Exists(resourceName, &conf3),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "immutable", "false"),
					testAccCheckSecretV1Recreated(&conf2, &conf3),
				),
			},
			// change immutable from false to true wont force recreate
			{
				Config: testAccKubernetesSecretV1Config_immutable(name, true, "password"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretV1Exists(resourceName, &conf4),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "immutable", "true"),
					testAccCheckSecretV1NotRecreated(&conf3, &conf4),
				),
			},
		},
	})
}

func TestAccKubernetesSecretV1_dotInName(t *testing.T) {
	var conf corev1.Secret
	resourceName := "kubernetes_secret_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesSecretV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesSecretV1Config_dotInSecretName,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", "dot.test"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "wait_for_service_account_token"},
			},
		},
	})
}

func TestAccKubernetesSecretV1_generatedName(t *testing.T) {
	var conf corev1.Secret
	prefix := "tf-acc-test-gen-"
	resourceName := "kubernetes_secret_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesSecretV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesSecretV1Config_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretV1Exists(resourceName, &conf),
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
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "wait_for_service_account_token"},
			},
		},
	})
}

func TestAccKubernetesSecretV1_binaryData(t *testing.T) {
	var conf corev1.Secret
	prefix := "tf-acc-test-gen-"
	resourceName := "kubernetes_secret_v1.test"
	baseDir := "."
	cwd, _ := os.Getwd()
	if filepath.Base(cwd) != "kubernetes" { // running from test binary
		baseDir = "kubernetes"
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesSecretV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesSecretV1Config_binaryData(prefix, baseDir),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "binary_data.%", "1"),
				),
			},
			{
				Config: testAccKubernetesSecretV1Config_binaryData2(prefix, baseDir),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "binary_data.%", "2"),
				),
			},
			{
				Config: testAccKubernetesSecretV1Config_binaryDataCombined(prefix, baseDir),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "data.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "binary_data.%", "2"),
				),
			},
		},
	})
}

func TestAccKubernetesSecretV1_data_wo(t *testing.T) {
	var conf corev1.Secret
	prefix := "tf-acc-test-gen-"
	resourceName := "kubernetes_secret_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version", "metadata.0.labels", "metadata.0.annotations"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesSecretV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesSecretV1Config_data_wo(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "data_wo_revision", "1"),
					resource.TestCheckResourceAttr(resourceName, "binary_data.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "data.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "binary_data_wo.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "data_wo.%", "0"),
				),
			},
			{
				Config: testAccKubernetesSecretV1Config_data_wo2(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "data_wo_revision", "2"),
					resource.TestCheckResourceAttr(resourceName, "binary_data.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "data.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "binary_data_wo.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "data_wo.%", "0"),
				),
			},
		},
	})
}

func TestAccKubernetesSecretV1_binaryData_wo(t *testing.T) {
	var conf corev1.Secret
	prefix := "tf-acc-test-gen-"
	resourceName := "kubernetes_secret_v1.test"
	baseDir := "."
	cwd, _ := os.Getwd()
	if filepath.Base(cwd) != "kubernetes" { // running from test binary
		baseDir = "kubernetes"
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version", "metadata.0.labels", "metadata.0.annotations"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesSecretV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesSecretV1Config_binaryData_wo(prefix, baseDir),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "binary_data_wo_revision", "1"),
					resource.TestCheckResourceAttr(resourceName, "binary_data.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "data.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "binary_data_wo.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "data_wo.%", "0"),
				),
			},
			{
				Config: testAccKubernetesSecretV1Config_binaryData_wo2(prefix, baseDir),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "binary_data_wo_revision", "2"),
					resource.TestCheckResourceAttr(resourceName, "binary_data.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "data.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "binary_data_wo.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "data_wo.%", "0"),
				),
			},
		},
	})
}

func TestAccKubernetesSecretV1_service_account_token(t *testing.T) {
	var conf corev1.Secret
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_secret_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesSecretV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesSecretV1Config_service_account_token(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesSecretV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttrSet(resourceName, "data.token"),
				),
			},
		},
	})
}

func testAccCheckSecretV1Data(m *corev1.Secret, expected map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(expected) == 0 && len(m.Data) == 0 {
			return nil
		}
		if !reflect.DeepEqual(flattenByteMapToStringMap(m.Data), expected) {
			return fmt.Errorf("%s data don't match.\nExpected: %q\nGiven: %q",
				m.Name, expected, m.Data)
		}
		return nil
	}
}

func testAccCheckSecretV1Recreated(sec1, sec2 *corev1.Secret) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		recreated := sec1.GetUID() != sec2.GetUID()
		if !recreated {
			return fmt.Errorf("secret %q should have been recreated", sec1.GetName())
		}
		return nil
	}
}

func testAccCheckSecretV1NotRecreated(sec1, sec2 *corev1.Secret) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		recreated := sec1.GetUID() != sec2.GetUID()
		if recreated {
			return fmt.Errorf("secret %q should not have been recreated", sec1.GetName())
		}
		return nil
	}
}

func testAccCheckKubernetesSecretV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_secret" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Secret still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesSecretV1Exists(n string, obj *corev1.Secret) resource.TestCheckFunc {
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

		out, err := conn.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

const testAccKubernetesSecretV1Config_dotInSecretName = `
resource "kubernetes_secret_v1" "test" {
  metadata {
    name = "dot.test"
  }
}
`

func testAccKubernetesSecretV1Config_emptyData(name string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
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

func testAccKubernetesSecretV1Config_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
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

func testAccKubernetesSecretV1Config_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
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

func testAccKubernetesSecretV1Config_noData(name string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
  metadata {
    name = "%s"
  }

  data = {}
}
`, name)
}

func testAccKubernetesSecretV1Config_typeSpecified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
  metadata {
    name = "%s"
  }

  data = {
    username = "admin"
    password = "password"
  }

  type = "kubernetes.io/basic-auth"
}
`, name)
}

func testAccKubernetesSecretV1Config_generatedName(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
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

func testAccKubernetesSecretV1Config_binaryData(prefix string, bd string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
  metadata {
    generate_name = "%s"
  }

  binary_data = {
    one = filebase64("%s/test-fixtures/binary.data")
  }
}
`, prefix, bd)
}

func testAccKubernetesSecretV1Config_data_wo(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
  metadata {
    generate_name = "%s"
    annotations = {
      TestAnnotationOne = "one"
      Different         = "1234"
    }
    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
    }
  }

  data_wo_revision = 1
  data_wo = {
    one = "one"
  }
}
`, prefix)
}

func testAccKubernetesSecretV1Config_data_wo2(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
  metadata {
    generate_name = "%s"
    annotations = {
      TestAnnotationOne = "one"
      Different         = "1234"
    }
    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
    }
  }

  data_wo_revision = 2
  data_wo = {
    one = "one"
    two = "two"
  }
}
`, prefix)
}

func testAccKubernetesSecretV1Config_binaryData_wo(prefix string, bd string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
  metadata {
    generate_name = "%s"
    annotations = {
      TestAnnotationOne = "one"
      Different         = "1234"
    }
    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
    }
  }

  binary_data_wo_revision = 1
  binary_data_wo = {
    one = filebase64("%s/test-fixtures/binary.data")
  }
}
`, prefix, bd)
}

func testAccKubernetesSecretV1Config_binaryData_wo2(prefix string, bd string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
  metadata {
    generate_name = "%s"
    annotations = {
      TestAnnotationOne = "one"
      Different         = "1234"
    }
    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
    }
  }

  binary_data_wo_revision = 2
  binary_data_wo = {
    one = filebase64("%s/test-fixtures/binary.data")
    two = filebase64("%s/test-fixtures/binary2.data")
  }
}
`, prefix, bd, bd)
}

func testAccKubernetesSecretV1Config_binaryData2(prefix string, bd string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
  metadata {
    generate_name = "%s"
  }

  binary_data = {
    one = filebase64("%s/test-fixtures/binary.data")
    two = filebase64("%s/test-fixtures/binary2.data")
  }
}
`, prefix, bd, bd)
}

func testAccKubernetesSecretV1Config_binaryDataCombined(prefix string, bd string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
  metadata {
    generate_name = "%s"
  }

  data = {
    "HOST" = "127.0.0.1"
    "PORT" = "80"
  }

  binary_data = {
    one = filebase64("%s/test-fixtures/binary.data")
    two = filebase64("%s/test-fixtures/binary2.data")
  }
}
`, prefix, bd, bd)
}

func testAccKubernetesSecretV1Config_immutable(name string, immutable bool, data string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
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
    SECRET = %q
  }
}
`, name, immutable, data)
}

func testAccKubernetesSecretV1Config_service_account_token(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service_account_v1" "test" {
  metadata {
    name = "%s"
  }
}

resource "kubernetes_secret_v1" "test" {
  metadata {
    annotations = {
      "kubernetes.io/service-account.name" = kubernetes_service_account_v1.test.metadata[0].name
    }
    name = "%s-token"
  }
  type = "kubernetes.io/service-account-token"
}
`, name, name)
}
