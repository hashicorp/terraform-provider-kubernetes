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

func TestAccKubernetesDaemonSetRestartAction_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(gversion.Must(gversion.NewVersion("1.14.0"))),
		},
		CheckDestroy: testAccCheckKubernetesDaemonSetDestroy(name),
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDaemonSetRestartActionConfig(name),
				Check:  testAccCheckKubernetesDaemonSetWasRestarted(name),
				// The restart action patches the pod template annotations
				// directly via the Kubernetes API, outside of what
				// kubernetes_daemon_set_v1 itself manages, so the next
				// refresh legitimately detects drift.
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccKubernetesDaemonSetRestartActionConfig(name string) string {
	return fmt.Sprintf(`resource "kubernetes_daemon_set_v1" "test" {
  metadata {
    name = %[1]q
  }

  spec {
    selector {
      match_labels = {
        foo = "bar"
      }
    }

    template {
      metadata {
        labels = {
          foo = "bar"
        }
      }

      spec {
        container {
          image   = "busybox:1.36"
          name    = "tf-acc-test"
          command = ["sleep", "300"]
        }
        termination_grace_period_seconds = 1
      }
    }
  }

  lifecycle {
    action_trigger {
      events  = [after_create]
      actions = [action.kubernetes_daemonset_restart.test]
    }
  }
}

action "kubernetes_daemonset_restart" "test" {
  config {
    namespace = kubernetes_daemon_set_v1.test.metadata[0].namespace
    name      = kubernetes_daemon_set_v1.test.metadata[0].name
    timeout   = "2m"
  }
}
`, name)
}

// testAccCheckKubernetesDaemonSetWasRestarted asserts, via a direct
// Kubernetes API call, that the daemonset's pod template carries a
// kubectl.kubernetes.io/restartedAt annotation set by the action, proving
// the action_trigger actually invoked kubernetes_daemonset_restart.
func testAccCheckKubernetesDaemonSetWasRestarted(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		clientset, err := testAccAppsV1Clientset()
		if err != nil {
			return err
		}
		conn, err := clientset.MainClientset()
		if err != nil {
			return err
		}

		daemonSet, err := conn.AppsV1().DaemonSets("default").Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("error getting daemonset %s: %s", name, err)
		}

		restartedAt := daemonSet.Spec.Template.ObjectMeta.Annotations["kubectl.kubernetes.io/restartedAt"]
		if restartedAt == "" {
			return fmt.Errorf("expected daemonset %s to have a kubectl.kubernetes.io/restartedAt annotation set by the restart action", name)
		}
		if _, err := time.Parse(time.RFC3339, restartedAt); err != nil {
			return fmt.Errorf("restartedAt annotation %q is not a valid RFC3339 timestamp: %s", restartedAt, err)
		}

		return nil
	}
}

func testAccCheckKubernetesDaemonSetDestroy(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		clientset, err := testAccAppsV1Clientset()
		if err != nil {
			return err
		}
		conn, err := clientset.MainClientset()
		if err != nil {
			return err
		}

		_, err = conn.AppsV1().DaemonSets("default").Get(context.Background(), name, metav1.GetOptions{})
		if err == nil {
			return fmt.Errorf("daemonset %s still exists after destroy", name)
		}

		return nil
	}
}
