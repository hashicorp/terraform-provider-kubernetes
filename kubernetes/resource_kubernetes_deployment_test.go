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

func TestAccKubernetesDeployment_minimal(t *testing.T) {
	var conf appsv1.Deployment
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_deployment.test",
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfig_minimal(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf),
					resource.TestCheckResourceAttrSet("kubernetes_deployment.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_deployment.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_deployment.test", "metadata.0.uid"),
				),
			},
			{
				Config:   testAccKubernetesDeploymentConfig_minimal(name, imageName),
				PlanOnly: true,
			},
		},
	})
}

func TestAccKubernetesDeployment_basic(t *testing.T) {
	var conf appsv1.Deployment
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_deployment.test",
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfig_basic(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_deployment.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_deployment.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_deployment.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.name", "tf-acc-test"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.strategy.0.type", "RollingUpdate"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.strategy.0.rolling_update.0.max_surge", "25%"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.strategy.0.rolling_update.0.max_unavailable", "25%"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "wait_for_rollout", "true"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_initContainerForceNew(t *testing.T) {
	var conf1, conf2 appsv1.Deployment
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImageVersion
	imageName1 := busyboxImageVersion1
	initCommand := "until nslookup init-service.default.svc.cluster.local; do echo waiting for init-service; sleep 2; done"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_deployment.test",
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfig_initContainer(
					name, imageName, imageName1, "64Mi", "testvar",
					"initcontainer2", initCommand, "IfNotPresent"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf1),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.init_container.0.image", imageName),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.init_container.0.resources.0.requests.memory", "64Mi"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.init_container.0.env.2.value", "testvar"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.init_container.1.name", "initcontainer2"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.init_container.1.image", imageName1),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.init_container.1.command.2", initCommand),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.init_container.1.image_pull_policy", "IfNotPresent"),
				),
			},
			{ // Test for non-empty plans. No modification.
				Config: testAccKubernetesDeploymentConfig_initContainer(
					name, imageName, imageName1, "64Mi", "testvar",
					"initcontainer2", initCommand, "IfNotPresent"),
				PlanOnly: true,
			},
			{ // Modify resources.limits.memory.
				Config: testAccKubernetesDeploymentConfig_initContainer(
					name, imageName, imageName1, "80Mi", "testvar",
					"initcontainer2", initCommand, "IfNotPresent"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf2),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.init_container.0.resources.0.requests.memory", "80Mi"),
					testAccCheckKubernetesDeploymentForceNew(&conf1, &conf2, false),
				),
			},
			{ // Modify name of environment variable.
				Config: testAccKubernetesDeploymentConfig_initContainer(
					name, imageName, imageName1, "64Mi", "testvar",
					"initcontainer2", initCommand, "IfNotPresent"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf2),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.init_container.0.env.2.value", "testvar"),
					testAccCheckKubernetesDeploymentForceNew(&conf1, &conf2, false),
				),
			},
			{ // Modify init_container's command.
				Config: testAccKubernetesDeploymentConfig_initContainer(
					name, imageName, imageName1, "64Mi", "testvar",
					"initcontainer2", "echo done", "IfNotPresent"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf2),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.init_container.1.command.2", "echo done"),
					testAccCheckKubernetesDeploymentForceNew(&conf1, &conf2, false),
				),
			},
			{ // Modify init_container's image_pull_policy.
				Config: testAccKubernetesDeploymentConfig_initContainer(
					name, imageName, imageName1, "64Mi", "testvar",
					"initcontainer2", "echo done", "Never"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf2),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.init_container.1.image_pull_policy", "Never"),
					testAccCheckKubernetesDeploymentForceNew(&conf1, &conf2, false),
				),
			},
			{ // Modify init_container's image
				Config: testAccKubernetesDeploymentConfig_initContainer(
					name, imageName, imageName, "64Mi", "testvar",
					"initcontainer2", "echo done", "Never"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf2),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.init_container.1.image", imageName),
					testAccCheckKubernetesDeploymentForceNew(&conf1, &conf2, false),
				),
			},
			{ // Modify init_container's name.
				Config: testAccKubernetesDeploymentConfig_initContainer(
					name, imageName, imageName, "64Mi", "testvar",
					"initcontainertwo", "echo done", "Never"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf2),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.init_container.1.name", "initcontainertwo"),
					testAccCheckKubernetesDeploymentForceNew(&conf1, &conf2, false),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_generatedName(t *testing.T) {
	var conf appsv1.Deployment
	prefix := "tf-acc-test-gen-"
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_deployment.test",
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfig_generatedName(prefix, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr("kubernetes_deployment.test", "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet("kubernetes_deployment.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_deployment.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_deployment.test", "metadata.0.uid"),
				),
			},
			{
				ResourceName:            "kubernetes_deployment.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "wait_for_rollout"},
			},
		},
	})
}

