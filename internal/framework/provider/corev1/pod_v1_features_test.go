// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/kubetest"
	api "k8s.io/api/core/v1"
)

func TestAccKubernetesPodV1_podResourcesAndResizePolicy(t *testing.T) {
	var conf api.Pod
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_pod_v1.test"
	config := testAccPodV1With(name, `
    resources = { limits = { cpu = "500m", memory = "128Mi" } }`, `
      resize_policy = [{ resource_name = "cpu", restart_policy = "NotRequired" }]`)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t); kubetest.SkipIfClusterVersionLessThan(t, "1.34.0") },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.limits.cpu", "500m"),
					// Kubernetes derives the requests of a Pod from its limits.
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.requests.memory", "128Mi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.resize_policy.0.restart_policy", "NotRequired"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
		},
	})
}

func TestAccKubernetesPodV1_hostnameOverride(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccPodV1With(name, `
    hostname_override = "tf-acc-host"`, "")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t) },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesPodV1Destroy,
		Steps: kubetest.FeatureSteps("1.35.0", "HostnameOverride", config,
			resource.TestCheckResourceAttr("kubernetes_pod_v1.test", "spec.0.hostname_override", "tf-acc-host")),
	})
}

// Kubernetes 1.34 drops the container restart policy without the feature;
// earlier versions reject it on a regular container.
func TestAccKubernetesPodV1_containerRestartRules(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_pod_v1.test"
	config := testAccPodV1With(name, "", `
      restart_policy       = "Never"
      restart_policy_rules = [{ action = "Restart", exit_codes = { operator = "In", values = [42] } }]`)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t) },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesPodV1Destroy,
		Steps: kubetest.FeatureSteps("1.35.0", "ContainerRestartRules|may not be set for non-init containers", config, resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.restart_policy", "Never"),
			resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.restart_policy_rules.0.exit_codes.values.0", "42"),
		)),
	})
}

// A cluster without the EnvFiles feature rejects the variable as having no source.
func TestAccKubernetesPodV1_envFileKeyRef(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccPodV1With(name, `
    init_container {
      name    = "write-env"
      image   = "`+kubetest.BusyboxImage+`"
      command = ["sh", "-c", "echo \"GREETING='hello'\" > /data/config.env"]
      volume_mount {
        name       = "data"
        mount_path = "/data"
      }
    }
    volume {
      name = "data"
      empty_dir {}
    }`, `
      env {
        name = "GREETING"
        value_from {
          file_key_ref = { volume_name = "data", path = "config.env", key = "GREETING" }
        }
      }`)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t) },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesPodV1Destroy,
		Steps: kubetest.FeatureSteps("1.35.0", "must specify one of", config,
			resource.TestCheckResourceAttr("kubernetes_pod_v1.test", "spec.0.container.0.env.0.value_from.0.file_key_ref.key", "GREETING")),
	})
}

// testAccPodV1With is a minimal Pod with extra Pod spec and container arguments.
func testAccPodV1With(name, podSpec, container string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
  metadata {
    name = %q
  }
  spec {
    termination_grace_period_seconds = 1
%s
    container {
      name    = "containername"
      image   = %q
      command = ["sleep", "300"]
%s
    }
  }
}
`, name, strings.TrimPrefix(podSpec, "\n"), kubetest.BusyboxImage, strings.TrimPrefix(container, "\n"))
}
