// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesPodV1_minimal(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_pod_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigMinimal(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config:   testAccKubernetesPodV1ConfigMinimal(name, imageName),
				PlanOnly: true,
			},
		},
	})
}

func TestAccKubernetesPodV1_basic(t *testing.T) {
	var conf1 api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	secretName := acctest.RandomWithPrefix("tf-acc-test")
	configMapName := acctest.RandomWithPrefix("tf-acc-test")

	imageName1 := busyboxImage
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigBasic(secretName, configMapName, podName, imageName1),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.app", "pod_label"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", podName),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.env.0.value_from.0.secret_key_ref.0.name", secretName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.env.0.value_from.0.secret_key_ref.0.key", "one"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.env.0.value_from.0.secret_key_ref.0.optional", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.env.1.value_from.0.config_map_key_ref.0.name", configMapName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.env.1.value_from.0.config_map_key_ref.0.key", "one"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.env.1.value_from.0.config_map_key_ref.0.optional", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.env_from.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.env_from.0.config_map_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.env_from.0.config_map_ref.0.name", fmt.Sprintf("%s-from", configMapName)),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.env_from.0.config_map_ref.0.optional", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.env_from.0.prefix", "FROM_CM_"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.env_from.1.secret_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.env_from.1.secret_ref.0.name", fmt.Sprintf("%s-from", secretName)),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.env_from.1.secret_ref.0.optional", "false"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.env_from.1.prefix", "FROM_S_"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.image", imageName1),
					resource.TestCheckResourceAttr(resourceName, "spec.0.topology_spread_constraint.#", "0"),
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

func TestAccKubernetesPodV1_scheduler(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	schedulerName := acctest.RandomWithPrefix("test-scheduler")
	imageName := busyboxImage
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.25.0")
			skipIfRunningInAks(t)
			setClusterVersionVar(t, "TF_VAR_scheduler_cluster_version") // should be in format 'vX.Y.Z'
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesCustomScheduler(schedulerName),
			},
			{
				Config: testAccKubernetesCustomScheduler(schedulerName) +
					testAccKubernetesPodV1ConfigScheduler(podName, schedulerName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", podName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scheduler_name", schedulerName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.image", imageName),
				),
			},
		},
	})
}

func TestAccKubernetesPodV1_initContainer_updateForcesNew(t *testing.T) {
	var conf1, conf2 api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	image := busyboxImage
	image1 := agnhostImage
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesPodV1ConfigWithInitContainer(podName, image),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", podName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.name", "container"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.init_container.0.name", "initcontainer"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.init_container.0.image", image),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesConfig_ignoreAnnotations() +
					testAccKubernetesPodV1ConfigWithInitContainer(podName, image1),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", podName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.name", "container"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.init_container.0.name", "initcontainer"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.init_container.0.image", image1),
					testAccCheckKubernetesPodForceNew(&conf1, &conf2, true),
				),
			},
		},
	})
}

func TestAccKubernetesPodV1_updateArgsForceNew(t *testing.T) {
	var conf1 api.Pod
	var conf2 api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")

	imageName := busyboxImage
	argsBefore := `["sleep", "60"]`
	argsAfter := `["sleep", "300"]`
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigArgsUpdate(podName, imageName, argsBefore),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", podName),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.args.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.args.0", "sleep"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.args.1", "60"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.name", "containername"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesPodV1ConfigArgsUpdate(podName, imageName, argsAfter),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", podName),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.args.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.args.0", "sleep"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.args.1", "300"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.name", "containername"),
					testAccCheckKubernetesPodForceNew(&conf1, &conf2, true),
				),
			},
		},
	})
}

func TestAccKubernetesPodV1_updateEnvForceNew(t *testing.T) {
	var conf1 api.Pod
	var conf2 api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")

	imageName := busyboxImage
	envBefore := "bar"
	envAfter := "baz"
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigEnvUpdate(podName, imageName, envBefore),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", podName),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.env.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.env.0.name", "foo"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.env.0.value", "bar"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.name", "containername"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesPodV1ConfigEnvUpdate(podName, imageName, envAfter),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", podName),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.env.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.env.0.name", "foo"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.env.0.value", "baz"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.name", "containername"),
					testAccCheckKubernetesPodForceNew(&conf1, &conf2, true),
				),
			},
		},
	})
}

