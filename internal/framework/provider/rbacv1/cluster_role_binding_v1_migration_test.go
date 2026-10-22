// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1_test

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
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
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

// clusterRoleBindingPreIdentityProviderVersion is the last release before
// resource identity was introduced.
const clusterRoleBindingPreIdentityProviderVersion = "2.37.1"

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
				ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
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
				ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
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

// TestAccMigrateClusterRoleBinding_basic verifies the simplest binding (a
// single User subject) migrates in place with no diff.
func TestAccMigrateClusterRoleBinding_basic(t *testing.T) {
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

// TestAccMigrateClusterRoleBinding_generatedName verifies that a binding
// created via metadata.generate_name (server-assigned name) migrates without a
// diff. This is the important computed-name case: the Framework version must
// reproduce the exact generated name that the SDKv2 version stored.
func TestAccMigrateClusterRoleBinding_generatedName(t *testing.T) {
	prefix := "tf-acc-test-gen:"

	migrationTestCase(t,
		testAccKubernetesClusterRoleBindingV1Config_generateName(prefix),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "metadata.0.generate_name", prefix),
		resource.TestCheckResourceAttrSet(clusterRoleBindingResourceName, "metadata.0.name"),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "1"),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.name", "notauser"),
	)
}

// TestAccMigrateClusterRoleBinding_preIdentityAndEmptyMetadata creates a
// binding with SDKv2 2.37.1 (before resource identity) and with the last SDKv2
// release, with and without explicitly empty labels and annotations, then
// switches to the Framework provider. Without metadata maps the plan must be
// empty; with empty maps SDKv2 stored null, so one in-place update is expected.
// The binding must keep its UID, and identity must be set.
func TestAccMigrateClusterRoleBinding_preIdentityAndEmptyMetadata(t *testing.T) {
	for _, version := range []string{clusterRoleBindingPreIdentityProviderVersion, migratedProviderVersion} {
		for _, empty := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/empty=%t", version, empty), func(t *testing.T) {
				name := "tf-acc-test:" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
				metadata := ""
				checkPlan := plancheck.ExpectEmptyPlan()
				var mapValue knownvalue.Check = knownvalue.Null()
				if empty {
					metadata = "labels = {}\nannotations = {}"
					checkPlan = plancheck.ExpectResourceAction(clusterRoleBindingResourceName, plancheck.ResourceActionUpdate)
					mapValue = knownvalue.MapExact(map[string]knownvalue.Check{})
				}
				config := testAccKubernetesClusterRoleBindingV1Config_metadata(name, metadata)
				checkUID := testAccClusterRoleBindingCheckSameUID()
				resource.Test(t, resource.TestCase{
					CheckDestroy: testAccKubernetesClusterRoleBindingV1Destroy,
					Steps: []resource.TestStep{
						{
							ExternalProviders: map[string]resource.ExternalProvider{
								"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: version},
							},
							Config: config,
							Check:  checkUID,
						},
						{
							ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
							Config:                   config,
							Check:                    checkUID,
							ConfigPlanChecks: resource.ConfigPlanChecks{
								PreApply:             []plancheck.PlanCheck{checkPlan},
								PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
							},
							ConfigStateChecks: []statecheck.StateCheck{
								statecheck.ExpectKnownValue(clusterRoleBindingResourceName,
									tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("labels"), mapValue),
								statecheck.ExpectKnownValue(clusterRoleBindingResourceName,
									tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("annotations"), mapValue),
								statecheck.ExpectIdentity(clusterRoleBindingResourceName, map[string]knownvalue.Check{
									"api_version": knownvalue.StringExact("rbac.authorization.k8s.io/v1"),
									"kind":        knownvalue.StringExact("ClusterRoleBinding"),
									"name":        knownvalue.StringExact(name),
								}),
							},
						},
					},
				})
			})
		}
	}
}

// TestAccMigrateClusterRoleBinding_multipleSubjects verifies that a binding
// with a mix of User, ServiceAccount (namespaced, empty api_group) and Group
// subjects migrates with no diff. This exercises the computed subject.api_group
// and subject.namespace defaulting logic across the migration boundary.
func TestAccMigrateClusterRoleBinding_multipleSubjects(t *testing.T) {
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

// TestAccMigrateClusterRoleBinding_serviceAccountSubject verifies migration of
// a binding whose only subject is a ServiceAccount that omits api_group and
// namespace, ensuring the computed/default handling matches SDKv2 state.
func TestAccMigrateClusterRoleBinding_serviceAccountSubject(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	migrationTestCase(t,
		testAccKubernetesClusterRoleBindingV1Config_serviceAccountSubject(name),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "1"),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.kind", "ServiceAccount"),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.name", "someservice"),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.api_group", ""),
	)
}

// TestAccMigrateClusterRoleBinding_groupSubject verifies migration of a
// binding with a single Group subject (non-empty api_group).
func TestAccMigrateClusterRoleBinding_groupSubject(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	migrationTestCase(t,
		testAccKubernetesClusterRoleBindingV1Config_groupSubject(name),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "1"),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.kind", "Group"),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.name", "somegroup"),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
	)
}

