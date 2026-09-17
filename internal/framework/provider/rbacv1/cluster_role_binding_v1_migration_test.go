// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// This file contains the state-migration compatibility tests for the
// kubernetes_cluster_role_binding_v1 resource.
//
// Each test provisions a ClusterRoleBinding with the last released SDKv2-based
// provider (hashicorp/kubernetes) and then re-plans/applies the identical
// configuration with the locally built, muxed Plugin Framework provider. An
// empty plan on the second step proves that the SDKv2 -> Framework migration
// does not cause a breaking change: existing state and configuration are
// carried over in place, with no diff and no destroy/recreate.
//
// migratedProviderVersion is the last released version that still ships the
// SDKv2 implementation of this resource. It is used as the "from" provider.
const migratedProviderVersion = "3.2.0"

// releasedKubernetesProvider returns the external provider map pointing at the
// last SDKv2-based release, used for the first step of every migration test.
func releasedKubernetesProvider() map[string]resource.ExternalProvider {
	return map[string]resource.ExternalProvider{
		"kubernetes": {
			VersionConstraint: migratedProviderVersion,
			Source:            "hashicorp/kubernetes",
		},
	}
}

// migrationTestCase runs a standard two-step migration test:
//   - step 1: create with the released SDKv2 provider
//   - step 2: same config on the local Framework provider, expecting no diff
//
// extraChecks are asserted after the SDKv2 create step so we know the fixture
// was provisioned as expected before verifying the migration.
func migrationTestCase(t *testing.T, config string, extraChecks ...resource.TestCheckFunc) {
	t.Helper()

	resource.Test(t, resource.TestCase{
		CheckDestroy: testAccKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: releasedKubernetesProvider(),
				Config:            config,
				Check:             resource.ComposeAggregateTestCheckFunc(extraChecks...),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				// Re-apply on the Framework provider and confirm the plan is
				// still empty afterwards (idempotent, no drift post-migration).
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// TestAccMigrateClusterRoleBindingV1_basic verifies the simplest binding (a
// single User subject) migrates in place with no diff.
func TestAccMigrateClusterRoleBindingV1_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	migrationTestCase(t,
		testAccKubernetesClusterRoleBindingV1Config_basic(name),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "metadata.0.name", name),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.name", "cluster-admin"),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "1"),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.name", "notauser"),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.kind", "User"),
	)
}

// TestAccMigrateClusterRoleBindingV1_generatedName verifies that a binding
// created via metadata.generate_name (server-assigned name) migrates without a
// diff. This is the important computed-name case: the Framework version must
// reproduce the exact generated name that the SDKv2 version stored.
func TestAccMigrateClusterRoleBindingV1_generatedName(t *testing.T) {
	prefix := "tf-acc-test-gen:"

	migrationTestCase(t,
		testAccKubernetesClusterRoleBindingV1Config_generateName(prefix),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "metadata.0.generate_name", prefix),
		resource.TestCheckResourceAttrSet(clusterRoleBindingResourceName, "metadata.0.name"),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "1"),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.name", "notauser"),
	)
}

// TestAccMigrateClusterRoleBindingV1_multipleSubjects verifies that a binding
// with a mix of User, ServiceAccount (namespaced, empty api_group) and Group
// subjects migrates with no diff. This exercises the computed subject.api_group
// and subject.namespace defaulting logic across the migration boundary.
func TestAccMigrateClusterRoleBindingV1_multipleSubjects(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	migrationTestCase(t,
		testAccKubernetesClusterRoleBindingV1Config_modified(name),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "3"),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.kind", "User"),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.kind", "ServiceAccount"),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.namespace", "kube-system"),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.api_group", ""),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.2.kind", "Group"),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.2.name", "system:masters"),
	)
}

// TestAccMigrateClusterRoleBindingV1_serviceAccountSubject verifies migration of
// a binding whose only subject is a ServiceAccount that omits api_group and
// namespace, ensuring the computed/default handling matches SDKv2 state.
func TestAccMigrateClusterRoleBindingV1_serviceAccountSubject(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	migrationTestCase(t,
		testAccKubernetesClusterRoleBindingV1Config_serviceAccountSubject(name),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "1"),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.kind", "ServiceAccount"),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.name", "someservice"),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.api_group", ""),
	)
}

// TestAccMigrateClusterRoleBindingV1_groupSubject verifies migration of a
// binding with a single Group subject (non-empty api_group).
func TestAccMigrateClusterRoleBindingV1_groupSubject(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	migrationTestCase(t,
		testAccKubernetesClusterRoleBindingV1Config_groupSubject(name),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "1"),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.kind", "Group"),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.name", "somegroup"),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
	)
}

// TestAccMigrateClusterRoleBindingV1_thenUpdate verifies the full lifecycle
// across the migration boundary: create with the released SDKv2 provider,
// migrate to the Framework provider with no diff, then apply an actual
// configuration change (adding subjects) using the Framework provider to prove
// Update works correctly on state that originated from SDKv2.
func TestAccMigrateClusterRoleBindingV1_thenUpdate(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		CheckDestroy: testAccKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: releasedKubernetesProvider(),
				Config:            testAccKubernetesClusterRoleBindingV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "1"),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesClusterRoleBindingV1Config_basic(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesClusterRoleBindingV1Config_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "3"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.2.name", "system:masters"),
				),
			},
		},
	})
}

