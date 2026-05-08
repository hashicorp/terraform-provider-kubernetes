// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesLimitRangeV1_basic(t *testing.T) {
	var conf api.LimitRange
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_limit_range_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesLimitRangeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesLimitRangeV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeExists(resourceName, &conf),
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
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.default.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.default.cpu", "200m"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.default.memory", "512M"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.default_request.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.default_request.cpu", "100m"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.default_request.memory", "256M"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.type", "Container"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesLimitRangeV1Config_metaModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeExists(resourceName, &conf),
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
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.default.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.default.cpu", "200m"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.default.memory", "512M"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.default_request.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.default_request.cpu", "100m"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.default_request.memory", "256M"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.type", "Container"),
				),
			},
			{
				Config: testAccKubernetesLimitRangeV1Config_specModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.default.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.default.cpu", "200m"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.default.memory", "1024M"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.default_request.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.default_request.cpu", "100m"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.default_request.memory", "256M"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.max.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.max.cpu", "500m"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.min.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.min.cpu", "10m"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.min.memory", "10M"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.type", "Container"),
				),
			},
		},
	})
}

func TestAccKubernetesLimitRangeV1_empty(t *testing.T) {
	var conf api.LimitRange
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_limit_range_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesLimitRangeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesLimitRangeV1Config_empty(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "0"),
				),
			},
		},
	})
}

func TestAccKubernetesLimitRangeV1_generatedName(t *testing.T) {
	var conf api.LimitRange
	prefix := "tf-acc-test-"
	ns := fmt.Sprintf("%s-%s", prefix, acctest.RandString(10))
	resourceName := "kubernetes_limit_range_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesLimitRangeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesLimitRangeV1Config_generatedName(prefix, ns),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.generate_name", prefix),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.type", "Pod"),
				),
			},
		},
	})
}

func TestAccKubernetesLimitRangeV1_typeChange(t *testing.T) {
	var conf api.LimitRange
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_limit_range_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesLimitRangeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesLimitRangeV1Config_typeChange(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.default.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.default.cpu", "200m"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.default.memory", "1024M"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.type", "Container"),
				),
			},
			{
				Config: testAccKubernetesLimitRangeV1Config_typeChangeModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.min.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.min.cpu", "200m"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.min.memory", "1024M"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.type", "Pod"),
				),
			},
		},
	})
}

func TestAccKubernetesLimitRangeV1_multipleLimits(t *testing.T) {
	var conf api.LimitRange
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_limit_range_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesLimitRangeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesLimitRangeV1Config_multipleLimits(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.max.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.max.cpu", "200m"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.max.memory", "1024M"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.0.type", "Pod"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.1.min.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.1.min.storage", "24M"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.1.type", "PersistentVolumeClaim"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.2.default.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.2.default.cpu", "50m"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.2.default.memory", "24M"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.limit.2.type", "Container"),
				),
			},
		},
	})
}

func testAccCheckKubernetesLimitRangeDestroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_limit_range_v1" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.CoreV1().LimitRanges(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Namespace == namespace && resp.Name == name {
				return fmt.Errorf("Limit Range still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesLimitRangeExists(n string, obj *api.LimitRange) resource.TestCheckFunc {
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

		out, err := conn.CoreV1().LimitRanges(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesLimitRangeV1Config_empty(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
}
`, name)
}

func testAccKubernetesLimitRangeV1Config_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
      TestLabelFour  = "four"
    }

    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }

  spec {
    limit {
      type = "Container"

      default = {
        cpu    = "200m"
        memory = "512M"
      }

      default_request = {
        cpu    = "100m"
        memory = "256M"
      }
    }
  }
}
`, name)
}

func testAccKubernetesLimitRangeV1Config_metaModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
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

    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }

  spec {
    limit {
      type = "Container"

      default = {
        cpu    = "200m"
        memory = "512M"
      }

      default_request = {
        cpu    = "100m"
        memory = "256M"
      }
    }
  }
}
`, name)
}

func testAccKubernetesLimitRangeV1Config_specModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }

  spec {
    limit {
      type = "Container"

      default = {
        cpu    = "200m"
        memory = "1024M"
      }

      max = {
        cpu = "500m"
      }

      min = {
        cpu    = "10m"
        memory = "10M"
      }
    }
  }
}
`, name)
}

func testAccKubernetesLimitRangeV1Config_generatedName(prefix, ns string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
  metadata {
    generate_name = %[2]q
    namespace     = kubernetes_namespace_v1.test.metadata.0.name
  }

  spec {
    limit {
      type = "Pod"
    }
  }
}
`, ns, prefix)
}

func testAccKubernetesLimitRangeV1Config_typeChange(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }

  spec {
    limit {
      type = "Container"

      default = {
        cpu    = "200m"
        memory = "1024M"
      }
    }
  }
}
`, name)
}

func testAccKubernetesLimitRangeV1Config_typeChangeModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }

  spec {
    limit {
      type = "Pod"

      min = {
        cpu    = "200m"
        memory = "1024M"
      }
    }
  }
}
`, name)
}

func testAccKubernetesLimitRangeV1Config_multipleLimits(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }

  spec {
    limit {
      type = "Pod"

      max = {
        cpu    = "200m"
        memory = "1024M"
      }
    }

    limit {
      type = "PersistentVolumeClaim"

      min = {
        storage = "24M"
      }
    }

    limit {
      type = "Container"

      default = {
        cpu    = "50m"
        memory = "24M"
      }
    }
  }
}
`, name)
}
