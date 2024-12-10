// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesDeploymentV1_minimal(t *testing.T) {
	var conf appsv1.Deployment
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_deployment_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1Config_minimal(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
				),
			},
			{
				Config:   testAccKubernetesDeploymentV1Config_minimal(name, imageName),
				PlanOnly: true,
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_basic(t *testing.T) {
	var conf appsv1.Deployment
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_deployment_v1.test"
	imageName := agnhostImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1Config_basic(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
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
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.0.max_surge", "25%"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.0.max_unavailable", "25%"),
					resource.TestCheckResourceAttr(resourceName, "wait_for_rollout", "true"),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_initContainerForceNew(t *testing.T) {
	var conf1, conf2 appsv1.Deployment
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	namespace := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_deployment_v1.test"
	imageName := busyboxImage
	imageName1 := agnhostImage
	initCommand := "until nslookup " + name + "-init-service." + namespace + ".svc.cluster.local; do echo waiting for init-service; sleep 2; done"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesDeploymentV1Config_initContainer(
						namespace, name, imageName, imageName1, "64Mi", "testvar",
						"initcontainer2", initCommand, "IfNotPresent"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.init_container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.init_container.0.resources.0.requests.memory", "64Mi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.init_container.0.env.2.value", "testvar"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.init_container.1.name", "initcontainer2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.init_container.1.image", imageName1),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.init_container.1.command.2", initCommand),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.init_container.1.image_pull_policy", "IfNotPresent"),
				),
			},
			{ // Test for non-empty plans. No modification.
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesDeploymentV1Config_initContainer(
						namespace, name, imageName, imageName1, "64Mi", "testvar",
						"initcontainer2", initCommand, "IfNotPresent"),
				PlanOnly: true,
			},
			{ // Modify resources.limits.memory.
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesDeploymentV1Config_initContainer(
						namespace, name, imageName, imageName1, "80Mi", "testvar",
						"initcontainer2", initCommand, "IfNotPresent"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.init_container.0.resources.0.requests.memory", "80Mi"),
					testAccCheckKubernetesDeploymentForceNew(&conf1, &conf2, false),
				),
			},
			{ // Modify name of environment variable.
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesDeploymentV1Config_initContainer(
						namespace, name, imageName, imageName1, "64Mi", "testvar",
						"initcontainer2", initCommand, "IfNotPresent"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.init_container.0.env.2.value", "testvar"),
					testAccCheckKubernetesDeploymentForceNew(&conf1, &conf2, false),
				),
			},
			{ // Modify init_container's command.
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesDeploymentV1Config_initContainer(
						namespace, name, imageName, imageName1, "64Mi", "testvar",
						"initcontainer2", "echo done", "IfNotPresent"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.init_container.1.command.2", "echo done"),
					testAccCheckKubernetesDeploymentForceNew(&conf1, &conf2, false),
				),
			},
			{ // Modify init_container's image_pull_policy.
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesDeploymentV1Config_initContainer(
						namespace, name, imageName, imageName1, "64Mi", "testvar",
						"initcontainer2", "echo done", "Never"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.init_container.1.image_pull_policy", "Never"),
					testAccCheckKubernetesDeploymentForceNew(&conf1, &conf2, false),
				),
			},
			{ // Modify init_container's image
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesDeploymentV1Config_initContainer(
						namespace, name, imageName, imageName, "64Mi", "testvar",
						"initcontainer2", "echo done", "Never"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.init_container.1.image", imageName),
					testAccCheckKubernetesDeploymentForceNew(&conf1, &conf2, false),
				),
			},
			{ // Modify init_container's name.
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesDeploymentV1Config_initContainer(
						namespace, name, imageName, imageName, "64Mi", "testvar",
						"initcontainertwo", "echo done", "Never"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.init_container.1.name", "initcontainertwo"),
					testAccCheckKubernetesDeploymentForceNew(&conf1, &conf2, false),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_generatedName(t *testing.T) {
	var conf appsv1.Deployment
	prefix := "tf-acc-test-gen-"
	resourceName := "kubernetes_deployment_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1Config_generatedName(prefix, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr(resourceName, "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "wait_for_rollout"},
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_with_security_context(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_deployment_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1ConfigWithSecurityContext(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.fs_group", "100"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.run_as_non_root", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.run_as_user", "101"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.supplemental_groups.0", "101"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.fs_group", "100"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.sysctl.#", "0"),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_with_security_context_run_as_group(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_deployment_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfUnsupportedSecurityContextRunAsGroup(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1ConfigWithSecurityContextRunAsGroup(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.fs_group", "100"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.run_as_group", "100"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.run_as_non_root", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.run_as_user", "101"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.supplemental_groups.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.supplemental_groups.0", "101"),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_with_security_context_sysctl(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_deployment_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1ConfigWithSecurityContextSysctl(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.fs_group", "100"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.run_as_non_root", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.run_as_user", "101"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.supplemental_groups.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.supplemental_groups.0", "101"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.sysctl.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.sysctl.0.name", "kernel.shm_rmid_forced"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.sysctl.0.value", "0"),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_with_tolerations(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_deployment_v1.test"
	imageName := busyboxImage
	tolerationSeconds := 6000
	operator := "Equal"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1ConfigWithTolerations(deploymentName, imageName, &tolerationSeconds, operator, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
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

func TestAccKubernetesDeploymentV1_with_tolerations_unset_toleration_seconds(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_deployment_v1.test"
	imageName := busyboxImage
	operator := "Equal"
	value := "value"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1ConfigWithTolerations(deploymentName, imageName, nil, operator, &value),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
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

func TestAccKubernetesDeploymentV1_with_container_liveness_probe_using_exec(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_deployment_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1ConfigWithLivenessProbeUsingExec(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.args.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.exec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.exec.0.command.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.exec.0.command.0", "cat"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.exec.0.command.1", "/tmp/healthy"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.failure_threshold", "3"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.initial_delay_seconds", "3"),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_with_container_liveness_probe_using_http_get(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_deployment_v1.test"
	imageName := agnhostImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1ConfigWithLivenessProbeUsingHTTPGet(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.args.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.http_get.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.http_get.0.path", "/healthz"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.http_get.0.port", "8080"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.http_get.0.http_header.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.http_get.0.http_header.0.name", "X-Custom-Header"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.http_get.0.http_header.0.value", "Awesome"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.initial_delay_seconds", "3"),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_with_container_liveness_probe_using_tcp(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_deployment_v1.test"
	imageName := agnhostImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1ConfigWithLivenessProbeUsingTCP(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.args.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.tcp_socket.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.tcp_socket.0.port", "8080"),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_with_container_lifecycle(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_deployment_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1ConfigWithLifeCycle(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.lifecycle.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.lifecycle.0.post_start.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.lifecycle.0.post_start.0.exec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.lifecycle.0.post_start.0.exec.0.command.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.lifecycle.0.post_start.0.exec.0.command.0", "ls"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.lifecycle.0.post_start.0.exec.0.command.1", "-al"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.lifecycle.0.pre_stop.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.lifecycle.0.pre_stop.0.exec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.lifecycle.0.pre_stop.0.exec.0.command.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.lifecycle.0.pre_stop.0.exec.0.command.0", "date"),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_with_container_security_context(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_deployment_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1ConfigWithContainerSecurityContext(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.security_context.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.security_context.0.capabilities.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.security_context.0.privileged", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.security_context.0.se_linux_options.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.security_context.0.se_linux_options.0.level", "s0:c123,c456"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.1.security_context.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.1.security_context.0.allow_privilege_escalation", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.1.security_context.0.capabilities.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.1.security_context.0.capabilities.0.add.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.1.security_context.0.capabilities.0.add.0", "NET_BIND_SERVICE"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.1.security_context.0.capabilities.0.drop.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.1.security_context.0.capabilities.0.drop.0", "all"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.1.security_context.0.privileged", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.1.security_context.0.read_only_root_filesystem", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.1.security_context.0.run_as_non_root", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.1.security_context.0.run_as_user", "201"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.1.security_context.0.se_linux_options.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.1.security_context.0.se_linux_options.0.level", "s0:c123,c789"),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_with_container_security_context_run_as_group(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_deployment_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfUnsupportedSecurityContextRunAsGroup(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1ConfigWithContainerSecurityContextRunAsGroup(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.security_context.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.security_context.0.privileged", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.security_context.0.se_linux_options.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.security_context.0.se_linux_options.0.level", "s0:c123,c456"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.1.security_context.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.1.security_context.0.allow_privilege_escalation", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.1.security_context.0.privileged", "false"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.1.security_context.0.run_as_group", "200"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.1.security_context.0.run_as_non_root", "false"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.1.security_context.0.run_as_user", "201"),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_with_container_security_context_seccomp_profile(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImage
	resourceName := "kubernetes_deployment_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfClusterVersionLessThan(t, "1.19.0") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1ConfigWithContainerSecurityContextSeccompProfile(deploymentName, imageName, "Unconfined"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.seccomp_profile.0.type", "Unconfined"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.security_context.0.seccomp_profile.0.type", "Unconfined"),
				),
			},
			{
				Config: testAccKubernetesDeploymentV1ConfigWithContainerSecurityContextSeccompProfile(deploymentName, imageName, "RuntimeDefault"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.seccomp_profile.0.type", "RuntimeDefault"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.security_context.0.seccomp_profile.0.type", "RuntimeDefault"),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_with_container_security_context_seccomp_localhost_profile(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImage
	resourceName := "kubernetes_deployment_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotRunningInKind(t); skipIfClusterVersionLessThan(t, "1.19.0") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1ConfigWithContainerSecurityContextSeccompProfileLocalhost(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.seccomp_profile.0.type", "Localhost"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.seccomp_profile.0.localhost_profile", "profiles/audit.json"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.security_context.0.seccomp_profile.0.type", "Localhost"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.security_context.0.seccomp_profile.0.localhost_profile", "profiles/audit.json"),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_with_volume_mount(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	secretName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_deployment_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1ConfigWithVolumeMounts(secretName, deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.mount_path", "/tmp/my_path"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.name", "db"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.read_only", "false"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.sub_path", ""),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.sub_path_expr", ""),
				),
			},
			{
				Config: testAccKubernetesDeploymentV1ConfigWithVolumeMountsNone(secretName, deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.#", "0"),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_ForceNew(t *testing.T) {
	var conf1, conf2 appsv1.Deployment
	resourceName := "kubernetes_deployment_v1.test"
	imageName := busyboxImage
	imageName1 := agnhostImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1Config_ForceNew("secret1", "label1", "deployment1", imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.app", "label1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.0.match_labels.app", "label1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.0.labels.app", "label1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.volume.0.secret.0.secret_name", "secret1"),
				),
			},
			{
				Config: testAccKubernetesDeploymentV1Config_ForceNew("secret2", "label1", "deployment1", imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.app", "label1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.0.match_labels.app", "label1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.0.labels.app", "label1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.volume.0.secret.0.secret_name", "secret2"),
					testAccCheckKubernetesDeploymentForceNew(&conf1, &conf2, false),
				),
			},
			{ // BUG: labels cannot be updated on a deployment without triggering a ForceNew.
				Config: testAccKubernetesDeploymentV1Config_ForceNew("secret2", "label2", "deployment1", imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.app", "label2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.0.match_labels.app", "label2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.0.labels.app", "label2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.volume.0.secret.0.secret_name", "secret2"),
					//					testAccCheckKubernetesDeploymentForceNew(&conf1, &conf2, false),
				),
			},
			{
				Config: testAccKubernetesDeploymentV1Config_ForceNew("secret2", "label2", "deployment1", imageName1),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.app", "label2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.0.match_labels.app", "label2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.0.labels.app", "label2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName1),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.volume.0.secret.0.secret_name", "secret2"),
					testAccCheckKubernetesDeploymentForceNew(&conf1, &conf2, true),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_with_resource_requirements(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImage
	resourceName := "kubernetes_deployment_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1ConfigWithResourceRequirements(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.resources.0.requests.memory", "50Mi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.resources.0.requests.cpu", "250m"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.resources.0.requests.nvidia/gpu", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.resources.0.limits.memory", "512Mi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.resources.0.limits.cpu", "500m"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.resources.0.limits.nvidia/gpu", "1"),
				),
			},
			{
				Config: testAccKubernetesDeploymentV1ConfigWithEmptyResourceRequirements(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.resources.0.requests.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.resources.0.limits.#", "0"),
				),
			},
			{
				Config: testAccKubernetesDeploymentV1ConfigWithResourceRequirementsLimitsOnly(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.resources.0.limits.memory", "512Mi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.resources.0.limits.cpu", "500m"),
				),
			},
			{
				Config: testAccKubernetesDeploymentV1ConfigWithResourceRequirementsRequestsOnly(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.resources.0.requests.memory", "512Mi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.resources.0.requests.cpu", "500m"),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_with_empty_dir_volume(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_deployment_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1ConfigWithEmptyDirVolumes(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.mount_path", "/cache"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.name", "cache-volume"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.volume.0.empty_dir.0.medium", "Memory"),
				),
			},
			{
				Config: testAccKubernetesDeploymentV1ConfigWithEmptyDirVolumesModified(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.mount_path", "/cache"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.name", "cache-volume"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.volume.0.empty_dir.0.medium", "Memory"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.volume.0.empty_dir.0.size_limit", "128Mi"),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_with_empty_dir_huge_page(t *testing.T) {
	var conf appsv1.Deployment

	imageName := busyboxImage
	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_deployment_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1ConfigWithEmptyDirHugePage(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.mount_path", "/cache"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.name", "cache-volume"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.volume.0.empty_dir.0.medium", "HugePages-1Gi"),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentV1Update_basic(t *testing.T) {
	var conf1, conf2 appsv1.Deployment
	resourceName := "kubernetes_deployment_v1.test"
	imageName := agnhostImage
	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1Config_basic(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf1),
					// Not to be changed
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					// To be removed
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					// To be added
					resource.TestCheckNoResourceAttr(resourceName, "metadata.0.annotations.Different"),
					// To be changed
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
				),
			},
			{
				Config: testAccKubernetesDeploymentV1Config_modified(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf2),
					// Unchanged
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					// Removed
					resource.TestCheckNoResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationTwo"),
					// Added
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.Different", "1234"),
					// Changed
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					testAccCheckKubernetesDeploymentForceNew(&conf1, &conf2, false),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_with_deployment_strategy_rollingupdate(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImage
	resourceName := "kubernetes_deployment_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1ConfigWithDeploymentStrategy(deploymentName, "RollingUpdate", imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.type", "RollingUpdate"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.0.max_surge", "25%"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.0.max_unavailable", "25%"),
				),
			},
			{
				Config: testAccKubernetesDeploymentV1ConfigWithDeploymentStrategy(deploymentName, "Recreate", imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.type", "Recreate"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.#", "0"),
				),
			},
			{
				Config: testAccKubernetesDeploymentV1ConfigWithDeploymentStrategy(deploymentName, "RollingUpdate", imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.type", "RollingUpdate"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.0.max_surge", "25%"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.0.max_unavailable", "25%"),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_with_share_process_namespace(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_deployment_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1ConfigWithShareProcessNamespace(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.share_process_namespace", "true"),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_with_deployment_strategy_rollingupdate_max_surge_30perc_max_unavailable_40perc(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_deployment_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1ConfigWithDeploymentStrategyRollingUpdate(deploymentName, "30%", "40%", imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.type", "RollingUpdate"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.0.max_surge", "30%"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.0.max_unavailable", "40%"),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_with_deployment_strategy_rollingupdate_max_surge_200perc_max_unavailable_0perc(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_deployment_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1ConfigWithDeploymentStrategyRollingUpdate(deploymentName, "200%", "0%", imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.type", "RollingUpdate"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.0.max_surge", "200%"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.0.max_unavailable", "0%"),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_with_deployment_strategy_rollingupdate_max_surge_0_max_unavailable_1(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_deployment_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1ConfigWithDeploymentStrategyRollingUpdate(deploymentName, "0", "1", imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.type", "RollingUpdate"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.0.max_surge", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.0.max_unavailable", "1"),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_with_deployment_strategy_rollingupdate_max_surge_1_max_unavailable_0(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_deployment_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1ConfigWithDeploymentStrategyRollingUpdate(deploymentName, "1", "0", imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.type", "RollingUpdate"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.0.max_surge", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.0.max_unavailable", "0"),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_with_deployment_strategy_rollingupdate_max_surge_1_max_unavailable_2(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_deployment_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1ConfigWithDeploymentStrategyRollingUpdate(deploymentName, "1", "2", imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.type", "RollingUpdate"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.0.max_surge", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.0.max_unavailable", "2"),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_with_deployment_strategy_recreate(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_deployment_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1ConfigWithDeploymentStrategy(deploymentName, "Recreate", imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.type", "Recreate"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.#", "0"),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_with_host_aliases(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_deployment_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1ConfigHostAliases(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.host_aliases.0.hostnames.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.host_aliases.0.hostnames.0", "abc.com"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.host_aliases.0.hostnames.1", "contoso.com"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.host_aliases.0.ip", "127.0.0.5"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.host_aliases.1.hostnames.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.host_aliases.1.hostnames.0", "xyz.com"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.host_aliases.1.ip", "127.0.0.6"),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_with_resource_field_selector(t *testing.T) {
	var conf appsv1.Deployment
	rcName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_deployment_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				ExpectError: regexp.MustCompile("quantities must match the regular expression"),
				Config:      testAccKubernetesDeploymentV1ConfigWithResourceFieldSelector(rcName, imageName, "limits.cpu", ""),
			},
			{
				ExpectError: regexp.MustCompile("only divisor's values 1m and 1 are supported with the cpu resource"),
				Config:      testAccKubernetesDeploymentV1ConfigWithResourceFieldSelector(rcName, imageName, "limits.cpu", "2"),
			},
			{
				Config: testAccKubernetesDeploymentV1ConfigWithResourceFieldSelector(rcName, imageName, "limits.cpu", "1m"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.resources.0.limits.memory", "512Mi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.env.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.env.0.name", "K8S_LIMITS_CPU"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.env.0.value_from.0.resource_field_ref.0.container_name", "containername"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.env.0.value_from.0.resource_field_ref.0.divisor", "1m"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.env.0.value_from.0.resource_field_ref.0.resource", "limits.cpu"),
				),
			},
			{
				Config: testAccKubernetesDeploymentV1ConfigWithResourceFieldSelector(rcName, imageName, "limits.memory", "1Mi"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.env.0.value_from.0.resource_field_ref.0.divisor", "1Mi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.env.0.value_from.0.resource_field_ref.0.resource", "limits.memory"),
				),
			},
			{
				Config: testAccKubernetesDeploymentV1ConfigWithResourceFieldSelector(rcName, imageName, "requests.memory", "1Ki"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.env.0.value_from.0.resource_field_ref.0.divisor", "1Ki"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.env.0.value_from.0.resource_field_ref.0.resource", "requests.memory"),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_config_with_automount_service_account_token(t *testing.T) {
	var confDeployment appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_deployment_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentV1ConfigWithAutomountServiceAccountToken(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &confDeployment),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.automount_service_account_token", "true"),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_with_restart_policy(t *testing.T) {
	var conf appsv1.Deployment
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_deployment_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccKubernetesDeploymentV1Config_with_restart_policy(name, imageName, "Never"),
				ExpectError: regexp.MustCompile("expected spec\\.0\\.template\\.0\\.spec\\.0\\.restart_policy to be one of \\[\"Always\"\\], got Never"),
			},
			{
				Config: testAccKubernetesDeploymentV1Config_with_restart_policy(name, imageName, "Always"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
				),
			},
		},
	})
}

func testAccCheckKubernetesDeploymentForceNew(old, new *appsv1.Deployment, wantNew bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if wantNew {
			if old.ObjectMeta.UID == new.ObjectMeta.UID {
				return fmt.Errorf("Expecting new resource for Deployment %s", old.ObjectMeta.UID)
			}
		} else {
			if old.ObjectMeta.UID != new.ObjectMeta.UID {
				return fmt.Errorf("Expecting Deployment UIDs to be the same: expected %s got %s", old.ObjectMeta.UID, new.ObjectMeta.UID)
			}
		}
		return nil
	}
}

func testAccCheckKubernetesDeploymentV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_deployment_v1" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Deployment still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func getDeploymentFromResourceName(s *terraform.State, n string) (*appsv1.Deployment, error) {
	rs, ok := s.RootModule().Resources[n]
	if !ok {
		return nil, fmt.Errorf("Not found: %s", n)
	}

	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return nil, err
	}
	ctx := context.TODO()

	namespace, name, err := idParts(rs.Primary.ID)
	if err != nil {
		return nil, err
	}

	out, err := conn.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return out, nil
}

func testAccCheckKubernetesDeploymentV1Exists(n string, obj *appsv1.Deployment) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		d, err := getDeploymentFromResourceName(s, n)
		if err != nil {
			return err
		}
		*obj = *d
		return nil
	}
}

func testAccKubernetesDeploymentV1Config_minimal(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    replicas = 2
    selector {
      match_labels = {
        TestLabelOne = "one"
      }
    }
    template {
      metadata {
        labels = {
          TestLabelOne = "one"
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

func testAccKubernetesDeploymentV1Config_basic(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
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
    replicas = 2

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
          image = "%s"
          name  = "tf-acc-test"
          args  = ["test-webserver"]

          port {
            container_port = 80
          }

          readiness_probe {
            initial_delay_seconds = 3
            period_seconds        = 1
            http_get {
              path = "/"
              port = 80
            }
          }

          resources {
            requests = {
              memory = "64Mi"
              cpu    = "50m"
            }
          }
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, name, imageName)
}

func testAccKubernetesDeploymentV1Config_with_restart_policy(name, imageName, restartPolicy string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    replicas = 2
    selector {
      match_labels = {
        TestLabelOne = "one"
      }
    }
    template {
      metadata {
        labels = {
          TestLabelOne = "one"
        }
      }
      spec {
        container {
          image   = "%s"
          name    = "tf-acc-test"
          command = ["sleep", "300"]
        }
        restart_policy                   = "%s"
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, name, imageName, restartPolicy)
}

func testAccKubernetesDeploymentV1Config_initContainer(namespace, name, imageName, imageName1, memory, envName, initName, initCommand, pullPolicy string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = "%s"
  }
}

resource "kubernetes_deployment_v1" "test" {
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
    namespace = kubernetes_namespace_v1.test.metadata.0.name
    name      = "%s"
  }
  spec {
    replicas = 1
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
          name    = "regularcontainer"
          image   = "%s"
          command = ["sleep"]
          args    = ["30s"]
        }
        init_container {
          name              = "initcontainer1"
          image             = "%s"
          image_pull_policy = "IfNotPresent"
          command = [
            "sh",
            "-c",
            "printenv SECRETENV CONFIGENV LIMITS_CPU CUSTOM",
          ]
          env {
            name = "SECRETENV"
            value_from {
              secret_key_ref {
                name = kubernetes_secret_v1.test.metadata.0.name
                key  = "SECRETENV"
              }
            }
          }
          resources {
            requests = {
              memory = "%s"
              cpu    = "50m"
            }
            limits = {
              memory = "100Mi"
              cpu    = "100m"
            }
          }
          env {
            name = "LIMITS_CPU"
            value_from {
              resource_field_ref {
                container_name = "initcontainer1"
                resource       = "requests.cpu"
                divisor        = "1m"
              }
            }
          }
          env_from {
            config_map_ref {
              name = kubernetes_config_map_v1.test.metadata.0.name
            }
            prefix = "CONFIG"
          }
          port {
            container_port = 80
          }
          env {
            name  = "CUSTOM"
            value = "%s"
          }
        }
        init_container {
          name              = "%s"
          image             = "%s"
          image_pull_policy = "%s"
          command = [
            "sh",
            "-c",
            "%s",
          ]
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}

resource "kubernetes_service_v1" "test" {
  metadata {
    name      = "%s-init-service"
    namespace = kubernetes_namespace_v1.test.metadata.0.name
    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }
  }
  spec {
    port {
      port        = 8080
      target_port = 80
    }
  }
}

resource "kubernetes_secret_v1" "test" {
  metadata {
    name      = "%s-test"
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
  data = {
    "SECRETENV" = "asdf1234"
  }
}


resource "kubernetes_config_map_v1" "test" {
  metadata {
    name      = "%s-test"
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
  data = {
    "ENV" = "somedata"
  }
}
`, namespace, name, imageName, imageName, memory, envName, initName, imageName1, pullPolicy, initCommand, name, name, name)
}

func testAccKubernetesDeploymentV1Config_modified(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      Different         = "1234"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
    }

    name = %q
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
          image = "%s"
          name  = "tf-acc-test"
          args  = ["test-webserver"]
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, name, imageName)
}

func testAccKubernetesDeploymentV1Config_generatedName(prefix, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
  metadata {
    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    generate_name = "%s"
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
          image   = %q
          name    = "tf-acc-test"
          command = ["sleep", "300"]
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, prefix, imageName)
}

func testAccKubernetesDeploymentV1ConfigWithSecurityContext(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
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
          fs_group            = 100
          run_as_non_root     = true
          run_as_user         = 101
          supplemental_groups = [101]
        }

        container {
          image   = "%s"
          name    = "containername"
          command = ["sleep", "300"]
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, deploymentName, imageName)
}

func testAccKubernetesDeploymentV1ConfigWithSecurityContextRunAsGroup(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
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
          fs_group            = 100
          run_as_group        = 100
          run_as_non_root     = true
          run_as_user         = 101
          supplemental_groups = [101]
        }

        container {
          image   = "%s"
          name    = "containername"
          command = ["sleep", "300"]
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, deploymentName, imageName)
}

func testAccKubernetesDeploymentV1ConfigWithSecurityContextSysctl(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
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
          fs_group            = 100
          run_as_non_root     = true
          run_as_user         = 101
          supplemental_groups = [101]

          sysctl {
            name  = "kernel.shm_rmid_forced"
            value = "0"
          }
        }

        container {
          image   = "%s"
          name    = "containername"
          command = ["sleep", "300"]
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, deploymentName, imageName)
}

func testAccKubernetesDeploymentV1ConfigWithTolerations(deploymentName, imageName string, tolerationSeconds *int, operator string, value *string) string {
	tolerationDuration := ""
	if tolerationSeconds != nil {
		tolerationDuration = fmt.Sprintf("toleration_seconds = %d", *tolerationSeconds)
	}
	valueString := ""
	if value != nil {
		valueString = fmt.Sprintf("value = \"%s\"", *value)
	}

	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
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
          command = ["sleep", "300"]
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, deploymentName, operator, valueString, tolerationDuration, imageName)
}

func testAccKubernetesDeploymentV1ConfigWithLivenessProbeUsingExec(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
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
          args  = ["/bin/sh", "-c", "touch /tmp/healthy; sleep 300; rm -rf /tmp/healthy; sleep 600"]

          liveness_probe {
            exec {
              command = ["cat", "/tmp/healthy"]
            }
            initial_delay_seconds = 3
            period_seconds        = 1
          }
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, deploymentName, imageName)
}

func testAccKubernetesDeploymentV1ConfigWithLivenessProbeUsingHTTPGet(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
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
          args  = ["liveness"]

          liveness_probe {
            http_get {
              path = "/healthz"
              port = 8080

              http_header {
                name  = "X-Custom-Header"
                value = "Awesome"
              }
            }
            initial_delay_seconds = 3
            period_seconds        = 1
          }
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, deploymentName, imageName)
}

func testAccKubernetesDeploymentV1ConfigWithLivenessProbeUsingTCP(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
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
          args  = ["liveness"]

          liveness_probe {
            tcp_socket {
              port = 8080
            }
            initial_delay_seconds = 3
            period_seconds        = 1
          }
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, deploymentName, imageName)
}

func testAccKubernetesDeploymentV1ConfigWithLifeCycle(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
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
          command = ["sleep", "60"]

          lifecycle {
            post_start {
              exec {
                command = ["ls", "-al"]
              }
            }

            pre_stop {
              exec {
                command = ["date"]
              }
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

func testAccKubernetesDeploymentV1ConfigWithContainerSecurityContext(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
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
          image   = %[2]q
          name    = "containername"
          command = ["sleep", "300"]

          security_context {
            privileged  = true
            run_as_user = 1

            se_linux_options {
              level = "s0:c123,c456"
            }
          }
        }

        container {
          image   = %[2]q
          name    = "containername2"
          command = ["sleep", "300"]

          security_context {
            allow_privilege_escalation = true

            capabilities {
              drop = ["all"]
              add  = ["NET_BIND_SERVICE"]
            }

            privileged                = true
            read_only_root_filesystem = true
            run_as_non_root           = true
            run_as_user               = 201

            se_linux_options {
              level = "s0:c123,c789"
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

func testAccKubernetesDeploymentV1ConfigWithContainerSecurityContextRunAsGroup(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
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
          command = ["sleep", "300"]

          security_context {
            privileged  = true
            run_as_user = 1

            se_linux_options {
              level = "s0:c123,c456"
            }
          }
        }

        container {
          name    = "container2"
          image   = "%s"
          command = ["sh", "-c", "echo The app is running! && sleep 300"]
          security_context {
            run_as_group = 200
            run_as_user  = 201
          }
        }

        termination_grace_period_seconds = 1
      }
    }
  }
}
`, deploymentName, imageName, imageName)
}

func testAccKubernetesDeploymentV1ConfigWithContainerSecurityContextSeccompProfile(deploymentName, imageName, seccompProfileType string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
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
          image   = "%s"
          name    = "containername"
          command = ["sleep", "300"]

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
}
`, deploymentName, seccompProfileType, imageName, seccompProfileType)
}

func testAccKubernetesDeploymentV1ConfigWithContainerSecurityContextSeccompProfileLocalhost(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
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
          image   = "%s"
          name    = "containername"
          command = ["sleep", "300"]

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
}
`, deploymentName, imageName)
}

func testAccKubernetesDeploymentV1ConfigWithVolumeMounts(secretName, deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "first"
  }
}

resource "kubernetes_deployment_v1" "test" {
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
          command = ["sleep", "300"]

          volume_mount {
            mount_path = "/tmp/my_path"
            name       = "db"
          }
        }

        volume {
          name = "db"

          secret {
            secret_name = "${kubernetes_secret_v1.test.metadata.0.name}"
          }
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, secretName, deploymentName, imageName)
}

func testAccKubernetesDeploymentV1ConfigWithVolumeMountsNone(secretName, deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "first"
  }
}

resource "kubernetes_deployment_v1" "test" {
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
          command = ["sleep", "300"]
        }

        volume {
          name = "db"

          secret {
            secret_name = "${kubernetes_secret_v1.test.metadata.0.name}"
          }
        }

        termination_grace_period_seconds = 1
      }
    }
  }
}
`, secretName, deploymentName, imageName)
}

func testAccKubernetesDeploymentV1ConfigWithResourceRequirements(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
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
            limits = {
              cpu          = "0.5"
              memory       = "512Mi"
              "nvidia/gpu" = "1"
            }

            requests = {
              cpu          = "250m"
              memory       = "50Mi"
              "nvidia/gpu" = "1"
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

func testAccKubernetesDeploymentV1ConfigWithEmptyResourceRequirements(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
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
            limits   = {}
            requests = {}
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

func testAccKubernetesDeploymentV1ConfigWithResourceRequirementsLimitsOnly(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
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

  wait_for_rollout = false
}
`, deploymentName, imageName)
}

