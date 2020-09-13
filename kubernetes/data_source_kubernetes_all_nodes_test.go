package kubernetes

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccKubernetesDataSourceAllNodes_basic(t *testing.T) {
	rxPosNum := regexp.MustCompile("^[1-9][0-9]*$")
	nsName := regexp.MustCompile("^[a-zA-Z][-\\w]*$")
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceAllNodesConfig_basic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("data.kubernetes_all_nodes.test", "nodes.#", rxPosNum),
					resource.TestCheckResourceAttrSet("data.kubernetes_all_nodes.test", "nodes.0"),
					resource.TestMatchResourceAttr("data.kubernetes_all_nodes.test", "nodes.0", nsName),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceAllNodesConfig_basic() string {
	return `
data "kubernetes_all_nodes" "test" {}
`
}
