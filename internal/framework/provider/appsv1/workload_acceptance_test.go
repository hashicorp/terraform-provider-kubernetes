// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
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
				Config: testAccWorkloadsV1Config(name, ""),
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   testAccWorkloadsV1Config(name, ""),
				ConfigPlanChecks:         resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   testAccWorkloadsV1Config(name, "\n          working_dir = \"/tmp\""),
				ConfigPlanChecks:         resource.ConfigPlanChecks{PreApply: expectWorkloadActions(plancheck.ResourceActionUpdate)},
			},
		},
	})
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
	return testAccWorkloadsV1Config(name, portBlocks.String())
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
          }`, handler, handler)),
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

func testAccWorkloadsV1Config(name, container string) string {
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
        }
      }
    }
  }
  wait_for_rollout = false
}
`, workload.resourceType, name, workload.extraSpec, workload.app, workload.app, busyboxImage, container)
	}
	return config.String()
}
