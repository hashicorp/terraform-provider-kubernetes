package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/errors"
)

func TestAccKubernetesValidatingWebhookConfiguration_basic(t *testing.T) {
	name := fmt.Sprintf("acc-test-%v.terraform.io", acctest.RandString(10))
	resourceName := "kubernetes_validating_webhook_configuration.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotAdmissionRegistrationV1(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesValdiatingWebhookConfigurationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesValidatingWebhookConfigurationConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesValidatingWebhookConfigurationExists(resourceName),
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
					resource.TestCheckResourceAttr(resourceName, "webhook.0.side_effects", "None"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.timeout_seconds", "10"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccKubernetesValidatingWebhookConfigurationConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
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
					resource.TestCheckResourceAttr(resourceName, "webhook.0.failure_policy", "Ignore"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.match_policy", "Exact"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.namespace_selector.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.object_selector.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.rule.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.rule.0.api_groups.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.rule.0.api_groups.0", "apps"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.rule.0.api_versions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.rule.0.api_versions.0", "v1"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.rule.0.operations.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.rule.0.operations.0", "CREATE"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.rule.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.rule.0.resources.0", "pods"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.rule.0.scope", "Namespaced"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.rule.1.api_groups.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.rule.1.api_groups.0", "batch"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.rule.1.api_versions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.rule.1.api_versions.0", "v1beta1"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.rule.1.operations.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.rule.1.operations.0", "CREATE"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.rule.1.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.rule.1.resources.0", "cronjobs"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.rule.1.scope", "Namespaced"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.side_effects", "NoneOnDryRun"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.timeout_seconds", "5"),
				),
			},
		},
	})
}

func TestAccKubernetesValidatingWebhookConfiguration_regressionV1(t *testing.T) {
	name := fmt.Sprintf("acc-test-%v.terraform.io", acctest.RandString(10))
	resourceName := "kubernetes_validating_webhook_configuration.test"
	var conf1, conf2 admissionregistrationv1.ValidatingWebhookConfiguration

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotAdmissionRegistrationV1(t) },
		IDRefreshName:     resourceName,
		ExternalProviders: testAccExternalProviders,
		CheckDestroy:      testAccCheckKubernetesValdiatingWebhookConfigurationDestroy,
		Steps: []resource.TestStep{
			{ // The first apply downloads and uses the latest released version of the provider.
				Config: requiredProviders() + testAccKubernetesValidatingWebhookConfigurationConfig_regression("kubernetes-released", name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesValidatingWebhookConfigurationExists(resourceName),
					testAccCheckKubernetesValidatingWebhookGetObjectV1(resourceName, &conf1),
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
					resource.TestCheckResourceAttr(resourceName, "webhook.0.side_effects", "None"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.timeout_seconds", "10"),
				),
			},
			{ // The second apply uses a local, compiled version of the current branch.
				Config: requiredProviders() + testAccKubernetesValidatingWebhookConfigurationConfig_regression("kubernetes-local", name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesValidatingWebhookConfigurationExists(resourceName),
					testAccCheckKubernetesValidatingWebhookGetObjectV1(resourceName, &conf2),
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
					resource.TestCheckResourceAttr(resourceName, "webhook.0.side_effects", "None"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.timeout_seconds", "10"),
					testAccCheckKubernetesValidatingWebhookConfigV1ForceNew(&conf1, &conf2, false),
				),
			},
		},
	})
}

func TestAccKubernetesValidatingWebhookConfiguration_regressionV1Beta1(t *testing.T) {
	name := fmt.Sprintf("acc-test-%v.terraform.io", acctest.RandString(10))
	resourceName := "kubernetes_validating_webhook_configuration.test"
	var conf1, conf2 admissionregistrationv1beta1.ValidatingWebhookConfiguration

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotAdmissionRegistrationV1Beta1(t) },
		IDRefreshName:     resourceName,
		ExternalProviders: testAccExternalProviders,
		CheckDestroy:      testAccCheckKubernetesValdiatingWebhookConfigurationDestroy,
		Steps: []resource.TestStep{
			{ // The first apply downloads and uses the latest released version of the provider.
				Config: requiredProviders() + testAccKubernetesValidatingWebhookConfigurationConfig_regression("kubernetes-released", name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesValidatingWebhookConfigurationExists(resourceName),
					testAccCheckKubernetesValidatingWebhookGetObjectV1Beta1(resourceName, &conf1),
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
					resource.TestCheckResourceAttr(resourceName, "webhook.0.side_effects", "None"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.timeout_seconds", "10"),
				),
			},
			{ // The second apply uses a local, compiled version of the current branch.
				Config: requiredProviders() + testAccKubernetesValidatingWebhookConfigurationConfig_regression("kubernetes-local", name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesValidatingWebhookConfigurationExists(resourceName),
					testAccCheckKubernetesValidatingWebhookGetObjectV1Beta1(resourceName, &conf2),
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
					resource.TestCheckResourceAttr(resourceName, "webhook.0.side_effects", "None"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.timeout_seconds", "10"),
					testAccCheckKubernetesValidatingWebhookConfigV1Beta1ForceNew(&conf1, &conf2, false),
				),
			},
			{
				Config: testAccKubernetesValidatingWebhookConfigurationConfig_without_rules(name),
				Check: resource.ComposeAggregateTestCheckFunc(
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
					resource.TestCheckResourceAttr(resourceName, "webhook.0.failure_policy", "Ignore"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.match_policy", "Exact"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.namespace_selector.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.object_selector.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.side_effects", "NoneOnDryRun"),
					resource.TestCheckResourceAttr(resourceName, "webhook.0.timeout_seconds", "5"),
				),
			},
		},
	})
}

