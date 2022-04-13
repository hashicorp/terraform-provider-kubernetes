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

func TestAccKubernetesResourceQuota_basic(t *testing.T) {
	var conf api.ResourceQuota
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_resource_quota.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesResourceQuotaDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesResourceQuotaConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaExists(resourceName, &conf),
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
				Config: testAccKubernetesResourceQuotaConfig_metaModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaExists(resourceName, &conf),
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
				Config: testAccKubernetesResourceQuotaConfig_specModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaExists(resourceName, &conf),
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

func TestAccKubernetesResourceQuota_generatedName(t *testing.T) {
	var conf api.ResourceQuota
	prefix := "tf-acc-test-"
	resourceName := "kubernetes_resource_quota.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesResourceQuotaDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesResourceQuotaConfig_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaExists(resourceName, &conf),
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

func TestAccKubernetesResourceQuota_withScopes(t *testing.T) {
	var conf api.ResourceQuota
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_resource_quota.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesResourceQuotaDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesResourceQuotaConfig_withScopes(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaExists(resourceName, &conf),
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
				Config: testAccKubernetesResourceQuotaConfig_withScopesModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaExists(resourceName, &conf),
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

func TestAccKubernetesResourceQuota_scopeSelector(t *testing.T) {
	var conf api.ResourceQuota
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_resource_quota.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesResourceQuotaDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesResourceQuotaConfigScopeSelector(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaExists(resourceName, &conf),
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
				Config: testAccKubernetesResourceQuotaConfigScopeSelectorModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaExists(resourceName, &conf),
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
				Config: testAccKubernetesResourceQuotaConfigMultipleMatchExpression(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaExists(resourceName, &conf),
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

func testAccCheckKubernetesResourceQuotaDestroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_resource_quota" {
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

func testAccCheckKubernetesResourceQuotaExists(n string, obj *api.ResourceQuota) resource.TestCheckFunc {
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

func testAccKubernetesResourceQuotaConfig_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_resource_quota" "test" {
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
    hard = {
      "limits.cpu"    = 2
      "limits.memory" = "2048Mi"
      pods            = 4
    }
  }
}
`, name)
}

func testAccKubernetesResourceQuotaConfig_metaModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_resource_quota" "test" {
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
    hard = {
      "limits.cpu"    = 2
      "limits.memory" = "2Gi"
      pods            = 4
    }
  }
}
`, name)
}

func testAccKubernetesResourceQuotaConfig_specModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_resource_quota" "test" {
  metadata {
    name = "%s"
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

func testAccKubernetesResourceQuotaConfig_generatedName(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_resource_quota" "test" {
  metadata {
    generate_name = "%s"
  }

  spec {
    hard = {
      pods = 10
    }
  }
}
`, prefix)
}

func testAccKubernetesResourceQuotaConfig_withScopes(name string) string {
	return fmt.Sprintf(`resource "kubernetes_resource_quota" "test" {
  metadata {
    name = "%s"
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

func testAccKubernetesResourceQuotaConfig_withScopesModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_resource_quota" "test" {
  metadata {
    name = "%s"
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

func testAccKubernetesResourceQuotaConfigScopeSelector(name string) string {
	return fmt.Sprintf(`resource "kubernetes_resource_quota" "test" {
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

func testAccKubernetesResourceQuotaConfigScopeSelectorModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_resource_quota" "test" {
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

func testAccKubernetesResourceQuotaConfigMultipleMatchExpression(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_resource_quota" "test" {
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
