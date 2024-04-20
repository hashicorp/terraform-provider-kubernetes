// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccKubernetesDataSourceMutatingWebhookConfigurationV1_basic(t *testing.T) {
	resourceName := "kubernetes_mutating_webhook_configuration_v1.test"
	dataSourceName := fmt.Sprintf("data.%s", resourceName)
	name := fmt.Sprintf("acc-test-%v.terraform.io", acctest.RandString(10))
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			// AKS sets up some namespaceSelectors and thus we have to skip these tests
			skipIfRunningInAks(t)
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceMutatingWebhookConfigurationV1_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesMutatingWebhookConfigurationV1Exists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "webhook.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.admission_review_versions.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.admission_review_versions.0", "v1"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.admission_review_versions.1", "v1beta1"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.client_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.client_config.0.service.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.client_config.0.service.0.name", "example-service"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.client_config.0.service.0.namespace", "example-namespace"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.client_config.0.service.0.port", "443"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.failure_policy", "Fail"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.match_policy", "Equivalent"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.namespace_selector.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.object_selector.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.rule.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.rule.0.api_groups.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.rule.0.api_groups.0", "apps"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.rule.0.api_versions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.rule.0.api_versions.0", "v1"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.rule.0.operations.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.rule.0.operations.0", "CREATE"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.rule.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.rule.0.resources.0", "pods"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.rule.0.scope", "Namespaced"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.reinvocation_policy", "IfNeeded"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.side_effects", "None"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.timeout_seconds", "10"),
				),
			},
			{
				Config: testAccKubernetesDataSourceMutatingWebhookConfigurationV1_basic(name) +
					testAccKubernetesDataSourceMutatingWebhookConfigurationV1_read(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(dataSourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(dataSourceName, "webhook.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "webhook.0.admission_review_versions.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "webhook.0.admission_review_versions.0", "v1"),
					resource.TestCheckResourceAttr(dataSourceName, "webhook.0.admission_review_versions.1", "v1beta1"),
					resource.TestCheckResourceAttr(dataSourceName, "webhook.0.client_config.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "webhook.0.client_config.0.service.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "webhook.0.client_config.0.service.0.name", "example-service"),
					resource.TestCheckResourceAttr(dataSourceName, "webhook.0.client_config.0.service.0.namespace", "example-namespace"),
					resource.TestCheckResourceAttr(dataSourceName, "webhook.0.client_config.0.service.0.port", "443"),
					resource.TestCheckResourceAttr(dataSourceName, "webhook.0.failure_policy", "Fail"),
					resource.TestCheckResourceAttr(dataSourceName, "webhook.0.match_policy", "Equivalent"),
					resource.TestCheckResourceAttr(dataSourceName, "webhook.0.name", name),
					resource.TestCheckResourceAttr(dataSourceName, "webhook.0.namespace_selector.#", "0"),
					resource.TestCheckResourceAttr(dataSourceName, "webhook.0.object_selector.#", "0"),
					resource.TestCheckResourceAttr(dataSourceName, "webhook.0.rule.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "webhook.0.rule.0.api_groups.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "webhook.0.rule.0.api_groups.0", "apps"),
					resource.TestCheckResourceAttr(dataSourceName, "webhook.0.rule.0.api_versions.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "webhook.0.rule.0.api_versions.0", "v1"),
					resource.TestCheckResourceAttr(dataSourceName, "webhook.0.rule.0.operations.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "webhook.0.rule.0.operations.0", "CREATE"),
					resource.TestCheckResourceAttr(dataSourceName, "webhook.0.rule.0.resources.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "webhook.0.rule.0.resources.0", "pods"),
					resource.TestCheckResourceAttr(dataSourceName, "webhook.0.rule.0.scope", "Namespaced"),
					resource.TestCheckResourceAttr(dataSourceName, "webhook.0.reinvocation_policy", "IfNeeded"),
					resource.TestCheckResourceAttr(dataSourceName, "webhook.0.side_effects", "None"),
					resource.TestCheckResourceAttr(dataSourceName, "webhook.0.timeout_seconds", "10"),
				),
			},
		},
	})
}

func TestAccKubernetesDataSourceMutatingWebhookConfigurationV1_not_found(t *testing.T) {
	dataSourceName := "data.kubernetes_mutating_webhook_configuration_v1.test"
	name := fmt.Sprintf("ceci-n.est-pas-une-webhook-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceMutatingWebhookConfigurationV1_nonexistent(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(dataSourceName, "webhook.#", "0"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceMutatingWebhookConfigurationV1_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_mutating_webhook_configuration_v1" "test" {
  metadata {
    name = %q
  }
  webhook {
    name = %q
    admission_review_versions = [
      "v1",
      "v1beta1"
    ]
    client_config {
      service {
        namespace = "example-namespace"
        name      = "example-service"
      }
    }
    rule {
      api_groups   = ["apps"]
      api_versions = ["v1"]
      operations   = ["CREATE"]
      resources    = ["pods"]
      scope        = "Namespaced"
    }
    reinvocation_policy = "IfNeeded"
    side_effects        = "None"
    timeout_seconds     = 10
  }
}
`, name, name)
}

func testAccKubernetesDataSourceMutatingWebhookConfigurationV1_read() string {
	return `data "kubernetes_mutating_webhook_configuration_v1" "test" {
  metadata {
    name = "${kubernetes_mutating_webhook_configuration_v1.test.metadata.0.name}"
  }
}
`
}

func testAccKubernetesDataSourceMutatingWebhookConfigurationV1_nonexistent(name string) string {
	return fmt.Sprintf(`data "kubernetes_mutating_webhook_configuration_v1" "test" {
  metadata {
    name = "%s"
  }
}
`, name)
}
