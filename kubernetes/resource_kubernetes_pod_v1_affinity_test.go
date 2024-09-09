// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	corev1 "k8s.io/api/core/v1"
)

func TestAccKubernetesPodV1_with_node_affinity_with_required_during_scheduling_ignored_during_execution(t *testing.T) {
	var conf corev1.Pod
	podName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImage
	keyName := "spec.0.affinity.0.node_affinity.0.required_during_scheduling_ignored_during_execution"
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigWithNodeAffinityWithRequiredDuringSchedulingIgnoredDuringExecution_MatchExpressions(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.affinity.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.affinity.0.node_affinity.#", "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.#", keyName), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.node_selector_term.#", keyName), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.node_selector_term.0.match_expressions.#", keyName), "2"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.node_selector_term.0.match_expressions.0.key", keyName), "kubernetes.io/hostname"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.node_selector_term.0.match_expressions.0.operator", keyName), "NotIn"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.node_selector_term.0.match_expressions.0.values.0", keyName), "bar"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.node_selector_term.0.match_expressions.0.values.1", keyName), "foo"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.node_selector_term.0.match_expressions.1.key", keyName), "kubernetes.io/os"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.node_selector_term.0.match_expressions.1.operator", keyName), "In"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.node_selector_term.0.match_expressions.1.values.0", keyName), "linux"),
				),
			},
			{
				Config: testAccKubernetesPodV1ConfigWithNodeAffinityWithRequiredDuringSchedulingIgnoredDuringExecution_MatchFields(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.affinity.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.affinity.0.node_affinity.#", "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.#", keyName), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.node_selector_term.0.match_fields.#", keyName), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.node_selector_term.0.match_fields.0.key", keyName), "metadata.name"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.node_selector_term.0.match_fields.0.%%", keyName), "3"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.node_selector_term.0.match_fields.0.operator", keyName), "NotIn"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.node_selector_term.0.match_fields.0.values.#", keyName), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.node_selector_term.0.match_fields.0.values.0", keyName), "foo"),
				),
			},
		},
	})
}

func TestAccKubernetesPodV1_with_node_affinity_with_preferred_during_scheduling_ignored_during_execution(t *testing.T) {
	var conf corev1.Pod
	podName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImage
	keyName := "spec.0.affinity.0.node_affinity.0.preferred_during_scheduling_ignored_during_execution"
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigWithNodeAffinityWithPreferredDuringSchedulingIgnoredDuringExecution_MatchExpressions(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.affinity.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.affinity.0.node_affinity.#", "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.#", keyName), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.preference.0.match_expressions.#", keyName), "2"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.preference.0.match_expressions.0.key", keyName), "kubernetes.io/hostname"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.preference.0.match_expressions.0.%%", keyName), "3"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.preference.0.match_expressions.0.operator", keyName), "NotIn"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.preference.0.match_expressions.0.values.#", keyName), "2"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.preference.0.match_expressions.0.values.0", keyName), "bar"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.preference.0.match_expressions.0.values.1", keyName), "foo"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.preference.0.match_expressions.1.%%", keyName), "3"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.preference.0.match_expressions.1.key", keyName), "kubernetes.io/os"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.preference.0.match_expressions.1.operator", keyName), "In"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.preference.0.match_expressions.1.values.#", keyName), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.preference.0.match_expressions.1.values.0", keyName), "linux"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.weight", keyName), "1"),
				),
			},
			{
				Config: testAccKubernetesPodV1ConfigWithNodeAffinityWithPreferredDuringSchedulingIgnoredDuringExecution_MatchFields(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.affinity.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.affinity.0.node_affinity.#", "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.#", keyName), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.preference.0.match_fields.#", keyName), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.preference.0.match_fields.0.key", keyName), "metadata.name"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.preference.0.match_fields.0.%%", keyName), "3"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.preference.0.match_fields.0.operator", keyName), "NotIn"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.preference.0.match_fields.0.values.#", keyName), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.preference.0.match_fields.0.values.0", keyName), "foo"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.weight", keyName), "1"),
				),
			},
		},
	})
}

