// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8types "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

func TestAccDeploymentV1_ExplicitEmptyRemoval(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-apps-deploy-empty-removal")
	config := testAccDeploymentExplicitEmptyConfig(name)
	removed := testAccDeploymentBareContainerConfig(name)
	address := "kubernetes_deployment_v1.test"
	var snapshot deploymentMigrationSnapshot
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{Config: config, Check: testAccDeploymentMigrationCapture(address, &snapshot)},
			{
				Config: removed,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check:             testAccDeploymentMigrationUnchanged(address, &snapshot),
				ConfigStateChecks: deploymentIdentityChecks(address, &snapshot),
			},
		},
	})
}

func TestAccDeploymentV1_WaitPolicyStateOnlyAndRealUpdate(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-apps-deploy-wait-policy")
	config := testAccFrameworkDeploymentConfigBasic(name, busyboxImage)
	noWait := config + "\n"
	noWait = strings.Replace(noWait, "  spec {", "  wait_for_rollout = false\n  spec {", 1)
	changed := strings.Replace(noWait, "  spec {", `  timeouts { update = "5s" }
  spec {
    paused = true
    replicas = "2"`, 1)
	address := "kubernetes_deployment_v1.test"
	var snapshot deploymentMigrationSnapshot
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{Config: config, Check: testAccDeploymentMigrationCapture(address, &snapshot)},
			{
				Config: noWait,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{deploymentWaitOnlyPlanCheck{before: true, after: false}},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDeploymentMigrationUnchanged(address, &snapshot),
					resource.TestCheckResourceAttr(address, "wait_for_rollout", "false"),
				),
			},
			{
				// The paused controller cannot reach two ready replicas. This
				// must honor the configured false policy, not prior state's true.
				Config: changed,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(address, "wait_for_rollout", "false"),
					resource.TestCheckResourceAttr(address, "spec.0.paused", "true"),
					resource.TestCheckResourceAttr(address, "spec.0.replicas", "2"),
				),
				ConfigStateChecks: deploymentIdentityChecks(address, &snapshot),
			},
		},
	})
}

func TestAccDeploymentV1_GeneratedNameUnrelatedEdit(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-apps-deploy-generated") + "-"
	config := testAccFrameworkDeploymentConfigBasic(name, busyboxImage)
	config = strings.Replace(config, "name = ", "generate_name = ", 1)
	updated := strings.Replace(config, `app = "demo"`, `app = "demo", changed = "true"`, 1)
	address := "kubernetes_deployment_v1.test"
	var before, after appsv1.Deployment
	var snapshot deploymentMigrationSnapshot
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(address, &before),
					testAccDeploymentMigrationCapture(address, &snapshot),
				),
			},
			{
				Config: updated,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(address, &after),
					testAccCheckKubernetesDeploymentForceNew(&before, &after, false),
					resource.TestCheckResourceAttr(address, "metadata.0.generate_name", name),
					resource.TestCheckResourceAttr(address, "metadata.0.labels.changed", "true"),
				),
				ConfigStateChecks: deploymentIdentityChecks(address, &snapshot),
			},
		},
	})
}

func TestAccDeploymentV1_OptionalRemoval(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-apps-deploy-removal")
	removed := testAccFrameworkDeploymentConfigReplicas(name, busyboxImage, "2")
	config := strings.Replace(removed, `replicas = "2"`, `replicas = "2"
    min_ready_seconds = 2
    progress_deadline_seconds = 120
    revision_history_limit = 2`, 1)
	// Unlike the defaulted scalar fields, omitted replicas is API-computed and
	// must retain the existing desired count rather than silently resetting it.
	removed = strings.Replace(removed, `replicas = "2"`, "", 1)
	address := "kubernetes_deployment_v1.test"
	var before, after appsv1.Deployment
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  testAccCheckKubernetesDeploymentV1Exists(address, &before),
			},
			{
				Config: removed,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(address, &after),
					testAccCheckKubernetesDeploymentForceNew(&before, &after, false),
					resource.TestCheckResourceAttr(address, "spec.0.min_ready_seconds", "0"),
					resource.TestCheckResourceAttr(address, "spec.0.progress_deadline_seconds", "600"),
					resource.TestCheckResourceAttr(address, "spec.0.revision_history_limit", "10"),
					resource.TestCheckResourceAttr(address, "spec.0.replicas", "2"),
					func(*terraform.State) error {
						if after.Spec.MinReadySeconds != 0 || *after.Spec.ProgressDeadlineSeconds != 600 ||
							*after.Spec.RevisionHistoryLimit != 10 || *after.Spec.Replicas != 2 {
							return fmt.Errorf("optional-field removal did not preserve the API contract: %#v", after.Spec)
						}
						return nil
					},
				),
			},
		},
	})
}

