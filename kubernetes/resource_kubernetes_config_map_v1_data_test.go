// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccKubernetesConfigMapV1Data_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	namespace := "default"
	resourceName := "kubernetes_config_map_v1_data.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createConfigMap(name, namespace)
		},

		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			return destroyConfigMap(name, namespace)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfigMapV1Data_empty(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "data.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "field_manager", "tftest"),
				),
			},
			{
				Config: testAccKubernetesConfigMapV1Data_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "data.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "data.test1", "one"),
					resource.TestCheckResourceAttr(resourceName, "data.test2", "two"),
					resource.TestCheckResourceAttr(resourceName, "field_manager", "tftest"),
				),
			},
			{
				Config: testAccKubernetesConfigMapV1Data_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "data.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "data.test1", "one"),
					resource.TestCheckResourceAttr(resourceName, "data.test3", "three"),
					resource.TestCheckResourceAttr(resourceName, "field_manager", "tftest"),
				),
			},
			{
				Config: testAccKubernetesConfigMapV1Data_empty(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "data.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "field_manager", "tftest"),
				),
			},
		},
	})
}

func TestAccKubernetesConfigMapV1Data_validation(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccKubernetesConfigMapV1Data_empty(name),
				ExpectError: regexp.MustCompile("The ConfigMap .* does not exist"),
			},
		},
	})
}

func TestAccKubernetesConfigMapV1Data_binaryData(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_config_map_v1_data.test"
	baseDir := "."
	cwd, _ := os.Getwd()
	if filepath.Base(cwd) != "kubernetes" { // running from test binary
		baseDir = "kubernetes"
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createConfigMap(name, "default")
		},
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			return destroyConfigMap(name, "default")
		},
		Steps: []resource.TestStep{

			{
				Config: testAccKubernetesConfigMapV1Data_binaryData(name, baseDir),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "data.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "data.text", "initial data"),
					resource.TestCheckResourceAttr(resourceName, "binary_data.%", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "binary_data.binary1"),
				),
			},

			{
				Config: testAccKubernetesConfigMapV1Data_binaryDataUpdated(name, baseDir),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "data.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "data.text", "updated data"),
					resource.TestCheckResourceAttr(resourceName, "binary_data.%", "3"),
					resource.TestCheckResourceAttrSet(resourceName, "binary_data.binary1"),
					resource.TestCheckResourceAttrSet(resourceName, "binary_data.binary2"),
					resource.TestCheckResourceAttr(resourceName, "binary_data.inline_binary", "UmF3IGlubGluZSBkYXRh"),
				),
			},
		},
	})
}

func testAccKubernetesConfigMapV1Data_empty(name string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1_data" "test" {
  metadata {
    name = %q
  }
  data          = {}
  field_manager = "tftest"
}
`, name)
}

func testAccKubernetesConfigMapV1Data_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1_data" "test" {
  metadata {
    name = %q
  }
  data = {
    "test1" = "one"
    "test2" = "two"
  }
  field_manager = "tftest"
}
`, name)
}

func testAccKubernetesConfigMapV1Data_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1_data" "test" {
  metadata {
    name = %q
  }
  data = {
    "test1" = "one"
    "test3" = "three"
  }
  field_manager = "tftest"
}
`, name)
}

func testAccKubernetesConfigMapV1Data_binaryData(name string, baseDir string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1_data" "test" {
  metadata {
    name = %q
  }

  data = {
    "text" = "initial data"
  }

  binary_data = {
    "binary1" = "${filebase64("%s/test-fixtures/binary.data")}"
  }

  field_manager = "tftest"
}
`, name, baseDir)
}

func testAccKubernetesConfigMapV1Data_binaryDataUpdated(name string, baseDir string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1_data" "test" {
  metadata {
    name = %q
  }

  data = {
    "text" = "updated data"
  }

  binary_data = {
    "binary1"       = "${filebase64("%s/test-fixtures/binary.data")}"
    "binary2"       = "${filebase64("%s/test-fixtures/binary2.data")}"
    "inline_binary" = "${base64encode("Raw inline data")}"
  }

  field_manager = "tftest"
}
`, name, baseDir, baseDir)
}
