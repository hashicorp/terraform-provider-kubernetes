// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	fieldManager = "tf-taint-test"
	taintKey     = "node-role.kubernetes.io/test"
	taintValue   = "true"
	taintEffect  = "PreferNoSchedule"
)

func TestAccKubernetesResourceNodeTaint_basic(t *testing.T) {
	resourceName := "kubernetes_node_taint.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccKubernetesNodeTaintDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNodeTaintConfig_basic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccKubernetesNodeTaintExists(resourceName),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.name"),
					resource.TestCheckResourceAttr(resourceName, "taint.0.key", taintKey),
					resource.TestCheckResourceAttr(resourceName, "taint.0.value", taintValue),
					resource.TestCheckResourceAttr(resourceName, "taint.0.effect", taintEffect),
					resource.TestCheckResourceAttr(resourceName, "field_manager", fieldManager),
				),
			},
		},
	})
}

func testAccKubernetesCheckNodeTaint(rs *terraform.ResourceState) error {
	nodeName, taint, err := idToNodeTaint(rs.Primary.ID)
	if err != nil {
		return fmt.Errorf("failed to parse id: %s", rs.Primary.ID)
	}

	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.Background()

	node, err := conn.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if !hasTaint(node.Spec.Taints, taint) {
		return fmt.Errorf("node %s does not have taint %+v", nodeName, taint)
	}
	return nil
}

func testAccKubernetesNodeTaintDestroy(s *terraform.State) error {
	rsType := "kubernetes_node_taint"
	for _, rs := range s.RootModule().Resources {
		if rs.Type != rsType {
			continue
		}
		if err := testAccKubernetesCheckNodeTaint(rs); err == nil {
			return fmt.Errorf("taint was not removed from node")
		}
		return nil
	}
	return fmt.Errorf("unable to find %s in state file", rsType)
}

func testAccKubernetesNodeTaintExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not in state file: %s", n)
		}
		return testAccKubernetesCheckNodeTaint(rs)
	}
}

func testAccKubernetesNodeTaintConfig_basic() string {
	return fmt.Sprintf(`
data "kubernetes_nodes" "test" {}

resource "kubernetes_node_taint" "test" {
  metadata {
    name = data.kubernetes_nodes.test.nodes.0.metadata.0.name
  }
  taint {
    key    = "%s"
    value  = "%s"
    effect = "%s"
  }
  field_manager = "%s"
}
`, taintKey, taintValue, taintEffect, fieldManager)
}
