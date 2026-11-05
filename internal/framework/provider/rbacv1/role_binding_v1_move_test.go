// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1_test

import (
	"context"
	"fmt"
	"testing"

	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/rbacv1"
)

func TestAccMoveRoleBinding(t *testing.T) {
	for _, version := range []string{migratedRoleBindingProviderVersion, preIdentityRoleBindingProviderVersion} {
		for _, tc := range []struct {
			name      string
			shape     roleBindingMoveShape
			noRefresh bool
		}{
			{name: "basic", shape: roleBindingMoveShape{}},
			{name: "generateName", shape: roleBindingMoveShape{generated: true}},
			{name: "multipleSubjects", shape: roleBindingMoveShape{multiple: true}},
			{name: "clusterRoleRef", shape: roleBindingMoveShape{roleKind: "ClusterRole"}},
			{name: "emptySubjectNamespace", shape: roleBindingMoveShape{emptySubjectNamespace: true}},
			{name: "emptyUserAPIGroup", shape: roleBindingMoveShape{emptyAPIGroup: true}},
			{name: "metadata", shape: roleBindingMoveShape{complete: true}},
			{name: "emptyMetadataMaps", shape: roleBindingMoveShape{emptyMaps: true}},
			{name: "noRefresh", shape: roleBindingMoveShape{}, noRefresh: true},
		} {
			t.Run(version+"/"+tc.name, func(t *testing.T) {
				name := acctest.RandomWithPrefix("tf-acc-rb")
				resourceName := "kubernetes_role_binding_v1.test"
				checkUID := testAccRoleBindingCheckSameUID()
				checkUnchanged := testAccRoleBindingCheckUnchanged()
				updated := tc.shape
				updated.user = "bob"
				check := plancheck.ExpectEmptyPlan()
				var importIgnore []string
				if tc.shape.emptyAPIGroup || tc.shape.emptyMaps {
					check = plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)
				}
				if tc.shape.emptyAPIGroup {
					importIgnore = append(importIgnore, "subject.0.api_group")
				}
				if tc.shape.emptyMaps {
					importIgnore = append(importIgnore, "metadata.0.annotations", "metadata.0.labels")
				}
				resource.ParallelTest(t, resource.TestCase{
					PreCheck:               func() { testAccPreCheck(t) },
					CheckDestroy:           testAccRoleBindingCheckDestroy,
					AdditionalCLIOptions:   &resource.AdditionalCLIOptions{Plan: resource.PlanOptions{NoRefresh: tc.noRefresh}},
					TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_8_0)},
					Steps: []resource.TestStep{
						{
							ExternalProviders: map[string]resource.ExternalProvider{
								"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: version},
							},
							Config: testAccRoleBindingMoveConfig("kubernetes_role_binding", name, tc.shape),
							Check:  resource.ComposeAggregateTestCheckFunc(checkUID, checkUnchanged),
						},
						{
							ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
							Config:                   testAccRoleBindingMoveConfig("kubernetes_role_binding_v1", name, tc.shape),
							Check:                    checkUnchanged,
							ConfigPlanChecks: resource.ConfigPlanChecks{
								PreApply:             []plancheck.PlanCheck{check},
								PostApplyPreRefresh:  []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
								PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
							},
						},
						{
							ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
							Config:                   testAccRoleBindingMoveConfig("kubernetes_role_binding_v1", name, updated),
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr(resourceName, "subject.0.name", "bob"),
								checkUID,
							),
							ConfigPlanChecks: resource.ConfigPlanChecks{
								PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)},
								PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
							},
						},
						{
							ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
							ResourceName:             resourceName,
							ImportState:              true,
							ImportStateVerify:        true,
							ImportStateVerifyIgnore:  importIgnore,
						},
					},
				})
			})
		}
	}
}

func TestAccMoveRoleBinding_identity(t *testing.T) {
	for _, version := range []string{migratedRoleBindingProviderVersion, preIdentityRoleBindingProviderVersion} {
		for _, generated := range []bool{false, true} {
			for _, complete := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/generated=%t/complete=%t", version, generated, complete), func(t *testing.T) {
					name := acctest.RandomWithPrefix("tf-acc-rb")
					shape := roleBindingMoveShape{generated: generated, complete: complete}
					resource.ParallelTest(t, resource.TestCase{
						PreCheck:               func() { testAccPreCheck(t) },
						CheckDestroy:           testAccRoleBindingCheckDestroy,
						TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)},
						Steps: []resource.TestStep{
							{
								ExternalProviders: map[string]resource.ExternalProvider{
									"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: version},
								},
								Config: testAccRoleBindingMoveConfig("kubernetes_role_binding", name, shape),
							},
							{
								ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
								Config:                   testAccRoleBindingMoveConfig("kubernetes_role_binding_v1", name, shape),
								ConfigPlanChecks:         resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
								ConfigStateChecks: []statecheck.StateCheck{
									statecheck.ExpectIdentityValueMatchesStateAtPath("kubernetes_role_binding_v1.test", tfjsonpath.New("name"), tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("name")),
									statecheck.ExpectIdentityValue("kubernetes_role_binding_v1.test", tfjsonpath.New("namespace"), knownvalue.StringExact("default")),
									statecheck.ExpectIdentityValue("kubernetes_role_binding_v1.test", tfjsonpath.New("kind"), knownvalue.StringExact("RoleBinding")),
									statecheck.ExpectIdentityValue("kubernetes_role_binding_v1.test", tfjsonpath.New("api_version"), knownvalue.StringExact("rbac.authorization.k8s.io/v1")),
								},
							},
						},
					})
				})
			}
		}
	}
}

