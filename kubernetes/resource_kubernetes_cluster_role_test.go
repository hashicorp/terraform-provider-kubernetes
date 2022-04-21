package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	api "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesClusterRole_basic(t *testing.T) {
	var conf api.ClusterRole
	resourceName := "kubernetes_cluster_role.test"
	name := acctest.RandomWithPrefix("tf-acc-test")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesClusterRoleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "rule.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resources.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resources.0", "pods"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resources.1", "pods/log"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.0", "get"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.1", "list"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesClusterRoleConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "rule.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.2", "watch"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.api_groups.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.resources.0", "deployments"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.verbs.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.verbs.0", "get"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.verbs.1", "list"),
					resource.TestCheckResourceAttr(resourceName, "rule.2.non_resource_urls.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.2.non_resource_urls.0", "/metrics"),
					resource.TestCheckResourceAttr(resourceName, "rule.2.verbs.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.2.verbs.0", "get"),
				),
			},
		},
	})
}

func TestAccKubernetesClusterRole_UpdatePatchOperationsOrderWithRemovals(t *testing.T) {
	var conf api.ClusterRole
	resourceName := "kubernetes_cluster_role.test"
	name := acctest.RandomWithPrefix("tf-acc-test")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesClusterRoleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleConfigBug_step_0(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "rule.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resources.0", "pods"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.0", "get"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.resources.0", "deployments"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.verbs.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.verbs.0", "list"),
					resource.TestCheckResourceAttr(resourceName, "rule.2.non_resource_urls.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.2.non_resource_urls.0", "/metrics"),
					resource.TestCheckResourceAttr(resourceName, "rule.2.verbs.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.2.verbs.0", "get"),
				),
			},
			{
				Config: testAccKubernetesClusterRoleConfigBug_step_1(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "rule.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resources.0", "deployments"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.0", "get"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.1", "list"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.api_groups.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.resources.0", "jobs"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.verbs.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.verbs.0", "get"),
				),
			},
			{
				Config: testAccKubernetesClusterRoleConfigBug_step_2(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "rule.#", "4"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resources.0", "pods"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.0", "list"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.resources.0", "deployments"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.verbs.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.verbs.0", "list"),
					resource.TestCheckResourceAttr(resourceName, "rule.2.non_resource_urls.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.2.non_resource_urls.0", "/metrics"),
					resource.TestCheckResourceAttr(resourceName, "rule.2.verbs.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.2.verbs.0", "get"),
					resource.TestCheckResourceAttr(resourceName, "rule.3.api_groups.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.3.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.3.resources.0", "jobs"),
					resource.TestCheckResourceAttr(resourceName, "rule.3.verbs.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.3.verbs.0", "get"),
				),
			},
		},
	})
}

func TestAccKubernetesClusterRole_aggregationRuleBasic(t *testing.T) {
	var conf api.ClusterRole
	resourceName := "kubernetes_cluster_role.test"
	name := acctest.RandomWithPrefix("tf-acc-test")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesClusterRoleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleConfig_aggRule(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "aggregation_rule.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "aggregation_rule.0.cluster_role_selectors.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "aggregation_rule.0.cluster_role_selectors.0.match_expressions.0.key", "environment"),
					resource.TestCheckResourceAttr(resourceName, "aggregation_rule.0.cluster_role_selectors.0.match_expressions.0.operator", "In"),
					resource.TestCheckResourceAttr(resourceName, "aggregation_rule.0.cluster_role_selectors.0.match_expressions.0.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "aggregation_rule.0.cluster_role_selectors.0.match_labels.foo", "bar"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesClusterRoleConfig_aggRuleModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "aggregation_rule.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "aggregation_rule.0.cluster_role_selectors.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "aggregation_rule.0.cluster_role_selectors.0.match_expressions.0.key", "env"),
					resource.TestCheckResourceAttr(resourceName, "aggregation_rule.0.cluster_role_selectors.0.match_expressions.0.operator", "NotIn"),
					resource.TestCheckResourceAttr(resourceName, "aggregation_rule.0.cluster_role_selectors.0.match_expressions.0.values.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "aggregation_rule.0.cluster_role_selectors.0.match_labels.bar", "foo"),
				),
			},
		},
	})
}

func TestAccKubernetesClusterRole_aggregationRuleMultiple(t *testing.T) {
	var conf api.ClusterRole
	resourceName := "kubernetes_cluster_role.test"
	name := acctest.RandomWithPrefix("tf-acc-test")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesClusterRoleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleConfig_aggRuleMultiple(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "aggregation_rule.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "aggregation_rule.0.cluster_role_selectors.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "aggregation_rule.0.cluster_role_selectors.0.match_labels.foo", "bar"),
					resource.TestCheckResourceAttr(resourceName, "aggregation_rule.0.cluster_role_selectors.1.match_labels.bar", "foo"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesClusterRoleConfig_aggRuleMultipleModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "aggregation_rule.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "aggregation_rule.0.cluster_role_selectors.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "aggregation_rule.0.cluster_role_selectors.0.match_labels.bar", "foo"),
					resource.TestCheckResourceAttr(resourceName, "aggregation_rule.0.cluster_role_selectors.1.match_labels.foo", "bar"),
				),
			},
		},
	})
}

