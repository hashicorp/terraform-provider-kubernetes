// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	daemonSetResourceName      = "kubernetes_daemon_set_v1.test"
	daemonSetSDKv2ProviderV321 = "3.2.1"
)

type daemonSetObjectCheck func(*appsv1.DaemonSet) error

type daemonSetMigrationCase struct {
	name            string
	sdkConfig       string
	frameworkConfig string
	apiCheck        daemonSetObjectCheck
	updateConfig    string
	updateCheck     daemonSetObjectCheck
}

func TestAccDaemonSetV1_UpgradeFromSDKV2_AfterNoOpApply(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-ds-refreshed-upgrade")
	config := strings.ReplaceAll(daemonSetSDKConfigMinimal("refreshed", busyboxImage), "${NAME}", name)
	external := map[string]resource.ExternalProvider{
		"kubernetes": {
			VersionConstraint: daemonSetSDKv2ProviderV321,
			Source:            "hashicorp/kubernetes",
		},
	}
	var before, after appsv1.DaemonSet

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckDaemonSetDestroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: external,
				Config:            config,
				Check:             testAccCheckDaemonSetExists(daemonSetResourceName, &before),
			},
			{
				ExternalProviders: external,
				Config:            config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckDaemonSetExists(daemonSetResourceName, &after),
					testAccCheckDaemonSetNotRecreated(&before, &after),
					testAccCheckDaemonSetSpecUnchanged(&before, &after),
				),
			},
		},
	})
}

func TestAccDaemonSetV1_MoveFromSDKV2Alias(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-apps-daemon-move")
	targetConfig := testAccKubernetesDaemonSetV1Config_minimal(name, busyboxImage)
	sourceConfig := strings.Replace(targetConfig, `"kubernetes_daemon_set_v1"`, `"kubernetes_daemonset"`, 1)
	movedConfig := targetConfig + `
moved {
  from = kubernetes_daemonset.test
  to   = kubernetes_daemon_set_v1.test
}
`
	var before, after appsv1.DaemonSet

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckDaemonSetDestroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						VersionConstraint: daemonSetSDKv2ProviderV321,
						Source:            "hashicorp/kubernetes",
					},
				},
				Config: sourceConfig,
				Check:  testAccCheckDaemonSetExists("kubernetes_daemonset.test", &before),
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   movedConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckDaemonSetExists(daemonSetResourceName, &after),
					testAccCheckDaemonSetNotRecreated(&before, &after),
					testAccCheckDaemonSetStateIdentity(daemonSetResourceName, &after),
				),
			},
		},
	})
}

func TestAccDaemonSetV1_StrategyTransitions(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-apps-daemon-strategy")
	config := func(strategy string) string {
		return strings.ReplaceAll(
			daemonSetFrameworkConfigWithStrategy("strategy-transition", busyboxImage, strategy),
			"${NAME}",
			name,
		)
	}
	rolling := config(`strategy = [{
  type = "RollingUpdate"
  rolling_update = [{
    max_surge       = "1"
    max_unavailable = "0"
  }]
}]`)
	onDelete := config(`strategy = [{
  type = "OnDelete"
}]`)
	var initial, afterOnDelete, afterRolling appsv1.DaemonSet

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckDaemonSetDestroy,
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   rolling,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckDaemonSetExists(daemonSetResourceName, &initial),
					testAccCheckDaemonSetObject(&initial, checkStrategyType(appsv1.RollingUpdateDaemonSetStrategyType)),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   onDelete,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(daemonSetResourceName, plancheck.ResourceActionUpdate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckDaemonSetExists(daemonSetResourceName, &afterOnDelete),
					testAccCheckDaemonSetNotRecreated(&initial, &afterOnDelete),
					testAccCheckDaemonSetObject(&afterOnDelete, checkOnDeleteClearsRollingUpdate()),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   rolling,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(daemonSetResourceName, plancheck.ResourceActionUpdate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckDaemonSetExists(daemonSetResourceName, &afterRolling),
					testAccCheckDaemonSetNotRecreated(&afterOnDelete, &afterRolling),
					testAccCheckDaemonSetObject(&afterRolling, checkStrategyType(appsv1.RollingUpdateDaemonSetStrategyType)),
				),
			},
		},
	})
}

