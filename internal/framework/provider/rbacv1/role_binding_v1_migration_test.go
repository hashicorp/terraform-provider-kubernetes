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

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

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
			},
		},
	})
}
