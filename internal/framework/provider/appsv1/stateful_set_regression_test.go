// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1_test

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

type statefulSetMigrationSnapshot struct {
	id         string
	object     *appsv1.StatefulSet
	pods, pvcs map[string]types.UID
}

func statefulSetMigrationSnapshotFromState(resourceName string, state *terraform.State) (statefulSetMigrationSnapshot, error) {
	obj, err := getStatefulSetFromResourceName(state, resourceName)
	if err != nil {
		return statefulSetMigrationSnapshot{}, err
	}
	conn, err := testAccWorkloadClient()
	if err != nil {
		return statefulSetMigrationSnapshot{}, err
	}
	snapshot := statefulSetMigrationSnapshot{
		id:     state.RootModule().Resources[resourceName].Primary.ID,
		object: obj.DeepCopy(), pods: map[string]types.UID{}, pvcs: map[string]types.UID{},
	}
	err = wait.PollUntilContextTimeout(context.Background(), 250*time.Millisecond, time.Minute, true, func(ctx context.Context) (bool, error) {
		pods, err := conn.CoreV1().Pods(obj.Namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return false, err
		}
		snapshot.pods = map[string]types.UID{}
		for _, pod := range pods.Items {
			for _, owner := range pod.OwnerReferences {
				if owner.UID == obj.UID {
					snapshot.pods[pod.Name] = pod.UID
				}
			}
		}
		return obj.Spec.Replicas != nil && len(snapshot.pods) == int(*obj.Spec.Replicas), nil
	})
	if err != nil {
		return snapshot, fmt.Errorf("waiting for StatefulSet-owned pods: %w", err)
	}
	for _, claim := range obj.Spec.VolumeClaimTemplates {
		for ordinal := int32(0); ordinal < *obj.Spec.Replicas; ordinal++ {
			name := fmt.Sprintf("%s-%s-%d", claim.Name, obj.Name, ordinal)
			pvc, err := conn.CoreV1().PersistentVolumeClaims(obj.Namespace).Get(context.Background(), name, metav1.GetOptions{})
			if err != nil {
				return snapshot, err
			}
			snapshot.pvcs[pvc.Name] = pvc.UID
		}
	}
	return snapshot, nil
}

func statefulSetCaptureMigrationSnapshot(resourceName string, snapshot *statefulSetMigrationSnapshot) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		var err error
		*snapshot, err = statefulSetMigrationSnapshotFromState(resourceName, state)
		return err
	}
}

func statefulSetCheckMigrationSnapshot(resourceName string, before *statefulSetMigrationSnapshot) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		after, err := statefulSetMigrationSnapshotFromState(resourceName, state)
		if err != nil {
			return err
		}
		if before.id != after.id || before.object.UID != after.object.UID ||
			before.object.Generation != after.object.Generation || !before.object.CreationTimestamp.Equal(&after.object.CreationTimestamp) {
			return fmt.Errorf("StatefulSet identity or generation changed during a no-op upgrade")
		}
		if !reflect.DeepEqual(before.object.Spec, after.object.Spec) {
			return fmt.Errorf("StatefulSet API spec changed during a no-op upgrade")
		}
		if !reflect.DeepEqual(before.pods, after.pods) || !reflect.DeepEqual(before.pvcs, after.pvcs) {
			return fmt.Errorf("StatefulSet pod/PVC identities changed: pods %v -> %v, claims %v -> %v", before.pods, after.pods, before.pvcs, after.pvcs)
		}
		return nil
	}
}

func statefulSetIdentityChecks(resourceName, name string) []statecheck.StateCheck {
	return []statecheck.StateCheck{statecheck.ExpectIdentity(resourceName, map[string]knownvalue.Check{
		"namespace": knownvalue.StringExact("default"), "name": knownvalue.StringExact(name),
		"api_version": knownvalue.StringExact("apps/v1"), "kind": knownvalue.StringExact("StatefulSet"),
	})}
}

