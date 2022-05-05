package kubernetes

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesReplicationController_minimal(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_replication_controller.test",
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesReplicationControllerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerConfigMinimal(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("kubernetes_replication_controller.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_replication_controller.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_replication_controller.test", "metadata.0.uid"),
				),
			},
			{
				Config:   testAccKubernetesReplicationControllerConfigMinimal(name, imageName),
				PlanOnly: true,
			},
		},
	})
}

func TestAccKubernetesReplicationController_basic(t *testing.T) {
	var conf api.ReplicationController
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_replication_controller.test",
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesReplicationControllerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerConfig_basic(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerExists("kubernetes_replication_controller.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_replication_controller.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_replication_controller.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_replication_controller.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.name", "tf-acc-test"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.metadata.0.annotations.TestAnnotationFive", "five"),
				),
			},
			{
				ResourceName:            "kubernetes_replication_controller.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesReplicationControllerConfig_modified(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerExists("kubernetes_replication_controller.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.annotations.Different", "1234"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_replication_controller.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_replication_controller.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_replication_controller.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.name", "tf-acc-test"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.metadata.0.annotations.TestAnnotationSix", "six"),
				),
			},
		},
	})
}

func TestAccKubernetesReplicationController_initContainer(t *testing.T) {
	var conf1, conf2 api.ReplicationController
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_replication_controller.test",
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesReplicationControllerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerConfig_initContainer(name, busyboxImageVersion),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerExists("kubernetes_replication_controller.test", &conf1),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.init_container.0.image", busyboxImageVersion),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.init_container.0.name", "install"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.init_container.0.command.0", "wget"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.init_container.0.command.1", "-O"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.init_container.0.command.2", "/work-dir/index.html"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.init_container.0.command.3", "http://kubernetes.io"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.init_container.0.volume_mount.0.name", "workdir"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.init_container.0.volume_mount.0.mount_path", "/work-dir"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.dns_config.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.dns_config.0.nameservers.#", "3"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.dns_config.0.nameservers.0", "1.1.1.1"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.dns_config.0.nameservers.1", "8.8.8.8"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.dns_config.0.nameservers.2", "9.9.9.9"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.dns_config.0.searches.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.dns_config.0.searches.0", "kubernetes.io"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.dns_config.0.option.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.dns_config.0.option.0.name", "ndots"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.dns_config.0.option.0.value", "1"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.dns_config.0.option.1.name", "use-vc"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.dns_config.0.option.1.value", ""),
				),
			},
			{
				Config: testAccKubernetesReplicationControllerConfig_initContainer(name, busyboxImageVersion1),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerExists("kubernetes_replication_controller.test", &conf2),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.init_container.0.image", busyboxImageVersion1),
					testAccCheckKubernetesReplicationControllerForceNew(&conf1, &conf2, true),
				),
			},
		},
	})
}

func TestAccKubernetesReplicationController_generatedName(t *testing.T) {
	var conf api.ReplicationController
	prefix := "tf-acc-test-gen-"
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_replication_controller.test",
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesReplicationControllerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerConfig_generatedName(prefix, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerExists("kubernetes_replication_controller.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr("kubernetes_replication_controller.test", "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet("kubernetes_replication_controller.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_replication_controller.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_replication_controller.test", "metadata.0.uid"),
				),
			},
			{
				ResourceName:            "kubernetes_replication_controller.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
		},
	})
}

func TestAccKubernetesReplicationController_with_security_context(t *testing.T) {
	var conf api.ReplicationController

	rcName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesReplicationControllerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerConfigWithSecurityContext(rcName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerExists("kubernetes_replication_controller.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.security_context.0.fs_group", "100"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.security_context.0.run_as_non_root", "true"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.security_context.0.run_as_user", "101"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.security_context.0.supplemental_groups.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.security_context.0.supplemental_groups.0", "101"),
				),
			},
		},
	})
}