// roleBindingMoveShape selects the config variant for the move tests. The zero
// value is a named binding with one User subject "alice" in namespace "default"
// and a Role ref.
type roleBindingMoveShape struct {
	generated, complete, multiple, emptyMaps bool
	emptySubjectNamespace, emptyAPIGroup     bool
	roleKind, user                           string
}

func testAccRoleBindingMoveConfig(resourceType, name string, shape roleBindingMoveShape) string {
	roleKind, user, namespace, apiGroup := "Role", "alice", "default", "rbac.authorization.k8s.io"
	if shape.roleKind != "" {
		roleKind = shape.roleKind
	}
	if shape.user != "" {
		user = shape.user
	}
	if shape.emptySubjectNamespace {
		namespace = ""
	}
	if shape.emptyAPIGroup {
		apiGroup = ""
	}
	generated, complete, multiple := shape.generated, shape.complete, shape.multiple
	nameAttribute := "name"
	if generated {
		nameAttribute = "generate_name"
	}
	metadata := ""
	if complete {
		metadata = `labels = { environment = "test" }
    annotations = { "example.com/note" = "preserve" }`
	}
	if shape.emptyMaps {
		metadata = `labels = {}
    annotations = {}`
	}
	extraSubjects := ""
	if multiple {
		extraSubjects = `
  subject {
    kind = "ServiceAccount"
    name = "default"
    namespace = "kube-system"
    api_group = ""
  }
  subject {
    kind = "Group"
    name = "system:masters"
    api_group = "rbac.authorization.k8s.io"
  }`
	}
	move := ""
	if resourceType == "kubernetes_role_binding_v1" {
		move = `
moved {
  from = kubernetes_role_binding.test
  to = kubernetes_role_binding_v1.test
}
`
	}
	return fmt.Sprintf(`
%s
resource %q "test" {
  metadata {
    %s = %q
    namespace = "default"
    %s
  }
  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind = %q
    name = "admin"
  }
  subject {
    kind = "User"
    name = %q
    namespace = %q
    api_group = %q
  }
  %s
}
`, move, resourceType, nameAttribute, name, metadata, roleKind, user, namespace, apiGroup, extraSubjects)
}

func TestRoleBindingMoveState(t *testing.T) {
	ctx := context.Background()
	r := &rbacv1.RoleBindingV1{}
	var schema frameworkresource.SchemaResponse
	r.Schema(ctx, frameworkresource.SchemaRequest{}, &schema)
	identitySchema := common.NamespacedIdentitySchema()
	movers := r.MoveState(ctx)
	for _, tc := range []struct {
		name, sourceType, provider, raw string
		version                         int64
		wantError                       bool
		emptyJSON                       bool
	}{
		{name: "wrongType", sourceType: "kubernetes_cluster_role_binding"},
		{name: "wrongProvider", provider: "registry.terraform.io/example/kubernetes"},
		{name: "lookalikeProvider", provider: "registry.example.com/nothashicorp/kubernetes"},
		{name: "wrongVersion", version: 1},
		{name: "nilRawState", wantError: true},
		{name: "emptyJSON", emptyJSON: true, wantError: true},
		{name: "malformedJSON", raw: "{", wantError: true},
		{name: "missingMetadata", raw: `{"id":"team/reader","role_ref":[{}]}`, wantError: true},
		{name: "multipleMetadata", raw: `{"id":"team/reader","metadata":[{},{}],"role_ref":[{}]}`, wantError: true},
		{name: "missingRoleRef", raw: `{"id":"team/reader","metadata":[{}]}`, wantError: true},
		{name: "multipleRoleRef", raw: `{"id":"team/reader","metadata":[{}],"role_ref":[{},{}]}`, wantError: true},
		{name: "emptyID", raw: `{"metadata":[{}],"role_ref":[{}]}`, wantError: true},
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
			if tc.raw != "" || tc.emptyJSON {
				req.SourceRawState = &tfprotov6.RawState{JSON: []byte(tc.raw)}
			}
			resp := frameworkresource.MoveStateResponse{
				TargetState:    tfsdk.State{Schema: schema.Schema, Raw: tftypes.NewValue(schema.Schema.Type().TerraformType(ctx), nil)},
				TargetIdentity: &tfsdk.ResourceIdentity{Schema: identitySchema, Raw: tftypes.NewValue(identitySchema.Type().TerraformType(ctx), nil)},
			}
			movers[0].StateMover(ctx, req, &resp)
			if resp.Diagnostics.HasError() != tc.wantError {
				t.Fatalf("diagnostics: %v", resp.Diagnostics)
			}
			if !resp.TargetState.Raw.IsNull() || !resp.TargetIdentity.Raw.IsNull() {
				t.Fatal("unexpected target state or identity")
			}
		})
	}
}