func TestAccKubernetesDeployment_with_security_context(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := "redis:5.0.2"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithSecurityContext(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.security_context.0.fs_group", "100"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.security_context.0.run_as_non_root", "true"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.security_context.0.run_as_user", "101"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.security_context.0.supplemental_groups.0", "101"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.security_context.0.fs_group", "100"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.security_context.0.sysctl.#", "0"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_security_context_run_as_group(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := "redis:5.0.2"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfUnsupportedSecurityContextRunAsGroup(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithSecurityContextRunAsGroup(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.security_context.0.fs_group", "100"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.security_context.0.run_as_group", "100"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.security_context.0.run_as_non_root", "true"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.security_context.0.run_as_user", "101"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.security_context.0.supplemental_groups.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.security_context.0.supplemental_groups.0", "101"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_security_context_sysctl(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := "redis:5.0.2"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithSecurityContextSysctl(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.security_context.0.fs_group", "100"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.security_context.0.run_as_non_root", "true"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.security_context.0.run_as_user", "101"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.security_context.0.supplemental_groups.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.security_context.0.supplemental_groups.0", "101"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.security_context.0.sysctl.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.security_context.0.sysctl.0.name", "kernel.shm_rmid_forced"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.security_context.0.sysctl.0.value", "0"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_tolerations(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := "redis:5.0.2"
	tolerationSeconds := 6000
	operator := "Equal"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithTolerations(deploymentName, imageName, &tolerationSeconds, operator, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.toleration.0.effect", "NoExecute"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.toleration.0.key", "myKey"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.toleration.0.operator", operator),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.toleration.0.toleration_seconds", "6000"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.toleration.0.value", ""),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_tolerations_unset_toleration_seconds(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := "redis:5.0.2"
	operator := "Equal"
	value := "value"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithTolerations(deploymentName, imageName, nil, operator, &value),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.toleration.0.effect", "NoExecute"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.toleration.0.key", "myKey"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.toleration.0.operator", operator),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.toleration.0.value", "value"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.toleration.0.toleration_seconds", ""),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_container_liveness_probe_using_exec(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := "gcr.io/google_containers/busybox"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithLivenessProbeUsingExec(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.args.#", "3"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.liveness_probe.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.liveness_probe.0.exec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.liveness_probe.0.exec.0.command.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.liveness_probe.0.exec.0.command.0", "cat"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.liveness_probe.0.exec.0.command.1", "/tmp/healthy"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.liveness_probe.0.failure_threshold", "3"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.liveness_probe.0.initial_delay_seconds", "5"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_container_liveness_probe_using_http_get(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := "gcr.io/google_containers/liveness"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithLivenessProbeUsingHTTPGet(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.args.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.liveness_probe.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.liveness_probe.0.http_get.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.liveness_probe.0.http_get.0.path", "/healthz"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.liveness_probe.0.http_get.0.port", "8080"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.liveness_probe.0.http_get.0.http_header.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.liveness_probe.0.http_get.0.http_header.0.name", "X-Custom-Header"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.liveness_probe.0.http_get.0.http_header.0.value", "Awesome"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.liveness_probe.0.initial_delay_seconds", "3"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_container_liveness_probe_using_tcp(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := "gcr.io/google_containers/liveness"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithLivenessProbeUsingTCP(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.args.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.liveness_probe.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.liveness_probe.0.tcp_socket.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.liveness_probe.0.tcp_socket.0.port", "8080"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_container_lifecycle(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := "gcr.io/google_containers/busybox"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithLifeCycle(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.lifecycle.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.lifecycle.0.post_start.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.lifecycle.0.post_start.0.exec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.lifecycle.0.post_start.0.exec.0.command.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.lifecycle.0.post_start.0.exec.0.command.0", "ls"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.lifecycle.0.post_start.0.exec.0.command.1", "-al"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.lifecycle.0.pre_stop.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.lifecycle.0.pre_stop.0.exec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.lifecycle.0.pre_stop.0.exec.0.command.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.lifecycle.0.pre_stop.0.exec.0.command.0", "date"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_container_security_context(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := "redis:5.0.2"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithContainerSecurityContext(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.security_context.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.security_context.0.capabilities.#", "0"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.security_context.0.privileged", "true"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.security_context.0.se_linux_options.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.security_context.0.se_linux_options.0.level", "s0:c123,c456"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.1.security_context.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.1.security_context.0.allow_privilege_escalation", "true"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.1.security_context.0.capabilities.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.1.security_context.0.capabilities.0.add.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.1.security_context.0.capabilities.0.add.0", "NET_BIND_SERVICE"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.1.security_context.0.capabilities.0.drop.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.1.security_context.0.capabilities.0.drop.0", "all"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.1.security_context.0.privileged", "true"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.1.security_context.0.read_only_root_filesystem", "true"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.1.security_context.0.run_as_non_root", "true"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.1.security_context.0.run_as_user", "201"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.1.security_context.0.se_linux_options.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.1.security_context.0.se_linux_options.0.level", "s0:c123,c789"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_container_security_context_run_as_group(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := "redis:5.0.2"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfUnsupportedSecurityContextRunAsGroup(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithContainerSecurityContextRunAsGroup(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.security_context.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.security_context.0.privileged", "true"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.security_context.0.se_linux_options.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.security_context.0.se_linux_options.0.level", "s0:c123,c456"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.1.security_context.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.1.security_context.0.allow_privilege_escalation", "true"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.1.security_context.0.privileged", "false"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.1.security_context.0.run_as_group", "200"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.1.security_context.0.run_as_non_root", "false"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.1.security_context.0.run_as_user", "201"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_container_security_context_seccomp_profile(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion
	resourceName := "kubernetes_deployment.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithContainerSecurityContextSeccompProfile(deploymentName, imageName, "Unconfined"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.seccomp_profile.0.type", "Unconfined"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.security_context.0.seccomp_profile.0.type", "Unconfined"),
				),
			},
			{
				Config: testAccKubernetesDeploymentConfigWithContainerSecurityContextSeccompProfile(deploymentName, imageName, "RuntimeDefault"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.seccomp_profile.0.type", "RuntimeDefault"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.security_context.0.seccomp_profile.0.type", "RuntimeDefault"),
				),
			},
			{
				Config: testAccKubernetesDeploymentConfigWithContainerSecurityContextSeccompProfileLocalhost(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.seccomp_profile.0.type", "Localhost"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.seccomp_profile.0.localhost_profile", ""),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.security_context.0.seccomp_profile.0.type", "Localhost"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.security_context.0.seccomp_profile.0.localhost_profile", ""),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_volume_mount(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	secretName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_deployment.test"
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithVolumeMounts(secretName, deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.mount_path", "/tmp/my_path"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.name", "db"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.read_only", "false"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.sub_path", ""),
				),
			},
			{
				Config: testAccKubernetesDeploymentConfigWithVolumeMountsNone(secretName, deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.#", "0"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_ForceNew(t *testing.T) {
	var conf1, conf2 appsv1.Deployment

	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfig_ForceNew("secret1", "label1", "deployment1", imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf1),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "metadata.0.labels.app", "label1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.selector.0.match_labels.app", "label1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.metadata.0.labels.app", "label1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.volume.0.secret.0.secret_name", "secret1"),
				),
			},
			{
				Config: testAccKubernetesDeploymentConfig_ForceNew("secret2", "label1", "deployment1", imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf2),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "metadata.0.labels.app", "label1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.selector.0.match_labels.app", "label1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.metadata.0.labels.app", "label1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.volume.0.secret.0.secret_name", "secret2"),
					testAccCheckKubernetesDeploymentForceNew(&conf1, &conf2, false),
				),
			},
			{ // BUG: labels cannot be updated on a deployment without triggering a ForceNew.
				Config: testAccKubernetesDeploymentConfig_ForceNew("secret2", "label2", "deployment1", imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf2),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "metadata.0.labels.app", "label2"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.selector.0.match_labels.app", "label2"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.metadata.0.labels.app", "label2"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.volume.0.secret.0.secret_name", "secret2"),
					//					testAccCheckKubernetesDeploymentForceNew(&conf1, &conf2, false),
				),
			},
			{
				Config: testAccKubernetesDeploymentConfig_ForceNew("secret2", "label2", "deployment1", "nginx:1.18"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf2),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "metadata.0.labels.app", "label2"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.selector.0.match_labels.app", "label2"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.metadata.0.labels.app", "label2"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.image", "nginx:1.18"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.volume.0.secret.0.secret_name", "secret2"),
					testAccCheckKubernetesDeploymentForceNew(&conf1, &conf2, true),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_resource_requirements(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithResourceRequirements(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.resources.0.requests.memory", "50Mi"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.resources.0.requests.cpu", "250m"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.resources.0.requests.nvidia/gpu", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.resources.0.limits.memory", "512Mi"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.resources.0.limits.cpu", "500m"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.resources.0.limits.nvidia/gpu", "1"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_empty_dir_volume(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithEmptyDirVolumes(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.volume_mount.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.volume_mount.0.mount_path", "/cache"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.volume_mount.0.name", "cache-volume"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.volume.0.empty_dir.0.medium", "Memory"),
				),
			},
			{
				Config: testAccKubernetesDeploymentConfigWithEmptyDirVolumesModified(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.volume_mount.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.volume_mount.0.mount_path", "/cache"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.volume_mount.0.name", "cache-volume"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.volume.0.empty_dir.0.medium", "Memory"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.volume.0.empty_dir.0.size_limit", "128Mi"),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentUpdate_basic(t *testing.T) {
	var conf1, conf2 appsv1.Deployment
	imageName := nginxImageVersion
	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfig_basic(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf1),
					// Not to be changed
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					// To be removed
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					// To be added
					resource.TestCheckNoResourceAttr("kubernetes_deployment.test", "metadata.0.annotations.Different"),
					// To be changed
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.image", imageName),
				),
			},
			{
				Config: testAccKubernetesDeploymentConfig_modified(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf2),
					// Unchanged
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					// Removed
					resource.TestCheckNoResourceAttr("kubernetes_deployment.test", "metadata.0.annotations.TestAnnotationTwo"),
					// Added
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "metadata.0.annotations.Different", "1234"),
					// Changed
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.image", nginxImageVersion),
					testAccCheckKubernetesDeploymentForceNew(&conf1, &conf2, false),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_deployment_strategy_rollingupdate(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion
	resourceName := "kubernetes_deployment.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithDeploymentStrategy(deploymentName, "RollingUpdate", imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.type", "RollingUpdate"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.0.max_surge", "25%"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.0.max_unavailable", "25%"),
				),
			},
			{
				Config: testAccKubernetesDeploymentConfigWithDeploymentStrategy(deploymentName, "Recreate", imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.type", "Recreate"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.strategy.0.rolling_update.#", "0"),
				),
			},
			{
				Config: testAccKubernetesDeploymentConfigWithDeploymentStrategy(deploymentName, "RollingUpdate", imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(resourceName, &conf),
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

func TestAccKubernetesDeployment_with_share_process_namespace(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithShareProcessNamespace(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.share_process_namespace", "true"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_no_rollout_wait(t *testing.T) {
	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithWaitForRolloutFalse(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentRollingOut("kubernetes_deployment.test"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "wait_for_rollout", "false"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_deployment_strategy_rollingupdate_max_surge_30perc_max_unavailable_40perc(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithDeploymentStrategyRollingUpdate(deploymentName, "30%", "40%", imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.strategy.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.strategy.0.type", "RollingUpdate"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.strategy.0.rolling_update.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.strategy.0.rolling_update.0.max_surge", "30%"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.strategy.0.rolling_update.0.max_unavailable", "40%"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_deployment_strategy_rollingupdate_max_surge_200perc_max_unavailable_0perc(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithDeploymentStrategyRollingUpdate(deploymentName, "200%", "0%", imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.strategy.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.strategy.0.type", "RollingUpdate"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.strategy.0.rolling_update.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.strategy.0.rolling_update.0.max_surge", "200%"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.strategy.0.rolling_update.0.max_unavailable", "0%"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_deployment_strategy_rollingupdate_max_surge_0_max_unavailable_1(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithDeploymentStrategyRollingUpdate(deploymentName, "0", "1", imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.strategy.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.strategy.0.type", "RollingUpdate"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.strategy.0.rolling_update.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.strategy.0.rolling_update.0.max_surge", "0"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.strategy.0.rolling_update.0.max_unavailable", "1"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_deployment_strategy_rollingupdate_max_surge_1_max_unavailable_0(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithDeploymentStrategyRollingUpdate(deploymentName, "1", "0", imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.strategy.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.strategy.0.type", "RollingUpdate"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.strategy.0.rolling_update.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.strategy.0.rolling_update.0.max_surge", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.strategy.0.rolling_update.0.max_unavailable", "0"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_deployment_strategy_rollingupdate_max_surge_1_max_unavailable_2(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithDeploymentStrategyRollingUpdate(deploymentName, "1", "2", imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.strategy.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.strategy.0.type", "RollingUpdate"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.strategy.0.rolling_update.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.strategy.0.rolling_update.0.max_surge", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.strategy.0.rolling_update.0.max_unavailable", "2"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_deployment_strategy_recreate(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithDeploymentStrategy(deploymentName, "Recreate", imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.strategy.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.strategy.0.type", "Recreate"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.strategy.0.rolling_update.#", "0"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_host_aliases(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigHostAliases(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.host_aliases.0.hostnames.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.host_aliases.0.hostnames.0", "abc.com"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.host_aliases.0.hostnames.1", "contoso.com"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.host_aliases.0.ip", "127.0.0.5"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.host_aliases.1.hostnames.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.host_aliases.1.hostnames.0", "xyz.com"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.host_aliases.1.ip", "127.0.0.6"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_resource_field_selector(t *testing.T) {
	var conf appsv1.Deployment
	rcName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				ExpectError: regexp.MustCompile("quantities must match the regular expression"),
				Config:      testAccKubernetesDeploymentConfigWithResourceFieldSelector(rcName, imageName, "limits.cpu", ""),
			},
			{
				ExpectError: regexp.MustCompile("only divisor's values 1m and 1 are supported with the cpu resource"),
				Config:      testAccKubernetesDeploymentConfigWithResourceFieldSelector(rcName, imageName, "limits.cpu", "2"),
			},
			{
				Config: testAccKubernetesDeploymentConfigWithResourceFieldSelector(rcName, imageName, "limits.cpu", "1m"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.resources.0.limits.memory", "512Mi"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.env.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.env.0.name", "K8S_LIMITS_CPU"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.env.0.value_from.0.resource_field_ref.0.container_name", "containername"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.env.0.value_from.0.resource_field_ref.0.divisor", "1m"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.env.0.value_from.0.resource_field_ref.0.resource", "limits.cpu"),
				),
			},
			{
				Config: testAccKubernetesDeploymentConfigWithResourceFieldSelector(rcName, imageName, "limits.memory", "1Mi"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.env.0.value_from.0.resource_field_ref.0.divisor", "1Mi"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.env.0.value_from.0.resource_field_ref.0.resource", "limits.memory"),
				),
			},
			{
				Config: testAccKubernetesDeploymentConfigWithResourceFieldSelector(rcName, imageName, "requests.memory", "1Ki"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.env.0.value_from.0.resource_field_ref.0.divisor", "1Ki"),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.env.0.value_from.0.resource_field_ref.0.resource", "requests.memory"),
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

func TestAccKubernetesDeployment_config_with_automount_service_account_token(t *testing.T) {
	var confDeployment appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithAutomountServiceAccountToken(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists("kubernetes_deployment.test", &confDeployment),
					resource.TestCheckResourceAttr("kubernetes_deployment.test", "spec.0.template.0.spec.0.automount_service_account_token", "true"),
				),
			},
		},
	})
}

func testAccCheckKubernetesDeploymentDestroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_deployment" {
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

func testAccCheckKubernetesDeploymentExists(n string, obj *appsv1.Deployment) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		d, err := getDeploymentFromResourceName(s, n)
		if err != nil {
			return err
		}
		*obj = *d
		return nil
	}
}

func testAccCheckKubernetesDeploymentRollingOut(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		d, err := getDeploymentFromResourceName(s, n)
		if err != nil {
			return err
		}

		if d.Status.Replicas == d.Status.ReadyReplicas {
			return fmt.Errorf("deployment has already rolled out")
		}

		return nil
	}
}

func testAccKubernetesDeploymentConfig_minimal(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment" "test" {
  metadata {
    name = "%s"
  }
  spec {
    replicas = 2
    selector {
      match_labels = {
        TestLabelOne   = "one"
      }
    }
    template {
      metadata {
        labels = {
          TestLabelOne   = "one"
        }
      }
      spec {
        container {
          image = "%s"
          name  = "tf-acc-test"
        }
      }
    }
  }
}
`, name, imageName)
}

func testAccKubernetesDeploymentConfig_basic(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment" "test" {
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
    replicas = 5

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

          port {
            container_port = 80
          }

          readiness_probe {
            initial_delay_seconds = 5
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
      }
    }
  }
}
`, name, imageName)
}

func testAccKubernetesDeploymentConfig_initContainer(name, imageName, imageName1, memory, envName, initName, initCommand, pullPolicy string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment" "test" {
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
          name = "regularcontainer"
          image = "%s"
          command = ["sleep"]
          args = ["120s"]
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
                name     = "test"
                key      = "SECRETENV"
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
              name     = kubernetes_config_map.test.metadata.0.name
            }
            prefix = "CONFIG"
          }
          port {
            container_port = 80
          }
        env {
          name = "CUSTOM"
          value = "%s"
        }
      }
        init_container {
          name = "%s"
          image = "%s"
          image_pull_policy = "%s"
          command = [
            "sh",
            "-c",
            "%s",
         ]
        }
      }
    }
  }
}

