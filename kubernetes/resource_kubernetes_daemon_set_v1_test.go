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
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccKubernetesDaemonSetV1_minimal(t *testing.T) {
	var conf appsv1.DaemonSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_daemon_set_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDaemonSetV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDaemonSetV1Config_minimal(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.name", "tf-acc-test"),
				),
			},
		},
	})
}

func TestAccKubernetesDaemonSetV1_identity(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_daemon_set_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDaemonSetV1Destroy,

		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},

		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDaemonSetV1Config_identity(name, imageName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(
						resourceName, map[string]knownvalue.Check{
							"namespace":   knownvalue.StringExact("default"),
							"name":        knownvalue.StringExact(name),
							"api_version": knownvalue.StringExact("apps/v1"),
							"kind":        knownvalue.StringExact("DaemonSet"),
						},
					),
				},
			},
			{
				ResourceName:    resourceName,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func TestAccKubernetesDaemonSetV1_basic(t *testing.T) {
	var conf appsv1.DaemonSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_daemon_set_v1.test"
	imageName := busyboxImage
	imageName1 := agnhostImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDaemonSetV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDaemonSetV1Config_basic(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetV1Exists(resourceName, &conf),
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
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.name", "tf-acc-test"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.type", "RollingUpdate"),
					resource.TestCheckResourceAttr(resourceName, "wait_for_rollout", "true"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "wait_for_rollout"},
			},
			{
				Config: testAccKubernetesDaemonSetV1Config_modified(name, imageName1),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.Different", "1234"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName1),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.name", "tf-acc-test"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.dns_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.dns_config.0.nameservers.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.dns_config.0.nameservers.0", "1.1.1.1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.dns_config.0.nameservers.1", "8.8.8.8"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.dns_config.0.nameservers.2", "9.9.9.9"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.dns_config.0.searches.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.dns_config.0.searches.0", "kubernetes.io"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.dns_config.0.option.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.dns_config.0.option.0.name", "ndots"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.dns_config.0.option.0.value", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.dns_config.0.option.1.name", "use-vc"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.dns_config.0.option.1.value", ""),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.dns_policy", "Default"),
					resource.TestCheckResourceAttr(resourceName, "wait_for_rollout", "true"),
				),
			},
		},
	})
}

func TestAccKubernetesDaemonSetV1_with_template_metadata(t *testing.T) {
	var conf appsv1.DaemonSet

	depName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_daemon_set_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,

		CheckDestroy: testAccCheckKubernetesDaemonSetV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDaemonSetV1ConfigWithTemplateMetadata(depName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.0.labels.foo", "bar"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.0.annotations.prometheus.io/scrape", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.0.annotations.prometheus.io/scheme", "https"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.0.annotations.prometheus.io/port", "4000"),
				),
			},
			{
				Config: testAccKubernetesDaemonSetV1ConfigWithTemplateMetadataModified(depName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.0.labels.foo", "bar"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.0.annotations.prometheus.io/scrape", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.0.annotations.prometheus.io/scheme", "http"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.0.annotations.prometheus.io/port", "8080"),
				),
			},
		},
	})
}

func TestAccKubernetesDaemonSetV1_initContainer(t *testing.T) {
	var conf appsv1.DaemonSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_daemon_set_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDaemonSetV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDaemonSetV1WithInitContainer(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.init_container.0.image", imageName),
				),
			},
		},
	})
}

func TestAccKubernetesDaemonSetV1_noTopLevelLabels(t *testing.T) {
	var conf appsv1.DaemonSet
	resourceName := "kubernetes_daemon_set_v1.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDaemonSetV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDaemonSetV1WithNoTopLevelLabels(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
				),
			},
		},
	})
}

