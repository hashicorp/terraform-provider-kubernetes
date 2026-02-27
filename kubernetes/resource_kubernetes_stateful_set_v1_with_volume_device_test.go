// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	appsv1 "k8s.io/api/apps/v1"
)

func TestAccKubernetesStatefulSetV1WIthVolumeDevice_basic(t *testing.T) {
	var conf appsv1.StatefulSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_stateful_set_v1.test"
	imageName := agnhostImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.27.0")
			skipIfNotRunningInEks(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesStatefulSetV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetV1ConfigBasicWithVolumeDevice(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "wait_for_rollout", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.namespace"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.replicas", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.min_ready_seconds", "10"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.revision_history_limit", "11"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_name", "ss-test-service"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_claim_retention_policy.0.when_deleted", "Delete"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_claim_retention_policy.0.when_scaled", "Delete"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.0.match_labels.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.0.match_labels.app", "ss-test"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.0.labels.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.0.labels.app", "ss-test"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.name", "ss-test"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.port.0.container_port", "80"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.port.0.name", "web"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.name", "ss-test"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.mount_path", "/work-dir"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_device.0.name", "ss-device-test"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_device.0.device_path", "/dev/xvda"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_strategy.0.type", "RollingUpdate"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_strategy.0.rolling_update.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_strategy.0.rolling_update.0.partition", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_claim_template.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_claim_template.0.metadata.0.name", "ss-test"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_claim_template.0.spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_claim_template.0.spec.0.access_modes.0", "ReadWriteOnce"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_claim_template.0.spec.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_claim_template.0.spec.0.resources.0.requests.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_claim_template.0.spec.0.resources.0.requests.storage", "1Gi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_claim_template.1.metadata.0.name", "ss-device-test"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_claim_template.1.spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_claim_template.1.spec.0.access_modes.0", "ReadWriteOnce"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_claim_template.1.spec.0.volume_mode", "Block"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_claim_template.1.spec.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_claim_template.1.spec.0.resources.0.requests.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_claim_template.1.spec.0.resources.0.requests.storage", "1Gi"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"wait_for_rollout",
					"spec.0.update_strategy.#",
					"spec.0.update_strategy.0.%",
					"spec.0.update_strategy.0.rolling_update.#",
					"spec.0.update_strategy.0.rolling_update.0.%",
					"spec.0.update_strategy.0.rolling_update.0.partition",
					"spec.0.update_strategy.0.type",
				},
			},
		},
	})
}

func TestAccKubernetesStatefulSetV1WIthVolumeDevice_basic_idempotency(t *testing.T) {
	var conf appsv1.StatefulSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_stateful_set_v1.test"
	imageName := agnhostImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.27.0")
			skipIfNotRunningInEks(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesStatefulSetV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetV1ConfigBasicWithVolumeDevice(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
				),
			},
			{
				Config:             testAccKubernetesStatefulSetV1ConfigBasicWithVolumeDevice(name, imageName),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
				),
			},
		},
	})
}

