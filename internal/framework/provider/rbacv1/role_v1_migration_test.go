// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

const roleSDKv2ProviderVersion = "3.2.1"

func TestAccRole_movedFromAlias_nameAndGenerateName(t *testing.T) {
	name := "tf-acc-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "kubernetes_role_v1.test"
	config := testAccRoleConfig_nameAndGenerateName(name)
	aliasConfig := strings.Replace(config, `"kubernetes_role_v1"`, `"kubernetes_role"`, 1)
	moveConfig := config + `
moved {
  from = kubernetes_role.test
  to   = kubernetes_role_v1.test
}
`
	updatedConfig := strings.Replace(moveConfig, `verbs      = ["get"]`, `verbs      = ["get", "list"]`, 1)
	var uid string

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		Steps: []resource.TestStep{
			{
				Config: aliasConfig,
				Check:  testAccRoleCheckExists("kubernetes_role.test", &uid),
			},
			{
				Config: moveConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccRoleCheckExists(resourceName, &uid),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.generate_name", name),
				),
			},
			{
				Config: updatedConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccRoleCheckExists(resourceName, &uid),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.verbs.*", "list"),
				),
			},
			{
				Config: updatedConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func testAccRoleMigration(t *testing.T, config string, checks ...resource.TestCheckFunc) {
	t.Helper()

	var uid string
	resourceName := "kubernetes_role_v1.test"
	checks = append([]resource.TestCheckFunc{
		testAccRoleCheckExists(resourceName, &uid),
		resource.TestCheckResourceAttr(resourceName, "metadata.0.namespace", "default"),
		resource.TestCheckResourceAttr(resourceName, "rule.#", "1"),
		resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.resources.*", "pods"),
	}, checks...)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccRoleCheckDestroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						VersionConstraint: roleSDKv2ProviderVersion,
						Source:            "hashicorp/kubernetes",
					},
				},
				Config: config,
				Check:  testAccRoleCheckExists(resourceName, &uid),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(checks...),
			},
		},
	})
}

func TestAccRole_UpgradeFromSDKv2_scenarios(t *testing.T) {
	resourceName := "kubernetes_role_v1.test"
	for _, scenario := range []struct {
		name          string
		generatedName bool
		metadata      string
		resourceNames string
		checks        []resource.TestCheckFunc
	}{
		{name: "minimal"},
		{name: "generateName", generatedName: true},
		{
			name:     "annotations",
			metadata: `annotations = { note = "retained" }`,
			checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.note", "retained"),
				resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
			},
		},
		{
			name:     "labels",
			metadata: `labels = { team = "platform" }`,
			checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.team", "platform"),
				resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
			},
		},
		{
			name:     "declaredInternalLabel",
			metadata: `labels = { "example.kubernetes.io/owner" = "terraform" }`,
			checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.example.kubernetes.io/owner", "terraform"),
			},
		},
		{
			name:     "declaredInternalAnnotation",
			metadata: `annotations = { "example.kubernetes.io/owner" = "terraform" }`,
			checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.example.kubernetes.io/owner", "terraform"),
			},
		},
		{
			name: "emptyValuesName",
			metadata: `labels = {}
    annotations = {}`,
			resourceNames: `resource_names = []`,
		},
		{
			name:          "emptyValuesGeneratedName",
			generatedName: true,
			metadata: `labels = {}
    annotations = {}`,
			resourceNames: `resource_names = []`,
		},
		{
			name: "completeName",
			metadata: `labels = { team = "platform" }
    annotations = { note = "retained" }`,
			resourceNames: `resource_names = ["one", "two"]`,
		},
		{
			name:          "completeGeneratedName",
			generatedName: true,
			metadata: `labels = { team = "platform" }
    annotations = { note = "retained" }`,
			resourceNames: `resource_names = ["one", "two"]`,
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			name := "tf-migration-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
			naming := fmt.Sprintf("name = %q", name)
			checks := append([]resource.TestCheckFunc{}, scenario.checks...)
			if scenario.generatedName {
				prefix := name + "-"
				naming = fmt.Sprintf("generate_name = %q", prefix)
				checks = append(checks,
					resource.TestCheckResourceAttr(resourceName, "metadata.0.generate_name", prefix),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.name"),
				)
			} else {
				checks = append(checks, resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name))
			}
			switch scenario.name {
			case "minimal", "generateName", "emptyValuesName", "emptyValuesGeneratedName":
				checks = append(checks,
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resource_names.#", "0"),
				)
			case "completeName", "completeGeneratedName":
				checks = append(checks,
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.team", "platform"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.note", "retained"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resource_names.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.resource_names.*", "one"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.resource_names.*", "two"),
				)
			}
			config := fmt.Sprintf(`
resource "kubernetes_role_v1" "test" {
  metadata {
    %s
    %s
  }
  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["get", "list"]
    %s
  }
}
`, naming, scenario.metadata, scenario.resourceNames)
			testAccRoleMigration(t, config, checks...)
		})
	}
}