func TestAccKubernetesPodV1_with_pod_affinity_with_required_during_scheduling_ignored_during_execution(t *testing.T) {
	var conf corev1.Pod
	podName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImage
	keyName := "spec.0.affinity.0.pod_affinity.0.required_during_scheduling_ignored_during_execution"
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigWithPodAffinityWithRequiredDuringSchedulingIgnoredDuringExecution(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.affinity.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.affinity.0.pod_affinity.#", "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.#", keyName), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.label_selector.#", keyName), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.label_selector.0.match_expressions.#", keyName), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.label_selector.0.match_expressions.0.key", keyName), "security"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.label_selector.0.match_expressions.0.operator", keyName), "NotIn"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.label_selector.0.match_expressions.0.values.#", keyName), "2"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.label_selector.0.match_expressions.0.values.0", keyName), "bar"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.label_selector.0.match_expressions.0.values.1", keyName), "foo"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.label_selector.0.match_labels.%%", keyName), "0"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.namespaces.#", keyName), "0"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.topology_key", keyName), "kubernetes.io/hostname"),
				),
			},
		},
	})
}

func TestAccKubernetesPodV1_with_pod_affinity_with_preferred_during_scheduling_ignored_during_execution(t *testing.T) {
	var conf corev1.Pod
	podName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImage
	keyName := "spec.0.affinity.0.pod_affinity.0.preferred_during_scheduling_ignored_during_execution"
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigWithPodAffinityWithPreferredDuringSchedulingIgnoredDuringExecution(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.affinity.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.affinity.0.pod_affinity.#", "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.#", keyName), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.pod_affinity_term.#", keyName), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.#", keyName), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.0.match_expressions.#", keyName), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.0.match_expressions.0.key", keyName), "security"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.0.match_expressions.0.operator", keyName), "NotIn"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.0.match_expressions.0.values.#", keyName), "2"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.0.match_expressions.0.values.0", keyName), "bar"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.0.match_expressions.0.values.1", keyName), "foo"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.0.match_labels.%%", keyName), "0"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.pod_affinity_term.0.namespaces.#", keyName), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.pod_affinity_term.0.namespaces.0", keyName), "default"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.pod_affinity_term.0.topology_key", keyName), "kubernetes.io/hostname"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.weight", keyName), "100"),
				),
			},
		},
	})
}

func TestAccKubernetesPodV1_with_pod_anti_affinity_with_required_during_scheduling_ignored_during_execution(t *testing.T) {
	var conf corev1.Pod
	podName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImage
	keyName := "spec.0.affinity.0.pod_anti_affinity.0.required_during_scheduling_ignored_during_execution"
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfRunningInMinikube(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigWithPodAntiAffinityWithRequiredDuringSchedulingIgnoredDuringExecution(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.affinity.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.affinity.0.pod_anti_affinity.#", "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.#", keyName), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.label_selector.#", keyName), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.label_selector.0.match_expressions.#", keyName), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.label_selector.0.match_expressions.0.key", keyName), "security"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.label_selector.0.match_expressions.0.operator", keyName), "NotIn"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.label_selector.0.match_expressions.0.values.#", keyName), "2"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.label_selector.0.match_expressions.0.values.0", keyName), "bar"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.label_selector.0.match_expressions.0.values.1", keyName), "foo"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.label_selector.0.match_labels.%%", keyName), "0"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.namespaces.#", keyName), "0"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.topology_key", keyName), "kubernetes.io/hostname"),
				),
			},
		},
	})
}

func TestAccKubernetesPodV1_with_pod_anti_affinity_with_preferred_during_scheduling_ignored_during_execution(t *testing.T) {
	var conf corev1.Pod
	podName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImage
	keyName := "spec.0.affinity.0.pod_anti_affinity.0.preferred_during_scheduling_ignored_during_execution"
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfRunningInMinikube(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigWithPodAntiAffinityWithPreferredDuringSchedulingIgnoredDuringExecution(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.affinity.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.affinity.0.pod_anti_affinity.#", "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.#", keyName), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.pod_affinity_term.#", keyName), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.#", keyName), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.0.match_expressions.#", keyName), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.0.match_expressions.0.key", keyName), "security"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.0.match_expressions.0.operator", keyName), "NotIn"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.0.match_expressions.0.values.#", keyName), "2"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.0.match_expressions.0.values.0", keyName), "bar"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.0.match_expressions.0.values.1", keyName), "foo"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.pod_affinity_term.0.label_selector.0.match_labels.%%", keyName), "0"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.pod_affinity_term.0.namespaces.#", keyName), "0"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.pod_affinity_term.0.topology_key", keyName), "kubernetes.io/hostname"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.weight", keyName), "100"),
				),
			},
		},
	})
}

