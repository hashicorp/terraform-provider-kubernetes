package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesHorizontalPodAutoscalerV2_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_horizontal_pod_autoscaler.test"

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_horizontal_pod_autoscaler.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesHorizontalPodAutoscalerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesHorizontalPodAutoscalerV2Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesHorizontalPodAutoscalerV2Exists(resourceName),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "metadata.0.annotations.test", "test"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "metadata.0.labels.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "metadata.0.labels.test", "test"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_horizontal_pod_autoscaler.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_horizontal_pod_autoscaler.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_horizontal_pod_autoscaler.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_horizontal_pod_autoscaler.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.max_replicas", "10"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.min_replicas", "1"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.scale_target_ref.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.scale_target_ref.0.kind", "Deployment"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.scale_target_ref.0.name", "TerraformAccTest"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.metric.#", "4"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.metric.0.type", "Resource"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.metric.0.resource.0.name", "test"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.metric.0.resource.0.target.0.type", "Utilization"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.metric.0.resource.0.target.0.average_utilization", "1"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.metric.1.type", "External"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.metric.1.external.0.metric.0.name", "queue_size"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.metric.1.external.0.metric.0.selector.0.match_labels.queue_name", "test"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.metric.1.external.0.target.0.type", "Value"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.metric.1.external.0.target.0.value", "10"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.metric.2.type", "Pods"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.metric.2.pods.0.metric.0.name", "packets-per-second"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.metric.2.pods.0.target.0.type", "AverageValue"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.metric.2.pods.0.target.0.average_value", "1k"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.metric.3.type", "Object"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.metric.3.object.0.metric.0.name", "requests-per-second"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.metric.3.object.0.target.0.type", "AverageValue"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.metric.3.object.0.target.0.average_value", "2k"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.metric.3.object.0.described_object.0.kind", "Ingress"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.metric.3.object.0.described_object.0.name", "main-route"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.metric.3.object.0.described_object.0.api_version", "networking.k8s.io/v1beta1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccKubernetesHorizontalPodAutoscalerV2Config_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "metadata.0.annotations.test", "test"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "metadata.0.labels.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "metadata.0.labels.test", "test"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.max_replicas", "100"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.min_replicas", "50"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.scale_target_ref.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.scale_target_ref.0.kind", "Deployment"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.scale_target_ref.0.name", "TerraformAccTest"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.metric.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.metric.0.type", "External"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.metric.0.external.0.metric.0.name", "latency"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.metric.0.external.0.metric.0.selector.0.match_labels.lb_name", "test"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.metric.0.external.0.target.0.type", "Value"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler.test", "spec.0.metric.0.external.0.target.0.value", "100"),
				),
			},
		},
	})
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

		_, err = conn.AutoscalingV2beta2().HorizontalPodAutoscalers(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccKubernetesHorizontalPodAutoscalerV2Config_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_horizontal_pod_autoscaler" "test" {
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
	return fmt.Sprintf(`resource "kubernetes_horizontal_pod_autoscaler" "test" {
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