func TestAccKubernetesStatefulSetV1_migrationConfigured(t *testing.T) {
	for _, scenario := range []string{"full", "zero_replicas", "empty_replicas"} {
		t.Run(scenario, func(t *testing.T) {
			name := acctest.RandomWithPrefix("tf-sts-upgrade")
			resourceName := "kubernetes_stateful_set_v1.test"
			var snapshot statefulSetMigrationSnapshot
			config := testAccKubernetesStatefulSetV1ConfigMinimal(name, busyboxImage)
			legacyConfig := config
			switch scenario {
			case "full":
				config = statefulSetFullMigrationConfig(name, false)
				legacyConfig = statefulSetFullMigrationConfig(name, true)
			case "zero_replicas":
				config = strings.Replace(config, "  spec {", "  spec {\n    replicas = \"0\"", 1)
				legacyConfig = config
			case "empty_replicas":
				config = strings.Replace(config, "  spec {", "  spec {\n    replicas = \"\"", 1)
				legacyConfig = config
			}
			resource.ParallelTest(t, resource.TestCase{
				PreCheck:               func() { testAccPreCheck(t) },
				CheckDestroy:           testAccCheckKubernetesStatefulSetV1Destroy,
				TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)},
				Steps: []resource.TestStep{
					{
						ExternalProviders: map[string]resource.ExternalProvider{"kubernetes": {
							Source: "hashicorp/kubernetes", VersionConstraint: statefulSetSDKv2ProviderVersion,
						}},
						Config: legacyConfig,
						Check:  statefulSetCaptureMigrationSnapshot(resourceName, &snapshot),
					},
					{
						ProtoV6ProviderFactories: testAccProviderFactories,
						Config:                   config,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check:             statefulSetCheckMigrationSnapshot(resourceName, &snapshot),
						ConfigStateChecks: statefulSetIdentityChecks(resourceName, name),
					},
				},
			})
		})
	}
}

func statefulSetFullMigrationConfig(name string, legacy bool) string {
	config := testAccKubernetesStatefulSetV1ConfigBasic(name, agnhostImage)
	resources := `resources = [{
            requests = { cpu = "25m", memory = "32Mi" }
            limits = { cpu = "100m", memory = "64Mi" }
          }]`
	if legacy {
		config = strings.Replace(config, "persistent_volume_claim_retention_policy = [{", "persistent_volume_claim_retention_policy {", 1)
		config = strings.Replace(config, "    }]", "    }", 1)
		resources = strings.Replace(resources, "resources = [{", "resources {", 1)
		resources = strings.Replace(resources, "}]", "}", 1)
	}
	config = strings.Replace(config, `          args  = ["test-webserver"]`, `          args  = ["test-webserver"]
          `+resources+`
          env {
            name = "MIGRATION_LITERAL"
            value = "unchanged"
          }
          env {
            name = "POD_NAME"
            value_from {
              field_ref { field_path = "metadata.name" }
            }
          }
          security_context { allow_privilege_escalation = false }`, 1)
	return config
}

func TestAccKubernetesStatefulSetV1_replicasEmptyAndDrift(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-sts-replicas")
	resourceName := "kubernetes_stateful_set_v1.test"
	minimal := testAccKubernetesStatefulSetV1ConfigMinimal(name, busyboxImage)
	replicas := func(value string) string {
		return strings.Replace(minimal, "  spec {", fmt.Sprintf("  spec {\n    replicas = %q", value), 1)
	}
	empty := replicas("")
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) }, ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy: testAccCheckKubernetesStatefulSetV1Destroy,
		Steps: []resource.TestStep{
			{Config: replicas("3"), Check: resource.TestCheckResourceAttr(resourceName, "spec.0.replicas", "3")},
			{
				Config:           empty,
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
				Check:            resource.TestCheckResourceAttr(resourceName, "spec.0.replicas", "3"),
			},
			{
				Config: strings.Replace(empty, "  metadata {", "  metadata {\n    labels = { edited = \"true\" }", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "spec.0.replicas", "3"),
					func(state *terraform.State) error {
						obj, err := getStatefulSetFromResourceName(state, resourceName)
						if err != nil {
							return err
						}
						conn, err := testAccWorkloadClient()
						if err != nil {
							return err
						}
						_, err = conn.AppsV1().StatefulSets(obj.Namespace).Patch(context.Background(), obj.Name, types.MergePatchType, []byte(`{"spec":{"replicas":2}}`), metav1.PatchOptions{})
						return err
					},
				),
			},
			{
				Config:           strings.Replace(empty, "  metadata {", "  metadata {\n    labels = { edited = \"true\" }", 1),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
				Check:            resource.TestCheckResourceAttr(resourceName, "spec.0.replicas", "2"),
			},
			{Config: replicas("0"), Check: resource.TestCheckResourceAttr(resourceName, "spec.0.replicas", "0")},
			{Config: minimal, Check: resource.TestCheckResourceAttr(resourceName, "spec.0.replicas", "0")},
		},
	})
}

func TestAccKubernetesStatefulSetV1_retentionRemoval(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-sts-retention")
	resourceName := "kubernetes_stateful_set_v1.test"
	base := testAccKubernetesStatefulSetV1ConfigMinimal(name, busyboxImage)
	deletePolicy := strings.Replace(base, "  spec {", `  spec {
    persistent_volume_claim_retention_policy = [{
      when_deleted = "Delete"
      when_scaled = "Delete"
    }]`, 1)
	retainPolicy := strings.ReplaceAll(deletePolicy, `"Delete"`, `"Retain"`)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) }, ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy: testAccCheckKubernetesStatefulSetV1Destroy,
		Steps: []resource.TestStep{
			{Config: deletePolicy},
			{
				Config:           base,
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_claim_retention_policy.0.when_deleted", "Delete"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_claim_retention_policy.0.when_scaled", "Delete"),
				),
			},
			{Config: retainPolicy},
			{
				Config:           base,
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_claim_retention_policy.0.when_deleted", "Retain"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_claim_retention_policy.0.when_scaled", "Retain"),
				),
			},
		},
	})
}

