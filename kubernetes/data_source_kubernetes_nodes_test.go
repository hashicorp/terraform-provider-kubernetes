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
	dataSourceName := "data.kubernetes_nodes.test"
	nodeName := regexp.MustCompile(`^[a-z0-9]+(?:[-.]{1}[a-z0-9]+)*$`)
	oneOrMore := regexp.MustCompile(`^[1-9][0-9]*$`)
	checkFuncs := resource.ComposeAggregateTestCheckFunc(
		resource.TestMatchResourceAttr(dataSourceName, "nodes.#", oneOrMore),
		resource.TestMatchResourceAttr(dataSourceName, "nodes.0.metadata.0.annotations.%", oneOrMore),
		resource.TestMatchResourceAttr(dataSourceName, "nodes.0.metadata.0.labels.%", oneOrMore),
		resource.TestCheckResourceAttrSet(dataSourceName, "nodes.0.metadata.0.resource_version"),
		resource.TestMatchResourceAttr(dataSourceName, "nodes.0.metadata.0.name", nodeName),
		resource.TestMatchResourceAttr(dataSourceName, "nodes.0.spec.0.%", oneOrMore),
		resource.TestCheckResourceAttrWith(dataSourceName, "nodes.0.status.0.capacity.cpu", checkParsableQuantity),
		resource.TestCheckResourceAttrWith(dataSourceName, "nodes.0.status.0.capacity.memory", checkParsableQuantity),
		resource.TestCheckResourceAttrSet(dataSourceName, "nodes.0.status.0.node_info.0.architecture"),
		resource.TestCheckResourceAttrSet(dataSourceName, "nodes.0.status.0.addresses.0.address"),
		resource.TestMatchResourceAttr(dataSourceName, "nodes.0.status.0.conditions.#", oneOrMore),
		resource.TestCheckResourceAttrSet(dataSourceName, "nodes.0.status.0.conditions.0.type"),
		resource.TestCheckResourceAttrSet(dataSourceName, "nodes.0.status.0.conditions.0.status"),
		resource.TestCheckResourceAttrSet(dataSourceName, "nodes.0.status.0.conditions.0.last_heartbeat_time"),
		resource.TestCheckResourceAttrSet(dataSourceName, "nodes.0.status.0.conditions.0.last_transition_time"),
		resource.TestCheckResourceAttrSet(dataSourceName, "nodes.0.status.0.conditions.0.reason"),
		resource.TestCheckResourceAttrSet(dataSourceName, "nodes.0.status.0.conditions.0.message"),
	)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceNodes_basic(),
				Check:  checkFuncs,
			},
			{
				Config: testAccKubernetesDataSourceNodes_labels(),
				Check:  checkFuncs,
			},
		},
	})
}

func TestAccKubernetesDataSourceNodes_not_found(t *testing.T) {
	dataSourceName := "data.kubernetes_nodes.test"
	checkFuncs := resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttr(dataSourceName, "nodes.#", "0"),
	)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceNodes_nonexistent(),
				Check:  checkFuncs,
			},
		},
	})
}

func testAccKubernetesDataSourceNodes_basic() string {
	return `data "kubernetes_nodes" "test" {}
`
}

func testAccKubernetesDataSourceNodes_labels() string {
	return `data "kubernetes_nodes" "test" {
  metadata {
    labels = {
      "kubernetes.io/os" = "linux"
    }
  }
}
`
}

func testAccKubernetesDataSourceNodes_nonexistent() string {
	return `data "kubernetes_nodes" "test" {
  metadata {
    labels = {
      "ceci-n.est-pas" = "une-node"
    }
  }
}
`
}