resource "kubernetes_service" "test" {
  metadata {
    name = "init-service"
  }
  spec {
    port {
      port        = 8080
      target_port = 80
    }
  }
}

resource "kubernetes_secret" "test" {
  metadata {
    name = "test"
  }
  data = {
    "SECRETENV" = "asdf1234"
  }
}


resource "kubernetes_config_map" "test" {
  metadata {
    name = "test"
  }
  data = {
    "ENV" = "somedata"
  }
}
`, name, imageName, imageName, memory, envName, initName, imageName1, pullPolicy, initCommand)
}

func testAccKubernetesDeploymentConfig_modified(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment" "test" {
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
          image = %q
          name  = "tf-acc-test"
        }
      }
    }
  }
}
`, name, nginxImageVersion)
}

func testAccKubernetesDeploymentConfig_generatedName(prefix, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment" "test" {
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
          image = %q
          name  = "tf-acc-test"
        }
      }
    }
  }
}
`, prefix, nginxImageVersion)
}

func testAccKubernetesDeploymentConfigWithSecurityContext(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment" "test" {
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
          image = "%s"
          name  = "containername"
        }
      }
    }
  }
}
`, deploymentName, imageName)
}

func testAccKubernetesDeploymentConfigWithSecurityContextRunAsGroup(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment" "test" {
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
          image = "%s"
          name  = "containername"
        }
      }
    }
  }
}
`, deploymentName, imageName)
}

func testAccKubernetesDeploymentConfigWithSecurityContextSysctl(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment" "test" {
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
          image = "%s"
          name  = "containername"
        }
      }
    }
  }
}
`, deploymentName, imageName)
}