func TestAccKubernetesReplicationController_with_security_context_run_as_group(t *testing.T) {
	var conf api.ReplicationController

	rcName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfUnsupportedSecurityContextRunAsGroup(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesReplicationControllerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerConfigWithSecurityContextRunAsGroup(rcName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerExists("kubernetes_replication_controller.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.security_context.0.fs_group", "100"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.security_context.0.run_as_group", "100"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.security_context.0.run_as_non_root", "true"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.security_context.0.run_as_user", "101"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.security_context.0.supplemental_groups.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.security_context.0.supplemental_groups.0", "101"),
				),
			},
		},
	})
}

func TestAccKubernetesReplicationController_with_container_liveness_probe_using_exec(t *testing.T) {
	var conf api.ReplicationController

	rcName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := "gcr.io/google_containers/busybox"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesReplicationControllerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerConfigWithLivenessProbeUsingExec(rcName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerExists("kubernetes_replication_controller.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.args.#", "3"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.liveness_probe.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.liveness_probe.0.exec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.liveness_probe.0.exec.0.command.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.liveness_probe.0.exec.0.command.0", "cat"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.liveness_probe.0.exec.0.command.1", "/tmp/healthy"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.liveness_probe.0.failure_threshold", "3"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.liveness_probe.0.initial_delay_seconds", "5"),
				),
			},
		},
	})
}

func TestAccKubernetesReplicationController_with_container_liveness_probe_using_http_get(t *testing.T) {
	var conf api.ReplicationController

	rcName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := "gcr.io/google_containers/liveness"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesReplicationControllerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerConfigWithLivenessProbeUsingHTTPGet(rcName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerExists("kubernetes_replication_controller.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.args.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.liveness_probe.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.liveness_probe.0.http_get.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.liveness_probe.0.http_get.0.path", "/healthz"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.liveness_probe.0.http_get.0.port", "8080"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.liveness_probe.0.http_get.0.http_header.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.liveness_probe.0.http_get.0.http_header.0.name", "X-Custom-Header"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.liveness_probe.0.http_get.0.http_header.0.value", "Awesome"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.liveness_probe.0.initial_delay_seconds", "3"),
				),
			},
		},
	})
}

func TestAccKubernetesReplicationController_with_container_liveness_probe_using_tcp(t *testing.T) {
	var conf api.ReplicationController

	rcName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := "gcr.io/google_containers/liveness"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesReplicationControllerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerConfigWithLivenessProbeUsingTCP(rcName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerExists("kubernetes_replication_controller.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.args.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.liveness_probe.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.liveness_probe.0.tcp_socket.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.liveness_probe.0.tcp_socket.0.port", "8080"),
				),
			},
		},
	})
}

func TestAccKubernetesReplicationController_with_container_lifecycle(t *testing.T) {
	var conf api.ReplicationController

	rcName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := "gcr.io/google_containers/liveness"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesReplicationControllerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerConfigWithLifeCycle(rcName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerExists("kubernetes_replication_controller.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.lifecycle.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.lifecycle.0.post_start.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.lifecycle.0.post_start.0.exec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.lifecycle.0.post_start.0.exec.0.command.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.lifecycle.0.post_start.0.exec.0.command.0", "ls"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.lifecycle.0.post_start.0.exec.0.command.1", "-al"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.lifecycle.0.pre_stop.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.lifecycle.0.pre_stop.0.exec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.lifecycle.0.pre_stop.0.exec.0.command.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.lifecycle.0.pre_stop.0.exec.0.command.0", "date"),
				),
			},
		},
	})
}

func TestAccKubernetesReplicationController_with_container_security_context(t *testing.T) {
	var conf api.ReplicationController

	rcName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesReplicationControllerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerConfigWithContainerSecurityContext(rcName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerExists("kubernetes_replication_controller.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.security_context.#", "1"),
				),
			},
		},
	})
}

