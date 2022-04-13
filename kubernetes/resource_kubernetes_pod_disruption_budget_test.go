package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	api "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesPodDisruptionBudget_basic(t *testing.T) {
	var conf api.PodDisruptionBudget
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_pod_disruption_budget.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDisruptionBudgetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodDisruptionBudgetConfig_maxUnavailable(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodDisruptionBudgetExists(resourceName, &conf),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.labels.TestLabelFour", "four"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_pod_disruption_budget.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_pod_disruption_budget.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_pod_disruption_budget.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "spec.0.max_unavailable", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "spec.0.min_available", ""),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "spec.0.selector.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "spec.0.selector.0.match_labels.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "spec.0.selector.0.match_labels.foo", "bar"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "spec.0.selector.0.match_expressions.#", "0"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccKubernetesPodDisruptionBudgetConfig_minAvailable(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodDisruptionBudgetExists(resourceName, &conf),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.labels.TestLabelFour", "four"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_pod_disruption_budget.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_pod_disruption_budget.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_pod_disruption_budget.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "spec.0.max_unavailable", ""),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "spec.0.min_available", "75%"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "spec.0.selector.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "spec.0.selector.0.match_labels.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "spec.0.selector.0.match_expressions.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "spec.0.selector.0.match_expressions.0.key", "name"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "spec.0.selector.0.match_expressions.0.operator", "In"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "spec.0.selector.0.match_expressions.0.values.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "spec.0.selector.0.match_expressions.0.values.1", "foo"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "spec.0.selector.0.match_expressions.0.values.0", "apps")),
			},
		},
	})
}

func testAccCheckKubernetesPodDisruptionBudgetDestroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_pod_disruption_budget" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.PolicyV1beta1().PodDisruptionBudgets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Namespace == namespace && resp.Name == name {
				return fmt.Errorf("Pod Disruption Budget still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesPodDisruptionBudgetExists(n string, obj *api.PodDisruptionBudget) resource.TestCheckFunc {
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

		out, err := conn.PolicyV1beta1().PodDisruptionBudgets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesPodDisruptionBudgetConfig_maxUnavailable(name string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_disruption_budget" "test" {
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

func testAccKubernetesPodDisruptionBudgetConfig_minAvailable(name string) string {
	// Note the percent sign in min_available is golang-escaped to be double percent signs
	return fmt.Sprintf(`resource "kubernetes_pod_disruption_budget" "test" {
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
