// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAccKubernetesDataSourceConfigMap_basic tests that the data source is able to read
// plaintext data, binary data, annotation, label, and name of the config map resource.
func TestAccKubernetesDataSourceConfigMapV1_basic(t *testing.T) {
	resourceName := "kubernetes_config_map_v1.test"
	dataSourceName := "data.kubernetes_config_map_v1.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{ // First, create the resource. Data sources are evaluated before resources, and therefore need to be created in a second apply.
				Config: testAccKubernetesDataSourceConfigMapV1_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "data.one", "first"),
					resource.TestCheckResourceAttr(resourceName, "binary_data.raw", "UmF3IGRhdGEgc2hvdWxkIGNvbWUgYmFjayBhcyBpcyBpbiB0aGUgcG9k"),
				),
			},
			{ // Use the data source to read the existing resource.
				Config: testAccKubernetesDataSourceConfigMapV1_basic(name) +
					testAccKubernetesDataSourceConfigMapV1_read(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(dataSourceName, "data.one", "first"),
					resource.TestCheckResourceAttr(dataSourceName, "binary_data.raw", "UmF3IGRhdGEgc2hvdWxkIGNvbWUgYmFjayBhcyBpcyBpbiB0aGUgcG9k"),
				),
			},
		},
	})
}

func TestAccKubernetesDataSourceConfigMapV1_not_found(t *testing.T) {
	dataSourceName := "data.kubernetes_config_map_v1.test"
	name := fmt.Sprintf("ceci-n.est-pas-une-config-map-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{ // Use the data source to read the existing resource.
				Config: testAccKubernetesDataSourceConfigMapV1_nonexistent(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(dataSourceName, "data.%", "0"),
				),
			},
		},
	})
}

// testAccKubernetesDataSourceConfigMapConfig_basic provides the terraform config
// used to test basic functionality of the config_map data source.
func testAccKubernetesDataSourceConfigMapV1_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
    }

    labels = {
      TestLabelOne = "one"
    }

    name = "%s"
  }

  data = {
    one = "first"
  }

  binary_data = {
    raw = "${base64encode("Raw data should come back as is in the pod")}"
  }
}
`, name)
}

func testAccKubernetesDataSourceConfigMapV1_read() string {
	return `data "kubernetes_config_map_v1" "test" {
  metadata {
    name = "${kubernetes_config_map_v1.test.metadata.0.name}"
  }
}
`
}

func testAccKubernetesDataSourceConfigMapV1_nonexistent(name string) string {
	return fmt.Sprintf(`data "kubernetes_config_map_v1" "test" {
  metadata {
    name = "%s"
  }
}
`, name)
}
