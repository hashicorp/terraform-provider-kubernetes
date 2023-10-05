package kubernetes

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccKubernetesDataSourceServerVersion_basic(t *testing.T) {
	dataSourceName := "data.kubernetes_server_version.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceServerVersionConfig_basic(),
				Check: func(st *terraform.State) error {
					v, err := getClusterVersion()
					if err != nil {
						t.Fail()
					}
					return resource.TestCheckResourceAttr(dataSourceName, "version", v.String())(st)
				},
			},
		},
	})
}

func testAccKubernetesDataSourceServerVersionConfig_basic() string {
	return `data "kubernetes_server_version" "test" {}`
}