// TestAccMigrateClusterRoleBinding_emptySubjectNamespace verifies migration of
// a binding whose subject sets namespace = "" explicitly alongside one that
// relies on the "default" namespace, with no diff afterwards.
func TestAccMigrateClusterRoleBinding_emptySubjectNamespace(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	migrationTestCase(t,
		testAccKubernetesClusterRoleBindingV1Config_emptySubjectNamespace(name),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.namespace", ""),
		resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.namespace", "default"),
	)
}

// TestAccMigrateClusterRoleBinding_emptyUserAPIGroup covers a User subject
// with api_group = "", which the API server defaults to
// rbac.authorization.k8s.io. SDKv2 treats "" as unset for this Optional+Computed
// field and stores the API value, so it never drifts. The Framework must plan
// the configured "", so SDKv2 state gets one in-place update (never a
// replacement) and then stays converged.
func TestAccMigrateClusterRoleBinding_emptyUserAPIGroup(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesClusterRoleBindingV1Config_emptyUserAPIGroup(name)
	checkUID := testAccClusterRoleBindingCheckSameUID()

	resource.Test(t, resource.TestCase{
		CheckDestroy: testAccKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: releasedKubernetesProvider(),
				Config:            config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
					checkUID,
				),
			},
			{
				ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(clusterRoleBindingResourceName, plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.api_group", ""),
					checkUID,
				),
			},
			{
				ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

// TestAccMigrateClusterRoleBinding_noRefresh verifies that SDKv2-written state,
// which stores an unset generate_name as "", is not replaced when the Framework
// provider plans with -refresh=false, i.e. without Read normalising the value.
func TestAccMigrateClusterRoleBinding_noRefresh(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesClusterRoleBindingV1Config_basic(name)

	resource.Test(t, resource.TestCase{
		CheckDestroy: testAccKubernetesClusterRoleBindingV1Destroy,
		AdditionalCLIOptions: &resource.AdditionalCLIOptions{
			Plan: resource.PlanOptions{NoRefresh: true},
		},
		Steps: []resource.TestStep{
			{
				ExternalProviders: releasedKubernetesProvider(),
				Config:            config,
			},
			{
				ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{expectNotReplaced(clusterRoleBindingResourceName)},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "metadata.0.name", name),
					resource.TestCheckNoResourceAttr(clusterRoleBindingResourceName, "metadata.0.generate_name"),
				),
			},
		},
	})
}

// expectNotReplaced fails when the plan destroys and recreates the resource.
type expectNotReplacedCheck struct{ address string }

func expectNotReplaced(address string) plancheck.PlanCheck {
	return expectNotReplacedCheck{address: address}
}

func (e expectNotReplacedCheck) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	for _, rc := range req.Plan.ResourceChanges {
		if rc.Address != e.address {
			continue
		}
		if rc.Change.Actions.Replace() {
			resp.Error = fmt.Errorf("%s is planned for replacement: %v", e.address, rc.Change.Actions)
		}
		return
	}
	resp.Error = fmt.Errorf("%s not found in plan", e.address)
}

// TestAccMigrateClusterRoleBinding_thenUpdate verifies the full lifecycle
// across the migration boundary: create with the released SDKv2 provider,
// migrate to the Framework provider with no diff, then apply an actual
// configuration change (adding subjects) using the Framework provider to prove
// Update works correctly on state that originated from SDKv2.
func TestAccMigrateClusterRoleBinding_thenUpdate(t *testing.T) {
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
				ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
				Config:                   testAccKubernetesClusterRoleBindingV1Config_basic(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
				Config:                   testAccKubernetesClusterRoleBindingV1Config_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "3"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.2.name", "system:masters"),
				),
			},
		},
	})
}

// TestAccMigrateClusterRoleBinding_thenForceReplace verifies that changing an
// immutable role_ref field after migration correctly forces replacement under
// the Framework provider, matching the SDKv2 ForceNew behaviour.
func TestAccMigrateClusterRoleBinding_thenForceReplace(t *testing.T) {
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
				ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
				Config:                   testAccKubernetesClusterRoleBindingV1Config_modified(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
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

// TestAccMigrateClusterRoleBinding_externalMetadataSurvivesSubjectUpdate
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
func TestAccMigrateClusterRoleBinding_externalMetadataSurvivesSubjectUpdate(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	const (
		// Matches the ignore pattern configured in the provider block below.
		externalKey   = "external.io/injected"
		externalValue = "true"
	)

	// injectAnnotation adds a key/value annotation directly to the live object.
	injectAnnotation := func(bindingName, key, value string) error {
		meta, err := sdkv2providerMeta()
		if err != nil {
			return err
		}
		conn, err := meta().(kubernetes.KubeClientsets).MainClientset()
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
			meta, err := sdkv2providerMeta()
			if err != nil {
				return err
			}
			conn, err := meta().(kubernetes.KubeClientsets).MainClientset()
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
				ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
				Config:                   testAccKubernetesClusterRoleBindingV1Config_ignoreExternalIO(name),
				Check: resource.TestCheckResourceAttr(
					clusterRoleBindingResourceName, "subject.#", "1",
				),
			},
			{
				// Step 2 — inject annotation in PreConfig (fires before the
				// plan), then plan-only: must be empty because the key is
				// filtered by ignore_annotations.
				ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
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
				ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
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
