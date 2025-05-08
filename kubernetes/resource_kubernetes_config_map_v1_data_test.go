// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"
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
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
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
	resourceName := "kubernetes_config_map_v1_data.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccKubernetesConfigMapV1Data_empty(name),
				ExpectError: regexp.MustCompile("The ConfigMap .* does not exist"),
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
