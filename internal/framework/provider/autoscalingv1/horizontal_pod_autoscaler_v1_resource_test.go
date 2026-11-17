// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package autoscalingv1_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

const hpaV1ResourceName = "kubernetes_horizontal_pod_autoscaler_v1.test"

func TestAccKubernetesHorizontalPodAutoscalerV1_basic(t *testing.T) {
	var uid string
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccHorizontalPodAutoscalerCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccHorizontalPodAutoscalerV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccHorizontalPodAutoscalerCheckExists(hpaV1ResourceName, &uid),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "metadata.0.labels.TestLabelFour", "four"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(hpaV1ResourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(hpaV1ResourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(hpaV1ResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "spec.0.max_replicas", "10"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "spec.0.min_replicas", "1"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "spec.0.scale_target_ref.#", "1"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "spec.0.scale_target_ref.0.kind", "ReplicationController"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "spec.0.scale_target_ref.0.name", "TerraformAccTest"),
				),
			},
			{
				ResourceName:            hpaV1ResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccHorizontalPodAutoscalerV1Config_metaModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccHorizontalPodAutoscalerCheckExists(hpaV1ResourceName, &uid),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(hpaV1ResourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(hpaV1ResourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(hpaV1ResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "spec.0.max_replicas", "10"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "spec.0.min_replicas", "1"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "spec.0.scale_target_ref.#", "1"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "spec.0.scale_target_ref.0.kind", "ReplicationController"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "spec.0.scale_target_ref.0.name", "TerraformAccTest"),
				),
			},
			{
				Config: testAccHorizontalPodAutoscalerV1Config_specModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccHorizontalPodAutoscalerCheckExists(hpaV1ResourceName, &uid),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(hpaV1ResourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(hpaV1ResourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(hpaV1ResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "spec.0.max_replicas", "8"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "spec.0.min_replicas", "1"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "spec.0.scale_target_ref.#", "1"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "spec.0.scale_target_ref.0.kind", "ReplicationController"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "spec.0.scale_target_ref.0.name", "TerraformAccTestModified"),
				),
			},
		},
	})
}

func TestAccKubernetesHorizontalPodAutoscalerV1_generatedName(t *testing.T) {
	prefix := "tf-acc-test-"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccHorizontalPodAutoscalerCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccHorizontalPodAutoscalerV1Config_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccHorizontalPodAutoscalerCheckExists(hpaV1ResourceName, nil),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "metadata.0.generate_name", prefix),
					resource.TestCheckResourceAttrSet(hpaV1ResourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(hpaV1ResourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(hpaV1ResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "spec.0.max_replicas", "1"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "spec.0.min_replicas", "1"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "spec.0.scale_target_ref.#", "1"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "spec.0.scale_target_ref.0.kind", "ReplicationController"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "spec.0.scale_target_ref.0.name", "TerraformAccTestGeneratedName"),
				),
			},
		},
	})
}

func TestAccKubernetesHorizontalPodAutoscalerV1_targetCPU(t *testing.T) {
	var uid string
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccHorizontalPodAutoscalerCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccHorizontalPodAutoscalerV1Config_targetCPU(name, 2, 50),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccHorizontalPodAutoscalerCheckExists(hpaV1ResourceName, &uid),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "spec.0.min_replicas", "2"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "spec.0.target_cpu_utilization_percentage", "50"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "spec.0.scale_target_ref.0.api_version", "apps/v1"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "spec.0.scale_target_ref.0.kind", "Deployment"),
				),
			},
			{
				ResourceName:            hpaV1ResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccHorizontalPodAutoscalerV1Config_targetCPU(name, 3, 70),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(hpaV1ResourceName, plancheck.ResourceActionUpdate)},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccHorizontalPodAutoscalerCheckExists(hpaV1ResourceName, &uid),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "spec.0.min_replicas", "3"),
					resource.TestCheckResourceAttr(hpaV1ResourceName, "spec.0.target_cpu_utilization_percentage", "70"),
				),
			},
			{
				ResourceName:      hpaV1ResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccHorizontalPodAutoscalerV1Config_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_horizontal_pod_autoscaler_v1" "test" {
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
    max_replicas = 10

    scale_target_ref {
      kind = "ReplicationController"
      name = "TerraformAccTest"
    }
  }
}
`, name)
}

func testAccHorizontalPodAutoscalerV1Config_metaModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_horizontal_pod_autoscaler_v1" "test" {
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

  spec {
    max_replicas = 10

    scale_target_ref {
      kind = "ReplicationController"
      name = "TerraformAccTest"
    }
  }
}
`, name)
}

func testAccHorizontalPodAutoscalerV1Config_specModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_horizontal_pod_autoscaler_v1" "test" {
  metadata {
    name = "%s"
  }

  spec {
    max_replicas = 8

    scale_target_ref {
      kind = "ReplicationController"
      name = "TerraformAccTestModified"
    }
  }
}
`, name)
}

func testAccHorizontalPodAutoscalerV1Config_generatedName(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_horizontal_pod_autoscaler_v1" "test" {
  metadata {
    generate_name = "%s"
  }

  spec {
    max_replicas = 1

    scale_target_ref {
      kind = "ReplicationController"
      name = "TerraformAccTestGeneratedName"
    }
  }
}
`, prefix)
}

func testAccHorizontalPodAutoscalerV1Config_targetCPU(name string, minReplicas, targetCPU int) string {
	return fmt.Sprintf(`resource "kubernetes_horizontal_pod_autoscaler_v1" "test" {
  metadata {
    name      = "%s"
    namespace = "default"
  }

  spec {
    min_replicas                      = %d
    max_replicas                      = 10
    target_cpu_utilization_percentage = %d

    scale_target_ref {
      api_version = "apps/v1"
      kind        = "Deployment"
      name        = "TerraformAccTest"
    }
  }
}
`, name, minReplicas, targetCPU)
}
