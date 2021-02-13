package kubernetes

import (
	"context"
	"fmt"
	"os"
	"testing"

	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccKubernetesPod_minimal(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-test")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigMinimal(name, busyboxImageVersion),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("kubernetes_pod.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_pod.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_pod.test", "metadata.0.uid"),
				),
			},
			{
				Config:   testAccKubernetesPodConfigMinimal(name, busyboxImageVersion),
				PlanOnly: true,
			},
		},
	})
}

func TestAccKubernetesPod_basic(t *testing.T) {
	var conf1 api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	secretName := acctest.RandomWithPrefix("tf-acc-test")
	configMapName := acctest.RandomWithPrefix("tf-acc-test")

	imageName1 := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigBasic(secretName, configMapName, podName, imageName1),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf1),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "metadata.0.labels.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "metadata.0.labels.app", "pod_label"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "metadata.0.name", podName),
					resource.TestCheckResourceAttrSet("kubernetes_pod.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_pod.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_pod.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.env.0.value_from.0.secret_key_ref.0.name", secretName),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.env.0.value_from.0.secret_key_ref.0.key", "one"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.env.0.value_from.0.secret_key_ref.0.optional", "true"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.env.1.value_from.0.config_map_key_ref.0.name", configMapName),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.env.1.value_from.0.config_map_key_ref.0.key", "one"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.env.1.value_from.0.config_map_key_ref.0.optional", "true"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.env_from.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.env_from.0.config_map_ref.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.env_from.0.config_map_ref.0.name", fmt.Sprintf("%s-from", configMapName)),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.env_from.0.config_map_ref.0.optional", "true"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.env_from.0.prefix", "FROM_CM_"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.env_from.1.secret_ref.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.env_from.1.secret_ref.0.name", fmt.Sprintf("%s-from", secretName)),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.env_from.1.secret_ref.0.optional", "false"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.env_from.1.prefix", "FROM_S_"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.image", imageName1),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.topology_spread_constraint.#", "0"),
				),
			},
			{
				ResourceName:            "kubernetes_pod.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
		},
	})
}

func TestAccKubernetesPod_initContainer_updateForcesNew(t *testing.T) {
	var conf1, conf2 api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	image := busyboxImageVersion
	image1 := busyboxImageVersion1

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigWithInitContainer(podName, image),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf1),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "metadata.0.name", podName),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.name", "container"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.init_container.0.name", "initcontainer"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.init_container.0.image", image),
				),
			},
			{
				Config: testAccKubernetesPodConfigWithInitContainer(podName, image1),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf2),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "metadata.0.name", podName),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.name", "container"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.init_container.0.name", "initcontainer"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.init_container.0.image", image1),
					testAccCheckKubernetesPodForceNew(&conf1, &conf2, true),
				),
			},
		},
	})
}

func TestAccKubernetesPod_updateArgsForceNew(t *testing.T) {
	var conf1 api.Pod
	var conf2 api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")

	imageName := "hashicorp/http-echo:latest"
	argsBefore := `["-listen=:80", "-text='before modification'"]`
	argsAfter := `["-listen=:80", "-text='after modification'"]`

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigArgsUpdate(podName, imageName, argsBefore),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf1),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "metadata.0.name", podName),
					resource.TestCheckResourceAttrSet("kubernetes_pod.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_pod.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_pod.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.args.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.args.0", "-listen=:80"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.args.1", "-text='before modification'"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.name", "containername"),
				),
			},
			{
				Config: testAccKubernetesPodConfigArgsUpdate(podName, imageName, argsAfter),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf2),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "metadata.0.name", podName),
					resource.TestCheckResourceAttrSet("kubernetes_pod.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_pod.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_pod.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.args.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.args.0", "-listen=:80"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.args.1", "-text='after modification'"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.name", "containername"),
					testAccCheckKubernetesPodForceNew(&conf1, &conf2, true),
				),
			},
		},
	})
}

func TestAccKubernetesPod_updateEnvForceNew(t *testing.T) {
	var conf1 api.Pod
	var conf2 api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")

	imageName := "hashicorp/http-echo:latest"
	envBefore := "bar"
	envAfter := "baz"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigEnvUpdate(podName, imageName, envBefore),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf1),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "metadata.0.name", podName),
					resource.TestCheckResourceAttrSet("kubernetes_pod.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_pod.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_pod.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.env.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.env.0.name", "foo"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.env.0.value", "bar"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.name", "containername"),
				),
			},
			{
				Config: testAccKubernetesPodConfigEnvUpdate(podName, imageName, envAfter),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf2),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "metadata.0.name", podName),
					resource.TestCheckResourceAttrSet("kubernetes_pod.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_pod.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_pod.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.env.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.env.0.name", "foo"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.env.0.value", "baz"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.name", "containername"),
					testAccCheckKubernetesPodForceNew(&conf1, &conf2, true),
				),
			},
		},
	})
}

