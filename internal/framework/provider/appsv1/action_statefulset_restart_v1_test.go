// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	gversion "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesStatefulSetRestartAction_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(gversion.Must(gversion.NewVersion("1.14.0"))),
		},
		CheckDestroy: testAccCheckKubernetesStatefulSetDestroy(name),
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetRestartActionConfig(name),
				Check:  testAccCheckKubernetesStatefulSetWasRestarted(name),
				// The restart action patches the pod template annotations
				// directly via the Kubernetes API, outside of what
				// kubernetes_stateful_set_v1 itself manages, so the next
				// refresh legitimately detects drift.
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccKubernetesStatefulSetRestartActionConfig(name string) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set_v1" "test" {
  metadata {
    name = %[1]q
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
          image   = "busybox:1.36"
          command = ["sleep", "300"]
        }
        termination_grace_period_seconds = 1
      }
    }
  }

  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.kubernetes_statefulset_restart.test]
    }
  }
}

action "kubernetes_statefulset_restart" "test" {
  config {
    namespace = kubernetes_stateful_set_v1.test.metadata[0].namespace
    name      = kubernetes_stateful_set_v1.test.metadata[0].name
    timeout   = "2m"
  }
}
`, name)
}

// testAccCheckKubernetesStatefulSetWasRestarted asserts, via a direct
// Kubernetes API call, that the statefulset's pod template carries a
// kubectl.kubernetes.io/restartedAt annotation set by the action, proving
// the action_trigger actually invoked kubernetes_statefulset_restart.
func testAccCheckKubernetesStatefulSetWasRestarted(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		clientset, err := testAccAppsV1Clientset()
		if err != nil {
			return err
		}
		conn, err := clientset.MainClientset()
		if err != nil {
			return err
		}

		statefulSet, err := conn.AppsV1().StatefulSets("default").Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("error getting statefulset %s: %s", name, err)
		}

		restartedAt := statefulSet.Spec.Template.ObjectMeta.Annotations["kubectl.kubernetes.io/restartedAt"]
		if restartedAt == "" {
			return fmt.Errorf("expected statefulset %s to have a kubectl.kubernetes.io/restartedAt annotation set by the restart action", name)
		}
		if _, err := time.Parse(time.RFC3339, restartedAt); err != nil {
			return fmt.Errorf("restartedAt annotation %q is not a valid RFC3339 timestamp: %s", restartedAt, err)
		}

		return nil
	}
}

func testAccCheckKubernetesStatefulSetDestroy(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		clientset, err := testAccAppsV1Clientset()
		if err != nil {
			return err
		}
		conn, err := clientset.MainClientset()
		if err != nil {
			return err
		}

		_, err = conn.AppsV1().StatefulSets("default").Get(context.Background(), name, metav1.GetOptions{})
		if err == nil {
			return fmt.Errorf("statefulset %s still exists after destroy", name)
		}

		return nil
	}
}