func testAccKubernetesDeploymentConfigWithTolerations(deploymentName, imageName string, tolerationSeconds *int, operator string, value *string) string {
	tolerationDuration := ""
	if tolerationSeconds != nil {
		tolerationDuration = fmt.Sprintf("toleration_seconds = %d", *tolerationSeconds)
	}
	valueString := ""
	if value != nil {
		valueString = fmt.Sprintf("value = \"%s\"", *value)
	}

	return fmt.Sprintf(`resource "kubernetes_deployment" "test" {
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
          image = "%s"
          name  = "containername"
        }
      }
    }
  }
}
`, deploymentName, operator, valueString, tolerationDuration, imageName)
}

func testAccKubernetesDeploymentConfigWithLivenessProbeUsingExec(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment" "test" {
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

            initial_delay_seconds = 5
            period_seconds        = 5
          }
        }
      }
    }
  }
}
`, deploymentName, imageName)
}

func testAccKubernetesDeploymentConfigWithLivenessProbeUsingHTTPGet(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment" "test" {
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
          args  = ["/server"]

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
            period_seconds        = 3
          }
        }
      }
    }
  }
}
`, deploymentName, imageName)
}

func testAccKubernetesDeploymentConfigWithLivenessProbeUsingTCP(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment" "test" {
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
          args  = ["/server"]

          liveness_probe {
            tcp_socket {
              port = 8080
            }

            initial_delay_seconds = 3
            period_seconds        = 3
          }
        }
      }
    }
  }
}
`, deploymentName, imageName)
}

func testAccKubernetesDeploymentConfigWithLifeCycle(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment" "test" {
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
      }
    }
  }
}
`, deploymentName, imageName)
}