func TestAccKubernetesPod_with_pod_security_context(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigWithSecurityContext(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.security_context.0.fs_group", "100"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.security_context.0.run_as_non_root", "true"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.security_context.0.run_as_user", "101"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.security_context.0.supplemental_groups.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.security_context.0.supplemental_groups.0", "101"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_with_pod_security_context_run_as_group(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfUnsupportedSecurityContextRunAsGroup(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigWithSecurityContextRunAsGroup(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.security_context.0.fs_group", "100"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.security_context.0.run_as_group", "100"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.security_context.0.run_as_non_root", "true"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.security_context.0.run_as_user", "101"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.security_context.0.supplemental_groups.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.security_context.0.supplemental_groups.0", "101"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_with_container_liveness_probe_using_exec(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := "gcr.io/google_containers/busybox"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigWithLivenessProbeUsingExec(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.args.#", "3"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.liveness_probe.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.liveness_probe.0.exec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.liveness_probe.0.exec.0.command.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.liveness_probe.0.exec.0.command.0", "cat"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.liveness_probe.0.exec.0.command.1", "/tmp/healthy"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.liveness_probe.0.failure_threshold", "3"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.liveness_probe.0.initial_delay_seconds", "5"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_with_container_liveness_probe_using_http_get(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := "gcr.io/google_containers/liveness"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigWithLivenessProbeUsingHTTPGet(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.args.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.liveness_probe.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.liveness_probe.0.http_get.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.liveness_probe.0.http_get.0.path", "/healthz"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.liveness_probe.0.http_get.0.port", "8080"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.liveness_probe.0.http_get.0.http_header.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.liveness_probe.0.http_get.0.http_header.0.name", "X-Custom-Header"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.liveness_probe.0.http_get.0.http_header.0.value", "Awesome"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.liveness_probe.0.initial_delay_seconds", "3"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_with_container_liveness_probe_using_tcp(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := "gcr.io/google_containers/liveness"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigWithLivenessProbeUsingTCP(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.args.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.liveness_probe.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.liveness_probe.0.tcp_socket.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.liveness_probe.0.tcp_socket.0.port", "8080"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_with_container_lifecycle(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := "gcr.io/google_containers/liveness"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigWithLifeCycle(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.lifecycle.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.lifecycle.0.post_start.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.lifecycle.0.post_start.0.exec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.lifecycle.0.post_start.0.exec.0.command.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.lifecycle.0.post_start.0.exec.0.command.0", "ls"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.lifecycle.0.post_start.0.exec.0.command.1", "-al"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.lifecycle.0.pre_stop.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.lifecycle.0.pre_stop.0.exec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.lifecycle.0.pre_stop.0.exec.0.command.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.lifecycle.0.pre_stop.0.exec.0.command.0", "date"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_with_container_security_context(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigWithContainerSecurityContext(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.security_context.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.security_context.0.privileged", "true"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.security_context.0.run_as_user", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.security_context.0.se_linux_options.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.security_context.0.se_linux_options.0.level", "s0:c123,c456"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.security_context.0.capabilities.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.security_context.0.capabilities.0.add.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.security_context.0.capabilities.0.add.0", "NET_ADMIN"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.security_context.0.capabilities.0.add.1", "SYS_TIME"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_with_volume_mount(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	secretName := acctest.RandomWithPrefix("tf-acc-test")

	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigWithVolumeMounts(secretName, podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.volume_mount.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.volume_mount.0.mount_path", "/tmp/my_path"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.volume_mount.0.name", "db"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.volume_mount.0.read_only", "false"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.volume_mount.0.sub_path", ""),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.volume_mount.0.mount_propagation", "HostToContainer"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_with_cfg_map_volume_mount(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	cfgMap := acctest.RandomWithPrefix("tf-acc-test")
	imageName := busyboxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigWithConfigMapVolume(cfgMap, podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.volume_mount.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.volume_mount.0.mount_path", "/tmp/my_path"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.volume_mount.0.name", "cfg"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.volume_mount.0.read_only", "false"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.volume_mount.0.sub_path", ""),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.volume_mount.0.mount_propagation", "None"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.volume_mount.1.mount_path", "/tmp/my_raw_path"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.volume_mount.1.name", "cfg-binary"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.volume_mount.1.read_only", "false"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.volume_mount.1.sub_path", ""),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.volume.0.name", "cfg"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.volume.0.config_map.0.name", cfgMap),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.volume.0.config_map.0.default_mode", "0777")),
			},
		},
	})
}

func TestAccKubernetesPod_with_projected_volume(t *testing.T) {
	var conf api.Pod

	cfgMapName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	cfgMap2Name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	secretName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	podName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodProjectedVolume(cfgMapName, cfgMap2Name, secretName, podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.volume.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.volume.0.name", "projected-vol"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.volume.0.projected.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.volume.0.projected.0.default_mode", "0777"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.volume.0.projected.0.sources.#", "4"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.volume.0.projected.0.sources.0.config_map.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.volume.0.projected.0.sources.0.config_map.0.name", cfgMapName),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.volume.0.projected.0.sources.1.config_map.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.volume.0.projected.0.sources.1.config_map.0.name", cfgMap2Name),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.volume.0.projected.0.sources.2.secret.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.volume.0.projected.0.sources.2.secret.0.name", secretName),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.volume.0.projected.0.sources.3.downward_api.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.volume.0.projected.0.sources.3.downward_api.0.items.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.volume.0.projected.0.sources.3.downward_api.0.items.0.path", "labels"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.volume.0.projected.0.sources.3.downward_api.0.items.0.field_ref.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.volume.0.projected.0.sources.3.downward_api.0.items.0.field_ref.0.field_path", "metadata.labels"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.volume.0.projected.0.sources.3.downward_api.0.items.1.path", "cpu_limit"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.volume.0.projected.0.sources.3.downward_api.0.items.1.resource_field_ref.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.volume.0.projected.0.sources.3.downward_api.0.items.1.resource_field_ref.0.container_name", "containername"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.volume.0.projected.0.sources.3.downward_api.0.items.1.resource_field_ref.0.resource", "limits.cpu"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_with_resource_requirements(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigWithResourceRequirements(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.resources.0.requests.memory", "50Mi"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.resources.0.requests.cpu", "250m"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.resources.0.requests.ephemeral-storage", "128Mi"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.resources.0.limits.memory", "512Mi"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.resources.0.limits.cpu", "500m"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.resources.0.limits.ephemeral-storage", "512Mi"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_with_empty_dir_volume(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigWithEmptyDirVolumes(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.volume_mount.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.volume_mount.0.mount_path", "/cache"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.volume_mount.0.name", "cache-volume"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.volume.0.empty_dir.0.medium", "Memory"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_with_empty_dir_volume_with_sizeLimit(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigWithEmptyDirVolumesSizeLimit(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.volume_mount.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.volume_mount.0.mount_path", "/cache"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.volume_mount.0.name", "cache-volume"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.volume.0.empty_dir.0.medium", "Memory"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.volume.0.empty_dir.0.size_limit", "512Mi"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_with_secret_vol_items(t *testing.T) {
	var conf api.Pod

	secretName := acctest.RandomWithPrefix("tf-acc-test")
	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigWithSecretItemsVolume(secretName, podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.volume.0.secret.0.items.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.volume.0.secret.0.items.0.key", "one"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.volume.0.secret.0.items.0.path", "path/to/one"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_gke_with_nodeSelector(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := nginxImageVersion
	region := os.Getenv("GOOGLE_REGION")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotRunningInGke(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigNodeSelector(podName, imageName, region),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.node_selector.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.node_selector.failure-domain.beta.kubernetes.io/region", region),
				),
			},
		},
	})
}

func TestAccKubernetesPod_config_with_automount_service_account_token(t *testing.T) {
	var confPod api.Pod
	var confSA api.ServiceAccount

	podName := acctest.RandomWithPrefix("tf-acc-test")
	saName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigWithAutomountServiceAccountToken(saName, podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceAccountExists("kubernetes_service_account.test", &confSA),
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &confPod),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.automount_service_account_token", "true"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.volume_mount.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.volume_mount.0.mount_path", "/var/run/secrets/kubernetes.io/serviceaccount"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.volume.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.volume.0.secret.#", "1"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_config_container_working_dir(t *testing.T) {
	var confPod api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigWorkingDir(podName, imageName, "/www"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &confPod),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "metadata.0.generation", "0"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.working_dir", "/www"),
				),
			},
			{
				Config: testAccKubernetesPodConfigWorkingDir(podName, imageName, "/srv"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &confPod),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "metadata.0.generation", "0"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.working_dir", "/srv"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_config_container_startup_probe(t *testing.T) {
	var confPod api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.17.0")
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodContainerStartupProbe(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &confPod),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "metadata.0.generation", "0"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.startup_probe.0.http_get.0.path", "/index.html"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.startup_probe.0.http_get.0.port", "80"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.startup_probe.0.initial_delay_seconds", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.startup_probe.0.timeout_seconds", "2"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_termination_message_policy_default(t *testing.T) {
	var confPod api.Pod

	podName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesTerminationMessagePolicyDefault(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &confPod),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.termination_message_policy", "File"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_termination_message_policy_override_as_file(t *testing.T) {
	var confPod api.Pod

	podName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesTerminationMessagePolicyWithFile(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &confPod),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.termination_message_policy", "File"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_termination_message_policy_override_as_fallback_to_logs_on_err(t *testing.T) {
	var confPod api.Pod

	podName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesTerminationMessagePolicyWithFallBackToLogsOnErr(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &confPod),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.termination_message_policy", "FallbackToLogsOnError"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_enableServiceLinks(t *testing.T) {
	var conf1 api.Pod

	rName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigEnableServiceLinks(rName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf1),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "metadata.0.labels.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "metadata.0.labels.app", "pod_label"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "metadata.0.name", rName),
					resource.TestCheckResourceAttrSet("kubernetes_pod.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_pod.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_pod.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.enable_service_links", "false"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_bug961EmptyBlocks(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-test")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigEmptyBlocks(name, busyboxImageVersion),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("kubernetes_pod.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_pod.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_pod.test", "metadata.0.uid"),
					resource.TestCheckResourceAttrSet("kubernetes_pod.test", "metadata.0.uid"),
				),
			},
			{
				Config:   testAccKubernetesPodConfigMinimal(name, busyboxImageVersion),
				PlanOnly: true,
			},
		},
	})
}

