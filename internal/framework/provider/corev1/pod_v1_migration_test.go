// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	corev1api "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const podV1ResourceName = "kubernetes_pod_v1.test"

func sdkv2PodExternalProvider() map[string]resource.ExternalProvider {
	return map[string]resource.ExternalProvider{
		"kubernetes": {
			VersionConstraint: sdkv2ProviderVersion,
			Source:            "hashicorp/kubernetes",
		},
	}
}

func TestAccKubernetesPodV1_UpgradeFromSDKV2_basic(t *testing.T) {
	var before, after corev1api.Pod
	name := acctest.RandomWithPrefix("tf-migration-pod")
	config := testAccKubernetesPodV1ConfigMinimal(name, busyboxImage)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPodV1PreCheck(t) },
		CheckDestroy: testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: sdkv2PodExternalProvider(),
				Config:            config,
				Check:             testAccCheckKubernetesPodV1Exists(podV1ResourceName, &before),
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(podV1ResourceName, &after),
					testAccCheckKubernetesPodForceNew(&before, &after, false),
				),
			},
		},
	})
}

func TestAccKubernetesPodV1_generatedNameLifecycle(t *testing.T) {
	var before, afterUpdate, afterNoop corev1api.Pod
	prefix := fmt.Sprintf("tf-migration-pod-%s-", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	initialConfig := testAccKubernetesPodV1ConfigMinimalGeneratedName(prefix, busyboxImage)
	updatedConfig := testAccKubernetesPodV1ConfigMinimalGeneratedNameWithLabel(prefix, busyboxImage, "updated")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPodV1PreCheck(t) },
		CheckDestroy: testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   initialConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(podV1ResourceName, &before),
					resource.TestCheckResourceAttr(podV1ResourceName, "metadata.0.generate_name", prefix),
					resource.TestCheckResourceAttrSet(podV1ResourceName, "metadata.0.name"),
					resource.TestCheckResourceAttrSet(podV1ResourceName, "metadata.0.uid"),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   updatedConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(podV1ResourceName, plancheck.ResourceActionUpdate)},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(podV1ResourceName, &afterUpdate),
					testAccCheckKubernetesPodForceNew(&before, &afterUpdate, false),
					testAccCheckKubernetesPodGeneratedNamePreserved(&before, &afterUpdate, prefix),
					testAccCheckKubernetesPodSpecPreserved(&before, &afterUpdate),
					resource.TestCheckResourceAttr(podV1ResourceName, "metadata.0.generate_name", prefix),
					resource.TestCheckResourceAttr(podV1ResourceName, "metadata.0.labels.managed", "updated"),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				ResourceName:             podV1ResourceName,
				ImportState:              true,
				ImportStateVerify:        true,
				ImportStateVerifyIgnore:  []string{"metadata.0.resource_version"},
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   updatedConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(podV1ResourceName, &afterNoop),
					testAccCheckKubernetesPodForceNew(&afterUpdate, &afterNoop, false),
					testAccCheckKubernetesPodGeneratedNamePreserved(&afterUpdate, &afterNoop, prefix),
					testAccCheckKubernetesPodSpecPreserved(&afterUpdate, &afterNoop),
					resource.TestCheckResourceAttr(podV1ResourceName, "metadata.0.generate_name", prefix),
					resource.TestCheckResourceAttr(podV1ResourceName, "metadata.0.labels.managed", "updated"),
				),
			},
		},
	})
}

func TestAccKubernetesPodV1_UpgradeFromSDKV2_legacyResourcesSyntaxToAssignment(t *testing.T) {
	var before, after corev1api.Pod
	name := acctest.RandomWithPrefix("tf-migration-pod")
	legacyConfig := testAccKubernetesPodV1ConfigWithResourceRequirementsLegacy(name, busyboxImage)
	assignmentConfig := testAccKubernetesPodV1ConfigWithResourceRequirements(name, busyboxImage)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPodV1PreCheck(t) },
		CheckDestroy: testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: sdkv2PodExternalProvider(),
				Config:            legacyConfig,
				Check:             testAccCheckKubernetesPodV1Exists(podV1ResourceName, &before),
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   assignmentConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(podV1ResourceName, &after),
					testAccCheckKubernetesPodForceNew(&before, &after, false),
				),
			},
		},
	})
}