func TestAccDaemonSetV1_UpgradeFromSDKV2_CompatibilityMatrix(t *testing.T) {
	testCases := []daemonSetMigrationCase{
		{
			name:            "minimal",
			sdkConfig:       daemonSetSDKConfigMinimal("minimal", busyboxImage),
			frameworkConfig: daemonSetSDKConfigMinimal("minimal", busyboxImage),
			apiCheck:        checkStrategyType(appsv1.RollingUpdateDaemonSetStrategyType),
		},
		{
			name:            "explicit-empty-template-namespace",
			sdkConfig:       withEmptyTemplateNamespace(t, daemonSetSDKConfigMinimal("empty-template-namespace", busyboxImage)),
			frameworkConfig: withEmptyTemplateNamespace(t, daemonSetSDKConfigMinimal("empty-template-namespace", busyboxImage)),
			apiCheck:        checkStrategyType(appsv1.RollingUpdateDaemonSetStrategyType),
		},
		{
			name:            "strategy-empty",
			sdkConfig:       daemonSetSDKConfigWithStrategy("strategy-empty", busyboxImage, "strategy {}"),
			frameworkConfig: daemonSetFrameworkConfigWithStrategy("strategy-empty", busyboxImage, "strategy = [{}]"),
			apiCheck:        checkStrategyType(appsv1.RollingUpdateDaemonSetStrategyType),
		},
		{
			name: "strategy-rolling-update-omitted-values",
			sdkConfig: daemonSetSDKConfigWithStrategy("strategy-rolling-defaults", busyboxImage, `strategy {
  rolling_update {}
}`),
			frameworkConfig: daemonSetFrameworkConfigWithStrategy("strategy-rolling-defaults", busyboxImage, "strategy = [{ rolling_update = [{}] }]"),
			apiCheck:        checkStrategyType(appsv1.RollingUpdateDaemonSetStrategyType),
		},
		{
			name:            "strategy-ondelete",
			sdkConfig:       daemonSetSDKConfigWithStrategy("strategy-ondelete", busyboxImage, "strategy { type = \"OnDelete\" }"),
			frameworkConfig: daemonSetFrameworkConfigWithStrategy("strategy-ondelete", busyboxImage, "strategy = [{ type = \"OnDelete\" }]"),
			apiCheck:        checkStrategyType(appsv1.OnDeleteDaemonSetStrategyType),
		},
		{
			name:            "container-resources",
			sdkConfig:       daemonSetSDKConfigWithContainerExtra("resources", busyboxImage, sdkContainerResourcesBlock),
			frameworkConfig: daemonSetFrameworkConfigWithContainerExtra("resources", busyboxImage, frameworkContainerResourcesAttribute),
			apiCheck:        checkContainerResources("250m", "500m"),
		},
		{
			name:            "image-pull-secrets",
			sdkConfig:       daemonSetSDKConfigWithPodSpecExtra("pull-secrets", busyboxImage, "image_pull_secrets { name = \"registry-secret\" }"),
			frameworkConfig: daemonSetFrameworkConfigWithPodSpecExtra("pull-secrets", busyboxImage, "image_pull_secrets = [{ name = \"registry-secret\" }]"),
			apiCheck:        checkImagePullSecret("registry-secret"),
		},
		{
			name:            "readiness-gate-and-lifecycle-update",
			sdkConfig:       daemonSetSDKConfigWithPodSpecExtra("readiness", busyboxImage, "readiness_gate { condition_type = \"example.com/ready\" }"),
			frameworkConfig: daemonSetFrameworkConfigWithPodSpecExtra("readiness", busyboxImage, "readiness_gate = [{ condition_type = \"example.com/ready\" }]"),
			apiCheck:        checkReadinessGate("example.com/ready"),
			updateConfig:    daemonSetFrameworkConfigWithPodSpecExtra("readiness", agnhostImage, "readiness_gate = [{ condition_type = \"example.com/ready\" }]"),
			updateCheck:     checkContainerImage(agnhostImage),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			name := fmt.Sprintf("tf-framework-daemonset-%s-%s", tc.name, acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))
			sdkConfig := strings.ReplaceAll(tc.sdkConfig, "${NAME}", name)
			frameworkConfig := strings.ReplaceAll(tc.frameworkConfig, "${NAME}", name)
			updateConfig := strings.ReplaceAll(tc.updateConfig, "${NAME}", name)
			if tc.name == "readiness-gate-and-lifecycle-update" {
				// No controller supplies this custom readiness condition.
				sdkConfig = daemonSetNoRolloutConfig(sdkConfig)
				frameworkConfig = daemonSetNoRolloutConfig(frameworkConfig)
				updateConfig = daemonSetNoRolloutConfig(updateConfig)
			}

			var before appsv1.DaemonSet
			var after appsv1.DaemonSet
			var afterUpdate appsv1.DaemonSet

			steps := []resource.TestStep{
				{
					ExternalProviders: map[string]resource.ExternalProvider{
						"kubernetes": {
							VersionConstraint: daemonSetSDKv2ProviderV321,
							Source:            "hashicorp/kubernetes",
						},
					},
					Config: sdkConfig,
					Check:  testAccCheckDaemonSetExists(daemonSetResourceName, &before),
				},
				{
					ProtoV6ProviderFactories: testAccProviderFactories,
					Config:                   frameworkConfig,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
					Check: resource.ComposeAggregateTestCheckFunc(
						testAccCheckDaemonSetExists(daemonSetResourceName, &after),
						testAccCheckDaemonSetNotRecreated(&before, &after),
						testAccCheckDaemonSetSpecUnchanged(&before, &after),
						testAccCheckDaemonSetStateIdentity(daemonSetResourceName, &after),
						testAccCheckDaemonSetObject(&after, tc.apiCheck),
					),
				},
			}

			if updateConfig != "" {
				steps = append(steps, resource.TestStep{
					ProtoV6ProviderFactories: testAccProviderFactories,
					Config:                   updateConfig,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(daemonSetResourceName, plancheck.ResourceActionUpdate)},
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
					Check: resource.ComposeAggregateTestCheckFunc(
						testAccCheckDaemonSetExists(daemonSetResourceName, &afterUpdate),
						testAccCheckDaemonSetNotRecreated(&after, &afterUpdate),
						testAccCheckDaemonSetStateIdentity(daemonSetResourceName, &afterUpdate),
						testAccCheckDaemonSetObject(&afterUpdate, tc.updateCheck),
					),
				})
			}

			resource.ParallelTest(t, resource.TestCase{
				PreCheck:     func() { testAccPreCheck(t) },
				CheckDestroy: testAccCheckDaemonSetDestroy,
				Steps:        steps,
			})
		})
	}
}

