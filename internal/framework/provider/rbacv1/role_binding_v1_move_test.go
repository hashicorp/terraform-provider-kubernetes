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
	for _, tc := range []struct {
		name                                 string
		generated, complete, noRefresh       bool
		subjectNamespace, apiGroup, roleKind string
		multiple                             bool
	}{
		{name: "basic", subjectNamespace: "default", apiGroup: "rbac.authorization.k8s.io", roleKind: "Role"},
		{name: "generateName", generated: true, subjectNamespace: "default", apiGroup: "rbac.authorization.k8s.io", roleKind: "Role"},
		{name: "multipleSubjects", multiple: true, subjectNamespace: "default", apiGroup: "rbac.authorization.k8s.io", roleKind: "Role"},
		{name: "clusterRoleRef", subjectNamespace: "default", apiGroup: "rbac.authorization.k8s.io", roleKind: "ClusterRole"},
		{name: "emptySubjectNamespace", subjectNamespace: "", apiGroup: "rbac.authorization.k8s.io", roleKind: "Role"},
		{name: "emptyUserAPIGroup", subjectNamespace: "default", apiGroup: "", roleKind: "Role"},
		{name: "metadata", complete: true, subjectNamespace: "default", apiGroup: "rbac.authorization.k8s.io", roleKind: "Role"},
		{name: "noRefresh", noRefresh: true, subjectNamespace: "default", apiGroup: "rbac.authorization.k8s.io", roleKind: "Role"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name := acctest.RandomWithPrefix("tf-acc-rb")
			config := testAccRoleBindingMoveConfig("kubernetes_role_binding", name, tc.generated, tc.complete, tc.multiple, tc.subjectNamespace, tc.apiGroup, tc.roleKind)
			moved := testAccRoleBindingMoveConfig("kubernetes_role_binding_v1", name, tc.generated, tc.complete, tc.multiple, tc.subjectNamespace, tc.apiGroup, tc.roleKind)
			check := plancheck.ExpectEmptyPlan()
			if tc.apiGroup == "" {
				check = plancheck.ExpectResourceAction("kubernetes_role_binding_v1.test", plancheck.ResourceActionUpdate)
			}
			resource.ParallelTest(t, resource.TestCase{
				PreCheck:               func() { testAccPreCheck(t) },
				CheckDestroy:           testAccRoleBindingCheckDestroy,
				AdditionalCLIOptions:   &resource.AdditionalCLIOptions{Plan: resource.PlanOptions{NoRefresh: tc.noRefresh}},
				TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_8_0)},
				Steps: []resource.TestStep{
					{ExternalProviders: releasedRoleBindingProvider(), Config: config},
					{
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Config:                   moved,
						ConfigPlanChecks:         resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{check}},
					},
				},
			})
		})
	}
}

func TestAccMoveRoleBinding_identity(t *testing.T) {
	for _, version := range []string{migratedRoleBindingProviderVersion, preIdentityRoleBindingProviderVersion} {
		for _, generated := range []bool{false, true} {
			for _, complete := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/generated=%t/complete=%t", version, generated, complete), func(t *testing.T) {
					name := acctest.RandomWithPrefix("tf-acc-rb")
					resource.ParallelTest(t, resource.TestCase{
						PreCheck:               func() { testAccPreCheck(t) },
						CheckDestroy:           testAccRoleBindingCheckDestroy,
						TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)},
						Steps: []resource.TestStep{
							{
								ExternalProviders: map[string]resource.ExternalProvider{
									"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: version},
								},
								Config: testAccRoleBindingMoveConfig("kubernetes_role_binding", name, generated, complete, false, "default", "rbac.authorization.k8s.io", "Role"),
							},
							{
								ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
								Config:                   testAccRoleBindingMoveConfig("kubernetes_role_binding_v1", name, generated, complete, false, "default", "rbac.authorization.k8s.io", "Role"),
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

func testAccRoleBindingMoveConfig(resourceType, name string, generated, complete, multiple bool, namespace, apiGroup, roleKind string) string {
	nameAttribute := "name"
	if generated {
		nameAttribute = "generate_name"
	}
	metadata := ""
	if complete {
		metadata = `labels = { environment = "test" }
    annotations = { "example.com/note" = "preserve" }`
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
    name = "alice"
    namespace = %q
    api_group = %q
  }
  %s
}
`, move, resourceType, nameAttribute, name, metadata, roleKind, namespace, apiGroup, extraSubjects)
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
