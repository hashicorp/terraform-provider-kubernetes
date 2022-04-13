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

func TestAccKubernetesStatefulSet_minimal(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_stateful_set.test",
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesStatefulSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetConfigMinimal(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("kubernetes_stateful_set.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_stateful_set.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_stateful_set.test", "metadata.0.uid"),
					resource.TestCheckResourceAttrSet("kubernetes_stateful_set.test", "metadata.0.namespace"),
				),
			},
		},
	})
}

func TestAccKubernetesStatefulSet_basic(t *testing.T) {
	var conf api.StatefulSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_stateful_set.test",
		IDRefreshIgnore: []string{
			"metadata.0.resource_version",
			"spec.0.template.0.spec.0.container.0.resources.0.limits",
			"spec.0.template.0.spec.0.container.0.resources.0.requests",
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesStatefulSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetConfigBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists("kubernetes_stateful_set.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "wait_for_rollout", "true"),
					resource.TestCheckResourceAttrSet("kubernetes_stateful_set.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_stateful_set.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_stateful_set.test", "metadata.0.uid"),
					resource.TestCheckResourceAttrSet("kubernetes_stateful_set.test", "metadata.0.namespace"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.replicas", "1"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.revision_history_limit", "11"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.service_name", "ss-test-service"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.selector.0.match_labels.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.selector.0.match_labels.app", "ss-test"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.metadata.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.metadata.0.labels.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.metadata.0.labels.app", "ss-test"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.spec.0.container.0.name", "ss-test"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.spec.0.container.0.image", "nginx:1.19"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.spec.0.container.0.port.0.container_port", "80"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.spec.0.container.0.port.0.name", "web"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.spec.0.container.0.volume_mount.0.name", "ss-test"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.spec.0.container.0.volume_mount.0.mount_path", "/work-dir"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.update_strategy.0.type", "RollingUpdate"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.update_strategy.0.rolling_update.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.update_strategy.0.rolling_update.0.partition", "1"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.volume_claim_template.0.metadata.0.name", "ss-test"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.volume_claim_template.0.spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.volume_claim_template.0.spec.0.access_modes.0", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.volume_claim_template.0.spec.0.resources.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.volume_claim_template.0.spec.0.resources.0.requests.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.volume_claim_template.0.spec.0.resources.0.requests.storage", "1Gi"),
				),
			},
			{
				ResourceName:      "kubernetes_stateful_set.test",
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

func TestAccKubernetesStatefulSet_basic_idempotency(t *testing.T) {
	var conf api.StatefulSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_stateful_set.test",
		IDRefreshIgnore: []string{
			"metadata.0.resource_version",
			"spec.0.template.0.spec.0.container.0.resources.0.limits",
			"spec.0.template.0.spec.0.container.0.resources.0.requests",
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesStatefulSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetConfigBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists("kubernetes_stateful_set.test", &conf),
				),
			},
			{
				Config:             testAccKubernetesStatefulSetConfigBasic(name),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists("kubernetes_stateful_set.test", &conf),
				),
			},
		},
	})
}