func TestAccKubernetesClusterRole_aggregationRuleRuleAggregation(t *testing.T) {
	var conf api.ClusterRole
	resourceName := "kubernetes_cluster_role.test"
	name := acctest.RandomWithPrefix("tf-acc-test")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesClusterRoleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleConfig_aggRule2(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleExists(resourceName, &conf),
				),
			},
			{
				Config: testAccKubernetesClusterRoleConfig_aggRule2(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesClusterRoleExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "rule.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resources.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resources.0", "pods"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resources.1", "pods/log"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.0", "get"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.1", "list"),
				),
			},
		},
	})
}

func testAccCheckKubernetesClusterRoleDestroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_cluster_role" {
			continue
		}
		resp, err := conn.RbacV1().ClusterRoles().Get(ctx, rs.Primary.ID, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Cluster Role still exists: %s", rs.Primary.ID)
			}
		}
	}
	return nil
}

func testAccCheckKubernetesClusterRoleExists(n string, obj *api.ClusterRole) resource.TestCheckFunc {
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

		out, err := conn.RbacV1().ClusterRoles().Get(ctx, rs.Primary.ID, metav1.GetOptions{})
		if err != nil {
			return err
		}
		*obj = *out
		return nil
	}
}

func testAccKubernetesClusterRoleConfig_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role" "test" {
  metadata {
    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    name = "%s"
  }

  rule {
    api_groups = [""]
    resources  = ["pods", "pods/log"]
    verbs      = ["get", "list"]
  }
}
`, name)
}

func testAccKubernetesClusterRoleConfig_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role" "test" {
  metadata {
    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
    }

    name = "%s"
  }

  rule {
    api_groups = [""]
    resources  = ["pods", "pods/log"]
    verbs      = ["get", "list", "watch"]
  }

  rule {
    api_groups = [""]
    resources  = ["deployments"]
    verbs      = ["get", "list"]
  }

  rule {
    non_resource_urls = ["/metrics"]
    verbs             = ["get"]
  }
}
`, name)
}

func testAccKubernetesClusterRoleConfigBug_step_0(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role" "test" {
  metadata {
    name = "%s"
  }

  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["get"]
  }

  rule {
    api_groups = [""]
    resources  = ["deployments"]
    verbs      = ["list"]
  }

  rule {
    non_resource_urls = ["/metrics"]
    verbs             = ["get"]
  }
}
`, name)
}

func testAccKubernetesClusterRoleConfigBug_step_1(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role" "test" {
  metadata {
    name = "%s"
  }

  rule {
    api_groups = [""]
    resources  = ["deployments"]
    verbs      = ["get", "list"]
  }

  rule {
    api_groups = [""]
    resources  = ["jobs"]
    verbs      = ["get"]
  }
}
`, name)
}

func testAccKubernetesClusterRoleConfigBug_step_2(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role" "test" {
  metadata {
    name = "%s"
  }

  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["list"]
  }

  rule {
    api_groups = [""]
    resources  = ["deployments"]
    verbs      = ["list"]
  }

  rule {
    non_resource_urls = ["/metrics"]
    verbs             = ["get"]
  }

  rule {
    api_groups = [""]
    resources  = ["jobs"]
    verbs      = ["get"]
  }
}
`, name)
}

func testAccKubernetesClusterRoleConfig_aggRule(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role" "test" {
  metadata {
    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    name = "%s"
  }

  aggregation_rule {
    cluster_role_selectors {
      match_labels = {
        foo = "bar"
      }

      match_expressions {
        key      = "environment"
        operator = "In"
        values   = ["non-exists-12345"]
      }
    }
  }
}
`, name)
}

func testAccKubernetesClusterRoleConfig_aggRuleModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role" "test" {
  metadata {
    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    name = "%s"
  }

  aggregation_rule {
    cluster_role_selectors {
      match_labels = {
        bar = "foo"
      }

      match_expressions {
        key      = "env"
        operator = "NotIn"
        values   = ["non"]
      }
    }
  }
}
`, name)
}

func testAccKubernetesClusterRoleConfig_aggRuleMultiple(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role" "test" {
  metadata {
    name = "%s"
  }

  aggregation_rule {
    cluster_role_selectors {
      match_labels = {
        foo = "bar"
      }
    }
    cluster_role_selectors {
      match_labels = {
        bar = "foo"
      }
    }
  }
}
`, name)
}

func testAccKubernetesClusterRoleConfig_aggRuleMultipleModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role" "test" {
  metadata {
    name = "%s"
  }

  aggregation_rule {
    cluster_role_selectors {
      match_labels = {
        bar = "foo"
      }
    }
    cluster_role_selectors {
      match_labels = {
        foo = "bar"
      }
    }
  }
}
`, name)
}

func testAccKubernetesClusterRoleConfig_aggRule2(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role" "test" {
  metadata {
    name = "%[1]s"
  }

  aggregation_rule {
    cluster_role_selectors {
      match_labels = {
        "rbac.example.com/aggregate-to-monitoring" = "true"
      }
    }
  }
}

resource "kubernetes_cluster_role" "test2" {
  metadata {
    labels = {
      "rbac.example.com/aggregate-to-monitoring" = "true"
    }
    name = "%[1]s-2"
  }

  rule {
    api_groups = [""]
    resources  = ["pods", "pods/log"]
    verbs      = ["get", "list"]
  }
}
`, name)
}