func TestAccKubernetesPod_bug1085(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-test")
	imageName := alpineImageVersion
	var conf api.Pod

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotRunningInMinikube(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigWithVolume(name, imageName, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.service_account_name", "default"),
				),
			},
			{
				Config: testAccKubernetesPodConfigWithVolume(name, imageName, `service_account_name="test"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.service_account_name", "test"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_readinessGate(t *testing.T) {
	var conf1 api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	secretName := acctest.RandomWithPrefix("tf-acc-test")
	configMapName := acctest.RandomWithPrefix("tf-acc-test")
	imageName1 := nginxImageVersion1

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigBasic(secretName, configMapName, podName, imageName1),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf1),
				),
			},
			{
				Config: testAccKubernetesPodConfigReadinessGate(secretName, configMapName, podName, imageName1),
				PreConfig: func() {
					conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
					if err != nil {
						t.Fatal(err)
					}
					ctx := context.TODO()

					conditions := conf1.Status.Conditions
					testCondition := api.PodCondition{
						Type:   api.PodConditionType("haha"),
						Status: api.ConditionTrue,
					}
					updatedConditions := append(conditions, testCondition)
					conf1.Status.Conditions = updatedConditions
					_, err = conn.CoreV1().Pods("default").UpdateStatus(ctx, &conf1, metav1.UpdateOptions{})
					if err != nil {
						t.Fatal(err)
					}
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf1),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.readiness_gate.0.condition_type", "haha"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_topologySpreadConstraint(t *testing.T) {
	var conf1 api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_pod.test"
	imageName := "nginx:1.7.9"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodTopologySpreadConstraintConfig(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "spec.0.topology_spread_constraint.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.topology_spread_constraint.0.max_skew", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.topology_spread_constraint.0.topology_key", "failure-domain.beta.kubernetes.io/zone"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.topology_spread_constraint.0.when_unsatisfiable", "ScheduleAnyway"),
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

func testAccCheckKubernetesPodDestroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_pod" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Namespace == namespace && resp.Name == name {
				return fmt.Errorf("Pod still exists: %s: %#v", rs.Primary.ID, resp.Status.Phase)
			}
		}
	}

	return nil
}

func TestAccKubernetesPod_regression(t *testing.T) {
	var conf1, conf2 api.Pod

	name := acctest.RandomWithPrefix("tf-acc-test")
	imageName := nginxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ExternalProviders: testAccExternalProviders,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: requiredProviders() + testAccKubernetesPodConfig_beforeUpdate(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf1),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.image", imageName),
				),
			},
			{
				Config: requiredProviders() + testAccKubernetesPodConfig_afterUpdate(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf2),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.container.0.image", imageName),
					testAccCheckKubernetesPodForceNew(&conf1, &conf2, false),
				),
			},
		},
	})
}

func testAccCheckKubernetesPodExists(n string, obj *api.Pod) resource.TestCheckFunc {
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

		out, err := conn.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		*obj = *out
		return nil
	}
}

func testAccCheckKubernetesPodForceNew(old, new *api.Pod, wantNew bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if wantNew {
			if old.ObjectMeta.UID == new.ObjectMeta.UID {
				return fmt.Errorf("Expecting new resource for pod %s", old.ObjectMeta.UID)
			}
		} else {
			if old.ObjectMeta.UID != new.ObjectMeta.UID {
				return fmt.Errorf("Expecting pod UIDs to be the same: expected %s got %s", old.ObjectMeta.UID, new.ObjectMeta.UID)
			}
		}
		return nil
	}
}

func testAccKubernetesPodConfigBasic(secretName, configMapName, podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_secret" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "first"
  }
}

resource "kubernetes_secret" "test_from" {
  metadata {
    name = "%s-from"
  }

  data = {
    one    = "first_from"
    second = "second_from"
  }
}

resource "kubernetes_config_map" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "ONE"
  }
}

resource "kubernetes_config_map" "test_from" {
  metadata {
    name = "%s-from"
  }

  data = {
    one = "ONE_FROM"
    two = "TWO_FROM"
  }
}

resource "kubernetes_pod" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }

    name = "%s"
  }

  spec {
	automount_service_account_token = false

    container {
      image = "%s"
      name  = "containername"

      env {
        name = "EXPORTED_VARIABLE_FROM_SECRET"

        value_from {
          secret_key_ref {
            name     = "${kubernetes_secret.test.metadata.0.name}"
            key      = "one"
            optional = true
          }
        }
      }
      env {
        name = "EXPORTED_VARIABLE_FROM_CONFIG_MAP"
        value_from {
          config_map_key_ref {
            name     = "${kubernetes_config_map.test.metadata.0.name}"
            key      = "one"
            optional = true
          }
        }
      }

      env_from {
        config_map_ref {
          name     = "${kubernetes_config_map.test_from.metadata.0.name}"
          optional = true
        }
        prefix = "FROM_CM_"
      }
      env_from {
        secret_ref {
          name     = "${kubernetes_secret.test_from.metadata.0.name}"
          optional = false
        }
        prefix = "FROM_S_"
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
`, secretName, secretName, configMapName, configMapName, podName, imageName)
}

func testAccKubernetesPodConfigWithInitContainer(podName, image string) string {
	return fmt.Sprintf(`resource "kubernetes_pod" "test" {
  metadata {
    name = "%s"
  }

  spec {
  automount_service_account_token = false
   container {
     name  = "container"
     image = "%s"
     command = ["sh", "-c", "echo The app is running! && sleep 3600"]

     resources {
       requests = {
         memory = "64Mi"
         cpu    = "50m"
       }
     }
   }

    init_container {
      name    = "initcontainer"
      image   = "%s"
      command = ["sh", "-c", "until nslookup init-service.default.svc.cluster.local; do echo waiting for init-service; sleep 2; done"]

      resources {
        requests = {
          memory = "64Mi"
          cpu    = "50m"
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
`, podName, image, image)
}

func testAccKubernetesPodConfigWithSecurityContext(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }

    name = "%s"
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
`, podName, imageName)
}

func testAccKubernetesPodConfigWithSecurityContextRunAsGroup(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }

    name = "%s"
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
`, podName, imageName)
}

