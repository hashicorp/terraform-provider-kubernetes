package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	api "k8s.io/api/core/v1"
)

func TestAccKubernetesPod_with_node_affinity_with_required_during_scheduling_ignored_during_execution(t *testing.T) {
	var conf api.Pod
	podName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := "nginx:1.7.9"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigWithNodeAffinityWithRequiredDuringSchedulingIgnoredDuringExecution(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.node_affinity.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.node_affinity.0.required_during_scheduling_ignored_during_execution.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.node_affinity.0.required_during_scheduling_ignored_during_execution.0.node_selector_term.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.node_affinity.0.required_during_scheduling_ignored_during_execution.0.node_selector_term.0.match_expressions.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.node_affinity.0.required_during_scheduling_ignored_during_execution.0.node_selector_term.0.match_expressions.0.key", "kubernetes.io/hostname"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.node_affinity.0.required_during_scheduling_ignored_during_execution.0.node_selector_term.0.match_expressions.0.operator", "NotIn"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.node_affinity.0.required_during_scheduling_ignored_during_execution.0.node_selector_term.0.match_expressions.0.values.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.node_affinity.0.required_during_scheduling_ignored_during_execution.0.node_selector_term.0.match_expressions.0.values.2356372769", "foo"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.node_affinity.0.required_during_scheduling_ignored_during_execution.0.node_selector_term.0.match_expressions.0.values.1996459178", "bar"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.node_affinity.0.required_during_scheduling_ignored_during_execution.0.node_selector_term.0.match_expressions.1.key", "beta.kubernetes.io/os"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.node_affinity.0.required_during_scheduling_ignored_during_execution.0.node_selector_term.0.match_expressions.1.operator", "In"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.node_affinity.0.required_during_scheduling_ignored_during_execution.0.node_selector_term.0.match_expressions.1.values.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.node_affinity.0.required_during_scheduling_ignored_during_execution.0.node_selector_term.0.match_expressions.1.values.2450605903", "linux"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_with_node_affinity_with_preferred_during_scheduling_ignored_during_execution(t *testing.T) {
	var conf api.Pod
	podName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := "nginx:1.7.9"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigWithNodeAffinityWithPreferredDuringSchedulingIgnoredDuringExecution(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.node_affinity.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.node_affinity.0.preferred_during_scheduling_ignored_during_execution.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.node_affinity.0.preferred_during_scheduling_ignored_during_execution.0.preference.0.match_expressions.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.node_affinity.0.preferred_during_scheduling_ignored_during_execution.0.preference.0.match_expressions.0.key", "kubernetes.io/hostname"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.node_affinity.0.preferred_during_scheduling_ignored_during_execution.0.preference.0.match_expressions.0.operator", "NotIn"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.node_affinity.0.preferred_during_scheduling_ignored_during_execution.0.preference.0.match_expressions.0.values.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.node_affinity.0.preferred_during_scheduling_ignored_during_execution.0.preference.0.match_expressions.0.values.1996459178", "bar"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.node_affinity.0.preferred_during_scheduling_ignored_during_execution.0.preference.0.match_expressions.0.values.2356372769", "foo"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.node_affinity.0.preferred_during_scheduling_ignored_during_execution.0.preference.0.match_expressions.1.key", "beta.kubernetes.io/os"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.node_affinity.0.preferred_during_scheduling_ignored_during_execution.0.preference.0.match_expressions.1.operator", "In"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.node_affinity.0.preferred_during_scheduling_ignored_during_execution.0.preference.0.match_expressions.1.values.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.node_affinity.0.preferred_during_scheduling_ignored_during_execution.0.preference.0.match_expressions.1.values.2450605903", "linux"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.node_affinity.0.preferred_during_scheduling_ignored_during_execution.0.weight", "1"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_with_pod_affinity_with_required_during_scheduling_ignored_during_execution(t *testing.T) {
	var conf api.Pod
	podName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := "nginx:1.7.9"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigWithPodAffinityWithRequiredDuringSchedulingIgnoredDuringExecution(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_affinity.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_affinity.0.required_during_scheduling_ignored_during_execution.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_affinity.0.required_during_scheduling_ignored_during_execution.0.label_selector.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_affinity.0.required_during_scheduling_ignored_during_execution.0.label_selector.0.match_expressions.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_affinity.0.required_during_scheduling_ignored_during_execution.0.label_selector.0.match_expressions.0.key", "security"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_affinity.0.required_during_scheduling_ignored_during_execution.0.label_selector.0.match_expressions.0.operator", "NotIn"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_affinity.0.required_during_scheduling_ignored_during_execution.0.label_selector.0.match_expressions.0.values.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_affinity.0.required_during_scheduling_ignored_during_execution.0.label_selector.0.match_expressions.0.values.1996459178", "bar"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_affinity.0.required_during_scheduling_ignored_during_execution.0.label_selector.0.match_expressions.0.values.2356372769", "foo"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_affinity.0.required_during_scheduling_ignored_during_execution.0.label_selector.0.match_labels.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_affinity.0.required_during_scheduling_ignored_during_execution.0.namespaces.#", "0"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_affinity.0.required_during_scheduling_ignored_during_execution.0.topology_key", "kubernetes.io/hostname"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_with_pod_affinity_with_preferred_during_scheduling_ignored_during_execution(t *testing.T) {
	var conf api.Pod
	podName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := "nginx:1.7.9"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigWithPodAffinityWithPreferredDuringSchedulingIgnoredDuringExecution(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_affinity.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_affinity.0.preferred_during_scheduling_ignored_during_execution.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_affinity.0.preferred_during_scheduling_ignored_during_execution.0.pod_affinity_term.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_affinity.0.preferred_during_scheduling_ignored_during_execution.0.pod_affinity_term.0.label_selector.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_affinity.0.preferred_during_scheduling_ignored_during_execution.0.pod_affinity_term.0.label_selector.0.match_expressions.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_affinity.0.preferred_during_scheduling_ignored_during_execution.0.pod_affinity_term.0.label_selector.0.match_expressions.0.key", "security"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_affinity.0.preferred_during_scheduling_ignored_during_execution.0.pod_affinity_term.0.label_selector.0.match_expressions.0.operator", "NotIn"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_affinity.0.preferred_during_scheduling_ignored_during_execution.0.pod_affinity_term.0.label_selector.0.match_expressions.0.values.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_affinity.0.preferred_during_scheduling_ignored_during_execution.0.pod_affinity_term.0.label_selector.0.match_expressions.0.values.1996459178", "bar"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_affinity.0.preferred_during_scheduling_ignored_during_execution.0.pod_affinity_term.0.label_selector.0.match_expressions.0.values.2356372769", "foo"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_affinity.0.preferred_during_scheduling_ignored_during_execution.0.pod_affinity_term.0.label_selector.0.match_labels.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_affinity.0.preferred_during_scheduling_ignored_during_execution.0.pod_affinity_term.0.namespaces.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_affinity.0.preferred_during_scheduling_ignored_during_execution.0.pod_affinity_term.0.namespaces.3814588639", "default"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_affinity.0.preferred_during_scheduling_ignored_during_execution.0.pod_affinity_term.0.topology_key", "kubernetes.io/hostname"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_affinity.0.preferred_during_scheduling_ignored_during_execution.0.weight", "100"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_with_pod_anti_affinity_with_required_during_scheduling_ignored_during_execution(t *testing.T) {
	var conf api.Pod
	podName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := "nginx:1.7.9"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigWithPodAntiAffinityWithRequiredDuringSchedulingIgnoredDuringExecution(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_anti_affinity.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_anti_affinity.0.required_during_scheduling_ignored_during_execution.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_anti_affinity.0.required_during_scheduling_ignored_during_execution.0.label_selector.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_anti_affinity.0.required_during_scheduling_ignored_during_execution.0.label_selector.0.match_expressions.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_anti_affinity.0.required_during_scheduling_ignored_during_execution.0.label_selector.0.match_expressions.0.key", "security"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_anti_affinity.0.required_during_scheduling_ignored_during_execution.0.label_selector.0.match_expressions.0.operator", "NotIn"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_anti_affinity.0.required_during_scheduling_ignored_during_execution.0.label_selector.0.match_expressions.0.values.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_anti_affinity.0.required_during_scheduling_ignored_during_execution.0.label_selector.0.match_expressions.0.values.1996459178", "bar"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_anti_affinity.0.required_during_scheduling_ignored_during_execution.0.label_selector.0.match_expressions.0.values.2356372769", "foo"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_anti_affinity.0.required_during_scheduling_ignored_during_execution.0.label_selector.0.match_labels.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_anti_affinity.0.required_during_scheduling_ignored_during_execution.0.namespaces.#", "0"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_anti_affinity.0.required_during_scheduling_ignored_during_execution.0.topology_key", "kubernetes.io/hostname"),
				),
			},
		},
	})
}

func TestAccKubernetesPod_with_pod_anti_affinity_with_preferred_during_scheduling_ignored_during_execution(t *testing.T) {
	var conf api.Pod
	podName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := "nginx:1.7.9"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodConfigWithPodAntiAffinityWithPreferredDuringSchedulingIgnoredDuringExecution(podName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodExists("kubernetes_pod.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_anti_affinity.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_anti_affinity.0.preferred_during_scheduling_ignored_during_execution.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_anti_affinity.0.preferred_during_scheduling_ignored_during_execution.0.pod_affinity_term.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_anti_affinity.0.preferred_during_scheduling_ignored_during_execution.0.pod_affinity_term.0.label_selector.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_anti_affinity.0.preferred_during_scheduling_ignored_during_execution.0.pod_affinity_term.0.label_selector.0.match_expressions.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_anti_affinity.0.preferred_during_scheduling_ignored_during_execution.0.pod_affinity_term.0.label_selector.0.match_expressions.0.key", "security"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_anti_affinity.0.preferred_during_scheduling_ignored_during_execution.0.pod_affinity_term.0.label_selector.0.match_expressions.0.operator", "NotIn"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_anti_affinity.0.preferred_during_scheduling_ignored_during_execution.0.pod_affinity_term.0.label_selector.0.match_expressions.0.values.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_anti_affinity.0.preferred_during_scheduling_ignored_during_execution.0.pod_affinity_term.0.label_selector.0.match_expressions.0.values.1996459178", "bar"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_anti_affinity.0.preferred_during_scheduling_ignored_during_execution.0.pod_affinity_term.0.label_selector.0.match_expressions.0.values.2356372769", "foo"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_anti_affinity.0.preferred_during_scheduling_ignored_during_execution.0.pod_affinity_term.0.label_selector.0.match_labels.%", "0"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_anti_affinity.0.preferred_during_scheduling_ignored_during_execution.0.pod_affinity_term.0.namespaces.#", "0"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_anti_affinity.0.preferred_during_scheduling_ignored_during_execution.0.pod_affinity_term.0.topology_key", "kubernetes.io/hostname"),
					resource.TestCheckResourceAttr("kubernetes_pod.test", "spec.0.affinity.0.pod_anti_affinity.0.preferred_during_scheduling_ignored_during_execution.0.weight", "100"),
				),
			},
		},
	})
}

