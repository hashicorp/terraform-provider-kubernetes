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

	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	frameworkprovider "github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider"
	appsv1resource "github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/appsv1"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8types "k8s.io/apimachinery/pkg/types"
)

const (
	deploymentSDKv2ProviderVersion    = "3.2.1"
	deploymentSDKv2PreIdentityVersion = "2.23.0"
)

func TestAccDeploymentV1_UpgradeFromSDKV2_CompatibilityMatrix(t *testing.T) {
	cases := []struct {
		name            string
		config          string
		frameworkConfig string
		planCheck       plancheck.PlanCheck
	}{
		{
			name:      "basic",
			config:    testAccFrameworkDeploymentConfigBasic(acctest.RandomWithPrefix("tf-apps-deploy"), busyboxImage),
			planCheck: plancheck.ExpectEmptyPlan(),
		},
		{
			name:      "paused-true",
			config:    testAccFrameworkDeploymentConfigPaused(acctest.RandomWithPrefix("tf-apps-deploy"), busyboxImage, true),
			planCheck: plancheck.ExpectEmptyPlan(),
		},
		{
			name:      "paused-false",
			config:    testAccFrameworkDeploymentConfigPaused(acctest.RandomWithPrefix("tf-apps-deploy"), busyboxImage, false),
			planCheck: plancheck.ExpectEmptyPlan(),
		},
		{
			name:      "replicas-zero",
			config:    testAccFrameworkDeploymentConfigReplicas(acctest.RandomWithPrefix("tf-apps-deploy"), busyboxImage, "0"),
			planCheck: plancheck.ExpectEmptyPlan(),
		},
		{
			name:      "replicas-two",
			config:    testAccFrameworkDeploymentConfigReplicas(acctest.RandomWithPrefix("tf-apps-deploy"), busyboxImage, "2"),
			planCheck: plancheck.ExpectEmptyPlan(),
		},
		{
			name:      "wait-for-rollout-false",
			config:    testAccFrameworkDeploymentConfigWaitForRollout(acctest.RandomWithPrefix("tf-apps-deploy"), busyboxImage, false),
			planCheck: plancheck.ExpectEmptyPlan(),
		},
		{
			name:      "metadata-annotations-labels",
			config:    testAccFrameworkDeploymentConfigMetadata(acctest.RandomWithPrefix("tf-apps-deploy"), busyboxImage),
			planCheck: plancheck.ExpectEmptyPlan(),
		},
		{
			name:      "explicit-empty-replicas",
			config:    testAccFrameworkDeploymentConfigReplicas(acctest.RandomWithPrefix("tf-apps-deploy"), busyboxImage, ""),
			planCheck: plancheck.ExpectEmptyPlan(),
		},
		{
			name:      "explicit-empty-collections",
			config:    testAccDeploymentExplicitEmptyConfig(acctest.RandomWithPrefix("tf-apps-deploy-empty")),
			planCheck: deploymentEmptyMapsNormalizationPlanCheck{},
		},
		{
			name: "explicit-empty-template-namespace",
			config: strings.Replace(testAccDeploymentBareContainerConfig(acctest.RandomWithPrefix("tf-apps-deploy-empty-ns")),
				"      metadata {\n", "      metadata {\n        namespace = \"\"\n", 1),
			planCheck: plancheck.ExpectEmptyPlan(),
		},
	}
	for _, strategy := range []string{"RollingUpdate", "Recreate"} {
		name := acctest.RandomWithPrefix("tf-apps-deploy-full")
		cases = append(cases, struct {
			name            string
			config          string
			frameworkConfig string
			planCheck       plancheck.PlanCheck
		}{
			name:            "fully-configured-" + strategy,
			config:          testAccDeploymentMigrationFullConfig(name, strategy, true),
			frameworkConfig: testAccDeploymentMigrationFullConfig(name, strategy, false),
			planCheck:       plancheck.ExpectEmptyPlan(),
		})
	}
	quantityName := acctest.RandomWithPrefix("tf-apps-deploy-quantity")
	cases = append(cases, struct {
		name            string
		config          string
		frameworkConfig string
		planCheck       plancheck.PlanCheck
	}{
		name:            "equivalent-resource-quantities",
		config:          testAccDeploymentMigrationFullConfig(quantityName, "RollingUpdate", true),
		frameworkConfig: strings.ReplaceAll(testAccDeploymentMigrationFullConfig(quantityName, "RollingUpdate", false), `"100m"`, `"0.1"`),
		planCheck:       plancheck.ExpectEmptyPlan(),
	})

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var before, after appsv1.Deployment
			var snapshot deploymentMigrationSnapshot
			frameworkConfig := tc.frameworkConfig
			if frameworkConfig == "" {
				frameworkConfig = tc.config
			}
			var baselinePlanChecks []plancheck.PlanCheck
			// Released 3.2.1 writes spec.paused to Kubernetes but omits it when
			// flattening state. Pin that pre-existing defect without permitting
			// an update (or rollout) during the Framework upgrade.
			baselinePausedDrift := tc.name == "paused-true"
			if baselinePausedDrift {
				baselinePlanChecks = []plancheck.PlanCheck{
					plancheck.ExpectResourceAction("kubernetes_deployment_v1.test", plancheck.ResourceActionUpdate),
				}
			}
			resource.ParallelTest(t, resource.TestCase{
				PreCheck:     func() { testAccPreCheck(t) },
				CheckDestroy: testAccCheckKubernetesDeploymentV1Destroy,
				Steps: []resource.TestStep{
					{
						ExternalProviders: map[string]resource.ExternalProvider{
							"kubernetes": {
								Source:            "hashicorp/kubernetes",
								VersionConstraint: deploymentSDKv2ProviderVersion,
							},
						},
						Config:             tc.config,
						ExpectNonEmptyPlan: baselinePausedDrift,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PostApplyPreRefresh:  baselinePlanChecks,
							PostApplyPostRefresh: baselinePlanChecks,
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							testAccCheckKubernetesDeploymentV1Exists("kubernetes_deployment_v1.test", &before),
							testAccDeploymentMigrationCapture("kubernetes_deployment_v1.test", &snapshot),
							func(state *terraform.State) error {
								if baselinePausedDrift && (!before.Spec.Paused ||
									state.RootModule().Resources["kubernetes_deployment_v1.test"].Primary.Attributes["spec.0.paused"] != "false") {
									return fmt.Errorf("released paused-state defect no longer matches the recorded baseline")
								}
								return nil
							},
						),
					},
					{
						ProtoV6ProviderFactories: testAccProviderFactories,
						Config:                   frameworkConfig,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply:             []plancheck.PlanCheck{tc.planCheck},
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							testAccCheckKubernetesDeploymentV1Exists("kubernetes_deployment_v1.test", &after),
							testAccCheckKubernetesDeploymentForceNew(&before, &after, false),
							resource.TestCheckResourceAttrSet("kubernetes_deployment_v1.test", "metadata.0.uid"),
							resource.TestCheckResourceAttr("kubernetes_deployment_v1.test", "metadata.0.namespace", "default"),
							resource.TestCheckResourceAttr("kubernetes_deployment_v1.test", "spec.0.template.0.spec.0.restart_policy", "Always"),
							testAccDeploymentMigrationUnchanged("kubernetes_deployment_v1.test", &snapshot),
						),
						ConfigStateChecks: deploymentIdentityChecks("kubernetes_deployment_v1.test", &snapshot),
					},
				},
			})
		})
	}
}