func testAccKubernetesPodConfigWithLivenessProbeUsingExec(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }

    name = "%s"
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
`, podName, imageName)
}

func testAccKubernetesPodConfigWithLivenessProbeUsingHTTPGet(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }

    name = "%s"
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
`, podName, imageName)
}

func testAccKubernetesPodConfigWithLivenessProbeUsingTCP(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }

    name = "%s"
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
`, podName, imageName)
}

func testAccKubernetesPodConfigWithLifeCycle(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }

    name = "%s"
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
`, podName, imageName)
}

func testAccKubernetesPodConfigWithContainerSecurityContext(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }

    name = "%s"
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

        capabilities {
          add = ["NET_ADMIN", "SYS_TIME"]
        }
      }
    }
  }
}
`, podName, imageName)
}

func testAccKubernetesPodConfigWithVolumeMounts(secretName, podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_secret" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "first"
  }
}

resource "kubernetes_pod" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }

    name = "%s"
  }

  spec {
	automount_service_account_token = false
	
	container {
      image = "%s"
      name  = "containername"

      volume_mount {
        mount_path        = "/tmp/my_path"
        name              = "db"
        mount_propagation = "HostToContainer"
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
`, secretName, podName, imageName)
}

func testAccKubernetesPodConfigWithSecretItemsVolume(secretName, podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_secret" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "first"
  }
}

