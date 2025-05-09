// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	storage "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesCSIDriverV1_basic(t *testing.T) {
	var conf storage.CSIDriver
	resourceName := "kubernetes_csi_driver_v1.test"
	name := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.22.0")
		},
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesCSIDriverV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesCSIDriverV1BasicConfig(name, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesCSIDriverV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.attach_required", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_info_on_mount", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_lifecycle_modes.0", "Ephemeral"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
		},
	})
}

func testAccCheckKubernetesCSIDriverV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}

	ctx := context.TODO()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_csi_driver" {
			continue
		}
		name := rs.Primary.ID
		resp, err := conn.StorageV1().CSIDrivers().Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("CSIDriver still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesCSIDriverV1Exists(n string, obj *storage.CSIDriver) resource.TestCheckFunc {
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
		name := rs.Primary.ID
		out, err := conn.StorageV1().CSIDrivers().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesCSIDriverV1BasicConfig(name string, attached bool) string {
	return fmt.Sprintf(`resource "kubernetes_csi_driver_v1" "test" {
  metadata {
    name = %[1]q
  }

  spec {
    attach_required        = %[2]t
    pod_info_on_mount      = %[2]t
    volume_lifecycle_modes = ["Ephemeral"]
  }
}
`, name, attached)
}