func testAccCheckDaemonSetDestroy(state *terraform.State) error {
	client, err := testAccWorkloadClient()
	if err != nil {
		return err
	}

	for _, resourceState := range state.RootModule().Resources {
		if resourceState.Type != "kubernetes_daemon_set_v1" && resourceState.Type != "kubernetes_daemonset" {
			continue
		}

		namespace, name, err := IdParts(resourceState.Primary.ID)
		if err != nil {
			return err
		}
		_, err = client.AppsV1().DaemonSets(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err == nil {
			return fmt.Errorf("daemonset still exists: %s/%s", namespace, name)
		}
		if !apierrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func testAccCheckDaemonSetExists(resourceName string, out *appsv1.DaemonSet) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		resourceState, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}
		if resourceState.Primary.ID == "" {
			return fmt.Errorf("resource %s has no ID", resourceName)
		}

		namespace, name, err := IdParts(resourceState.Primary.ID)
		if err != nil {
			return err
		}

		client, err := testAccWorkloadClient()
		if err != nil {
			return err
		}
		daemonSet, err := client.AppsV1().DaemonSets(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*out = *daemonSet
		return nil
	}
}

func testAccCheckDaemonSetNotRecreated(before, after *appsv1.DaemonSet) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if before == nil || after == nil {
			return fmt.Errorf("daemonset snapshot missing")
		}
		if before.UID != after.UID {
			return fmt.Errorf("daemonset was recreated: before UID %q, after UID %q", before.UID, after.UID)
		}
		return nil
	}
}