func TestAccRole_UpgradeFromSDKv2(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_role_v1.test"
	var uid string

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccRoleCheckDestroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						VersionConstraint: roleSDKv2ProviderVersion,
						Source:            "hashicorp/kubernetes",
					},
				},
				Config: testAccRoleConfig_basic(name),
				Check: resource.ComposeTestCheckFunc(
					testAccRoleCheckExists(resourceName, &uid),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "rule.#", "2"),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccRoleConfig_basic(name),
				Check:                    testAccRoleCheckExists(resourceName, &uid),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccRoleConfig_modified(name),
				Check: resource.ComposeTestCheckFunc(
					testAccRoleCheckExists(resourceName, &uid),
					resource.TestCheckResourceAttr(resourceName, "rule.#", "1"),
				),
			},
		},
	})
}

func TestAccRole_UpgradeFromSDKv2_nullMetadata(t *testing.T) {
	for _, sourceType := range []string{"kubernetes_role_v1", "kubernetes_role"} {
		t.Run(sourceType, func(t *testing.T) {
			name := "tf-acc-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
			address := "kubernetes_role_v1.test"
			config := testAccRoleConfig_nullMetadata(name, "null")
			sourceConfig := strings.Replace(config, `"kubernetes_role_v1"`, fmt.Sprintf("%q", sourceType), 1)
			if sourceType == "kubernetes_role" {
				config += `
moved {
  from = kubernetes_role.test
  to   = kubernetes_role_v1.test
}
`
			}
			var uid, resourceVersion string
			checkResourceVersion := func(value string) error {
				if resourceVersion != "" && value != resourceVersion {
					return fmt.Errorf("migration modified the Role: resource version changed from %q to %q", resourceVersion, value)
				}
				resourceVersion = value
				return nil
			}

			resource.ParallelTest(t, resource.TestCase{
				CheckDestroy: testAccRoleCheckDestroy,
				TerraformVersionChecks: []tfversion.TerraformVersionCheck{
					tfversion.SkipBelow(tfversion.Version1_8_0),
				},
				Steps: []resource.TestStep{
					{
						ExternalProviders: map[string]resource.ExternalProvider{
							"kubernetes": {
								VersionConstraint: roleSDKv2ProviderVersion,
								Source:            "hashicorp/kubernetes",
							},
						},
						Config: sourceConfig,
						Check: resource.ComposeAggregateTestCheckFunc(
							testAccRoleCheckExists(sourceType+".test", &uid),
							resource.TestCheckResourceAttrWith(sourceType+".test", "metadata.0.resource_version", checkResourceVersion),
						),
					},
					{
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Config:                   config,
						ConfigStateChecks:        testAccRoleNullMetadataStateChecks(address),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{
								plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate),
							},
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							testAccRoleCheckExists(address, &uid),
							testAccRoleCheckNullMetadata(address),
							resource.TestCheckResourceAttrWith(address, "metadata.0.resource_version", checkResourceVersion),
						),
					},
					{
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Config:                   config,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
					},
					{
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Config:                   strings.Replace(config, `verbs      = ["get"]`, `verbs      = ["get", "list"]`, 1),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{
								plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate),
							},
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							testAccRoleCheckExists(address, &uid),
							testAccRoleCheckNullMetadata(address),
							resource.TestCheckTypeSetElemAttr(address, "rule.0.verbs.*", "list"),
						),
					},
				},
			})
		})
	}
}

func TestAccRole_movedFromAlias(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	aliasResource := "kubernetes_role.test"
	v1Resource := "kubernetes_role_v1.test"
	var uid string

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccRoleCheckDestroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						VersionConstraint: roleSDKv2ProviderVersion,
						Source:            "hashicorp/kubernetes",
					},
				},
				Config: testAccRoleAliasConfig(name),
				Check: resource.ComposeTestCheckFunc(
					testAccRoleCheckExists(aliasResource, &uid),
					resource.TestCheckResourceAttr(aliasResource, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(aliasResource, "metadata.0.uid"),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccRoleMoveConfig(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					testAccRoleCheckExists(v1Resource, &uid),
					resource.TestCheckResourceAttrSet(v1Resource, "metadata.0.uid"),
					resource.TestCheckResourceAttr(v1Resource, "metadata.0.name", name),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccRoleMoveUpdatedConfig(name),
				Check: resource.ComposeTestCheckFunc(
					testAccRoleCheckExists(v1Resource, &uid),
					resource.TestCheckResourceAttr(v1Resource, "rule.#", "1"),
				),
			},
		},
	})
}

func testAccRoleAliasConfig(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role" "test" {
  metadata {
    name      = %[1]q
    namespace = "default"
  }

  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["get", "list"]
  }

  rule {
    api_groups = ["apps"]
    resources  = ["deployments"]
    verbs      = ["get"]
  }
}
`, name)
}

func testAccRoleMoveConfig(name string) string {
	return fmt.Sprintf(`
moved {
  from = kubernetes_role.test
  to   = kubernetes_role_v1.test
}

resource "kubernetes_role_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = "default"
  }

  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["get", "list"]
  }

  rule {
    api_groups = ["apps"]
    resources  = ["deployments"]
    verbs      = ["get"]
  }
}
`, name)
}

func testAccRoleMoveUpdatedConfig(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = "default"
  }

  rule {
    api_groups = ["batch"]
    resources  = ["jobs"]
    verbs      = ["get", "list", "watch"]
  }
}
`, name)
}
