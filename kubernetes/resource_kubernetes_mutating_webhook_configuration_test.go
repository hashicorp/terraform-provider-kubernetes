package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/errors"
)

func TestAccKubernetesMutatingWebhookConfiguration_basic(t *testing.T) {
	name := fmt.Sprintf("acc-test-%v.terraform.io", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_mutating_webhook_configuration.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesMutatingWebhookConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesMutatingWebhookConfigurationConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesMutatingWebhookConfigurationExists("kubernetes_mutating_webhook_configuration.test"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_mutating_webhook_configuration.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_mutating_webhook_configuration.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_mutating_webhook_configuration.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_mutating_webhook_configuration.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.admission_review_versions.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.admission_review_versions.0", "v1"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.admission_review_versions.1", "v1beta1"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.client_config.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.client_config.0.service.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.client_config.0.service.0.name", "example-service"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.client_config.0.service.0.namespace", "example-namespace"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.client_config.0.service.0.port", "443"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.failure_policy", "Fail"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.match_policy", "Equivalent"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.namespace_selector.#", "0"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.object_selector.#", "0"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.rule.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.rule.0.api_groups.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.rule.0.api_groups.0", "apps"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.rule.0.api_versions.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.rule.0.api_versions.0", "v1"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.rule.0.operations.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.rule.0.operations.0", "CREATE"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.rule.0.resources.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.rule.0.resources.0", "pods"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.rule.0.scope", "Namespaced"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.reinvocation_policy", "IfNeeded"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.side_effects", "None"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.timeout_seconds", "10"),
				),
			},
			{
				Config: testAccKubernetesMutatingWebhookConfigurationConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_mutating_webhook_configuration.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_mutating_webhook_configuration.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_mutating_webhook_configuration.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_mutating_webhook_configuration.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.admission_review_versions.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.admission_review_versions.0", "v1"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.admission_review_versions.1", "v1beta1"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.client_config.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.client_config.0.service.#", "0"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.client_config.0.url", "https://test"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.failure_policy", "Ignore"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.match_policy", "Exact"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.namespace_selector.#", "0"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.object_selector.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.rule.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.rule.0.api_groups.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.rule.0.api_groups.0", "apps"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.rule.0.api_versions.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.rule.0.api_versions.0", "v1"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.rule.0.operations.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.rule.0.operations.0", "CREATE"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.rule.0.resources.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.rule.0.resources.0", "pods"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.rule.0.scope", "Namespaced"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.rule.1.api_groups.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.rule.1.api_groups.0", "batch"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.rule.1.api_versions.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.rule.1.api_versions.0", "v1beta1"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.rule.1.operations.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.rule.1.operations.0", "CREATE"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.rule.1.resources.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.rule.1.resources.0", "cronjobs"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.rule.1.scope", "Namespaced"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.reinvocation_policy", "Never"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.side_effects", "NoneOnDryRun"),
					resource.TestCheckResourceAttr("kubernetes_mutating_webhook_configuration.test", "webhook.0.timeout_seconds", "5"),
				),
			},
		},
	})
}

func TestAccKubernetesMutatingWebhookConfiguration_importBasic(t *testing.T) {
	resourceName := "kubernetes_mutating_webhook_configuration.test"
	name := fmt.Sprintf("acc-test-%v.terraform.io", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesMutatingWebhookConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesMutatingWebhookConfigurationConfig_basic(name),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckKubernetesMutatingWebhookConfigurationDestroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_mutating_webhook_configuration" {
			continue
		}

		name := rs.Primary.ID

		useadmissionregistrationv1beta1, err := useAdmissionregistrationV1beta1(conn)
		if err != nil {
			return err
		}
		if useadmissionregistrationv1beta1 {
			_, err = conn.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Get(ctx, name, metav1.GetOptions{})
		} else {
			_, err = conn.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(ctx, name, metav1.GetOptions{})
		}

		if err != nil {
			if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
				return nil
			}
			return err
		}

		return fmt.Errorf("MutatingWebhookConfiguration still exists: %s", rs.Primary.ID)
	}

	return nil
}

func testAccCheckKubernetesMutatingWebhookConfigurationExists(n string) resource.TestCheckFunc {
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

		name := rs.Primary.ID

		useadmissionregistrationv1beta1, err := useAdmissionregistrationV1beta1(conn)
		if err != nil {
			return err
		}
		if useadmissionregistrationv1beta1 {
			_, err = conn.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Get(ctx, name, metav1.GetOptions{})
		} else {
			_, err = conn.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(ctx, name, metav1.GetOptions{})
		}
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccKubernetesMutatingWebhookConfigurationConfig_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_mutating_webhook_configuration" "test" {
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

func testAccKubernetesMutatingWebhookConfigurationConfig_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_mutating_webhook_configuration" "test" {
  metadata {
    name = %q
  }

  webhook {
    name = %q

    failure_policy = "Ignore"
    match_policy   = "Exact"

    admission_review_versions = [
      "v1",
      "v1beta1"
    ]

    client_config {
      url = "https://test"
    }

    rule {
      api_groups   = ["apps"]
      api_versions = ["v1"]
      operations   = ["CREATE"]
      resources    = ["pods"]
      scope        = "Namespaced"
    }

    rule {
      api_groups   = ["batch"]
      api_versions = ["v1beta1"]
      operations   = ["CREATE"]
      resources    = ["cronjobs"]
      scope        = "Namespaced"
    }

    object_selector {
      match_labels = {
        app = "test"
      }
    }

    reinvocation_policy = "Never"
    side_effects        = "NoneOnDryRun"
    timeout_seconds     = 5
  }
}
`, name, name)
}
