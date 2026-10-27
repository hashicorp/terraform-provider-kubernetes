// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	corev1api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const limitRangeResourceName = "kubernetes_limit_range_v1.test"

// splitID splits a "namespace/name" resource ID — the format BuildId produces.
func splitID(id string) (string, string, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected ID format %q, expected namespace/name", id)
	}
	return parts[0], parts[1], nil
}

// ── lifecycle tests ───────────────────────────────────────────────────────────

func TestAccKubernetesLimitRangeV1_basic(t *testing.T) {
	var conf corev1api.LimitRange
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesLimitRangeV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesLimitRangeV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeV1Exists(limitRangeResourceName, &conf),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.labels.TestLabelFour", "four"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(limitRangeResourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(limitRangeResourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(limitRangeResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.#", "1"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.default.%", "2"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.default.cpu", "200m"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.default.memory", "512M"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.default_request.%", "2"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.default_request.cpu", "100m"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.default_request.memory", "256M"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.type", "Container"),
				),
			},
			// Import round-trip: the ID is "namespace/name".
			{
				ResourceName:            limitRangeResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			// Verify import produces a no-op plan (ImportStateVerify cannot see null vs empty).
			{
				Config: testAccKubernetesLimitRangeV1Config_basic(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: testAccKubernetesLimitRangeV1Config_metaModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeV1Exists(limitRangeResourceName, &conf),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.#", "1"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.default.cpu", "200m"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.default.memory", "512M"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.type", "Container"),
				),
			},
			{
				Config: testAccKubernetesLimitRangeV1Config_specModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeV1Exists(limitRangeResourceName, &conf),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.#", "1"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.default.cpu", "200m"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.default.memory", "1024M"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.max.%", "1"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.max.cpu", "500m"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.min.%", "2"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.min.cpu", "10m"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.min.memory", "10M"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.type", "Container"),
				),
			},
		},
	})
}

func TestAccKubernetesLimitRangeV1_empty(t *testing.T) {
	var conf corev1api.LimitRange
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesLimitRangeV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesLimitRangeV1Config_empty(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeV1Exists(limitRangeResourceName, &conf),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(limitRangeResourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(limitRangeResourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(limitRangeResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.#", "0"),
				),
			},
		},
	})
}

func TestAccKubernetesLimitRangeV1_generatedName(t *testing.T) {
	var conf corev1api.LimitRange
	prefix := "tf-acc-test-"
	ns := fmt.Sprintf("%s-%s", prefix, acctest.RandString(10))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesLimitRangeV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesLimitRangeV1Config_generatedName(prefix, ns),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeV1Exists(limitRangeResourceName, &conf),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.generate_name", prefix),
					resource.TestCheckResourceAttrSet(limitRangeResourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(limitRangeResourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(limitRangeResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.#", "1"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.type", "Pod"),
				),
			},
		},
	})
}

func TestAccKubernetesLimitRangeV1_typeChange(t *testing.T) {
	var conf corev1api.LimitRange
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesLimitRangeV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesLimitRangeV1Config_typeChange(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeV1Exists(limitRangeResourceName, &conf),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.default.cpu", "200m"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.default.memory", "1024M"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.type", "Container"),
				),
			},
			{
				Config: testAccKubernetesLimitRangeV1Config_typeChangeModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeV1Exists(limitRangeResourceName, &conf),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.min.cpu", "200m"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.min.memory", "1024M"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.type", "Pod"),
				),
			},
		},
	})
}

func TestAccKubernetesLimitRangeV1_multipleLimits(t *testing.T) {
	var conf corev1api.LimitRange
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesLimitRangeV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesLimitRangeV1Config_multipleLimits(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeV1Exists(limitRangeResourceName, &conf),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.#", "3"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.max.cpu", "200m"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.max.memory", "1024M"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.type", "Pod"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.1.min.storage", "24M"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.1.type", "PersistentVolumeClaim"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.2.default.cpu", "50m"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.2.default.memory", "24M"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.2.type", "Container"),
				),
			},
		},
	})
}

// ── CheckDestroy + Exists helpers ────────────────────────────────────────────

func testAccCheckKubernetesLimitRangeV1Destroy(s *terraform.State) error {
	conn, err := mainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_limit_range_v1" {
			continue
		}
		ns, name, err := splitID(rs.Primary.ID)
		if err != nil {
			return err
		}
		resp, err := conn.CoreV1().LimitRanges(ns).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Namespace == ns && resp.Name == name {
				return fmt.Errorf("LimitRange still exists: %s", rs.Primary.ID)
			}
		}
	}
	return nil
}

func testAccCheckKubernetesLimitRangeV1Exists(n string, obj *corev1api.LimitRange) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}
		conn, err := mainClientset()
		if err != nil {
			return err
		}
		ns, name, err := splitID(rs.Primary.ID)
		if err != nil {
			return err
		}
		out, err := conn.CoreV1().LimitRanges(ns).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		*obj = *out
		return nil
	}
}

// ── HCL configs ──────────────────────────────────────────────────────────────

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
