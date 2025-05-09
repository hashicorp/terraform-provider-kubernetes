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

	policy "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesPodDisruptionBudgetV1_basic(t *testing.T) {
	var conf policy.PodDisruptionBudget
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_pod_disruption_budget_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDisruptionBudgetV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodDisruptionBudgetV1Config_maxUnavailable(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodDisruptionBudgetV1Exists(resourceName, &conf),
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
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.max_unavailable", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.min_available", ""),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.0.match_labels.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.0.match_labels.foo", "bar"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.0.match_expressions.#", "0"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesPodDisruptionBudgetV1Config_minAvailable(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodDisruptionBudgetV1Exists(resourceName, &conf),
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
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.max_unavailable", ""),
					resource.TestCheckResourceAttr(resourceName, "spec.0.min_available", "75%"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.0.match_labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.0.match_expressions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.0.match_expressions.0.key", "name"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.0.match_expressions.0.operator", "In"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.0.match_expressions.0.values.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.0.match_expressions.0.values.1", "foo"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.0.match_expressions.0.values.0", "apps")),
			},
		},
	})
}

func testAccCheckKubernetesPodDisruptionBudgetV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_pod_disruption_budget_v1" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.PolicyV1().PodDisruptionBudgets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Namespace == namespace && resp.Name == name {
				return fmt.Errorf("Pod Disruption Budget still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesPodDisruptionBudgetV1Exists(n string, obj *policy.PodDisruptionBudget) resource.TestCheckFunc {
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

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		out, err := conn.PolicyV1().PodDisruptionBudgets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesPodDisruptionBudgetV1Config_maxUnavailable(name string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_disruption_budget_v1" "test" {
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

  spec {
    max_unavailable = 1
    selector {
      match_labels = {
        foo = "bar"
      }
    }
  }
}
`, name)
}

func testAccKubernetesPodDisruptionBudgetV1Config_minAvailable(name string) string {
	// Note the percent sign in min_available is golang-escaped to be double percent signs
	return fmt.Sprintf(`resource "kubernetes_pod_disruption_budget_v1" "test" {
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

  spec {
    min_available = "75%%"
    selector {
      match_expressions {
        key      = "name"
        operator = "In"
        values   = ["foo", "apps"]
      }
    }
  }
}
`, name)
}
