// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	nodev1 "k8s.io/api/node/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesRuntimeClassV1_basic(t *testing.T) {
	var conf nodev1.RuntimeClass
	rcName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_runtime_class_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesRuntimeClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesRuntimeClassV1Config_basic(rcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRuntimeClassV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", rcName),
					resource.TestCheckResourceAttr(resourceName, "handler", "myclass"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesRuntimeClassV1Config_annotations(rcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRuntimeClassV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", rcName),
					resource.TestCheckResourceAttr(resourceName, "handler", "newclass"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
				),
			},
			{
				Config: testAccKubernetesRuntimeClassV1Config_labels(rcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesRuntimeClassV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", rcName),
					resource.TestCheckResourceAttr(resourceName, "handler", "my-class"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
				),
			},
		},
	})
}

func testAccKubernetesRuntimeClassV1Config_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_runtime_class_v1" "test" {
  metadata {
    name = %q
  }
  handler = "myclass"
}
	`, name)
}

func testAccKubernetesRuntimeClassV1Config_annotations(name string) string {
	return fmt.Sprintf(`resource "kubernetes_runtime_class_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }
    name = %q
  }
  handler = "newclass"
}
`, name)
}

func testAccKubernetesRuntimeClassV1Config_labels(name string) string {
	return fmt.Sprintf(`resource "kubernetes_runtime_class_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }

    labels = {
      TestLabelOne = "one"
      TestLabelTwo = "two"
    }
    name = %q
  }
  handler = "my-class"
}
`, name)
}

func testAccCheckKubernetesRuntimeClassV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_runtime_class_v1" {
			continue
		}

		resp, err := conn.NodeV1().RuntimeClasses().Get(ctx, rs.Primary.ID, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Runtime Class still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesRuntimeClassV1Exists(n string, obj *nodev1.RuntimeClass) resource.TestCheckFunc {
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
