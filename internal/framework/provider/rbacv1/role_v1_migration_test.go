// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1_test

import (
	"context"
	"fmt"
	"regexp"
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

func roleExternalProvider(version string) map[string]resource.ExternalProvider {
	return map[string]resource.ExternalProvider{
		"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: version},
	}
}

func TestRoleUpgradeIdentity(t *testing.T) {
	ctx := context.Background()
	r := &rbacv1.RoleV1{}
	var schema frameworkresource.IdentitySchemaResponse
	r.IdentitySchema(ctx, frameworkresource.IdentitySchemaRequest{}, &schema)
	if schema.IdentitySchema.Version != 1 {
		t.Fatalf("identity version = %d; must remain compatible with SDKv2 version 1", schema.IdentitySchema.Version)
	}
	upgrader, ok := r.UpgradeIdentity(ctx)[0]
	if !ok {
		t.Fatal("missing version 0 upgrader")
	}
	for _, tc := range []struct {
		name, raw string
		wantError bool
	}{
		{name: "preIdentity"},
		{name: "stored", raw: `{"name":"reader","namespace":"team"}`},
		{name: "invalidJSON", raw: `{`, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := frameworkresource.UpgradeIdentityRequest{}
			if tc.raw != "" {
				req.RawIdentity = &tfprotov6.RawState{JSON: []byte(tc.raw)}
			}
			resp := frameworkresource.UpgradeIdentityResponse{Identity: &tfsdk.ResourceIdentity{
				Schema: schema.IdentitySchema,
				Raw:    tftypes.NewValue(schema.IdentitySchema.Type().TerraformType(ctx), nil),
			}}
			upgrader.IdentityUpgrader(ctx, req, &resp)
			if resp.Diagnostics.HasError() != tc.wantError {
				t.Fatalf("diagnostics: %v", resp.Diagnostics)
			}
			if tc.wantError {
				return
			}
			var got common.NamespacedResourceIdentity
			if diags := resp.Identity.Get(ctx, &got); diags.HasError() {
				t.Fatal(diags)
			}
			if tc.raw == "" {
				if !got.Name.IsNull() || !got.Namespace.IsNull() || !got.Kind.IsNull() || !got.APIVersion.IsNull() {
					t.Fatalf("pre-identity state must remain null until Read: %#v", got)
				}
			} else if got.Name.ValueString() != "reader" || got.Namespace.ValueString() != "team" ||
				got.Kind.ValueString() != "Role" || got.APIVersion.ValueString() != "rbac.authorization.k8s.io/v1" {
				t.Fatalf("incorrect upgraded identity: %#v", got)
			}
		})
	}
}

const roleSDKv2ProviderVersion = "3.2.1"

const rolePreIdentityProviderVersion = "2.37.1"

func testAccRoleMigration(t *testing.T, config string, checks ...resource.TestCheckFunc) {
	t.Helper()
	testAccRoleMigrationWithPlanCheck(t, config, plancheck.ExpectEmptyPlan(), checks...)
}

// testAccRoleMigrationExpectingUpdate is for shapes that cannot migrate to an empty plan.
// SDKv2 stored null where a config said {}, so the first framework plan reconciles the two.
// The update is non-destructive and settles after one apply; asserting the specific action
// keeps a replacement from slipping in under the same expectation.
func testAccRoleMigrationExpectingUpdate(t *testing.T, config string, checks ...resource.TestCheckFunc) {
	t.Helper()
	testAccRoleMigrationWithPlanCheck(t, config,
		plancheck.ExpectResourceAction("kubernetes_role_v1.test", plancheck.ResourceActionUpdate), checks...)
}

func testAccRoleMigrationWithPlanCheck(t *testing.T, config string, planCheck plancheck.PlanCheck, checks ...resource.TestCheckFunc) {
	testAccRoleMigrationFrom(t, roleSDKv2ProviderVersion, config, planCheck, checks...)
}