resource "kubernetes_pod" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }

    name = "%s"
  }

  spec {
	automount_service_account_token = false

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

        items {
          key  = "one"
          path = "path/to/one"
        }
      }
    }
  }
}
`, secretName, podName, imageName)
}

func testAccKubernetesPodConfigWithConfigMapVolume(secretName, podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map" "test" {
  metadata {
    name = "%s"
  }

  binary_data = {
    raw = "${base64encode("Raw data should come back as is in the pod")}"
  }

  data = {
    one = "first"
  }
}

resource "kubernetes_pod" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }

    name = "%s"
  }

  spec {
	restart_policy = "Never"
	automount_service_account_token = false

    container {
      image = "%s"
      name  = "containername"

      args = ["/bin/sh", "-xc", "ls -l /tmp/my_raw_path ; cat /tmp/my_raw_path/raw.txt ; sleep 10"]

      lifecycle {
        post_start {
          exec {
            command = ["/bin/sh", "-xc", "grep 'Raw data should come back as is in the pod' /tmp/my_raw_path/raw.txt"]
          }
        }
      }

      volume_mount {
        mount_path = "/tmp/my_path"
        name       = "cfg"
      }

      volume_mount {
        mount_path = "/tmp/my_raw_path"
        name       = "cfg-binary"
      }
    }

    volume {
      name = "cfg"

      config_map {
        name         = "${kubernetes_config_map.test.metadata.0.name}"
        default_mode = "0777"
      }
    }

    volume {
      name = "cfg-item"

      config_map {
        name = "${kubernetes_config_map.test.metadata.0.name}"

        items {
          key  = "one"
          path = "one.txt"
        }
      }
    }

    volume {
      name = "cfg-item-with-mode"

      config_map {
        name = "${kubernetes_config_map.test.metadata.0.name}"

        items {
          key  = "one"
          path = "one-with-mode.txt"
          mode = "0444"
        }
      }
    }

    volume {
      name = "cfg-binary"

      config_map {
        name = "${kubernetes_config_map.test.metadata.0.name}"

        items {
          key  = "raw"
          path = "raw.txt"
        }
      }
    }
  }
}
`, secretName, podName, imageName)
}