func TestAccKubernetesStatefulSetV1WIthVolumeDevice_Update(t *testing.T) {
	var conf appsv1.StatefulSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_stateful_set_v1.test"
	imageName := agnhostImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.27.0")
			skipIfNotRunningInEks(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesStatefulSetV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetV1ConfigMinimalWithVolumeDevice(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigUpdateImageWithVolumeDevice(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigUpdatedSelectorLabelsWithVolumeDevice(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.0.match_labels.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.0.match_labels.app", "ss-test"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.0.match_labels.layer", "ss-test-layer"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.0.labels.app", "ss-test"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.0.labels.layer", "ss-test-layer"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigUpdateReplicasWithVolumeDevice(name, imageName, "3"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.replicas", "3"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigUpdateReplicasWithVolumeDevice(name, imageName, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					// NOTE setting to empty should preserve the current replica count
					resource.TestCheckResourceAttr(resourceName, "spec.0.replicas", "3"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigUpdateReplicasWithVolumeDevice(name, imageName, "0"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.replicas", "0"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigUpdateMinReadySecondsWithVolumeDevice(name, imageName, 10),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.min_ready_seconds", "10"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigUpdateMinReadySecondsWithVolumeDevice(name, imageName, 0),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.min_ready_seconds", "0"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigRollingUpdatePartitionWithVolumeDevice(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_strategy.0.rolling_update.0.partition", "2"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigUpdateStrategyOnDeleteWithVolumeDevice(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_strategy.0.type", "OnDelete"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_strategy.0.rolling_update.#", "0"),
					resource.TestCheckNoResourceAttr(resourceName, "spec.0.update_strategy.0.rolling_update.0.partition"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigUpdateStrategyOnDeleteWithVolumeDevice(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_strategy.0.type", "OnDelete"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_strategy.0.rolling_update.#", "0"),
					resource.TestCheckNoResourceAttr(resourceName, "spec.0.update_strategy.0.rolling_update.0.partition"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigUpdateTemplateWithVolumeDevice(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.port.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.port.0.container_port", "80"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.port.0.name", "web"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.port.1.container_port", "443"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.port.1.name", "secure"),
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
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigUpdatePersistentVolumeClaimRetentionPolicyWithVolumeDevice(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_claim_retention_policy.0.when_deleted", "Retain"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_claim_retention_policy.0.when_scaled", "Retain"),
				),
			},
		},
	})
}

func TestAccKubernetesStatefulSetV1WIthVolumeDevice_waitForRollout(t *testing.T) {
	var conf1, conf2 appsv1.StatefulSet
	imageName := busyboxImage
	imageName1 := agnhostImage
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_stateful_set_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfNotRunningInEks(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesStatefulSetV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetV1ConfigWaitForRolloutWithVolumeDevice(name, imageName, "true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "wait_for_rollout", "true"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigWaitForRolloutWithVolumeDevice(name, imageName1, "false"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "wait_for_rollout", "false"),
					testAccCheckKubernetesStatefulSetForceNew(&conf1, &conf2, false),
				),
			},
		},
	})
}

func testAccKubernetesStatefulSetV1ConfigMinimalWithVolumeDevice(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    selector {
      match_labels = {
        app = "ss-test"
      }
    }
    service_name = "ss-test-service"
    template {
      metadata {
        labels = {
          app = "ss-test"
        }
      }
      spec {
        container {
          name    = "ss-test"
          image   = "%s"
          command = ["sleep", "300"]
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, name, imageName)
}

func testAccKubernetesStatefulSetV1ConfigBasicWithVolumeDevice(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set_v1" "test" {
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
    min_ready_seconds      = 10
    pod_management_policy  = "OrderedReady"
    replicas               = 1
    revision_history_limit = 11

    selector {
      match_labels = {
        app = "ss-test"
      }
    }

    service_name = "ss-test-service"

    persistent_volume_claim_retention_policy {
      when_deleted = "Delete"
      when_scaled  = "Delete"
    }

    template {
      metadata {
        labels = {
          app = "ss-test"
        }
      }

      spec {
        container {
          name  = "ss-test"
          image = %q
          args  = ["test-webserver"]

          port {
            name           = "web"
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

          volume_mount {
            name       = "ss-test"
            mount_path = "/work-dir"
          }

          volume_device {
            name        = "ss-device-test"
            device_path = "/dev/xvda"
          }
        }
      }
    }

    update_strategy {
      type = "RollingUpdate"

      rolling_update {
        partition = 1
      }
    }

    volume_claim_template {
      metadata {
        name = "ss-test"
      }

      spec {
        access_modes = ["ReadWriteOnce"]
        resources {
          requests = {
            storage = "1Gi"
          }
        }
      }
    }

    volume_claim_template {
      metadata {
        name = "ss-device-test"
      }

      spec {
        access_modes = ["ReadWriteOnce"]
        volume_mode  = "Block"
        resources {
          requests = {
            storage = "1Gi"
          }
        }
      }
    }
  }
}
`, name, imageName)
}

func testAccKubernetesStatefulSetV1ConfigUpdateImageWithVolumeDevice(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set_v1" "test" {
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
    pod_management_policy  = "OrderedReady"
    replicas               = 1
    revision_history_limit = 11

    selector {
      match_labels = {
        app = "ss-test"
      }
    }

    service_name = "ss-test-service"

    template {
      metadata {
        labels = {
          app = "ss-test"
        }
      }

      spec {
        container {
          name  = "ss-test"
          image = %q
          args  = ["pause"]

          port {
            container_port = "80"
            name           = "web"
          }

          volume_mount {
            name       = "ss-test"
            mount_path = "/work-dir"
          }

          volume_device {
            name        = "ss-device-test"
            device_path = "/dev/xvda"
          }
        }
      }
    }

    update_strategy {
      type = "RollingUpdate"

      rolling_update {
        partition = 1
      }
    }

    volume_claim_template {
      metadata {
        name = "ss-test"
      }

      spec {
        access_modes = ["ReadWriteOnce"]
        resources {
          requests = {
            storage = "1Gi"
          }
        }
      }
    }

    volume_claim_template {
      metadata {
        name = "ss-device-test"
      }

      spec {
        access_modes = ["ReadWriteOnce"]
        volume_mode  = "Block"
        resources {
          requests = {
            storage = "1Gi"
          }
        }
      }
    }
  }
}
`, name, imageName)
}

func testAccKubernetesStatefulSetV1ConfigUpdatedSelectorLabelsWithVolumeDevice(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set_v1" "test" {
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
    pod_management_policy  = "OrderedReady"
    replicas               = 1
    revision_history_limit = 11

    selector {
      match_labels = {
        app   = "ss-test"
        layer = "ss-test-layer"
      }
    }

    service_name = "ss-test-service"

    template {
      metadata {
        labels = {
          app   = "ss-test"
          layer = "ss-test-layer"
        }
      }

      spec {
        container {
          name  = "ss-test"
          image = %q
          args  = ["pause"]

          port {
            container_port = "80"
            name           = "web"
          }

          volume_mount {
            name       = "ss-test"
            mount_path = "/work-dir"
          }

          volume_device {
            name        = "ss-device-test"
            device_path = "/dev/xvda"
          }
        }
      }
    }

    update_strategy {
      type = "RollingUpdate"

      rolling_update {
        partition = 0
      }
    }

    volume_claim_template {
      metadata {
        name = "ss-test"
      }

      spec {
        access_modes = ["ReadWriteOnce"]
        resources {
          requests = {
            storage = "1Gi"
          }
        }
      }
    }

    volume_claim_template {
      metadata {
        name = "ss-device-test"
      }

      spec {
        access_modes = ["ReadWriteOnce"]
        volume_mode  = "Block"
        resources {
          requests = {
            storage = "1Gi"
          }
        }
      }
    }
  }
}
`, name, imageName)
}

func testAccKubernetesStatefulSetV1ConfigUpdateReplicasWithVolumeDevice(name, imageName, replicas string) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set_v1" "test" {
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
    pod_management_policy  = "OrderedReady"
    replicas               = %q
    revision_history_limit = 11

    selector {
      match_labels = {
        app = "ss-test"
      }
    }

    service_name = "ss-test-service"

    template {
      metadata {
        labels = {
          app = "ss-test"
        }
      }

      spec {
        container {
          name  = "ss-test"
          image = %q
          args  = ["pause"]

          port {
            container_port = "80"
            name           = "web"
          }

          volume_mount {
            name       = "ss-test"
            mount_path = "/work-dir"
          }

          volume_device {
            name        = "ss-device-test"
            device_path = "/dev/xvda"
          }
        }
        termination_grace_period_seconds = 1
      }
    }

    update_strategy {
      type = "RollingUpdate"

      rolling_update {
        partition = 1
      }
    }

    volume_claim_template {
      metadata {
        name = "ss-test"
      }

      spec {
        access_modes = ["ReadWriteOnce"]
        resources {
          requests = {
            storage = "1Gi"
          }
        }
      }
    }

    volume_claim_template {
      metadata {
        name = "ss-device-test"
      }

      spec {
        access_modes = ["ReadWriteOnce"]
        volume_mode  = "Block"
        resources {
          requests = {
            storage = "1Gi"
          }
        }
      }
    }
  }
}
`, name, replicas, imageName)
}