func TestAccKubernetesPodV1_UpgradeFromSDKV2_legacyPodSpecCollectionsSyntaxToAssignment(t *testing.T) {
	var before, after corev1api.Pod
	podName := acctest.RandomWithPrefix("tf-migration-pod")
	secretName := acctest.RandomWithPrefix("tf-migration-pull-secret")
	legacyConfig := testAccKubernetesPodV1ConfigWithPodSpecCollectionsLegacy(podName, secretName, busyboxImage)
	assignmentConfig := testAccKubernetesPodV1ConfigWithPodSpecCollectionsAssignment(podName, secretName, busyboxImage)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPodV1PreCheck(t) },
		CheckDestroy: testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: sdkv2PodExternalProvider(),
				Config:            legacyConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(podV1ResourceName, &before),
					resource.TestCheckResourceAttr(podV1ResourceName, "spec.0.image_pull_secrets.0.name", secretName),
					resource.TestCheckResourceAttr(podV1ResourceName, "spec.0.readiness_gate.0.condition_type", "example.com/Ready"),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   assignmentConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(podV1ResourceName, &after),
					testAccCheckKubernetesPodForceNew(&before, &after, false),
					resource.TestCheckResourceAttr(podV1ResourceName, "spec.0.image_pull_secrets.0.name", secretName),
					resource.TestCheckResourceAttr(podV1ResourceName, "spec.0.readiness_gate.0.condition_type", "example.com/Ready"),
				),
			},
		},
	})
}

func TestAccKubernetesPodV1_UpgradeFromSDKV2_ignoredMetadataAndImageUpdateReplacesPod(t *testing.T) {
	var before, afterIgnore, afterUpdate corev1api.Pod
	name := acctest.RandomWithPrefix("tf-migration-pod")
	ignoredKey := "cloud.google.com/neg"
	initialConfig := testAccKubernetesPodV1ConfigMinimal(name, busyboxImage)
	ignoredConfig := testAccKubernetesConfig_ignoreAnnotations() + testAccKubernetesPodV1ConfigMinimal(name, busyboxImage)
	updatedConfig := testAccKubernetesConfig_ignoreAnnotations() + testAccKubernetesPodV1ConfigMinimal(name, agnhostImage)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPodV1PreCheck(t) },
		CheckDestroy: testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: sdkv2PodExternalProvider(),
				Config:            initialConfig,
				Check:             testAccCheckKubernetesPodV1Exists(podV1ResourceName, &before),
			},
			{
				PreConfig:                func() { testAccInjectIgnoredPodAnnotation(t, name, ignoredKey, "enabled") },
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   ignoredConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(podV1ResourceName, &afterIgnore),
					testAccCheckKubernetesPodForceNew(&before, &afterIgnore, false),
					testAccCheckIgnoredPodAnnotation(name, ignoredKey),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   updatedConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(podV1ResourceName, plancheck.ResourceActionDestroyBeforeCreate)},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(podV1ResourceName, &afterUpdate),
					testAccCheckKubernetesPodForceNew(&afterIgnore, &afterUpdate, true),
				),
			},
		},
	})
}

func TestAccKubernetesPodV1_UpgradeFromSDKV2_disappears(t *testing.T) {
	var before, after corev1api.Pod
	name := acctest.RandomWithPrefix("tf-migration-pod")
	config := testAccKubernetesPodV1ConfigMinimal(name, busyboxImage)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPodV1PreCheck(t) },
		CheckDestroy: testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: sdkv2PodExternalProvider(),
				Config:            config,
				Check:             testAccCheckKubernetesPodV1Exists(podV1ResourceName, &before),
			},
			{
				PreConfig:                func() { testAccDeletePodByName(t, "default", name) },
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(podV1ResourceName, plancheck.ResourceActionCreate)},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(podV1ResourceName, &after),
					testAccCheckKubernetesPodForceNew(&before, &after, true),
				),
			},
		},
	})
}

func TestAccKubernetesPodV1_MoveStateFromUnversioned(t *testing.T) {
	var before, after corev1api.Pod
	name := acctest.RandomWithPrefix("tf-migration-pod")
	config := testAccKubernetesPodV1ConfigMinimal(name, busyboxImage)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccPodV1PreCheck(t) },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		CheckDestroy: testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: sdkv2PodExternalProvider(),
				Config:            testAccPodMoveStateSource(config),
				Check:             testAccCheckKubernetesPodV1Exists("kubernetes_pod.test", &before),
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   testAccPodMoveStateTarget(config),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(podV1ResourceName, &after),
					testAccCheckKubernetesPodForceNew(&before, &after, false),
				),
			},
		},
	})
}

func testAccPodMoveStateSource(config string) string {
	return strings.ReplaceAll(config, `"kubernetes_pod_v1"`, `"kubernetes_pod"`)
}

func testAccPodMoveStateTarget(config string) string {
	return config + `
moved {
  from = kubernetes_pod.test
  to   = kubernetes_pod_v1.test
}
`
}

func testAccKubernetesPodV1ConfigMinimalGeneratedName(prefix, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
  metadata {
    generate_name = %q
  }

  spec {
    container {
      image = %q
      name  = "containername"
    }
  }
}
`, prefix, imageName)
}

func testAccKubernetesPodV1ConfigMinimalGeneratedNameWithLabel(prefix, imageName, label string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
  metadata {
    generate_name = %q
    labels = {
      managed = %q
    }
  }

  spec {
    container {
      image = %q
      name  = "containername"
    }
  }
}
`, prefix, label, imageName)
}