func testAccCheckDaemonSetStateIdentity(resourceName string, obj *appsv1.DaemonSet) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		resourceState, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}

		expectedID := fmt.Sprintf("%s/%s", obj.Namespace, obj.Name)
		if resourceState.Primary.ID != expectedID {
			return fmt.Errorf("state ID mismatch: got %q, want %q", resourceState.Primary.ID, expectedID)
		}

		attrs := resourceState.Primary.Attributes
		if got := attrs["metadata.0.uid"]; got != string(obj.UID) {
			return fmt.Errorf("state metadata.0.uid mismatch: got %q, want %q", got, obj.UID)
		}
		if got := attrs["metadata.0.name"]; got != obj.Name {
			return fmt.Errorf("state metadata.0.name mismatch: got %q, want %q", got, obj.Name)
		}
		if got := attrs["metadata.0.namespace"]; got != obj.Namespace {
			return fmt.Errorf("state metadata.0.namespace mismatch: got %q, want %q", got, obj.Namespace)
		}
		if attrs["metadata.0.resource_version"] == "" {
			return fmt.Errorf("state metadata.0.resource_version is empty")
		}
		return nil
	}
}

func testAccCheckDaemonSetObject(obj *appsv1.DaemonSet, check daemonSetObjectCheck) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if obj == nil {
			return fmt.Errorf("daemonset snapshot missing")
		}
		if check == nil {
			return nil
		}
		return check(obj)
	}
}

func daemonSetSDKConfigMinimal(caseName, image string) string {
	return daemonSetSDKConfig(caseName, image, "", "", "")
}

func daemonSetSDKConfigWithStrategy(caseName, image, strategyBlock string) string {
	return daemonSetSDKConfig(caseName, image, "", "", strategyBlock)
}

func daemonSetFrameworkConfigWithStrategy(caseName, image, strategyExpression string) string {
	return daemonSetFrameworkConfig(caseName, image, "", "", strategyExpression)
}

func daemonSetSDKConfigWithContainerExtra(caseName, image, containerExtra string) string {
	return daemonSetSDKConfig(caseName, image, containerExtra, "", "")
}

func daemonSetFrameworkConfigWithContainerExtra(caseName, image, containerExtra string) string {
	return daemonSetFrameworkConfig(caseName, image, containerExtra, "", "")
}

func daemonSetSDKConfigWithPodSpecExtra(caseName, image, podSpecExtra string) string {
	return daemonSetSDKConfig(caseName, image, "", podSpecExtra, "")
}

func daemonSetFrameworkConfigWithPodSpecExtra(caseName, image, podSpecExtra string) string {
	return daemonSetFrameworkConfig(caseName, image, "", podSpecExtra, "")
}

func daemonSetSDKConfig(caseName, image, containerExtra, podSpecExtra, strategy string) string {
	return fmt.Sprintf(`resource "kubernetes_daemon_set_v1" "test" {
  metadata {
    name = "${NAME}"
    labels = {
      case = %q
      app  = "daemonset-migration"
    }
  }

  spec {
    selector {
      match_labels = {
        app = "daemonset-migration"
      }
    }

    template {
      metadata {
        labels = {
          app = "daemonset-migration"
        }
      }

      spec {
        container {
          name    = "app"
          image   = %q
          command = ["sleep", "300"]
%s
        }
        termination_grace_period_seconds = 1
%s
      }
    }
%s
  }
}
`, caseName, image, indentBlock(containerExtra, 10), indentBlock(podSpecExtra, 8), indentBlock(strategy, 4))
}

func daemonSetFrameworkConfig(caseName, image, containerExtra, podSpecExtra, strategy string) string {
	return fmt.Sprintf(`resource "kubernetes_daemon_set_v1" "test" {
  metadata {
    name = "${NAME}"
    labels = {
      case = %q
      app  = "daemonset-migration"
    }
  }

  spec {
    selector {
      match_labels = {
        app = "daemonset-migration"
      }
    }

    template {
      metadata {
        labels = {
          app = "daemonset-migration"
        }
      }

      spec {
        container {
          name    = "app"
          image   = %q
          command = ["sleep", "300"]
%s
        }
        termination_grace_period_seconds = 1
%s
      }
    }
%s
  }
}
`, caseName, image, indentBlock(containerExtra, 10), indentBlock(podSpecExtra, 8), indentBlock(strategy, 4))
}