func testAccKubernetesDeploymentConfigWithContainerSecurityContext(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment" "test" {
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

          security_context {
            privileged  = true
            run_as_user = 1

            se_linux_options {
              level = "s0:c123,c456"
            }
          }
        }

        container {
          image = "gcr.io/google_containers/liveness"
          name  = "containername2"
          args  = ["/server"]

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
      }
    }
  }
}
`, deploymentName, imageName)
}

func testAccKubernetesDeploymentConfigWithContainerSecurityContextRunAsGroup(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment" "test" {
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

          security_context {
            privileged  = true
            run_as_user = 1

            se_linux_options {
              level = "s0:c123,c456"
            }
          }
        }

        container {
          name  = "container2"
          image = "%s"
          command = ["sh", "-c", "echo The app is running! && sleep 3600"]
          security_context {
            run_as_group = 200
            run_as_user  = 201
          }
        }
      }
    }
  }
}
`, deploymentName, imageName, imageName)
}

func testAccKubernetesDeploymentConfigWithContainerSecurityContextSeccompProfile(deploymentName, imageName, seccompProfileType string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment" "test" {
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
      }
    }
  }
}
`, deploymentName, seccompProfileType, imageName, seccompProfileType)
}

func testAccKubernetesDeploymentConfigWithContainerSecurityContextSeccompProfileLocalhost(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment" "test" {
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
            localhost_profile = ""
          }
        }
        container {
          image = "%s"
          name  = "containername"

          security_context {
            seccomp_profile {
              type              = "Localhost"
              localhost_profile = ""
            }
          }
        }
      }
    }
  }
}
`, deploymentName, imageName)
}