func TestAccKubernetesPodV1_with_pod_security_context(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := busyboxImage
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigWithSecurityContext(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.security_context.0.fs_group", "100"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.security_context.0.run_as_non_root", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.security_context.0.run_as_user", "101"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.security_context.0.supplemental_groups.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.security_context.0.supplemental_groups.0", "101"),
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

func TestAccKubernetesPodV1_with_pod_security_context_fs_group_change_policy(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := busyboxImage
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfUnsupportedSecurityContextRunAsGroup(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigWithSecurityContextFSChangePolicy(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.security_context.0.fs_group", "100"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.security_context.0.run_as_group", "100"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.security_context.0.run_as_non_root", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.security_context.0.run_as_user", "101"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.security_context.0.fs_group_change_policy", "OnRootMismatch"),
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

func testAccKubernetesPodV1ConfigWithSecurityContextFSChangePolicy(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }
    name = "%s"
  }
  spec {
    security_context {
      fs_group               = 100
      run_as_group           = 100
      run_as_non_root        = true
      run_as_user            = 101
      fs_group_change_policy = "OnRootMismatch"
    }
    container {
      image = "%s"
      name  = "containername"
    }
  }
}
`, podName, imageName)
}

func TestAccKubernetesPodV1_with_pod_security_context_run_as_group(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := busyboxImage
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfUnsupportedSecurityContextRunAsGroup(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigWithSecurityContextRunAsGroup(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.security_context.0.fs_group", "100"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.security_context.0.run_as_group", "100"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.security_context.0.run_as_non_root", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.security_context.0.run_as_user", "101"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.security_context.0.supplemental_groups.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.security_context.0.supplemental_groups.0", "101"),
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

func TestAccKubernetesPodV1_with_pod_security_context_seccomp_profile(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := busyboxImage
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigWithSecurityContextSeccompProfile(podName, imageName, "Unconfined"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.security_context.0.seccomp_profile.0.type", "Unconfined"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.security_context.0.seccomp_profile.0.type", "Unconfined"),
				),
			},
			{
				Config: testAccKubernetesPodV1ConfigWithSecurityContextSeccompProfile(podName, imageName, "RuntimeDefault"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.security_context.0.seccomp_profile.0.type", "RuntimeDefault"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.security_context.0.seccomp_profile.0.type", "RuntimeDefault"),
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

func TestAccKubernetesPodV1_with_pod_security_context_seccomp_localhost_profile(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := busyboxImage
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotRunningInKind(t); skipIfClusterVersionLessThan(t, "1.19.0") },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigWithSecurityContextSeccompProfileLocalhost(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.security_context.0.seccomp_profile.0.type", "Localhost"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.security_context.0.seccomp_profile.0.localhost_profile", "profiles/audit.json"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.security_context.0.seccomp_profile.0.type", "Localhost"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.security_context.0.seccomp_profile.0.localhost_profile", "profiles/audit.json"),
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

func TestAccKubernetesPodV1_with_container_liveness_probe_using_exec(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := agnhostImage
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigWithLivenessProbeUsingExec(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.args.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.liveness_probe.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.liveness_probe.0.exec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.liveness_probe.0.exec.0.command.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.liveness_probe.0.exec.0.command.0", "cat"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.liveness_probe.0.exec.0.command.1", "/tmp/healthy"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.liveness_probe.0.failure_threshold", "3"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.liveness_probe.0.initial_delay_seconds", "3"),
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

func TestAccKubernetesPodV1_with_container_liveness_probe_using_http_get(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := agnhostImage
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigWithLivenessProbeUsingHTTPGet(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.args.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.liveness_probe.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.liveness_probe.0.http_get.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.liveness_probe.0.http_get.0.path", "/healthz"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.liveness_probe.0.http_get.0.port", "8080"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.liveness_probe.0.http_get.0.http_header.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.liveness_probe.0.http_get.0.http_header.0.name", "X-Custom-Header"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.liveness_probe.0.http_get.0.http_header.0.value", "Awesome"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.liveness_probe.0.initial_delay_seconds", "3"),
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

func TestAccKubernetesPodV1_with_container_liveness_probe_using_tcp(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := agnhostImage
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigWithLivenessProbeUsingTCP(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.args.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.liveness_probe.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.liveness_probe.0.tcp_socket.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.liveness_probe.0.tcp_socket.0.port", "8080"),
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

func TestAccKubernetesPodV1_with_container_liveness_probe_using_grpc(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := agnhostImage
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.24.0")
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigWithLivenessProbeUsingGRPC(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.args.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.liveness_probe.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.liveness_probe.0.grpc.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.liveness_probe.0.grpc.0.port", "8888"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.liveness_probe.0.grpc.0.service", "EchoService"),
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

func TestAccKubernetesPodV1_with_container_lifecycle(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := agnhostImage
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigWithLifeCycle(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.lifecycle.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.lifecycle.0.post_start.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.lifecycle.0.post_start.0.exec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.lifecycle.0.post_start.0.exec.0.command.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.lifecycle.0.post_start.0.exec.0.command.0", "ls"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.lifecycle.0.post_start.0.exec.0.command.1", "-al"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.lifecycle.0.pre_stop.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.lifecycle.0.pre_stop.0.exec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.lifecycle.0.pre_stop.0.exec.0.command.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.lifecycle.0.pre_stop.0.exec.0.command.0", "date"),
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

func TestAccKubernetesPodV1_with_container_security_context(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := busyboxImage
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigWithContainerSecurityContext(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.security_context.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.security_context.0.privileged", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.security_context.0.run_as_user", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.security_context.0.se_linux_options.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.security_context.0.se_linux_options.0.level", "s0:c123,c456"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.security_context.0.capabilities.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.security_context.0.capabilities.0.add.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.security_context.0.capabilities.0.add.0", "NET_ADMIN"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.security_context.0.capabilities.0.add.1", "SYS_TIME"),
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

func TestAccKubernetesPodV1_with_volume_mount(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	secretName := acctest.RandomWithPrefix("tf-acc-test")

	imageName := busyboxImage
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigWithVolumeMounts(secretName, podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.0.mount_path", "/tmp/my_path"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.0.name", "db"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.0.read_only", "false"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.0.sub_path", ""),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.0.sub_path_expr", ""),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.0.mount_propagation", "HostToContainer"),
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

func TestAccKubernetesPodV1_with_cfg_map_volume_mount(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	cfgMap := acctest.RandomWithPrefix("tf-acc-test")
	imageName := busyboxImage
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigWithConfigMapVolume(cfgMap, podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.0.mount_path", "/tmp/my_path"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.0.name", "cfg"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.0.read_only", "false"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.0.sub_path", ""),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.0.sub_path_expr", ""),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.0.mount_propagation", "None"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.1.mount_path", "/tmp/my_raw_path"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.1.name", "cfg-binary"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.1.read_only", "false"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.1.sub_path", ""),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.1.sub_path_expr", ""),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.name", "cfg"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.config_map.0.name", cfgMap),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.config_map.0.default_mode", "0777")),
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

func TestAccKubernetesPodV1_with_csi_volume_hostpath(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	secretName := acctest.RandomWithPrefix("tf-acc-test")
	volumeName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_pod_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if err := testAccCheckCSIDriverExists("hostpath.csi.k8s.io"); err != nil {
				t.Skip(err.Error())
			}
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1CSIVolume(imageName, podName, secretName, volumeName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.0.mount_path", "/volume"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.0.name", volumeName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.0.read_only", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.name", volumeName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.csi.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.csi.0.driver", "hostpath.csi.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.csi.0.read_only", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.csi.0.node_publish_secret_ref.0.name", secretName),
				),
			},
		},
	})
}

func TestAccKubernetesPodV1_with_projected_volume(t *testing.T) {
	var conf api.Pod

	cfgMapName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	cfgMap2Name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	secretName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	podName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImage
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ProjectedVolume(cfgMapName, cfgMap2Name, secretName, podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.name", "projected-vol"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.projected.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.projected.0.default_mode", "0777"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.projected.0.sources.#", "4"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.projected.0.sources.0.config_map.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.projected.0.sources.0.config_map.0.name", cfgMapName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.projected.0.sources.1.config_map.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.projected.0.sources.1.config_map.0.name", cfgMap2Name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.projected.0.sources.2.secret.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.projected.0.sources.2.secret.0.name", secretName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.projected.0.sources.3.downward_api.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.projected.0.sources.3.downward_api.0.items.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.projected.0.sources.3.downward_api.0.items.0.path", "labels"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.projected.0.sources.3.downward_api.0.items.0.field_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.projected.0.sources.3.downward_api.0.items.0.field_ref.0.field_path", "metadata.labels"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.projected.0.sources.3.downward_api.0.items.1.path", "cpu_limit"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.projected.0.sources.3.downward_api.0.items.1.resource_field_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.projected.0.sources.3.downward_api.0.items.1.resource_field_ref.0.container_name", "containername"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.projected.0.sources.3.downward_api.0.items.1.resource_field_ref.0.resource", "limits.cpu"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.projected.0.sources.3.downward_api.0.items.1.resource_field_ref.0.divisor", "1"),
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

func TestAccKubernetesPodV1_with_resource_requirements(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := busyboxImage
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigWithResourceRequirements(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.resources.0.requests.memory", "50Mi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.resources.0.requests.cpu", "250m"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.resources.0.requests.ephemeral-storage", "128Mi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.resources.0.limits.memory", "512Mi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.resources.0.limits.cpu", "500m"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.resources.0.limits.ephemeral-storage", "512Mi"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesPodV1ConfigWithEmptyResourceRequirements(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.resources.0.requests.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.resources.0.limits.#", "0"),
				),
			},
			{
				Config: testAccKubernetesPodV1ConfigWithResourceRequirementsLimitsOnly(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.resources.0.requests.memory", "512Mi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.resources.0.requests.cpu", "500m"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.resources.0.limits.memory", "512Mi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.resources.0.limits.cpu", "500m"),
				),
			},
			{
				Config: testAccKubernetesPodV1ConfigWithResourceRequirementsRequestsOnly(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.resources.0.requests.memory", "512Mi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.resources.0.requests.cpu", "500m"),
				),
			},
		},
	})
}

func TestAccKubernetesPodV1_with_empty_dir_volume(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := busyboxImage
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigWithEmptyDirVolumes(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.0.mount_path", "/cache"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.0.name", "cache-volume"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.empty_dir.0.medium", "Memory"),
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

func TestAccKubernetesPodV1_with_empty_dir_volume_with_sizeLimit(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := busyboxImage
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigWithEmptyDirVolumesSizeLimit(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.0.mount_path", "/cache"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.0.name", "cache-volume"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.empty_dir.0.medium", "Memory"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.empty_dir.0.size_limit", "512Mi"),
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

func TestAccKubernetesPodV1_with_secret_vol_items(t *testing.T) {
	var conf api.Pod

	secretName := acctest.RandomWithPrefix("tf-acc-test")
	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := busyboxImage
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigWithSecretItemsVolume(secretName, podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.secret.0.items.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.secret.0.items.0.key", "one"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.secret.0.items.0.path", "path/to/one"),
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

func TestAccKubernetesPodV1_gke_with_nodeSelector(t *testing.T) {
	var conf api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := busyboxImage
	region := os.Getenv("GOOGLE_REGION")
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotRunningInGke(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigNodeSelector(podName, imageName, region),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.node_selector.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.node_selector.topology.kubernetes.io/region", region),
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

func TestAccKubernetesPodV1_config_with_automount_service_account_token(t *testing.T) {
	var confPod api.Pod
	var confSA api.ServiceAccount

	podName := acctest.RandomWithPrefix("tf-acc-test")
	saName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := busyboxImage
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigWithAutomountServiceAccountToken(saName, podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceAccountV1Exists("kubernetes_service_account_v1.test", &confSA),
					testAccCheckKubernetesPodV1Exists(resourceName, &confPod),
					resource.TestCheckResourceAttr(resourceName, "spec.0.automount_service_account_token", "true"),
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

func TestAccKubernetesPodV1_config_container_working_dir(t *testing.T) {
	var confPod api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := busyboxImage
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigWorkingDir(podName, imageName, "/www"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &confPod),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.generation", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.working_dir", "/www"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesPodV1ConfigWorkingDir(podName, imageName, "/srv"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &confPod),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.generation", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.working_dir", "/srv"),
				),
			},
		},
	})
}

func TestAccKubernetesPodV1_config_container_startup_probe(t *testing.T) {
	var confPod api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := busyboxImage
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.17.0")
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ContainerStartupProbe(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &confPod),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.generation", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.startup_probe.0.http_get.0.path", "/index.html"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.startup_probe.0.http_get.0.port", "80"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.startup_probe.0.initial_delay_seconds", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.startup_probe.0.timeout_seconds", "2"),
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

func TestAccKubernetesPodV1_termination_message_policy_default(t *testing.T) {
	var confPod api.Pod

	podName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImage
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesTerminationMessagePolicyDefault(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &confPod),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.termination_message_policy", "File"),
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

func TestAccKubernetesPodV1_termination_message_policy_override_as_file(t *testing.T) {
	var confPod api.Pod

	podName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImage
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesTerminationMessagePolicyWithFile(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &confPod),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.termination_message_policy", "File"),
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

func TestAccKubernetesPodV1_termination_message_policy_override_as_fallback_to_logs_on_err(t *testing.T) {
	var confPod api.Pod

	podName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImage
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesTerminationMessagePolicyWithFallBackToLogsOnErr(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &confPod),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.termination_message_policy", "FallbackToLogsOnError"),
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

func TestAccKubernetesPodV1_enableServiceLinks(t *testing.T) {
	var conf1 api.Pod

	rName := acctest.RandomWithPrefix("tf-acc-test")
	imageName := busyboxImage
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigEnableServiceLinks(rName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.app", "pod_label"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", rName),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.enable_service_links", "false"),
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

func TestAccKubernetesPodV1_bug961EmptyBlocks(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-test")
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				ExpectError: regexp.MustCompile("Missing required argument"),
				Config:      testAccKubernetesPodV1ConfigEmptyBlocks(name, imageName),
			},
		},
	})
}

func TestAccKubernetesPodV1_bug1085(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-test")
	imageName := busyboxImage
	var conf api.Pod
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotRunningInMinikube(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigWithVolume(name, imageName, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_account_name", "default"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesPodV1ConfigWithVolume(name, imageName, `service_account_name="test"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_account_name", "test"),
				),
			},
		},
	})
}