func testAccKubernetesStatefulSetV1ConfigUpdateMinReadySecondsWithVolumeDevice(name string, imageName string, minReadySeconds int) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set_v1" "test" {
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
    min_ready_seconds      = %d
    pod_management_policy  = "OrderedReady"
    replicas               = 1
    revision_history_limit = 11

    selector {
      match_labels = {
        app = "ss-test"
      }
    }

    service_name = "ss-test-service"

    template {
      metadata {
        labels = {
          app = "ss-test"
        }
      }

      spec {
        container {
          name  = "ss-test"
          image = %q
          args  = ["pause"]

          port {
            container_port = "80"
            name           = "web"
          }

          volume_mount {
            name       = "ss-test"
            mount_path = "/work-dir"
          }

          volume_device {
            name        = "ss-device-test"
            device_path = "/dev/xvda"
          }
        }
        termination_grace_period_seconds = 1
      }
    }

    update_strategy {
      type = "RollingUpdate"

      rolling_update {
        partition = 1
      }
    }

    volume_claim_template {
      metadata {
        name = "ss-test"
      }

      spec {
        access_modes = ["ReadWriteOnce"]
        resources {
          requests = {
            storage = "1Gi"
          }
        }
      }
    }

    volume_claim_template {
      metadata {
        name = "ss-device-test"
      }

      spec {
        access_modes = ["ReadWriteOnce"]
        volume_mode  = "Block"
        resources {
          requests = {
            storage = "1Gi"
          }
        }
      }
    }
  }
}
`, name, minReadySeconds, imageName)
}

func testAccKubernetesStatefulSetV1ConfigUpdateTemplateWithVolumeDevice(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set_v1" "test" {
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
    pod_management_policy  = "OrderedReady"
    replicas               = 1
    revision_history_limit = 11

    selector {
      match_labels = {
        app = "ss-test"
      }
    }

    service_name = "ss-test-service"

    template {
      metadata {
        labels = {
          app = "ss-test"
        }
      }

      spec {
        container {
          name  = "ss-test"
          image = %q
          args  = ["pause"]

          port {
            container_port = "80"
            name           = "web"
          }

          port {
            container_port = "443"
            name           = "secure"
          }

          volume_mount {
            name       = "ss-test"
            mount_path = "/work-dir"
          }

          volume_device {
            name        = "ss-device-test"
            device_path = "/dev/xvda"
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
      }
    }

    update_strategy {
      type = "RollingUpdate"

      rolling_update {
        partition = 1
      }
    }

    volume_claim_template {
      metadata {
        name = "ss-test"
      }

      spec {
        access_modes = ["ReadWriteOnce"]
        resources {
          requests = {
            storage = "1Gi"
          }
        }
      }
    }

    volume_claim_template {
      metadata {
        name = "ss-device-test"
      }

      spec {
        access_modes = ["ReadWriteOnce"]
        volume_mode  = "Block"
        resources {
          requests = {
            storage = "1Gi"
          }
        }
      }
    }
  }
}
`, name, imageName)
}

