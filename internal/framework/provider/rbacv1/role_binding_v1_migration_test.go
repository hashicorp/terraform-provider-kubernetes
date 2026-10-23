// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

const migratedRoleBindingProviderVersion = "3.2.1"
const preIdentityRoleBindingProviderVersion = "2.37.1"

func releasedRoleBindingProvider() map[string]resource.ExternalProvider {
	return map[string]resource.ExternalProvider{
		"kubernetes": {VersionConstraint: migratedRoleBindingProviderVersion, Source: "hashicorp/kubernetes"},
	}
}

func TestAccMigrateRoleBinding_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-migrate")
	resourceName := "kubernetes_role_binding_v1.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccRoleBindingCheckDestroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: releasedRoleBindingProvider(),
				Config:            testAccRoleBindingV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.name", "admin"),
					resource.TestCheckResourceAttr(resourceName, "subject.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.name", "notauser"),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccRoleBindingV1Config_basic(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestAccMigrateRoleBinding_identity(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-migrate")
	resourceName := "kubernetes_role_binding_v1.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:               func() { testAccPreCheck(t) },
		CheckDestroy:           testAccRoleBindingCheckDestroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)},
		Steps: []resource.TestStep{
			{
				ExternalProviders: releasedRoleBindingProvider(),
				Config:            testAccRoleBindingV1Config_basic(name),
				ConfigStateChecks: []statecheck.StateCheck{testAccRoleBindingExpectIdentity(resourceName, name)},
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccRoleBindingV1Config_basic(name),
				ConfigPlanChecks:         resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
				ConfigStateChecks:        []statecheck.StateCheck{testAccRoleBindingExpectIdentity(resourceName, name)},
			},
		},
	})
}

func TestAccMigrateRoleBinding_preIdentity(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config func(string) string
	}{
		{"name", testAccRoleBindingV1Config_basic},
		{"generateName", testAccRoleBindingV1Config_generateName},
		{"complete", testAccRoleBindingV1Config_updated},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name := acctest.RandomWithPrefix("tf-acc-migrate")
			config := tc.config(name)
			resource.ParallelTest(t, resource.TestCase{
				PreCheck:               func() { testAccPreCheck(t) },
				CheckDestroy:           testAccRoleBindingCheckDestroy,
				TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)},
				Steps: []resource.TestStep{
					{
						ExternalProviders: map[string]resource.ExternalProvider{
							"kubernetes": {VersionConstraint: preIdentityRoleBindingProviderVersion, Source: "hashicorp/kubernetes"},
						},
						Config: config,
					},
					{
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Config:                   config,
						ConfigPlanChecks:         resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
						ConfigStateChecks: []statecheck.StateCheck{
							statecheck.ExpectIdentity("kubernetes_role_binding_v1.test", map[string]knownvalue.Check{
								"name":        knownvalue.StringRegexp(regexp.MustCompile("^" + regexp.QuoteMeta(name))),
								"namespace":   knownvalue.StringExact("default"),
								"kind":        knownvalue.StringExact("RoleBinding"),
								"api_version": knownvalue.StringExact("rbac.authorization.k8s.io/v1"),
							}),
						},
					},
				},
			})
		})
	}
}

func TestAccMigrateRoleBinding_emptySubjectNamespace(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-migrate")
	config := testAccRoleBindingV1Config_emptySubjectNamespace(name)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccRoleBindingCheckDestroy,
		Steps: []resource.TestStep{
			{ExternalProviders: releasedRoleBindingProvider(), Config: config,
				Check: resource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.0.namespace", "")},
			{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}},
		},
	})
}

func TestAccMigrateRoleBinding_emptyUserAPIGroup(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-migrate")
	resourceName := "kubernetes_role_binding_v1.test"
	config := testAccRoleBindingV1Config_emptyUserAPIGroup(name)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccRoleBindingCheckDestroy,
		Steps: []resource.TestStep{
			{ExternalProviders: releasedRoleBindingProvider(), Config: config,
				Check: resource.TestCheckResourceAttr(resourceName, "subject.0.api_group", "rbac.authorization.k8s.io")},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Config: config,
				Check: resource.TestCheckResourceAttr(resourceName, "subject.0.api_group", ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)},
					PostApplyPreRefresh:  []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestAccMigrateRoleBinding_complete_name(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-migrate")
	config := testAccRoleBindingV1Config_updated(name)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccRoleBindingCheckDestroy,
		Steps: []resource.TestStep{
			{ExternalProviders: releasedRoleBindingProvider(), Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.#", "3"),
					resource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.1.namespace", "kube-system"),
					resource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.1.api_group", ""),
				)},
			{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}},
		},
	})
}

