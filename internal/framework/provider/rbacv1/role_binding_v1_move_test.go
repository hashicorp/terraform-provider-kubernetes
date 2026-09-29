// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1_test

// ---------------------------------------------------------------------------
// MoveState tests for kubernetes_role_binding_v1.
//
// This file is the single home for all MoveState coverage — the mechanism that
// lets practitioners migrate from the deprecated kubernetes_role_binding
// (SDKv2) resource to kubernetes_role_binding_v1 (Framework) using a
// moved { } block:
//
//	moved {
//	  from = kubernetes_role_binding.example
//	  to   = kubernetes_role_binding_v1.example
//	}
//
// It contains two families of tests that describe the same state shapes and so
// cannot drift apart:
//
//   - Acceptance tests (TestAccMoveRoleBinding_*) drive real Terraform: they
//     create state with the released SDKv2 provider, add a moved block, switch
//     to the local Framework provider, and assert an empty post-move plan.
//   - Unit tests (TestMigration_MoveState_*) call the MoveState handler
//     directly with raw SDKv2 JSON and assert the translated model field by
//     field. These were relocated here from role_binding_v1_migration_test.go
//     so that every MoveState test lives together.
//
// Correspondence between the two families (see requirement: every migration
// scenario has a matching MoveState test):
//
//	Acceptance test                        Unit test
//	──────────────────────────────────────────────────────────────────────────
//	TestAccMoveRoleBinding_basic           TestMigration_MoveState_basic
//	                                        TestMigration_MoveState_emptyGenerateName
//	                                        TestMigration_MoveState_emptyMapsAreNil
//	TestAccMoveRoleBinding_generateName    TestMigration_MoveState_nonEmptyGenerateName
//	TestAccMoveRoleBinding_multipleSubjects TestMigration_MoveState_multipleSubjects
//	TestAccMoveRoleBinding_clusterRoleRef  TestMigration_MoveState_clusterRoleRef
//
// The remaining unit tests are guard cases (handler wiring and early-return
// safety) with no state shape to reproduce at the acceptance level:
// TestMigration_MoveState_handlersRegistered,
// TestMigration_MoveState_wrongSourceTypeIsIgnored,
// TestMigration_MoveState_wrongProviderAddressIsIgnored,
// TestMigration_MoveState_wrongSchemaVersionIsIgnored,
// TestMigration_MoveState_nilRawStateIsIgnored.
// ---------------------------------------------------------------------------

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	testacctest "github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	rbacv1 "github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/rbacv1"
)

