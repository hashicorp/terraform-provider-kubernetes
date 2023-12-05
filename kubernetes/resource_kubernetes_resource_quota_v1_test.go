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
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesResourceQuotaV1_basic(t *testing.T) {
	var conf api.ResourceQuota
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_resource_quota_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesResourceQuotaV1Destroy,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesResourceQuotaV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaV1Exists(resourceName, &conf),
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
					resource.TestCheckResourceAttr(resourceName, "spec.0.hard.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.hard.limits.cpu", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.hard.limits.memory", "2Gi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.hard.pods", "4"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesResourceQuotaV1Config_metaModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaV1Exists(resourceName, &conf),
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
					resource.TestCheckResourceAttr(resourceName, "spec.0.hard.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.hard.limits.cpu", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.hard.limits.memory", "2Gi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.hard.pods", "4"),
				),
			},
			{
				Config: testAccKubernetesResourceQuotaV1Config_specModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.hard.%", "4"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.hard.limits.cpu", "4"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.hard.requests.cpu", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.hard.limits.memory", "4Gi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.hard.pods", "10"),
				),
			},
		},
	})
}

func TestAccKubernetesResourceQuotaV1_generatedName(t *testing.T) {
	var conf api.ResourceQuota
	prefix := "tf-acc-test-"
	ns := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_resource_quota_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesResourceQuotaV1Destroy,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesResourceQuotaV1Config_generatedName(ns, prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.generate_name", prefix),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.hard.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.hard.pods", "10"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scopes.#", "0"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
		},
	})
}

func TestAccKubernetesResourceQuotaV1_withScopes(t *testing.T) {
	var conf api.ResourceQuota
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_resource_quota_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesResourceQuotaV1Destroy,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesResourceQuotaV1Config_withScopes(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.hard.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.hard.pods", "10"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scopes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scopes.0", "BestEffort"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesResourceQuotaV1Config_withScopesModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.hard.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.hard.pods", "10"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scopes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scopes.0", "NotBestEffort"),
				),
			},
		},
	})
}

func TestAccKubernetesResourceQuotaV1_scopeSelector(t *testing.T) {
	var conf api.ResourceQuota
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_resource_quota_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesResourceQuotaV1Destroy,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesResourceQuotaV1ConfigScopeSelector(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaV1Exists(resourceName, &conf),
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
					resource.TestCheckResourceAttr(resourceName, "spec.0.scope_selector.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scope_selector.0.match_expression.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scope_selector.0.match_expression.0.scope_name", "PriorityClass"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scope_selector.0.match_expression.0.operator", "In"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scope_selector.0.match_expression.0.values.#", "1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "spec.0.scope_selector.0.match_expression.0.values.*", "medium"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesResourceQuotaV1ConfigScopeSelectorModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaV1Exists(resourceName, &conf),
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
					resource.TestCheckResourceAttr(resourceName, "spec.0.scope_selector.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scope_selector.0.match_expression.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scope_selector.0.match_expression.0.scope_name", "PriorityClass"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scope_selector.0.match_expression.0.operator", "NotIn"),
					resource.TestCheckTypeSetElemAttr(resourceName, "spec.0.scope_selector.0.match_expression.0.values.*", "large"),
				),
			},
			{
				Config: testAccKubernetesResourceQuotaV1ConfigMultipleMatchExpression(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaV1Exists(resourceName, &conf),
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
					resource.TestCheckResourceAttr(resourceName, "spec.0.scope_selector.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scope_selector.0.match_expression.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scope_selector.0.match_expression.0.scope_name", "PriorityClass"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scope_selector.0.match_expression.0.operator", "NotIn"),
					resource.TestCheckTypeSetElemAttr(resourceName, "spec.0.scope_selector.0.match_expression.0.values.*", "large"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scope_selector.0.match_expression.1.scope_name", "PriorityClass"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scope_selector.0.match_expression.1.operator", "In"),
					resource.TestCheckTypeSetElemAttr(resourceName, "spec.0.scope_selector.0.match_expression.1.values.*", "low"),
				),
			},
		},
	})
}

func testAccCheckKubernetesResourceQuotaV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_resource_quota_v1" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.CoreV1().ResourceQuotas(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Namespace == namespace && resp.Name == name {
				return fmt.Errorf("Resource Quota still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesResourceQuotaV1Exists(n string, obj *api.ResourceQuota) resource.TestCheckFunc {
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

		out, err := conn.CoreV1().ResourceQuotas(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesResourceQuotaV1Config_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_resource_quota_v1" "test" {
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
    hard = {
      "limits.cpu"    = 2
      "limits.memory" = "2048Mi"
      pods            = 4
    }
  }
}
`, name)
}

func testAccKubernetesResourceQuotaV1Config_metaModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_resource_quota_v1" "test" {
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
    hard = {
      "limits.cpu"    = 2
      "limits.memory" = "2Gi"
      pods            = 4
    }
  }
}
`, name)
}

func testAccKubernetesResourceQuotaV1Config_specModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_resource_quota_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }

  spec {
    hard = {
      "limits.cpu"    = 4
      "requests.cpu"  = 1
      "limits.memory" = "4Gi"
      pods            = 10
    }
  }
}
`, name)
}

func testAccKubernetesResourceQuotaV1Config_generatedName(ns, prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_resource_quota_v1" "test" {
  metadata {
    generate_name = %[2]q
    namespace     = kubernetes_namespace_v1.test.metadata.0.name
  }

  spec {
    hard = {
      pods = 10
    }
  }
}
`, ns, prefix)
}

func testAccKubernetesResourceQuotaV1Config_withScopes(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_resource_quota_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }

  spec {
    hard = {
      pods = 10
    }

    scopes = ["BestEffort"]
  }
}
`, name)
}

func testAccKubernetesResourceQuotaV1Config_withScopesModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_resource_quota_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }

  spec {
    hard = {
      pods = 10
    }

    scopes = ["NotBestEffort"]
  }
}
`, name)
}

func testAccKubernetesResourceQuotaV1ConfigScopeSelector(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_resource_quota_v1" "test" {
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
    hard = {
      "limits.cpu"    = 2
      "limits.memory" = "2Gi"
      pods            = 4
    }

    scope_selector {
      match_expression {
        scope_name = "PriorityClass"
        operator   = "In"
        values     = ["medium"]
      }
    }
  }
}
`, name)
}

func testAccKubernetesResourceQuotaV1ConfigScopeSelectorModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_resource_quota_v1" "test" {
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
    hard = {
      "limits.cpu"    = 2
      "limits.memory" = "2Gi"
      pods            = 4
    }

    scope_selector {
      match_expression {
        scope_name = "PriorityClass"
        operator   = "NotIn"
        values     = ["large"]
      }
    }
  }
}
`, name)
}

func testAccKubernetesResourceQuotaV1ConfigMultipleMatchExpression(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_resource_quota_v1" "test" {
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
    hard = {
      "limits.cpu"    = 2
      "limits.memory" = "2Gi"
      pods            = 4
    }

    scope_selector {
      match_expression {
        scope_name = "PriorityClass"
        operator   = "NotIn"
        values     = ["large"]
      }
      match_expression {
        scope_name = "PriorityClass"
        operator   = "In"
        values     = ["low"]
      }
    }
  }
}
`, name)
}
