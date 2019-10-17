package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccKubernetesDataSourceStorageClass_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceStorageClassConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("data.kubernetes_storage_class.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("data.kubernetes_storage_class.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("data.kubernetes_storage_class.test", "metadata.0.self_link"),
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
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "parameters.%", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_storage_class.test", "parameters.type", "pd-ssd"),
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
