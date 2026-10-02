// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1_test

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Container ports are merged by containerPort and topology spread constraints
// by topologyKey in a strategic merge patch, while the API identifies them by
// containerPort+protocol and topologyKey+whenUnsatisfiable. These tests apply
// updates that only differ in the wider key and check that the live Pod
// template holds exactly the configured lists after each step.

type listMergeAccPort struct {
	name     string
	number   int32
	protocol corev1.Protocol
}

type listMergeAccConstraint struct {
	when corev1.UnsatisfiableConstraintAction
	skew int32
}

type listMergeAccStep struct {
	ports       []listMergeAccPort
	constraints []listMergeAccConstraint
}

var listMergeAccSteps = []listMergeAccStep{
	{
		ports:       []listMergeAccPort{{"dns-tcp", 53, corev1.ProtocolTCP}},
		constraints: []listMergeAccConstraint{{corev1.DoNotSchedule, 1}},
	},
	{
		ports:       []listMergeAccPort{{"dns-tcp", 53, corev1.ProtocolTCP}, {"dns-udp", 53, corev1.ProtocolUDP}},
		constraints: []listMergeAccConstraint{{corev1.DoNotSchedule, 1}, {corev1.ScheduleAnyway, 2}},
	},
	{
		ports:       []listMergeAccPort{{"dns-udp", 53, corev1.ProtocolUDP}},
		constraints: []listMergeAccConstraint{{corev1.DoNotSchedule, 1}, {corev1.ScheduleAnyway, 3}},
	},
	{
		ports:       []listMergeAccPort{{"dns-udp", 53, corev1.ProtocolUDP}, {"dns-tcp", 53, corev1.ProtocolTCP}},
		constraints: []listMergeAccConstraint{{corev1.ScheduleAnyway, 3}},
	},
}

func TestAccKubernetesDeploymentV1_ambiguousMergeKeyLists(t *testing.T) {
	testAccWorkloadAmbiguousMergeKeyLists(t, "kubernetes_deployment_v1", testAccCheckKubernetesDeploymentV1Destroy, func(ctx context.Context, namespace, name string) (*corev1.PodSpec, error) {
		conn, err := testAccWorkloadClient()
		if err != nil {
			return nil, err
		}
		out, err := conn.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return &out.Spec.Template.Spec, nil
	})
}

func TestAccKubernetesDaemonSetV1_ambiguousMergeKeyLists(t *testing.T) {
	testAccWorkloadAmbiguousMergeKeyLists(t, "kubernetes_daemon_set_v1", testAccCheckKubernetesDaemonSetV1Destroy, func(ctx context.Context, namespace, name string) (*corev1.PodSpec, error) {
		conn, err := testAccWorkloadClient()
		if err != nil {
			return nil, err
		}
		out, err := conn.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return &out.Spec.Template.Spec, nil
	})
}

func TestAccKubernetesStatefulSetV1_ambiguousMergeKeyLists(t *testing.T) {
	testAccWorkloadAmbiguousMergeKeyLists(t, "kubernetes_stateful_set_v1", testAccCheckKubernetesStatefulSetV1Destroy, func(ctx context.Context, namespace, name string) (*corev1.PodSpec, error) {
		conn, err := testAccWorkloadClient()
		if err != nil {
			return nil, err
		}
		out, err := conn.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return &out.Spec.Template.Spec, nil
	})
}

func testAccWorkloadAmbiguousMergeKeyLists(t *testing.T, resourceType string, destroy resource.TestCheckFunc, livePodSpec func(context.Context, string, string) (*corev1.PodSpec, error)) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := resourceType + ".test"
	steps := make([]resource.TestStep, 0, len(listMergeAccSteps))
	for _, step := range listMergeAccSteps {
		steps = append(steps, resource.TestStep{
			Config: testAccWorkloadAmbiguousMergeKeyListsConfig(resourceType, name, step),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.port.#", fmt.Sprint(len(step.ports))),
				resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.topology_spread_constraint.#", fmt.Sprint(len(step.constraints))),
				testAccCheckWorkloadLiveLists(resourceName, step, livePodSpec),
			),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
			},
		})
	}
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             destroy,
		Steps:                    steps,
	})
}

func testAccCheckWorkloadLiveLists(resourceName string, step listMergeAccStep, livePodSpec func(context.Context, string, string) (*corev1.PodSpec, error)) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}
		namespace, name, err := IdParts(rs.Primary.ID)
		if err != nil {
			return err
		}
		spec, err := livePodSpec(context.Background(), namespace, name)
		if err != nil {
			return err
		}
		var ports []listMergeAccPort
		for _, p := range spec.Containers[0].Ports {
			ports = append(ports, listMergeAccPort{p.Name, p.ContainerPort, p.Protocol})
		}
		if !reflect.DeepEqual(ports, step.ports) {
			return fmt.Errorf("live ports = %v, want %v", ports, step.ports)
		}
		var constraints []listMergeAccConstraint
		for _, c := range spec.TopologySpreadConstraints {
			constraints = append(constraints, listMergeAccConstraint{c.WhenUnsatisfiable, c.MaxSkew})
		}
		if !reflect.DeepEqual(constraints, step.constraints) {
			return fmt.Errorf("live topology spread constraints = %v, want %v", constraints, step.constraints)
		}
		return nil
	}
}

func testAccWorkloadAmbiguousMergeKeyListsConfig(resourceType, name string, step listMergeAccStep) string {
	var extraSpec string
	switch resourceType {
	case "kubernetes_deployment_v1":
		extraSpec = `replicas = 1`
	case "kubernetes_stateful_set_v1":
		extraSpec = `replicas     = 1
    service_name = "` + name + `"`
	}
	var ports, constraints strings.Builder
	for _, p := range step.ports {
		fmt.Fprintf(&ports, `
          port {
            name           = %q
            container_port = %d
            protocol       = %q
          }`, p.name, p.number, p.protocol)
	}
	for _, c := range step.constraints {
		fmt.Fprintf(&constraints, `
        topology_spread_constraint {
          max_skew           = %d
          topology_key       = "kubernetes.io/hostname"
          when_unsatisfiable = %q
          label_selector {
            match_labels = {
              app = %q
            }
          }
        }`, c.skew, c.when, name)
	}
	return fmt.Sprintf(`resource %q "test" {
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
`, resourceType, name, extraSpec, name, name, busyboxImage, ports.String(), constraints.String())
}