func testAccKubernetesStatefulSetV1ConfigRollingUpdatePartitionWithVolumeDevice(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set_v1" "test" {
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
    pod_management_policy  = "OrderedReady"
    replicas               = 1
    revision_history_limit = 11

    selector {
      match_labels = {
        app = "ss-test"
      }
    }

    service_name = "ss-test-service"

    template {
      metadata {
        labels = {
          app = "ss-test"
        }
      }

      spec {
        container {
          name  = "ss-test"
          image = %q
          args  = ["pause"]

          port {
            container_port = "80"
            name           = "web"
          }

          volume_mount {
            name       = "ss-test"
            mount_path = "/work-dir"
          }

          volume_device {
            name        = "ss-device-test"
            device_path = "/dev/xvda"
          }
        }
      }
    }

    update_strategy {
      type = "RollingUpdate"

      rolling_update {
        partition = 2
      }
    }

    volume_claim_template {
      metadata {
        name = "ss-test"
      }

      spec {
        access_modes = ["ReadWriteOnce"]
        resources {
          requests = {
            storage = "1Gi"
          }
        }
      }
    }

    volume_claim_template {
      metadata {
        name = "ss-device-test"
      }

      spec {
        access_modes = ["ReadWriteOnce"]
        volume_mode  = "Block"
        resources {
          requests = {
            storage = "1Gi"
          }
        }
      }
    }
  }
}
`, name, imageName)
}

func testAccKubernetesStatefulSetV1ConfigUpdateStrategyOnDeleteWithVolumeDevice(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set_v1" "test" {
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
    pod_management_policy  = "OrderedReady"
    replicas               = 1
    revision_history_limit = 11

    selector {
      match_labels = {
        app = "ss-test"
      }
    }

    service_name = "ss-test-service"

    template {
      metadata {
        labels = {
          app = "ss-test"
        }
      }

      spec {
        container {
          name  = "ss-test"
          image = %q
          args  = ["pause"]

          port {
            container_port = "80"
            name           = "web"
          }

          volume_mount {
            name       = "ss-test"
            mount_path = "/work-dir"
          }

          volume_device {
            name        = "ss-device-test"
            device_path = "/dev/xvda"
          }
        }
      }
    }

    update_strategy {
      type = "OnDelete"
    }

    volume_claim_template {
      metadata {
        name = "ss-test"
      }

      spec {
        access_modes = ["ReadWriteOnce"]
        resources {
          requests = {
            storage = "1Gi"
          }
        }
      }
    }

    volume_claim_template {
      metadata {
        name = "ss-device-test"
      }

      spec {
        access_modes = ["ReadWriteOnce"]
        volume_mode  = "Block"
        resources {
          requests = {
            storage = "1Gi"
          }
        }
      }
    }
  }

  wait_for_rollout = false
}
`, name, imageName)
}