func TestAccDeploymentV1_UpgradeFromSDKV2AfterRefresh(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-apps-deploy-refresh")
	config := testAccDeploymentBareContainerConfig(name)
	address := "kubernetes_deployment_v1.test"
	source := map[string]resource.ExternalProvider{
		"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: deploymentSDKv2ProviderVersion},
	}
	var snapshot deploymentMigrationSnapshot
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{ExternalProviders: source, Config: config},
			{
				// An ordinary unchanged SDKv2 apply persists Read's empty values.
				ExternalProviders: source,
				Config:            config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: testAccDeploymentMigrationCapture(address, &snapshot),
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan(), deploymentRefreshNormalizationPlanCheck{}},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan(), deploymentRefreshNormalizationPlanCheck{}},
				},
				Check:             testAccDeploymentMigrationUnchanged(address, &snapshot),
				ConfigStateChecks: deploymentIdentityChecks(address, &snapshot),
			},
		},
	})
}

func TestAccDeploymentV1_MoveFromSDKV2Alias(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-apps-deploy-move")
	versionedConfig := testAccDeploymentMigrationFullConfig(name, "RollingUpdate", false)
	aliasConfig := strings.Replace(testAccDeploymentMigrationFullConfig(name, "RollingUpdate", true), `"kubernetes_deployment_v1"`, `"kubernetes_deployment"`, 1)
	movedConfig := versionedConfig + `
moved {
  from = kubernetes_deployment.test
  to   = kubernetes_deployment_v1.test
}
`
	var before, after appsv1.Deployment
	var snapshot deploymentMigrationSnapshot

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						Source:            "hashicorp/kubernetes",
						VersionConstraint: deploymentSDKv2ProviderVersion,
					},
				},
				Config: aliasConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists("kubernetes_deployment.test", &before),
					testAccDeploymentMigrationCapture("kubernetes_deployment.test", &snapshot),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   movedConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists("kubernetes_deployment_v1.test", &after),
					testAccCheckKubernetesDeploymentForceNew(&before, &after, false),
					resource.TestCheckResourceAttrSet("kubernetes_deployment_v1.test", "metadata.0.uid"),
					testAccDeploymentMigrationUnchanged("kubernetes_deployment_v1.test", &snapshot),
				),
				ConfigStateChecks: deploymentIdentityChecks("kubernetes_deployment_v1.test", &snapshot),
			},
		},
	})
}