func testAccKubernetesPodProjectedVolume(cfgMapName, cfgMap2Name, secretName, podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map" "test" {
    metadata {
      name = "%s"
    }

    binary_data = {
      raw = "${base64encode("Raw data should come back as is in the pod")}"
    }

    data = {
      one = "first"
    }
}

resource "kubernetes_config_map" "test2" {
    metadata {
      name = "%s"
    }

    binary_data = {
      raw = "${base64encode("Raw data should come back as is in the pod")}"
    }

    data = {
      one = "first"
    }
}

resource "kubernetes_secret" "test" {
    metadata {
      name = "%s"
    }

    data = {
      one = "first"
    }
}

resource "kubernetes_pod" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }

    name = "%s"
  }

  spec {
	restart_policy = "Never"
	automount_service_account_token = false

    container {
      image = "%s"
      name  = "containername"

    command = ["sleep", "3600"]

      lifecycle {
        post_start {
          exec {
            command = ["/bin/sh", "-xc", "grep 'Raw data should come back as is in the pod' /tmp/my-projected-volume/raw.txt"]
          }
        }
      }

      volume_mount {
        mount_path = "/tmp/my-projected-volume"
        name       = "projected-vol"
      }
    }

    volume {
      name = "projected-vol"
      projected {
        default_mode = "0777"
        sources {
          config_map {
            name = "${kubernetes_config_map.test.metadata.0.name}"
            items {
              key  = "raw"
              path = "raw.txt"
            }
      }
    }
    sources {
      config_map {
            name = "${kubernetes_config_map.test2.metadata.0.name}"
            items {
              key  = "raw"
              path = "raw-again.txt"
            }
      }
    }
    sources {
          secret {
            name = "${kubernetes_secret.test.metadata.0.name}"
            items {
              key  = "one"
              path = "secret.txt"
            }
      }
    }
    sources {
          downward_api {
            items {
              path = "labels"
              field_ref {
                field_path = "metadata.labels"
              }
            }
            items {
              path = "cpu_limit"
              resource_field_ref {
                container_name = "containername"
                resource = "limits.cpu"
              }
            }
      }
        }
      }
    }
  }
}
`, cfgMapName, cfgMap2Name, secretName, podName, imageName)
}

func testAccKubernetesPodConfigWithResourceRequirements(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }

    name = "%s"
  }

  spec {
    container {
      image = "%s"
      name  = "containername"

      resources {
        limits = {
          cpu    = "0.5"
		  memory = "512Mi"
		  "ephemeral-storage" = "512Mi"
        }

        requests = {
          cpu    = "250m"
		  memory = "50Mi"
		  "ephemeral-storage" = "128Mi"
        }
      }
    }
  }
}
`, podName, imageName)
}

func testAccKubernetesPodConfigWithEmptyDirVolumes(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }

    name = "%s"
  }

  spec {
	automount_service_account_token = false

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
`, podName, imageName)
}

func testAccKubernetesPodConfigWithEmptyDirVolumesSizeLimit(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }

    name = "%s"
  }

  spec {
	automount_service_account_token = false

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
        medium     = "Memory"
        size_limit = "512Mi"
      }
    }
  }
}
`, podName, imageName)
}

