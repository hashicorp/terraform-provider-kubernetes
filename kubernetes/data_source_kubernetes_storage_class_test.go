package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccKubernetesDataSourceStorageClass_minikube(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotRunningInMinikube(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{ // The first apply creates the resource. The second apply reads the resource using a data source.
				Config: testAccKubernetesDataSourceStorageClassConfig_basic(name, "k8s.io/minikube-hostpath"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_storage_class.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_storage_class.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_storage_class.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "reclaim_policy", "Delete"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "storage_provisioner", "k8s.io/minikube-hostpath"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "allow_volume_expansion", "true"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "volume_binding_mode", "Immediate"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "mount_options.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "mount_options.1", "foo"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "mount_options.0", "bar"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "parameters.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "parameters.type", "pd-ssd"),
				),
			},
			{
				Config: testAccKubernetesDataSourceStorageClassConfig_basic(name, "k8s.io/minikube-hostpath") +
					testAccKubernetesDataSourceStorageClassConfig_read(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("data.kubernetes_storage_class.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("data.kubernetes_storage_class.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("data.kubernetes_storage_class.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "reclaim_policy", "Delete"),
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "storage_provisioner", "k8s.io/minikube-hostpath"),
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "allow_volume_expansion", "true"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "volume_binding_mode", "Immediate"),
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "mount_options.#", "2"),
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "mount_options.1", "foo"),
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "mount_options.0", "bar"),
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "parameters.%", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "parameters.type", "pd-ssd"),
				),
			},
		},
	})
}

func TestAccKubernetesDataSourceStorageClass_gke(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotRunningInGke(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{ // The first apply creates the resource. The second apply reads the resource using a data source.
				Config: testAccKubernetesDataSourceStorageClassConfig_basic(name, "kubernetes.io/gce-pd"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_storage_class.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_storage_class.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_storage_class.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "reclaim_policy", "Delete"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "storage_provisioner", "kubernetes.io/gce-pd"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "allow_volume_expansion", "true"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "volume_binding_mode", "Immediate"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "mount_options.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "mount_options.1", "foo"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "mount_options.0", "bar"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "parameters.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "parameters.type", "pd-ssd"),
				),
			},
			{
				Config: testAccKubernetesDataSourceStorageClassConfig_basic(name, "kubernetes.io/gce-pd") +
					testAccKubernetesDataSourceStorageClassConfig_read(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("data.kubernetes_storage_class.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("data.kubernetes_storage_class.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("data.kubernetes_storage_class.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "reclaim_policy", "Delete"),
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "storage_provisioner", "kubernetes.io/gce-pd"),
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "allow_volume_expansion", "true"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "volume_binding_mode", "Immediate"),
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "mount_options.#", "2"),
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "mount_options.1", "foo"),
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "mount_options.0", "bar"),
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "parameters.%", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "parameters.type", "pd-ssd"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceStorageClassConfig_basic(name, provisioner string) string {
	return fmt.Sprintf(`resource "kubernetes_storage_class" "test" {
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

  mount_options = ["foo", "bar"]

  parameters = {
    type = "pd-ssd"
  }

 allowed_topologies {
    match_label_expressions {
      key = "failure-domain.beta.kubernetes.io/zone"
      values = [
        "us-east-1a",
        "us-east-1b"
      ]
    }
  }
}
`, name, provisioner)
}

func testAccKubernetesDataSourceStorageClassConfig_read() string {
	return `
data "kubernetes_storage_class" "test" {
  metadata {
    name = "${kubernetes_storage_class.test.metadata.0.name}"
  }
}
`
}