func TestAccDeploymentV1_UpgradeFromSDKV2PreIdentity(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-apps-deploy-pre-identity")
	config := testAccDeploymentMigrationFullConfig(name, "RollingUpdate", true)
	var before, after appsv1.Deployment
	var snapshot deploymentMigrationSnapshot

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						Source:            "hashicorp/kubernetes",
						VersionConstraint: deploymentSDKv2PreIdentityVersion,
					},
				},
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists("kubernetes_deployment_v1.test", &before),
					testAccDeploymentMigrationCapture("kubernetes_deployment_v1.test", &snapshot),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   testAccDeploymentMigrationFullConfig(name, "RollingUpdate", false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists("kubernetes_deployment_v1.test", &after),
					testAccCheckKubernetesDeploymentForceNew(&before, &after, false),
					resource.TestCheckResourceAttrSet("kubernetes_deployment_v1.test", "metadata.0.uid"),
					testAccDeploymentMigrationUnchanged("kubernetes_deployment_v1.test", &snapshot),
					resource.TestCheckResourceAttr("kubernetes_deployment_v1.test", "spec.0.template.0.spec.0.container.0.resources.0.requests.cpu", "100m"),
					resource.TestCheckResourceAttr("kubernetes_deployment_v1.test", "spec.0.template.0.spec.0.init_container.0.resources.0.limits.memory", "64Mi"),
				),
				ConfigStateChecks: deploymentIdentityChecks("kubernetes_deployment_v1.test", &snapshot),
			},
		},
	})
}