func TestAccKubernetesReplicationController_with_volume_mount(t *testing.T) {
	var conf api.ReplicationController

	rcName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	secretName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesReplicationControllerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerConfigWithVolumeMounts(secretName, rcName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerExists("kubernetes_replication_controller.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.volume_mount.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.volume_mount.0.mount_path", "/tmp/my_path"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.volume_mount.0.name", "db"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.volume_mount.0.read_only", "false"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.volume_mount.0.sub_path", ""),
				),
			},
		},
	})
}

func TestAccKubernetesReplicationController_with_resource_requirements(t *testing.T) {
	var conf api.ReplicationController

	rcName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesReplicationControllerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerConfigWithResourceRequirements(rcName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerExists("kubernetes_replication_controller.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.resources.0.requests.memory", "50Mi"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.resources.0.requests.cpu", "250m"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.resources.0.limits.memory", "512Mi"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.resources.0.limits.cpu", "500m"),
				),
			},
		},
	})
}

func TestAccKubernetesReplicationController_with_empty_dir_volume(t *testing.T) {
	var conf api.ReplicationController

	rcName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesReplicationControllerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerConfigWithEmptyDirVolumes(rcName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerExists("kubernetes_replication_controller.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.volume_mount.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.volume_mount.0.mount_path", "/cache"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.container.0.volume_mount.0.name", "cache-volume"),
					resource.TestCheckResourceAttr("kubernetes_replication_controller.test", "spec.0.template.0.spec.0.volume.0.empty_dir.0.medium", "Memory"),
				),
			},
		},
	})
}

func testAccCheckKubernetesReplicationControllerDestroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_replication_controller" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.CoreV1().ReplicationControllers(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Replication Controller still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesReplicationControllerExists(n string, obj *api.ReplicationController) resource.TestCheckFunc {
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

		out, err := conn.CoreV1().ReplicationControllers(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccCheckKubernetesReplicationControllerForceNew(old, new *api.ReplicationController, wantNew bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if wantNew {
			if old.ObjectMeta.UID == new.ObjectMeta.UID {
				return fmt.Errorf("Expecting new resource for ReplicationController %s", old.ObjectMeta.UID)
			}
		} else {
			if old.ObjectMeta.UID != new.ObjectMeta.UID {
				return fmt.Errorf("Expecting ReplicationController UIDs to be the same: expected %s got %s", old.ObjectMeta.UID, new.ObjectMeta.UID)
			}
		}
		return nil
	}
}

