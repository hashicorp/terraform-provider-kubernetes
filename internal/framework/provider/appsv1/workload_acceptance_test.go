// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1_test

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

var workloadAddresses = []string{"kubernetes_deployment_v1.test", "kubernetes_daemon_set_v1.test", "kubernetes_stateful_set_v1.test"}

func expectWorkloadActions(action plancheck.ResourceActionType) []plancheck.PlanCheck {
	checks := make([]plancheck.PlanCheck, len(workloadAddresses))
	for i, address := range workloadAddresses {
		checks[i] = plancheck.ExpectResourceAction(address, action)
	}
	return checks
}

// State written by 3.3.0 plans no change before its first refresh, while a
// template edit is still an in-place update.
func TestAccKubernetesWorkloadsV1_upgradeWithoutRefresh(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:             func() { testAccPreCheck(t) },
		AdditionalCLIOptions: &resource.AdditionalCLIOptions{Plan: resource.PlanOptions{NoRefresh: true}},
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccCheckKubernetesDeploymentV1Destroy,
			testAccCheckKubernetesDaemonSetV1Destroy,
			testAccCheckKubernetesStatefulSetV1Destroy,
		),
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.3.0"},
				},
				Config: testAccWorkloadsV1Config(name, "", ""),
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   testAccWorkloadsV1Config(name, "", ""),
				ConfigPlanChecks:         resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   testAccWorkloadsV1Config(name, "\n          working_dir = \"/tmp\"", ""),
				ConfigPlanChecks:         resource.ConfigPlanChecks{PreApply: expectWorkloadActions(plancheck.ResourceActionUpdate)},
			},
		},
	})
}

// kubectl rollout restart annotates the pod template. Every workload reports
// the annotation as drift, and ignore_changes on it keeps the plan empty.
func TestAccKubernetesWorkloadsV1_templateRestartAnnotation(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	const annotation = "kubectl.kubernetes.io/restartedAt"
	ignored := strings.ReplaceAll(testAccWorkloadsV1Config(name, "", ""), "  wait_for_rollout = false\n",
		fmt.Sprintf("  wait_for_rollout = false\n  lifecycle {\n    ignore_changes = [spec[0].template[0].metadata[0].annotations[%q]]\n  }\n", annotation))
	restart := func() {
		if err := testAccRestartWorkloads(name, annotation); err != nil {
			t.Fatal(err)
		}
	}
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccCheckKubernetesDeploymentV1Destroy,
			testAccCheckKubernetesDaemonSetV1Destroy,
			testAccCheckKubernetesStatefulSetV1Destroy,
		),
		Steps: []resource.TestStep{
			{Config: testAccWorkloadsV1Config(name, "", "")},
			{
				PreConfig:        restart,
				Config:           testAccWorkloadsV1Config(name, "", ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: expectWorkloadActions(plancheck.ResourceActionUpdate)},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(workloadAddresses[0], "spec.0.template.0.metadata.0.annotations."+annotation),
					resource.TestCheckNoResourceAttr(workloadAddresses[1], "spec.0.template.0.metadata.0.annotations."+annotation),
					resource.TestCheckNoResourceAttr(workloadAddresses[2], "spec.0.template.0.metadata.0.annotations."+annotation),
				),
			},
			{
				PreConfig:        restart,
				Config:           ignored,
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
			},
		},
	})
}