func TestAccKubernetesDaemonSetV1_with_tolerations(t *testing.T) {
	var conf appsv1.DaemonSet

	rcName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_daemon_set_v1.test"
	imageName := busyboxImage
	tolerationSeconds := 6000
	operator := "Equal"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,

		CheckDestroy: testAccCheckKubernetesDaemonSetV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDaemonSetV1ConfigWithTolerations(rcName, imageName, &tolerationSeconds, operator, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.toleration.0.effect", "NoExecute"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.toleration.0.key", "myKey"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.toleration.0.operator", operator),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.toleration.0.toleration_seconds", "6000"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.toleration.0.value", ""),
				),
			},
		},
	})
}

func TestAccKubernetesDaemonSetV1_with_tolerations_unset_toleration_seconds(t *testing.T) {
	var conf appsv1.DaemonSet

	rcName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_daemon_set_v1.test"
	imageName := busyboxImage
	operator := "Equal"
	value := "value"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,

		CheckDestroy: testAccCheckKubernetesDaemonSetV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDaemonSetV1ConfigWithTolerations(rcName, imageName, nil, operator, &value),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.toleration.0.effect", "NoExecute"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.toleration.0.key", "myKey"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.toleration.0.operator", operator),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.toleration.0.value", "value"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.toleration.0.toleration_seconds", ""),
				),
			},
		},
	})
}

func TestAccKubernetesDaemonSetV1_with_container_security_context_seccomp_profile(t *testing.T) {
	var conf appsv1.DaemonSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImage
	resourceName := "kubernetes_daemon_set_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDaemonSetV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDaemonSetV1ConfigWithContainerSecurityContextSeccompProfile(name, imageName, "Unconfined"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.seccomp_profile.0.type", "Unconfined"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.security_context.0.seccomp_profile.0.type", "Unconfined"),
				),
			},
			{
				Config: testAccKubernetesDaemonSetV1ConfigWithContainerSecurityContextSeccompProfile(name, imageName, "RuntimeDefault"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.seccomp_profile.0.type", "RuntimeDefault"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.security_context.0.seccomp_profile.0.type", "RuntimeDefault"),
				),
			},
		},
	})
}

func TestAccKubernetesDaemonSetV1_with_container_security_context_seccomp_localhost_profile(t *testing.T) {
	var conf appsv1.DaemonSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_daemon_set_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t); skipIfNotRunningInKind(t); skipIfClusterVersionLessThan(t, "1.19.0") },

		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDaemonSetV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDaemonSetV1ConfigWithContainerSecurityContextSeccompProfileLocalhost(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.seccomp_profile.0.type", "Localhost"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.seccomp_profile.0.localhost_profile", "profiles/audit.json"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.security_context.0.seccomp_profile.0.type", "Localhost"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.security_context.0.seccomp_profile.0.localhost_profile", "profiles/audit.json"),
				),
			},
		},
	})
}

func TestAccKubernetesDaemonSetV1_with_resource_requirements(t *testing.T) {
	var conf appsv1.DaemonSet

	daemonSetName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_daemon_set_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDaemonSetV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDaemonSetV1ConfigWithResourceRequirements(daemonSetName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.resources.0.requests.memory", "50Mi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.resources.0.requests.cpu", "250m"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.resources.0.limits.memory", "512Mi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.resources.0.limits.cpu", "500m"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"metadata.0.resource_version",
					"wait_for_rollout",
				},
			},
			{
				Config: testAccKubernetesDaemonSetV1ConfigWithEmptyResourceRequirements(daemonSetName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.resources.0.requests.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.resources.0.limits.#", "0"),
				),
			},
			{
				Config: testAccKubernetesDaemonSetV1ConfigWithResourceRequirementsLimitsOnly(daemonSetName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.resources.0.limits.memory", "512Mi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.resources.0.limits.cpu", "500m"),
				),
			},
			{
				Config: testAccKubernetesDaemonSetV1ConfigWithResourceRequirementsRequestsOnly(daemonSetName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.resources.0.requests.memory", "512Mi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.resources.0.requests.cpu", "500m"),
				),
			},
		},
	})
}