func testAccRoleMigrationFrom(t *testing.T, version, config string, planCheck plancheck.PlanCheck, checks ...resource.TestCheckFunc) {
	t.Helper()

	var uid string
	resourceName := "kubernetes_role_v1.test"
	checks = append([]resource.TestCheckFunc{
		testAccRoleCheckExists(resourceName, &uid),
		resource.TestCheckResourceAttr(resourceName, "metadata.0.namespace", "default"),
		resource.TestCheckResourceAttr(resourceName, "rule.#", "1"),
		resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.resources.*", "pods"),
	}, checks...)

	testAccRoleWithImports(t, resource.TestCase{
		CheckDestroy: testAccRoleCheckDestroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						VersionConstraint: version,
						Source:            "hashicorp/kubernetes",
					},
				},
				Config: config,
				Check:  testAccRoleCheckExists(resourceName, &uid),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						planCheck,
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(checks...),
			},
		},
	})
}

func TestAccRole_UpgradeFromSDKv2_scenarios(t *testing.T) {
	testAccRoleScenarios(t, func(t *testing.T, config string, update bool, checks ...resource.TestCheckFunc) {
		if update {
			testAccRoleMigrationExpectingUpdate(t, config, checks...)
			return
		}
		testAccRoleMigration(t, config, checks...)
	})
}

func TestAccRole_UpgradeFromSDKv2_preIdentity(t *testing.T) {
	testAccRolePreIdentityScenarios(t, func(t *testing.T, config string, update bool, checks ...resource.TestCheckFunc) {
		testAccRoleMigrationFrom(t, rolePreIdentityProviderVersion, config, plancheck.ExpectEmptyPlan(), checks...)
	})
}

type roleScenarioRunner func(*testing.T, string, bool, ...resource.TestCheckFunc)

func testAccRolePreIdentityScenarios(t *testing.T, run roleScenarioRunner) {
	testAccRoleScenarios(t, run, "minimal", "completeName", "completeGeneratedName")
}

func testAccRoleScenarios(t *testing.T, run roleScenarioRunner, selected ...string) {
	t.Helper()
	resourceName := "kubernetes_role_v1.test"
	for _, scenario := range []struct {
		name          string
		generatedName bool
		metadata      string
		resourceNames string
		checks        []resource.TestCheckFunc
	}{
		{name: "minimal"},
		{name: "generateName", generatedName: true},
		{
			name:     "annotations",
			metadata: `annotations = { note = "retained" }`,
			checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.note", "retained"),
				resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
			},
		},
		{
			name:     "labels",
			metadata: `labels = { team = "platform" }`,
			checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.team", "platform"),
				resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
			},
		},
		{
			name:     "declaredInternalLabel",
			metadata: `labels = { "example.kubernetes.io/owner" = "terraform" }`,
			checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.example.kubernetes.io/owner", "terraform"),
			},
		},
		{
			name:     "declaredInternalAnnotation",
			metadata: `annotations = { "example.kubernetes.io/owner" = "terraform" }`,
			checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.example.kubernetes.io/owner", "terraform"),
			},
		},
		{
			name: "emptyValuesName",
			metadata: `labels = {}
    annotations = {}`,
			resourceNames: `resource_names = []`,
		},
		{
			name:          "emptyValuesGeneratedName",
			generatedName: true,
			metadata: `labels = {}
    annotations = {}`,
			resourceNames: `resource_names = []`,
		},
		{
			name: "completeName",
			metadata: `labels = { team = "platform" }
    annotations = { note = "retained" }
    namespace = "default"`,
			resourceNames: `resource_names = ["one", "two"]`,
		},
		{
			name:          "completeGeneratedName",
			generatedName: true,
			metadata: `labels = { team = "platform" }
    annotations = { note = "retained" }
    namespace = "default"`,
			resourceNames: `resource_names = ["one", "two"]`,
		},
	} {
		if len(selected) > 0 {
			found := false
			for _, name := range selected {
				found = found || name == scenario.name
			}
			if !found {
				continue
			}
		}
		t.Run(scenario.name, func(t *testing.T) {
			name := "tf-migration-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
			naming := fmt.Sprintf("name = %q", name)
			checks := append([]resource.TestCheckFunc{}, scenario.checks...)
			if scenario.generatedName {
				prefix := name + "-"
				naming = fmt.Sprintf("generate_name = %q", prefix)
				checks = append(checks,
					resource.TestCheckResourceAttr(resourceName, "metadata.0.generate_name", prefix),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.name"),
				)
			} else {
				checks = append(checks, resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name))
			}
			switch scenario.name {
			case "minimal", "generateName", "emptyValuesName", "emptyValuesGeneratedName":
				checks = append(checks,
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resource_names.#", "0"),
				)
			case "completeName", "completeGeneratedName":
				checks = append(checks,
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.team", "platform"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.note", "retained"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resource_names.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.resource_names.*", "one"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.resource_names.*", "two"),
				)
			}
			config := fmt.Sprintf(`
resource "kubernetes_role_v1" "test" {
  metadata {
    %s
    %s
  }
  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["get", "list"]
    %s
  }
}
`, naming, scenario.metadata, scenario.resourceNames)
			// SDKv2 wrote null for a config declaring {}, so these two reconcile on the
			// first framework plan rather than migrating silently.
			if scenario.name == "emptyValuesName" || scenario.name == "emptyValuesGeneratedName" {
				run(t, config, true, checks...)
				return
			}
			run(t, config, false, checks...)
		})
	}
}