func testAccKubernetesPodV1ConfigWithResourceRequirementsLegacy(podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }

    name = %q
  }
  spec {
    container {
      image = %q
      name  = "containername"
      resources {
        limits = {
          cpu                 = "0.5"
          memory              = "512Mi"
          "ephemeral-storage" = "512Mi"
        }

        requests = {
          cpu                 = "250m"
          memory              = "50Mi"
          "ephemeral-storage" = "128Mi"
        }
      }
    }
  }
}
`, podName, imageName)
}

func testAccKubernetesPodV1ConfigWithPodSpecCollectionsLegacy(podName, secretName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "pull" {
  metadata {
    name = %q
  }
  type = "kubernetes.io/dockerconfigjson"
  data = {
    ".dockerconfigjson" = "{}"
  }
}

resource "kubernetes_pod_v1" "test" {
  metadata {
    name = %q
  }
  spec {
    image_pull_secrets {
      name = kubernetes_secret_v1.pull.metadata[0].name
    }
    readiness_gate {
      condition_type = "example.com/Ready"
    }
    container {
      image = %q
      name  = "containername"
    }
  }
}
`, secretName, podName, imageName)
}

func testAccKubernetesPodV1ConfigWithPodSpecCollectionsAssignment(podName, secretName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "pull" {
  metadata {
    name = %q
  }
  type = "kubernetes.io/dockerconfigjson"
  data = {
    ".dockerconfigjson" = "{}"
  }
}

resource "kubernetes_pod_v1" "test" {
  metadata {
    name = %q
  }
  spec {
    image_pull_secrets = [{
      name = kubernetes_secret_v1.pull.metadata[0].name
    }]
    readiness_gate = [{
      condition_type = "example.com/Ready"
    }]
    container {
      image = %q
      name  = "containername"
    }
  }
}
`, secretName, podName, imageName)
}

func testAccInjectIgnoredPodAnnotation(t *testing.T, name, key, value string) {
	t.Helper()
	conn, err := testAccPodV1MainClientset()
	if err != nil {
		t.Fatalf("pod annotation injection client: %v", err)
	}
	pod, err := conn.CoreV1().Pods("default").Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get pod for annotation injection: %v", err)
	}
	annotations := pod.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[key] = value
	pod.SetAnnotations(annotations)
	if _, err := conn.CoreV1().Pods("default").Update(context.Background(), pod, metav1.UpdateOptions{}); err != nil {
		t.Fatalf("update pod annotation %q: %v", key, err)
	}
}

func testAccCheckIgnoredPodAnnotation(name, key string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn, err := testAccPodV1MainClientset()
		if err != nil {
			return err
		}
		pod, err := conn.CoreV1().Pods("default").Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if _, ok := pod.GetAnnotations()[key]; !ok {
			return fmt.Errorf("expected ignored annotation %q on pod %q", key, name)
		}
		return nil
	}
}

func testAccDeletePodByName(t *testing.T, namespace, name string) {
	t.Helper()
	conn, err := testAccPodV1MainClientset()
	if err != nil {
		t.Fatalf("delete pod %s/%s: client: %v", namespace, name, err)
	}
	if err := conn.CoreV1().Pods(namespace).Delete(context.Background(), name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		t.Fatalf("delete pod %s/%s: %v", namespace, name, err)
	}
	for i := 0; i < 60; i++ {
		_, err := conn.CoreV1().Pods(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("delete pod %s/%s: pod still present after 30s", namespace, name)
}

func testAccCheckKubernetesPodGeneratedNamePreserved(before, after *corev1api.Pod, expectedPrefix string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if before.ObjectMeta.Name != after.ObjectMeta.Name {
			return fmt.Errorf("expected generated pod name to remain stable across update: before=%q after=%q", before.ObjectMeta.Name, after.ObjectMeta.Name)
		}
		if before.ObjectMeta.GenerateName != after.ObjectMeta.GenerateName {
			return fmt.Errorf("expected generate_name to remain stable across update: before=%q after=%q", before.ObjectMeta.GenerateName, after.ObjectMeta.GenerateName)
		}
		if before.ObjectMeta.GenerateName != expectedPrefix {
			return fmt.Errorf("expected generate_name prefix %q, got %q", expectedPrefix, before.ObjectMeta.GenerateName)
		}
		if !strings.HasPrefix(before.ObjectMeta.Name, expectedPrefix) {
			return fmt.Errorf("expected generated pod name %q to use prefix %q", before.ObjectMeta.Name, expectedPrefix)
		}
		return nil
	}
}

func testAccCheckKubernetesPodSpecPreserved(before, after *corev1api.Pod) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if !reflect.DeepEqual(before.Spec, after.Spec) {
			return fmt.Errorf("expected pod spec to be preserved across metadata update")
		}
		return nil
	}
}