func TestDeploymentV1RegistrationAndMigrationGuards(t *testing.T) {
	if _, ok := kubernetes.Provider().ResourcesMap["kubernetes_deployment"]; !ok {
		t.Fatal("SDKv2 deprecated deployment alias is not registered")
	}
	if _, ok := kubernetes.Provider().ResourcesMap["kubernetes_deployment_v1"]; ok {
		t.Fatal("SDKv2 versioned deployment must be removed after Framework registration")
	}

	type providerWithResources interface {
		Resources(context.Context) []func() frameworkresource.Resource
	}
	p, ok := frameworkprovider.New("test", nil).(providerWithResources)
	if !ok {
		t.Fatal("Framework provider does not expose resources")
	}
	deploymentCount := 0
	for _, factory := range p.Resources(context.Background()) {
		var metadata frameworkresource.MetadataResponse
		resourceUnderTest := factory()
		resourceUnderTest.Metadata(context.Background(), frameworkresource.MetadataRequest{ProviderTypeName: "kubernetes"}, &metadata)
		if metadata.TypeName != "kubernetes_deployment_v1" {
			continue
		}
		deploymentCount++
		deployment, ok := resourceUnderTest.(*appsv1resource.DeploymentV1)
		if !ok {
			t.Fatalf("deployment factory returned %T", resourceUnderTest)
		}
		var schemaResponse frameworkresource.SchemaResponse
		deployment.Schema(context.Background(), frameworkresource.SchemaRequest{}, &schemaResponse)
		if schemaResponse.Diagnostics.HasError() {
			t.Fatal(schemaResponse.Diagnostics)
		}
		if schemaResponse.Schema.Version != 1 {
			t.Fatalf("schema version=%d want=1", schemaResponse.Schema.Version)
		}
		if got := deployment.UpgradeState(context.Background()); len(got) != 1 || got[0].StateUpgrader == nil {
			t.Fatalf("upgrade state routes=%v, want direct v0-to-v1 upgrader", got)
		}
		if got := deployment.MoveState(context.Background()); len(got) != 1 || got[0].StateMover == nil {
			t.Fatalf("move state routes=%v, want SDKv2 alias mover", got)
		}
	}
	if deploymentCount != 1 {
		t.Fatalf("Framework deployment registrations=%d want=1", deploymentCount)
	}
}

func testAccFrameworkDeploymentConfigBasic(name, image string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
  metadata {
    name = "%s"
    labels = {
      app = "demo"
    }
  }
  spec {
    selector {
      match_labels = {
        app = "demo"
      }
    }
    template {
      metadata {
        labels = {
          app = "demo"
        }
      }
      spec {
        container {
          name    = "demo"
          image   = "%s"
          command = ["sleep", "3600"]
        }
      }
    }
  }
}
`, name, image)
}

func testAccDeploymentExplicitEmptyConfig(name string) string {
	config := testAccDeploymentBareContainerConfig(name)
	config = strings.Replace(config, "  metadata {", "  metadata {\n    annotations = {}", 1)
	config = strings.Replace(config, "      metadata {", "      metadata {\n        annotations = {}", 1)
	config = strings.Replace(config, "      spec {", "      spec {\n        node_selector = {}", 1)
	return strings.Replace(config, "        container {", "        container {\n          args = []\n          command = []", 1)
}

func testAccDeploymentBareContainerConfig(name string) string {
	config := testAccFrameworkDeploymentConfigBasic(name, "registry.k8s.io/pause:3.10")
	return strings.Replace(config, `          command = ["sleep", "3600"]`+"\n", "", 1)
}

func testAccFrameworkDeploymentConfigPaused(name, image string, paused bool) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
  metadata { name = "%s" }
  wait_for_rollout = false
  spec {
    paused = %t
    selector { match_labels = { app = "paused" } }
    template {
      metadata { labels = { app = "paused" } }
      spec {
        container {
          name    = "demo"
          image   = "%s"
          command = ["sleep", "3600"]
        }
      }
    }
  }
}
`, name, paused, image)
}

