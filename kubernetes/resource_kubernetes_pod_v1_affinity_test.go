// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	api "k8s.io/api/core/v1"
)

func TestAccKubernetesPod_with_node_affinity_with_required_during_scheduling_ignored_during_execution(t *testing.T) {
	var conf api.Pod
	podName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion
	keyName := "spec.0.affinity.0.node_affinity.0.required_during_scheduling_ignored_during_execution"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigWithNodeAffinityWithRequiredDuringSchedulingIgnoredDuringExecution(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists("kubernetes_pod.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.node_affinity.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.#", keyName), "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.node_selector_term.#", keyName), "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.node_selector_term.0.match_expressions.#", keyName), "2"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.node_selector_term.0.match_expressions.0.key", keyName), "kubernetes.io/hostname"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.node_selector_term.0.match_expressions.0.operator", keyName), "NotIn"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.node_selector_term.0.match_expressions.0.values.0", keyName), "bar"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.node_selector_term.0.match_expressions.0.values.1", keyName), "foo"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.node_selector_term.0.match_expressions.1.key", keyName), "kubernetes.io/os"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.node_selector_term.0.match_expressions.1.operator", keyName), "In"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.node_selector_term.0.match_expressions.1.values.0", keyName), "linux"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_with_node_affinity_with_preferred_during_scheduling_ignored_during_execution(t *testing.T) {
	var conf api.Pod
	podName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion
	keyName := "spec.0.affinity.0.node_affinity.0.preferred_during_scheduling_ignored_during_execution"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigWithNodeAffinityWithPreferredDuringSchedulingIgnoredDuringExecution(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists("kubernetes_pod.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.node_affinity.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.#", keyName), "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.preference.0.match_expressions.#", keyName), "2"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.preference.0.match_expressions.0.key", keyName), "kubernetes.io/hostname"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.preference.0.match_expressions.0.%%", keyName), "3"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.preference.0.match_expressions.0.operator", keyName), "NotIn"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.preference.0.match_expressions.0.values.#", keyName), "2"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.preference.0.match_expressions.0.values.0", keyName), "bar"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.preference.0.match_expressions.0.values.1", keyName), "foo"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.preference.0.match_expressions.1.%%", keyName), "3"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.preference.0.match_expressions.1.key", keyName), "kubernetes.io/os"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.preference.0.match_expressions.1.operator", keyName), "In"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.preference.0.match_expressions.1.values.#", keyName), "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.preference.0.match_expressions.1.values.0", keyName), "linux"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.weight", keyName), "1"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_with_pod_affinity_with_required_during_scheduling_ignored_during_execution(t *testing.T) {
	var conf api.Pod
	podName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion
	keyName := "spec.0.affinity.0.pod_affinity.0.required_during_scheduling_ignored_during_execution"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigWithPodAffinityWithRequiredDuringSchedulingIgnoredDuringExecution(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists("kubernetes_pod.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_affinity.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.#", keyName), "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.label_selector.#", keyName), "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.label_selector.0.match_expressions.#", keyName), "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.label_selector.0.match_expressions.0.key", keyName), "security"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.label_selector.0.match_expressions.0.operator", keyName), "NotIn"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.label_selector.0.match_expressions.0.values.#", keyName), "2"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.label_selector.0.match_expressions.0.values.0", keyName), "bar"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.label_selector.0.match_expressions.0.values.1", keyName), "foo"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.label_selector.0.match_labels.%%", keyName), "0"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.namespaces.#", keyName), "0"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.topology_key", keyName), "kubernetes.io/hostname"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_with_pod_affinity_with_preferred_during_scheduling_ignored_during_execution(t *testing.T) {
	var conf api.Pod
	podName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion
	keyName := "spec.0.affinity.0.pod_affinity.0.preferred_during_scheduling_ignored_during_execution"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigWithPodAffinityWithPreferredDuringSchedulingIgnoredDuringExecution(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists("kubernetes_pod.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_affinity.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.#", keyName), "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.pod_affinity_term.#", keyName), "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.#", keyName), "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.0.match_expressions.#", keyName), "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.0.match_expressions.0.key", keyName), "security"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.0.match_expressions.0.operator", keyName), "NotIn"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.0.match_expressions.0.values.#", keyName), "2"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.0.match_expressions.0.values.0", keyName), "bar"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.0.match_expressions.0.values.1", keyName), "foo"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.0.match_labels.%%", keyName), "0"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.pod_affinity_term.0.namespaces.#", keyName), "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.pod_affinity_term.0.namespaces.0", keyName), "default"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.pod_affinity_term.0.topology_key", keyName), "kubernetes.io/hostname"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.weight", keyName), "100"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_with_pod_anti_affinity_with_required_during_scheduling_ignored_during_execution(t *testing.T) {
	var conf api.Pod
	podName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion
	keyName := "spec.0.affinity.0.pod_anti_affinity.0.required_during_scheduling_ignored_during_execution"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfRunningInMinikube(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigWithPodAntiAffinityWithRequiredDuringSchedulingIgnoredDuringExecution(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists("kubernetes_pod.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_anti_affinity.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.#", keyName), "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.label_selector.#", keyName), "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.label_selector.0.match_expressions.#", keyName), "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.label_selector.0.match_expressions.0.key", keyName), "security"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.label_selector.0.match_expressions.0.operator", keyName), "NotIn"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.label_selector.0.match_expressions.0.values.#", keyName), "2"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.label_selector.0.match_expressions.0.values.0", keyName), "bar"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.label_selector.0.match_expressions.0.values.1", keyName), "foo"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.label_selector.0.match_labels.%%", keyName), "0"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.namespaces.#", keyName), "0"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.topology_key", keyName), "kubernetes.io/hostname"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_with_pod_anti_affinity_with_preferred_during_scheduling_ignored_during_execution(t *testing.T) {
	var conf api.Pod
	podName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := nginxImageVersion
	keyName := "spec.0.affinity.0.pod_anti_affinity.0.preferred_during_scheduling_ignored_during_execution"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfRunningInMinikube(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigWithPodAntiAffinityWithPreferredDuringSchedulingIgnoredDuringExecution(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists("kubernetes_pod.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_anti_affinity.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.#", keyName), "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.pod_affinity_term.#", keyName), "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.#", keyName), "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.0.match_expressions.#", keyName), "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.0.match_expressions.0.key", keyName), "security"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.0.match_expressions.0.operator", keyName), "NotIn"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.0.match_expressions.0.values.#", keyName), "2"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.0.match_expressions.0.values.0", keyName), "bar"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.0.match_expressions.0.values.1", keyName), "foo"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.0.match_labels.%%", keyName), "0"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.pod_affinity_term.0.namespaces.#", keyName), "0"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.pod_affinity_term.0.topology_key", keyName), "kubernetes.io/hostname"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", fmt.Sprintf("%s.0.weight", keyName), "100"),
				),
			},
		},
	})
}

