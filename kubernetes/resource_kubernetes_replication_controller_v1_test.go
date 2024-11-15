// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesReplicationControllerV1_minimal(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImage
	resourceName := "kubernetes_replication_controller_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesReplicationControllerV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerV1ConfigMinimal(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
				),
			},
			{
				Config:   testAccKubernetesReplicationControllerV1ConfigMinimal(name, imageName),
				PlanOnly: true,
			},
		},
	})
}

func TestAccKubernetesReplicationControllerV1_basic(t *testing.T) {
	var conf api.ReplicationController
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImage
	replicas := getReplicaCount()
	resourceName := "kubernetes_replication_controller_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesReplicationControllerV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerV1Config_basic(name, imageName, replicas),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerV1Exists(resourceName, &conf),
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
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.0.annotations.TestAnnotationFive", "five"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesReplicationControllerV1Config_modified(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerV1Exists(resourceName, &conf),
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
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.name", "tf-acc-test"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.0.annotations.TestAnnotationSix", "six"),
				),
			},
		},
	})
}

func TestAccKubernetesReplicationControllerV1_initContainer(t *testing.T) {
	var conf1, conf2 api.ReplicationController
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	replicas := getReplicaCount()
	resourceName := "kubernetes_replication_controller_v1.test"
	imageName := busyboxImage
	imageName1 := agnhostImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesReplicationControllerV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerV1Config_initContainer(name, imageName, replicas),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerV1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.init_container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.init_container.0.name", "install"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.init_container.0.command.0", "wget"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.init_container.0.command.1", "-O"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.init_container.0.command.2", "/work-dir/index.html"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.init_container.0.command.3", "http://kubernetes.io"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.init_container.0.volume_mount.0.name", "workdir"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.init_container.0.volume_mount.0.mount_path", "/work-dir"),
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
				),
			},
			{
				Config: testAccKubernetesReplicationControllerV1Config_initContainer(name, imageName1, replicas),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.init_container.0.image", imageName1),
					testAccCheckKubernetesReplicationControllerV1ForceNew(&conf1, &conf2, true),
				),
			},
		},
	})
}

func TestAccKubernetesReplicationControllerV1_generatedName(t *testing.T) {
	var conf api.ReplicationController
	prefix := "tf-acc-test-gen-"
	imageName := busyboxImage
	resourceName := "kubernetes_replication_controller_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesReplicationControllerV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerV1Config_generatedName(prefix, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerV1Exists(resourceName, &conf),
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
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
		},
	})
}

func TestAccKubernetesReplicationControllerV1_with_security_context(t *testing.T) {
	var conf api.ReplicationController

	rcName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImage
	resourceName := "kubernetes_replication_controller_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesReplicationControllerV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerV1ConfigWithSecurityContext(rcName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.fs_group", "100"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.run_as_non_root", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.run_as_user", "101"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.supplemental_groups.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.security_context.0.supplemental_groups.0", "101"),
				),
			},
		},
	})
}

func TestAccKubernetesReplicationControllerV1_with_security_context_run_as_group(t *testing.T) {
	var conf api.ReplicationController

	rcName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImage
	resourceName := "kubernetes_replication_controller_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfUnsupportedSecurityContextRunAsGroup(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesReplicationControllerV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerV1ConfigWithSecurityContextRunAsGroup(rcName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerV1Exists(resourceName, &conf),
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

func TestAccKubernetesReplicationControllerV1_with_container_liveness_probe_using_exec(t *testing.T) {
	var conf api.ReplicationController

	rcName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImage
	resourceName := "kubernetes_replication_controller_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesReplicationControllerV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerV1ConfigWithLivenessProbeUsingExec(rcName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerV1Exists(resourceName, &conf),
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

func TestAccKubernetesReplicationControllerV1_with_container_liveness_probe_using_http_get(t *testing.T) {
	var conf api.ReplicationController

	rcName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := agnhostImage
	resourceName := "kubernetes_replication_controller_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesReplicationControllerV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerV1ConfigWithLivenessProbeUsingHTTPGet(rcName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerV1Exists(resourceName, &conf),
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

func TestAccKubernetesReplicationControllerV1_with_container_liveness_probe_using_tcp(t *testing.T) {
	var conf api.ReplicationController

	rcName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := agnhostImage
	resourceName := "kubernetes_replication_controller_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesReplicationControllerV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerV1ConfigWithLivenessProbeUsingTCP(rcName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.args.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.tcp_socket.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.liveness_probe.0.tcp_socket.0.port", "8080"),
				),
			},
		},
	})
}

func TestAccKubernetesReplicationControllerV1_with_container_lifecycle(t *testing.T) {
	var conf api.ReplicationController

	rcName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := agnhostImage
	resourceName := "kubernetes_replication_controller_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesReplicationControllerV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerV1ConfigWithLifeCycle(rcName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerV1Exists(resourceName, &conf),
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

func TestAccKubernetesReplicationControllerV1_with_container_security_context(t *testing.T) {
	var conf api.ReplicationController

	rcName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImage
	resourceName := "kubernetes_replication_controller_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesReplicationControllerV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerV1ConfigWithContainerSecurityContext(rcName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.security_context.#", "1"),
				),
			},
		},
	})
}

