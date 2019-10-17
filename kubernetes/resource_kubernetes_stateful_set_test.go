package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	api "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const statefulSetTestResourceName = "kubernetes_stateful_set.test"

func TestAccKubernetesStatefulSet_basic(t *testing.T) {
	var conf api.StatefulSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: statefulSetTestResourceName,
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesStatefulSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetConfigBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists(statefulSetTestResourceName, &conf),
					testAccKubernetesStatefulSetChecksBasic(name),
				),
			},
		},
	})
}
func TestAccKubernetesStatefulSet_basic_idempotency(t *testing.T) {
	var conf api.StatefulSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: statefulSetTestResourceName,
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesStatefulSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetConfigBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists(statefulSetTestResourceName, &conf),
					testAccKubernetesStatefulSetChecksBasic(name),
				),
			},
			{
				Config:             testAccKubernetesStatefulSetConfigBasic(name),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists(statefulSetTestResourceName, &conf),
					testAccKubernetesStatefulSetChecksBasic(name),
				),
			},
		},
	})
}

func TestAccKubernetesStatefulSet_update_image(t *testing.T) {
	var conf api.StatefulSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: statefulSetTestResourceName,
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesStatefulSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetConfigBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists(statefulSetTestResourceName, &conf),
					testAccKubernetesStatefulSetChecksBasic(name),
				),
			},
			{
				Config: testAccKubernetesStatefulSetConfigUpdateImage(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists(statefulSetTestResourceName, &conf),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.template.0.spec.0.container.0.image", "k8s.gcr.io/liveness:latest"),
				),
			},
		},
	})
}

func TestAccKubernetesStatefulSet_update_template_selector_labels(t *testing.T) {
	var conf api.StatefulSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: statefulSetTestResourceName,
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesStatefulSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetConfigBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists(statefulSetTestResourceName, &conf),
					testAccKubernetesStatefulSetChecksBasic(name),
				),
			},
			{
				Config: testAccKubernetesStatefulSetConfigUpdatedSelectorLabels(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists(statefulSetTestResourceName, &conf),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.selector.0.match_labels.%", "2"),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.selector.0.match_labels.app", "ss-test"),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.selector.0.match_labels.layer", "ss-test-layer"),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.template.0.metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.template.0.metadata.0.labels.app", "ss-test"),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.template.0.metadata.0.labels.layer", "ss-test-layer"),
				),
			},
		},
	})
}

func TestAccKubernetesStatefulSet_update_replicas_to_five(t *testing.T) {
	var conf api.StatefulSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: statefulSetTestResourceName,
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesStatefulSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetConfigBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists(statefulSetTestResourceName, &conf),
					testAccKubernetesStatefulSetChecksBasic(name),
				),
			},
			{
				Config: testAccKubernetesStatefulSetConfigUpdateReplicas(name, 5),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists(statefulSetTestResourceName, &conf),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.replicas", "5"),
				),
			},
		},
	})
}

func TestAccKubernetesStatefulSet_update_replicas_to_zero(t *testing.T) {
	var conf api.StatefulSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: statefulSetTestResourceName,
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesStatefulSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetConfigBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists(statefulSetTestResourceName, &conf),
					testAccKubernetesStatefulSetChecksBasic(name),
				),
			},
			{
				Config: testAccKubernetesStatefulSetConfigUpdateReplicas(name, 0),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists(statefulSetTestResourceName, &conf),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.replicas", "0"),
				),
			},
		},
	})
}

func TestAccKubernetesStatefulSet_update_rolling_update_partition(t *testing.T) {
	var conf api.StatefulSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: statefulSetTestResourceName,
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesStatefulSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetConfigBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists(statefulSetTestResourceName, &conf),
					testAccKubernetesStatefulSetChecksBasic(name),
				),
			},
			{
				Config: testAccKubernetesStatefulSetConfigRollingUpdatePartition(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists(statefulSetTestResourceName, &conf),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.update_strategy.0.rolling_update.0.partition", "2"),
				),
			},
		},
	})
}

