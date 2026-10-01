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
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

// migratedRoleBindingProviderVersion is the last released version that ships
// the SDKv2 implementation of kubernetes_role_binding_v1. Used as the
// "from" provider in every migration test.
const migratedRoleBindingProviderVersion = "3.2.1"

// releasedRoleBindingProvider returns the external-provider map pointing at
// the last SDKv2-based release, used for step 1 of every migration test.
func releasedRoleBindingProvider() map[string]tfresource.ExternalProvider {
	return map[string]tfresource.ExternalProvider{
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
func roleBindingMigrationTestCase(t *testing.T, config string, extraChecks ...tfresource.TestCheckFunc) {
	t.Helper()

	tfresource.Test(t, tfresource.TestCase{
		Steps: []tfresource.TestStep{
			{
				ExternalProviders: releasedRoleBindingProvider(),
				Config:            config,
				Check:             tfresource.ComposeAggregateTestCheckFunc(extraChecks...),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// TestAccMigrateRoleBinding_basic verifies that the simplest RoleBinding
// (a single User subject) migrates in place with no diff and that the
// Framework resource identity is populated correctly after migration.
func TestAccMigrateRoleBinding_basic(t *testing.T) {
	name := "tf-acc-migrate:" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "kubernetes_role_binding_v1.test"

	tfresource.Test(t, tfresource.TestCase{
		Steps: []tfresource.TestStep{
			{
				ExternalProviders: releasedRoleBindingProvider(),
				Config:            testAccRoleBindingV1Config_basic(name),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					tfresource.TestCheckResourceAttr(resourceName, "role_ref.0.name", "admin"),
					tfresource.TestCheckResourceAttr(resourceName, "subject.#", "1"),
					tfresource.TestCheckResourceAttr(resourceName, "subject.0.name", "notauser"),
					tfresource.TestCheckResourceAttr(resourceName, "subject.0.kind", "User"),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccRoleBindingV1Config_basic(name),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
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

// TestAccMigrateRoleBinding_complete_name verifies that a RoleBinding with
// metadata.name and multiple subject kinds (User, ServiceAccount, Group)
// migrates with no diff, exercising the computed subject.api_group and
// subject.namespace defaulting logic across the migration boundary.
func TestAccMigrateRoleBinding_complete_name(t *testing.T) {
	name := "tf-acc-migrate:" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	roleBindingMigrationTestCase(t,
		testAccRoleBindingV1Config_updated(name),
		tfresource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.#", "3"),
		tfresource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.0.kind", "User"),
		tfresource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.1.kind", "ServiceAccount"),
		tfresource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.1.namespace", "kube-system"),
		tfresource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.1.api_group", ""),
		tfresource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.2.kind", "Group"),
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
		tfresource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "metadata.0.generate_name", prefix),
		tfresource.TestCheckResourceAttrSet("kubernetes_role_binding_v1.test", "metadata.0.name"),
		tfresource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.#", "1"),
		tfresource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.0.name", "notauser"),
	)
}

// TestAccMigrateRoleBinding_saSubject verifies migration of a RoleBinding
// whose only subject is a ServiceAccount that omits api_group.
func TestAccMigrateRoleBinding_saSubject(t *testing.T) {
	name := "tf-acc-migrate:" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	roleBindingMigrationTestCase(t,
		testAccRoleBindingV1Config_saSubject(name),
		tfresource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.#", "1"),
		tfresource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.0.kind", "ServiceAccount"),
		tfresource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.0.name", "default"),
		tfresource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.0.api_group", ""),
	)
}

// TestAccMigrateRoleBinding_groupSubject verifies migration of a RoleBinding
// with a single Group subject (non-empty api_group).
func TestAccMigrateRoleBinding_groupSubject(t *testing.T) {
	name := "tf-acc-migrate:" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	roleBindingMigrationTestCase(t,
		testAccRoleBindingV1Config_groupSubject(name),
		tfresource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.#", "1"),
		tfresource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.0.kind", "Group"),
		tfresource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.0.name", "dev-team"),
		tfresource.TestCheckResourceAttr("kubernetes_role_binding_v1.test", "subject.0.api_group", "rbac.authorization.k8s.io"),
	)
}

// TestAccRoleBinding_upgradeFromSDKv2 provisions the resource with the
// last SDKv2 release (state schema version 0) then switches to the local
// Framework provider and asserts zero plan diff — proving the Framework reads
// the SDKv2 state without any upgrade step.
//
// Skipped in -short mode because it downloads from the Terraform registry.
func TestAccRoleBinding_upgradeFromSDKv2(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping registry-dependent upgrade test in -short mode")
	}

	name := acctest.RandomWithPrefix("tf-acc-rb")
	resourceName := "kubernetes_role_binding_v1.test"

	tfresource.ParallelTest(t, tfresource.TestCase{
		Steps: []tfresource.TestStep{
			// Step 1: provision with the last SDKv2 release.
			// Writes state at schema version 0 with TypeList metadata.
			{
				ExternalProviders: map[string]tfresource.ExternalProvider{
					"kubernetes": {
						Source:            "hashicorp/kubernetes",
						VersionConstraint: "3.2.1",
					},
				},
				Config: testAccRoleBindingV1Config_basic(name),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					tfresource.TestCheckResourceAttr(resourceName, "subject.0.name", "notauser"),
				),
			},
			// Step 2: switch to the local Framework provider.
			// ListNestedBlock produces identical state JSON to TypeList{MaxItems:1}
			// so no UpgradeState is needed — plan must be empty.
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccRoleBindingV1Config_basic(name),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
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
