// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccKubernetesDataSourceSecretV1_basic creates a secret resource and reads it
// back via data.kubernetes_secret_v1, asserting all metadata.0.* paths, data keys,
// type, and immutable are consistent.
func TestAccKubernetesDataSourceSecretV1_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	datasourceName := "data.kubernetes_secret_v1.test"
	resourceName := "kubernetes_secret_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create the secret.
				Config: testAccKubernetesDataSourceSecretV1Config_basic(name),
			},
			{
				// Read it back and assert.
				Config: testAccKubernetesDataSourceSecretV1Config_basic(name) +
					testAccKubernetesDataSourceSecretV1Config_read(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Regression for K8S-MIGRATE-006: id must be present and equal namespace/name.
					resource.TestCheckResourceAttr(datasourceName, "id", fmt.Sprintf("default/%s", name)),
					resource.TestCheckResourceAttrPair(datasourceName, "metadata.0.name", resourceName, "metadata.0.name"),
					resource.TestCheckResourceAttrPair(datasourceName, "metadata.0.namespace", resourceName, "metadata.0.namespace"),
					resource.TestCheckResourceAttrPair(datasourceName, "metadata.0.generation", resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrPair(datasourceName, "metadata.0.resource_version", resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrPair(datasourceName, "metadata.0.uid", resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttrPair(datasourceName, "metadata.0.annotations.%", resourceName, "metadata.0.annotations.%"),
					resource.TestCheckResourceAttrPair(datasourceName, "metadata.0.annotations.TestAnnotationOne", resourceName, "metadata.0.annotations.TestAnnotationOne"),
					resource.TestCheckResourceAttrPair(datasourceName, "metadata.0.annotations.TestAnnotationTwo", resourceName, "metadata.0.annotations.TestAnnotationTwo"),
					resource.TestCheckResourceAttrPair(datasourceName, "metadata.0.labels.TestLabelOne", resourceName, "metadata.0.labels.TestLabelOne"),
					resource.TestCheckResourceAttrPair(datasourceName, "metadata.0.labels.TestLabelTwo", resourceName, "metadata.0.labels.TestLabelTwo"),
					resource.TestCheckResourceAttrPair(datasourceName, "metadata.0.labels.TestLabelThree", resourceName, "metadata.0.labels.TestLabelThree"),
					resource.TestCheckResourceAttrPair(datasourceName, "data.%", resourceName, "data.%"),
					resource.TestCheckResourceAttrPair(datasourceName, "data.one", resourceName, "data.one"),
					resource.TestCheckResourceAttrPair(datasourceName, "data.two", resourceName, "data.two"),
					resource.TestCheckResourceAttrPair(datasourceName, "type", resourceName, "type"),
					resource.TestCheckResourceAttrPair(datasourceName, "immutable", resourceName, "immutable"),
					resource.TestCheckResourceAttrPair(datasourceName, "binary_data.raw", resourceName, "binary_data.raw"),
				),
			},
		},
	})
}

// TestAccKubernetesDataSourceSecretV1_generateName creates a secret with
// generate_name and reads it back using the server-assigned name.
func TestAccKubernetesDataSourceSecretV1_generateName(t *testing.T) {
	generateName := "testing-name-"
	datasourceName := "data.kubernetes_secret_v1.test"
	resourceName := "kubernetes_secret_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceSecretV1Config_generateName(generateName),
			},
			{
				Config: testAccKubernetesDataSourceSecretV1Config_generateName(generateName) +
					testAccKubernetesDataSourceSecretV1Config_readGenerateName(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(datasourceName, "metadata.0.name", resourceName, "metadata.0.name"),
					resource.TestCheckResourceAttrPair(datasourceName, "metadata.0.generation", resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrPair(datasourceName, "metadata.0.resource_version", resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrPair(datasourceName, "metadata.0.uid", resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttrPair(datasourceName, "data.%", resourceName, "data.%"),
					resource.TestCheckResourceAttrPair(datasourceName, "data.one", resourceName, "data.one"),
					resource.TestCheckResourceAttrPair(datasourceName, "data.two", resourceName, "data.two"),
					resource.TestCheckResourceAttrPair(datasourceName, "type", resourceName, "type"),
					resource.TestCheckResourceAttrPair(datasourceName, "immutable", resourceName, "immutable"),
					resource.TestCheckResourceAttrPair(datasourceName, "binary_data.raw", resourceName, "binary_data.raw"),
				),
			},
		},
	})
}

