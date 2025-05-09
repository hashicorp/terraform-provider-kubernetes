// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	corev1 "k8s.io/api/core/v1"
)

func TestAccKubernetesDataSourcePersistentVolumeClaimV1_basic(t *testing.T) {
	resourceName := "kubernetes_persistent_volume_claim_v1.test"
	dataSourceName := "data.kubernetes_persistent_volume_claim_v1.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{ // The first apply creates the resource. The second apply reads the resource using the data source.
				Config: testAccKubernetesDataSourcePersistentVolumeClaimV1_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.0", "ReadWriteOnce"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.0.requests.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.0.requests.storage", "1Gi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_mode", string(corev1.PersistentVolumeFilesystem)),
				),
			},
			{
				Config: testAccKubernetesDataSourcePersistentVolumeClaimV1_basic(name) +
					testAccKubernetesDataSourcePersistentVolumeClaimV1_read(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.access_modes.0", "ReadWriteOnce"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.resources.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.resources.0.requests.%", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.resources.0.requests.storage", "1Gi"),
					resource.TestCheckResourceAttr(dataSourceName, "spec.0.volume_mode", string(corev1.PersistentVolumeFilesystem)),
				),
			},
		},
	})
}

func TestAccKubernetesDataSourcePersistentVolumeClaimV1_not_found(t *testing.T) {
	dataSourceName := "data.kubernetes_persistent_volume_claim_v1.test"
	name := fmt.Sprintf("ceci-n.est-pas-une-pvc-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{ // The first apply creates the resource. The second apply reads the resource using the data source.
				Config: testAccKubernetesDataSourcePersistentVolumeClaimV1_nonexistent(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(dataSourceName, "spec.#", "0"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourcePersistentVolumeClaimV1_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume_claim_v1" "test" {
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
  spec {
    access_modes = ["ReadWriteOnce"]
    resources {
      requests = {
        storage = "1Gi"
      }
    }
    selector {
      match_expressions {
        key      = "environment"
        operator = "In"
        values   = ["non-exists-12345"]
      }
    }
  }
  wait_until_bound = false
}
`, name)
}

func testAccKubernetesDataSourcePersistentVolumeClaimV1_read() string {
	return `data "kubernetes_persistent_volume_claim_v1" "test" {
  metadata {
    name = "${kubernetes_persistent_volume_claim_v1.test.metadata.0.name}"
  }
}
`
}

func testAccKubernetesDataSourcePersistentVolumeClaimV1_nonexistent(name string) string {
	return fmt.Sprintf(`data "kubernetes_persistent_volume_claim_v1" "test" {
  metadata {
    name = "%s"
  }
}
`, name)
}
