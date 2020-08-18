package kubernetes

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultNginxImage          = "nginx:1.19"
	deploymentTestResourceName = "kubernetes_deployment.test"
)

func TestAccKubernetesDeployment_basic(t *testing.T) {
	var conf appsv1.Deployment
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: deploymentTestResourceName,
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(deploymentTestResourceName, &conf),
					testAccCheckKubernetesDeploymentRolledOut(deploymentTestResourceName),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "TestAnnotationTwo": "two"}),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelTwo": "two", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(deploymentTestResourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(deploymentTestResourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(deploymentTestResourceName, "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet(deploymentTestResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.image", defaultNginxImage),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.name", "tf-acc-test"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.0.type", "RollingUpdate"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.0.rolling_update.0.max_surge", "25%"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.0.rolling_update.0.max_unavailable", "25%"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "wait_for_rollout", "true"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_initContainer(t *testing.T) {
	var conf appsv1.Deployment
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: deploymentTestResourceName,
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfig_initContainer(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(deploymentTestResourceName, &conf),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.init_container.0.image", "busybox"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.init_container.0.name", "install"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.init_container.0.command.0", "wget"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.init_container.0.command.1", "-O"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.init_container.0.command.2", "/work-dir/index.html"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.init_container.0.command.3", "http://kubernetes.io"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.init_container.0.volume_mount.0.name", "workdir"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.init_container.0.volume_mount.0.mount_path", "/work-dir"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.dns_config.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.dns_config.0.nameservers.#", "3"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.dns_config.0.nameservers.0", "1.1.1.1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.dns_config.0.nameservers.1", "8.8.8.8"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.dns_config.0.nameservers.2", "9.9.9.9"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.dns_config.0.searches.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.dns_config.0.searches.0", "kubernetes.io"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.dns_config.0.option.#", "2"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.dns_config.0.option.0.name", "ndots"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.dns_config.0.option.0.value", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.dns_config.0.option.1.name", "use-vc"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.dns_config.0.option.1.value", ""),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.dns_policy", "Default"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_importBasic(t *testing.T) {
	resourceName := deploymentTestResourceName
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfig_basic(name),
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

func TestAccKubernetesDeployment_generatedName(t *testing.T) {
	var conf appsv1.Deployment
	prefix := "tf-acc-test-gen-"

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: deploymentTestResourceName,
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfig_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(deploymentTestResourceName, &conf),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "metadata.0.labels.%", "3"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelTwo": "two", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr(deploymentTestResourceName, "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet(deploymentTestResourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(deploymentTestResourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(deploymentTestResourceName, "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet(deploymentTestResourceName, "metadata.0.uid"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_importGeneratedName(t *testing.T) {
	resourceName := deploymentTestResourceName
	prefix := "tf-acc-test-gen-import-"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfig_generatedName(prefix),
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

func TestAccKubernetesDeployment_with_security_context(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := "redis:5.0.2"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithSecurityContext(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(deploymentTestResourceName, &conf),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.security_context.0.fs_group", "100"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.security_context.0.run_as_non_root", "true"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.security_context.0.run_as_user", "101"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.security_context.0.supplemental_groups.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.security_context.0.supplemental_groups.988695518", "101"),
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
		PreCheck:     func() { testAccPreCheck(t); skipIfUnsupportedSecurityContextRunAsGroup(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithSecurityContextRunAsGroup(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(deploymentTestResourceName, &conf),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.security_context.0.fs_group", "100"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.security_context.0.run_as_group", "100"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.security_context.0.run_as_non_root", "true"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.security_context.0.run_as_user", "101"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.security_context.0.supplemental_groups.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.security_context.0.supplemental_groups.988695518", "101"),
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
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithTolerations(deploymentName, imageName, &tolerationSeconds, operator, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(deploymentTestResourceName, &conf),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.toleration.0.effect", "NoExecute"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.toleration.0.key", "myKey"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.toleration.0.operator", operator),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.toleration.0.toleration_seconds", "6000"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.toleration.0.value", ""),
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
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithTolerations(deploymentName, imageName, nil, operator, &value),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(deploymentTestResourceName, &conf),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.toleration.0.effect", "NoExecute"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.toleration.0.key", "myKey"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.toleration.0.operator", operator),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.toleration.0.value", "value"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.toleration.0.toleration_seconds", ""),
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
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithLivenessProbeUsingExec(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(deploymentTestResourceName, &conf),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.args.#", "3"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.exec.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.exec.0.command.#", "2"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.exec.0.command.0", "cat"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.exec.0.command.1", "/tmp/healthy"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.failure_threshold", "3"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.initial_delay_seconds", "5"),
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
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithLivenessProbeUsingHTTPGet(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(deploymentTestResourceName, &conf),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.args.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.http_get.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.http_get.0.path", "/healthz"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.http_get.0.port", "8080"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.http_get.0.http_header.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.http_get.0.http_header.0.name", "X-Custom-Header"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.http_get.0.http_header.0.value", "Awesome"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.initial_delay_seconds", "3"),
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
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithLivenessProbeUsingTCP(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(deploymentTestResourceName, &conf),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.args.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.tcp_socket.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.tcp_socket.0.port", "8080"),
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
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithLifeCycle(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(deploymentTestResourceName, &conf),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.lifecycle.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.lifecycle.0.post_start.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.lifecycle.0.post_start.0.exec.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.lifecycle.0.post_start.0.exec.0.command.#", "2"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.lifecycle.0.post_start.0.exec.0.command.0", "ls"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.lifecycle.0.post_start.0.exec.0.command.1", "-al"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.lifecycle.0.pre_stop.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.lifecycle.0.pre_stop.0.exec.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.lifecycle.0.pre_stop.0.exec.0.command.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.lifecycle.0.pre_stop.0.exec.0.command.0", "date"),
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
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithContainerSecurityContext(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(deploymentTestResourceName, &conf),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.#", "2"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.security_context.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.security_context.0.capabilities.#", "0"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.security_context.0.privileged", "true"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.security_context.0.se_linux_options.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.security_context.0.se_linux_options.0.level", "s0:c123,c456"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.1.security_context.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.1.security_context.0.allow_privilege_escalation", "true"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.1.security_context.0.capabilities.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.1.security_context.0.capabilities.0.add.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.1.security_context.0.capabilities.0.add.0", "NET_BIND_SERVICE"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.1.security_context.0.capabilities.0.drop.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.1.security_context.0.capabilities.0.drop.0", "all"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.1.security_context.0.privileged", "true"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.1.security_context.0.read_only_root_filesystem", "true"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.1.security_context.0.run_as_non_root", "true"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.1.security_context.0.run_as_user", "201"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.1.security_context.0.se_linux_options.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.1.security_context.0.se_linux_options.0.level", "s0:c123,c789"),
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
		PreCheck:     func() { testAccPreCheck(t); skipIfUnsupportedSecurityContextRunAsGroup(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithContainerSecurityContextRunAsGroup(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(deploymentTestResourceName, &conf),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.#", "2"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.security_context.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.security_context.0.capabilities.#", "0"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.security_context.0.privileged", "true"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.security_context.0.se_linux_options.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.security_context.0.se_linux_options.0.level", "s0:c123,c456"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.1.security_context.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.1.security_context.0.allow_privilege_escalation", "true"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.1.security_context.0.capabilities.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.1.security_context.0.capabilities.0.add.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.1.security_context.0.capabilities.0.add.0", "NET_BIND_SERVICE"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.1.security_context.0.capabilities.0.drop.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.1.security_context.0.capabilities.0.drop.0", "all"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.1.security_context.0.privileged", "true"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.1.security_context.0.read_only_root_filesystem", "true"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.1.security_context.0.run_as_group", "200"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.1.security_context.0.run_as_non_root", "true"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.1.security_context.0.run_as_user", "201"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.1.security_context.0.se_linux_options.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.1.security_context.0.se_linux_options.0.level", "s0:c123,c789"),
				),
			},
		},
	})
}
func TestAccKubernetesDeployment_with_volume_mount(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	secretName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	imageName := defaultNginxImage

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithVolumeMounts(secretName, deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(deploymentTestResourceName, &conf),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.volume_mount.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.mount_path", "/tmp/my_path"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.name", "db"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.read_only", "false"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.sub_path", ""),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_resource_requirements(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := defaultNginxImage

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithResourceRequirements(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(deploymentTestResourceName, &conf),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.resources.0.requests.0.memory", "50Mi"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.resources.0.requests.0.cpu", "250m"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.resources.0.limits.0.memory", "512Mi"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.resources.0.limits.0.cpu", "500m"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_empty_dir_volume(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := defaultNginxImage

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithEmptyDirVolumes(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(deploymentTestResourceName, &conf),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.volume_mount.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.mount_path", "/cache"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.name", "cache-volume"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.volume.0.empty_dir.0.medium", "Memory"),
				),
			},
		},
	})
}

func TestAccKubernetesDeploymentUpdate_basic(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfig_basic(deploymentName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(deploymentTestResourceName, &conf),
					// Not to be changed
					resource.TestCheckResourceAttr(deploymentTestResourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					// To be removed
					resource.TestCheckResourceAttr(deploymentTestResourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					// To be added
					resource.TestCheckNoResourceAttr(deploymentTestResourceName, "metadata.0.annotations.Different"),
					// To be changed
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.image", defaultNginxImage),
				),
			},
			{
				Config: testAccKubernetesDeploymentConfig_modified(deploymentName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(deploymentTestResourceName, &conf),
					// Unchanged
					resource.TestCheckResourceAttr(deploymentTestResourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					// Removed
					resource.TestCheckNoResourceAttr(deploymentTestResourceName, "metadata.0.annotations.TestAnnotationTwo"),
					// Added
					resource.TestCheckResourceAttr(deploymentTestResourceName, "metadata.0.annotations.Different", "1234"),
					// Changed
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.container.0.image", defaultNginxImage),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_deployment_strategy_rollingupdate(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := defaultNginxImage

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithDeploymentStrategy(deploymentName, "RollingUpdate", imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(deploymentTestResourceName, &conf),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.0.type", "RollingUpdate"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.0.rolling_update.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.0.rolling_update.0.max_surge", "25%"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.0.rolling_update.0.max_unavailable", "25%"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_share_process_namespace(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := defaultNginxImage

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithShareProcessNamespace(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(deploymentTestResourceName, &conf),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.share_process_namespace", "true"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_no_rollout_wait(t *testing.T) {
	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithWaitForRolloutFalse(deploymentName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentRollingOut(deploymentTestResourceName),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "wait_for_rollout", "false"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_deployment_strategy_rollingupdate_max_surge_30perc_max_unavailable_40perc(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := defaultNginxImage

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithDeploymentStrategyRollingUpdate(deploymentName, "30%", "40%", imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(deploymentTestResourceName, &conf),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.0.type", "RollingUpdate"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.0.rolling_update.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.0.rolling_update.0.max_surge", "30%"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.0.rolling_update.0.max_unavailable", "40%"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_deployment_strategy_rollingupdate_max_surge_200perc_max_unavailable_0perc(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := defaultNginxImage

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithDeploymentStrategyRollingUpdate(deploymentName, "200%", "0%", imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(deploymentTestResourceName, &conf),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.0.type", "RollingUpdate"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.0.rolling_update.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.0.rolling_update.0.max_surge", "200%"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.0.rolling_update.0.max_unavailable", "0%"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_deployment_strategy_rollingupdate_max_surge_0_max_unavailable_1(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := defaultNginxImage

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithDeploymentStrategyRollingUpdate(deploymentName, "0", "1", imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(deploymentTestResourceName, &conf),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.0.type", "RollingUpdate"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.0.rolling_update.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.0.rolling_update.0.max_surge", "0"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.0.rolling_update.0.max_unavailable", "1"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_deployment_strategy_rollingupdate_max_surge_1_max_unavailable_0(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := defaultNginxImage

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithDeploymentStrategyRollingUpdate(deploymentName, "1", "0", imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(deploymentTestResourceName, &conf),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.0.type", "RollingUpdate"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.0.rolling_update.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.0.rolling_update.0.max_surge", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.0.rolling_update.0.max_unavailable", "0"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_deployment_strategy_rollingupdate_max_surge_1_max_unavailable_2(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := defaultNginxImage

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithDeploymentStrategyRollingUpdate(deploymentName, "1", "2", imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(deploymentTestResourceName, &conf),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.0.type", "RollingUpdate"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.0.rolling_update.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.0.rolling_update.0.max_surge", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.0.rolling_update.0.max_unavailable", "2"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_deployment_strategy_recreate(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := defaultNginxImage

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigWithDeploymentStrategy(deploymentName, "Recreate", imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(deploymentTestResourceName, &conf),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.0.type", "Recreate"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.strategy.0.rolling_update.#", "0"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_with_host_aliases(t *testing.T) {
	var conf appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := defaultNginxImage

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDeploymentConfigHostAliases(deploymentName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentExists(deploymentTestResourceName, &conf),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.host_aliases.0.hostnames.#", "2"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.host_aliases.0.hostnames.0", "abc.com"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.host_aliases.0.hostnames.1", "contoso.com"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.host_aliases.0.ip", "127.0.0.5"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.host_aliases.1.hostnames.#", "1"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.host_aliases.1.hostnames.0", "xyz.com"),
					resource.TestCheckResourceAttr(deploymentTestResourceName, "spec.0.template.0.spec.0.host_aliases.1.ip", "127.0.0.6"),
				),
			},
		},
	})
}

func TestAccKubernetesDeployment_config_with_automount_service_account_token(t *testing.T) {
	var confDeployment appsv1.Deployment

	deploymentName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := defaultNginxImage

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesPodDestroy,
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

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_deployment" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{})
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

	namespace, name, err := idParts(rs.Primary.ID)
	if err != nil {
		return nil, err
	}

	out, err := conn.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{})
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

func testAccCheckKubernetesDeploymentRolledOut(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		d, err := getDeploymentFromResourceName(s, n)
		if err != nil {
			return err
		}

		if d.Status.Replicas != d.Status.ReadyReplicas {
			return fmt.Errorf("deployment is still rolling out")
		}

		return nil
	}
}

func testAccKubernetesDeploymentConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_deployment" "test" {
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
            requests {
              memory = "64Mi"
              cpu    = "50m"
            }
          }
        }
      }
    }
	}
}
`, name, defaultNginxImage)
}

func testAccKubernetesDeploymentConfig_initContainer(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_deployment" "test" {
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
          name  = "nginx"
          image = "nginx"

          port {
            container_port = 80
          }

          readiness_probe {
            initial_delay_seconds = 10
            http_get {
              path = "/"
              port = 80
            }
          }

          resources {
            requests {
              memory = "64Mi"
              cpu    = "50m"
            }
          }

          volume_mount {
            name       = "workdir"
            mount_path = "/usr/share/nginx/html"
          }
        }

        init_container {
          name    = "install"
          image   = "busybox"
          command = ["wget", "-O", "/work-dir/index.html", "http://kubernetes.io"]

          resources {
            requests {
              memory = "64Mi"
              cpu    = "50m"
            }
          }

          volume_mount {
            name       = "workdir"
            mount_path = "/work-dir"
          }
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

        dns_policy = "Default"

        volume {
          name      = "workdir"
          empty_dir {}
        }
      }
    }
  }
}
`, name)
}

func testAccKubernetesDeploymentConfig_modified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_deployment" "test" {
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
`, name, defaultNginxImage)
}

func testAccKubernetesDeploymentConfig_generatedName(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_deployment" "test" {
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
`, prefix, defaultNginxImage)
}

func testAccKubernetesDeploymentConfigWithSecurityContext(deploymentName, imageName string) string {
	return fmt.Sprintf(`
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
	return fmt.Sprintf(`
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

func testAccKubernetesDeploymentConfigWithTolerations(deploymentName, imageName string, tolerationSeconds *int, operator string, value *string) string {
	tolerationDuration := ""
	if tolerationSeconds != nil {
		tolerationDuration = fmt.Sprintf("toleration_seconds = %d", *tolerationSeconds)
	}
	valueString := ""
	if value != nil {
		valueString = fmt.Sprintf("value = \"%s\"", *value)
	}

	return fmt.Sprintf(`
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
        toleration {
          effect             = "NoExecute"
          key                = "myKey"
          operator           = "%s"
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
	return fmt.Sprintf(`
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
	return fmt.Sprintf(`
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
	return fmt.Sprintf(`
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
	return fmt.Sprintf(`
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
	return fmt.Sprintf(`
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
	return fmt.Sprintf(`
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
            run_as_group              = 200
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

func testAccKubernetesDeploymentConfigWithVolumeMounts(secretName, deploymentName, imageName string) string {
	return fmt.Sprintf(`
resource "kubernetes_secret" "test" {
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

func testAccKubernetesDeploymentConfigWithResourceRequirements(deploymentName, imageName string) string {
	return fmt.Sprintf(`
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

          resources {
            limits {
              cpu    = "0.5"
              memory = "512Mi"
            }

            requests {
              cpu    = "250m"
              memory = "50Mi"
            }
          }
        }
      }
    }
  }
}
`, deploymentName, imageName)
}

func testAccKubernetesDeploymentConfigWithEmptyDirVolumes(deploymentName, imageName string) string {
	return fmt.Sprintf(`
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

func testAccKubernetesDeploymentConfigWithDeploymentStrategy(deploymentName, strategy, imageName string) string {
	return fmt.Sprintf(`
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
	return fmt.Sprintf(`
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
	return fmt.Sprintf(`
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
	return fmt.Sprintf(`
resource "kubernetes_deployment" "test" {
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
            requests {
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
	return fmt.Sprintf(`
resource "kubernetes_deployment" "test" {
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

func testAccKubernetesDeploymentConfigWithWaitForRolloutFalse(deploymentName string) string {
	return fmt.Sprintf(`
resource "kubernetes_deployment" "test" {
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
`, deploymentName, defaultNginxImage)
}