func TestAccKubernetesDaemonSetV1_minimalWithTemplateNamespace(t *testing.T) {
	var conf1, conf2 appsv1.DaemonSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_daemon_set_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDaemonSetV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDaemonSetV1Config_minimal(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetV1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.namespace"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.0.namespace", ""),
				),
			},
			{
				Config: testAccKubernetesDaemonSetV1ConfigMinimalWithTemplateNamespace(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.namespace"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.template.0.metadata.0.namespace"),
					testAccCheckKubernetesDaemonSetV1ForceNew(&conf1, &conf2, true),
				),
			},
		},
	})
}

func TestAccKubernetesDaemonSetV1_MaxSurge(t *testing.T) {
	var conf appsv1.DaemonSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_daemon_set_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDaemonSetV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDaemonSetV1ConfigWithMaxSurge(name, imageName, "0"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.0.max_surge", "0"),
				),
			},
			{
				Config: testAccKubernetesDaemonSetV1ConfigWithMaxSurge(name, imageName, "5"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.0.max_surge", "5"),
				),
			},
			{
				Config: testAccKubernetesDaemonSetV1ConfigWithMaxSurge(name, imageName, "10"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.0.max_surge", "10"),
				),
			},
			{
				Config: testAccKubernetesDaemonSetV1ConfigWithMaxSurge(name, imageName, "100"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.0.max_surge", "100"),
				),
			},
			{
				Config: testAccKubernetesDaemonSetV1ConfigWithMaxSurge(name, imageName, "5%"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.0.max_surge", "5%"),
				),
			},
			{
				Config: testAccKubernetesDaemonSetV1ConfigWithMaxSurge(name, imageName, "10%"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.0.max_surge", "10%"),
				),
			},
			{
				Config: testAccKubernetesDaemonSetV1ConfigWithMaxSurge(name, imageName, "100%"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.0.max_surge", "100%"),
				),
			},
		},
	})
}

func testAccCheckKubernetesDaemonSetV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_daemon_set_v1" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("DaemonSet still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesDaemonSetV1Exists(n string, obj *appsv1.DaemonSet) resource.TestCheckFunc {
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
		out, err := conn.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccCheckKubernetesDaemonSetV1ForceNew(old, new *appsv1.DaemonSet, wantNew bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if wantNew {
			if old.ObjectMeta.UID == new.ObjectMeta.UID {
				return fmt.Errorf("Expecting new resource for DaemonSet %s", old.ObjectMeta.UID)
			}
		} else {
			if old.ObjectMeta.UID != new.ObjectMeta.UID {
				return fmt.Errorf("Expecting DaemonSet UIDs to be the same: expected %s got %s", old.ObjectMeta.UID, new.ObjectMeta.UID)
			}
		}
		return nil
	}
}

func testAccKubernetesDaemonSetV1ConfigWithMaxSurge(name, imageName, maxSurge string) string {
	// If maxSurge is set to 0, maxUnavailable = 1
	if maxSurge == "0" {
		return fmt.Sprintf(`resource "kubernetes_daemon_set_v1" "test" {
  metadata {
    name = "%s"
  }

  spec {
    selector {
      match_labels = {
        foo = "bar"
      }
    }

    template {
      metadata {
        labels = {
          foo = "bar"
        }
      }

      spec {
        container {
          image   = "%s"
          name    = "tf-acc-test"
          command = ["sleep", "300"]
        }
        termination_grace_period_seconds = 1
      }
    }

    strategy {
      rolling_update {
        max_surge       = "%s"
        max_unavailable = "1" # Set maxUnavailable to 1 if maxSurge is 0
      }
    }
  }
}
`, name, imageName, maxSurge)
	}

	// If maxSurge is != 0
	return fmt.Sprintf(`resource "kubernetes_daemon_set_v1" "test" {
  metadata {
    name = "%s"
  }

  spec {
    selector {
      match_labels = {
        foo = "bar"
      }
    }

    template {
      metadata {
        labels = {
          foo = "bar"
        }
      }

      spec {
        container {
          image   = "%s"
          name    = "tf-acc-test"
          command = ["sleep", "300"]
        }
        termination_grace_period_seconds = 1
      }
    }

    strategy {
      rolling_update {
        max_surge       = "%s"
        max_unavailable = "0" # Set maxUnavailable to 0 if maxSurge is set
      }
    }
  }
}
`, name, imageName, maxSurge)
}