func TestAccKubernetesPodV1_readinessGate(t *testing.T) {
	var conf1 api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	secretName := acctest.RandomWithPrefix("tf-acc-test")
	configMapName := acctest.RandomWithPrefix("tf-acc-test")
	imageName1 := busyboxImage
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigBasic(secretName, configMapName, podName, imageName1),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf1),
				),
			},
			{
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
					p, err := conn.CoreV1().Pods("default").Get(ctx, podName, metav1.GetOptions{})
					if err != nil {
						t.Fatal(err)
					}
					_, err = conn.CoreV1().Pods("default").UpdateStatus(ctx, p, metav1.UpdateOptions{})
					if err != nil {
						t.Fatal(err)
					}
				},
				Config: testAccKubernetesPodV1ConfigReadinessGate(secretName, configMapName, podName, imageName1),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "spec.0.readiness_gate.0.condition_type", "haha"),
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

func TestAccKubernetesPodV1_topologySpreadConstraint(t *testing.T) {
	var conf1 api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_pod_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.27.0")
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1TopologySpreadConstraintConfig(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "spec.0.topology_spread_constraint.#", "1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "spec.0.topology_spread_constraint.0.match_label_keys.*", "pod-template-hash"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.topology_spread_constraint.0.max_skew", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.topology_spread_constraint.0.node_affinity_policy", "Ignore"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.topology_spread_constraint.0.node_taints_policy", "Honor"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.topology_spread_constraint.0.topology_key", "topology.kubernetes.io/zone"),
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

func TestAccKubernetesPodV1_topologySpreadConstraintMinDomains(t *testing.T) {
	var conf1 api.Pod

	podName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_pod_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.27.0")
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1TopologySpreadConstraintConfigMinDomains(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "spec.0.topology_spread_constraint.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.topology_spread_constraint.0.min_domains", "1"),
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

func TestAccKubernetesPodV1_runtimeClassName(t *testing.T) {
	var conf1 api.Pod

	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_pod_v1.test"
	runtimeHandler := fmt.Sprintf("runc-%s", name)
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfRunningInEks(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigRuntimeClassName(name, imageName, runtimeHandler),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "spec.0.runtime_class_name", runtimeHandler),
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

func TestAccKubernetesPodV1_with_ephemeral_storage(t *testing.T) {
	var (
		pod api.Pod
		pvc api.PersistentVolumeClaim
	)

	podName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImage
	resourceName := "kubernetes_pod_v1.test"
	volumeName := "ephemeral"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNotRunningInKind(t)
			skipIfClusterVersionLessThan(t, "1.23.0")
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1EphemeralStorageClass(podName) +
					testAccKubernetesPodV1EphemeralStorage(podName, imageName, volumeName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &pod),
					testAccCheckKubernetesPersistentVolumeClaimCreated(fmt.Sprintf("%s-%s", podName, volumeName), &pvc),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.name", volumeName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.ephemeral.0.volume_claim_template.0.spec.0.storage_class_name", podName),
				),
			},
			// Do a second test with only the storage class and check that the PVC has been deleted by the ephemeral volume
			{
				Config: testAccKubernetesPodV1EphemeralStorageClass(podName),
				Check:  testAccCheckKubernetesPersistentVolumeClaimV1IsDestroyed(&pvc),
			},
		},
	})
}

