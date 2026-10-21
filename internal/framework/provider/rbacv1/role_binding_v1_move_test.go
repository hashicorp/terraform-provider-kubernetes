// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1_test

// ---------------------------------------------------------------------------
// MoveState tests for kubernetes_role_binding_v1.
//
// Cover migrating from the deprecated kubernetes_role_binding (SDKv2) resource
// to kubernetes_role_binding_v1 (Framework) with a moved block:
//
//	moved {
//	  from = kubernetes_role_binding.example
//	  to   = kubernetes_role_binding_v1.example
//	}
//
// Acceptance tests (TestAccMoveRoleBinding_*) create state with the released
// SDKv2 provider, add a moved block, switch to the local Framework provider,
// and assert an empty post-move plan. TestRoleBindingMoveState calls the
// handler directly for paths Terraform cannot reach, following
// TestRoleMoveState and TestNamespaceMoveState.
// ---------------------------------------------------------------------------

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	rbacv1 "github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/rbacv1"
)

// TestAccMoveRoleBinding_basic verifies that a binding created as
// kubernetes_role_binding can be moved to kubernetes_role_binding_v1 via a
// moved block with no diff.
func TestAccMoveRoleBinding_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccRoleBindingCheckDestroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		Steps: []resource.TestStep{
			{
				// Step 1 — create with the released SDKv2 provider using the
				// deprecated kubernetes_role_binding resource type.
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						Source:            "hashicorp/kubernetes",
						VersionConstraint: "3.2.1",
					},
				},
				Config: testAccKubernetesRoleBindingConfig_deprecatedType(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"kubernetes_role_binding.test",
						"metadata.0.name", name,
					),
					resource.TestCheckResourceAttr(
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
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"kubernetes_role_binding_v1.test",
						"metadata.0.name", name,
					),
					resource.TestCheckResourceAttr(
						"kubernetes_role_binding_v1.test",
						"subject.#", "1",
					),
					resource.TestCheckResourceAttr(
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
				ConfigPlanChecks: resource.ConfigPlanChecks{
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
// moved block with no diff.
func TestAccMoveRoleBinding_generateName(t *testing.T) {
	prefix := "tf-acc-rb-"

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccRoleBindingCheckDestroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						Source:            "hashicorp/kubernetes",
						VersionConstraint: "3.2.1",
					},
				},
				Config: testAccKubernetesRoleBindingConfig_deprecatedTypeGenerateName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"kubernetes_role_binding.test",
						"metadata.0.generate_name", prefix,
					),
					resource.TestCheckResourceAttrSet(
						"kubernetes_role_binding.test",
						"metadata.0.name",
					),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesRoleBindingConfig_movedToV1GenerateName(prefix),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"kubernetes_role_binding_v1.test",
						"metadata.0.generate_name", prefix,
					),
					resource.TestCheckResourceAttrSet(
						"kubernetes_role_binding_v1.test",
						"metadata.0.name",
					),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesRoleBindingConfig_movedToV1GenerateName(prefix),
				ConfigPlanChecks: resource.ConfigPlanChecks{
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
// block with no diff.
func TestAccMoveRoleBinding_multipleSubjects(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccRoleBindingCheckDestroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						Source:            "hashicorp/kubernetes",
						VersionConstraint: "3.2.1",
					},
				},
				Config: testAccKubernetesRoleBindingConfig_deprecatedTypeMultiSubject(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"kubernetes_role_binding.test",
						"subject.#", "3",
					),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesRoleBindingConfig_movedToV1MultiSubject(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"kubernetes_role_binding_v1.test",
						"subject.#", "3",
					),
					resource.TestCheckResourceAttr(
						"kubernetes_role_binding_v1.test",
						"subject.0.kind", "User",
					),
					resource.TestCheckResourceAttr(
						"kubernetes_role_binding_v1.test",
						"subject.1.kind", "ServiceAccount",
					),
					resource.TestCheckResourceAttr(
						"kubernetes_role_binding_v1.test",
						"subject.2.kind", "Group",
					),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesRoleBindingConfig_movedToV1MultiSubject(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
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
func TestAccMoveRoleBinding_clusterRoleRef(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccRoleBindingCheckDestroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						Source:            "hashicorp/kubernetes",
						VersionConstraint: "3.2.1",
					},
				},
				Config: testAccKubernetesRoleBindingConfig_deprecatedTypeClusterRoleRef(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"kubernetes_role_binding.test",
						"role_ref.0.kind", "ClusterRole",
					),
					resource.TestCheckResourceAttr(
						"kubernetes_role_binding.test",
						"role_ref.0.name", "cluster-admin",
					),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesRoleBindingConfig_movedToV1ClusterRoleRef(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"kubernetes_role_binding_v1.test",
						"role_ref.0.kind", "ClusterRole",
					),
					resource.TestCheckResourceAttr(
						"kubernetes_role_binding_v1.test",
						"role_ref.0.name", "cluster-admin",
					),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesRoleBindingConfig_movedToV1ClusterRoleRef(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// TestAccMoveRoleBinding_identity verifies that after a moved block the
// Framework resource has a complete identity and no plan diff. Terraform
// refreshes during apply and Read also sets identity, so this checks the
// end result rather than MoveState's TargetIdentity in isolation. Kept separate
// because identity requires Terraform 1.12, while the other move tests run
// from 1.8.
func TestAccMoveRoleBinding_identity(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccRoleBindingCheckDestroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						Source:            "hashicorp/kubernetes",
						VersionConstraint: "3.2.1",
					},
				},
				Config: testAccKubernetesRoleBindingConfig_deprecatedType(name),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesRoleBindingConfig_movedToV1(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					testAccRoleBindingExpectIdentity("kubernetes_role_binding_v1.test", name),
				},
			},
		},
	})
}