func testAccKubernetesDaemonSetV1Config_minimal(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_daemon_set_v1" "test" {
  metadata {
    name = "%s"
  }

  spec {
    selector {
      match_labels = {
        foo = "bar"
      }
    }

    template {
      metadata {
        labels = {
          foo = "bar"
        }
      }

      spec {
        container {
          image   = "%s"
          name    = "tf-acc-test"
          command = ["sleep", "300"]
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, name, imageName)
}

func testAccKubernetesDaemonSetV1Config_identity(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_daemon_set_v1" "test" {
  metadata {
    name = "%s"
  }

  spec {
    selector {
      match_labels = {
        foo = "bar"
      }
    }

    template {
      metadata {
        labels = {
          foo = "bar"
        }
      }

      spec {
        container {
          image   = "%s"
          name    = "tf-acc-test"
          command = ["sleep", "300"]
        }
        termination_grace_period_seconds = 1
      }
    }
  }
  wait_for_rollout = false
}
`, name, imageName)
}

func testAccKubernetesDaemonSetV1Config_basic(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_daemon_set_v1" "test" {
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
    selector {
      match_labels = {
        TestLabelOne   = "one"
        TestLabelTwo   = "two"
        TestLabelThree = "three"
      }
    }

    template {
      metadata {
        labels = {
          TestLabelOne   = "one"
          TestLabelTwo   = "two"
          TestLabelThree = "three"
        }
      }

      spec {
        container {
          image   = "%s"
          name    = "tf-acc-test"
          command = ["sleep", "300"]
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, name, imageName)
}

func testAccKubernetesDaemonSetV1Config_modified(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_daemon_set_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      Different         = "1234"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
    }

    name = "%s"
  }

  spec {
    selector {
      match_labels = {
        TestLabelOne   = "one"
        TestLabelTwo   = "two"
        TestLabelThree = "three"
      }
    }

    template {
      metadata {
        labels = {
          TestLabelOne   = "one"
          TestLabelTwo   = "two"
          TestLabelThree = "three"
        }
      }

      spec {
        container {
          image   = "%s"
          name    = "tf-acc-test"
          command = ["sleep", "300"]
        }

        dns_config {
          nameservers = ["1.1.1.1", "8.8.8.8", "9.9.9.9"]
          searches    = ["kubernetes.io"]

          option {
            name  = "ndots"
            value = 1
          }

          option {
            name = "use-vc"
          }
        }
        dns_policy                       = "Default"
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, name, imageName)
}

func testAccKubernetesDaemonSetV1ConfigWithTemplateMetadata(depName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_daemon_set_v1" "test" {
  metadata {
    name = "%s"

    labels = {
      Test = "TfAcceptanceTest"
    }
  }

  spec {
    selector {
      match_labels = {
        Test = "TfAcceptanceTest"
      }
    }

    template {
      metadata {
        labels = {
          foo  = "bar"
          Test = "TfAcceptanceTest"
        }

        annotations = {
          "prometheus.io/scrape" = "true"
          "prometheus.io/scheme" = "https"
          "prometheus.io/port"   = "4000"
        }
      }

      spec {
        container {
          image   = "%s"
          name    = "containername"
          command = ["sleep", "infinity"]
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, depName, imageName)
}

func testAccKubernetesDaemonSetV1ConfigWithTemplateMetadataModified(depName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_daemon_set_v1" "test" {
  metadata {
    name = "%s"

    labels = {
      Test = "TfAcceptanceTest"
    }
  }

  spec {
    selector {
      match_labels = {
        Test = "TfAcceptanceTest"
      }
    }

    template {
      metadata {
        labels = {
          foo  = "bar"
          Test = "TfAcceptanceTest"
        }

        annotations = {
          "prometheus.io/scrape" = "true"
          "prometheus.io/scheme" = "http"
          "prometheus.io/port"   = "8080"
        }
      }

      spec {
        container {
          image = "%s"
          name  = "containername"
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, depName, imageName)
}

func testAccKubernetesDaemonSetV1WithInitContainer(depName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_daemon_set_v1" "test" {
  metadata {
    name = "%s"

    labels = {
      foo = "bar"
    }
  }

  spec {
    selector {
      match_labels = {
        foo = "bar"
      }
    }

    template {
      metadata {
        labels = {
          foo = "bar"
        }
      }

      spec {
        init_container {
          name    = "hello"
          image   = "%s"
          command = ["echo", "'hello'"]
        }

        container {
          image   = "%s"
          name    = "containername"
          command = ["sleep", "infinity"]
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, depName, imageName, imageName)
}

func testAccKubernetesDaemonSetV1WithNoTopLevelLabels(depName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_daemon_set_v1" "test" {
  metadata {
    name = "%s"
  }

  spec {
    selector {
      match_labels = {
        foo = "bar"
      }
    }

    template {
      metadata {
        labels = {
          foo = "bar"
        }
      }

      spec {
        container {
          image   = "%s"
          name    = "containername"
          command = ["sleep", "infinity"]
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, depName, imageName)
}

func testAccKubernetesDaemonSetV1ConfigWithTolerations(rcName, imageName string, tolerationSeconds *int, operator string, value *string) string {
	tolerationDuration := ""
	if tolerationSeconds != nil {
		tolerationDuration = fmt.Sprintf("toleration_seconds = %d", *tolerationSeconds)
	}
	valueString := ""
	if value != nil {
		valueString = fmt.Sprintf("value = \"%s\"", *value)
	}

	return fmt.Sprintf(`resource "kubernetes_daemon_set_v1" "test" {
  metadata {
    name = "%s"

    labels = {
      Test = "TfAcceptanceTest"
    }
  }

  spec {
    selector {
      match_labels = {
        Test = "TfAcceptanceTest"
      }
    }

    template {
      metadata {
        labels = {
          Test = "TfAcceptanceTest"
        }
      }

      spec {
        toleration {
          effect   = "NoExecute"
          key      = "myKey"
          operator = "%s"
          %s
          %s
        }

        container {
          image   = "%s"
          name    = "containername"
          command = ["sleep", "infinity"]
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, rcName, operator, valueString, tolerationDuration, imageName)
}

func testAccKubernetesDaemonSetV1ConfigWithContainerSecurityContextSeccompProfile(deploymentName, imageName, seccompProfileType string) string {
	return fmt.Sprintf(`resource "kubernetes_daemon_set_v1" "test" {
  metadata {
    name = "%s"

    labels = {
      Test = "TfAcceptanceTest"
    }
  }

  spec {
    selector {
      match_labels = {
        Test = "TfAcceptanceTest"
      }
    }

    template {
      metadata {
        labels = {
          Test = "TfAcceptanceTest"
        }
      }

      spec {
        security_context {
          seccomp_profile {
            type = "%s"
          }
        }
        container {
          image = "%s"
          name  = "containername"

          security_context {
            seccomp_profile {
              type = "%s"
            }
          }
        }
        termination_grace_period_seconds = 1
      }
    }
  }
  wait_for_rollout = false
}
`, deploymentName, seccompProfileType, imageName, seccompProfileType)
}

func testAccKubernetesDaemonSetV1ConfigWithContainerSecurityContextSeccompProfileLocalhost(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_daemon_set_v1" "test" {
  metadata {
    name = "%s"

    labels = {
      Test = "TfAcceptanceTest"
    }
  }

  spec {
    selector {
      match_labels = {
        Test = "TfAcceptanceTest"
      }
    }

    template {
      metadata {
        labels = {
          Test = "TfAcceptanceTest"
        }
      }

      spec {
        security_context {
          seccomp_profile {
            type              = "Localhost"
            localhost_profile = "profiles/audit.json"
          }
        }
        container {
          image = "%s"
          name  = "containername"

          security_context {
            seccomp_profile {
              type              = "Localhost"
              localhost_profile = "profiles/audit.json"
            }
          }
        }
        termination_grace_period_seconds = 1
      }
    }
  }
  wait_for_rollout = false
}
`, deploymentName, imageName)
}

func testAccKubernetesDaemonSetV1ConfigWithResourceRequirements(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_daemon_set_v1" "test" {
  metadata {
    name = "%s"

    labels = {
      Test = "TfAcceptanceTest"
    }
  }

  spec {
    selector {
      match_labels = {
        Test = "TfAcceptanceTest"
      }
    }

    template {
      metadata {
        labels = {
          Test = "TfAcceptanceTest"
        }
      }

      spec {
        container {
          image   = "%s"
          name    = "containername"
          command = ["sleep", "infinity"]

          resources {
            limits = {
              cpu    = "0.5"
              memory = "512Mi"
            }

            requests = {
              cpu    = "250m"
              memory = "50Mi"
            }
          }
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, deploymentName, imageName)
}

func testAccKubernetesDaemonSetV1ConfigWithEmptyResourceRequirements(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_daemon_set_v1" "test" {
  metadata {
    name = "%s"

    labels = {
      Test = "TfAcceptanceTest"
    }
  }

  spec {
    selector {
      match_labels = {
        Test = "TfAcceptanceTest"
      }
    }

    template {
      metadata {
        labels = {
          Test = "TfAcceptanceTest"
        }
      }

      spec {
        container {
          image   = "%s"
          name    = "containername"
          command = ["sleep", "infinity"]

          resources {
            limits   = {}
            requests = {}
          }
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, deploymentName, imageName)
}

func testAccKubernetesDaemonSetV1ConfigWithResourceRequirementsLimitsOnly(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_daemon_set_v1" "test" {
  metadata {
    name = "%s"

    labels = {
      Test = "TfAcceptanceTest"
    }
  }

  spec {
    selector {
      match_labels = {
        Test = "TfAcceptanceTest"
      }
    }

    template {
      metadata {
        labels = {
          Test = "TfAcceptanceTest"
        }
      }

      spec {
        container {
          image   = "%s"
          name    = "containername"
          command = ["sleep", "infinity"]

          resources {
            limits = {
              cpu    = "500m"
              memory = "512Mi"
            }
          }
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, deploymentName, imageName)
}

func testAccKubernetesDaemonSetV1ConfigWithResourceRequirementsRequestsOnly(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_daemon_set_v1" "test" {
  metadata {
    name = "%s"

    labels = {
      Test = "TfAcceptanceTest"
    }
  }

  spec {
    selector {
      match_labels = {
        Test = "TfAcceptanceTest"
      }
    }

    template {
      metadata {
        labels = {
          Test = "TfAcceptanceTest"
        }
      }

      spec {
        container {
          image = "%s"
          name  = "containername"

          resources {
            requests = {
              cpu    = "500m"
              memory = "512Mi"
            }
          }
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, deploymentName, imageName)
}

func testAccKubernetesDaemonSetV1ConfigMinimalWithTemplateNamespace(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_daemon_set_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    selector {
      match_labels = {
        foo = "bar"
      }
    }
    template {
      metadata {
        // The namespace field is just a stub and does not influence where the Pod will be created.
        // The Pod will be created within the same Namespace as the Daemon Set resource.
        namespace = "fake" // Doesn't have to exist.
        labels = {
          foo = "bar"
        }
      }
      spec {
        container {
          image   = "%s"
          name    = "tf-acc-test"
          command = ["sleep", "300"]
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, name, imageName)
}
