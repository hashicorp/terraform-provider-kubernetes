// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccKubernetesStatefulSetV1_minimal(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_stateful_set_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesStatefulSetV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetV1ConfigMinimal(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.namespace"),
				),
			},
		},
	})
}

func TestAccKubernetesStatefulSetV1_identity(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_stateful_set_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesStatefulSetV1Destroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetV1Config_identity(name, imageName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(
						resourceName, map[string]knownvalue.Check{
							"namespace":   knownvalue.StringExact("default"),
							"name":        knownvalue.StringExact(name),
							"api_version": knownvalue.StringExact("apps/v1"),
							"kind":        knownvalue.StringExact("StatefulSet"),
						},
					),
				},
			},
			{
				ResourceName:    resourceName,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func TestAccKubernetesStatefulSetV1_basic(t *testing.T) {
	var conf appsv1.StatefulSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_stateful_set_v1.test"
	imageName := agnhostImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.27.0")
			skipIfRunningInEks(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesStatefulSetV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetV1ConfigBasic(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "wait_for_rollout", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.namespace"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.replicas", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.min_ready_seconds", "10"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.revision_history_limit", "11"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_name", "ss-test-service"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_claim_retention_policy.0.when_deleted", "Delete"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_claim_retention_policy.0.when_scaled", "Delete"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.0.match_labels.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.0.match_labels.app", "ss-test"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.0.labels.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.0.labels.app", "ss-test"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.name", "ss-test"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.port.0.container_port", "80"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.port.0.name", "web"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.name", "ss-test"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.volume_mount.0.mount_path", "/work-dir"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_strategy.0.type", "RollingUpdate"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_strategy.0.rolling_update.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_strategy.0.rolling_update.0.partition", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_claim_template.0.metadata.0.name", "ss-test"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_claim_template.0.spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_claim_template.0.spec.0.access_modes.0", "ReadWriteOnce"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_claim_template.0.spec.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_claim_template.0.spec.0.resources.0.requests.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_claim_template.0.spec.0.resources.0.requests.storage", "1Gi"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"wait_for_rollout",
					"spec.0.update_strategy.#",
					"spec.0.update_strategy.0.%",
					"spec.0.update_strategy.0.rolling_update.#",
					"spec.0.update_strategy.0.rolling_update.0.%",
					"spec.0.update_strategy.0.rolling_update.0.partition",
					"spec.0.update_strategy.0.type",
				},
			},
		},
	})
}

func TestAccKubernetesStatefulSetV1_basic_idempotency(t *testing.T) {
	var conf appsv1.StatefulSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_stateful_set_v1.test"
	imageName := agnhostImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.27.0")
			skipIfRunningInEks(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesStatefulSetV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetV1ConfigBasic(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
				),
			},
			{
				Config:             testAccKubernetesStatefulSetV1ConfigBasic(name, imageName),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
				),
			},
		},
	})
}

