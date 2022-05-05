package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	api "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesHorizontalPodAutoscalerV1_basic(t *testing.T) {
	var conf api.HorizontalPodAutoscaler
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_horizontal_pod_autoscaler_v1.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_horizontal_pod_autoscaler_v1.test",
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesHorizontalPodAutoscalerV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesHorizontalPodAutoscalerV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesHorizontalPodAutoscalerV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.labels.TestLabelFour", "four"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "spec.0.max_replicas", "10"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "spec.0.min_replicas", "1"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "spec.0.scale_target_ref.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "spec.0.scale_target_ref.0.kind", "ReplicationController"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "spec.0.scale_target_ref.0.name", "TerraformAccTest"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccKubernetesHorizontalPodAutoscalerV1Config_metaModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesHorizontalPodAutoscalerV1Exists("kubernetes_horizontal_pod_autoscaler_v1.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "spec.0.max_replicas", "10"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "spec.0.min_replicas", "1"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "spec.0.scale_target_ref.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "spec.0.scale_target_ref.0.kind", "ReplicationController"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "spec.0.scale_target_ref.0.name", "TerraformAccTest"),
				),
			},
			{
				Config: testAccKubernetesHorizontalPodAutoscalerV1Config_specModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesHorizontalPodAutoscalerV1Exists("kubernetes_horizontal_pod_autoscaler_v1.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "spec.0.max_replicas", "8"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "spec.0.min_replicas", "1"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "spec.0.scale_target_ref.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "spec.0.scale_target_ref.0.kind", "ReplicationController"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "spec.0.scale_target_ref.0.name", "TerraformAccTestModified"),
				),
			},
		},
	})
}

func TestAccKubernetesHorizontalPodAutoscalerV1_generatedName(t *testing.T) {
	var conf api.HorizontalPodAutoscaler
	prefix := "tf-acc-test-"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_horizontal_pod_autoscaler_v1.test",
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesHorizontalPodAutoscalerV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesHorizontalPodAutoscalerV1Config_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesHorizontalPodAutoscalerV1Exists("kubernetes_horizontal_pod_autoscaler_v1.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.generate_name", prefix),
					resource.TestCheckResourceAttrSet("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_horizontal_pod_autoscaler_v1.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "spec.0.max_replicas", "1"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "spec.0.min_replicas", "1"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "spec.0.scale_target_ref.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "spec.0.scale_target_ref.0.kind", "ReplicationController"),
					resource.TestCheckResourceAttr("kubernetes_horizontal_pod_autoscaler_v1.test", "spec.0.scale_target_ref.0.name", "TerraformAccTestGeneratedName"),
				),
			},
		},
	})
}

func testAccCheckKubernetesHorizontalPodAutoscalerV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_horizontal_pod_autoscaler_v1" {
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

func testAccCheckKubernetesHorizontalPodAutoscalerV1Exists(n string, obj *api.HorizontalPodAutoscaler) resource.TestCheckFunc {
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

func testAccKubernetesHorizontalPodAutoscalerV1Config_basic(name string) string {
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

func testAccKubernetesHorizontalPodAutoscalerV1Config_metaModified(name string) string {
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

func testAccKubernetesHorizontalPodAutoscalerV1Config_specModified(name string) string {
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

func testAccKubernetesHorizontalPodAutoscalerV1Config_generatedName(prefix string) string {
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