func TestAccKubernetesStatefulSet_update_update_strategy_on_delete(t *testing.T) {
	var conf api.StatefulSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: statefulSetTestResourceName,
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesStatefulSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetConfigBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists(statefulSetTestResourceName, &conf),
					testAccKubernetesStatefulSetChecksBasic(name),
				),
			},
			{
				Config: testAccKubernetesStatefulSetConfigUpdateStrategyOnDelete(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists(statefulSetTestResourceName, &conf),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.update_strategy.0.type", "OnDelete"),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.update_strategy.0.rolling_update.#", "0"),
					resource.TestCheckNoResourceAttr(statefulSetTestResourceName, "spec.0.update_strategy.0.rolling_update"),
					resource.TestCheckNoResourceAttr(statefulSetTestResourceName, "spec.0.update_strategy.0.rolling_update.0.partition"),
				),
			},
		},
	})
}
func TestAccKubernetesStatefulSet_update_update_strategy_rolling_update(t *testing.T) {
	var conf api.StatefulSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: statefulSetTestResourceName,
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesStatefulSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetConfigUpdateStrategyOnDelete(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists(statefulSetTestResourceName, &conf),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.update_strategy.0.type", "OnDelete"),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.update_strategy.0.rolling_update.#", "0"),
					resource.TestCheckNoResourceAttr(statefulSetTestResourceName, "spec.0.update_strategy.0.rolling_update"),
					resource.TestCheckNoResourceAttr(statefulSetTestResourceName, "spec.0.update_strategy.0.rolling_update.0.partition"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetConfigBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists(statefulSetTestResourceName, &conf),
					testAccKubernetesStatefulSetChecksBasic(name),
				),
			},
		},
	})
}

func TestAccKubernetesStatefulSet_update_pod_template(t *testing.T) {
	var conf api.StatefulSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: statefulSetTestResourceName,
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesStatefulSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetConfigBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists(statefulSetTestResourceName, &conf),
					testAccKubernetesStatefulSetChecksBasic(name),
				),
			},
			{
				Config: testAccKubernetesStatefulSetConfigUpdateTemplate(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists(statefulSetTestResourceName, &conf),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.template.0.spec.0.container.0.port.#", "2"),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.template.0.spec.0.container.0.port.0.container_port", "80"),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.template.0.spec.0.container.0.port.0.name", "web"),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.template.0.spec.0.container.0.port.1.container_port", "443"),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.template.0.spec.0.container.0.port.1.name", "secure"),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.template.0.spec.0.dns_config.#", "1"),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.template.0.spec.0.dns_config.0.nameservers.#", "3"),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.template.0.spec.0.dns_config.0.nameservers.0", "1.1.1.1"),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.template.0.spec.0.dns_config.0.nameservers.1", "8.8.8.8"),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.template.0.spec.0.dns_config.0.nameservers.2", "9.9.9.9"),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.template.0.spec.0.dns_config.0.searches.#", "1"),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.template.0.spec.0.dns_config.0.searches.0", "kubernetes.io"),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.template.0.spec.0.dns_config.0.option.#", "2"),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.template.0.spec.0.dns_config.0.option.0.name", "ndots"),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.template.0.spec.0.dns_config.0.option.0.value", "1"),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.template.0.spec.0.dns_config.0.option.1.name", "use-vc"),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.template.0.spec.0.dns_config.0.option.1.value", ""),
					resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.template.0.spec.0.dns_policy", "Default"),
				),
			},
		},
	})
}

func testAccCheckKubernetesStatefulSetDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*KubeClientsets).MainClientset

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_stateful_set" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.AppsV1().StatefulSets(namespace).Get(name, metav1.GetOptions{})
		if err == nil {
			if resp.Namespace == namespace && resp.Name == name {
				return fmt.Errorf("StatefulSet still exists: %s: (Generation %#v)", rs.Primary.ID, resp.Status.ObservedGeneration)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesStatefulSetExists(n string, obj *api.StatefulSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*KubeClientsets).MainClientset

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		out, err := conn.AppsV1().StatefulSets(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		*obj = *out
		return nil
	}
}

func testAccKubernetesStatefulSetChecksBasic(name string) resource.TestCheckFunc {
	return resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttrSet(statefulSetTestResourceName, "metadata.0.generation"),
		resource.TestCheckResourceAttrSet(statefulSetTestResourceName, "metadata.0.resource_version"),
		resource.TestCheckResourceAttrSet(statefulSetTestResourceName, "metadata.0.self_link"),
		resource.TestCheckResourceAttrSet(statefulSetTestResourceName, "metadata.0.uid"),
		resource.TestCheckResourceAttrSet(statefulSetTestResourceName, "metadata.0.namespace"),
		resource.TestCheckResourceAttr(statefulSetTestResourceName, "metadata.0.annotations.%", "2"),
		resource.TestCheckResourceAttr(statefulSetTestResourceName, "metadata.0.labels.%", "3"),
		resource.TestCheckResourceAttr(statefulSetTestResourceName, "metadata.0.labels.TestLabelOne", "one"),
		resource.TestCheckResourceAttr(statefulSetTestResourceName, "metadata.0.labels.TestLabelTwo", "two"),
		resource.TestCheckResourceAttr(statefulSetTestResourceName, "metadata.0.labels.TestLabelThree", "three"),
		resource.TestCheckResourceAttr(statefulSetTestResourceName, "metadata.0.name", name),
		resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.#", "1"),
		resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.replicas", "1"),
		resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.revision_history_limit", "11"),
		resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.service_name", "ss-test-service"),
		resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.selector.0.match_labels.%", "1"),
		resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.selector.0.match_labels.app", "ss-test"),
		resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.template.0.metadata.#", "1"),
		resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.template.0.metadata.0.labels.%", "1"),
		resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.template.0.metadata.0.labels.app", "ss-test"),
		resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.template.0.spec.0.container.0.name", "ss-test"),
		resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.template.0.spec.0.container.0.image", "k8s.gcr.io/pause:latest"),
		resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.template.0.spec.0.container.0.port.0.container_port", "80"),
		resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.template.0.spec.0.container.0.port.0.name", "web"),
		resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.name", "workdir"),
		resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.mount_path", "/work-dir"),
		resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.update_strategy.0.type", "RollingUpdate"),
		resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.update_strategy.0.rolling_update.#", "1"),
		resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.update_strategy.0.rolling_update.0.partition", "1"),
		resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.volume_claim_template.0.metadata.0.name", "ss-test"),
		resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.volume_claim_template.0.spec.0.access_modes.#", "1"),
		resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.volume_claim_template.0.spec.0.access_modes.1245328686", "ReadWriteOnce"),
		resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.volume_claim_template.0.spec.0.resources.#", "1"),
		resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.volume_claim_template.0.spec.0.resources.0.requests.%", "1"),
		resource.TestCheckResourceAttr(statefulSetTestResourceName, "spec.0.volume_claim_template.0.spec.0.resources.0.requests.storage", "1Gi"),
	)
}

func testAccKubernetesStatefulSetConfigBasic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_stateful_set" "test" {
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
            name       = "workdir"
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
	return fmt.Sprintf(`
resource "kubernetes_stateful_set" "test" {
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
            name       = "workdir"
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
	return fmt.Sprintf(`
resource "kubernetes_stateful_set" "test" {
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
            name       = "workdir"
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

func testAccKubernetesStatefulSetConfigUpdateReplicas(name string, replicas int) string {
	return fmt.Sprintf(`
resource "kubernetes_stateful_set" "test" {
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
    replicas               = %d
    revision_history_limit = 11

    selector {
      match_labels ={
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
            name       = "workdir"
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
	return fmt.Sprintf(`
resource "kubernetes_stateful_set" "test" {
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
	return fmt.Sprintf(`
resource "kubernetes_stateful_set" "test" {
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
            name       = "workdir"
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
	return fmt.Sprintf(`
resource "kubernetes_stateful_set" "test" {
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
            name       = "workdir"
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
}
`, name)
}
