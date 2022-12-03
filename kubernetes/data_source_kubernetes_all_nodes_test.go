package kubernetes

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccKubernetesDataSourceAllNodes_basic(t *testing.T) {
	rxPosNum := regexp.MustCompile("^[1-9][0-9]*$")
	nodeName := regexp.MustCompile(`^([a-z0-9]+(-[a-z0-9]+)*\.)+[a-z]{2,}$`)
	nodeLen := regexp.MustCompile(`^.{2,63}$`)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceAllNodesConfig_basic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("data.kubernetes_all_nodes.test", "nodes.#", rxPosNum),
					resource.TestCheckResourceAttrSet("data.kubernetes_all_nodes.test", "nodes.0"),
					resource.TestMatchResourceAttr("data.kubernetes_all_nodes.test", "nodes.0", nodeName),
					resource.TestMatchResourceAttr("data.kubernetes_all_nodes.test", "nodes.0", nodeLen),
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