// TestAccMoveRoleBinding_basic verifies that a binding created as
// kubernetes_role_binding can be moved to kubernetes_role_binding_v1 via a
// moved block with no diff. Corresponds to TestMigration_MoveState_basic.
func TestAccMoveRoleBinding_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	tfresource.Test(t, tfresource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		Steps: []tfresource.TestStep{
			{
				// Step 1 — create with the released SDKv2 provider using the
				// deprecated kubernetes_role_binding resource type.
				ExternalProviders: map[string]tfresource.ExternalProvider{
					"kubernetes": {
						Source:            "hashicorp/kubernetes",
						VersionConstraint: "3.2.1",
					},
				},
				Config: testAccKubernetesRoleBindingConfig_deprecatedType(name),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr(
						"kubernetes_role_binding.test",
						"metadata.0.name", name,
					),
					tfresource.TestCheckResourceAttr(
						"kubernetes_role_binding.test",
						"subject.#", "1",
					),
				),
			},
			{
				// Step 2 — switch to the local Framework provider with a
				// moved block renaming the resource type. The plan must be
				// empty: MoveState copies state in place with no changes.
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesRoleBindingConfig_movedToV1(name),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr(
						"kubernetes_role_binding_v1.test",
						"metadata.0.name", name,
					),
					tfresource.TestCheckResourceAttr(
						"kubernetes_role_binding_v1.test",
						"subject.#", "1",
					),
					tfresource.TestCheckResourceAttr(
						"kubernetes_role_binding_v1.test",
						"role_ref.0.name", "admin",
					),
				),
			},
			{
				// Step 3 — idempotency after the move: plan must still be
				// empty on the Framework provider.
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesRoleBindingConfig_movedToV1(name),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// TestAccMoveRoleBinding_generateName verifies that a binding created with
// metadata.generate_name can be moved to kubernetes_role_binding_v1 via a
// moved block with no diff. Corresponds to TestMigration_MoveState_nonEmptyGenerateName.
func TestAccMoveRoleBinding_generateName(t *testing.T) {
	prefix := "tf-acc-rb-"

	tfresource.Test(t, tfresource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		Steps: []tfresource.TestStep{
			{
				ExternalProviders: map[string]tfresource.ExternalProvider{
					"kubernetes": {
						Source:            "hashicorp/kubernetes",
						VersionConstraint: "3.2.1",
					},
				},
				Config: testAccKubernetesRoleBindingConfig_deprecatedTypeGenerateName(prefix),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr(
						"kubernetes_role_binding.test",
						"metadata.0.generate_name", prefix,
					),
					tfresource.TestCheckResourceAttrSet(
						"kubernetes_role_binding.test",
						"metadata.0.name",
					),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesRoleBindingConfig_movedToV1GenerateName(prefix),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr(
						"kubernetes_role_binding_v1.test",
						"metadata.0.generate_name", prefix,
					),
					tfresource.TestCheckResourceAttrSet(
						"kubernetes_role_binding_v1.test",
						"metadata.0.name",
					),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesRoleBindingConfig_movedToV1GenerateName(prefix),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// TestAccMoveRoleBinding_multipleSubjects verifies that a binding with all
// three subject types (User, ServiceAccount, Group) can be moved via a moved
// block with no diff. Corresponds to TestMigration_MoveState_multipleSubjects.
func TestAccMoveRoleBinding_multipleSubjects(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	tfresource.Test(t, tfresource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		Steps: []tfresource.TestStep{
			{
				ExternalProviders: map[string]tfresource.ExternalProvider{
					"kubernetes": {
						Source:            "hashicorp/kubernetes",
						VersionConstraint: "3.2.1",
					},
				},
				Config: testAccKubernetesRoleBindingConfig_deprecatedTypeMultiSubject(name),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr(
						"kubernetes_role_binding.test",
						"subject.#", "3",
					),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesRoleBindingConfig_movedToV1MultiSubject(name),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr(
						"kubernetes_role_binding_v1.test",
						"subject.#", "3",
					),
					tfresource.TestCheckResourceAttr(
						"kubernetes_role_binding_v1.test",
						"subject.0.kind", "User",
					),
					tfresource.TestCheckResourceAttr(
						"kubernetes_role_binding_v1.test",
						"subject.1.kind", "ServiceAccount",
					),
					tfresource.TestCheckResourceAttr(
						"kubernetes_role_binding_v1.test",
						"subject.2.kind", "Group",
					),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesRoleBindingConfig_movedToV1MultiSubject(name),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// TestAccMoveRoleBinding_clusterRoleRef verifies that a RoleBinding referencing
// a ClusterRole can be moved via a moved block with no diff.
// Corresponds to TestMigration_MoveState_clusterRoleRef.
func TestAccMoveRoleBinding_clusterRoleRef(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	tfresource.Test(t, tfresource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		Steps: []tfresource.TestStep{
			{
				ExternalProviders: map[string]tfresource.ExternalProvider{
					"kubernetes": {
						Source:            "hashicorp/kubernetes",
						VersionConstraint: "3.2.1",
					},
				},
				Config: testAccKubernetesRoleBindingConfig_deprecatedTypeClusterRoleRef(name),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr(
						"kubernetes_role_binding.test",
						"role_ref.0.kind", "ClusterRole",
					),
					tfresource.TestCheckResourceAttr(
						"kubernetes_role_binding.test",
						"role_ref.0.name", "cluster-admin",
					),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesRoleBindingConfig_movedToV1ClusterRoleRef(name),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr(
						"kubernetes_role_binding_v1.test",
						"role_ref.0.kind", "ClusterRole",
					),
					tfresource.TestCheckResourceAttr(
						"kubernetes_role_binding_v1.test",
						"role_ref.0.name", "cluster-admin",
					),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesRoleBindingConfig_movedToV1ClusterRoleRef(name),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// ── Config helpers ────────────────────────────────────────────────────────────

// testAccKubernetesRoleBindingConfig_deprecatedType returns HCL that creates a
// RoleBinding using the deprecated kubernetes_role_binding resource type.
func testAccKubernetesRoleBindingConfig_deprecatedType(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding" "test" {
  metadata {
    name      = %q
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

// testAccKubernetesRoleBindingConfig_movedToV1 returns HCL that moves the
// kubernetes_role_binding.test resource to kubernetes_role_binding_v1.test
// using a moved { } block.
func testAccKubernetesRoleBindingConfig_movedToV1(name string) string {
	return fmt.Sprintf(`
moved {
  from = kubernetes_role_binding.test
  to   = kubernetes_role_binding_v1.test
}

resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name      = %q
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

// testAccKubernetesRoleBindingConfig_deprecatedTypeGenerateName returns HCL for a
// binding with generate_name, using the deprecated type.
func testAccKubernetesRoleBindingConfig_deprecatedTypeGenerateName(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding" "test" {
  metadata {
    generate_name = %q
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

// testAccKubernetesRoleBindingConfig_movedToV1GenerateName is the moved-block
// equivalent of testAccKubernetesRoleBindingConfig_deprecatedTypeGenerateName.
func testAccKubernetesRoleBindingConfig_movedToV1GenerateName(prefix string) string {
	return fmt.Sprintf(`
moved {
  from = kubernetes_role_binding.test
  to   = kubernetes_role_binding_v1.test
}

resource "kubernetes_role_binding_v1" "test" {
  metadata {
    generate_name = %q
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

// testAccKubernetesRoleBindingConfig_deprecatedTypeMultiSubject returns HCL for
// a binding with three subjects (User, ServiceAccount, Group), using the deprecated type.
func testAccKubernetesRoleBindingConfig_deprecatedTypeMultiSubject(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding" "test" {
  metadata {
    name      = %q
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

// testAccKubernetesRoleBindingConfig_movedToV1MultiSubject is the moved-block
// equivalent of testAccKubernetesRoleBindingConfig_deprecatedTypeMultiSubject.
func testAccKubernetesRoleBindingConfig_movedToV1MultiSubject(name string) string {
	return fmt.Sprintf(`
moved {
  from = kubernetes_role_binding.test
  to   = kubernetes_role_binding_v1.test
}

resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name      = %q
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

// testAccKubernetesRoleBindingConfig_deprecatedTypeClusterRoleRef returns HCL
// for a binding that references a ClusterRole, using the deprecated type.
func testAccKubernetesRoleBindingConfig_deprecatedTypeClusterRoleRef(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding" "test" {
  metadata {
    name      = %q
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

// testAccKubernetesRoleBindingConfig_movedToV1ClusterRoleRef is the moved-block
// equivalent of testAccKubernetesRoleBindingConfig_deprecatedTypeClusterRoleRef.
func testAccKubernetesRoleBindingConfig_movedToV1ClusterRoleRef(name string) string {
	return fmt.Sprintf(`
moved {
  from = kubernetes_role_binding.test
  to   = kubernetes_role_binding_v1.test
}

resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name      = %q
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

// ── helpers ───────────────────────────────────────────────────────────────────

// sdkv2RawJSON produces raw JSON bytes that mirror what the SDKv2 provider
// writes into terraform.tfstate for kubernetes_role_binding.
func sdkv2RawJSON(
	id, name, generateName, namespace string,
	annotations, labels map[string]string,
	resourceVersion, uid string,
	generation int,
	roleRefAPIGroup, roleRefKind, roleRefName string,
	subjects []map[string]string,
) []byte {
	meta := map[string]interface{}{
		"name":             name,
		"generate_name":    generateName,
		"namespace":        namespace,
		"resource_version": resourceVersion,
		"uid":              uid,
		"generation":       generation,
		"annotations":      annotations,
		"labels":           labels,
	}

	roleRef := []interface{}{
		map[string]interface{}{
			"api_group": roleRefAPIGroup,
			"kind":      roleRefKind,
			"name":      roleRefName,
		},
	}

	subjectList := make([]interface{}, 0, len(subjects))
	for _, s := range subjects {
		subjectList = append(subjectList, map[string]interface{}{
			"api_group": s["api_group"],
			"kind":      s["kind"],
			"name":      s["name"],
			"namespace": s["namespace"],
		})
	}

	state := map[string]interface{}{
		"id":       id,
		"metadata": []interface{}{meta},
		"role_ref": roleRef,
		"subject":  subjectList,
	}
	raw, _ := json.Marshal(state)
	return raw
}

// runMoveStateWithReq calls the MoveState handler with a full MoveStateRequest.
func runMoveStateWithReq(t *testing.T, req resource.MoveStateRequest) *resource.MoveStateResponse {
	t.Helper()
	r := rbacv1.NewRoleBindingV1()
	movers := r.(interface {
		MoveState(context.Context) []resource.StateMover
	}).MoveState(context.Background())

	if len(movers) == 0 {
		t.Fatal("expected at least 1 StateMover")
	}

	resp := &resource.MoveStateResponse{
		TargetState: tfsdk.State{Schema: rbacv1.RoleBindingV1Schema()},
	}
	movers[0].StateMover(context.Background(), req, resp)
	return resp
}

// runMoveState calls the MoveState handler with the given source type and raw JSON,
// using default provider address ("hashicorp/kubernetes") and schema version (0).
func runMoveState(t *testing.T, sourceTypeName string, rawJSON []byte) *resource.MoveStateResponse {
	t.Helper()
	var rawState *tfprotov6.RawState
	if rawJSON != nil {
		rawState = &tfprotov6.RawState{JSON: rawJSON}
	}
	req := resource.MoveStateRequest{
		SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes",
		SourceTypeName:        sourceTypeName,
		SourceSchemaVersion:   0,
		SourceRawState:        rawState,
	}
	return runMoveStateWithReq(t, req)
}

// readMovedModel reads the moved RoleBindingModel out of resp.TargetState.
func readMovedModel(t *testing.T, resp *resource.MoveStateResponse) rbacv1.RoleBindingModel {
	t.Helper()
	if resp.Diagnostics.HasError() {
		t.Fatalf("move state produced errors: %s", resp.Diagnostics)
	}
	var m rbacv1.RoleBindingModel
	resp.Diagnostics.Append(resp.TargetState.Get(context.Background(), &m)...)
	if resp.Diagnostics.HasError() {
		t.Fatalf("reading moved state: %s", resp.Diagnostics)
	}
	return m
}

// ── MoveState tests ───────────────────────────────────────────────────────────

// TestMigration_MoveState_handlersRegistered verifies the StateMover is wired
// up — a compile-time regression guard.
func TestMigration_MoveState_handlersRegistered(t *testing.T) {
	t.Parallel()
	r := rbacv1.NewRoleBindingV1()
	movers := r.(interface {
		MoveState(context.Context) []resource.StateMover
	}).MoveState(context.Background())

	if len(movers) == 0 {
		t.Error("expected at least 1 StateMover registered")
	}
}

// TestMigration_MoveState_basic verifies the core translation from
// kubernetes_role_binding raw JSON state to RoleBindingModel.
func TestMigration_MoveState_basic(t *testing.T) {
	t.Parallel()

	raw := sdkv2RawJSON(
		"default/my-binding",
		"my-binding", "", "default",
		map[string]string{"example.com/note": "test"},
		map[string]string{"managed-by": "terraform"},
		"567273", "a6da86ec-b80d-44c0-9007-aafa4d982d4a", 1,
		"rbac.authorization.k8s.io", "Role", "pod-reader",
		[]map[string]string{
			{"api_group": "rbac.authorization.k8s.io", "kind": "User", "name": "alice", "namespace": "default"},
		},
	)

	resp := runMoveState(t, "kubernetes_role_binding", raw)
	got := readMovedModel(t, resp)

	if got.ID.ValueString() != "default/my-binding" {
		t.Errorf("id: got %q, want default/my-binding", got.ID.ValueString())
	}
	if len(got.Metadata) != 1 {
		t.Fatalf("metadata: got %d elements, want 1", len(got.Metadata))
	}
	if got.Metadata[0].Name.ValueString() != "my-binding" {
		t.Errorf("name: got %q, want my-binding", got.Metadata[0].Name.ValueString())
	}
	if got.Metadata[0].Namespace.ValueString() != "default" {
		t.Errorf("namespace: got %q, want default", got.Metadata[0].Namespace.ValueString())
	}
	if len(got.RoleRef) != 1 {
		t.Fatalf("role_ref: got %d elements, want 1", len(got.RoleRef))
	}
	if got.RoleRef[0].Kind.ValueString() != "Role" {
		t.Errorf("role_ref.kind: got %q, want Role", got.RoleRef[0].Kind.ValueString())
	}
	if got.RoleRef[0].Name.ValueString() != "pod-reader" {
		t.Errorf("role_ref.name: got %q, want pod-reader", got.RoleRef[0].Name.ValueString())
	}
	if len(got.Subject) != 1 {
		t.Fatalf("subject: got %d elements, want 1", len(got.Subject))
	}
	if got.Subject[0].Name.ValueString() != "alice" {
		t.Errorf("subject[0].name: got %q, want alice", got.Subject[0].Name.ValueString())
	}
	// Annotations and Labels are now types.Map; extract elements to check values.
	annotationElems := got.Metadata[0].Annotations.Elements()
	if v, ok := annotationElems["example.com/note"]; !ok {
		t.Error("annotation 'example.com/note' missing")
	} else if s, ok := v.(interface{ ValueString() string }); !ok || s.ValueString() != "test" {
		t.Errorf("annotation: got %v, want test", v)
	}
	labelElems := got.Metadata[0].Labels.Elements()
	if v, ok := labelElems["managed-by"]; !ok {
		t.Error("label 'managed-by' missing")
	} else if s, ok := v.(interface{ ValueString() string }); !ok || s.ValueString() != "terraform" {
		t.Errorf("label: got %v, want terraform", v)
	}
}

// TestMigration_MoveState_emptyGenerateName verifies that an empty string
// generate_name from SDKv2 is normalised to null to prevent plan drift.
func TestMigration_MoveState_emptyGenerateName(t *testing.T) {
	t.Parallel()

	raw := sdkv2RawJSON(
		"default/my-binding", "my-binding", "" /* empty generate_name */, "default",
		nil, nil, "1", "uid-1", 0,
		"rbac.authorization.k8s.io", "Role", "admin",
		[]map[string]string{
			{"api_group": "rbac.authorization.k8s.io", "kind": "User", "name": "bob", "namespace": "default"},
		},
	)

	resp := runMoveState(t, "kubernetes_role_binding", raw)
	got := readMovedModel(t, resp)

	if !got.Metadata[0].GenerateName.IsNull() {
		t.Errorf("generate_name: expected null for empty string, got %q",
			got.Metadata[0].GenerateName.ValueString())
	}
}

// TestMigration_MoveState_nonEmptyGenerateName verifies that a set
// generate_name is preserved after the move.
func TestMigration_MoveState_nonEmptyGenerateName(t *testing.T) {
	t.Parallel()

	raw := sdkv2RawJSON(
		"default/rb-gen-xk9p2", "rb-gen-xk9p2", "rb-gen-", "default",
		nil, nil, "2", "uid-2", 0,
		"rbac.authorization.k8s.io", "Role", "admin",
		[]map[string]string{
			{"api_group": "rbac.authorization.k8s.io", "kind": "User", "name": "carol", "namespace": "default"},
		},
	)

	resp := runMoveState(t, "kubernetes_role_binding", raw)
	got := readMovedModel(t, resp)

	if got.Metadata[0].GenerateName.IsNull() {
		t.Error("generate_name: expected non-null for 'rb-gen-', got null")
	}
	if got.Metadata[0].GenerateName.ValueString() != "rb-gen-" {
		t.Errorf("generate_name: got %q, want rb-gen-",
			got.Metadata[0].GenerateName.ValueString())
	}
}

// TestMigration_MoveState_emptyMapsAreEmpty verifies that SDKv2 JSON `{}` maps for
// annotations and labels are preserved as empty (zero-element, known) types.Map values,
// not as null. SDKv2 stores `{}` when the user explicitly set annotations = {} or
// labels = {}; the null/empty distinction must be preserved across the state move
// so the Framework schema can produce an in-place update (not a perpetual diff).
func TestMigration_MoveState_emptyMapsAreEmpty(t *testing.T) {
	t.Parallel()

	raw := sdkv2RawJSON(
		"default/my-binding", "my-binding", "", "default",
		map[string]string{}, map[string]string{}, // SDKv2 stored {} (not null)
		"1", "uid-3", 0,
		"rbac.authorization.k8s.io", "Role", "admin",
		[]map[string]string{
			{"api_group": "rbac.authorization.k8s.io", "kind": "User", "name": "dave", "namespace": "default"},
		},
	)

	resp := runMoveState(t, "kubernetes_role_binding", raw)
	got := readMovedModel(t, resp)

	// {} from SDKv2 → empty known types.Map (not null): the plan for a config that
	// omits annotations/labels will show an in-place update, not a replacement.
	if got.Metadata[0].Annotations.IsNull() {
		t.Error("annotations: expected empty known map for SDKv2 {}, got null")
	}
	if got.Metadata[0].Annotations.IsUnknown() {
		t.Error("annotations: expected empty known map for SDKv2 {}, got unknown")
	}
	if len(got.Metadata[0].Annotations.Elements()) != 0 {
		t.Errorf("annotations: expected 0 elements for SDKv2 {}, got %d", len(got.Metadata[0].Annotations.Elements()))
	}
	if got.Metadata[0].Labels.IsNull() {
		t.Error("labels: expected empty known map for SDKv2 {}, got null")
	}
	if got.Metadata[0].Labels.IsUnknown() {
		t.Error("labels: expected empty known map for SDKv2 {}, got unknown")
	}
	if len(got.Metadata[0].Labels.Elements()) != 0 {
		t.Errorf("labels: expected 0 elements for SDKv2 {}, got %d", len(got.Metadata[0].Labels.Elements()))
	}
}

// TestMigration_MoveState_multipleSubjects verifies that all three subject
// types (User, ServiceAccount, Group) are preserved correctly after the move.
func TestMigration_MoveState_multipleSubjects(t *testing.T) {
	t.Parallel()

	raw := sdkv2RawJSON(
		"default/my-binding", "my-binding", "", "default",
		nil, nil, "1", "uid-4", 0,
		"rbac.authorization.k8s.io", "Role", "pod-reader",
		[]map[string]string{
			{"api_group": "rbac.authorization.k8s.io", "kind": "User", "name": "alice", "namespace": "default"},
			{"api_group": "", "kind": "ServiceAccount", "name": "default", "namespace": "kube-system"},
			{"api_group": "rbac.authorization.k8s.io", "kind": "Group", "name": "dev-team", "namespace": "default"},
		},
	)

	resp := runMoveState(t, "kubernetes_role_binding", raw)
	got := readMovedModel(t, resp)

	if len(got.Subject) != 3 {
		t.Fatalf("subject: got %d elements, want 3", len(got.Subject))
	}
	if got.Subject[0].Kind.ValueString() != "User" {
		t.Errorf("subject[0].kind: got %q, want User", got.Subject[0].Kind.ValueString())
	}
	if got.Subject[1].Kind.ValueString() != "ServiceAccount" {
		t.Errorf("subject[1].kind: got %q, want ServiceAccount", got.Subject[1].Kind.ValueString())
	}
	if got.Subject[1].Namespace.ValueString() != "kube-system" {
		t.Errorf("subject[1].namespace: got %q, want kube-system", got.Subject[1].Namespace.ValueString())
	}
	if got.Subject[2].Kind.ValueString() != "Group" {
		t.Errorf("subject[2].kind: got %q, want Group", got.Subject[2].Kind.ValueString())
	}
}

// TestMigration_MoveState_clusterRoleRef verifies that a ClusterRole reference
// in role_ref is preserved correctly.
func TestMigration_MoveState_clusterRoleRef(t *testing.T) {
	t.Parallel()

	raw := sdkv2RawJSON(
		"default/my-binding", "my-binding", "", "default",
		nil, nil, "1", "uid-5", 0,
		"rbac.authorization.k8s.io", "ClusterRole", "cluster-admin",
		[]map[string]string{
			{"api_group": "rbac.authorization.k8s.io", "kind": "User", "name": "alice", "namespace": "default"},
		},
	)

	resp := runMoveState(t, "kubernetes_role_binding", raw)
	got := readMovedModel(t, resp)

	if got.RoleRef[0].Kind.ValueString() != "ClusterRole" {
		t.Errorf("role_ref.kind: got %q, want ClusterRole", got.RoleRef[0].Kind.ValueString())
	}
	if got.RoleRef[0].Name.ValueString() != "cluster-admin" {
		t.Errorf("role_ref.name: got %q, want cluster-admin", got.RoleRef[0].Name.ValueString())
	}
}

// TestMigration_MoveState_wrongSourceTypeIsIgnored verifies that the handler
// returns early without error or writing any state when SourceTypeName does
// not match "kubernetes_role_binding". This ensures the handler is safe to
// call for any moved block in the configuration.
func TestMigration_MoveState_wrongSourceTypeIsIgnored(t *testing.T) {
	t.Parallel()

	raw := sdkv2RawJSON(
		"default/some-other", "some-other", "", "default",
		nil, nil, "1", "uid-6", 0,
		"rbac.authorization.k8s.io", "Role", "admin",
		[]map[string]string{
			{"api_group": "rbac.authorization.k8s.io", "kind": "User", "name": "alice", "namespace": "default"},
		},
	)

	resp := runMoveState(t, "kubernetes_some_other_resource", raw)

	// No errors expected — handler must silently return
	if resp.Diagnostics.HasError() {
		t.Errorf("expected no errors for unrecognised source type, got: %s",
			resp.Diagnostics)
	}

	// TargetState must be empty — handler must not have written anything
	var m rbacv1.RoleBindingModel
	diags := resp.TargetState.Get(context.Background(), &m)
	if !diags.HasError() && m.ID.ValueString() != "" {
		t.Errorf("expected empty target state for unrecognised source type, got id=%q",
			m.ID.ValueString())
	}
}

// TestMigration_MoveState_wrongProviderAddressIsIgnored verifies that the handler
// returns early without error or writing state when SourceProviderAddress does
// not end with "hashicorp/kubernetes".
func TestMigration_MoveState_wrongProviderAddressIsIgnored(t *testing.T) {
	t.Parallel()

	raw := sdkv2RawJSON(
		"default/my-binding", "my-binding", "", "default",
		nil, nil, "1", "uid-7", 0,
		"rbac.authorization.k8s.io", "Role", "admin",
		[]map[string]string{
			{"api_group": "rbac.authorization.k8s.io", "kind": "User", "name": "alice", "namespace": "default"},
		},
	)

	req := resource.MoveStateRequest{
		SourceProviderAddress: "registry.terraform.io/other-org/other-provider",
		SourceTypeName:        "kubernetes_role_binding",
		SourceSchemaVersion:   0,
		SourceRawState:        &tfprotov6.RawState{JSON: raw},
	}
	resp := runMoveStateWithReq(t, req)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no errors for wrong provider address, got: %s", resp.Diagnostics)
	}

	var m rbacv1.RoleBindingModel
	diags := resp.TargetState.Get(context.Background(), &m)
	if !diags.HasError() && m.ID.ValueString() != "" {
		t.Errorf("expected empty target state for wrong provider address, got id=%q", m.ID.ValueString())
	}
}

// TestMigration_MoveState_wrongSchemaVersionIsIgnored verifies that the handler
// returns early without error or writing state when SourceSchemaVersion != 0.
func TestMigration_MoveState_wrongSchemaVersionIsIgnored(t *testing.T) {
	t.Parallel()

	raw := sdkv2RawJSON(
		"default/my-binding", "my-binding", "", "default",
		nil, nil, "1", "uid-8", 0,
		"rbac.authorization.k8s.io", "Role", "admin",
		[]map[string]string{
			{"api_group": "rbac.authorization.k8s.io", "kind": "User", "name": "alice", "namespace": "default"},
		},
	)

	req := resource.MoveStateRequest{
		SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes",
		SourceTypeName:        "kubernetes_role_binding",
		SourceSchemaVersion:   1, // not 0
		SourceRawState:        &tfprotov6.RawState{JSON: raw},
	}
	resp := runMoveStateWithReq(t, req)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no errors for wrong schema version, got: %s", resp.Diagnostics)
	}

	var m rbacv1.RoleBindingModel
	diags := resp.TargetState.Get(context.Background(), &m)
	if !diags.HasError() && m.ID.ValueString() != "" {
		t.Errorf("expected empty target state for wrong schema version, got id=%q", m.ID.ValueString())
	}
}

// TestMigration_MoveState_nilRawStateIsIgnored verifies that the handler
// returns early without error or panic when SourceRawState is nil.
func TestMigration_MoveState_nilRawStateIsIgnored(t *testing.T) {
	t.Parallel()

	req := resource.MoveStateRequest{
		SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes",
		SourceTypeName:        "kubernetes_role_binding",
		SourceSchemaVersion:   0,
		SourceRawState:        nil,
	}
	resp := runMoveStateWithReq(t, req)

	if resp.Diagnostics.HasError() {
		t.Errorf("expected no errors for nil raw state, got: %s", resp.Diagnostics)
	}

	var m rbacv1.RoleBindingModel
	diags := resp.TargetState.Get(context.Background(), &m)
	if !diags.HasError() && m.ID.ValueString() != "" {
		t.Errorf("expected empty target state for nil raw state, got id=%q", m.ID.ValueString())
	}
}

// TestAccRoleBindingV1_moved provisions the deprecated kubernetes_role_binding
// with the last SDKv2 release then uses a moved block to migrate state to
// kubernetes_role_binding_v1 with the Framework provider. The plan must be
// empty — proving MoveState translates the state without drift.
//
// Skipped in -short mode because it downloads from the Terraform registry.
func TestAccRoleBinding_moved(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping registry-dependent moved-block test in -short mode")
	}

	name := testacctest.RandomWithPrefix("tf-acc-rb")

	tfresource.ParallelTest(t, tfresource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
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
