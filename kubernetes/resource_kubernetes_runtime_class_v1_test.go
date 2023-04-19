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
				Config: testAccKubernetesruntime_class_v1_basic(rcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesruntime_class_v1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", rcName),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesruntime_class_v1_addAnnotations(rcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesruntime_class_v1Exists("kubernetes_runtime_class_v1.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_runtime_class_v1.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_runtime_class_v1.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_runtime_class_v1.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_runtime_class_v1.test", "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_runtime_class_v1.test", "metadata.0.name", rcName),
					resource.TestCheckResourceAttrSet("kubernetes_runtime_class_v1.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kkubernetes_runtime_class_v1.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_runtime_class_v1.test", "metadata.0.uid"),
				),
			},
			{
				Config: testAccKubernetesruntime_class_v1_addLabels(rcName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesruntime_class_v1Exists("kubernetes_runtime_class_v1.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_runtime_class_v1.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_runtime_class_v1.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_runtime_class_v1.test", "metadata.0.annotations.Different", "1234"),
					resource.TestCheckResourceAttr("kubernetes_runtime_class_v1.test", "metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_runtime_class_v1.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_runtime_class_v1.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("kubernetes_runtime_class_v1.test", "metadata.0.name", rcName),
					resource.TestCheckResourceAttrSet("kubernetes_runtime_class_v1.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_runtime_class_v1.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_runtime_class_v1.test", "metadata.0.uid"),
				),
			},
		},
	})
}

func testAccKubernetesruntime_class_v1_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_runtime_class_v1" "test" {
  metadata {
    name = %q
  }
  handler = "myclass"
}
	`, name)
}

func testAccKubernetesruntime_class_v1_addAnnotations(name string) string {
	return fmt.Sprintf(`resource "kubernetes_runtime_class_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }
    name = %q
  }
}
`, name)
}

func testAccKubernetesruntime_class_v1_addLabels(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    name = %q
  }
}
`, name)
}

func testAccCheckKubernetesruntime_class_v1Exists(n string, obj *nodev1.RuntimeClass) resource.TestCheckFunc {
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