func TestAccKubernetesPodV1_phase(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_pod_v1.test"
	image := "this-fake-image-has-never-exist"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigPhase(name, image),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"metadata.0.resource_version",
					"spec.0.node_name",
					"target_state",
				},
			},
		},
	})
}

func TestAccKubernetesPodV1_os(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_pod_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.24.0")
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigOS(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.os.0.name", "linux"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config:   testAccKubernetesPodV1ConfigOS(name, imageName),
				PlanOnly: true,
			},
		},
	})
}

func TestAccKubernetesPodV1_with_volume_mount_sub_path_expr(t *testing.T) {
	var conf api.Pod
	podName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	secretName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImage
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigWithVolumeMountsSubPathExpr(secretName, podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.0.mount_path", "/tmp/my_path"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.0.name", "db"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.0.read_only", "false"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.0.sub_path", ""),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.0.sub_path_expr", "$(POD_NAME)"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.volume_mount.0.mount_propagation", "HostToContainer"),
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

func testAccCheckCSIDriverExists(csiDriverName string) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.Background()
	_, err = conn.StorageV1().CSIDrivers().Get(ctx, csiDriverName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("could not find CSIDriver %q", csiDriverName)
	}
	return nil
}

func testAccCheckKubernetesPodV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_pod_v1" {
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

