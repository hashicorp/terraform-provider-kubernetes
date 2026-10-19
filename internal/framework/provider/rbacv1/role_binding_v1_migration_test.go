// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1_test

// ---------------------------------------------------------------------------
// Migration tests for kubernetes_role_binding_v1.
//
// This file covers the *in-place upgrade* path: state written by the released
// SDKv2 kubernetes_role_binding_v1 resource is read back by the local Framework
// resource under the same type name, and the post-upgrade plan must be empty.
//
// No UpgradeState handler is needed because the Framework schema uses
// ListNestedBlock for metadata and other blocks, producing an identical JSON
// state shape to the SDKv2 TypeList{MaxItems:1}; schema_version stays at 0 and
// Terraform reads the existing state directly.
//
// The *cross-type rename* path (moving from the deprecated
// kubernetes_role_binding alias to kubernetes_role_binding_v1 via a moved
// block) is covered separately — together with the MoveState unit tests — in
// role_binding_v1_move_test.go.
// ---------------------------------------------------------------------------

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// migratedRoleBindingProviderVersion is the last released version that ships
// the SDKv2 implementation of kubernetes_role_binding_v1. Used as the
// "from" provider in every migration test.
const migratedRoleBindingProviderVersion = "3.2.1"

// releasedRoleBindingProvider returns the external-provider map pointing at
// the last SDKv2-based release, used for step 1 of every migration test.
func releasedRoleBindingProvider() map[string]resource.ExternalProvider {
	return map[string]resource.ExternalProvider{
		"kubernetes": {
			VersionConstraint: migratedRoleBindingProviderVersion,
			Source:            "hashicorp/kubernetes",
		},
	}
}

// roleBindingMigrationTestCase runs a standard three-step migration test:
//   - step 1: create with the released SDKv2 provider
//   - step 2: same config on the local Framework provider, ExpectEmptyPlan
//   - step 3: re-apply and confirm idempotent (PostApplyPostRefresh empty plan)
//
// extraChecks are asserted after the SDKv2 create step.
func roleBindingMigrationTestCase(t *testing.T, config string, extraChecks ...resource.TestCheckFunc) {
	t.Helper()

	steps := roleBindingMigrationSteps(config)
	steps[0].Check = resource.ComposeAggregateTestCheckFunc(extraChecks...)
	resource.ParallelTest(t, resource.TestCase{Steps: steps})
}

// frameworkChecks run after both Framework steps, independently of SDKv2 checks.
func roleBindingMigrationSteps(config string, frameworkChecks ...resource.TestCheckFunc) []resource.TestStep {
	return []resource.TestStep{
		{
			ExternalProviders: releasedRoleBindingProvider(),
			Config:            config,
		},
		{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Config:                   config,
			Check:                    resource.ComposeAggregateTestCheckFunc(frameworkChecks...),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectEmptyPlan(),
				},
			},
		},
		{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Config:                   config,
			Check:                    resource.ComposeAggregateTestCheckFunc(frameworkChecks...),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PostApplyPostRefresh: []plancheck.PlanCheck{
					plancheck.ExpectEmptyPlan(),
				},
			},
		},
	}
}

// TestAccMigrateRoleBinding_basic verifies that the simplest RoleBinding
// (a single User subject) migrates in place with no diff and that the
// Framework resource identity is populated correctly after migration.
func TestAccMigrateRoleBinding_basic(t *testing.T) {
	name := "tf-acc-migrate:" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "kubernetes_role_binding_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: releasedRoleBindingProvider(),
				Config:            testAccRoleBindingV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.name", "admin"),
					resource.TestCheckResourceAttr(resourceName, "subject.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.name", "notauser"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.kind", "User"),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccRoleBindingV1Config_basic(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				// Verify identity is fully populated after migration.
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(resourceName, map[string]knownvalue.Check{
						"api_version": knownvalue.StringExact("rbac.authorization.k8s.io/v1"),
						"kind":        knownvalue.StringExact("RoleBinding"),
						"name":        knownvalue.StringExact(name),
						"namespace":   knownvalue.StringExact("default"),
					}),
				},
			},
		},
	})
}

// TestAccMigrateRoleBinding_emptySubjectNamespace verifies that state written by
// the SDKv2 provider for a subject with an explicitly empty namespace migrates
// with no diff.
func TestAccMigrateRoleBinding_emptySubjectNamespace(t *testing.T) {
	name := "tf-acc-migrate-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	roleBindingMigrationTestCase(t, testAccRoleBindingV1Config_emptySubjectNamespace(name),
		resource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.0.namespace", ""),
	)
}

