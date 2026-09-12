// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1_test

import (
	"fmt"
	"os/exec"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	rbacv1 "github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/rbacv1"
)

// compile-time check: RoleBindingV1 satisfies resource.Resource.
var _ resource.Resource = (*rbacv1.RoleBindingV1)(nil)

// TestAccRoleBindingV1_basic creates a RoleBinding with a single User subject,
// verifies all computed fields are populated, and confirms import round-trips
// without drift.
func TestAccRoleBindingV1_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-rb")
	resourceName := "kubernetes_role_binding_v1.test"

	tfresource.ParallelTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: testAccRoleBindingV1Config_basic(name),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.namespace", "default"),
					tfresource.TestCheckResourceAttr(resourceName, "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					tfresource.TestCheckResourceAttr(resourceName, "role_ref.0.kind", "Role"),
					tfresource.TestCheckResourceAttr(resourceName, "role_ref.0.name", "admin"),
					tfresource.TestCheckResourceAttr(resourceName, "subject.#", "1"),
					tfresource.TestCheckResourceAttr(resourceName, "subject.0.kind", "User"),
					tfresource.TestCheckResourceAttr(resourceName, "subject.0.name", "notauser"),
					tfresource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					tfresource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
				),
			},
			// Import by "namespace/name" — verify no drift after import.
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     "default/" + name,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"metadata.0.resource_version",
					"metadata.0.generation",
				},
			},
		},
	})
}

// TestAccRoleBindingV1_update verifies that mutable fields — subjects, labels,
// and annotations — can be changed in-place without destroy/recreate.
func TestAccRoleBindingV1_update(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-rb")
	resourceName := "kubernetes_role_binding_v1.test"

	tfresource.ParallelTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			// Step 1: create with single subject.
			{
				Config: testAccRoleBindingV1Config_basic(name),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr(resourceName, "subject.#", "1"),
				),
			},
			// Step 2: add two more subjects and metadata — must be in-place update.
			{
				Config: testAccRoleBindingV1Config_updated(name),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr(resourceName, "subject.#", "3"),
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.labels.managed-by", "terraform"),
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.example.com/note", "updated"),
				),
			},
			// Step 3: revert to basic — verify subjects removed cleanly.
			{
				Config: testAccRoleBindingV1Config_basic(name),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr(resourceName, "subject.#", "1"),
					tfresource.TestCheckNoResourceAttr(resourceName, "metadata.0.labels.managed-by"),
				),
			},
		},
	})
}

// TestAccRoleBindingV1_generateName creates a RoleBinding using generate_name
// so the server assigns the full name with a unique suffix.
func TestAccRoleBindingV1_generateName(t *testing.T) {
	resourceName := "kubernetes_role_binding_v1.test"

	tfresource.ParallelTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: testAccRoleBindingV1Config_generateName("tf-acc-rb-"),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttrSet(resourceName, "metadata.0.name"),
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.generate_name", "tf-acc-rb-"),
					tfresource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
				),
			},
		},
	})
}

// TestAccRoleBindingV1_saSubject verifies that a ServiceAccount subject works
// correctly, including the empty api_group and explicit namespace.
func TestAccRoleBindingV1_saSubject(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-rb")
	resourceName := "kubernetes_role_binding_v1.test"

	tfresource.ParallelTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: testAccRoleBindingV1Config_saSubject(name),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr(resourceName, "subject.#", "1"),
					tfresource.TestCheckResourceAttr(resourceName, "subject.0.kind", "ServiceAccount"),
					tfresource.TestCheckResourceAttr(resourceName, "subject.0.name", "default"),
					tfresource.TestCheckResourceAttr(resourceName, "subject.0.api_group", ""),
				),
			},
		},
	})
}

// TestAccRoleBindingV1_groupSubject verifies that a Group subject works correctly.
func TestAccRoleBindingV1_groupSubject(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-rb")
	resourceName := "kubernetes_role_binding_v1.test"

	tfresource.ParallelTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: testAccRoleBindingV1Config_groupSubject(name),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr(resourceName, "subject.#", "1"),
					tfresource.TestCheckResourceAttr(resourceName, "subject.0.kind", "Group"),
					tfresource.TestCheckResourceAttr(resourceName, "subject.0.name", "dev-team"),
					tfresource.TestCheckResourceAttr(resourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
				),
			},
		},
	})
}

// TestAccRoleBindingV1_roleRefRequiresReplace verifies that changing role_ref
// destroys the existing RoleBinding and creates a new one (RequiresReplace).
func TestAccRoleBindingV1_roleRefRequiresReplace(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-rb")
	resourceName := "kubernetes_role_binding_v1.test"

	tfresource.ParallelTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: testAccRoleBindingV1Config_basic(name),
				Check:  tfresource.TestCheckResourceAttr(resourceName, "role_ref.0.kind", "Role"),
			},
			// Changing role_ref.kind must trigger destroy + create, not in-place update.
			{
				Config: testAccRoleBindingV1Config_clusterRoleRef(name),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: tfresource.TestCheckResourceAttr(resourceName, "role_ref.0.kind", "ClusterRole"),
			},
		},
	})
}