func testAccCheckKubernetesPodV1Exists(n string, obj *api.Pod) resource.TestCheckFunc {
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

func testAccCheckKubernetesPersistentVolumeClaimCreated(name string, obj *api.PersistentVolumeClaim) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
		if err != nil {
			return err
		}
		ctx := context.TODO()
		out, err := conn.CoreV1().PersistentVolumeClaims("default").Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesPodV1ConfigBasic(secretName, configMapName, podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "first"
  }
}

resource "kubernetes_secret_v1" "test_from" {
  metadata {
    name = "%s-from"
  }

  data = {
    one    = "first_from"
    second = "second_from"
  }
}

resource "kubernetes_config_map_v1" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "ONE"
  }
}

resource "kubernetes_config_map_v1" "test_from" {
  metadata {
    name = "%s-from"
  }

  data = {
    one = "ONE_FROM"
    two = "TWO_FROM"
  }
}

resource "kubernetes_pod_v1" "test" {
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
            name     = "${kubernetes_secret_v1.test.metadata.0.name}"
            key      = "one"
            optional = true
          }
        }
      }
      env {
        name = "EXPORTED_VARIABLE_FROM_CONFIG_MAP"
        value_from {
          config_map_key_ref {
            name     = "${kubernetes_config_map_v1.test.metadata.0.name}"
            key      = "one"
            optional = true
          }
        }
      }

      env_from {
        config_map_ref {
          name     = "${kubernetes_config_map_v1.test_from.metadata.0.name}"
          optional = true
        }
        prefix = "FROM_CM_"
      }
      env_from {
        secret_ref {
          name     = "${kubernetes_secret_v1.test_from.metadata.0.name}"
          optional = false
        }
        prefix = "FROM_S_"
      }
    }

    volume {
      name = "db"

      secret {
        secret_name = "${kubernetes_secret_v1.test.metadata.0.name}"
      }
    }
  }
}
`, secretName, secretName, configMapName, configMapName, podName, imageName)
}

func testAccKubernetesPodV1ConfigScheduler(podName, schedulerName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }
    name = "%s"
  }

  spec {
    automount_service_account_token = false
    scheduler_name                  = %q
    container {
      image = "%s"
      name  = "containername"
    }
  }
  timeouts {
    create = "1m"
  }
}
`, podName, schedulerName, imageName)
}

func testAccKubernetesPodV1ConfigWithInitContainer(podName, image string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
  metadata {
    name = "%s"
    labels = {
      "app.kubernetes.io/name" = "acctest"
    }
  }

  spec {
    automount_service_account_token = false
    container {
      name    = "container"
      image   = "%s"
      command = ["sh", "-c", "echo The app is running! && sleep 300"]

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
      command = ["sh", "-c", "until nslookup %s-init-service.default.svc.cluster.local; do echo waiting for init-service; sleep 2; done"]

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

resource "kubernetes_service_v1" "test" {
  metadata {
    name = "%s-init-service"
  }

  spec {
    selector = {
      "app.kubernetes.io/name" = "acctest"
    }
    port {
      port        = 8080
      target_port = 80
    }
  }
}
`, podName, image, image, podName, podName)
}

func testAccKubernetesPodV1ConfigWithSecurityContext(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
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

func testAccKubernetesPodV1ConfigWithSecurityContextRunAsGroup(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
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

func testAccKubernetesPodV1ConfigWithSecurityContextSeccompProfile(podName, imageName, seccompProfileType string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }

    name = "%s"
  }

  spec {
    automount_service_account_token = false
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
`, podName, seccompProfileType, imageName, seccompProfileType)
}

func testAccKubernetesPodV1ConfigWithSecurityContextSeccompProfileLocalhost(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }

    name = "%s"
  }

  spec {
    automount_service_account_token = false
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
  }
}
`, podName, imageName)
}

func testAccKubernetesPodV1ConfigWithLivenessProbeUsingExec(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
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
        initial_delay_seconds = 3
        period_seconds        = 1
      }
    }
    termination_grace_period_seconds = 1
  }
}
`, podName, imageName)
}

func testAccKubernetesPodV1ConfigWithLivenessProbeUsingHTTPGet(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
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
  }
}
`, podName, imageName)
}

func testAccKubernetesPodV1ConfigWithLivenessProbeUsingTCP(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
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
      args  = ["liveness"]

      liveness_probe {
        tcp_socket {
          port = 8080
        }
        initial_delay_seconds = 3
        period_seconds        = 1
      }
    }
  }
}
`, podName, imageName)
}

