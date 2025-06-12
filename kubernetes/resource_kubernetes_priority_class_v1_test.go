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
	api "k8s.io/api/scheduling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesPriorityClassV1_basic(t *testing.T) {
	var conf api.PriorityClass
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_priority_class_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPriorityClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPriorityClassV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPriorityClassV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelFour", "four"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "value", "100"),
					resource.TestCheckResourceAttr(resourceName, "preemption_policy", "Never"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesPriorityClassV1Config_metaModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPriorityClassV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "value", "100"),
					resource.TestCheckResourceAttr(resourceName, "preemption_policy", "Never"),
				),
			},
			{
				Config: testAccKubernetesPriorityClassV1Config_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPriorityClassV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "value", "100"),
					resource.TestCheckResourceAttr(resourceName, "description", "Foobar"),
					resource.TestCheckResourceAttr(resourceName, "preemption_policy", "Never"),
				),
			},
		},
	})
}

func TestAccKubernetesPriorityClassV1_generatedName(t *testing.T) {
	var conf api.PriorityClass
	prefix := "tf-acc-test-"
	resourceName := "kubernetes_priority_class_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPriorityClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPriorityClassV1Config_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPriorityClassV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.generate_name", prefix),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "value", "999"),
				),
			},
		},
	})
}

func TestAccKubernetesPriorityClassV1_globalDefault(t *testing.T) {
	var conf api.PriorityClass
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_priority_class_v1.test"

	// This test has a global cluster effect and thus should be run sequentially before all parallel tests.
	// Otherwise, it may affect all Pod-related tests due to this setting: `global_default = true`.
	// The globalDefault field indicates that the value of this PriorityClass should be used for Pods without a priorityClassName.
	// More information: https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/#priorityclass
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPriorityClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPriorityClassV1Config_globalDefault(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPriorityClassV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "value", "100"),
					resource.TestCheckResourceAttr(resourceName, "description", "Foobar"),
					resource.TestCheckResourceAttr(resourceName, "global_default", "true"),
					resource.TestCheckResourceAttr(resourceName, "preemption_policy", "Never"),
				),
			},
		},
	})
}

func testAccCheckKubernetesPriorityClassV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_priority_class_v1" {
			continue
		}

		name := rs.Primary.ID

		resp, err := conn.SchedulingV1().PriorityClasses().Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == name {
				return fmt.Errorf("Resource Quota still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesPriorityClassV1Exists(n string, obj *api.PriorityClass) resource.TestCheckFunc {
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

		out, err := conn.SchedulingV1().PriorityClasses().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesPriorityClassV1Config_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_priority_class_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
      TestLabelFour  = "four"
    }

    name = "%s"
  }

  value             = 100
  preemption_policy = "Never"
}
`, name)
}

func testAccKubernetesPriorityClassV1Config_metaModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_priority_class_v1" "test" {
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

    name = "%s"
  }

  value             = 100
  preemption_policy = "Never"
}
`, name)
}

func testAccKubernetesPriorityClassV1Config_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_priority_class_v1" "test" {
  metadata {
    name = "%s"
  }

  value             = 100
  description       = "Foobar"
  preemption_policy = "Never"
}
`, name)
}

func testAccKubernetesPriorityClassV1Config_generatedName(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_priority_class_v1" "test" {
  metadata {
    generate_name = "%s"
  }

  value = 999
}
`, prefix)
}

func testAccKubernetesPriorityClassV1Config_globalDefault(name string) string {
	return fmt.Sprintf(`resource "kubernetes_priority_class_v1" "test" {
  metadata {
    name = "%s"
  }

  value             = 100
  description       = "Foobar"
  preemption_policy = "Never"
  global_default    = true
}
`, name)
}