func testAccKubernetesDeploymentConfigWithVolumeMounts(secretName, deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_secret" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "first"
  }
}

resource "kubernetes_deployment" "test" {
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

          volume_mount {
            mount_path = "/tmp/my_path"
            name       = "db"
          }
        }

        volume {
          name = "db"

          secret {
            secret_name = "${kubernetes_secret.test.metadata.0.name}"
          }
        }
      }
    }
  }
}
`, secretName, deploymentName, imageName)
}

func testAccKubernetesDeploymentConfigWithVolumeMountsNone(secretName, deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_secret" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "first"
  }
}

resource "kubernetes_deployment" "test" {
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
        }

        volume {
          name = "db"

          secret {
            secret_name = "${kubernetes_secret.test.metadata.0.name}"
          }
        }
      }
    }
  }
}
`, secretName, deploymentName, imageName)
}

func testAccKubernetesDeploymentConfigWithResourceRequirements(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment" "test" {
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
              cpu    = "0.5"
              memory = "512Mi"
              "nvidia/gpu" = "1"
            }

            requests = {
              cpu    = "250m"
              memory = "50Mi"
              "nvidia/gpu" = "1"
            }
          }
        }
      }
    }
  }

  wait_for_rollout = false
}
`, deploymentName, imageName)
}

func testAccKubernetesDeploymentConfigWithEmptyDirVolumes(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment" "test" {
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
      }
    }
  }
}
`, deploymentName, imageName)
}

