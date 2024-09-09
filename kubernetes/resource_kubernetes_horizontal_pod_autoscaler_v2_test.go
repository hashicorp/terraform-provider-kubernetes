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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesHorizontalPodAutoscalerV2_minimal(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_horizontal_pod_autoscaler_v2.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.23.0")
		},
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesHorizontalPodAutoscalerV2Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesHorizontalPodAutoscalerV2Config_minimal(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesHorizontalPodAutoscalerV2Exists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.test", "test"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.test", "test"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.max_replicas", "10"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.min_replicas", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scale_target_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scale_target_ref.0.kind", "Deployment"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scale_target_ref.0.name", "TerraformAccTest"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.#", "1"),
				),
			},
		},
	})
}

func TestAccKubernetesHorizontalPodAutoscalerV2_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_horizontal_pod_autoscaler_v2.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.23.0")
		},
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesHorizontalPodAutoscalerV2Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesHorizontalPodAutoscalerV2Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesHorizontalPodAutoscalerV2Exists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.test", "test"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.test", "test"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_down.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_down.0.policy.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_down.0.policy.0.period_seconds", "120"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_down.0.policy.0.type", "Pods"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_down.0.policy.0.value", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_down.0.policy.1.period_seconds", "310"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_down.0.policy.1.type", "Percent"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_down.0.policy.1.value", "100"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_down.0.select_policy", "Min"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_down.0.stabilization_window_seconds", "300"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_up.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_up.0.policy.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_up.0.policy.0.period_seconds", "180"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_up.0.policy.0.type", "Percent"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_up.0.policy.0.value", "100"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_up.0.policy.1.period_seconds", "600"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_up.0.policy.1.type", "Pods"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_up.0.policy.1.value", "5"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_up.0.select_policy", "Max"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_up.0.stabilization_window_seconds", "600"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.max_replicas", "10"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.min_replicas", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scale_target_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scale_target_ref.0.kind", "Deployment"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scale_target_ref.0.name", "TerraformAccTest"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.#", "4"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.0.type", "Resource"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.0.resource.0.name", "test"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.0.resource.0.target.0.type", "Utilization"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.0.resource.0.target.0.average_utilization", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.1.type", "External"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.1.external.0.metric.0.name", "queue_size"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.1.external.0.metric.0.selector.0.match_labels.queue_name", "test"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.1.external.0.target.0.type", "Value"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.1.external.0.target.0.value", "10"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.2.type", "Pods"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.2.pods.0.metric.0.name", "packets-per-second"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.2.pods.0.target.0.type", "AverageValue"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.2.pods.0.target.0.average_value", "1k"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.3.type", "Object"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.3.object.0.metric.0.name", "requests-per-second"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.3.object.0.target.0.type", "AverageValue"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.3.object.0.target.0.average_value", "2k"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.3.object.0.described_object.0.kind", "Ingress"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.3.object.0.described_object.0.name", "main-route"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.3.object.0.described_object.0.api_version", "networking.k8s.io/v1beta1"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesHorizontalPodAutoscalerV2Config_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.test", "test"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.test", "test"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_down.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_down.0.policy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_down.0.policy.0.period_seconds", "120"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_down.0.policy.0.type", "Pods"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_down.0.policy.0.value", "10"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_down.0.select_policy", "Max"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_down.0.stabilization_window_seconds", "100"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_up.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_up.0.select_policy", "Disabled"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_up.0.policy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_up.0.policy.0.period_seconds", "60"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_up.0.policy.0.type", "Pods"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.behavior.0.scale_up.0.policy.0.value", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.max_replicas", "100"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.min_replicas", "50"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scale_target_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scale_target_ref.0.kind", "Deployment"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scale_target_ref.0.name", "TerraformAccTest"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.0.type", "External"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.0.external.0.metric.0.name", "latency"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.0.external.0.metric.0.selector.0.match_labels.lb_name", "test"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.0.external.0.target.0.type", "Value"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.0.external.0.target.0.value", "100"),
				),
			},
		},
	})
}

func TestAccKubernetesHorizontalPodAutoscalerV2_containerResource(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_horizontal_pod_autoscaler_v2.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.23.0")
		},
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesHorizontalPodAutoscalerV2Destroy,
		ErrorCheck: func(err error) error {
			t.Skipf("HPAContainerMetrics feature might not be enabled on the cluster and therefore this step will be skipped if an error occurs. Refer to the error for more details:\n%s", err)
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesHorizontalPodAutoscalerV2Config_containerResource(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesHorizontalPodAutoscalerV2Exists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.test", "test"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.test", "test"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.min_replicas", "50"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.max_replicas", "100"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scale_target_ref.0.api_version", "apps/v1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scale_target_ref.0.kind", "Deployment"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scale_target_ref.0.name", "TerraformAccTest"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.0.type", "ContainerResource"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.0.container_resource.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.0.container_resource.0.name", "cpu"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.0.container_resource.0.container", "test"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.0.container_resource.0.target.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.0.container_resource.0.target.0.type", "Utilization"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.metric.0.container_resource.0.target.0.average_utilization", "75"),
				),
			},
		},
	})
}