func testAccKubernetesStatefulSetV1ConfigWaitForRolloutWithVolumeDevice(name, imageName, waitForRollout string) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set_v1" "test" {
  metadata {
    name = "%s"
  }

  timeouts {
    create = "10m"
    read   = "10m"
    update = "10m"
    delete = "10m"
  }

  spec {
    replicas = 2

    selector {
      match_labels = {
        app = "ss-test"
      }
    }

    update_strategy {
      type = "RollingUpdate"
    }

    service_name = "ss-test-service"

    template {
      metadata {
        labels = {
          app = "ss-test"
        }
      }

      spec {
        container {
          name    = "ss-test"
          image   = "%s"
          command = ["/bin/httpd", "-f", "-p", "80"]
          args    = ["test-webserver"]

          port {
            container_port = 80
          }

          readiness_probe {
            initial_delay_seconds = 3
            period_seconds        = 1
            tcp_socket {
              port = 80
            }
          }
        }
      }
    }
  }

  wait_for_rollout = %s
}
`, name, imageName, waitForRollout)
}

func testAccKubernetesStatefulSetV1ConfigUpdatePersistentVolumeClaimRetentionPolicyWithVolumeDevice(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set_v1" "test" {
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
    pod_management_policy  = "OrderedReady"
    replicas               = 1
    revision_history_limit = 11

    selector {
      match_labels = {
        app = "ss-test"
      }
    }

    service_name = "ss-test-service"

    persistent_volume_claim_retention_policy {
      when_deleted = "Retain"
      when_scaled  = "Retain"
    }

    template {
      metadata {
        labels = {
          app = "ss-test"
        }
      }

      spec {
        container {
          name  = "ss-test"
          image = %q
          args  = ["test-webserver"]

          port {
            name           = "web"
            container_port = 80
          }

          readiness_probe {
            initial_delay_seconds = 5
            http_get {
              path = "/"
              port = 80
            }
          }

          volume_mount {
            name       = "ss-test"
            mount_path = "/work-dir"
          }

          volume_device {
            name        = "ss-device-test"
            device_path = "/dev/xvda"
          }
        }
      }
    }

    update_strategy {
      type = "RollingUpdate"

      rolling_update {
        partition = 1
      }
    }

    volume_claim_template {
      metadata {
        name = "ss-test"
      }

      spec {
        access_modes = ["ReadWriteOnce"]
        resources {
          requests = {
            storage = "1Gi"
          }
        }
      }
    }

    volume_claim_template {
      metadata {
        name = "ss-device-test"
      }

      spec {
        access_modes = ["ReadWriteOnce"]
        volume_mode  = "Block"
        resources {
          requests = {
            storage = "1Gi"
          }
        }
      }
    }
  }
}
`, name, imageName)
}