// TestAccMigrateClusterRoleBindingV1_thenForceReplace verifies that changing an
// immutable role_ref field after migration correctly forces replacement under
// the Framework provider, matching the SDKv2 ForceNew behaviour.
func TestAccMigrateClusterRoleBindingV1_thenForceReplace(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		CheckDestroy: testAccKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: releasedKubernetesProvider(),
				Config:            testAccKubernetesClusterRoleBindingV1Config_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.name", "cluster-admin"),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesClusterRoleBindingV1Config_modified(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesClusterRoleBindingV1Config_modifiedRoleRef(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(clusterRoleBindingResourceName, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.name", "admin"),
				),
			},
		},
	})
}

// TestAccMigrateClusterRoleBindingV1_externalMetadataSurvivesSubjectUpdate
// verifies that an annotation added to the Kubernetes object outside Terraform
// is not removed when the Framework resource patches only the subject list.
//
// The annotation key matches the provider-level ignore_annotations pattern so
// filterManagedMetadataKeys strips it from Terraform state on every Read.
// Because it never enters state it never appears in the patch diff and cannot
// be deleted by a subject-only update.
//
// Sequence:
//  1. Create with ignore_annotations = ["^external\\.io/"].
//  2. PreConfig of Step 2 injects "external.io/injected=true" via the
//     Kubernetes client before the plan runs. Step 2 is PlanOnly — confirms
//     no diff (the ignored key is filtered and does not create drift).
//  3. Step 3 updates subjects and verifies the annotation still exists on the
//     live Kubernetes object.
func TestAccMigrateClusterRoleBindingV1_externalMetadataSurvivesSubjectUpdate(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	const (
		// Matches the ignore pattern configured in the provider block below.
		externalKey   = "external.io/injected"
		externalValue = "true"
	)

	// injectAnnotation adds a key/value annotation directly to the live object.
	injectAnnotation := func(bindingName, key, value string) error {
		conn, err := sdkv2providerMeta()().(kubernetes.KubeClientsets).MainClientset()
		if err != nil {
			return err
		}
		ctx := context.TODO()
		obj, err := conn.RbacV1().ClusterRoleBindings().Get(ctx, bindingName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("get ClusterRoleBinding %q: %w", bindingName, err)
		}
		if obj.Annotations == nil {
			obj.Annotations = make(map[string]string)
		}
		obj.Annotations[key] = value
		_, err = conn.RbacV1().ClusterRoleBindings().Update(ctx, obj, metav1.UpdateOptions{})
		return err
	}

	// assertAnnotationOnObject reads the live object and verifies the annotation.
	assertAnnotationOnObject := func(bindingName, key, value string) resource.TestCheckFunc {
		return func(_ *terraform.State) error {
			conn, err := sdkv2providerMeta()().(kubernetes.KubeClientsets).MainClientset()
			if err != nil {
				return err
			}
			obj, err := conn.RbacV1().ClusterRoleBindings().Get(context.TODO(), bindingName, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("get ClusterRoleBinding %q: %w", bindingName, err)
			}
			got, ok := obj.Annotations[key]
			if !ok {
				return fmt.Errorf("annotation %q missing from object; annotations=%v", key, obj.Annotations)
			}
			if got != value {
				return fmt.Errorf("annotation %q = %q, want %q", key, got, value)
			}
			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		CheckDestroy: testAccKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				// Step 1 — create. Provider ignores external.io/* so the
				// injected annotation is never tracked in Terraform state.
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesClusterRoleBindingV1Config_ignoreExternalIO(name),
				Check: resource.TestCheckResourceAttr(
					clusterRoleBindingResourceName, "subject.#", "1",
				),
			},
			{
				// Step 2 — inject annotation in PreConfig (fires before the
				// plan), then plan-only: must be empty because the key is
				// filtered by ignore_annotations.
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesClusterRoleBindingV1Config_ignoreExternalIO(name),
				PreConfig: func() {
					if err := injectAnnotation(name, externalKey, externalValue); err != nil {
						t.Fatalf("inject annotation: %v", err)
					}
				},
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				// Step 3 — update subjects only; external annotation must survive.
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesClusterRoleBindingV1Config_ignoreExternalIOModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "3"),
					assertAnnotationOnObject(name, externalKey, externalValue),
				),
			},
		},
	})
}

func testAccKubernetesClusterRoleBindingV1Config_ignoreExternalIO(name string) string {
	return fmt.Sprintf(`
provider "kubernetes" {
  ignore_annotations = ["^external\\.io/"]
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

func testAccKubernetesClusterRoleBindingV1Config_ignoreExternalIOModified(name string) string {
	return fmt.Sprintf(`
provider "kubernetes" {
  ignore_annotations = ["^external\\.io/"]
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
