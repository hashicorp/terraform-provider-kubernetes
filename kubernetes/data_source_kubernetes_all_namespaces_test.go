// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccKubernetesDataSourceAllNamespaces_basic(t *testing.T) {
	dataSourceName := "data.kubernetes_all_namespaces.test"
	rxPosNum := regexp.MustCompile("^[1-9][0-9]*$")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceAllNamespacesConfig_basic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(dataSourceName, "namespaces.#", rxPosNum),
					resource.TestCheckTypeSetElemAttr(dataSourceName, "namespaces.*", "default"),
					resource.TestCheckTypeSetElemAttr(dataSourceName, "namespaces.*", "kube-system"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceAllNamespacesConfig_basic() string {
	return `data "kubernetes_all_namespaces" "test" {}`
}
