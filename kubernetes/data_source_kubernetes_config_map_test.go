package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccKubernetesDataSourceConfigMap_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceConfigMapConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubernetes_config_map.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("data.kubernetes_config_map.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("data.kubernetes_config_map.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("data.kubernetes_config_map.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("data.kubernetes_config_map.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("data.kubernetes_config_map.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("data.kubernetes_config_map.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("data.kubernetes_config_map.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr("data.kubernetes_config_map.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("data.kubernetes_config_map.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("data.kubernetes_config_map.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("data.kubernetes_config_map.test", "data.%", "2"),
					resource.TestCheckResourceAttr("data.kubernetes_config_map.test", "data.one", "first"),
					resource.TestCheckResourceAttr("data.kubernetes_config_map.test", "data.two", "second"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceConfigMapConfig_basic(name string) string {
	return testAccKubernetesConfigMapConfig_basic(name) + `
data "kubernetes_config_map" "test" {
	metadata {
		name = "${kubernetes_config_map.test.metadata.0.name}"
	}
}
`
}
