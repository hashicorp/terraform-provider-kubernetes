// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/rbacv1"
)

func testAccRoleMoveTarget(config string) string {
	return config + `
moved {
  from = kubernetes_role.test
  to   = kubernetes_role_v1.test
}
`
}

func TestAccRole_movedFromAlias_scenarios(t *testing.T) {
	testAccRoleScenarios(t, testAccRoleMoveRunner(roleSDKv2ProviderVersion))
}

func TestAccRole_movedFromAlias_preIdentity(t *testing.T) {
	testAccRolePreIdentityScenarios(t, testAccRoleMoveRunner(rolePreIdentityProviderVersion))
}

func TestAccRole_movedFromAlias_nullMetadata(t *testing.T) {
	testAccRoleNullMetadataMigration(t, "kubernetes_role", testAccRoleMoveTarget)
}

func testAccRoleMoveRunner(version string) roleScenarioRunner {
	return func(t *testing.T, config string, update bool, internalMetadata string, checks ...resource.TestCheckFunc) {
		var uid string
		plan := plancheck.ExpectEmptyPlan()
		if update {
			plan = plancheck.ExpectResourceAction("kubernetes_role_v1.test", plancheck.ResourceActionUpdate)
		}
		checks = append(checks, testAccRoleCheckExists("kubernetes_role_v1.test", &uid),
			resource.TestCheckResourceAttr("kubernetes_role_v1.test", "metadata.0.namespace", "default"),
			resource.TestCheckResourceAttr("kubernetes_role_v1.test", "rule.#", "1"),
			resource.TestCheckTypeSetElemAttr("kubernetes_role_v1.test", "rule.0.resources.*", "pods"))
		testAccRoleValidateAndRun(t, resource.TestCase{
			CheckDestroy:           testAccRoleCheckDestroy,
			TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_8_0)},
			Steps: roleImportSteps("kubernetes_role_v1.test", []resource.TestStep{
				{
					ExternalProviders: roleExternalProvider(version),
					Config:            strings.ReplaceAll(config, `"kubernetes_role_v1"`, `"kubernetes_role"`),
					Check:             testAccRoleCheckExists("kubernetes_role.test", &uid),
				},
				{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Config:                   testAccRoleMoveTarget(config),
					ConfigPlanChecks:         resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plan}},
					Check:                    resource.ComposeAggregateTestCheckFunc(checks...),
				},
				{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Config:                   config,
					ConfigPlanChecks:         resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
				},
			}, internalMetadata),
		})
	}
}

func TestAccRole_movedFromAlias_names(t *testing.T) {
	name := "tf-acc-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	config := testAccRoleConfig_names(name)
	testAccRoleMoveWithUpdate(t, name, config, strings.Replace(config, `verbs      = ["get"]`, `verbs      = ["get", "list"]`, 1), "list", true)
}

func TestAccRole_movedFromAlias(t *testing.T) {
	name := "tf-acc-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	config := fmt.Sprintf(`
resource "kubernetes_role_v1" "test" {
  metadata {
    name      = %q
    namespace = "default"
  }

  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["get", "list"]
  }
  rule {
    api_groups = ["apps"]
    resources  = ["deployments"]
    verbs      = ["get"]
  }
}
`, name)
	updated := fmt.Sprintf(`
