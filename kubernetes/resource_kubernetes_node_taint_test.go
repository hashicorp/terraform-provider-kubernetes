// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/util/taints"
)

const (
	fieldManager = "tf-taint-test"
	taintKey     = "node-role.kubernetes.io/test"
	taintValue   = "true"
	taintEffect  = "PreferNoSchedule"
)

//Due to the nature of this resource it will not be modified to run in parallel

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

func TestAccKubernetesResourceNodeTaint_MultipleBasic(t *testing.T) {
	resourceName := "kubernetes_node_taint.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccKubernetesNodeTaintDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNodeTaintConfig_multipleBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccKubernetesNodeTaintExists(resourceName),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.name"),
					resource.TestCheckResourceAttr(resourceName, "taint.0.key", taintKey+"-1"),
					resource.TestCheckResourceAttr(resourceName, "taint.0.value", taintValue),
					resource.TestCheckResourceAttr(resourceName, "taint.0.effect", taintEffect),
					resource.TestCheckResourceAttr(resourceName, "taint.1.key", taintKey+"-2"),
					resource.TestCheckResourceAttr(resourceName, "taint.1.value", taintValue),
					resource.TestCheckResourceAttr(resourceName, "taint.1.effect", taintEffect),
					resource.TestCheckResourceAttr(resourceName, "taint.2.key", taintKey+"-3"),
					resource.TestCheckResourceAttr(resourceName, "taint.2.value", taintValue),
					resource.TestCheckResourceAttr(resourceName, "taint.2.effect", taintEffect),
				),
			},
			{
				Config: testAccKubernetesNodeTaintConfig_updateTaint(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccKubernetesNodeTaintExists(resourceName),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.name"),
					resource.TestCheckResourceAttr(resourceName, "taint.0.key", taintKey+"-1"),
					resource.TestCheckResourceAttr(resourceName, "taint.0.value", taintValue),
					resource.TestCheckResourceAttr(resourceName, "taint.0.effect", "NoSchedule"),
					resource.TestCheckResourceAttr(resourceName, "taint.1.key", taintKey+"-2"),
					resource.TestCheckResourceAttr(resourceName, "taint.1.value", taintValue),
					resource.TestCheckResourceAttr(resourceName, "taint.1.effect", "NoSchedule"),
					resource.TestCheckResourceAttr(resourceName, "taint.2.key", taintKey+"-3"),
					resource.TestCheckResourceAttr(resourceName, "taint.2.value", taintValue),
					resource.TestCheckResourceAttr(resourceName, "taint.2.effect", taintEffect),
				),
			},
			{
				Config: testAccKubernetesNodeTaintConfig_removeTaint(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccKubernetesNodeTaintExists(resourceName),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.name"),
					resource.TestCheckResourceAttr(resourceName, "taint.0.key", taintKey+"-1"),
					resource.TestCheckResourceAttr(resourceName, "taint.0.value", taintValue),
					resource.TestCheckResourceAttr(resourceName, "taint.0.effect", "NoSchedule"),
					resource.TestCheckResourceAttr(resourceName, "taint.1.key", taintKey+"-2"),
					resource.TestCheckResourceAttr(resourceName, "taint.1.value", taintValue),
					resource.TestCheckResourceAttr(resourceName, "taint.1.effect", "NoSchedule"),
					resource.TestCheckResourceAttr(resourceName, "taint.#", "2"),
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
	return fmt.Sprintf(`data "kubernetes_nodes" "test" {}

resource "kubernetes_node_taint" "test" {
  metadata {
    name = data.kubernetes_nodes.test.nodes.0.metadata.0.name
  }
  taint {
    key    = %q
    value  = %q
    effect = %q
  }
  field_manager = %q
}
`, taintKey, taintValue, taintEffect, fieldManager)
}

func testAccKubernetesNodeTaintConfig_multipleBasic() string {
	return fmt.Sprintf(`data "kubernetes_nodes" "test" {}

resource "kubernetes_node_taint" "test" {
  metadata {
    name = data.kubernetes_nodes.test.nodes.0.metadata.0.name
  }
  taint {
    key    = %q
    value  = %q
    effect = %q
  }
  taint {
    key    = %q
    value  = %q
    effect = %q
  }
  taint {
    key    = %q
    value  = %q
    effect = %q
  }
  field_manager = %q
}
`, taintKey+"-1", taintValue, taintEffect, taintKey+"-2", taintValue, taintEffect, taintKey+"-3", taintValue, taintEffect, fieldManager)
}

func testAccKubernetesNodeTaintConfig_updateTaint() string {
	return fmt.Sprintf(`data "kubernetes_nodes" "test" {}

resource "kubernetes_node_taint" "test" {
  metadata {
    name = data.kubernetes_nodes.test.nodes.0.metadata.0.name
  }
  taint {
    key    = %q
    value  = %q
    effect = %q
  }
  taint {
    key    = %q
    value  = %q
    effect = %q
  }
  taint {
    key    = %q
    value  = %q
    effect = %q
  }
  field_manager = %q
}
`, taintKey+"-1", taintValue, "NoSchedule", taintKey+"-2", taintValue, "NoSchedule", taintKey+"-3", taintValue, taintEffect, fieldManager)
}

func testAccKubernetesNodeTaintConfig_removeTaint() string {
	return fmt.Sprintf(`data "kubernetes_nodes" "test" {}

resource "kubernetes_node_taint" "test" {
  metadata {
    name = data.kubernetes_nodes.test.nodes.0.metadata.0.name
  }
  taint {
    key    = %q
    value  = %q
    effect = %q
  }
  taint {
    key    = %q
    value  = %q
    effect = %q
  }
  field_manager = %q
}
`, taintKey+"-1", taintValue, "NoSchedule", taintKey+"-2", taintValue, "NoSchedule", fieldManager)
}

func hasTaint(taints []v1.Taint, taint *v1.Taint) bool {
	for i := range taints {
		if taint.MatchTaint(&taints[i]) {
			return true
		}
	}
	return false
}

func idToNodeTaint(id string) (string, *v1.Taint, error) {
	idVals := strings.Split(id, ",")
	nodeName := idVals[0]
	taintStr := idVals[1]
	taints, _, err := taints.ParseTaints([]string{taintStr})
	if err != nil {
		return "", nil, err
	}
	if len(taints) == 0 {
		return "", nil, fmt.Errorf("failed to parse taint %s", taintStr)
	}
	return nodeName, &taints[0], nil
}