func testAccKubernetesPodV1ConfigWithLivenessProbeUsingGRPC(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
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
      args  = ["liveness"]

      liveness_probe {
        grpc {
          port    = 8888
          service = "EchoService"
        }
        initial_delay_seconds = 3
        period_seconds        = 1
      }
    }
  }
}
`, podName, imageName)
}

func testAccKubernetesPodV1ConfigWithLifeCycle(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
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
      args  = ["liveness"]

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

func testAccKubernetesPodV1ConfigWithContainerSecurityContext(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
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

func testAccKubernetesPodV1ConfigWithVolumeMounts(secretName, podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "first"
  }
}

resource "kubernetes_pod_v1" "test" {
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
        secret_name = "${kubernetes_secret_v1.test.metadata.0.name}"
      }
    }
  }
}
`, secretName, podName, imageName)
}

func testAccKubernetesPodV1ConfigWithSecretItemsVolume(secretName, podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "first"
  }
}

resource "kubernetes_pod_v1" "test" {
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
        secret_name = "${kubernetes_secret_v1.test.metadata.0.name}"

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

func testAccKubernetesPodV1ConfigWithConfigMapVolume(secretName, podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1" "test" {
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

resource "kubernetes_pod_v1" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }

    name = "%s"
  }

  spec {
    restart_policy                  = "Never"
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
        name         = "${kubernetes_config_map_v1.test.metadata.0.name}"
        default_mode = "0777"
      }
    }

    volume {
      name = "cfg-item"

      config_map {
        name = "${kubernetes_config_map_v1.test.metadata.0.name}"

        items {
          key  = "one"
          path = "one.txt"
        }
      }
    }

    volume {
      name = "cfg-item-with-mode"

      config_map {
        name = "${kubernetes_config_map_v1.test.metadata.0.name}"

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
        name = "${kubernetes_config_map_v1.test.metadata.0.name}"

        items {
          key  = "raw"
          path = "raw.txt"
        }
      }
    }
    termination_grace_period_seconds = 1
  }
}
`, secretName, podName, imageName)
}

func testAccKubernetesPodV1CSIVolume(imageName, podName, secretName, volumeName string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test-secret" {
  metadata {
    name = %[3]q
  }

  data = {
    secret = "test-secret"
  }
}

resource "kubernetes_pod_v1" "test" {
  metadata {
    labels = {
      "label" = "web"
    }
    name = %[1]q
  }
  spec {
    container {
      image   = %[2]q
      name    = %[1]q
      command = ["sleep", "300"]
      volume_mount {
        name       = %[4]q
        mount_path = "/volume"
        read_only  = true
      }
    }
    restart_policy = "Never"
    volume {
      name = %[4]q
      csi {
        driver    = "hostpath.csi.k8s.io"
        read_only = true
        volume_attributes = {
          "secretProviderClass" = "secret-provider"
        }
        node_publish_secret_ref {
          name = %[3]q
        }
      }
    }
    termination_grace_period_seconds = 1
  }
}`, podName, imageName, secretName, volumeName)
}

func testAccKubernetesPodV1ProjectedVolume(cfgMapName, cfgMap2Name, secretName, podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_config_map_v1" "test" {
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

resource "kubernetes_config_map_v1" "test2" {
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

resource "kubernetes_secret_v1" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "first"
  }
}

resource "kubernetes_pod_v1" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }

    name = "%s"
  }

  spec {
    restart_policy                  = "Never"
    automount_service_account_token = false

    container {
      image = "%s"
      name  = "containername"

      command = ["sleep", "300"]

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
            name = "${kubernetes_config_map_v1.test.metadata.0.name}"
            items {
              key  = "raw"
              path = "raw.txt"
            }
          }
        }
        sources {
          config_map {
            name = "${kubernetes_config_map_v1.test2.metadata.0.name}"
            items {
              key  = "raw"
              path = "raw-again.txt"
            }
          }
        }
        sources {
          secret {
            name = "${kubernetes_secret_v1.test.metadata.0.name}"
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
                resource       = "limits.cpu"
              }
            }
          }
        }
      }
    }
    termination_grace_period_seconds = 1
  }
}
`, cfgMapName, cfgMap2Name, secretName, podName, imageName)
}

func testAccKubernetesPodV1ConfigWithResourceRequirements(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
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
          cpu                 = "0.5"
          memory              = "512Mi"
          "ephemeral-storage" = "512Mi"
        }

        requests = {
          cpu                 = "250m"
          memory              = "50Mi"
          "ephemeral-storage" = "128Mi"
        }
      }
    }
  }
}
`, podName, imageName)
}

func testAccKubernetesPodV1ConfigWithEmptyResourceRequirements(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
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
        limits   = {}
        requests = {}
      }
    }
  }
}
`, podName, imageName)
}

func testAccKubernetesPodV1ConfigWithResourceRequirementsLimitsOnly(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
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
          cpu    = "500m"
          memory = "512Mi"
        }
      }
    }
  }
}
`, podName, imageName)
}

func testAccKubernetesPodV1ConfigWithResourceRequirementsRequestsOnly(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
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
        requests = {
          cpu    = "500m"
          memory = "512Mi"
        }
      }
    }
  }
}
`, podName, imageName)
}

func testAccKubernetesPodV1ConfigWithEmptyDirVolumes(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
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

func testAccKubernetesPodV1ConfigWithEmptyDirVolumesSizeLimit(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
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

func testAccKubernetesPodV1ConfigNodeSelector(podName, imageName, region string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
  metadata {
    name = "%s"
  }

  spec {
    container {
      image = "%s"
      name  = "containername"
    }

    node_selector = {
      "topology.kubernetes.io/region" = "%s"
    }
  }
}
`, podName, imageName, region)
}

