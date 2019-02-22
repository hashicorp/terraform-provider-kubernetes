package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	api "k8s.io/api/policy/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetes "k8s.io/client-go/kubernetes"
)

func TestAccKubernetesPodDisruptionBudget_basic(t *testing.T) {
	var conf api.PodDisruptionBudget
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_pod_disruption_budget.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesPodDisruptionBudgetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodDisruptionBudgetConfig_maxUnavailable(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodDisruptionBudgetExists("kubernetes_pod_disruption_budget.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one"}),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.labels.TestLabelFour", "four"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelThree": "three", "TestLabelFour": "four"}),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_pod_disruption_budget.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_pod_disruption_budget.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_pod_disruption_budget.test", "metadata.0.self_link"),
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
				Config: testAccKubernetesPodDisruptionBudgetConfig_minAvailable(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodDisruptionBudgetExists("kubernetes_pod_disruption_budget.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one"}),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.labels.TestLabelFour", "four"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelThree": "three", "TestLabelFour": "four"}),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_pod_disruption_budget.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_pod_disruption_budget.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_pod_disruption_budget.test", "metadata.0.self_link"),
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
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "spec.0.selector.0.match_expressions.0.values.2356372769", "foo"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "spec.0.selector.0.match_expressions.0.values.270302810", "apps")),
			},
			{
				Config: testAccKubernetesPodDisruptionBudgetConfig_noSelector(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodDisruptionBudgetExists("kubernetes_pod_disruption_budget.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_pod_disruption_budget.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_pod_disruption_budget.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_pod_disruption_budget.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_pod_disruption_budget.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "spec.0.max_unavailable", "10%"),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "spec.0.min_available", ""),
					resource.TestCheckResourceAttr("kubernetes_pod_disruption_budget.test", "spec.0.selector.#", "0"),
				),
			},
		},
	})
}

func TestAccKubernetesPodDisruptionBudget_importBasic(t *testing.T) {
	resourceName := "kubernetes_pod_disruption_budget.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesPodDisruptionBudgetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodDisruptionBudgetConfig_minAvailable(name),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckKubernetesPodDisruptionBudgetDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*kubernetes.Clientset)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_pod_disruption_budget" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.PolicyV1beta1().PodDisruptionBudgets(namespace).Get(name, meta_v1.GetOptions{})
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

		conn := testAccProvider.Meta().(*kubernetes.Clientset)

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		out, err := conn.PolicyV1beta1().PodDisruptionBudgets(namespace).Get(name, meta_v1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesPodDisruptionBudgetConfig_maxUnavailable(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_pod_disruption_budget" "test" {
  metadata {
    annotations {
      TestAnnotationOne = "one"
    }

    labels {
      TestLabelOne   = "one"
      TestLabelThree = "three"
      TestLabelFour  = "four"
    }

    name = "%s"
  }

  spec {
    max_unavailable = 1
    selector {
      match_labels {
        foo = "bar"
      }
    }
  }
}
`, name)
}

func testAccKubernetesPodDisruptionBudgetConfig_minAvailable(name string) string {
	// Note the percent sign in min_available is golang-escaped to be double percent signs
	return fmt.Sprintf(`
resource "kubernetes_pod_disruption_budget" "test" {
  metadata {
    annotations {
      TestAnnotationOne = "one"
    }

    labels {
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
        key = "name"
        operator = "In"
        values = ["foo", "apps"]
      }
    }
  }
}
`, name)
}

func testAccKubernetesPodDisruptionBudgetConfig_noSelector(name string) string {
	// Note the percent sign in max_unavailable is golang-escaped to be double percent signs
	return fmt.Sprintf(`
resource "kubernetes_pod_disruption_budget" "test" {
  metadata {
    name = "%s"
  }

  spec {
    max_unavailable = "10%%"
  }
}
`, name)
}
