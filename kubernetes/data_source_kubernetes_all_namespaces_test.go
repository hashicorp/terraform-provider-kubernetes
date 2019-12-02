package kubernetes

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccKubernetesDataSourceAllNamespaces_basic(t *testing.T) {
	rxPosNum := regexp.MustCompile("^[1-9][0-9]*$")
	nsName := regexp.MustCompile("^[a-zA-Z][-\\w]*$")
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceAllNamespacesConfig_basic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("data.kubernetes_all_namespaces.test", "namespaces.#", rxPosNum),
					resource.TestCheckResourceAttrSet("data.kubernetes_all_namespaces.test", "namespaces.0"),
					resource.TestMatchResourceAttr("data.kubernetes_all_namespaces.test", "namespaces.0", nsName),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceAllNamespacesConfig_basic() string {
	return `
data "kubernetes_all_namespaces" "test" {}
`
}