func testAccKubernetesPodConfigWithNodeAffinityWithRequiredDuringSchedulingIgnoredDuringExecution(podName, imageName string) string {
	return fmt.Sprintf(`
	resource "kubernetes_pod" "test" {
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
								key = "kubernetes.io/hostname"
								operator = "NotIn"
								values = ["foo", "bar"]
							}
							match_expressions {
								key = "beta.kubernetes.io/os"
								operator = "In"
								values = ["linux"]
							}
						}
					}
				}
			}
			container {
				image = "%s"
				name  = "containername"
			}
		}
	}
		`, podName, imageName)
}

func testAccKubernetesPodConfigWithNodeAffinityWithPreferredDuringSchedulingIgnoredDuringExecution(podName, imageName string) string {
	return fmt.Sprintf(`
	resource "kubernetes_pod" "test" {
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
								key = "kubernetes.io/hostname"
								operator = "NotIn"
								values = ["foo", "bar"]
							}
							match_expressions {
								key = "beta.kubernetes.io/os"
								operator = "In"
								values = ["linux"]
							}
						}
					}
				}
			}
			container {
				image = "%s"
				name  = "containername"
			}
		}
	}
		`, podName, imageName)
}

func testAccKubernetesPodConfigWithPodAffinityWithRequiredDuringSchedulingIgnoredDuringExecution(podName, imageName string) string {
	return fmt.Sprintf(`
	resource "kubernetes_pod" "test" {
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
								key = "security"
								operator = "NotIn"
								values = ["foo", "bar"]
							}
						}
						topology_key = "kubernetes.io/hostname"
					}
				}
			}
			container {
				image = "%s"
				name  = "containername"
			}
		}
	}
		`, podName, imageName)
}