func TestAccKubernetesStatefulSet_Update(t *testing.T) {
	var conf api.StatefulSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_stateful_set.test",
		IDRefreshIgnore: []string{
			"metadata.0.resource_version",
			"spec.0.template.0.spec.0.container.0.resources.0.limits",
			"spec.0.template.0.spec.0.container.0.resources.0.requests",
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesStatefulSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetConfigBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists("kubernetes_stateful_set.test", &conf),
				),
			},
			{
				Config: testAccKubernetesStatefulSetConfigUpdateImage(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists("kubernetes_stateful_set.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.spec.0.container.0.image", "k8s.gcr.io/liveness:latest"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetConfigUpdatedSelectorLabels(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists("kubernetes_stateful_set.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.selector.0.match_labels.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.selector.0.match_labels.app", "ss-test"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.selector.0.match_labels.layer", "ss-test-layer"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.metadata.0.labels.app", "ss-test"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.metadata.0.labels.layer", "ss-test-layer"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetConfigUpdateReplicas(name, "5"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists("kubernetes_stateful_set.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.replicas", "5"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetConfigUpdateReplicas(name, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists("kubernetes_stateful_set.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "metadata.0.name", name),
					// NOTE setting to empty should preserve the current replica count
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.replicas", "5"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetConfigUpdateReplicas(name, "0"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists("kubernetes_stateful_set.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.replicas", "0"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetConfigRollingUpdatePartition(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists("kubernetes_stateful_set.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.update_strategy.0.rolling_update.0.partition", "2"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetConfigUpdateStrategyOnDelete(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists("kubernetes_stateful_set.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.update_strategy.0.type", "OnDelete"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.update_strategy.0.rolling_update.#", "0"),
					resource.TestCheckNoResourceAttr("kubernetes_stateful_set.test", "spec.0.update_strategy.0.rolling_update"),
					resource.TestCheckNoResourceAttr("kubernetes_stateful_set.test", "spec.0.update_strategy.0.rolling_update.0.partition"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetConfigUpdateStrategyOnDelete(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists("kubernetes_stateful_set.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.update_strategy.0.type", "OnDelete"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.update_strategy.0.rolling_update.#", "0"),
					resource.TestCheckNoResourceAttr("kubernetes_stateful_set.test", "spec.0.update_strategy.0.rolling_update"),
					resource.TestCheckNoResourceAttr("kubernetes_stateful_set.test", "spec.0.update_strategy.0.rolling_update.0.partition"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetConfigUpdateTemplate(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists("kubernetes_stateful_set.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.spec.0.container.0.port.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.spec.0.container.0.port.0.container_port", "80"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.spec.0.container.0.port.0.name", "web"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.spec.0.container.0.port.1.container_port", "443"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.spec.0.container.0.port.1.name", "secure"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.spec.0.dns_config.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.spec.0.dns_config.0.nameservers.#", "3"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.spec.0.dns_config.0.nameservers.0", "1.1.1.1"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.spec.0.dns_config.0.nameservers.1", "8.8.8.8"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.spec.0.dns_config.0.nameservers.2", "9.9.9.9"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.spec.0.dns_config.0.searches.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.spec.0.dns_config.0.searches.0", "kubernetes.io"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.spec.0.dns_config.0.option.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.spec.0.dns_config.0.option.0.name", "ndots"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.spec.0.dns_config.0.option.0.value", "1"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.spec.0.dns_config.0.option.1.name", "use-vc"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.spec.0.dns_config.0.option.1.value", ""),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.spec.0.dns_policy", "Default"),
				),
			},
		},
	})
}

func TestAccKubernetesStatefulSet_waitForRollout(t *testing.T) {
	var conf1, conf2 api.StatefulSet
	imageName := nginxImageVersion
	imageName1 := nginxImageVersion1
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_stateful_set.test",
		IDRefreshIgnore: []string{
			"spec.0.template.0.spec.0.container.0.resources.0.limits",
			"spec.0.template.0.spec.0.container.0.resources.0.requests",
			"metadata.0.resource_version",
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesStatefulSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetConfigWaitForRollout(name, imageName, "true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists("kubernetes_stateful_set.test", &conf1),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "wait_for_rollout", "true"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetConfigWaitForRollout(name, imageName1, "false"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists("kubernetes_stateful_set.test", &conf2),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "wait_for_rollout", "false"),
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
		if rs.Type != "kubernetes_stateful_set" {
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

func testAccCheckKubernetesStatefulSetExists(n string, obj *appsv1.StatefulSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		d, err := getStatefulSetFromResourceName(s, n)
		if err != nil {
			return err
		}
		*obj = *d
		return nil
	}
}

func testAccKubernetesStatefulSetConfigMinimal(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set" "test" {
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

func testAccKubernetesStatefulSetConfigBasic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set" "test" {
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
          name = "ss-test"
          image = "nginx:1.19"

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

func testAccKubernetesStatefulSetConfigUpdateImage(name string) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set" "test" {
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
          image = "k8s.gcr.io/liveness:latest"

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

func testAccKubernetesStatefulSetConfigUpdatedSelectorLabels(name string) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set" "test" {
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
          image = "k8s.gcr.io/pause:latest"

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

func testAccKubernetesStatefulSetConfigUpdateReplicas(name string, replicas string) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set" "test" {
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
          image = "k8s.gcr.io/pause:latest"

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

func testAccKubernetesStatefulSetConfigUpdateTemplate(name string) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set" "test" {
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
          image = "k8s.gcr.io/pause:latest"

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

func testAccKubernetesStatefulSetConfigRollingUpdatePartition(name string) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set" "test" {
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
          image = "k8s.gcr.io/pause:latest"

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

func testAccKubernetesStatefulSetConfigUpdateStrategyOnDelete(name string) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set" "test" {
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
          image = "k8s.gcr.io/pause:latest"

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

func testAccKubernetesStatefulSetConfigWaitForRollout(name, imageName, waitForRollout string) string {
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

resource "kubernetes_stateful_set" "test" {
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