// TestAccRoleBindingV1_disappears verifies that if the RoleBinding is deleted
// outside Terraform (e.g. kubectl delete), the next plan detects it is gone
// and proposes to recreate it.
func TestAccRoleBindingV1_disappears(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-rb")
	resourceName := "kubernetes_role_binding_v1.test"

	tfresource.ParallelTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			// Step 1: create the resource.
			{
				Config: testAccRoleBindingV1Config_basic(name),
				Check:  tfresource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
			},
			// Step 2: delete out-of-band, then refresh state.
			// RefreshState: true triggers a Read without applying config.
			// Read returns 404 → provider calls RemoveResource → state cleared.
			// The subsequent plan (ExpectNonEmptyPlan) then shows a create.
			{
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				PreConfig: func() {
					_ = exec.Command(
						"kubectl", "delete", "rolebinding", name,
						"-n", "default", "--ignore-not-found",
					).Run()
				},
			},
		},
	})
}

// TestAccRoleBindingV1_upgradeFromSDKv2 provisions the resource with the
// last SDKv2 release (state schema version 0) then switches to the local
// Framework provider and asserts zero plan diff — proving the Framework reads
// the SDKv2 state without any upgrade step.
//
// Skipped in -short mode because it downloads from the Terraform registry.
func TestAccRoleBindingV1_upgradeFromSDKv2(t *testing.T) {
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

// TestAccRoleBindingV1_moved provisions the deprecated kubernetes_role_binding
// with the last SDKv2 release then uses a moved block to migrate state to
// kubernetes_role_binding_v1 with the Framework provider. The plan must be
// empty — proving MoveState translates the state without drift.
//
// Skipped in -short mode because it downloads from the Terraform registry.
func TestAccRoleBindingV1_moved(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping registry-dependent moved-block test in -short mode")
	}

	name := acctest.RandomWithPrefix("tf-acc-rb")

	tfresource.ParallelTest(t, tfresource.TestCase{
		Steps: []tfresource.TestStep{
			// Step 1: provision kubernetes_role_binding (deprecated) with the
			// last SDKv2 release. Writes state at schema version 0.
			{
				ExternalProviders: map[string]tfresource.ExternalProvider{
					"kubernetes": {
						Source:            "hashicorp/kubernetes",
						VersionConstraint: "3.2.1",
					},
				},
				Config: testAccRoleBindingConfig_deprecated(name),
			},
			// Step 2: add a moved block and switch to the Framework provider.
			// MoveState translates kubernetes_role_binding → kubernetes_role_binding_v1.
			// Plan must be empty — no destroy, no create.
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccRoleBindingV1Config_movedFrom(name),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// ── HCL config helpers ────────────────────────────────────────────────────────

func testAccRoleBindingV1Config_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = "default"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }

  subject {
    kind      = "User"
    name      = "notauser"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, name)
}

func testAccRoleBindingV1Config_updated(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = "default"
    labels = {
      managed-by = "terraform"
    }
    annotations = {
      "example.com/note" = "updated"
    }
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }

  subject {
    kind      = "User"
    name      = "notauser"
    api_group = "rbac.authorization.k8s.io"
  }

  subject {
    kind      = "ServiceAccount"
    name      = "default"
    namespace = "kube-system"
    api_group = ""
  }

  subject {
    kind      = "Group"
    name      = "system:masters"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, name)
}

func testAccRoleBindingV1Config_generateName(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding_v1" "test" {
  metadata {
    generate_name = %[1]q
    namespace     = "default"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }

  subject {
    kind      = "User"
    name      = "notauser"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, prefix)
}

func testAccRoleBindingV1Config_saSubject(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = "default"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }

  subject {
    kind      = "ServiceAccount"
    name      = "default"
    namespace = "default"
    api_group = ""
  }
}
`, name)
}

func testAccRoleBindingV1Config_groupSubject(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = "default"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }

  subject {
    kind      = "Group"
    name      = "dev-team"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, name)
}

func testAccRoleBindingV1Config_clusterRoleRef(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = "default"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = "cluster-admin"
  }

  subject {
    kind      = "User"
    name      = "notauser"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, name)
}

// testAccRoleBindingConfig_deprecated uses the old resource type name —
// the one still registered in the SDKv2 provider for backwards compatibility.
func testAccRoleBindingConfig_deprecated(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding" "test" {
  metadata {
    name      = %[1]q
    namespace = "default"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }

  subject {
    kind      = "User"
    name      = "notauser"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, name)
}

// testAccRoleBindingV1Config_movedFrom contains the moved block that migrates
// kubernetes_role_binding.test → kubernetes_role_binding_v1.test.
func testAccRoleBindingV1Config_movedFrom(name string) string {
	return fmt.Sprintf(`
moved {
  from = kubernetes_role_binding.test
  to   = kubernetes_role_binding_v1.test
}

resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = "default"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }

  subject {
    kind      = "User"
    name      = "notauser"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, name)
}