func TestAccKubernetesStatefulSetV1_Update(t *testing.T) {
	var conf appsv1.StatefulSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_stateful_set_v1.test"
	imageName := agnhostImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.27.0")
			skipIfRunningInEks(t)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesStatefulSetV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetV1ConfigMinimal(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigUpdateImage(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigUpdatedSelectorLabels(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.0.match_labels.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.0.match_labels.app", "ss-test"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.0.match_labels.layer", "ss-test-layer"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.0.labels.app", "ss-test"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.0.labels.layer", "ss-test-layer"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigUpdateReplicas(name, imageName, "3"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.replicas", "3"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigUpdateReplicas(name, imageName, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					// NOTE setting to empty should preserve the current replica count
					resource.TestCheckResourceAttr(resourceName, "spec.0.replicas", "3"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigUpdateReplicas(name, imageName, "0"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.replicas", "0"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigUpdateMinReadySeconds(name, imageName, 10),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.min_ready_seconds", "10"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigUpdateMinReadySeconds(name, imageName, 0),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.min_ready_seconds", "0"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigRollingUpdatePartition(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_strategy.0.rolling_update.0.partition", "2"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigUpdateStrategyOnDelete(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_strategy.0.type", "OnDelete"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_strategy.0.rolling_update.#", "0"),
					resource.TestCheckNoResourceAttr(resourceName, "spec.0.update_strategy.0.rolling_update.0.partition"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigUpdateStrategyOnDelete(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_strategy.0.type", "OnDelete"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.update_strategy.0.rolling_update.#", "0"),
					resource.TestCheckNoResourceAttr(resourceName, "spec.0.update_strategy.0.rolling_update.0.partition"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigUpdateTemplate(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.port.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.port.0.container_port", "80"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.port.0.name", "web"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.port.1.container_port", "443"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.port.1.name", "secure"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.dns_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.dns_config.0.nameservers.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.dns_config.0.nameservers.0", "1.1.1.1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.dns_config.0.nameservers.1", "8.8.8.8"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.dns_config.0.nameservers.2", "9.9.9.9"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.dns_config.0.searches.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.dns_config.0.searches.0", "kubernetes.io"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.dns_config.0.option.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.dns_config.0.option.0.name", "ndots"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.dns_config.0.option.0.value", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.dns_config.0.option.1.name", "use-vc"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.dns_config.0.option.1.value", ""),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.dns_policy", "Default"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigUpdatePersistentVolumeClaimRetentionPolicy(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_claim_retention_policy.0.when_deleted", "Retain"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_claim_retention_policy.0.when_scaled", "Retain"),
				),
			},
		},
	})
}

func TestAccKubernetesStatefulSetV1_host_users(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_stateful_set_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.25.0") // User namespaces is beta in 1.25
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesStatefulSetV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetV1ConfigHostUsers(name, imageName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.host_users", "true"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigHostUsers(name, imageName, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.host_users", "false"),
				),
			},
		},
	})
}

func TestAccKubernetesStatefulSetV1_waitForRollout(t *testing.T) {
	var conf1, conf2 appsv1.StatefulSet
	imageName := busyboxImage
	imageName1 := agnhostImage
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_stateful_set_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfRunningInEks(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesStatefulSetV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetV1ConfigWaitForRollout(name, imageName, "true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "wait_for_rollout", "true"),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigWaitForRollout(name, imageName1, "false"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "wait_for_rollout", "false"),
					testAccCheckKubernetesStatefulSetForceNew(&conf1, &conf2, false),
				),
			},
		},
	})
}

func TestAccKubernetesStatefulSetV1_minimalWithTemplateNamespace(t *testing.T) {
	var conf1, conf2 appsv1.StatefulSet

	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_stateful_set_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesStatefulSetV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetV1ConfigMinimal(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.namespace"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.metadata.0.namespace", ""),
				),
			},
			{
				Config: testAccKubernetesStatefulSetV1ConfigMinimalWithTemplateNamespace(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.namespace"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.template.0.metadata.0.namespace"),
					testAccCheckKubernetesStatefulSetForceNew(&conf1, &conf2, true),
				),
			},
		},
	})
}

func testAccCheckKubernetesStatefulSetForceNew(old, new *appsv1.StatefulSet, wantNew bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if wantNew {
			if old.ObjectMeta.UID == new.ObjectMeta.UID {
				return fmt.Errorf("Expecting new resource for StatefulSet %s", old.ObjectMeta.UID)
			}
		} else {
			if old.ObjectMeta.UID != new.ObjectMeta.UID {
				return fmt.Errorf("Expecting StatefulSet UIDs to be the same: expected %s got %s", old.ObjectMeta.UID, new.ObjectMeta.UID)
			}
		}
		return nil
	}
}

func testAccCheckKubernetesStatefulSetV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_stateful_set_v1" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Namespace == namespace && resp.Name == name {
				return fmt.Errorf("StatefulSet still exists: %s: (Generation %#v)", rs.Primary.ID, resp.Status.ObservedGeneration)
			}
		}

		// StatefulSet can create a PVC via volumeClaimTemplate. However, once the StatefulSet is removed, the PVC remains.
		// There is a beta feature(persistentVolumeClaimRetentionPolicy) since 1.27 that aims to address this problem:
		// - https://kubernetes.io/blog/2021/12/16/kubernetes-1-23-statefulset-pvc-auto-deletion/
		// - https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#persistentvolumeclaim-retention
		// By default, StatefulSetAutoDeletePVC feature is not enabled in the feature gate.
		// That is why we clean up resources manually here.
		pvc, err := conn.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("Failed to list PVCs in %q namespace", namespace)
		}
		for _, p := range pvc.Items {
			// PVC gets generated in the following format:
			// *.volumeClaimTemplate.metatada.name-statefulSet.metatada.name
			//
			// Since statefulSet.metatada.name is uniq, we could use it as a match.
			if strings.Contains(p.Name, name) {
				err := conn.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, p.Name, metav1.DeleteOptions{})
				if err != nil {
					if !errors.IsNotFound(err) {
						return fmt.Errorf("Failed to delete PVC %q in namespace %q", p.Name, namespace)
					}
				}
			}
		}
	}

	return nil
}