func testAccKubernetesPodConfigNodeSelector(podName, imageName, region string) string {
	return fmt.Sprintf(`resource "kubernetes_pod" "test" {
  metadata {
    name = "%s"
  }

  spec {
    container {
      image = "%s"
      name  = "containername"
    }

    node_selector = {
      "failure-domain.beta.kubernetes.io/region" = "%s"
    }
  }
}
`, podName, imageName, region)
}

func testAccKubernetesPodConfigArgsUpdate(podName, imageName, args string) string {
	return fmt.Sprintf(`resource "kubernetes_pod" "test" {
  metadata {
    name = "%s"
  }

  spec {
    container {
      image = "%s"
      args  = %s
      name  = "containername"
    }
  }
}
`, podName, imageName, args)
}

func testAccKubernetesPodConfigEnvUpdate(podName, imageName, val string) string {
	return fmt.Sprintf(`resource "kubernetes_pod" "test" {
  metadata {
    name = "%s"
  }

  spec {
    container {
      image = "%s"
      name  = "containername"

      env {
        name  = "foo"
        value = "%s"
      }
    }
  }
}
`, podName, imageName, val)
}

func testAccKubernetesPodConfigWithAutomountServiceAccountToken(saName string, podName string, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_service_account" "test" {
  metadata {
    name = "%s"
  }
}

resource "kubernetes_pod" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }

    name = "%s"
  }

  spec {
    service_account_name            = kubernetes_service_account.test.metadata.0.name
    automount_service_account_token = true

    container {
      image = "%s"
      name  = "containername"

      lifecycle {
        post_start {
          exec {
            command = ["/bin/sh", "-xc", "mount | grep /run/secrets/kubernetes.io/serviceaccount"]
          }
        }
      }
    }
  }
}
`, saName, podName, imageName)
}

func testAccKubernetesPodConfigWorkingDir(podName, imageName, val string) string {
	return fmt.Sprintf(`resource "kubernetes_pod" "test" {
  metadata {
    name = "%s"
  }

  spec {
    container {
      image       = "%s"
      name        = "containername"
      working_dir = "%s"
    }
  }
}
`, podName, imageName, val)
}

func testAccKubernetesPodContainerStartupProbe(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod" "test" {
  metadata {
    name = "%s"
  }

  spec {
    container {
      image = "%s"
      name  = "containername"

      startup_probe {
        http_get {
          path = "/index.html"
          port = 80
        }

        initial_delay_seconds = 1
        timeout_seconds       = 2
      }
    }
  }
}
`, podName, imageName)
}

func testAccKubernetesTerminationMessagePolicyDefault(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod" "test" {
  metadata {
    name = "%s"
  }

  spec {
    container {
      image = "%s"
      name  = "containername"
    }
  }
}
`, podName, imageName)
}

func testAccKubernetesTerminationMessagePolicyWithOverride(podName, imageName, terminationMessagePolicy string) string {
	return fmt.Sprintf(`resource "kubernetes_pod" "test" {
  metadata {
    name = "%s"
  }

  spec {
    container {
      image                      = "%s"
      name                       = "containername"
      termination_message_policy = "%s"
    }
  }
}
`, podName, imageName, terminationMessagePolicy)
}

func testAccKubernetesTerminationMessagePolicyWithFile(podName, imageName string) string {
	return testAccKubernetesTerminationMessagePolicyWithOverride(podName, imageName, "File")
}

func testAccKubernetesTerminationMessagePolicyWithFallBackToLogsOnErr(podName, imageName string) string {
	return testAccKubernetesTerminationMessagePolicyWithOverride(podName, imageName, "FallbackToLogsOnError")
}

func testAccKubernetesPodConfigEnableServiceLinks(podName, imageName string) string {
	return fmt.Sprintf(`
resource "kubernetes_pod" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }

    name = "%s"
  }

  spec {
    container {
      image = "%s"
      name  = "containername"
    }
    enable_service_links = false
  }
}
`, podName, imageName)
}

func testAccKubernetesPodConfigReadinessGate(secretName, configMapName, podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_secret" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "first"
  }
}

resource "kubernetes_secret" "test_from" {
  metadata {
    name = "%s-from"
  }

  data = {
    one    = "first_from"
    second = "second_from"
  }
}

resource "kubernetes_config_map" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "ONE"
  }
}

resource "kubernetes_config_map" "test_from" {
  metadata {
    name = "%s-from"
  }

  data = {
    one = "ONE_FROM"
    two = "TWO_FROM"
  }
}

resource "kubernetes_pod" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }

    name = "%s"
  }

  spec {
	automount_service_account_token = false

    readiness_gate {
      condition_type = "haha"
    }
    container {
      image = "%s"
      name  = "containername"

      env {
        name = "EXPORTED_VARIABLE_FROM_SECRET"

        value_from {
          secret_key_ref {
            name     = "${kubernetes_secret.test.metadata.0.name}"
            key      = "one"
            optional = true
          }
        }
      }
      env {
        name = "EXPORTED_VARIABLE_FROM_CONFIG_MAP"
        value_from {
          config_map_key_ref {
            name     = "${kubernetes_config_map.test.metadata.0.name}"
            key      = "one"
            optional = true
          }
        }
      }

      env_from {
        config_map_ref {
          name     = "${kubernetes_config_map.test_from.metadata.0.name}"
          optional = true
        }
        prefix = "FROM_CM_"
      }
      env_from {
        secret_ref {
          name     = "${kubernetes_secret.test_from.metadata.0.name}"
          optional = false
        }
        prefix = "FROM_S_"
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
`, secretName, secretName, configMapName, configMapName, podName, imageName)
}

