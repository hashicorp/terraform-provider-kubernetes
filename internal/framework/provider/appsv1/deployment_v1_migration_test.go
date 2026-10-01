// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	frameworkprovider "github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider"
	appsv1resource "github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/appsv1"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	appsv1 "k8s.io/api/apps/v1"
)

const (
	deploymentSDKv2ProviderVersion = "3.2.1"
	deploymentSDKv2SchemaV0Version = "2.23.0"
)

func TestAccDeploymentV1_UpgradeFromSDKV2_SevenDimensionMatrix(t *testing.T) {
	cases := []struct {
		name      string
		config    string
		planCheck plancheck.PlanCheck
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
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var before, after appsv1.Deployment
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
						Config: tc.config,
						Check:  testAccCheckKubernetesDeploymentV1Exists("kubernetes_deployment_v1.test", &before),
					},
					{
						ProtoV6ProviderFactories: testAccProviderFactories,
						Config:                   tc.config,
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
						),
					},
				},
			})
		})
	}
}

func TestAccDeploymentV1_MoveFromSDKV2Alias(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-apps-deploy-move")
	versionedConfig := testAccFrameworkDeploymentConfigBasic(name, busyboxImage)
	aliasConfig := strings.Replace(versionedConfig, `"kubernetes_deployment_v1"`, `"kubernetes_deployment"`, 1)
	movedConfig := versionedConfig + `
moved {
  from = kubernetes_deployment.test
  to   = kubernetes_deployment_v1.test
}
`
	var before, after appsv1.Deployment

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
				Check:  testAccCheckKubernetesDeploymentV1Exists("kubernetes_deployment.test", &before),
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
				),
			},
		},
	})
}

func TestAccDeploymentV1_UpgradeFromSDKV2SchemaV0(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-apps-deploy-v0")
	config := testAccFrameworkDeploymentConfigBasic(name, busyboxImage)
	var before, after appsv1.Deployment

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						Source:            "hashicorp/kubernetes",
						VersionConstraint: deploymentSDKv2SchemaV0Version,
					},
				},
				Config: config,
				Check:  testAccCheckKubernetesDeploymentV1Exists("kubernetes_deployment_v1.test", &before),
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists("kubernetes_deployment_v1.test", &after),
					testAccCheckKubernetesDeploymentForceNew(&before, &after, false),
					resource.TestCheckResourceAttrSet("kubernetes_deployment_v1.test", "metadata.0.uid"),
				),
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
          name  = "demo"
          image = "%s"
          command = ["sleep", "3600"]
        }
      }
    }
  }
}
`, name, image)
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
          name  = "demo"
          image = "%s"
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
          name  = "demo"
          image = "%s"
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
          name  = "demo"
          image = "%s"
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
          name  = "demo"
          image = "%s"
          command = ["sleep", "3600"]
        }
      }
    }
  }
}
`, name, image)
}