// TestAccMoveRoleBinding_preIdentity verifies a moved block from a
// kubernetes_role_binding written before resource identity existed: the source
// has no identity, the move plans no diff, and the target gets a complete
// identity.
func TestAccMoveRoleBinding_preIdentity(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccRoleBindingCheckDestroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						Source:            "hashicorp/kubernetes",
						VersionConstraint: preIdentityRoleBindingProviderVersion,
					},
				},
				Config: testAccKubernetesRoleBindingConfig_deprecatedType(name),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesRoleBindingConfig_movedToV1(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					testAccRoleBindingExpectIdentity("kubernetes_role_binding_v1.test", name),
				},
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccKubernetesRoleBindingConfig_movedToV1(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
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

// TestRoleBindingMoveState covers MoveState paths acceptance tests cannot
// reach: early returns for other sources, malformed source state, null versus
// empty maps, and the target identity.
func TestRoleBindingMoveState(t *testing.T) {
	ctx := context.Background()
	r := &rbacv1.RoleBindingV1{}
	var schema frameworkresource.SchemaResponse
	r.Schema(ctx, frameworkresource.SchemaRequest{}, &schema)
	identitySchema := common.NamespacedIdentitySchema()
	movers := r.MoveState(ctx)
	if len(movers) != 1 {
		t.Fatalf("expected 1 state mover, got %d", len(movers))
	}
	valid := `{"id":"team/reader","metadata":[{"name":"reader","namespace":"team","uid":"retained","resource_version":"42","generation":3,"labels":{},"annotations":null,"generate_name":""}],` +
		`"role_ref":[{"api_group":"rbac.authorization.k8s.io","kind":"Role","name":"reader"}],` +
		`"subject":[{"api_group":"rbac.authorization.k8s.io","kind":"User","name":"alice","namespace":""},` +
		`{"api_group":"","kind":"ServiceAccount","name":"builder","namespace":"kube-system"},` +
		`{"api_group":"rbac.authorization.k8s.io","kind":"Group","name":"devs","namespace":""}]}`
	for _, tc := range []struct {
		name, sourceType, provider, raw string
		version                         int64
		wantError, wantSkip, noIdentity bool
		wantDetail                      string
		nullLabels, emptyAnnotations    bool
		wantGenerateName                string
		wantRoleRefKind                 string
	}{
		{name: "valid", raw: valid},
		{name: "bothNullMaps", raw: strings.Replace(valid, `"labels":{}`, `"labels":null`, 1), nullLabels: true},
		{name: "bothEmptyMaps", raw: strings.Replace(valid, `"annotations":null`, `"annotations":{}`, 1), emptyAnnotations: true},
		{name: "omittedMaps", raw: strings.ReplaceAll(strings.ReplaceAll(valid, `"labels":{},`, ""), `"annotations":null,`, ""), nullLabels: true},
		{name: "generatedName", raw: strings.Replace(valid, `"generate_name":""`, `"generate_name":"reader-"`, 1), wantGenerateName: "reader-"},
		{name: "clusterRoleRef", raw: strings.Replace(valid, `"kind":"Role","name":"reader"`, `"kind":"ClusterRole","name":"reader"`, 1), wantRoleRefKind: "ClusterRole"},
		{name: "withoutTargetIdentity", raw: valid, noIdentity: true},
		{name: "privateRegistry", provider: "registry.example.com/hashicorp/kubernetes", raw: valid},
		{name: "wrongType", sourceType: "kubernetes_cluster_role_binding", raw: valid, wantSkip: true},
		{name: "wrongProvider", provider: "registry.terraform.io/example/kubernetes", raw: valid, wantSkip: true},
		{name: "lookalikeProvider", provider: "registry.example.com/nothashicorp/kubernetes", raw: valid, wantSkip: true},
		{name: "wrongVersion", version: 1, raw: valid, wantSkip: true},
		{name: "missingJSON", wantSkip: true},
		{name: "malformedJSON", raw: "{", wantError: true},
		{name: "missingMetadata", raw: `{"id":"team/reader","role_ref":[{}]}`, wantError: true, wantDetail: "expected exactly 1 metadata element in source state, got 0"},
		{name: "multipleMetadata", raw: `{"id":"team/reader","metadata":[{},{}],"role_ref":[{}]}`, wantError: true, wantDetail: "expected exactly 1 metadata element in source state, got 2"},
		{name: "missingRoleRef", raw: `{"id":"team/reader","metadata":[{}]}`, wantError: true, wantDetail: "expected exactly 1 role_ref element in source state, got 0"},
		{name: "emptyID", raw: `{"metadata":[{}],"role_ref":[{}]}`, wantError: true, wantDetail: "empty 'id' in source state"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := frameworkresource.MoveStateRequest{
				SourceTypeName: "kubernetes_role_binding", SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes",
				SourceSchemaVersion: tc.version,
			}
			if tc.sourceType != "" {
				req.SourceTypeName = tc.sourceType
			}
			if tc.provider != "" {
				req.SourceProviderAddress = tc.provider
			}
			if tc.raw != "" {
				req.SourceRawState = &tfprotov6.RawState{JSON: []byte(tc.raw)}
			}
			resp := frameworkresource.MoveStateResponse{
				TargetState: tfsdk.State{Schema: schema.Schema, Raw: tftypes.NewValue(schema.Schema.Type().TerraformType(ctx), nil)},
			}
			if !tc.noIdentity {
				resp.TargetIdentity = &tfsdk.ResourceIdentity{Schema: identitySchema, Raw: tftypes.NewValue(identitySchema.Type().TerraformType(ctx), nil)}
			}
			movers[0].StateMover(ctx, req, &resp)
			if resp.Diagnostics.HasError() != tc.wantError {
				t.Fatalf("diagnostics: %v", resp.Diagnostics)
			}
			if tc.wantDetail != "" && (len(resp.Diagnostics) != 1 || resp.Diagnostics[0].Detail() != tc.wantDetail) {
				t.Fatalf("expected diagnostic detail %q, got %v", tc.wantDetail, resp.Diagnostics)
			}
			if tc.wantError || tc.wantSkip {
				if !resp.TargetState.Raw.IsNull() {
					t.Fatal("unexpected target state")
				}
				if resp.TargetIdentity != nil && !resp.TargetIdentity.Raw.IsNull() {
					t.Fatal("unexpected target identity")
				}
				return
			}

			var state rbacv1.RoleBindingModel
			if diags := resp.TargetState.Get(ctx, &state); diags.HasError() {
				t.Fatal(diags)
			}
			meta := state.Metadata[0]
			if state.ID.ValueString() != "team/reader" || meta.Name.ValueString() != "reader" || meta.Namespace.ValueString() != "team" ||
				meta.UID.ValueString() != "retained" || meta.ResourceVersion.ValueString() != "42" || meta.Generation.ValueInt64() != 3 ||
				meta.Annotations.IsNull() == tc.emptyAnnotations || meta.Labels.IsNull() != tc.nullLabels {
				t.Fatalf("incorrect metadata: %#v", state)
			}
			if tc.wantGenerateName != "" {
				if meta.GenerateName.ValueString() != tc.wantGenerateName {
					t.Fatalf("generate_name = %v, want %q", meta.GenerateName, tc.wantGenerateName)
				}
			} else if !meta.GenerateName.IsNull() {
				t.Fatal("unset generate_name must normalize to null")
			}
			wantRoleRefKind := "Role"
			if tc.wantRoleRefKind != "" {
				wantRoleRefKind = tc.wantRoleRefKind
			}
			if len(state.RoleRef) != 1 || state.RoleRef[0].Kind.ValueString() != wantRoleRefKind ||
				state.RoleRef[0].Name.ValueString() != "reader" || state.RoleRef[0].APIGroup.ValueString() != "rbac.authorization.k8s.io" {
				t.Fatalf("incorrect role_ref: %#v", state.RoleRef)
			}
			wantSubjects := []rbacv1.SubjectModel{
				{APIGroup: types.StringValue("rbac.authorization.k8s.io"), Kind: types.StringValue("User"), Name: types.StringValue("alice"), Namespace: types.StringValue("")},
				{APIGroup: types.StringValue(""), Kind: types.StringValue("ServiceAccount"), Name: types.StringValue("builder"), Namespace: types.StringValue("kube-system")},
				{APIGroup: types.StringValue("rbac.authorization.k8s.io"), Kind: types.StringValue("Group"), Name: types.StringValue("devs"), Namespace: types.StringValue("")},
			}
			if len(state.Subject) != len(wantSubjects) {
				t.Fatalf("subject count = %d, want %d", len(state.Subject), len(wantSubjects))
			}
			for i, want := range wantSubjects {
				if state.Subject[i] != want {
					t.Fatalf("subject[%d] = %#v, want %#v", i, state.Subject[i], want)
				}
			}

			if tc.noIdentity {
				return
			}
			var identity common.NamespacedResourceIdentity
			if diags := resp.TargetIdentity.Get(ctx, &identity); diags.HasError() {
				t.Fatal(diags)
			}
			if identity.Kind.ValueString() != "RoleBinding" || identity.APIVersion.ValueString() != "rbac.authorization.k8s.io/v1" ||
				identity.Name.ValueString() != "reader" || identity.Namespace.ValueString() != "team" {
				t.Fatalf("identity: %#v", identity)
			}
		})
	}
	// Ensure the fixture itself is JSON, rather than testing a decode failure accidentally.
	if !json.Valid([]byte(valid)) {
		t.Fatal("invalid fixture")
	}
}
