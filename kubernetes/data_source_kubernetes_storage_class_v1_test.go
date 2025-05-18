// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccKubernetesDataSourceStorageClassV1_minikube(t *testing.T) {
	resourceName := "kubernetes_storage_class_v1.test"
	dataSourceName := "data.kubernetes_storage_class_v1.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotRunningInMinikube(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{ // The first apply creates the resource. The second apply reads the resource using a data source.
				Config: testAccKubernetesDataSourceStorageClassV1_basic(name, "k8s.io/minikube-hostpath"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(resourceName, "reclaim_policy", "Delete"),
					resource.TestCheckResourceAttr(resourceName, "storage_provisioner", "k8s.io/minikube-hostpath"),
					resource.TestCheckResourceAttr(resourceName, "allow_volume_expansion", "true"),
					resource.TestCheckResourceAttr(resourceName, "volume_binding_mode", "Immediate"),
					resource.TestCheckResourceAttr(resourceName, "mount_options.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "mount_options.1", "foo"),
					resource.TestCheckResourceAttr(resourceName, "mount_options.0", "bar"),
					resource.TestCheckResourceAttr(resourceName, "parameters.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "parameters.type", "pd-ssd"),
				),
			},
			{
				Config: testAccKubernetesDataSourceStorageClassV1_basic(name, "k8s.io/minikube-hostpath") +
					testAccKubernetesDataSourceStorageClassV1_read(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(dataSourceName, "reclaim_policy", "Delete"),
					resource.TestCheckResourceAttr(dataSourceName, "storage_provisioner", "k8s.io/minikube-hostpath"),
					resource.TestCheckResourceAttr(dataSourceName, "allow_volume_expansion", "true"),
					resource.TestCheckResourceAttr(resourceName, "volume_binding_mode", "Immediate"),
					resource.TestCheckResourceAttr(dataSourceName, "mount_options.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "mount_options.1", "foo"),
					resource.TestCheckResourceAttr(dataSourceName, "mount_options.0", "bar"),
					resource.TestCheckResourceAttr(dataSourceName, "parameters.%", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "parameters.type", "pd-ssd"),
				),
			},
		},
	})
}

func TestAccKubernetesDataSourceStorageClassV1_not_found(t *testing.T) {
	dataSourceName := "data.kubernetes_storage_class_v1.test"
	name := fmt.Sprintf("ceci-n.est-pas-une-storage-class-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotRunningInKind(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceStorageClassV1_nonexistent(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(dataSourceName, "mount_options.#", "0"),
					resource.TestCheckResourceAttr(dataSourceName, "parameters.%", "0"),
				),
			},
		},
	})
}

func TestAccKubernetesDataSourceStorageClassV1_gke(t *testing.T) {
	resourceName := "kubernetes_storage_class_v1.test"
	dataSourceName := "data.kubernetes_storage_class_v1.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotRunningInGke(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{ // The first apply creates the resource. The second apply reads the resource using a data source.
				Config: testAccKubernetesDataSourceStorageClassV1_basic(name, "kubernetes.io/gce-pd"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(resourceName, "reclaim_policy", "Delete"),
					resource.TestCheckResourceAttr(resourceName, "storage_provisioner", "kubernetes.io/gce-pd"),
					resource.TestCheckResourceAttr(resourceName, "allow_volume_expansion", "true"),
					resource.TestCheckResourceAttr(resourceName, "volume_binding_mode", "Immediate"),
					resource.TestCheckResourceAttr(resourceName, "mount_options.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "mount_options.1", "foo"),
					resource.TestCheckResourceAttr(resourceName, "mount_options.0", "bar"),
					resource.TestCheckResourceAttr(resourceName, "parameters.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "parameters.type", "pd-ssd"),
				),
			},
			{
				Config: testAccKubernetesDataSourceStorageClassV1_basic(name, "kubernetes.io/gce-pd") +
					testAccKubernetesDataSourceStorageClassV1_read(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(dataSourceName, "reclaim_policy", "Delete"),
					resource.TestCheckResourceAttr(dataSourceName, "storage_provisioner", "kubernetes.io/gce-pd"),
					resource.TestCheckResourceAttr(dataSourceName, "allow_volume_expansion", "true"),
					resource.TestCheckResourceAttr(resourceName, "volume_binding_mode", "Immediate"),
					resource.TestCheckResourceAttr(dataSourceName, "mount_options.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "mount_options.1", "foo"),
					resource.TestCheckResourceAttr(dataSourceName, "mount_options.0", "bar"),
					resource.TestCheckResourceAttr(dataSourceName, "parameters.%", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "parameters.type", "pd-ssd"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceStorageClassV1_basic(name, provisioner string) string {
	return fmt.Sprintf(`resource "kubernetes_storage_class_v1" "test" {
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
  storage_provisioner    = "%s"
  reclaim_policy         = "Delete"
  volume_binding_mode    = "Immediate"
  allow_volume_expansion = true
  mount_options          = ["foo", "bar"]
  parameters = {
    type = "pd-ssd"
  }
  allowed_topologies {
    match_label_expressions {
      key = "topology.kubernetes.io/zone"
      values = [
        "us-east-1a",
        "us-east-1b"
      ]
    }
  }
}
`, name, provisioner)
}

func testAccKubernetesDataSourceStorageClassV1_read() string {
	return `data "kubernetes_storage_class_v1" "test" {
  metadata {
    name = "${kubernetes_storage_class_v1.test.metadata.0.name}"
  }
}
`
}

func testAccKubernetesDataSourceStorageClassV1_nonexistent(name string) string {
	return fmt.Sprintf(`data "kubernetes_storage_class_v1" "test" {
  metadata {
    name = "%s"
  }
}
`, name)
}
