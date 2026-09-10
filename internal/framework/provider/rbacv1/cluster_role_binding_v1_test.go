// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

const clusterRoleBindingResourceName = "kubernetes_cluster_role_binding_v1.test"

func TestAccFrameworkClusterRoleBindingV1_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(clusterRoleBindingResourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(clusterRoleBindingResourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(clusterRoleBindingResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.kind", "ClusterRole"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.name", "cluster-admin"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.name", "notauser"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.kind", "User"),
				),
			},
			{
				Config: testAccKubernetesClusterRoleBindingV1Config_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.name", "cluster-admin"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "3"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.name", "notauser"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.kind", "User"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.namespace", "kube-system"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.name", "default"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.kind", "ServiceAccount"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.api_group", ""),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.2.name", "system:masters"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.2.kind", "Group"),
				),
			},
			{
				// role_ref change forces replacement (ForceNew).
				Config: testAccKubernetesClusterRoleBindingV1Config_modifiedRoleRef(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.name", "admin"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "3"),
				),
			},
			{
				ResourceName:            clusterRoleBindingResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
		},
	})
}

func TestAccFrameworkClusterRoleBindingV1_generatedName(t *testing.T) {
	prefix := "tf-acc-test-gen:"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingV1Config_generateName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(clusterRoleBindingResourceName, "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "metadata.0.generate_name", prefix),
					resource.TestCheckResourceAttrSet(clusterRoleBindingResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.name", "notauser"),
				),
			},
		},
	})
}

func TestAccFrameworkClusterRoleBindingV1_serviceAccountSubject(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingV1Config_serviceAccountSubject(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.api_group", ""),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.name", "someservice"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.kind", "ServiceAccount"),
				),
			},
		},
	})
}

func TestAccFrameworkClusterRoleBindingV1_groupSubject(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingV1Config_groupSubject(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.name", "somegroup"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.kind", "Group"),
				),
			},
			{
				ResourceName:            clusterRoleBindingResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
		},
	})
}

func TestAccFrameworkClusterRoleBindingV1_identity(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccKubernetesClusterRoleBindingV1Destroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingV1Config_basic(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(
						clusterRoleBindingResourceName, map[string]knownvalue.Check{
							"name":        knownvalue.StringExact(name),
							"api_version": knownvalue.StringExact("rbac.authorization.k8s.io/v1"),
							"kind":        knownvalue.StringExact("ClusterRoleBinding"),
						},
					),
				},
			},
			{
				ResourceName:    clusterRoleBindingResourceName,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func testAccKubernetesClusterRoleBindingV1Config_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = "%s"
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

func testAccKubernetesClusterRoleBindingV1Config_generateName(namePrefix string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    generate_name = "%s"
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
`, namePrefix)
}

func testAccKubernetesClusterRoleBindingV1Config_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = "%s"
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

func testAccKubernetesClusterRoleBindingV1Config_modifiedRoleRef(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = "%s"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
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

func testAccKubernetesClusterRoleBindingV1Config_serviceAccountSubject(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = "%s"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = "cluster-admin"
  }

  subject {
    kind = "ServiceAccount"
    name = "someservice"
  }
}
`, name)
}

func testAccKubernetesClusterRoleBindingV1Config_groupSubject(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = "%s"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = "cluster-admin"
  }

  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Group"
    name      = "somegroup"
  }
}
`, name)
}

// TestAccFrameworkClusterRoleBindingV1_ignoreAnnotations verifies that:
//   - provider-level ignore_annotations prevents drift from untracked
//     server-side annotations matching the pattern
//   - annotations the user has explicitly configured are retained even
//     when their key matches an ignore pattern
func TestAccFrameworkClusterRoleBindingV1_ignoreAnnotations(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				// Step 1 — create a binding with an annotation that is
				// tracked in config. The provider is configured to ignore
				// the same pattern so that any server-added keys with that
				// prefix are transparently dropped.
				Config: testAccKubernetesClusterRoleBindingV1Config_ignoreAnnotations(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "metadata.0.name", name),
					// The user-configured annotation must appear in state.
					resource.TestCheckResourceAttr(
						clusterRoleBindingResourceName,
						"metadata.0.annotations.acme.io/managed",
						"true",
					),
				),
			},
			{
				// Step 2 — re-plan; must be empty (no phantom diff from
				// internal Kubernetes annotations or the ignored pattern).
				Config:             testAccKubernetesClusterRoleBindingV1Config_ignoreAnnotations(name),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func testAccKubernetesClusterRoleBindingV1Config_ignoreAnnotations(name string) string {
	return fmt.Sprintf(`