func TestAccDeploymentV1_FullImportPlan(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-apps-deploy-import")
	config := testAccDeploymentMigrationFullConfig(name, "RollingUpdate", false)
	noWait := strings.Replace(config, "  spec {", "  wait_for_rollout = false\n  spec {", 1)
	address := "kubernetes_deployment_v1.test"
	var snapshot deploymentMigrationSnapshot
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{Config: config, Check: testAccDeploymentMigrationCapture(address, &snapshot)},
			{
				ResourceName:      address,
				ImportState:       true,
				ImportStateVerify: true,
				// Controllers can update status while Terraform runs the import.
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: noWait,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{deploymentWaitOnlyPlanCheck{before: true, after: false}},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(address, "wait_for_rollout", "false"),
					testAccDeploymentMigrationUnchanged(address, &snapshot),
				),
			},
			{
				ResourceName:    address,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
				ImportPlanChecks: resource.ImportPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: noWait,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check:             testAccDeploymentMigrationUnchanged(address, &snapshot),
				ConfigStateChecks: deploymentIdentityChecks(address, &snapshot),
			},
		},
	})
}

func TestAccDeploymentV1_DriftAndDisappearance(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-apps-deploy-drift")
	config := testAccFrameworkDeploymentConfigReplicas(name, busyboxImage, "1")
	address := "kubernetes_deployment_v1.test"
	var before, restored, recreated appsv1.Deployment
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(address, &before),
					func(*terraform.State) error {
						client, err := testAccWorkloadClient()
						if err != nil {
							return err
						}
						_, err = client.AppsV1().Deployments(before.Namespace).Patch(context.Background(), before.Name, k8types.MergePatchType,
							[]byte(`{"spec":{"replicas":0},"metadata":{"annotations":{"example.com/external":"reconcile","deployment.kubernetes.io/external":"preserve"}}}`), metav1.PatchOptions{})
						return err
					},
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(address, &restored),
					testAccCheckKubernetesDeploymentForceNew(&before, &restored, false),
					resource.TestCheckResourceAttr(address, "spec.0.replicas", "1"),
					func(*terraform.State) error {
						if restored.Annotations["deployment.kubernetes.io/external"] != "preserve" {
							return fmt.Errorf("drift reconciliation removed a controller-owned annotation")
						}
						if _, found := restored.Annotations["example.com/external"]; found {
							return fmt.Errorf("drift reconciliation retained an unconfigured ordinary annotation")
						}
						client, err := testAccWorkloadClient()
						if err != nil {
							return err
						}
						if err := client.AppsV1().Deployments(restored.Namespace).Delete(context.Background(), restored.Name, metav1.DeleteOptions{}); err != nil {
							return err
						}
						return wait.PollUntilContextTimeout(context.Background(), 100*time.Millisecond, 30*time.Second, true, func(ctx context.Context) (bool, error) {
							_, err := client.AppsV1().Deployments(restored.Namespace).Get(ctx, restored.Name, metav1.GetOptions{})
							if apierrors.IsNotFound(err) {
								return true, nil
							}
							return false, err
						})
					},
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionCreate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(address, &recreated),
					testAccCheckKubernetesDeploymentForceNew(&before, &recreated, true),
				),
			},
		},
	})
}