func testAccKubernetesReplicationControllerConfig_basic(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_replication_controller" "test" {
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
    replicas = 500 # This is intentionally high to exercise the waiter

    selector = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    template {
      metadata {
        labels = {
          TestLabelOne   = "one"
          TestLabelTwo   = "two"
          TestLabelThree = "three"
        }

        annotations = {
          TestAnnotationFive = "five"
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

func testAccKubernetesReplicationControllerConfig_initContainer(name, image string) string {
	return fmt.Sprintf(`resource "kubernetes_replication_controller" "test" {
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
    replicas = 500 # This is intentionally high to exercise the waiter
    selector = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
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
          image = "%s"

          port {
            container_port = 80
          }

          volume_mount {
            name       = "workdir"
            mount_path = "/usr/share/nginx/html"
          }
        }

        init_container {
          name    = "install"
          image   = "%s"
          command = ["wget", "-O", "/work-dir/index.html", "http://kubernetes.io"]

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
          name = "workdir"
          empty_dir {}
        }
      }
    }
  }
}
`, name, image, image)
}

func testAccKubernetesReplicationControllerConfig_modified(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_replication_controller" "test" {
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
    selector = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    template {
      metadata {
        labels = {
          TestLabelOne   = "one"
          TestLabelTwo   = "two"
          TestLabelThree = "three"
        }

        annotations = {
          TestAnnotationSix = "six"
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

func testAccKubernetesReplicationControllerConfig_generatedName(prefix, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_replication_controller" "test" {
  metadata {
    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    generate_name = "%s"
  }

  spec {
    selector = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
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
        }
      }
    }
  }
}
`, prefix, imageName)
}

func testAccKubernetesReplicationControllerConfigWithSecurityContext(rcName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_replication_controller" "test" {
  metadata {
    name = "%s"

    labels = {
      Test = "TfAcceptanceTest"
    }
  }

  spec {
    selector = {
      Test = "TfAcceptanceTest"
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
`, rcName, imageName)
}

func testAccKubernetesReplicationControllerConfigWithSecurityContextRunAsGroup(rcName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_replication_controller" "test" {
  metadata {
    name = "%s"

    labels = {
      Test = "TfAcceptanceTest"
    }
  }

  spec {
    selector = {
      Test = "TfAcceptanceTest"
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
`, rcName, imageName)
}

func testAccKubernetesReplicationControllerConfigWithLivenessProbeUsingExec(rcName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_replication_controller" "test" {
  metadata {
    name = "%s"

    labels = {
      Test = "TfAcceptanceTest"
    }
  }

  spec {
    selector = {
      Test = "TfAcceptanceTest"
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
`, rcName, imageName)
}

func testAccKubernetesReplicationControllerConfigWithLivenessProbeUsingHTTPGet(rcName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_replication_controller" "test" {
  metadata {
    name = "%s"

    labels = {
      Test = "TfAcceptanceTest"
    }
  }

  spec {
    selector = {
      Test = "TfAcceptanceTest"
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
`, rcName, imageName)
}

func testAccKubernetesReplicationControllerConfigWithLivenessProbeUsingTCP(rcName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_replication_controller" "test" {
  metadata {
    name = "%s"

    labels = {
      Test = "TfAcceptanceTest"
    }
  }

  spec {
    selector = {
      Test = "TfAcceptanceTest"
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
`, rcName, imageName)
}

func testAccKubernetesReplicationControllerConfigWithLifeCycle(rcName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_replication_controller" "test" {
  metadata {
    name = "%s"

    labels = {
      Test = "TfAcceptanceTest"
    }
  }

  spec {
    selector = {
      Test = "TfAcceptanceTest"
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
`, rcName, imageName)
}

func testAccKubernetesReplicationControllerConfigWithContainerSecurityContext(rcName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_replication_controller" "test" {
  metadata {
    name = "%s"

    labels = {
      Test = "TfAcceptanceTest"
    }
  }

  spec {
    selector = {
      Test = "TfAcceptanceTest"
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
      }
    }
  }
}
`, rcName, imageName)
}

func testAccKubernetesReplicationControllerConfigWithVolumeMounts(secretName, rcName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_secret" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "first"
  }
}

resource "kubernetes_replication_controller" "test" {
  metadata {
    name = "%s"

    labels = {
      Test = "TfAcceptanceTest"
    }
  }

  spec {
    selector = {
      Test = "TfAcceptanceTest"
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
`, secretName, rcName, imageName)
}

func testAccKubernetesReplicationControllerConfigWithResourceRequirements(rcName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_replication_controller" "test" {
  metadata {
    name = "%s"

    labels = {
      Test = "TfAcceptanceTest"
    }
  }

  spec {
    selector = {
      Test = "TfAcceptanceTest"
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
            }

            requests = {
              cpu    = "250m"
              memory = "50Mi"
            }
          }
        }
      }
    }
  }
}
`, rcName, imageName)
}

func testAccKubernetesReplicationControllerConfigWithEmptyDirVolumes(rcName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_replication_controller" "test" {
  metadata {
    name = "%s"

    labels = {
      Test = "TfAcceptanceTest"
    }
  }

  spec {
    selector = {
      Test = "TfAcceptanceTest"
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
`, rcName, imageName)
}

func testAccKubernetesReplicationControllerConfigMinimal(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_replication_controller" "test" {
  metadata {
    name = "%s"
    labels = {
      test = "%s"
    }
  }
  spec {
    selector = {
      test = "%s"
    }
    template {
      metadata {
        labels = {
          test = "%s"
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
`, name, name, name, name, imageName)
}