func testAccKubernetesPodConfigWithNodeAffinityWithRequiredDuringSchedulingIgnoredDuringExecution(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }
    name = "%s"
  }
  spec {
    affinity {
      node_affinity {
        required_during_scheduling_ignored_during_execution {
          node_selector_term {
            match_expressions {
              key      = "kubernetes.io/hostname"
              operator = "NotIn"
              values   = ["foo", "bar"]
            }
            match_expressions {
              key      = "kubernetes.io/os"
              operator = "In"
              values   = ["linux"]
            }
          }
        }
      }
    }
    container {
      image = "%s"
      name  = "containername"
      resources {
        limits = {
          cpu    = "50m"
          memory = "50M"
        }
      }
    }
  }
}
    `, podName, imageName)
}

func testAccKubernetesPodConfigWithNodeAffinityWithPreferredDuringSchedulingIgnoredDuringExecution(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }
    name = "%s"
  }
  spec {
    affinity {
      node_affinity {
        preferred_during_scheduling_ignored_during_execution {
          weight = 1
          preference {
            match_expressions {
              key      = "kubernetes.io/hostname"
              operator = "NotIn"
              values   = ["foo", "bar"]
            }
            match_expressions {
              key      = "kubernetes.io/os"
              operator = "In"
              values   = ["linux"]
            }
          }
        }
      }
    }
    container {
      image = "%s"
      name  = "containername"
      resources {
        limits = {
          cpu    = "50m"
          memory = "50M"
        }
      }
    }
  }
}
    `, podName, imageName)
}

func testAccKubernetesPodConfigWithPodAffinityWithRequiredDuringSchedulingIgnoredDuringExecution(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }
    name = "%s"
  }
  spec {
    affinity {
      pod_affinity {
        required_during_scheduling_ignored_during_execution {
          label_selector {
            match_expressions {
              key      = "security"
              operator = "NotIn"
              values   = ["foo", "bar"]
            }
          }
          topology_key = "kubernetes.io/hostname"
        }
      }
    }
    container {
      image = "%s"
      name  = "containername"
      resources {
        limits = {
          cpu    = "200m"
          memory = "1024M"
        }
      }
    }
  }
}
    `, podName, imageName)
}

func testAccKubernetesPodConfigWithPodAffinityWithPreferredDuringSchedulingIgnoredDuringExecution(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }
    name = "%s"
  }
  spec {
    affinity {
      pod_affinity {
        preferred_during_scheduling_ignored_during_execution {
          weight = 100
          pod_affinity_term {
            label_selector {
              match_expressions {
                key      = "security"
                operator = "NotIn"
                values   = ["foo", "bar"]
              }
            }
            namespaces   = ["default"]
            topology_key = "kubernetes.io/hostname"
          }
        }
      }
    }
    container {
      image = "%s"
      name  = "containername"
      resources {
        limits = {
          cpu    = "200m"
          memory = "1024M"
        }
      }
    }
  }
}
    `, podName, imageName)
}

func testAccKubernetesPodConfigWithPodAntiAffinityWithRequiredDuringSchedulingIgnoredDuringExecution(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }
    name = "%s"
  }
  spec {
    affinity {
      pod_anti_affinity {
        required_during_scheduling_ignored_during_execution {
          label_selector {
            match_expressions {
              key      = "security"
              operator = "NotIn"
              values   = ["foo", "bar"]
            }
          }
          topology_key = "kubernetes.io/hostname"
        }
      }
    }
    container {
      image = "%s"
      name  = "containername"
      resources {
        limits = {
          cpu    = "200m"
          memory = "1024M"
        }
      }

    }
  }
}
    `, podName, imageName)
}

func testAccKubernetesPodConfigWithPodAntiAffinityWithPreferredDuringSchedulingIgnoredDuringExecution(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }
    name = "%s"
  }
  spec {
    affinity {
      pod_anti_affinity {
        preferred_during_scheduling_ignored_during_execution {
          weight = 100
          pod_affinity_term {
            label_selector {
              match_expressions {
                key      = "security"
                operator = "NotIn"
                values   = ["foo", "bar"]
              }
            }
            topology_key = "kubernetes.io/hostname"
          }
        }
      }
    }
    container {
      image = "%s"
      name  = "containername"
      resources {
        limits = {
          cpu    = "200m"
          memory = "1024M"
        }
      }
    }
  }
}
    `, podName, imageName)
}
