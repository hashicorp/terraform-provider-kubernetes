// Copyright IBM Corp. 2017, 2025
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"
	"testing"

	gversion "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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
					meta := testAccProvider.Meta()
					if meta == nil {
						return fmt.Errorf("Provider not initialized, unable to check cluster version")
					}
					conn, err := meta.(KubeClientsets).MainClientset()
					if err != nil {
						return err
					}
					ver, err := conn.ServerVersion()
					if err != nil {
						return err
					}
					gver, err := gversion.NewVersion(ver.String())
					if err != nil {
						return err
					}
					return resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr(dataSourceName, "version", gver.String()),
						resource.TestCheckResourceAttr(dataSourceName, "build_date", ver.BuildDate),
						resource.TestCheckResourceAttr(dataSourceName, "compiler", ver.Compiler),
						resource.TestCheckResourceAttr(dataSourceName, "git_commit", ver.GitCommit),
						resource.TestCheckResourceAttr(dataSourceName, "git_tree_state", ver.GitTreeState),
						resource.TestCheckResourceAttr(dataSourceName, "git_version", ver.GitVersion),
						resource.TestCheckResourceAttr(dataSourceName, "major", ver.Major),
						resource.TestCheckResourceAttr(dataSourceName, "minor", ver.Minor),
						resource.TestCheckResourceAttr(dataSourceName, "platform", ver.Platform),
						resource.TestCheckResourceAttr(dataSourceName, "go_version", ver.GoVersion),
					)(st)
				},
			},
		},
	})
}

func testAccKubernetesDataSourceServerVersionConfig_basic() string {
	return `data "kubernetes_server_version" "test" {}`
}