func getStatefulSetFromResourceName(s *terraform.State, n string) (*appsv1.StatefulSet, error) {
	rs, ok := s.RootModule().Resources[n]
	if !ok {
		return nil, fmt.Errorf("Not found: %s", n)
	}

	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return nil, err
	}
	ctx := context.TODO()

	namespace, name, err := idParts(rs.Primary.ID)
	if err != nil {
		return nil, err
	}

	out, err := conn.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return out, nil
}

func testAccCheckKubernetesStatefulSetV1Exists(n string, obj *appsv1.StatefulSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		d, err := getStatefulSetFromResourceName(s, n)
		if err != nil {
			return err
		}
		*obj = *d
		return nil
	}
}

func testAccKubernetesStatefulSetV1ConfigMinimal(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set_v1" "test" {
  metadata {
    name = "%s"
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
          image   = "%s"
          command = ["sleep", "300"]
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, name, imageName)
}

func testAccKubernetesStatefulSetV1Config_identity(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set_v1" "test" {
  metadata {
    name = "%s"
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
          image   = "%s"
          command = ["sleep", "300"]
        }
        termination_grace_period_seconds = 1
      }
    }
  }
  wait_for_rollout = false
}
`, name, imageName)
}

func testAccKubernetesStatefulSetV1ConfigBasic(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    name = "%s"
  }

  spec {
    min_ready_seconds      = 10
    pod_management_policy  = "OrderedReady"
    replicas               = 1
    revision_history_limit = 11

    selector {
      match_labels = {
        app = "ss-test"
      }
    }

    service_name = "ss-test-service"

    persistent_volume_claim_retention_policy {
      when_deleted = "Delete"
      when_scaled  = "Delete"
    }

    template {
      metadata {
        labels = {
          app = "ss-test"
        }
      }

      spec {
        container {
          name  = "ss-test"
          image = %q
          args  = ["test-webserver"]

          port {
            name           = "web"
            container_port = 80
          }

          readiness_probe {
            initial_delay_seconds = 3
            period_seconds        = 1
            http_get {
              path = "/"
              port = 80
            }
          }

          volume_mount {
            name       = "ss-test"
            mount_path = "/work-dir"
          }
        }
      }
    }

    update_strategy {
      type = "RollingUpdate"

      rolling_update {
        partition = 1
      }
    }

    volume_claim_template {
      metadata {
        name = "ss-test"
      }

      spec {
        access_modes = ["ReadWriteOnce"]
        resources {
          requests = {
            storage = "1Gi"
          }
        }
      }
    }
  }
}
`, name, imageName)
}

func testAccKubernetesStatefulSetV1ConfigUpdateImage(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    name = "%s"
  }

  spec {
    pod_management_policy  = "OrderedReady"
    replicas               = 1
    revision_history_limit = 11

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
          name  = "ss-test"
          image = %q
          args  = ["pause"]

          port {
            container_port = "80"
            name           = "web"
          }

          volume_mount {
            name       = "ss-test"
            mount_path = "/work-dir"
          }
        }
      }
    }

    update_strategy {
      type = "RollingUpdate"

      rolling_update {
        partition = 1
      }
    }

    volume_claim_template {
      metadata {
        name = "ss-test"
      }

      spec {
        access_modes = ["ReadWriteOnce"]

        resources {
          requests = {
            storage = "1Gi"
          }
        }
      }
    }
  }
}
`, name, imageName)
}

func testAccKubernetesStatefulSetV1ConfigUpdatedSelectorLabels(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    name = "%s"
  }

  spec {
    pod_management_policy  = "OrderedReady"
    replicas               = 1
    revision_history_limit = 11

    selector {
      match_labels = {
        app   = "ss-test"
        layer = "ss-test-layer"
      }
    }

    service_name = "ss-test-service"

    template {
      metadata {
        labels = {
          app   = "ss-test"
          layer = "ss-test-layer"
        }
      }

      spec {
        container {
          name  = "ss-test"
          image = %q
          args  = ["pause"]

          port {
            container_port = "80"
            name           = "web"
          }

          volume_mount {
            name       = "ss-test"
            mount_path = "/work-dir"
          }
        }
      }
    }

    update_strategy {
      type = "RollingUpdate"

      rolling_update {
        partition = 0
      }
    }

    volume_claim_template {
      metadata {
        name = "ss-test"
      }

      spec {
        access_modes = ["ReadWriteOnce"]

        resources {
          requests = {
            storage = "1Gi"
          }
        }
      }
    }
  }
}
`, name, imageName)
}