func TestAccRole_UpgradeFromSDKv2(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_role_v1.test"
	var uid string

	testAccRoleWithImports(t, resource.TestCase{
		CheckDestroy: testAccRoleCheckDestroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						VersionConstraint: roleSDKv2ProviderVersion,
						Source:            "hashicorp/kubernetes",
					},
				},
				Config: testAccRoleConfig_basic(name),
				Check: resource.ComposeTestCheckFunc(
					testAccRoleCheckExists(resourceName, &uid),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "rule.#", "2"),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccRoleConfig_basic(name),
				Check:                    testAccRoleCheckExists(resourceName, &uid),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccRoleConfig_modified(name),
				Check: resource.ComposeTestCheckFunc(
					testAccRoleCheckExists(resourceName, &uid),
					resource.TestCheckResourceAttr(resourceName, "rule.#", "1"),
				),
			},
		},
	})
}

func TestAccRole_UpgradeFromSDKv2_nullMetadata(t *testing.T) {
	testAccRoleNullMetadataMigration(t, "kubernetes_role_v1", func(config string) string { return config })
}

func testAccRoleNullMetadataMigration(t *testing.T, sourceType string, targetConfig func(string) string) {
	// SDKv2 accepted a null map value and silently dropped the key. The framework rejects
	// it at plan, matching SDKv2's own validateLabels/validateAnnotations, which check keys
	// but never values. So state created by SDKv2 from a config containing nulls cannot be
	// planned until the practitioner removes those keys.
	//
	// That is the intended fix, not a regression: under SDKv2 the dropped key reappeared in
	// every subsequent plan, because Update wrote the config's null into state while Read
	// rebuilt the map from a server response that never had it. Needs a release note.
	t.Run(sourceType, func(t *testing.T) {
		name := "tf-acc-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
		config := testAccRoleConfig_nullMetadata(name, "null")
		sourceConfig := strings.Replace(config, `"kubernetes_role_v1"`, fmt.Sprintf("%q", sourceType), 1)
		config = targetConfig(config)
		var uid string

		testAccRoleWithImports(t, resource.TestCase{
			CheckDestroy: testAccRoleCheckDestroy,
			TerraformVersionChecks: []tfversion.TerraformVersionCheck{
				tfversion.SkipBelow(tfversion.Version1_8_0),
			},
			Steps: []resource.TestStep{
				{
					ExternalProviders: map[string]resource.ExternalProvider{
						"kubernetes": {
							VersionConstraint: roleSDKv2ProviderVersion,
							Source:            "hashicorp/kubernetes",
						},
					},
					Config: sourceConfig,
					Check:  testAccRoleCheckExists(sourceType+".test", &uid),
				},
				{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Config:                   config,
					ExpectError:              regexp.MustCompile("(?s)Invalid Attribute Value.*value must be a string"),
				},
				{
					// A configuration without nulls plans and applies, which is the
					// documented remediation. Also leaves state the harness can destroy.
					ExternalProviders: map[string]resource.ExternalProvider{
						"kubernetes": {
							VersionConstraint: roleSDKv2ProviderVersion,
							Source:            "hashicorp/kubernetes",
						},
					},
					Config: strings.ReplaceAll(sourceConfig, ", optional = null", ""),
				},
			},
		})
	})
}