func testAccKubernetesDeploymentV1ConfigWithResourceRequirementsRequestsOnly(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
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

  wait_for_rollout = false
}
`, deploymentName, imageName)
}

func testAccKubernetesDeploymentV1ConfigWithEmptyDirVolumes(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
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
          command = ["sleep", "300"]

          volume_mount {
            mount_path = "/cache"
            name       = "cache-volume"
          }
        }

        volume {
          name = "cache-volume"

          empty_dir {
            medium = "Memory"
          }
        }

        termination_grace_period_seconds = 1
      }
    }
  }
}
`, deploymentName, imageName)
}

func testAccKubernetesDeploymentV1ConfigWithEmptyDirHugePage(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
  metadata {
    name = "%s"

    labels = {
      Test = "TfAcceptanceTest"
    }
  }

  spec {
    replicas = 0 # We request zero replicas, since the K8S backing this test may not have huge pages available.
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
          command = ["sleep", "300"]

          volume_mount {
            mount_path = "/cache"
            name       = "cache-volume"
          }
        }

        volume {
          name = "cache-volume"

          empty_dir {
            medium = "HugePages-1Gi"
          }
        }

        termination_grace_period_seconds = 1
      }
    }
  }
}
`, deploymentName, imageName)
}

func testAccKubernetesDeploymentV1ConfigWithEmptyDirVolumesModified(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
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
          command = ["sleep", "300"]

          volume_mount {
            mount_path = "/cache"
            name       = "cache-volume"
          }
        }

        volume {
          name = "cache-volume"

          empty_dir {
            medium     = "Memory"
            size_limit = "128Mi"
          }
        }

        termination_grace_period_seconds = 1
      }
    }
  }
}
`, deploymentName, imageName)
}