provider "kubernetes" {
  ignore_annotations = ["^acme\\.io/"]
}

resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = %q
    annotations = {
      "acme.io/managed" = "true"
    }
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

// TestAccFrameworkClusterRoleBindingV1_subjectOrderWithRemovals is the
// Framework port of the SDKv2
// TestAccKubernetesClusterRoleBindingV1_UpdatePatchOperationsOrderWithRemovals
// test. It verifies that removing a subject, then re-adding subjects in a
// different order, produces the exact expected list with no stale entries.
// This validates the single replace /subjects operation instead of the
// error-prone per-index patch approach.
func TestAccFrameworkClusterRoleBindingV1_subjectOrderWithRemovals(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				// Step 1 — three subjects.
				Config: testAccKubernetesClusterRoleBindingV1Config_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "3"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.kind", "User"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.kind", "ServiceAccount"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.2.kind", "Group"),
				),
			},
			{
				// Step 2 — remove the ServiceAccount; two subjects remain.
				Config: testAccKubernetesClusterRoleBindingV1Config_subjectRemoval(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "2"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.kind", "User"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.kind", "Group"),
				),
			},
			{
				// Step 3 — add subjects back in a different order.
				Config: testAccKubernetesClusterRoleBindingV1Config_subjectReorder(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "3"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.kind", "Group"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.kind", "ServiceAccount"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.2.kind", "User"),
				),
			},
		},
	})
}

func testAccKubernetesClusterRoleBindingV1Config_subjectRemoval(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = "%s"
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
    kind      = "Group"
    name      = "system:masters"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, name)
}

func testAccKubernetesClusterRoleBindingV1Config_subjectReorder(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = "%s"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = "cluster-admin"
  }

  subject {
    kind      = "Group"
    name      = "system:masters"
    api_group = "rbac.authorization.k8s.io"
  }

  subject {
    kind      = "ServiceAccount"
    name      = "default"
    api_group = ""
    namespace = "kube-system"
  }

  subject {
    kind      = "User"
    name      = "notauser"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, name)
}

// TestAccFrameworkClusterRoleBindingV1_invalidMetadataName verifies that a
// metadata.name containing a path separator is rejected during Terraform
// validation, matching the SDKv2 validateRBACNameFunc behavior.
func TestAccFrameworkClusterRoleBindingV1_invalidMetadataName(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccKubernetesClusterRoleBindingV1Config_invalidName(),
				ExpectError: regexp.MustCompile(`may not contain`),
			},
		},
	})
}

// TestAccFrameworkClusterRoleBindingV1_invalidMetadataGenerateName verifies
// that an invalid generate_name prefix is also rejected during validation.
func TestAccFrameworkClusterRoleBindingV1_invalidMetadataGenerateName(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccKubernetesClusterRoleBindingV1Config_invalidGenerateName(),
				ExpectError: regexp.MustCompile(`may not contain`),
			},
		},
	})
}

func testAccKubernetesClusterRoleBindingV1Config_invalidName() string {
	return `
resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = "invalid/name"
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
`
}

func testAccKubernetesClusterRoleBindingV1Config_invalidGenerateName() string {
	return `
resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    generate_name = "invalid/prefix"
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
`
}