func indentBlock(input string, spaces int) string {
	if input == "" {
		return ""
	}
	padding := strings.Repeat(" ", spaces)
	lines := strings.Split(input, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines[i] = padding + line
	}
	return strings.Join(lines, "\n")
}

const sdkContainerResourcesBlock = `resources {
  limits = {
    cpu    = "0.5"
    memory = "512Mi"
  }

  requests = {
    cpu    = "250m"
    memory = "50Mi"
  }
}`

const frameworkContainerResourcesAttribute = `resources = [{
  limits = {
    cpu    = "0.5"
    memory = "512Mi"
  }

  requests = {
    cpu    = "250m"
    memory = "50Mi"
  }
}]`

func checkStrategyType(expected appsv1.DaemonSetUpdateStrategyType) daemonSetObjectCheck {
	return func(obj *appsv1.DaemonSet) error {
		if obj.Spec.UpdateStrategy.Type != expected {
			return fmt.Errorf("strategy type mismatch: got %q, want %q", obj.Spec.UpdateStrategy.Type, expected)
		}
		return nil
	}
}

func checkOnDeleteClearsRollingUpdate() daemonSetObjectCheck {
	return func(obj *appsv1.DaemonSet) error {
		if obj.Spec.UpdateStrategy.Type != appsv1.OnDeleteDaemonSetStrategyType {
			return fmt.Errorf("strategy type mismatch: got %q, want %q", obj.Spec.UpdateStrategy.Type, appsv1.OnDeleteDaemonSetStrategyType)
		}
		if obj.Spec.UpdateStrategy.RollingUpdate != nil {
			return fmt.Errorf("OnDelete strategy retained rollingUpdate: %#v", obj.Spec.UpdateStrategy.RollingUpdate)
		}
		return nil
	}
}

func checkContainerResources(expectedRequestCPU, expectedLimitCPU string) daemonSetObjectCheck {
	return func(obj *appsv1.DaemonSet) error {
		if len(obj.Spec.Template.Spec.Containers) == 0 {
			return fmt.Errorf("expected at least one container")
		}
		container := obj.Spec.Template.Spec.Containers[0]
		if got := container.Resources.Requests.Cpu().String(); got != expectedRequestCPU {
			return fmt.Errorf("requests.cpu mismatch: got %q, want %q", got, expectedRequestCPU)
		}
		if got := container.Resources.Limits.Cpu().String(); got != expectedLimitCPU {
			return fmt.Errorf("limits.cpu mismatch: got %q, want %q", got, expectedLimitCPU)
		}
		return nil
	}
}

func checkImagePullSecret(expected string) daemonSetObjectCheck {
	return func(obj *appsv1.DaemonSet) error {
		secrets := obj.Spec.Template.Spec.ImagePullSecrets
		if len(secrets) != 1 {
			return fmt.Errorf("expected 1 image_pull_secret, got %d", len(secrets))
		}
		if got := secrets[0].Name; got != expected {
			return fmt.Errorf("image_pull_secret mismatch: got %q, want %q", got, expected)
		}
		return nil
	}
}

func checkReadinessGate(expected string) daemonSetObjectCheck {
	return func(obj *appsv1.DaemonSet) error {
		gates := obj.Spec.Template.Spec.ReadinessGates
		if len(gates) != 1 {
			return fmt.Errorf("expected 1 readiness_gate, got %d", len(gates))
		}
		if got := string(gates[0].ConditionType); got != expected {
			return fmt.Errorf("readiness_gate mismatch: got %q, want %q", got, expected)
		}
		return nil
	}
}

func checkContainerImage(expected string) daemonSetObjectCheck {
	return func(obj *appsv1.DaemonSet) error {
		if len(obj.Spec.Template.Spec.Containers) == 0 {
			return fmt.Errorf("expected at least one container")
		}
		if got := obj.Spec.Template.Spec.Containers[0].Image; got != expected {
			return fmt.Errorf("container image mismatch: got %q, want %q", got, expected)
		}
		return nil
	}
}