func testAccKubernetesPodV1ConfigArgsUpdate(podName, imageName, args string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
  metadata {
    name = "%s"
  }

  spec {
    container {
      image = "%s"
      args  = %s
      name  = "containername"
    }
    termination_grace_period_seconds = 1
  }
}
`, podName, imageName, args)
}

func testAccKubernetesPodV1ConfigEnvUpdate(podName, imageName, val string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
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

func testAccKubernetesPodV1ConfigWithAutomountServiceAccountToken(saName string, podName string, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_service_account_v1" "test" {
  metadata {
    name = "%s"
  }
}

resource "kubernetes_pod_v1" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }

    name = "%s"
  }

  spec {
    service_account_name            = kubernetes_service_account_v1.test.metadata.0.name
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

func testAccKubernetesPodV1ConfigWorkingDir(podName, imageName, val string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
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

func testAccKubernetesPodV1ContainerStartupProbe(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
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
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
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
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
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

func testAccKubernetesPodV1ConfigEnableServiceLinks(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
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

func testAccKubernetesPodV1ConfigReadinessGate(secretName, configMapName, podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "first"
  }
}

resource "kubernetes_secret_v1" "test_from" {
  metadata {
    name = "%s-from"
  }

  data = {
    one    = "first_from"
    second = "second_from"
  }
}

resource "kubernetes_config_map_v1" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "ONE"
  }
}

resource "kubernetes_config_map_v1" "test_from" {
  metadata {
    name = "%s-from"
  }

  data = {
    one = "ONE_FROM"
    two = "TWO_FROM"
  }
}

resource "kubernetes_pod_v1" "test" {
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
            name     = "${kubernetes_secret_v1.test.metadata.0.name}"
            key      = "one"
            optional = true
          }
        }
      }
      env {
        name = "EXPORTED_VARIABLE_FROM_CONFIG_MAP"
        value_from {
          config_map_key_ref {
            name     = "${kubernetes_config_map_v1.test.metadata.0.name}"
            key      = "one"
            optional = true
          }
        }
      }

      env_from {
        config_map_ref {
          name     = "${kubernetes_config_map_v1.test_from.metadata.0.name}"
          optional = true
        }
        prefix = "FROM_CM_"
      }
      env_from {
        secret_ref {
          name     = "${kubernetes_secret_v1.test_from.metadata.0.name}"
          optional = false
        }
        prefix = "FROM_S_"
      }
    }

    volume {
      name = "db"

      secret {
        secret_name = "${kubernetes_secret_v1.test.metadata.0.name}"
      }
    }
  }
}
`, secretName, secretName, configMapName, configMapName, podName, imageName)
}