func testAccFrameworkDeploymentConfigReplicas(name, image, replicas string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
  metadata { name = "%s" }
  spec {
    replicas = "%s"
    selector { match_labels = { app = "replicas" } }
    template {
      metadata { labels = { app = "replicas" } }
      spec {
        container {
          name    = "demo"
          image   = "%s"
          command = ["sleep", "3600"]
        }
      }
    }
  }
}
`, name, replicas, image)
}

func testAccFrameworkDeploymentConfigWaitForRollout(name, image string, wait bool) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
  metadata { name = "%s" }
  spec {
    selector { match_labels = { app = "wait" } }
    template {
      metadata { labels = { app = "wait" } }
      spec {
        container {
          name    = "demo"
          image   = "%s"
          command = ["sleep", "3600"]
        }
      }
    }
  }
  wait_for_rollout = %t
}
`, name, image, wait)
}

func testAccFrameworkDeploymentConfigMetadata(name, image string) string {
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
  metadata {
    name = "%s"
    annotations = {
      "example.com/a" = "1"
    }
    labels = {
      app = "meta"
    }
  }
  spec {
    selector { match_labels = { app = "meta" } }
    template {
      metadata {
        annotations = { "example.com/a" = "1" }
        labels      = { app = "meta" }
      }
      spec {
        container {
          name    = "demo"
          image   = "%s"
          command = ["sleep", "3600"]
        }
      }
    }
  }
}
`, name, image)
}

type deploymentMigrationSnapshot struct {
	deployment appsv1.Deployment
	children   map[string]k8types.UID
	id         string
}

func testAccDeploymentMigrationCapture(address string, snapshot *deploymentMigrationSnapshot) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		if err := testAccCheckKubernetesDeploymentV1Exists(address, &snapshot.deployment)(state); err != nil {
			return err
		}
		snapshot.id = state.RootModule().Resources[address].Primary.ID
		client, err := testAccWorkloadClient()
		if err != nil {
			return err
		}
		if !snapshot.deployment.Spec.Paused {
			if err := retry.RetryContext(context.Background(), 2*time.Minute,
				kubernetes.WaitForDeploymentReplicasForFramework(context.Background(), client, snapshot.deployment.Namespace, snapshot.deployment.Name)); err != nil {
				return err
			}
			if err := testAccCheckKubernetesDeploymentV1Exists(address, &snapshot.deployment)(state); err != nil {
				return err
			}
		}
		replicaSets, err := client.AppsV1().ReplicaSets(snapshot.deployment.Namespace).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return err
		}
		snapshot.children = map[string]k8types.UID{}
		owners := map[k8types.UID]bool{}
		for _, rs := range replicaSets.Items {
			if !metav1.IsControlledBy(&rs, &snapshot.deployment) {
				continue
			}
			owners[rs.UID] = true
			snapshot.children["ReplicaSet/"+rs.Name] = rs.UID
		}
		pods, err := client.CoreV1().Pods(snapshot.deployment.Namespace).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return err
		}
		for _, pod := range pods.Items {
			if owner := metav1.GetControllerOf(&pod); owner != nil && owners[owner.UID] {
				snapshot.children["Pod/"+pod.Name] = pod.UID
			}
		}
		return nil
	}
}

func testAccDeploymentMigrationUnchanged(address string, before *deploymentMigrationSnapshot) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		var after deploymentMigrationSnapshot
		if err := testAccDeploymentMigrationCapture(address, &after)(state); err != nil {
			return err
		}
		if before.id != after.id || before.deployment.UID != after.deployment.UID {
			return fmt.Errorf("deployment identity changed: %s/%s -> %s/%s", before.id, before.deployment.UID, after.id, after.deployment.UID)
		}
		if before.deployment.Generation != after.deployment.Generation {
			return fmt.Errorf("no-op migration changed deployment generation: %d -> %d", before.deployment.Generation, after.deployment.Generation)
		}
		if !reflect.DeepEqual(before.deployment.Spec, after.deployment.Spec) {
			return fmt.Errorf("no-op migration changed the live deployment spec")
		}
		if !reflect.DeepEqual(before.deployment.Labels, after.deployment.Labels) || !reflect.DeepEqual(before.deployment.Annotations, after.deployment.Annotations) {
			return fmt.Errorf("no-op migration changed live deployment metadata")
		}
		if !reflect.DeepEqual(before.children, after.children) {
			return fmt.Errorf("no-op migration changed ReplicaSet/Pod identities: before=%v after=%v", before.children, after.children)
		}
		return nil
	}
}

type deploymentMigrationIdentityCheck struct {
	address  string
	snapshot *deploymentMigrationSnapshot
}

func (check deploymentMigrationIdentityCheck) CheckState(ctx context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	statecheck.ExpectIdentity(check.address, map[string]knownvalue.Check{
		"namespace":   knownvalue.StringExact(check.snapshot.deployment.Namespace),
		"name":        knownvalue.StringExact(check.snapshot.deployment.Name),
		"api_version": knownvalue.StringExact("apps/v1"),
		"kind":        knownvalue.StringExact("Deployment"),
	}).CheckState(ctx, req, resp)
}

func deploymentIdentityChecks(address string, snapshot *deploymentMigrationSnapshot) []statecheck.StateCheck {
	return []statecheck.StateCheck{deploymentMigrationIdentityCheck{address: address, snapshot: snapshot}}
}

func testAccDeploymentMigrationFullConfig(name, strategy string, legacy bool) string {
	resources := `resources = [{
          limits = { cpu = "250m", memory = "64Mi" }
          requests = { cpu = "100m", memory = "32Mi" }
        }]`
	strategyConfig := `strategy = [{ type = "Recreate" }]`
	if strategy == "RollingUpdate" {
		strategyConfig = `strategy = [{
      type = "RollingUpdate"
      rolling_update = [{ max_surge = "50%", max_unavailable = "0" }]
    }]`
	}
	if legacy {
		resources = strings.Replace(resources, "resources = [{", "resources {", 1)
		resources = strings.Replace(resources, "}]", "}", 1)
		strategyConfig = `strategy { type = "Recreate" }`
		if strategy == "RollingUpdate" {
			strategyConfig = `strategy {
      type = "RollingUpdate"
      rolling_update {
        max_surge = "50%"
        max_unavailable = "0"
      }
    }`
		}
	}
	return fmt.Sprintf(`resource "kubernetes_deployment_v1" "test" {
  metadata {
    name        = "%[1]s"
    labels      = { app = "%[1]s", managed = "true" }
    annotations = { "example.com/owner" = "migration" }
  }
  spec {
    replicas                  = "2"
    min_ready_seconds         = 1
    progress_deadline_seconds = 120
    revision_history_limit    = 2
    paused                    = false
    selector { match_labels = { app = "%[1]s" } }
    %[3]s
    template {
      metadata {
        labels      = { app = "%[1]s", tier = "worker" }
        annotations = { "example.com/template" = "preserved" }
      }
      spec {
        automount_service_account_token  = false
        termination_grace_period_seconds = 1
        dns_policy                       = "ClusterFirst"
        restart_policy                   = "Always"
        security_context { run_as_user = "1000" }
        init_container {
          name    = "prepare"
          image   = "%[2]s"
          command = ["sh", "-c", "touch /work/ready"]
          %[4]s
          volume_mount {
            name       = "work"
            mount_path = "/work"
          }
        }
        container {
          name              = "main"
          image             = "%[2]s"
          command           = ["sh", "-c", "test -f /work/ready && sleep 3600"]
          image_pull_policy = "IfNotPresent"
          %[4]s
          env {
            name  = "DEPLOYMENT_NAME"
            value = "%[1]s"
          }
          env {
            name = "POD_NAME"
            value_from {
              field_ref { field_path = "metadata.name" }
            }
          }
          port {
            name           = "metrics"
            container_port = 8080
          }
          security_context { allow_privilege_escalation = false }
          volume_mount {
            name       = "work"
            mount_path = "/work"
          }
        }
        volume {
          name = "work"
          empty_dir { medium = "Memory" }
        }
      }
    }
  }
}
`, name, busyboxImage, strategyConfig, resources)
}
