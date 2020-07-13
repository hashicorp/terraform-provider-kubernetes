package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccKubernetesDataSourceStorageClass_basic(t *testing.T) {
	dataSourceName := "data.kubernetes_storage_class.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceStorageClassConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.self_link"),
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
					resource.TestCheckResourceAttr(dataSourceName, "mount_options.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "mount_options.2356372769", "foo"),
					resource.TestCheckResourceAttr(dataSourceName, "mount_options.1996459178", "bar"),
					resource.TestCheckResourceAttr(dataSourceName, "parameters.%", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "parameters.type", "pd-ssd"),
					resource.TestCheckResourceAttr(dataSourceName, "allowed_topologies.%", "0"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceStorageClassConfig_basic(name string) string {
	return testAccKubernetesStorageClassConfig_basic(name) + `
data "kubernetes_storage_class" "test" {
	metadata {
		name = "${kubernetes_storage_class.test.metadata.0.name}"
	}
}
`
}