func testAccKubernetesPodV1ConfigWithNodeAffinityWithRequiredDuringSchedulingIgnoredDuringExecution_MatchExpressions(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_pod_v1" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }
    namespace = kubernetes_namespace_v1.test.metadata.0.name
    name      = %[1]q
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
      image = %[2]q
      name  = "containername"
      args  = ["sleep", "300"]
      resources {
        limits = {
          cpu    = "50m"
          memory = "64M"
        }
      }
    }
    termination_grace_period_seconds = 1
  }
}
`, podName, imageName)
}

func testAccKubernetesPodV1ConfigWithNodeAffinityWithRequiredDuringSchedulingIgnoredDuringExecution_MatchFields(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_pod_v1" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }
    namespace = kubernetes_namespace_v1.test.metadata.0.name
    name      = %[1]q
  }
  spec {
    affinity {
      node_affinity {
        required_during_scheduling_ignored_during_execution {
          node_selector_term {
            match_fields {
              key      = "metadata.name"
              operator = "NotIn"
              values   = ["foo"]
            }
          }
        }
      }
    }
    container {
      image = %[2]q
      name  = "containername"
      args  = ["sleep", "300"]
      resources {
        limits = {
          cpu    = "50m"
          memory = "64M"
        }
      }
    }
    termination_grace_period_seconds = 1
  }
}
`, podName, imageName)
}

func testAccKubernetesPodV1ConfigWithNodeAffinityWithPreferredDuringSchedulingIgnoredDuringExecution_MatchExpressions(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_pod_v1" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }
    namespace = kubernetes_namespace_v1.test.metadata.0.name
    name      = %[1]q
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
      image = %[2]q
      name  = "containername"
      args  = ["sleep", "300"]
      resources {
        limits = {
          cpu    = "50m"
          memory = "50M"
        }
      }
    }
    termination_grace_period_seconds = 1
  }
}
`, podName, imageName)
}

func testAccKubernetesPodV1ConfigWithNodeAffinityWithPreferredDuringSchedulingIgnoredDuringExecution_MatchFields(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_pod_v1" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }
    namespace = kubernetes_namespace_v1.test.metadata.0.name
    name      = %[1]q
  }
  spec {
    affinity {
      node_affinity {
        preferred_during_scheduling_ignored_during_execution {
          weight = 1
          preference {
            match_fields {
              key      = "metadata.name"
              operator = "NotIn"
              values   = ["foo"]
            }
          }
        }
      }
    }
    container {
      image = %[2]q
      name  = "containername"
      args  = ["sleep", "300"]
      resources {
        limits = {
          cpu    = "50m"
          memory = "50M"
        }
      }
    }
    termination_grace_period_seconds = 1
  }
}
`, podName, imageName)
}

func testAccKubernetesPodV1ConfigWithPodAffinityWithRequiredDuringSchedulingIgnoredDuringExecution(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_pod_v1" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }
    namespace = kubernetes_namespace_v1.test.metadata.0.name
    name      = %[1]q
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
      image = %[2]q
      name  = "containername"
      args  = ["sleep", "300"]
      resources {
        limits = {
          cpu    = "200m"
          memory = "64M"
        }
      }
    }
    termination_grace_period_seconds = 1
  }
}
`, podName, imageName)
}

func testAccKubernetesPodV1ConfigWithPodAffinityWithPreferredDuringSchedulingIgnoredDuringExecution(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_pod_v1" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }
    namespace = kubernetes_namespace_v1.test.metadata.0.name
    name      = %[1]q
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
      image = %[2]q
      name  = "containername"
      args  = ["sleep", "300"]
      resources {
        limits = {
          cpu    = "200m"
          memory = "64M"
        }
      }
    }
    termination_grace_period_seconds = 1
  }
}
`, podName, imageName)
}

func testAccKubernetesPodV1ConfigWithPodAntiAffinityWithRequiredDuringSchedulingIgnoredDuringExecution(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_pod_v1" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }
    namespace = kubernetes_namespace_v1.test.metadata.0.name
    name      = %[1]q
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
      image = %[2]q
      name  = "containername"
      args  = ["sleep", "300"]
      resources {
        limits = {
          cpu    = "200m"
          memory = "64M"
        }
      }
    }
    termination_grace_period_seconds = 1
  }
}
`, podName, imageName)
}

func testAccKubernetesPodV1ConfigWithPodAntiAffinityWithPreferredDuringSchedulingIgnoredDuringExecution(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_pod_v1" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }
    namespace = kubernetes_namespace_v1.test.metadata.0.name
    name      = %[1]q
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
      image = %[2]q
      name  = "containername"
      args  = ["sleep", "300"]
      resources {
        limits = {
          cpu    = "200m"
          memory = "64M"
        }
      }
    }
    termination_grace_period_seconds = 1
  }
}
`, podName, imageName)
}