func testAccKubernetesPodConfigWithPodAffinityWithPreferredDuringSchedulingIgnoredDuringExecution(podName, imageName string) string {
	return fmt.Sprintf(`
	resource "kubernetes_pod" "test" {
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
									key = "security"
									operator = "NotIn"
									values = ["foo", "bar"]
								}
							}
              namespaces = ["default"]
							topology_key = "kubernetes.io/hostname"
						}
					}
				}
			}
			container {
				image = "%s"
				name  = "containername"
			}
		}
	}
		`, podName, imageName)
}

func testAccKubernetesPodConfigWithPodAntiAffinityWithRequiredDuringSchedulingIgnoredDuringExecution(podName, imageName string) string {
	return fmt.Sprintf(`
	resource "kubernetes_pod" "test" {
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
								key = "security"
								operator = "NotIn"
								values = ["foo", "bar"]
							}
						}
						topology_key = "kubernetes.io/hostname"
					}
				}
			}
			container {
				image = "%s"
				name  = "containername"
			}
		}
	}
		`, podName, imageName)
}

func testAccKubernetesPodConfigWithPodAntiAffinityWithPreferredDuringSchedulingIgnoredDuringExecution(podName, imageName string) string {
	return fmt.Sprintf(`
	resource "kubernetes_pod" "test" {
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
									key = "security"
									operator = "NotIn"
									values = ["foo", "bar"]
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
			}
		}
	}
		`, podName, imageName)
}