func testAccKubernetesPodConfigMinimal(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod" "test" {
  metadata {
    name = "%s"
  }
  spec {
    container {
        image = "%s"
        name  = "containername"
    }
  }
}
`, name, imageName)
}

func testAccKubernetesPodConfigEmptyBlocks(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }

    name = "%s"
  }

  spec {
    container {
      image = "%s"
      name  = "containername"

      env {}
      env_from {
        config_map_ref {}
      }
      env_from {
        secret_ref {}
        }
      env_from {}
    }
    volume {
      name = "empty"
      secret {}
    }
    volume {}
  }
}
`, name, imageName)
}

func testAccKubernetesPodConfigWithVolume(name, imageName, serviceAccount string) string {
	return fmt.Sprintf(`resource "kubernetes_storage_class" "test" {
  metadata {
    name = "test"
  }
  storage_provisioner = "k8s.io/minikube-hostpath"
}

resource "kubernetes_service_account" "test" {
  metadata {
    name = "test"
  }
}

resource "kubernetes_persistent_volume" "test" {
  metadata {
    name = "test"
  }
  spec {
    capacity = {
      storage = "1Gi"
    }
    access_modes = ["ReadWriteOnce"]
    storage_class_name = kubernetes_storage_class.test.metadata.0.name
    persistent_volume_source {
      host_path {
        path = "/mnt/minikube"
        type = "DirectoryOrCreate"
      }
    }
  }
}

resource "kubernetes_persistent_volume_claim" "test" {
  wait_until_bound = false
  metadata {
    name = "test"
  }
  spec {
    access_modes = ["ReadWriteOnce"]
    storage_class_name = kubernetes_storage_class.test.metadata.0.name
    volume_name = kubernetes_persistent_volume.test.metadata.0.name
    resources {
      requests = {
        storage = "1G"
      }
    }
  }
}

resource "kubernetes_pod" "test" {
  metadata {
    name = "%s"
  }
  spec {
    %s
    container {
      name = "default"
      image = "%s"
      command = ["sleep", "3600s"]
      volume_mount {
        mount_path = "/etc/test"
        name = "pvc"
      }
    }
    volume {
      name = "pvc"
      persistent_volume_claim {
        claim_name = kubernetes_persistent_volume_claim.test.metadata[0].name
      }
    }
  }
}
`, name, serviceAccount, imageName)
}

func testAccKubernetesPodConfig_beforeUpdate(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod" "test" {
  provider = kubernetes-released
  metadata {
    name = "%s"
  }
  spec {
	automount_service_account_token = false
    container {
      image = "%s"
      name  = "containername"
      resources {
        limits {
          memory = "512M"
          cpu = "1"
        }
        requests {
          memory = "256M"
          cpu = "50m"
        }
      }
    }
  }
}
`, name, imageName)
}

func testAccKubernetesPodConfig_afterUpdate(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod" "test" {
  provider = kubernetes-local
  metadata {
    name = "%s"
  }
  spec {
	automount_service_account_token = false
    container {
      image = "%s"
      name  = "containername"
      resources {
        limits = {
          memory = "512M"
          cpu = "1"
        }
        requests = {
          memory = "256M"
          cpu = "50m"
        }
      }
    }
  }
}
`, name, imageName)
}

func testAccKubernetesPodTopologySpreadConstraintConfig(podName, imageName string) string {
	return fmt.Sprintf(`
resource "kubernetes_pod" "test" {
  metadata {
    name = "%s"
  }
  spec {
    container {
      image = "%s"
      name  = "containername"
    }
    topology_spread_constraint {
      max_skew           = 1
      topology_key       = "failure-domain.beta.kubernetes.io/zone"
      when_unsatisfiable = "ScheduleAnyway"
      label_selector {
        match_labels = {
          "app.kubernetes.io/instance" = "terraform-example"
        }
      }
    }
  }
}
`, podName, imageName)
}
