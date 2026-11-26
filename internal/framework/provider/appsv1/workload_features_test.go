// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/kubetest"
	appsv1 "k8s.io/api/apps/v1"
)

// The claims and the class name objects that do not exist, so the Pods stay pending.
func TestAccKubernetesDeploymentV1_podResourcesAndClaims(t *testing.T) {
	var conf appsv1.Deployment
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_deployment_v1.test"
	config := testAccWorkloadWith(testAccKubernetesDeploymentV1Config_minimal(name, kubetest.BusyboxImage), `
        resources       = { limits = { cpu = "1", memory = "256Mi" } }
        resource_claims = [{ name = "device", resource_claim_template_name = "tf-acc-missing" }]
        volume {
          name = "scratch"
          ephemeral {
            volume_claim_template {
              spec {
                access_modes                 = ["ReadWriteOnce"]
                volume_attributes_class_name = "tf-acc-missing"
                resources {
                  requests = {
                    storage = "16Mi"
                  }
                }
              }
            }
          }
        }`, `
          resources = { claims = [{ name = "device" }] }`)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t); kubetest.SkipIfClusterVersionLessThan(t, "1.34.0") },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.resources.limits.memory", "256Mi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.resource_claims.0.resource_claim_template_name", "tf-acc-missing"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.resources.claims.0.name", "device"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.volume.0.ephemeral.0.volume_claim_template.0.spec.0.volume_attributes_class_name", "tf-acc-missing"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "wait_for_rollout"},
			},
		},
	})
}

func TestAccKubernetesDeploymentV1_hostnameOverride(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccWorkloadWith(testAccKubernetesDeploymentV1Config_minimal(name, kubetest.BusyboxImage), `
        hostname_override = "tf-acc-host"`, "")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t) },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesDeploymentV1Destroy,
		Steps: kubetest.FeatureSteps("1.35.0", "HostnameOverride", config,
			resource.TestCheckResourceAttr("kubernetes_deployment_v1.test", "spec.0.template.0.spec.0.hostname_override", "tf-acc-host")),
	})
}

// The signer does not exist, so the Pods stay pending.
func TestAccKubernetesDeploymentV1_projectedCertificates(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_deployment_v1.test"
	config := testAccWorkloadWith(testAccKubernetesDeploymentV1Config_minimal(name, kubetest.BusyboxImage), `
        volume {
          name = "certificates"
          projected {
            sources {
              cluster_trust_bundle = { signer_name = "example.com/tf-acc", label_selector = {}, path = "ca.pem", optional = true }
            }
            sources {
              pod_certificate = { signer_name = "example.com/tf-acc", key_type = "ED25519", credential_bundle_path = "credentials.pem" }
            }
          }
        }`, "")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t) },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesDeploymentV1Destroy,
		Steps: kubetest.FeatureSteps("1.37.0", "ClusterTrustBundleProjection", config, resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.volume.0.projected.0.sources.0.cluster_trust_bundle.path", "ca.pem"),
			resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.volume.0.projected.0.sources.1.pod_certificate.key_type", "ED25519"),
		)),
	})
}

func TestAccKubernetesStatefulSetV1_maxUnavailable(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := strings.Replace(testAccKubernetesStatefulSetV1ConfigMinimal(name, kubetest.BusyboxImage), "  spec {", `  spec {
    update_strategy {
      type = "RollingUpdate"
      rolling_update {
        max_unavailable = "2"
      }
    }`, 1)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t) },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesStatefulSetV1Destroy,
		Steps: kubetest.FeatureSteps("1.37.0", "MaxUnavailableStatefulSet", config,
			resource.TestCheckResourceAttr("kubernetes_stateful_set_v1.test", "spec.0.update_strategy.0.rolling_update.0.max_unavailable", "2")),
	})
}

// The class does not exist, so the claim and the Pod stay pending.
func TestAccKubernetesStatefulSetV1_volumeAttributesClass(t *testing.T) {
	var conf appsv1.StatefulSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_stateful_set_v1.test"
	config := strings.Replace(testAccKubernetesStatefulSetV1ConfigMinimal(name, kubetest.BusyboxImage), "  spec {", `  wait_for_rollout = false
  spec {
    persistent_volume_claim_retention_policy = {
      when_deleted = "Delete"
    }
    volume_claim_template {
      metadata {
        name = "data"
      }
      spec {
        access_modes                 = ["ReadWriteOnce"]
        volume_attributes_class_name = "tf-acc-missing"
        resources {
          requests = {
            storage = "16Mi"
          }
        }
      }
    }`, 1)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t); kubetest.SkipIfClusterVersionLessThan(t, "1.34.0") },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesStatefulSetV1Destroy,
		Steps: []resource.TestStep{{
			Config: config,
			Check: resource.ComposeAggregateTestCheckFunc(
				testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
				resource.TestCheckResourceAttr(resourceName, "spec.0.volume_claim_template.0.spec.0.volume_attributes_class_name", "tf-acc-missing"),
			),
		}},
	})
}

// testAccWorkloadWith adds arguments to the Pod spec and to the first
// container of a minimal workload configuration, and does not wait for Pods.
func testAccWorkloadWith(config, podSpec, container string) string {
	config = strings.Replace(config, "  metadata {", "  wait_for_rollout = false\n  metadata {", 1)
	config = strings.Replace(config, "        container {", strings.TrimPrefix(podSpec, "\n")+"\n        container {", 1)
	return strings.Replace(config, "          command = ", strings.TrimPrefix(container, "\n")+"\n          command = ", 1)
}