resource "kubernetes_role_v1" "test" {
  metadata {
    name      = %q
    namespace = "default"
  }
  rule {
    api_groups = ["batch"]
    resources  = ["jobs"]
    verbs      = ["get", "list", "watch"]
  }
}
`, name)
	testAccRoleMoveWithUpdate(t, name, config, updated, "list", false)
}

func TestAccRole_movedFromAlias_upgradeBaseline(t *testing.T) {
	name := "tf-acc-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	testAccRoleMoveWithUpdate(t, name, testAccRoleConfig_basic(name), testAccRoleConfig_modified(name), "watch", false)
}

func testAccRoleMoveWithUpdate(t *testing.T, name, config, updated, verb string, localAlias bool) {
	t.Helper()
	var uid string
	resourceName := "kubernetes_role_v1.test"
	source := resource.TestStep{
		ExternalProviders: roleExternalProvider(roleSDKv2ProviderVersion),
		Config:            strings.ReplaceAll(config, `"kubernetes_role_v1"`, `"kubernetes_role"`),
		Check: resource.ComposeAggregateTestCheckFunc(
			testAccRoleCheckExists("kubernetes_role.test", &uid),
			resource.TestCheckResourceAttr("kubernetes_role.test", "metadata.0.name", name),
			resource.TestCheckResourceAttrSet("kubernetes_role.test", "metadata.0.uid")),
	}
	if localAlias {
		source.ExternalProviders = nil
		source.ProtoV6ProviderFactories = testAccMuxProtoV6ProviderFactories
	}
	updateChecks := []resource.TestCheckFunc{
		testAccRoleCheckExists(resourceName, &uid),
		resource.TestCheckResourceAttr(resourceName, "rule.#", "1"),
	}
	updateChecks = append(updateChecks, resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.verbs.*", verb))
	testAccRoleValidateAndRun(t, resource.TestCase{
		CheckDestroy:           testAccRoleCheckDestroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_8_0)},
		Steps: roleImportSteps(resourceName, []resource.TestStep{
			source,
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccRoleMoveTarget(config),
				ConfigPlanChecks:         resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccRoleCheckExists(resourceName, &uid),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid")),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   updated,
				ConfigPlanChecks:         resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)}},
				Check:                    resource.ComposeAggregateTestCheckFunc(updateChecks...),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   updated,
				ConfigPlanChecks:         resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
			},
		}),
	})
}

func TestRoleMoveState(t *testing.T) {
	ctx := context.Background()
	r := &rbacv1.RoleV1{}
	var schema frameworkresource.SchemaResponse
	r.Schema(ctx, frameworkresource.SchemaRequest{}, &schema)
	identitySchema := common.NamespacedIdentitySchema()
	valid := `{"id":"team/reader","metadata":[{"name":"reader","namespace":"team","uid":"retained","labels":{},"annotations":null,"generate_name":""}],"rule":[{"api_groups":[""],"resources":["pods"],"verbs":["get"],"resource_names":[]}]}`
	for _, tc := range []struct {
		name, sourceType, provider, raw string
		version                         int64
		wantError, wantSkip, noIdentity bool
	}{
		{name: "valid", raw: valid},
		{name: "generatedName", raw: strings.Replace(valid, `"generate_name":""`, `"generate_name":"reader-"`, 1)},
		{name: "withoutTargetIdentity", raw: valid, noIdentity: true},
		{name: "privateRegistry", provider: "registry.example.com/hashicorp/kubernetes", raw: valid},
		{name: "wrongType", sourceType: "kubernetes_cluster_role", raw: valid, wantSkip: true},
		{name: "wrongProvider", provider: "registry.terraform.io/example/kubernetes", raw: valid, wantSkip: true},
		{name: "lookalikeProvider", provider: "registry.example.com/nothashicorp/kubernetes", raw: valid, wantSkip: true},
		{name: "wrongVersion", version: 1, raw: valid, wantSkip: true},
		{name: "missingJSON", wantError: true},
		{name: "malformedJSON", raw: "{", wantError: true},
		{name: "emptyID", raw: `{"metadata":[{}]}`, wantError: true},
		{name: "missingMetadata", raw: `{"id":"team/reader"}`, wantError: true},
		{name: "multipleMetadata", raw: `{"id":"team/reader","metadata":[{},{}]}`, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := frameworkresource.MoveStateRequest{
				SourceTypeName: "kubernetes_role", SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes",
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
			r.MoveState(ctx)[0].StateMover(ctx, req, &resp)
			if resp.Diagnostics.HasError() != tc.wantError {
				t.Fatalf("diagnostics: %v", resp.Diagnostics)
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
			var state rbacv1.RoleModel
			if diags := resp.TargetState.Get(ctx, &state); diags.HasError() {
				t.Fatal(diags)
			}
			if state.ID.ValueString() != "team/reader" || state.Metadata[0].Namespace.ValueString() != "team" ||
				state.Metadata[0].UID.ValueString() != "retained" ||
				!state.Metadata[0].Annotations.IsNull() || state.Metadata[0].Labels.IsNull() ||
				len(state.Rule) != 1 || len(state.Rule[0].Verbs.Elements()) != 1 {
				t.Fatalf("incorrect state: %#v", state)
			}
			if tc.name == "generatedName" {
				if state.Metadata[0].GenerateName.ValueString() != "reader-" {
					t.Fatal("lost generate_name")
				}
			} else if !state.Metadata[0].GenerateName.IsNull() {
				t.Fatal("unset generate_name must normalize to null")
			}
			if resp.TargetIdentity != nil {
				var identity common.NamespacedResourceIdentity
				if diags := resp.TargetIdentity.Get(ctx, &identity); diags.HasError() {
					t.Fatal(diags)
				}
				if identity.Kind.ValueString() != "Role" || identity.APIVersion.ValueString() != "rbac.authorization.k8s.io/v1" ||
					identity.Name.ValueString() != "reader" || identity.Namespace.ValueString() != "team" {
					t.Fatalf("identity: %#v", identity)
				}
			}
		})
	}
	// Ensure the fixture itself is JSON, rather than testing a decode failure accidentally.
	if !json.Valid([]byte(valid)) {
		t.Fatal("invalid fixture")
	}
}