func testAccKubernetesStatefulSetV1ConfigUpdateReplicas(name, imageName, replicas string) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    name = "%s"
  }

  spec {
    pod_management_policy  = "OrderedReady"
    replicas               = %q
    revision_history_limit = 11

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
          name  = "ss-test"
          image = %q
          args  = ["pause"]

          port {
            container_port = "80"
            name           = "web"
          }

          volume_mount {
            name       = "ss-test"
            mount_path = "/work-dir"
          }
        }
        termination_grace_period_seconds = 1
      }
    }

    update_strategy {
      type = "RollingUpdate"

      rolling_update {
        partition = 1
      }
    }

    volume_claim_template {
      metadata {
        name = "ss-test"
      }

      spec {
        access_modes = ["ReadWriteOnce"]

        resources {
          requests = {
            storage = "1Gi"
          }
        }
      }
    }
  }
}
`, name, replicas, imageName)
}

func testAccKubernetesStatefulSetV1ConfigUpdateMinReadySeconds(name string, imageName string, minReadySeconds int) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    name = "%s"
  }

  spec {
    min_ready_seconds      = %d
    pod_management_policy  = "OrderedReady"
    replicas               = 1
    revision_history_limit = 11

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
          name  = "ss-test"
          image = %q
          args  = ["pause"]

          port {
            container_port = "80"
            name           = "web"
          }

          volume_mount {
            name       = "ss-test"
            mount_path = "/work-dir"
          }
        }
        termination_grace_period_seconds = 1
      }
    }

    update_strategy {
      type = "RollingUpdate"

      rolling_update {
        partition = 1
      }
    }

    volume_claim_template {
      metadata {
        name = "ss-test"
      }

      spec {
        access_modes = ["ReadWriteOnce"]

        resources {
          requests = {
            storage = "1Gi"
          }
        }
      }
    }
  }
}
`, name, minReadySeconds, imageName)
}

func testAccKubernetesStatefulSetV1ConfigUpdateTemplate(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    name = "%s"
  }

  spec {
    pod_management_policy  = "OrderedReady"
    replicas               = 1
    revision_history_limit = 11

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
          name  = "ss-test"
          image = %q
          args  = ["pause"]

          port {
            container_port = "80"
            name           = "web"
          }

          port {
            container_port = "443"
            name           = "secure"
          }

          volume_mount {
            name       = "ss-test"
            mount_path = "/work-dir"
          }
        }

        dns_config {
          nameservers = ["1.1.1.1", "8.8.8.8", "9.9.9.9"]
          searches    = ["kubernetes.io"]

          option {
            name  = "ndots"
            value = 1
          }

          option {
            name = "use-vc"
          }
        }

        dns_policy = "Default"
      }
    }

    update_strategy {
      type = "RollingUpdate"

      rolling_update {
        partition = 1
      }
    }

    volume_claim_template {
      metadata {
        name = "ss-test"
      }

      spec {
        access_modes = ["ReadWriteOnce"]

        resources {
          requests = {
            storage = "1Gi"
          }
        }
      }
    }
  }
}
`, name, imageName)
}

func testAccKubernetesStatefulSetV1ConfigRollingUpdatePartition(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    name = "%s"
  }

  spec {
    pod_management_policy  = "OrderedReady"
    replicas               = 1
    revision_history_limit = 11

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
          name  = "ss-test"
          image = %q
          args  = ["pause"]

          port {
            container_port = "80"
            name           = "web"
          }

          volume_mount {
            name       = "ss-test"
            mount_path = "/work-dir"
          }
        }
      }
    }

    update_strategy {
      type = "RollingUpdate"

      rolling_update {
        partition = 2
      }
    }

    volume_claim_template {
      metadata {
        name = "ss-test"
      }

      spec {
        access_modes = ["ReadWriteOnce"]

        resources {
          requests = {
            storage = "1Gi"
          }
        }
      }
    }
  }
}
`, name, imageName)
}