func testAccCheckKubernetesValdiatingWebhookConfigurationDestroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_validating_webhook_configuration" {
			continue
		}

		name := rs.Primary.ID

		useadmissionregistrationv1beta1, err := useAdmissionregistrationV1beta1(conn)
		if err != nil {
			return err
		}
		if useadmissionregistrationv1beta1 {
			_, err = conn.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Get(ctx, name, metav1.GetOptions{})
		} else {
			_, err = conn.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(ctx, name, metav1.GetOptions{})
		}

		if err != nil {
			if statusErr, ok := err.(*errors.StatusError); ok && errors.IsNotFound(statusErr) {
				return nil
			}
			return err
		}

		return fmt.Errorf("ValidatingWebhookConfiguration still exists: %s", rs.Primary.ID)
	}

	return nil
}

func testAccCheckKubernetesValidatingWebhookConfigurationExists(n string) resource.TestCheckFunc {
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
			_, err = conn.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Get(ctx, name, metav1.GetOptions{})
		} else {
			_, err = conn.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(ctx, name, metav1.GetOptions{})
		}
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckKubernetesValidatingWebhookGetObjectV1(n string, obj *admissionregistrationv1.ValidatingWebhookConfiguration) resource.TestCheckFunc {
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
		obj, err = conn.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(ctx, name, metav1.GetOptions{})
		return nil
	}
}

func testAccCheckKubernetesValidatingWebhookGetObjectV1Beta1(n string, obj *admissionregistrationv1beta1.ValidatingWebhookConfiguration) resource.TestCheckFunc {
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
		obj, err = conn.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Get(ctx, name, metav1.GetOptions{})
		return nil
	}
}

func testAccCheckKubernetesValidatingWebhookConfigV1ForceNew(old, new *admissionregistrationv1.ValidatingWebhookConfiguration, wantNew bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if wantNew {
			if old.ObjectMeta.UID == new.ObjectMeta.UID {
				return fmt.Errorf("Expecting new resource for ValidatingWebhookConfiguration %s", old.ObjectMeta.UID)
			}
		} else {
			if old.ObjectMeta.UID != new.ObjectMeta.UID {
				return fmt.Errorf("Expecting ValidatingWebhookConfiguration UIDs to be the same: expected %s got %s", old.ObjectMeta.UID, new.ObjectMeta.UID)
			}
		}
		return nil
	}
}

func testAccCheckKubernetesValidatingWebhookConfigV1Beta1ForceNew(old, new *admissionregistrationv1beta1.ValidatingWebhookConfiguration, wantNew bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if wantNew {
			if old.ObjectMeta.UID == new.ObjectMeta.UID {
				return fmt.Errorf("Expecting new resource for ValidatingWebhookConfiguration %s", old.ObjectMeta.UID)
			}
		} else {
			if old.ObjectMeta.UID != new.ObjectMeta.UID {
				return fmt.Errorf("Expecting ValidatingWebhookConfiguration UIDs to be the same: expected %s got %s", old.ObjectMeta.UID, new.ObjectMeta.UID)
			}
		}
		return nil
	}
}

func testAccKubernetesValidatingWebhookConfigurationConfig_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_validating_webhook_configuration" "test" {
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

    side_effects    = "None"
    timeout_seconds = 10
  }
}
`, name, name)
}

func testAccKubernetesValidatingWebhookConfigurationConfig_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_validating_webhook_configuration" "test" {
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
      service {
        namespace = "example-namespace"
        name      = "example-service"
      }

      ca_bundle = "test"
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

    side_effects    = "NoneOnDryRun"
    timeout_seconds = 5
  }
}
`, name, name)
}

func testAccKubernetesValidatingWebhookConfigurationConfig_without_rules(name string) string {
	return fmt.Sprintf(`resource "kubernetes_validating_webhook_configuration_v1" "test" {
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
      service {
        namespace = "example-namespace"
        name      = "example-service"
      }

      ca_bundle = "test"
    }

    object_selector {
      match_labels = {
        app = "test"
      }
    }

    side_effects    = "NoneOnDryRun"
    timeout_seconds = 5
  }
}
`, name, name)
}

func testAccKubernetesValidatingWebhookConfigurationConfig_regression(provider, name string) string {
	return fmt.Sprintf(`resource "kubernetes_validating_webhook_configuration" "test" {
  provider = %s
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

    side_effects    = "None"
    timeout_seconds = 10
  }
}
`, provider, name, name)
}

func skipIfNotAdmissionRegistrationV1Beta1(t *testing.T) {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		t.Fatal(err)
	}
	useadmissionregistrationv1beta1, err := useAdmissionregistrationV1beta1(conn)
	if err != nil {
		t.Fatal(err)
	}
	if !useadmissionregistrationv1beta1 {
		t.Skip("The Kubernetes endpoint is not using admissionregistrationv1beta1 - skipping.")
	}
}

func skipIfNotAdmissionRegistrationV1(t *testing.T) {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		t.Fatal(err)
	}
	useadmissionregistrationv1beta1, err := useAdmissionregistrationV1beta1(conn)
	if err != nil {
		t.Fatal(err)
	}
	if useadmissionregistrationv1beta1 {
		t.Skip("The Kubernetes endpoint is not using admissionregistrationv1 - skipping.")
	}
}
