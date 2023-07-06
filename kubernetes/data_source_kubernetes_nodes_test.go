// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	kuberesource "k8s.io/apimachinery/pkg/api/resource"
)

func checkParsableQuantity(value string) error {
	if _, err := kuberesource.ParseQuantity(value); err != nil {
		return err
	}
	return nil
}

func TestAccKubernetesDataSourceNodes_basic(t *testing.T) {
	nodeName := regexp.MustCompile(`^[a-z0-9]+(?:[-.]{1}[a-z0-9]+)*$`)
	zeroOrMore := regexp.MustCompile(`^[0-9]+$`)
	oneOrMore := regexp.MustCompile(`^[1-9][0-9]*$`)
	checkFuncs := resource.ComposeAggregateTestCheckFunc(
		resource.TestMatchResourceAttr("data.kubernetes_nodes.test", "nodes.#", oneOrMore),
		resource.TestMatchResourceAttr("data.kubernetes_nodes.test", "nodes.0.metadata.0.labels.%", zeroOrMore),
		resource.TestCheckResourceAttrSet("data.kubernetes_nodes.test", "nodes.0.metadata.0.resource_version"),
		resource.TestMatchResourceAttr("data.kubernetes_nodes.test", "nodes.0.metadata.0.name", nodeName),
		resource.TestMatchResourceAttr("data.kubernetes_nodes.test", "nodes.0.spec.0.%", oneOrMore),
		resource.TestCheckResourceAttrWith("data.kubernetes_nodes.test", "nodes.0.status.0.capacity.cpu", checkParsableQuantity),
		resource.TestCheckResourceAttrWith("data.kubernetes_nodes.test", "nodes.0.status.0.capacity.memory", checkParsableQuantity),
		resource.TestCheckResourceAttrSet("data.kubernetes_nodes.test", "nodes.0.status.0.node_info.0.architecture"),
		resource.TestCheckResourceAttrSet("data.kubernetes_nodes.test", "nodes.0.status.0.addresses.0.address"),
	)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceNodesConfig_basic(),
				Check:  checkFuncs,
			},
			{
				Config: testAccKubernetesDataSourceNodesConfig_labels(),
				Check:  checkFuncs,
			},
		},
	})
}

func testAccKubernetesDataSourceNodesConfig_basic() string {
	return `
data "kubernetes_nodes" "test" {}
`
}

func testAccKubernetesDataSourceNodesConfig_labels() string {
	return `
data "kubernetes_nodes" "test" {
  metadata {
    labels = {
      "kubernetes.io/os" = "linux"
    }
  }
}
`
}