func TestAccKubernetesStatefulSetV1_importPersistedNoop(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-sts-import")
	resourceName := "kubernetes_stateful_set_v1.test"
	config := testAccKubernetesStatefulSetV1ConfigMinimal(name, busyboxImage)
	var snapshot statefulSetMigrationSnapshot
	t.Cleanup(func() {
		if snapshot.object == nil {
			return
		}
		conn, err := testAccWorkloadClient()
		if err != nil {
			t.Error(err)
			return
		}
		obj, err := conn.AppsV1().StatefulSets("default").Get(context.Background(), name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return
		}
		if err != nil {
			t.Error(err)
			return
		}
		if obj.UID != snapshot.object.UID {
			t.Errorf("refusing cleanup of unexpected StatefulSet UID %s", obj.UID)
			return
		}
		if err := conn.AppsV1().StatefulSets("default").Delete(context.Background(), name, metav1.DeleteOptions{
			Preconditions: &metav1.Preconditions{UID: &snapshot.object.UID},
		}); err != nil && !apierrors.IsNotFound(err) {
			t.Error(err)
			return
		}
		if err := wait.PollUntilContextTimeout(context.Background(), 250*time.Millisecond, time.Minute, true, func(ctx context.Context) (bool, error) {
			_, err := conn.AppsV1().StatefulSets("default").Get(ctx, name, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}); err != nil {
			t.Error(err)
		}
	})
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) }, ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:           testAccCheckKubernetesStatefulSetV1Destroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)},
		Steps: []resource.TestStep{
			{Config: config, Check: statefulSetCaptureMigrationSnapshot(resourceName, &snapshot)},
			{
				Config: `removed {
  from = kubernetes_stateful_set_v1.test
  lifecycle { destroy = false }
}`,
			},
			{
				Config: config, ResourceName: resourceName,
				ImportState: true, ImportStatePersist: true, ImportStateId: "default/" + name,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 || states[0].ID != snapshot.id ||
						states[0].Attributes["metadata.0.uid"] != string(snapshot.object.UID) ||
						states[0].Attributes["wait_for_rollout"] != "true" {
						return fmt.Errorf("import did not preserve StatefulSet identity and default rollout policy: %v", states)
					}
					return nil
				},
			},
			{
				Config:            config,
				ConfigPlanChecks:  resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
				Check:             statefulSetCheckMigrationSnapshot(resourceName, &snapshot),
				ConfigStateChecks: statefulSetIdentityChecks(resourceName, name),
			},
		},
	})
}

func TestAccKubernetesStatefulSetV1_volumeClaimReplacementBoundaries(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-sts-claims")
	resourceName := "kubernetes_stateful_set_v1.test"
	config := testAccKubernetesStatefulSetV1ConfigBasic(name, agnhostImage)
	start := strings.Index(config, "    volume_claim_template {")
	end := strings.LastIndex(config, "\n  }\n}")
	if start < 0 || end < start {
		t.Fatal("missing volume claim template in test configuration")
	}
	claim := config[start:end]
	var snapshot statefulSetMigrationSnapshot
	steps := []resource.TestStep{{
		Config: config,
		Check:  statefulSetCaptureMigrationSnapshot(resourceName, &snapshot),
	}}
	for _, changed := range []string{
		strings.Replace(config, `storage = "1Gi"`, `storage = "2Gi"`, 1),
		strings.Replace(config, `access_modes = ["ReadWriteOnce"]`, `access_modes = ["ReadWriteMany"]`, 1),
		config[:start] + strings.Replace(claim, "      metadata {", "      metadata {\n        labels = { changed = \"true\" }", 1) + config[end:],
		config[:start] + config[end:],
		config[:end] + "\n" + strings.Replace(claim, `name = "ss-test"`, `name = "other"`, 1) + config[end:],
	} {
		steps = append(steps, resource.TestStep{
			Config: changed, PlanOnly: true, ExpectNonEmptyPlan: true,
			ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPreRefresh: []plancheck.PlanCheck{
				plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionDestroyBeforeCreate),
			}},
		})
	}
	steps = append(steps, resource.TestStep{
		Config:           config,
		ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
		Check:            statefulSetCheckMigrationSnapshot(resourceName, &snapshot),
	})
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) }, ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy: testAccCheckKubernetesStatefulSetV1Destroy, Steps: steps,
	})
}
