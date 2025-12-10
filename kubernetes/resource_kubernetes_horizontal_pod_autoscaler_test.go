// Copyright IBM Corp. 2017, 2025
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	api "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesHorizontalPodAutoscaler_basic(t *testing.T) {
	var conf api.HorizontalPodAutoscaler
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_horizontal_pod_autoscaler.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesHorizontalPodAutoscalerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesHorizontalPodAutoscalerConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesHorizontalPodAutoscalerExists(resourceName, &conf),
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
					resource.TestCheckResourceAttr(resourceName, "spec.0.max_replicas", "10"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.min_replicas", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scale_target_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scale_target_ref.0.kind", "ReplicationController"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scale_target_ref.0.name", "TerraformAccTest"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesHorizontalPodAutoscalerConfig_metaModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesHorizontalPodAutoscalerExists(resourceName, &conf),
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
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.max_replicas", "10"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.min_replicas", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scale_target_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scale_target_ref.0.kind", "ReplicationController"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scale_target_ref.0.name", "TerraformAccTest"),
				),
			},
			{
				Config: testAccKubernetesHorizontalPodAutoscalerConfig_specModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesHorizontalPodAutoscalerExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.max_replicas", "8"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.min_replicas", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scale_target_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scale_target_ref.0.kind", "ReplicationController"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scale_target_ref.0.name", "TerraformAccTestModified"),
				),
			},
		},
	})
}

func TestAccKubernetesHorizontalPodAutoscaler_generatedName(t *testing.T) {
	var conf api.HorizontalPodAutoscaler
	prefix := "tf-acc-test-"
	resourceName := "kubernetes_horizontal_pod_autoscaler.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesHorizontalPodAutoscalerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesHorizontalPodAutoscalerConfig_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesHorizontalPodAutoscalerExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.generate_name", prefix),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.max_replicas", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.min_replicas", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scale_target_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scale_target_ref.0.kind", "ReplicationController"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scale_target_ref.0.name", "TerraformAccTestGeneratedName"),
				),
			},
		},
	})
}

func testAccCheckKubernetesHorizontalPodAutoscalerDestroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_horizontal_pod_autoscaler" {
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

func testAccCheckKubernetesHorizontalPodAutoscalerExists(n string, obj *api.HorizontalPodAutoscaler) resource.TestCheckFunc {
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

		out, err := conn.AutoscalingV1().HorizontalPodAutoscalers(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesHorizontalPodAutoscalerConfig_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_horizontal_pod_autoscaler" "test" {
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

func testAccKubernetesHorizontalPodAutoscalerConfig_metaModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_horizontal_pod_autoscaler" "test" {
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

func testAccKubernetesHorizontalPodAutoscalerConfig_specModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_horizontal_pod_autoscaler" "test" {
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

func testAccKubernetesHorizontalPodAutoscalerConfig_generatedName(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_horizontal_pod_autoscaler" "test" {
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