func testAccCheckKubernetesHorizontalPodAutoscalerV2Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_horizontal_pod_autoscaler_v2" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.AutoscalingV1().HorizontalPodAutoscalers(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Namespace == namespace && resp.Name == name {
				return fmt.Errorf("Horizontal Pod Autoscaler still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesHorizontalPodAutoscalerV2Exists(n string) resource.TestCheckFunc {
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

		_, err = conn.AutoscalingV2().HorizontalPodAutoscalers(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccKubernetesHorizontalPodAutoscalerV2Config_minimal(name string) string {
	return fmt.Sprintf(`resource "kubernetes_horizontal_pod_autoscaler_v2" "test" {
  metadata {
    name = %q

    annotations = {
      test = "test"
    }

    labels = {
      test = "test"
    }
  }

  spec {
    max_replicas = 10

    scale_target_ref {
      kind = "Deployment"
      name = "TerraformAccTest"
    }
  }
}
`, name)
}

func testAccKubernetesHorizontalPodAutoscalerV2Config_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_horizontal_pod_autoscaler_v2" "test" {
  metadata {
    name = %q

    annotations = {
      test = "test"
    }

    labels = {
      test = "test"
    }
  }

  spec {
    max_replicas = 10

    scale_target_ref {
      kind = "Deployment"
      name = "TerraformAccTest"
    }

    behavior {
      scale_down {
        stabilization_window_seconds = 300
        select_policy                = "Min"

        policy {
          period_seconds = 120
          type           = "Pods"
          value          = 1
        }

        policy {
          period_seconds = 310
          type           = "Percent"
          value          = 100
        }
      }

      scale_up {
        stabilization_window_seconds = 600
        select_policy                = "Max"

        policy {
          period_seconds = 180
          type           = "Percent"
          value          = 100
        }

        policy {
          period_seconds = 600
          type           = "Pods"
          value          = 5
        }
      }
    }

    metric {
      type = "Resource"
      resource {
        name = "test"
        target {
          type                = "Utilization"
          average_utilization = 1
        }
      }
    }

    metric {
      type = "External"
      external {
        metric {
          name = "queue_size"
          selector {
            match_labels = {
              queue_name = "test"
            }
          }
        }
        target {
          type  = "Value"
          value = "10"
        }
      }
    }

    metric {
      type = "Pods"
      pods {
        metric {
          name = "packets-per-second"
        }
        target {
          type          = "AverageValue"
          average_value = "1k"
        }
      }
    }

    metric {
      type = "Object"
      object {
        metric {
          name = "requests-per-second"
        }
        described_object {
          kind        = "Ingress"
          name        = "main-route"
          api_version = "networking.k8s.io/v1beta1"
        }
        target {
          type          = "AverageValue"
          average_value = "2k"
        }
      }
    }
  }
}
`, name)
}

func testAccKubernetesHorizontalPodAutoscalerV2Config_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_horizontal_pod_autoscaler_v2" "test" {
  metadata {
    name = %q

    annotations = {
      test = "test"
    }

    labels = {
      test = "test"
    }
  }

  spec {
    min_replicas = 50
    max_replicas = 100

    scale_target_ref {
      kind = "Deployment"
      name = "TerraformAccTest"
    }

    behavior {
      scale_down {
        stabilization_window_seconds = 100
        select_policy                = "Max"

        policy {
          period_seconds = 120
          type           = "Pods"
          value          = 10
        }
      }

      scale_up {
        select_policy = "Disabled"

        policy {
          period_seconds = 60
          type           = "Pods"
          value          = 1
        }
      }
    }

    metric {
      type = "External"
      external {
        metric {
          name = "latency"
          selector {
            match_labels = {
              lb_name = "test"
            }
          }
        }
        target {
          type  = "Value"
          value = "100"
        }
      }
    }
  }
}
`, name)
}

func testAccKubernetesHorizontalPodAutoscalerV2Config_containerResource(name string) string {
	return fmt.Sprintf(`resource "kubernetes_horizontal_pod_autoscaler_v2" "test" {
  metadata {
    name = %q

    annotations = {
      test = "test"
    }

    labels = {
      test = "test"
    }
  }

  spec {
    min_replicas = 50
    max_replicas = 100

    scale_target_ref {
      api_version = "apps/v1"
      kind        = "Deployment"
      name        = "TerraformAccTest"
    }

    metric {
      type = "ContainerResource"
      container_resource {
        name      = "cpu"
        container = "test"
        target {
          type                = "Utilization"
          average_utilization = "75"
        }
      }
    }
  }
}
`, name)
}