func TestAccKubernetesReplicationControllerV1_with_volume_mount(t *testing.T) {
	var conf api.ReplicationController

	rcName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	secretName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	imageName := busyboxImage
	resourceName := "kubernetes_replication_controller_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesReplicationControllerV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerV1ConfigWithVolumeMounts(secretName, rcName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.mount_path", "/tmp/my_path"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.name", "db"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.read_only", "false"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.sub_path", ""),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.sub_path_expr", ""),
				),
			},
		},
	})
}

func TestAccKubernetesReplicationControllerV1_with_resource_requirements(t *testing.T) {
	var conf api.ReplicationController

	rcName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	imageName := busyboxImage
	resourceName := "kubernetes_replication_controller_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesReplicationControllerV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerV1ConfigWithResourceRequirements(rcName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.resources.0.requests.memory", "50Mi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.resources.0.requests.cpu", "250m"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.resources.0.limits.memory", "512Mi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.resources.0.limits.cpu", "500m"),
				),
			},
		},
	})
}

func TestAccKubernetesReplicationControllerV1_with_empty_dir_volume(t *testing.T) {
	var conf api.ReplicationController

	rcName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImage
	resourceName := "kubernetes_replication_controller_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesReplicationControllerV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesReplicationControllerV1ConfigWithEmptyDirVolumes(rcName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesReplicationControllerV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.mount_path", "/cache"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.name", "cache-volume"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.volume.0.empty_dir.0.medium", "Memory"),
				),
			},
		},
	})
}

func testAccCheckKubernetesReplicationControllerV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_replication_controller_v1" {
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

func testAccCheckKubernetesReplicationControllerV1Exists(n string, obj *api.ReplicationController) resource.TestCheckFunc {
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

func testAccCheckKubernetesReplicationControllerV1ForceNew(old, new *api.ReplicationController, wantNew bool) resource.TestCheckFunc {
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

func testAccKubernetesReplicationControllerV1Config_basic(name, imageName string, replicas int) string {
	return fmt.Sprintf(`resource "kubernetes_replication_controller_v1" "test" {
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
    replicas = "%d" # This is intentionally high to exercise the waiter

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
`, name, replicas, imageName)
}

func testAccKubernetesReplicationControllerV1Config_initContainer(name, image string, replicas int) string {
	return fmt.Sprintf(`resource "kubernetes_replication_controller_v1" "test" {
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
    replicas = "%d" # This is intentionally high to exercise the waiter
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
`, name, replicas, image, image)
}

func testAccKubernetesReplicationControllerV1Config_modified(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_replication_controller_v1" "test" {
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

func testAccKubernetesReplicationControllerV1Config_generatedName(prefix, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_replication_controller_v1" "test" {
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

func testAccKubernetesReplicationControllerV1ConfigWithSecurityContext(rcName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_replication_controller_v1" "test" {
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

func testAccKubernetesReplicationControllerV1ConfigWithSecurityContextRunAsGroup(rcName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_replication_controller_v1" "test" {
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

func testAccKubernetesReplicationControllerV1ConfigWithLivenessProbeUsingExec(rcName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_replication_controller_v1" "test" {
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
            initial_delay_seconds = 3
            period_seconds        = 1
          }
        }
      }
    }
  }
}
`, rcName, imageName)
}

func testAccKubernetesReplicationControllerV1ConfigWithLivenessProbeUsingHTTPGet(rcName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_replication_controller_v1" "test" {
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
  }
}
`, rcName, imageName)
}

func testAccKubernetesReplicationControllerV1ConfigWithLivenessProbeUsingTCP(rcName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_replication_controller_v1" "test" {
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
  }
}
`, rcName, imageName)
}

func testAccKubernetesReplicationControllerV1ConfigWithLifeCycle(rcName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_replication_controller_v1" "test" {
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
  }
}
`, rcName, imageName)
}

func testAccKubernetesReplicationControllerV1ConfigWithContainerSecurityContext(rcName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_replication_controller_v1" "test" {
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

func testAccKubernetesReplicationControllerV1ConfigWithVolumeMounts(secretName, rcName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "first"
  }
}

resource "kubernetes_replication_controller_v1" "test" {
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
            secret_name = "${kubernetes_secret_v1.test.metadata.0.name}"
          }
        }
      }
    }
  }
}
`, secretName, rcName, imageName)
}

func testAccKubernetesReplicationControllerV1ConfigWithResourceRequirements(rcName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_replication_controller_v1" "test" {
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

func testAccKubernetesReplicationControllerV1ConfigWithEmptyDirVolumes(rcName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_replication_controller_v1" "test" {
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

func testAccKubernetesReplicationControllerV1ConfigMinimal(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_replication_controller_v1" "test" {
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

func getReplicaCount() int {
	envReplicas, ok := os.LookupEnv("TF_ACC_KUBERNETES_RC_REPLICAS")
	if ok {
		rv, err := strconv.Atoi(envReplicas)
		if err == nil {
			return rv
		}
		fmt.Printf("Failed to parse value of TF_ACC_KUBERNETES_RC_REPLICAS env var: %s", err)
	}
	return 50 // historical default value
}