func TestAccMigrateRoleBinding_complete_generateName(t *testing.T) {
	prefix := acctest.RandomWithPrefix("tf-acc-migrate") + "-"
	config := testAccRoleBindingV1Config_generateName(prefix)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccRoleBindingCheckDestroy,
		Steps: []resource.TestStep{
			{ExternalProviders: releasedRoleBindingProvider(), Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "metadata.0.generate_name", prefix),
					resource.TestCheckResourceAttrSet("kubernetes_role_binding_v1.test", "metadata.0.name"),
				)},
			{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}},
		},
	})
}

func TestAccMigrateRoleBinding_saSubject(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-migrate")
	config := testAccRoleBindingV1Config_saSubject(name)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccRoleBindingCheckDestroy,
		Steps: []resource.TestStep{
			{ExternalProviders: releasedRoleBindingProvider(), Config: config,
				Check: resource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.0.api_group", "")},
			{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}},
		},
	})
}

func TestAccMigrateRoleBinding_groupSubject(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-migrate")
	config := testAccRoleBindingV1Config_groupSubject(name)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccRoleBindingCheckDestroy,
		Steps: []resource.TestStep{
			{ExternalProviders: releasedRoleBindingProvider(), Config: config,
				Check: resource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.0.api_group", "rbac.authorization.k8s.io")},
			{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}},
		},
	})
}

func TestAccMigrateRoleBinding_noRefresh(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-migrate")
	resourceName := "kubernetes_role_binding_v1.test"
	config := testAccRoleBindingV1Config_basic(name)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:               func() { testAccPreCheck(t) },
		CheckDestroy:           testAccRoleBindingCheckDestroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)},
		AdditionalCLIOptions:   &resource.AdditionalCLIOptions{Plan: resource.PlanOptions{NoRefresh: true}},
		Steps: []resource.TestStep{
			{
				ExternalProviders: releasedRoleBindingProvider(), Config: config,
				ConfigStateChecks: []statecheck.StateCheck{testAccRoleBindingExpectIdentity(resourceName, name)},
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)},
					PostApplyPreRefresh:  []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				ConfigStateChecks: []statecheck.StateCheck{testAccRoleBindingExpectIdentity(resourceName, name)},
			},
		},
	})
}

func TestAccMigrateRoleBinding_updateAndReplace(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-migrate")
	resourceName := "kubernetes_role_binding_v1.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:               func() { testAccPreCheck(t) },
		CheckDestroy:           testAccRoleBindingCheckDestroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)},
		Steps: []resource.TestStep{
			{
				ExternalProviders: releasedRoleBindingProvider(),
				Config:            testAccRoleBindingV1Config_basic(name),
				ConfigStateChecks: []statecheck.StateCheck{testAccRoleBindingExpectIdentity(resourceName, name)},
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccRoleBindingV1Config_basic(name),
				ConfigPlanChecks:         resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
				ConfigStateChecks:        []statecheck.StateCheck{testAccRoleBindingExpectIdentity(resourceName, name)},
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccRoleBindingV1Config_updated(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				ConfigStateChecks: []statecheck.StateCheck{testAccRoleBindingExpectIdentity(resourceName, name)},
				Check:             resource.TestCheckResourceAttr(resourceName, "subject.#", "3"),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccRoleBindingV1Config_clusterRoleRef(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionDestroyBeforeCreate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				ConfigStateChecks: []statecheck.StateCheck{testAccRoleBindingExpectIdentity(resourceName, name)},
				Check:             resource.TestCheckResourceAttr(resourceName, "role_ref.0.kind", "ClusterRole"),
			},
		},
	})
}