// TestAccKubernetesDataSourceSecretV1_not_found reads a non-existent secret.
// The Framework data source must return empty state with no error — matching SDKv2 behaviour.
func TestAccKubernetesDataSourceSecretV1_not_found(t *testing.T) {
	name := fmt.Sprintf("ceci-n-est-pas-un-secret-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	datasourceName := "data.kubernetes_secret_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceSecretV1Config_notFound(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(datasourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(datasourceName, "data.%", "0"),
				),
			},
		},
	})
}

// TestAccKubernetesDataSourceSecretV1_binaryData verifies the selective binary_data
// extraction: declared binary keys appear in binary_data (base64-encoded) and are
// absent from data; undeclared string keys appear only in data.
func TestAccKubernetesDataSourceSecretV1_binaryData(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	datasourceName := "data.kubernetes_secret_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceSecretV1Config_binaryData(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					// The binary key must appear in binary_data (base64) …
					resource.TestCheckResourceAttrSet(datasourceName, "binary_data.raw"),
					// … and must NOT appear in data.
					resource.TestCheckNoResourceAttr(datasourceName, "data.raw"),
					// The plain text key must appear only in data.
					resource.TestCheckResourceAttr(datasourceName, "data.one", "first"),
					resource.TestCheckNoResourceAttr(datasourceName, "binary_data.one"),
				),
			},
		},
	})
}

// TestAccKubernetesDataSourceSecretV1_ignoreMetadata proves that configured
// ignore_annotations and ignore_labels do NOT filter metadata returned by the
// Framework data source — matching the SDKv2 flattenMetadataFields (unfiltered) behaviour.
func TestAccKubernetesDataSourceSecretV1_ignoreMetadata(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	datasourceName := "data.kubernetes_secret_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceSecretV1Config_ignoreMetadata(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					// The ignored annotation must still be present in the data source output.
					resource.TestCheckResourceAttr(datasourceName, "metadata.0.annotations.ignored-annotation", "yes"),
					// The ignored label must still be present in the data source output.
					resource.TestCheckResourceAttr(datasourceName, "metadata.0.labels.ignored-label", "yes"),
				),
			},
		},
	})
}

// ─── HCL config helpers ───────────────────────────────────────────────────────

func testAccKubernetesDataSourceSecretV1Config_basic(name string) string {
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
    name = %q
  }
  data = {
    one = "first"
    two = "second"
  }
  binary_data = {
    raw = "${base64encode("Raw data should come back as is in the pod")}"
  }
}
`, name)
}

func testAccKubernetesDataSourceSecretV1Config_read() string {
	return `data "kubernetes_secret_v1" "test" {
  metadata {
    name = kubernetes_secret_v1.test.metadata.0.name
  }
  binary_data = {
    raw = ""
  }
}
`
}

func testAccKubernetesDataSourceSecretV1Config_generateName(generateName string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
  metadata {
    generate_name = %q
  }
  data = {
    one = "first"
    two = "second"
  }
  binary_data = {
    raw = "${base64encode("Raw data should come back as is in the pod")}"
  }
}
`, generateName)
}

func testAccKubernetesDataSourceSecretV1Config_readGenerateName() string {
	return `data "kubernetes_secret_v1" "test" {
  metadata {
    name = kubernetes_secret_v1.test.metadata.0.name
  }
  binary_data = {
    raw = ""
  }
}
`
}

func testAccKubernetesDataSourceSecretV1Config_notFound(name string) string {
	return fmt.Sprintf(`data "kubernetes_secret_v1" "test" {
  metadata {
    name = %q
  }
}
`, name)
}

func testAccKubernetesDataSourceSecretV1Config_binaryData(name string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
  metadata {
    name = %q
  }
  data = {
    one = "first"
  }
  binary_data = {
    raw = "${base64encode("Raw data should come back as is in the pod")}"
  }
}

data "kubernetes_secret_v1" "test" {
  metadata {
    name = kubernetes_secret_v1.test.metadata.0.name
  }
  binary_data = {
    raw = ""
  }
}
`, name)
}

func testAccKubernetesDataSourceSecretV1Config_ignoreMetadata(name string) string {
	return fmt.Sprintf(`provider "kubernetes" {
  ignore_annotations = ["ignored-annotation"]
  ignore_labels      = ["ignored-label"]
}

resource "kubernetes_secret_v1" "test" {
  metadata {
    name = %q
    annotations = {
      "ignored-annotation" = "yes"
    }
    labels = {
      "ignored-label" = "yes"
    }
  }
  data = {
    key = "value"
  }
}

data "kubernetes_secret_v1" "test" {
  metadata {
    name = kubernetes_secret_v1.test.metadata.0.name
  }
}
`, name)
}
