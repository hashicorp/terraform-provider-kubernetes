// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	nodev1 "k8s.io/api/node/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesruntime_class_v1_basic(t *testing.T) {
	var conf nodev1.RuntimeClass
	rcName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_runtime_class_v1.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		//CheckDestroy:      testAccCheckKubernetesRuntimeClassDestroy,
		Steps: []resource.TestStep{
			//creating a run time class
			{
				Config: testAccKubernetesruntime_class_v1(rcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesruntime_classExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", rcName),
				),
			},
		},
	})
}

func testAccKubernetesruntime_class_v1(name string) string {
	return fmt.Sprintf(`resource "kubernetes_runtime_class_v1" "test" {
  metadata {
    name = %q
  }
  handler = "myclass"
}
	`, name)
}

func testAccCheckKubernetesruntime_classExists(n string, obj *nodev1.RuntimeClass) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
		if err != nil {
			return err
		}
		ctx := context.TODO()

		out, err := conn.NodeV1().RuntimeClasses().Get(ctx, rs.Primary.ID, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}