func testAccKubernetesDeploymentConfigWithEmptyDirVolumesModified(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment" "test" {
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

          volume_mount {
            mount_path = "/cache"
            name       = "cache-volume"
          }
        }

        volume {
          name = "cache-volume"

          empty_dir {
            medium = "Memory"
            size_limit = "128Mi"
          }
        }
      }
    }
  }
}
`, deploymentName, imageName)
}

func testAccKubernetesDeploymentConfigWithDeploymentStrategy(deploymentName, strategy, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment" "test" {
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
          image = "%s"
          name  = "containername"
        }
      }
    }
  }
}
`, deploymentName, strategy, imageName)
}

func testAccKubernetesDeploymentConfigWithShareProcessNamespace(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment" "test" {
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
          image = "%s"
          name  = "containername1"
        }
        container {
          image = "%s"
          name  = "containername2"
        }
      }
    }
  }
}
`, deploymentName, imageName, imageName)
}

func testAccKubernetesDeploymentConfigWithDeploymentStrategyRollingUpdate(deploymentName, maxSurge, maxUnavailable, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment" "test" {
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
          image = "%s"
          name  = "containername"
        }
      }
    }
  }
}
`, deploymentName, maxSurge, maxUnavailable, imageName)
}

func testAccKubernetesDeploymentConfigHostAliases(name string, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment" "test" {
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
          image = "%s"
          name  = "tf-acc-test"

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
      }
    }
  }
}
`, name, imageName)
}

func testAccKubernetesDeploymentConfigWithAutomountServiceAccountToken(deploymentName string, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment" "test" {
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
          name  = "containername"
          image = "%s"
        }
      }
    }
  }
}
`, deploymentName, imageName)
}