func testAccKubernetesStatefulSetV1ConfigUpdateStrategyOnDelete(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    name = "%s"
  }

  spec {
    pod_management_policy  = "OrderedReady"
    replicas               = 1
    revision_history_limit = 11

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
          name  = "ss-test"
          image = %q
          args  = ["pause"]

          port {
            container_port = "80"
            name           = "web"
          }

          volume_mount {
            name       = "ss-test"
            mount_path = "/work-dir"
          }
        }
      }
    }

    update_strategy {
      type = "OnDelete"
    }

    volume_claim_template {
      metadata {
        name = "ss-test"
      }

      spec {
        access_modes = ["ReadWriteOnce"]

        resources {
          requests = {
            storage = "1Gi"
          }
        }
      }
    }
  }

  wait_for_rollout = false
}
`, name, imageName)
}

func testAccKubernetesStatefulSetV1ConfigWaitForRollout(name, imageName, waitForRollout string) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set_v1" "test" {
  metadata {
    name = "%s"
  }

  timeouts {
    create = "10m"
    read   = "10m"
    update = "10m"
    delete = "10m"
  }

  spec {
    replicas = 2

    selector {
      match_labels = {
        app = "ss-test"
      }
    }

    update_strategy {
      type = "RollingUpdate"
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
          image   = "%s"
          command = ["/bin/httpd", "-f", "-p", "80"]
          args    = ["test-webserver"]

          port {
            container_port = 80
          }

          readiness_probe {
            initial_delay_seconds = 3
            period_seconds        = 1
            tcp_socket {
              port = 80
            }
          }
        }
      }
    }
  }

  wait_for_rollout = %s
}
`, name, imageName, waitForRollout)
}

func testAccKubernetesStatefulSetV1ConfigUpdatePersistentVolumeClaimRetentionPolicy(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    name = "%s"
  }

  spec {
    pod_management_policy  = "OrderedReady"
    replicas               = 1
    revision_history_limit = 11

    selector {
      match_labels = {
        app = "ss-test"
      }
    }

    service_name = "ss-test-service"

    persistent_volume_claim_retention_policy {
      when_deleted = "Retain"
      when_scaled  = "Retain"
    }

    template {
      metadata {
        labels = {
          app = "ss-test"
        }
      }

      spec {
        container {
          name  = "ss-test"
          image = %q
          args  = ["test-webserver"]

          port {
            name           = "web"
            container_port = 80
          }

          readiness_probe {
            initial_delay_seconds = 5
            http_get {
              path = "/"
              port = 80
            }
          }

          volume_mount {
            name       = "ss-test"
            mount_path = "/work-dir"
          }
        }
      }
    }

    update_strategy {
      type = "RollingUpdate"

      rolling_update {
        partition = 1
      }
    }

    volume_claim_template {
      metadata {
        name = "ss-test"
      }

      spec {
        access_modes = ["ReadWriteOnce"]
        resources {
          requests = {
            storage = "1Gi"
          }
        }
      }
    }
  }
}
`, name, imageName)
}

func testAccKubernetesStatefulSetV1ConfigMinimalWithTemplateNamespace(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_stateful_set_v1" "test" {
  metadata {
    name = "%s"
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
        // The namespace field is just a stub and does not influence where the Pod will be created.
        // The Pod will be created within the same Namespace as the Stateful Set resource.
        namespace = "fake" // Doesn't have to exist.
        labels = {
          app = "ss-test"
        }
      }
      spec {
        container {
          name    = "ss-test"
          image   = "%s"
          command = ["sleep", "300"]
        }
        termination_grace_period_seconds = 1
      }
    }
  }
}
`, name, imageName)
}

func testAccKubernetesStatefulSetV1ConfigHostUsers(name, image string, hostUsers bool) string {
	return fmt.Sprintf(`
resource "kubernetes_stateful_set_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    replicas = 1
    selector {
      match_labels = {
        app = "tf-acc-test"
      }
    }
	service_name = "nginx"
    template {
      metadata {
        labels = {
          app = "tf-acc-test"
        }
      }
      spec {
        host_users = %t
        container {
          image = "%s"
          name  = "test"
        }
      }
    }
  }
}
`, name, hostUsers, image)
}