func testAccKubernetesPodV1ConfigMinimal(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
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

func testAccKubernetesPodV1ConfigEmptyBlocks(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
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

func testAccKubernetesPodV1ConfigWithVolume(name, imageName, serviceAccount string) string {
	return fmt.Sprintf(`resource "kubernetes_storage_class_v1" "test" {
  metadata {
    name = "test"
  }
  storage_provisioner = "k8s.io/minikube-hostpath"
}

resource "kubernetes_service_account_v1" "test" {
  metadata {
    name = "test"
  }
}

resource "kubernetes_persistent_volume_v1" "test" {
  metadata {
    name = "test"
  }
  spec {
    capacity = {
      storage = "1Gi"
    }
    access_modes       = ["ReadWriteOnce"]
    storage_class_name = kubernetes_storage_class_v1.test.metadata.0.name
    persistent_volume_source {
      host_path {
        path = "/mnt/minikube"
        type = "DirectoryOrCreate"
      }
    }
  }
}

resource "kubernetes_persistent_volume_claim_v1" "test" {
  wait_until_bound = false
  metadata {
    name = "test"
  }
  spec {
    access_modes       = ["ReadWriteOnce"]
    storage_class_name = kubernetes_storage_class_v1.test.metadata.0.name
    volume_name        = kubernetes_persistent_volume_v1.test.metadata.0.name
    resources {
      requests = {
        storage = "1G"
      }
    }
  }
}

resource "kubernetes_pod_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    %s
    container {
      name    = "default"
      image   = "%s"
      command = ["sleep", "300"]
      volume_mount {
        mount_path = "/etc/test"
        name       = "pvc"
      }
    }
    volume {
      name = "pvc"
      persistent_volume_claim {
        claim_name = kubernetes_persistent_volume_claim_v1.test.metadata[0].name
      }
    }
    termination_grace_period_seconds = 1
  }
}
`, name, serviceAccount, imageName)
}

func testAccKubernetesPodV1TopologySpreadConstraintConfig(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    container {
      image = "%s"
      name  = "containername"
    }
    topology_spread_constraint {
      match_label_keys     = ["pod-template-hash"]
      max_skew             = 1
      node_affinity_policy = "Ignore"
      node_taints_policy   = "Honor"
      topology_key         = "topology.kubernetes.io/zone"
      when_unsatisfiable   = "ScheduleAnyway"
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

func testAccKubernetesPodV1TopologySpreadConstraintConfigMinDomains(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    container {
      image = "%s"
      name  = "containername"
    }
    topology_spread_constraint {
      min_domains  = 1
      topology_key = "kubernetes.io/hostname"
      label_selector {
        match_labels = {
          "test" = "test"
        }
      }
    }
  }
}
`, podName, imageName)
}

func testAccKubernetesPodV1ConfigRuntimeClassName(name, imageName, runtimeHandler string) string {
	return fmt.Sprintf(`resource "kubernetes_runtime_class_v1" "test" {
  metadata {
    name = %[3]q
  }
  handler = "runc"
}

resource "kubernetes_pod_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    runtime_class_name = kubernetes_runtime_class_v1.test.metadata.0.name
    container {
      image = %[2]q
      name  = "containername"
    }
  }
}
`, name, imageName, runtimeHandler)
}

func testAccKubernetesCustomScheduler(name string) string {
	// Source: https://kubernetes.io/docs/tasks/extend-kubernetes/configure-multiple-schedulers/
	return fmt.Sprintf(`variable "namespace" {
  default = "kube-system"
}

variable "scheduler_name" {
  default = %q
}

variable "scheduler_cluster_version" {
  default = ""
}

resource "kubernetes_service_account_v1" "scheduler" {
  metadata {
    name      = var.scheduler_name
    namespace = var.namespace
  }
}

resource "kubernetes_cluster_role_binding_v1" "kube_scheduler" {
  metadata {
    name = "${var.scheduler_name}-as-kube-scheduler"
  }
  subject {
    kind      = "ServiceAccount"
    name      = var.scheduler_name
    namespace = var.namespace
  }
  role_ref {
    kind      = "ClusterRole"
    name      = "system:kube-scheduler"
    api_group = "rbac.authorization.k8s.io"
  }
}

resource "kubernetes_cluster_role_binding_v1" "volume_scheduler" {
  metadata {
    name = "${var.scheduler_name}-as-volume-scheduler"
  }
  subject {
    kind      = "ServiceAccount"
    name      = var.scheduler_name
    namespace = var.namespace
  }
  role_ref {
    kind      = "ClusterRole"
    name      = "system:volume-scheduler"
    api_group = "rbac.authorization.k8s.io"
  }
}

resource "kubernetes_role_binding_v1" "authentication_reader" {
  metadata {
    name      = "${var.scheduler_name}-extension-apiserver-authentication-reader"
    namespace = var.namespace
  }
  role_ref {
    kind      = "Role"
    name      = "extension-apiserver-authentication-reader"
    api_group = "rbac.authorization.k8s.io"
  }
  subject {
    kind      = "ServiceAccount"
    name      = var.scheduler_name
    namespace = var.namespace
  }
}

resource "kubernetes_config_map_v1" "scheduler_config" {
  metadata {
    name      = "${var.scheduler_name}-config"
    namespace = var.namespace
  }
  data = {
    "scheduler-config.yaml" = yamlencode(
      {
        "apiVersion" : "kubescheduler.config.k8s.io/v1",
        "kind" : "KubeSchedulerConfiguration",
        profiles : [{
          "schedulerName" : var.scheduler_name
        }],
        "leaderElection" : { "leaderElect" : false }
      }
    )
  }
}

resource "kubernetes_pod_v1" "scheduler" {
  metadata {
    labels = {
      component = "scheduler"
      tier      = "control-plane"
    }
    name      = var.scheduler_name
    namespace = var.namespace
  }

  spec {
    service_account_name = kubernetes_service_account_v1.scheduler.metadata.0.name
    container {
      name = var.scheduler_name
      command = [
        "/usr/local/bin/kube-scheduler",
        "--config=/etc/kubernetes/scheduler/scheduler-config.yaml"
      ]
      image = "registry.k8s.io/kube-scheduler:${var.scheduler_cluster_version}"
      liveness_probe {
        http_get {
          path   = "/healthz"
          port   = 10259
          scheme = "HTTPS"
        }
        initial_delay_seconds = 15
      }
      resources {
        requests = {
          cpu = "0.1"
        }
      }
      security_context {
        privileged = false
      }
      volume_mount {
        name       = "config-volume"
        mount_path = "/etc/kubernetes/scheduler"
      }
    }
    volume {
      name = "config-volume"
      config_map {
        name = kubernetes_config_map_v1.scheduler_config.metadata.0.name
      }
    }
  }

  timeouts {
    create = "1m"
  }
}
`, name)
}

func testAccKubernetesPodV1EphemeralStorage(podName, imageName, volumeName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    container {
      name  = "containername"
      image = %[2]q
      volume_mount {
        mount_path = "/ephemeral"
        name       = %[3]q
      }
    }
    volume {
      name = %[3]q
      ephemeral {
        volume_claim_template {
          metadata {
            labels = {
              label = %[3]q
            }
          }
          spec {
            access_modes       = ["ReadWriteOnce"]
            storage_class_name = %[1]q
            resources {
              requests = {
                storage = "1Gi"
              }
            }
          }
        }
      }
    }
    termination_grace_period_seconds = 1
  }
}
`, podName, imageName, volumeName)
}

func testAccKubernetesPodV1EphemeralStorageClass(name string) string {
	return fmt.Sprintf(`resource "kubernetes_storage_class_v1" "test" {
  metadata {
    name = %[1]q
  }
  storage_provisioner = "rancher.io/local-path"
  reclaim_policy      = "Delete"
  volume_binding_mode = "WaitForFirstConsumer"
}
`, name)
}

func testAccKubernetesPodConfigPhase(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    container {
      image = "%s"
      name  = "containername"
    }
  }
  target_state = ["Pending"]
}
`, name, imageName)
}

func testAccKubernetesPodV1ConfigOS(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    os {
      name = "linux"
    }
    container {
      image = "%s"
      name  = "containername"
    }
  }
}
`, name, imageName)
}

func testAccKubernetesPodV1ConfigWithVolumeMountsSubPathExpr(secretName, podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
  metadata {
    name = "%s"
  }
  data = {
    one = "first"
  }
}

resource "kubernetes_pod_v1" "test" {
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
      env {
        name = "POD_NAME"
        value_from {
          field_ref {
            field_path = "metadata.name"
          }
        }
      }
      volume_mount {
        mount_path        = "/tmp/my_path"
        name              = "db"
        mount_propagation = "HostToContainer"
        sub_path_expr     = "$(POD_NAME)"
      }
    }
    volume {
      name = "db"
      secret {
        secret_name = kubernetes_secret_v1.test.metadata[0].name
      }
    }
  }
}
`, secretName, podName, imageName)
}
