// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	api "k8s.io/api/apps/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesStatefulSetV1_minimal(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_stateful_set_v1.test"
	imageName := busyboxImageVersion
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesStatefulSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetV1ConfigMinimal(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.namespace"),
				),
			},
		},
	})
}

func TestAccKubernetesStatefulSetV1_basic(t *testing.T) {
	var conf api.StatefulSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_stateful_set_v1.test"
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: resourceName,
		IDRefreshIgnore: []string{
			"metadata.0.resource_version",
			"spec.0.template.0.spec.0.container.0.resources.0.limits",
			"spec.0.template.0.spec.0.container.0.resources.0.requests",
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesStatefulSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetV1ConfigBasic(name),
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
					resource.TestCheckResourceAttr(resourceName, "spec.0.revision_history_limit", "11"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_name", "ss-test-service"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.0.match_labels.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.0.match_labels.app", "ss-test"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.0.labels.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.0.labels.app", "ss-test"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.name", "ss-test"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", "busybox:1.32"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.port.0.container_port", "80"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.port.0.name", "web"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.name", "ss-test"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.mount_path", "/work-dir"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_strategy.0.type", "RollingUpdate"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_strategy.0.rolling_update.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_strategy.0.rolling_update.0.partition", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_claim_template.0.metadata.0.name", "ss-test"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_claim_template.0.spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_claim_template.0.spec.0.access_modes.0", "ReadWriteOnce"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_claim_template.0.spec.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_claim_template.0.spec.0.resources.0.requests.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_claim_template.0.spec.0.resources.0.requests.storage", "1Gi"),
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

func TestAccKubernetesStatefulSetV1_basic_idempotency(t *testing.T) {
	var conf api.StatefulSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_stateful_set_v1.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: resourceName,
		IDRefreshIgnore: []string{
			"metadata.0.resource_version",
			"spec.0.template.0.spec.0.container.0.resources.0.limits",
			"spec.0.template.0.spec.0.container.0.resources.0.requests",
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesStatefulSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetV1ConfigBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
				),
			},
			{
				Config:             testAccKubernetesStatefulSetV1ConfigBasic(name),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
				),
			},
		},
	})
}

func TestAccKubernetesStatefulSetV1_Update(t *testing.T) {
	var conf api.StatefulSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_stateful_set_v1.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: resourceName,
		IDRefreshIgnore: []string{
			"metadata.0.resource_version",
			"spec.0.template.0.spec.0.container.0.resources.0.limits",
			"spec.0.template.0.spec.0.container.0.resources.0.requests",
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesStatefulSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetV1ConfigBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigUpdateImage(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", "registry.k8s.io/e2e-test-images/agnhost:2.40"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigUpdatedSelectorLabels(name),
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
				Config: testAccKubernetesStatefulSetV1ConfigUpdateReplicas(name, "5"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.replicas", "5"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigUpdateReplicas(name, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					// NOTE setting to empty should preserve the current replica count
					resource.TestCheckResourceAttr(resourceName, "spec.0.replicas", "5"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigUpdateReplicas(name, "0"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.replicas", "0"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigRollingUpdatePartition(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_strategy.0.rolling_update.0.partition", "2"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigUpdateStrategyOnDelete(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_strategy.0.type", "OnDelete"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_strategy.0.rolling_update.#", "0"),
					resource.TestCheckNoResourceAttr(resourceName, "spec.0.update_strategy.0.rolling_update.0.partition"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigUpdateStrategyOnDelete(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_strategy.0.type", "OnDelete"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_strategy.0.rolling_update.#", "0"),
					resource.TestCheckNoResourceAttr(resourceName, "spec.0.update_strategy.0.rolling_update.0.partition"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigUpdateTemplate(name),
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
		},
	})
}

func TestAccKubernetesStatefulSetV1_waitForRollout(t *testing.T) {
	var conf1, conf2 api.StatefulSet
	imageName := busyboxImageVersion
	imageName1 := busyboxImageVersion1
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_stateful_set_v1.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: resourceName,
		IDRefreshIgnore: []string{
			"spec.0.template.0.spec.0.container.0.resources.0.limits",
			"spec.0.template.0.spec.0.container.0.resources.0.requests",
			"metadata.0.resource_version",
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesStatefulSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetV1ConfigWaitForRollout(name, imageName, "true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "wait_for_rollout", "true"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigWaitForRollout(name, imageName1, "false"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "wait_for_rollout", "false"),
					testAccCheckKubernetesStatefulSetForceNew(&conf1, &conf2, false),
				),
			},
		},
	})
}

func testAccCheckKubernetesStatefulSetForceNew(old, new *api.StatefulSet, wantNew bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if wantNew {
			if old.ObjectMeta.UID == new.ObjectMeta.UID {
				return fmt.Errorf("Expecting new resource for StatefulSet %s", old.ObjectMeta.UID)
			}
		} else {
			if old.ObjectMeta.UID != new.ObjectMeta.UID {
				return fmt.Errorf("Expecting StatefulSet UIDs to be the same: expected %s got %s", old.ObjectMeta.UID, new.ObjectMeta.UID)
			}
		}
		return nil
	}
}

func testAccCheckKubernetesStatefulSetDestroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_stateful_set_v1" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Namespace == namespace && resp.Name == name {
				return fmt.Errorf("StatefulSet still exists: %s: (Generation %#v)", rs.Primary.ID, resp.Status.ObservedGeneration)
			}
		}
	}

	return nil
}

func getStatefulSetFromResourceName(s *terraform.State, n string) (*appsv1.StatefulSet, error) {
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

	out, err := conn.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return out, nil
}

func testAccCheckKubernetesStatefulSetV1Exists(n string, obj *appsv1.StatefulSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		d, err := getStatefulSetFromResourceName(s, n)
		if err != nil {
			return err
		}
		*obj = *d
		return nil
	}
}

func testAccKubernetesStatefulSetV1ConfigMinimal(name, imageName string) string {
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
          name  = "ss-test"
          image = "%s"
        }
      }
    }
  }
}
`, name, imageName)
}

func testAccKubernetesStatefulSetV1ConfigBasic(name string) string {
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
          image = "busybox:1.32"

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
  }
}
`, name)
}

func testAccKubernetesStatefulSetV1ConfigUpdateImage(name string) string {
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
          image = "registry.k8s.io/e2e-test-images/agnhost:2.40"
          args  = ["pause"]

          port {
            container_port = "80"
            name           = "web"
          }

          volume_mount {
            name       = "ss-test"
            mount_path = "/work-dir"
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
  }
}
`, name)
}

func testAccKubernetesStatefulSetV1ConfigUpdatedSelectorLabels(name string) string {
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
          image = "registry.k8s.io/e2e-test-images/agnhost:2.40"
          args  = ["pause"]

          port {
            container_port = "80"
            name           = "web"
          }

          volume_mount {
            name       = "ss-test"
            mount_path = "/work-dir"
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
  }
}
`, name)
}

func testAccKubernetesStatefulSetV1ConfigUpdateReplicas(name string, replicas string) string {
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
          image = "registry.k8s.io/e2e-test-images/agnhost:2.40"
          args  = ["pause"]

          port {
            container_port = "80"
            name           = "web"
          }

          volume_mount {
            name       = "ss-test"
            mount_path = "/work-dir"
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
  }
}
`, name, replicas)
}

func testAccKubernetesStatefulSetV1ConfigUpdateTemplate(name string) string {
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
          image = "registry.k8s.io/e2e-test-images/agnhost:2.40"
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
  }
}
`, name)
}

func testAccKubernetesStatefulSetV1ConfigRollingUpdatePartition(name string) string {
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
          image = "registry.k8s.io/e2e-test-images/agnhost:2.40"
          args  = ["pause"]

          port {
            container_port = "80"
            name           = "web"
          }

          volume_mount {
            name       = "ss-test"
            mount_path = "/work-dir"
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
  }
}
`, name)
}

func testAccKubernetesStatefulSetV1ConfigUpdateStrategyOnDelete(name string) string {
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
          image = "registry.k8s.io/e2e-test-images/agnhost:2.40"
          args  = ["pause"]

          port {
            container_port = "80"
            name           = "web"
          }

          volume_mount {
            name       = "ss-test"
            mount_path = "/work-dir"
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
  }

  wait_for_rollout = false
}
`, name)
}

func testAccKubernetesStatefulSetV1ConfigWaitForRollout(name, imageName, waitForRollout string) string {
	return fmt.Sprintf(`resource "kubernetes_service" "test" {
  metadata {
    name = "ss-test"
  }
  spec {
    port {
      port = 80
    }
  }
}

resource "kubernetes_stateful_set_v1" "test" {
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

    service_name = kubernetes_service.test.metadata.0.name

    template {
      metadata {
        labels = {
          app = "ss-test"
        }
      }

      spec {
        container {
          name  = "ss-test"
          image = "%s"

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

  wait_for_rollout = %s
}
`, name, imageName, waitForRollout)
}