// testAccRestartWorkloads sets the template annotation kubectl rollout restart sets.
func testAccRestartWorkloads(name, annotation string) error {
	client, err := testAccWorkloadClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	patch := []byte(fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{%q:%q}}}}}`, annotation, time.Now().Format(time.RFC3339Nano)))
	apps := client.AppsV1()
	if _, err := apps.Deployments("default").Patch(ctx, name, k8stypes.StrategicMergePatchType, patch, metav1.PatchOptions{}); err != nil {
		return err
	}
	if _, err := apps.DaemonSets("default").Patch(ctx, name, k8stypes.StrategicMergePatchType, patch, metav1.PatchOptions{}); err != nil {
		return err
	}
	_, err = apps.StatefulSets("default").Patch(ctx, name, k8stypes.StrategicMergePatchType, patch, metav1.PatchOptions{})
	return err
}

// Container ports sharing a number with different protocols must keep the
// configured order through updates of each workload kind.
func TestAccKubernetesWorkloadsV1_containerPortsSharingNumber(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	steps := [][]string{
		{"53/TCP"},
		{"53/TCP", "8080/TCP", "53/UDP"},
		{"53/TCP", "8080/TCP"},
	}
	var testSteps []resource.TestStep
	for _, ports := range steps {
		var checks []resource.TestCheckFunc
		for _, resourceName := range []string{"kubernetes_deployment_v1.test", "kubernetes_daemon_set_v1.test", "kubernetes_stateful_set_v1.test"} {
			prefix := "spec.0.template.0.spec.0.container.0.port"
			checks = append(checks, resource.TestCheckResourceAttr(resourceName, prefix+".#", fmt.Sprint(len(ports))))
			for i, port := range ports {
				number, protocol, _ := strings.Cut(port, "/")
				checks = append(checks,
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.%d.container_port", prefix, i), number),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.%d.protocol", prefix, i), protocol),
				)
			}
		}
		testSteps = append(testSteps, resource.TestStep{
			Config: testAccWorkloadsV1ContainerPortsConfig(name, ports),
			Check:  resource.ComposeAggregateTestCheckFunc(checks...),
		})
	}
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccCheckKubernetesDeploymentV1Destroy,
			testAccCheckKubernetesDaemonSetV1Destroy,
			testAccCheckKubernetesStatefulSetV1Destroy,
		),
		Steps: testSteps,
	})
}

func testAccWorkloadsV1ContainerPortsConfig(name string, ports []string) string {
	var portBlocks strings.Builder
	for _, port := range ports {
		number, protocol, _ := strings.Cut(port, "/")
		fmt.Fprintf(&portBlocks, `
          port {
            container_port = %s
            protocol       = %q
          }`, number, protocol)
	}
	return testAccWorkloadsV1Config(name, portBlocks.String(), "")
}

// An http_get path left unset is defaulted to "/" by the API and must not
// produce an inconsistent result or a diff, including after it was set.
func TestAccKubernetesWorkloadsV1_httpGetDefaultPath(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	var testSteps []resource.TestStep
	for _, path := range []string{"", "/healthz", ""} {
		attribute := ""
		if path != "" {
			attribute = fmt.Sprintf("\n              path = %q", path)
		}
		handler := fmt.Sprintf(`
            http_get {
              port = 8080%s
            }`, attribute)
		var checks []resource.TestCheckFunc
		for _, resourceName := range []string{"kubernetes_deployment_v1.test", "kubernetes_daemon_set_v1.test", "kubernetes_stateful_set_v1.test"} {
			prefix := "spec.0.template.0.spec.0.container.0."
			checks = append(checks,
				resource.TestCheckResourceAttr(resourceName, prefix+"liveness_probe.0.http_get.0.path", path),
				resource.TestCheckResourceAttr(resourceName, prefix+"lifecycle.0.pre_stop.0.http_get.0.path", path),
			)
		}
		testSteps = append(testSteps, resource.TestStep{
			Config: testAccWorkloadsV1Config(name, fmt.Sprintf(`
          liveness_probe {%s
          }
          lifecycle {
            pre_stop {%s
            }
          }`, handler, handler), ""),
			Check: resource.ComposeAggregateTestCheckFunc(checks...),
		})
	}
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccCheckKubernetesDeploymentV1Destroy,
			testAccCheckKubernetesDaemonSetV1Destroy,
			testAccCheckKubernetesStatefulSetV1Destroy,
		),
		Steps: testSteps,
	})
}

// Adding a volume and an affinity whose nested blocks come from dynamic blocks
// over a value known only after apply updates each workload in place, as in
// SDKv2: neither unknown block is itself replace-on-change.
func TestAccKubernetesWorkloadsV1_addBlocksWithUnknownContent(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	var updates []plancheck.PlanCheck
	var checks []resource.TestCheckFunc
	for _, resourceName := range []string{"kubernetes_deployment_v1.test", "kubernetes_daemon_set_v1.test", "kubernetes_stateful_set_v1.test"} {
		updates = append(updates, plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate))
		checks = append(checks,
			resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.volume.0.config_map.0.items.0.key", "k1"),
			resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.affinity.0.node_affinity.0.required_during_scheduling_ignored_during_execution.0.node_selector_term.0.match_expressions.0.key", "example.com/k1"),
		)
	}
	volume := `
        volume {
          name = "config"
          config_map {
            name     = "absent"
            optional = true
            dynamic "items" {
              for_each = terraform_data.keys.output
              content {
                key  = items.value
                path = items.value
              }
            }
          }
        }
        affinity {
          node_affinity {
            required_during_scheduling_ignored_during_execution {
              dynamic "node_selector_term" {
                for_each = terraform_data.keys.output
                content {
                  match_expressions {
                    key      = "example.com/${node_selector_term.value}"
                    operator = "DoesNotExist"
                  }
                }
              }
            }
          }
        }`
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccCheckKubernetesDeploymentV1Destroy,
			testAccCheckKubernetesDaemonSetV1Destroy,
			testAccCheckKubernetesStatefulSetV1Destroy,
		),
		Steps: []resource.TestStep{
			{Config: testAccWorkloadsV1Config(name, "", "")},
			{
				Config:           "resource \"terraform_data\" \"keys\" {\n  input = [\"k1\"]\n}\n" + testAccWorkloadsV1Config(name, "", volume),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: updates},
				Check:            resource.ComposeAggregateTestCheckFunc(checks...),
			},
		},
	})
}

// An empty list does not omit a list-of-object argument such as strategy, so
// it is rejected rather than planned and then contradicted by the API.
func TestAccKubernetesWorkloadsV1_emptyStrategyRejected(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{{
			Config:      strings.Replace(testAccWorkloadsV1Config(name, "", ""), "replicas = 1", "replicas = 1\n    strategy = []", 1),
			PlanOnly:    true,
			ExpectError: regexp.MustCompile(`must\s+be\s+omitted\s+or\s+null`),
		}},
	})
}

// Kubernetes stores replicas, max_surge and max_unavailable as numbers. A
// configured "01" is kept as spelled rather than read back as "1".
func TestAccKubernetesWorkloadsV1_numberSpelling(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	strategy := `strategy = [{ rolling_update = [{ max_surge = "01", max_unavailable = "00" }] }]`
	config := testAccWorkloadsV1Config(name, "", "")
	// The Deployment, then the StatefulSet, sets replicas; the DaemonSet has
	// no other spec arguments.
	config = strings.Replace(config, "replicas = 1", "replicas = \"01\"\n    "+strategy, 1)
	config = strings.Replace(config, "replicas = 1", "replicas = \"01\"", 1)
	config = strings.Replace(config, "spec {\n    \n", "spec {\n    "+strategy+"\n", 1)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccCheckKubernetesDeploymentV1Destroy,
			testAccCheckKubernetesDaemonSetV1Destroy,
			testAccCheckKubernetesStatefulSetV1Destroy,
		),
		Steps: []resource.TestStep{{
			Config: config,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("kubernetes_deployment_v1.test", "spec.0.replicas", "01"),
				resource.TestCheckResourceAttr("kubernetes_deployment_v1.test", "spec.0.strategy.0.rolling_update.0.max_surge", "01"),
				resource.TestCheckResourceAttr("kubernetes_deployment_v1.test", "spec.0.strategy.0.rolling_update.0.max_unavailable", "00"),
				resource.TestCheckResourceAttr("kubernetes_daemon_set_v1.test", "spec.0.strategy.0.rolling_update.0.max_surge", "01"),
				resource.TestCheckResourceAttr("kubernetes_stateful_set_v1.test", "spec.0.replicas", "01"),
			),
		}},
	})
}

func testAccWorkloadsV1Config(name, container, podSpec string) string {
	var config strings.Builder
	for _, workload := range []struct{ resourceType, app, extraSpec string }{
		{"kubernetes_deployment_v1", name + "-deploy", "replicas = 1"},
		{"kubernetes_daemon_set_v1", name + "-ds", ""},
		{"kubernetes_stateful_set_v1", name + "-sts", fmt.Sprintf("replicas = 1\n    service_name = %q", name)},
	} {
		fmt.Fprintf(&config, `resource %q "test" {
  metadata {
    name = %q
  }
  spec {
    %s
    selector {
      match_labels = {
        app = %q
      }
    }
    template {
      metadata {
        labels = {
          app = %q
        }
      }
      spec {
        termination_grace_period_seconds = 1
        container {
          name    = "main"
          image   = %q
          command = ["sleep", "300"]%s
        }%s
      }
    }
  }
  wait_for_rollout = false
}
`, workload.resourceType, name, workload.extraSpec, workload.app, workload.app, busyboxImage, container, podSpec)
	}
	return config.String()
}