func testAccKubernetesDeploymentConfigWithWaitForRolloutFalse(deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment" "test" {
  metadata {
    name = %q
  }
  spec {
    replicas = 5
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
        container {
          name  = "nginx"
          image = %q
          port {
            container_port = 80
          }
          readiness_probe {
            initial_delay_seconds = 5
            http_get {
              path = "/"
              port = 80
            }
          }
        }
      }
    }
  }
  wait_for_rollout = false
}
`, deploymentName, nginxImageVersion)
}

func testAccKubernetesDeploymentConfigLocal(provider, name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment" "test" {
  provider = %s
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
    replicas = 5

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
          image = %q
          name  = "containername"

          port {
            container_port = 80
          }

          readiness_probe {
            initial_delay_seconds = 5
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
          env {
            name = "LIMITS_CPU"
            value_from {
              resource_field_ref {
                container_name = "containername"
                resource       = "requests.cpu"
              }
            }
          }
          env {
           name = "LIMITS_MEM"
            value_from {
              resource_field_ref {
                container_name = "containername"
                resource       = "requests.memory"
              }
            } 
          }
        }
      }
    }
  }
}
`, provider, name, imageName)
}

func testAccKubernetesDeploymentConfig_ForceNew(secretName, label, deploymentName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_secret" "test" {
  metadata {
    name = %[1]q
  }

  data = {
    one = "first"
  }
}

resource "kubernetes_deployment" "test" {
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
          image = %[4]q
          name  = "containername"

          volume_mount {
            mount_path = "/tmp/my_path"
            name       = "db"
          }
        }

        volume {
          name = "db"

          secret {
            secret_name = "${kubernetes_secret.test.metadata.0.name}"
          }
        }
      }
    }
  }
}
`, secretName, label, deploymentName, imageName)
}

func testAccKubernetesDeploymentConfigWithResourceFieldSelector(rcName, imageName, resourceName, divisor string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment" "test" {
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
      }
    }
  }
}
`, rcName, imageName, resourceName, divisor)
}