func testAccKubernetesDeploymentV1ConfigWithDeploymentStrategy(deploymentName, strategy, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
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

    strategy {
      type = "%s"
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
          command = ["sleep", "300"]
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, deploymentName, strategy, imageName)
}

func testAccKubernetesDeploymentV1ConfigWithShareProcessNamespace(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
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
        share_process_namespace = true
        container {
          image   = "%s"
          name    = "containername1"
          command = ["sleep", "300"]
        }
        container {
          image   = "%s"
          name    = "containername2"
          command = ["sleep", "300"]
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, deploymentName, imageName, imageName)
}

func testAccKubernetesDeploymentV1ConfigWithDeploymentStrategyRollingUpdate(deploymentName, maxSurge, maxUnavailable, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
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

    strategy {
      type = "RollingUpdate"

      rolling_update {
        max_surge       = "%s"
        max_unavailable = "%s"
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
          command = ["sleep", "300"]
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, deploymentName, maxSurge, maxUnavailable, imageName)
}

func testAccKubernetesDeploymentV1ConfigHostAliases(name string, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
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
    replicas = 1

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

          resources {
            requests = {
              memory = "64Mi"
              cpu    = "50m"
            }
          }
        }

        host_aliases {
          ip        = "127.0.0.5"
          hostnames = ["abc.com", "contoso.com"]
        }

        host_aliases {
          ip        = "127.0.0.6"
          hostnames = ["xyz.com"]
        }

        termination_grace_period_seconds = 1
      }
    }
  }
}
`, name, imageName)
}

func testAccKubernetesDeploymentV1ConfigWithAutomountServiceAccountToken(deploymentName string, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
  metadata {
    name = "%s"
  }

  spec {
    selector {
      match_labels = {
        "app" = "test"
      }
    }
    template {
      metadata {
        name = "test-automount"
        labels = {
          "app" = "test"
        }
      }
      spec {
        automount_service_account_token = true
        container {
          name    = "containername"
          image   = "%s"
          command = ["sleep", "300"]
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, deploymentName, imageName)
}

func testAccKubernetesDeploymentV1Config_ForceNew(secretName, label, name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
  metadata {
    name = %[1]q
  }

  data = {
    one = "first"
  }
}

resource "kubernetes_deployment_v1" "test" {
  metadata {
    name = %[3]q

    labels = {
      app = %[2]q
    }
  }

  spec {
    selector {
      match_labels = {
        app = %[2]q
      }
    }

    template {
      metadata {
        labels = {
          app = %[2]q
        }
      }

      spec {
        container {
          image   = %[4]q
          name    = "containername"
          command = ["sleep", "300"]

          volume_mount {
            mount_path = "/tmp/my_path"
            name       = "db"
          }
        }

        volume {
          name = "db"

          secret {
            secret_name = "${kubernetes_secret_v1.test.metadata.0.name}"
          }
        }

        termination_grace_period_seconds = 1
      }
    }
  }
}
`, secretName, label, name, imageName)
}

func testAccKubernetesDeploymentV1ConfigWithResourceFieldSelector(rcName, imageName, resourceName, divisor string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
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
          command = ["sleep", "300"]
          resources {
            limits = {
              memory = "512Mi"
            }
          }
          env {
            name = "K8S_LIMITS_CPU"
            value_from {
              resource_field_ref {
                container_name = "containername"
                resource       = "%s"
                divisor        = "%s"
              }
            }
          }
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, rcName, imageName, resourceName, divisor)
}