// SDKv2 stores the canonical API group despite api_group = "". Framework must
// reconcile it once, then preserve the configured empty value across refreshes.
func TestAccMigrateRoleBinding_emptyUserAPIGroup(t *testing.T) {
	name := "tf-acc-migrate-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "kubernetes_role_binding_v1.test"
	steps := roleBindingMigrationSteps(testAccRoleBindingV1Config_emptyUserAPIGroup(name),
		resource.TestCheckResourceAttr(resourceName, "subject.0.api_group", ""),
	)
	steps[0].Check = resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttr(resourceName, "subject.0.kind", "User"),
		resource.TestCheckResourceAttr(resourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
	)
	steps[1].ConfigPlanChecks = resource.ConfigPlanChecks{
		PreApply: []plancheck.PlanCheck{
			plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
		},
		PostApplyPreRefresh:  []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
		PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
	}
	steps[2].ConfigPlanChecks.PreApply = []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}
	resource.ParallelTest(t, resource.TestCase{
		Steps: steps,
		CheckDestroy: func(_ *terraform.State) error {
			metaFunc, err := sdkv2providerMeta()
			if err != nil {
				return err
			}
			meta := metaFunc()
			clients, ok := meta.(kubernetes.KubeClientsets)
			if !ok {
				return fmt.Errorf("expected kubernetes.KubeClientsets, got %T", meta)
			}
			client, err := clients.MainClientset()
			if err != nil {
				return err
			}
			_, err = client.RbacV1().RoleBindings("default").Get(context.Background(), name, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				return nil
			}
			if err != nil {
				return err
			}
			return fmt.Errorf("RoleBinding default/%s still exists after destroy", name)
		},
	})
}

// TestAccMigrateRoleBinding_complete_name verifies that a RoleBinding with
// metadata.name and multiple subject kinds (User, ServiceAccount, Group)
// migrates with no diff, exercising the computed subject.api_group and
// subject.namespace defaulting logic across the migration boundary.
func TestAccMigrateRoleBinding_complete_name(t *testing.T) {
	name := "tf-acc-migrate:" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	roleBindingMigrationTestCase(t,
		testAccRoleBindingV1Config_updated(name),
		resource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.#", "3"),
		resource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.0.kind", "User"),
		resource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.1.kind", "ServiceAccount"),
		resource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.1.namespace", "kube-system"),
		resource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.1.api_group", ""),
		resource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.2.kind", "Group"),
	)
}

// TestAccMigrateRoleBinding_complete_generateName verifies that a RoleBinding
// created with metadata.generate_name (server-assigned name) migrates without
// a diff. This is the computed-name case: the Framework resource must preserve
// the exact generated name stored by the SDKv2 version.
func TestAccMigrateRoleBinding_complete_generateName(t *testing.T) {
	prefix := "tf-acc-migrate-gen:"

	roleBindingMigrationTestCase(t,
		testAccRoleBindingV1Config_generateName(prefix),
		resource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "metadata.0.generate_name", prefix),
		resource.TestCheckResourceAttrSet("kubernetes_role_binding_v1.test", "metadata.0.name"),
		resource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.#", "1"),
		resource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.0.name", "notauser"),
	)
}

// TestAccMigrateRoleBinding_saSubject verifies migration of a RoleBinding
// whose only subject is a ServiceAccount that omits api_group.
func TestAccMigrateRoleBinding_saSubject(t *testing.T) {
	name := "tf-acc-migrate:" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	roleBindingMigrationTestCase(t,
		testAccRoleBindingV1Config_saSubject(name),
		resource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.#", "1"),
		resource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.0.kind", "ServiceAccount"),
		resource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.0.name", "default"),
		resource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.0.api_group", ""),
	)
}

// TestAccMigrateRoleBinding_groupSubject verifies migration of a RoleBinding
// with a single Group subject (non-empty api_group).
func TestAccMigrateRoleBinding_groupSubject(t *testing.T) {
	name := "tf-acc-migrate:" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	roleBindingMigrationTestCase(t,
		testAccRoleBindingV1Config_groupSubject(name),
		resource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.#", "1"),
		resource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.0.kind", "Group"),
		resource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.0.name", "dev-team"),
		resource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.0.api_group", "rbac.authorization.k8s.io"),
	)
}
