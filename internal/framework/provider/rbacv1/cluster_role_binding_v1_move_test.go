// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

// ---------------------------------------------------------------------------
// MoveState tests — cross-type rename via moved { } block
//
// These tests exercise the ResourceWithMoveState implementation added to
// kubernetes_cluster_role_binding_v1. The scenario is:
//
//  1. The practitioner has existing state for the deprecated resource type
//     kubernetes_cluster_role_binding (bare name, SDKv2 only).
//  2. They add a moved { } block in their configuration to rename it to
//     kubernetes_cluster_role_binding_v1 (the Framework resource).
//  3. Terraform calls MoveResourceState on the new provider.
//  4. The Framework MoveState implementation copies state directly — schemas
//     are identical because ClusterRoleBinding is non-namespaced.
//  5. The post-move plan must be empty (no destroy/recreate).
// ---------------------------------------------------------------------------

// TestAccMoveClusterRoleBinding_basic verifies that a binding created
// as kubernetes_cluster_role_binding can be moved to
// kubernetes_cluster_role_binding_v1 via a moved block with no diff.
func TestAccMoveClusterRoleBinding_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		CheckDestroy: testAccKubernetesClusterRoleBindingV1Destroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		Steps: []resource.TestStep{
			{
				// Step 1 — create with the released SDKv2 provider using the
				// deprecated kubernetes_cluster_role_binding resource type.
				ExternalProviders: releasedKubernetesProvider(),
				Config:            testAccKubernetesClusterRoleBindingConfig_deprecatedType(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"kubernetes_cluster_role_binding.test",
						"metadata.0.name", name,
					),
					resource.TestCheckResourceAttr(
						"kubernetes_cluster_role_binding.test",
						"subject.#", "1",
					),
				),
			},
			{
				// Step 2 — switch to the local Framework provider with a
				// moved block renaming the resource type. The plan must be
				// empty: MoveState copies state in place with no changes.
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesClusterRoleBindingConfig_movedToV1(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						clusterRoleBindingResourceName,
						"metadata.0.name", name,
					),
					resource.TestCheckResourceAttr(
						clusterRoleBindingResourceName,
						"subject.#", "1",
					),
					resource.TestCheckResourceAttr(
						clusterRoleBindingResourceName,
						"role_ref.0.name", "cluster-admin",
					),
				),
			},
			{
				// Step 3 — idempotency after the move: plan must still be
				// empty on the Framework provider.
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesClusterRoleBindingConfig_movedToV1(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// TestAccMoveClusterRoleBinding_complete_name verifies the move
// with a complete binding using metadata.name and multiple subjects of different kinds,
// exercising the computed api_group and namespace defaulting across the move boundary.
func TestAccMoveClusterRoleBinding_complete_name(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		CheckDestroy: testAccKubernetesClusterRoleBindingV1Destroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		Steps: []resource.TestStep{
			{
				ExternalProviders: releasedKubernetesProvider(),
				Config:            testAccKubernetesClusterRoleBindingConfig_deprecatedTypeMultiSubject(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"kubernetes_cluster_role_binding.test",
						"subject.#", "3",
					),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesClusterRoleBindingConfig_movedToV1MultiSubject(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						clusterRoleBindingResourceName,
						"subject.#", "3",
					),
					resource.TestCheckResourceAttr(
						clusterRoleBindingResourceName,
						"subject.0.kind", "User",
					),
					resource.TestCheckResourceAttr(
						clusterRoleBindingResourceName,
						"subject.1.kind", "ServiceAccount",
					),
					resource.TestCheckResourceAttr(
						clusterRoleBindingResourceName,
						"subject.2.kind", "Group",
					),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesClusterRoleBindingConfig_movedToV1MultiSubject(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// TestAccMoveClusterRoleBinding_complete_generateName verifies that a binding
// created with metadata.generate_name can be moved to kubernetes_cluster_role_binding_v1
// via a moved block with no diff, ensuring computed server-assigned names survive the move.
func TestAccMoveClusterRoleBinding_complete_generateName(t *testing.T) {
	prefix := "tf-acc-test-gen:"

	resource.Test(t, resource.TestCase{
		CheckDestroy: testAccKubernetesClusterRoleBindingV1Destroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		Steps: []resource.TestStep{
			{
				ExternalProviders: releasedKubernetesProvider(),
				Config:            testAccKubernetesClusterRoleBindingConfig_deprecatedTypeGenerateName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"kubernetes_cluster_role_binding.test",
						"metadata.0.generate_name", prefix,
					),
					resource.TestCheckResourceAttrSet(
						"kubernetes_cluster_role_binding.test",
						"metadata.0.name",
					),
					resource.TestCheckResourceAttr(
						"kubernetes_cluster_role_binding.test",
						"subject.#", "3",
					),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesClusterRoleBindingConfig_movedToV1GenerateName(prefix),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						clusterRoleBindingResourceName,
						"metadata.0.generate_name", prefix,
					),
					resource.TestCheckResourceAttrSet(
						clusterRoleBindingResourceName,
						"metadata.0.name",
					),
					resource.TestCheckResourceAttr(
						clusterRoleBindingResourceName,
						"subject.#", "3",
					),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesClusterRoleBindingConfig_movedToV1GenerateName(prefix),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// testAccKubernetesClusterRoleBindingConfig_deprecatedType returns HCL that
// creates a ClusterRoleBinding using the deprecated
// kubernetes_cluster_role_binding resource type (SDKv2 bare name).
func testAccKubernetesClusterRoleBindingConfig_deprecatedType(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_cluster_role_binding" "test" {
  metadata {
    name = %q
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

// testAccKubernetesClusterRoleBindingConfig_movedToV1 returns HCL that moves
// the kubernetes_cluster_role_binding.test resource to
// kubernetes_cluster_role_binding_v1.test using a moved { } block.
// The resource configuration itself is identical — only the type changes.
func testAccKubernetesClusterRoleBindingConfig_movedToV1(name string) string {
	return fmt.Sprintf(`
moved {
  from = kubernetes_cluster_role_binding.test
  to   = kubernetes_cluster_role_binding_v1.test
}

resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = %q
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

// testAccKubernetesClusterRoleBindingConfig_deprecatedTypeMultiSubject returns
// HCL for a binding with three subjects, using the deprecated type.
func testAccKubernetesClusterRoleBindingConfig_deprecatedTypeMultiSubject(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_cluster_role_binding" "test" {
  metadata {
    name = %q
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

  subject {
    kind      = "ServiceAccount"
    name      = "default"
    api_group = ""
    namespace = "kube-system"
  }

  subject {
    kind      = "Group"
    name      = "system:masters"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, name)
}

// testAccKubernetesClusterRoleBindingConfig_movedToV1MultiSubject is the
// moved-block equivalent of testAccKubernetesClusterRoleBindingConfig_deprecatedTypeMultiSubject.
func testAccKubernetesClusterRoleBindingConfig_movedToV1MultiSubject(name string) string {
	return fmt.Sprintf(`
moved {
  from = kubernetes_cluster_role_binding.test
  to   = kubernetes_cluster_role_binding_v1.test
}

resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = %q
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

  subject {
    kind      = "ServiceAccount"
    name      = "default"
    api_group = ""
    namespace = "kube-system"
  }

  subject {
    kind      = "Group"
    name      = "system:masters"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, name)
}

// testAccKubernetesClusterRoleBindingConfig_deprecatedTypeGenerateName returns
// HCL for a binding created with generate_name and three subjects, using the deprecated type.
func testAccKubernetesClusterRoleBindingConfig_deprecatedTypeGenerateName(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_cluster_role_binding" "test" {
  metadata {
    generate_name = %q
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

  subject {
    kind      = "ServiceAccount"
    name      = "default"
    api_group = ""
    namespace = "kube-system"
  }

  subject {
    kind      = "Group"
    name      = "system:masters"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, prefix)
}

// testAccKubernetesClusterRoleBindingConfig_movedToV1GenerateName is the
// moved-block equivalent of testAccKubernetesClusterRoleBindingConfig_deprecatedTypeGenerateName.
func testAccKubernetesClusterRoleBindingConfig_movedToV1GenerateName(prefix string) string {
	return fmt.Sprintf(`
moved {
  from = kubernetes_cluster_role_binding.test
  to   = kubernetes_cluster_role_binding_v1.test
}

resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    generate_name = %q
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

  subject {
    kind      = "ServiceAccount"
    name      = "default"
    api_group = ""
    namespace = "kube-system"
  }

  subject {
    kind      = "Group"
    name      = "system:masters"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, prefix)
}
