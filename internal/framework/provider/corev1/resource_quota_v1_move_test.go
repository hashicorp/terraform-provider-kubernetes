// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/corev1"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// The valid fixture is the shape SDKv2 3.4.0 wrote for a configured spec, captured live.
const resourceQuotaMoveValid = `{"id":"team/quota","timeouts":null,` +
	`"metadata":[{"name":"quota","namespace":"team","uid":"retained","generation":0,"resource_version":"7","labels":{},"annotations":null,"generate_name":""}],` +
	`"spec":[{"hard":{"pods":"4","limits.memory":"2Gi"},"scopes":null,"scope_selector":[]}]}`

func TestResourceQuotaMoveState(t *testing.T) {
	ctx := context.Background()
	r := &corev1.ResourceQuotaV1{}
	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	identitySchema := common.NamespacedIdentitySchema()

	for _, tc := range []struct {
		name, sourceType, provider, raw string
		version                         int64
		wantError, wantSkip, noIdentity bool
		wantDetail                      string
		check                           func(t *testing.T, state corev1.ResourceQuotaV1Model)
	}{
		{name: "valid", raw: resourceQuotaMoveValid, check: func(t *testing.T, s corev1.ResourceQuotaV1Model) {
			if len(s.Spec) != 1 || len(s.Spec[0].Hard.Elements()) != 2 {
				t.Fatalf("hard not carried over: %#v", s.Spec)
			}
			if !s.Spec[0].Scopes.IsNull() {
				t.Error("a null scopes must stay null")
			}
			if len(s.Spec[0].ScopeSelector) != 0 {
				t.Error("an empty scope_selector must stay empty")
			}
			if !s.Metadata[0].Labels.IsNull() && len(s.Metadata[0].Labels.Elements()) != 0 {
				t.Error("labels changed")
			}
			if !s.Timeouts.Object.IsNull() {
				t.Error("absent timeouts must stay null")
			}
		}},
		{name: "emptySpecFromOmittedSpec", raw: strings.Replace(resourceQuotaMoveValid,
			`"spec":[{"hard":{"pods":"4","limits.memory":"2Gi"},"scopes":null,"scope_selector":[]}]`,
			`"spec":[{"hard":{},"scopes":[],"scope_selector":[]}]`, 1), check: func(t *testing.T, s corev1.ResourceQuotaV1Model) {
			if s.Spec[0].Hard.IsNull() || len(s.Spec[0].Hard.Elements()) != 0 {
				t.Error("an empty hard must stay empty, not null")
			}
			if s.Spec[0].Scopes.IsNull() || len(s.Spec[0].Scopes.Elements()) != 0 {
				t.Error("an empty scopes must stay empty, not null")
			}
		}},
		{name: "noSpec", raw: strings.Replace(resourceQuotaMoveValid,
			`"spec":[{"hard":{"pods":"4","limits.memory":"2Gi"},"scopes":null,"scope_selector":[]}]`, `"spec":[]`, 1),
			check: func(t *testing.T, s corev1.ResourceQuotaV1Model) {
				if len(s.Spec) != 0 {
					t.Error("spec must stay absent")
				}
			}},
		{name: "scopesAndSelector", raw: strings.Replace(resourceQuotaMoveValid,
			`"scopes":null,"scope_selector":[]`,
			`"scopes":["BestEffort"],"scope_selector":[{"match_expression":[{"scope_name":"PriorityClass","operator":"In","values":["medium"]},{"scope_name":"BestEffort","operator":"Exists","values":null}]}]`, 1),
			check: func(t *testing.T, s corev1.ResourceQuotaV1Model) {
				if len(s.Spec[0].Scopes.Elements()) != 1 {
					t.Error("scopes lost")
				}
				sel := s.Spec[0].ScopeSelector
				if len(sel) != 1 || len(sel[0].MatchExpression) != 2 {
					t.Fatalf("scope_selector lost: %#v", sel)
				}
				if len(sel[0].MatchExpression[0].Values.Elements()) != 1 || !sel[0].MatchExpression[1].Values.IsNull() {
					t.Error("match_expression values: set stays a set and null stays null")
				}
			}},
		{name: "timeouts", raw: strings.Replace(resourceQuotaMoveValid, `"timeouts":null`, `"timeouts":{"create":"2m","update":""}`, 1),
			check: func(t *testing.T, s corev1.ResourceQuotaV1Model) {
				if s.Timeouts.Object.IsNull() {
					t.Fatal("timeouts lost")
				}
				attrs := s.Timeouts.Object.Attributes()
				if attrs["create"].String() != `"2m"` || !attrs["update"].IsNull() {
					t.Errorf("timeouts: %v", attrs)
				}
			}},
		{name: "generatedName", raw: strings.Replace(resourceQuotaMoveValid, `"generate_name":""`, `"generate_name":"quota-"`, 1)},
		{name: "withoutTargetIdentity", raw: resourceQuotaMoveValid, noIdentity: true},
		{name: "privateRegistry", provider: "registry.example.com/hashicorp/kubernetes", raw: resourceQuotaMoveValid},
		{name: "wrongType", sourceType: "kubernetes_limit_range", raw: resourceQuotaMoveValid, wantSkip: true},
		{name: "wrongProvider", provider: "registry.terraform.io/example/kubernetes", raw: resourceQuotaMoveValid, wantSkip: true},
		{name: "lookalikeProvider", provider: "registry.example.com/nothashicorp/kubernetes", raw: resourceQuotaMoveValid, wantSkip: true},
		{name: "wrongVersion", version: 1, raw: resourceQuotaMoveValid, wantSkip: true},
		{name: "missingJSON", wantError: true},
		{name: "malformedJSON", raw: "{", wantError: true},
		{name: "emptyID", raw: `{"metadata":[{}]}`, wantError: true, wantDetail: "The source state has an empty id."},
		{name: "missingMetadata", raw: `{"id":"team/quota"}`, wantError: true, wantDetail: "Expected exactly 1 metadata element in the source state, got 0."},
		{name: "multipleMetadata", raw: `{"id":"team/quota","metadata":[{},{}]}`, wantError: true, wantDetail: "Expected exactly 1 metadata element in the source state, got 2."},
		{name: "multipleSpec", raw: `{"id":"team/quota","metadata":[{}],"spec":[{},{}]}`, wantError: true, wantDetail: "Expected at most 1 spec element in the source state, got 2."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := fwresource.MoveStateRequest{
				SourceTypeName:        "kubernetes_resource_quota",
				SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes",
				SourceSchemaVersion:   tc.version,
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
			resp := fwresource.MoveStateResponse{
				TargetState: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)},
			}
			if !tc.noIdentity {
				resp.TargetIdentity = &tfsdk.ResourceIdentity{Schema: identitySchema, Raw: tftypes.NewValue(identitySchema.Type().TerraformType(ctx), nil)}
			}

			movers := r.MoveState(ctx)
			if len(movers) != 1 {
				t.Fatalf("expected 1 state mover, got %d", len(movers))
			}
			if movers[0].SourceSchema != nil {
				t.Fatal("the mover must not declare a SourceSchema: the framework would decode against it first")
			}
			movers[0].StateMover(ctx, req, &resp)

			if resp.Diagnostics.HasError() != tc.wantError {
				t.Fatalf("diagnostics: %v", resp.Diagnostics)
			}
			if tc.wantDetail != "" && (len(resp.Diagnostics) != 1 || resp.Diagnostics[0].Detail() != tc.wantDetail) {
				t.Fatalf("expected diagnostic detail %q, got %v", tc.wantDetail, resp.Diagnostics)
			}
			if tc.wantError || tc.wantSkip {
				if tc.wantSkip && len(resp.Diagnostics) != 0 {
					t.Fatalf("a skipped source must add no diagnostics, got %v", resp.Diagnostics)
				}
				if !resp.TargetState.Raw.IsNull() {
					t.Fatal("unexpected target state")
				}
				if resp.TargetIdentity != nil && !resp.TargetIdentity.Raw.IsNull() {
					t.Fatal("unexpected target identity")
				}
				return
			}

			var state corev1.ResourceQuotaV1Model
			if diags := resp.TargetState.Get(ctx, &state); diags.HasError() {
				t.Fatal(diags)
			}
			if state.ID.ValueString() != "team/quota" || state.Metadata[0].Namespace.ValueString() != "team" ||
				state.Metadata[0].Name.ValueString() != "quota" || state.Metadata[0].UID.ValueString() != "retained" ||
				state.Metadata[0].ResourceVersion.ValueString() != "7" {
				t.Fatalf("incorrect state: %#v", state)
			}
			if tc.name == "generatedName" {
				if state.Metadata[0].GenerateName.ValueString() != "quota-" {
					t.Fatal("lost generate_name")
				}
			} else if !state.Metadata[0].GenerateName.IsNull() {
				t.Fatal("unset generate_name must normalize to null")
			}
			if tc.check != nil {
				tc.check(t, state)
			}

			if resp.TargetIdentity != nil {
				var identity common.NamespacedResourceIdentity
				if diags := resp.TargetIdentity.Get(ctx, &identity); diags.HasError() {
					t.Fatal(diags)
				}
				if identity.Kind.ValueString() != "ResourceQuota" || identity.APIVersion.ValueString() != "v1" ||
					identity.Name.ValueString() != "quota" || identity.Namespace.ValueString() != "team" {
					t.Fatalf("identity: %#v", identity)
				}
			}
		})
	}
}

func TestResourceQuotaImportState(t *testing.T) {
	ctx := context.Background()
	r := &corev1.ResourceQuotaV1{}
	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	identitySchema := common.NamespacedIdentitySchema()

	identity := func(t *testing.T, kind, apiVersion, namespace, name string) *tfsdk.ResourceIdentity {
		t.Helper()
		id := &tfsdk.ResourceIdentity{Schema: identitySchema, Raw: tftypes.NewValue(identitySchema.Type().TerraformType(ctx), nil)}
		if diags := id.Set(ctx, common.NamespacedResourceIdentity{
			ResourceIdentity: common.ResourceIdentity{APIVersion: types.StringValue(apiVersion), Kind: types.StringValue(kind), Name: types.StringValue(name)},
			Namespace:        types.StringValue(namespace),
		}); diags.HasError() {
			t.Fatal(diags)
		}
		return id
	}

	for _, tc := range []struct {
		name      string
		id        string
		identity  func(t *testing.T) *tfsdk.ResourceIdentity
		wantID    string
		wantError bool
	}{
		{name: "byID", id: "team/quota", wantID: "team/quota"},
		{name: "byIdentity", identity: func(t *testing.T) *tfsdk.ResourceIdentity {
			return identity(t, "ResourceQuota", "v1", "team", "quota")
		}, wantID: "team/quota"},
		{name: "wrongKind", identity: func(t *testing.T) *tfsdk.ResourceIdentity {
			return identity(t, "LimitRange", "v1", "team", "quota")
		}, wantError: true},
		{name: "wrongAPIVersion", identity: func(t *testing.T) *tfsdk.ResourceIdentity {
			return identity(t, "ResourceQuota", "v2", "team", "quota")
		}, wantError: true},
		{name: "missingNamespace", identity: func(t *testing.T) *tfsdk.ResourceIdentity {
			return identity(t, "ResourceQuota", "v1", "", "quota")
		}, wantError: true},
		{name: "missingName", identity: func(t *testing.T) *tfsdk.ResourceIdentity {
			return identity(t, "ResourceQuota", "v1", "team", "")
		}, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := fwresource.ImportStateRequest{ID: tc.id}
			if tc.identity != nil {
				req.Identity = tc.identity(t)
			} else {
				req.Identity = &tfsdk.ResourceIdentity{Schema: identitySchema, Raw: tftypes.NewValue(identitySchema.Type().TerraformType(ctx), nil)}
			}
			resp := fwresource.ImportStateResponse{
				State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)},
			}
			r.ImportState(ctx, req, &resp)

			if resp.Diagnostics.HasError() != tc.wantError {
				t.Fatalf("diagnostics: %v", resp.Diagnostics)
			}
			if tc.wantError {
				return
			}
			var got string
			if diags := resp.State.GetAttribute(ctx, path.Root("id"), &got); diags.HasError() {
				t.Fatal(diags)
			}
			if got != tc.wantID {
				t.Fatalf("id = %q, want %q", got, tc.wantID)
			}
		})
	}
}

// Acceptance tests: moving state from the deprecated kubernetes_resource_quota alias, which
// stays on SDKv2, to this resource. Distinct from the in-place upgrade tests in the migration
// file: there the framework decodes SDKv2 state itself, here MoveState translates it field by
// field, so each stored shape is a separate branch to cover. The scenarios come from that file,
// so both families describe the same shapes and cannot drift apart.
//
// Cross-type moved needs Terraform 1.8+, and the destination provider must implement
// MoveState or Terraform fails the move outright.

// testAccResourceQuotaMoveSource rewrites a kubernetes_resource_quota_v1 config to the
// deprecated alias for the step that runs under the released SDKv2 provider. The namespace
// block keeps its type.
func testAccResourceQuotaMoveSource(config string) string {
	return strings.ReplaceAll(config, `"kubernetes_resource_quota_v1"`, `"kubernetes_resource_quota"`)
}

// testAccResourceQuotaMoveTarget appends the moved block. Apart from that and the type name
// the two steps' configs are identical, so a non-empty plan can only come from the move.
func testAccResourceQuotaMoveTarget(config string) string {
	return config + `
moved {
  from = kubernetes_resource_quota.test
  to   = kubernetes_resource_quota_v1.test
}
`
}

// testAccResourceQuotaMoveScenario applies the scenario's config as the alias under an SDKv2
// release, moves it, and asserts the outcome the scenario declares. Import is verified before
// and after the move, and the plan after the move settles. CheckDestroy matches both type
// names, since a failure at the move leaves state under the old address.
func testAccResourceQuotaMoveScenario(t *testing.T, version string, s quotaScenario) {
	t.Helper()

	name := testAccResourceQuotaMigrationName()
	config := s.config(name)
	var before, after k8sv1.ResourceQuota
	ignore := quotaConcat(s.importIgnore, quotaInternalIgnore(s.internalMetadata))
	importStep := func(address string, sdkv2 bool) resource.TestStep {
		step := resource.TestStep{
			ResourceName:            address,
			ImportState:             true,
			ImportStateVerify:       true,
			ImportStateVerifyIgnore: ignore,
			ImportStateCheck:        testAccCheckResourceQuotaImportOwnership(s.internalMetadata),
		}
		if sdkv2 {
			step.ExternalProviders = testAccResourceQuotaExternalProvider(version)
		} else {
			step.ProtoV6ProviderFactories = testAccProtoV6ProviderFactories
		}
		return step
	}

	checks := []resource.TestCheckFunc{
		testAccCheckKubernetesResourceQuotaV1Exists(resourceQuotaResourceName, &after),
		testAccCheckResourceQuotaNotRecreated(&before, &after),
	}
	if s.outcome == quotaRespelled {
		checks = append(checks, testAccCheckResourceQuotaUntouched(&before, &after))
	}

	steps := []resource.TestStep{
		{
			ExternalProviders: testAccResourceQuotaExternalProvider(version),
			Config:            testAccResourceQuotaMoveSource(config),
			// Captured by name: the quota is managed as kubernetes_resource_quota here.
			Check: testAccStoreResourceQuota(&before),
			// See quotaScenario.sourceNotIdempotent.
			ExpectNonEmptyPlan: s.sourceNotIdempotent,
		},
		importStep("kubernetes_resource_quota.test", true),
		{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Config:                   testAccResourceQuotaMoveTarget(config),
			ConfigPlanChecks:         quotaUpgradePlanChecks(s),
			Check:                    resource.ComposeAggregateTestCheckFunc(checks...),
		},
		importStep(resourceQuotaResourceName, false),
		{
			// With the moved block gone the address is simply the new type, and stays settled.
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Config:                   config,
			ConfigPlanChecks:         resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
		},
	}
	if s.updated != nil {
		steps = append(steps,
			resource.TestStep{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   s.updated(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceQuotaResourceName, plancheck.ResourceActionUpdate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaV1Exists(resourceQuotaResourceName, &after),
					testAccCheckResourceQuotaNotRecreated(&before, &after),
				),
			},
			importStep(resourceQuotaResourceName, false),
		)
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:               func() { testAccPreCheck(t) },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_8_0)},
		CheckDestroy:           testAccCheckKubernetesResourceQuotaV1Destroy,
		Steps:                  steps,
	})
}

func TestAccKubernetesResourceQuotaV1_MoveStateFromUnversioned_scenarios(t *testing.T) {
	for _, s := range quotaScenarios() {
		t.Run(s.name, func(t *testing.T) { testAccResourceQuotaMoveScenario(t, resourceQuotaSDKv2Version, s) })
	}
}

// Pre-identity: req.SourceIdentity is nil, so MoveState builds the identity rather than
// carrying one across.
func TestAccKubernetesResourceQuotaV1_MoveStateFromUnversionedPreIdentity_scenarios(t *testing.T) {
	for _, name := range quotaPreIdentityScenarios {
		s := quotaScenarioByName(t, name)
		t.Run(s.name, func(t *testing.T) { testAccResourceQuotaMoveScenario(t, sdkv2PreIdentityProviderVersion, s) })
	}
}

// A moved resource is as importable by identity as a created one, and carries the identity
// MoveState built.
func testAccResourceQuotaMoveIdentity(t *testing.T, version string) {
	t.Helper()
	name := testAccResourceQuotaMigrationName()
	config := testAccKubernetesResourceQuotaV1Config_withScopes(name)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:               func() { testAccPreCheck(t) },
		CheckDestroy:           testAccCheckKubernetesResourceQuotaV1Destroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)},
		Steps: []resource.TestStep{
			{ExternalProviders: testAccResourceQuotaExternalProvider(version), Config: testAccResourceQuotaMoveSource(config)},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccResourceQuotaMoveTarget(config),
				ConfigPlanChecks:         resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(resourceQuotaResourceName, map[string]knownvalue.Check{
						"namespace":   knownvalue.StringExact(name),
						"name":        knownvalue.StringExact(name),
						"api_version": knownvalue.StringExact("v1"),
						"kind":        knownvalue.StringExact("ResourceQuota"),
					}),
				},
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				ResourceName:             resourceQuotaResourceName,
				ImportState:              true,
				ImportStateKind:          resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func TestAccKubernetesResourceQuotaV1_MoveStateFromUnversioned_identity(t *testing.T) {
	testAccResourceQuotaMoveIdentity(t, resourceQuotaSDKv2Version)
}

func TestAccKubernetesResourceQuotaV1_MoveStateFromUnversionedPreIdentity_identity(t *testing.T) {
	testAccResourceQuotaMoveIdentity(t, sdkv2PreIdentityProviderVersion)
}

// The move must preserve metadata written by another controller, and so must the first update
// after it. Same hazard as the in-place upgrade twin, reached by the other route: state arrives
// through MoveState rather than UpgradeResourceState, but it is equally free of the ignored
// key, so the update that follows is where an unguarded diff would replace the live map.
func TestAccKubernetesResourceQuotaV1_MoveStateFromUnversioned_ignoredMetadataSurvivesUpdate(t *testing.T) {
	for _, tc := range []struct{ ignored, managed string }{
		{ignored: "annotations", managed: "labels"},
		{ignored: "labels", managed: "annotations"},
	} {
		t.Run(tc.ignored, func(t *testing.T) {
			var before, after k8sv1.ResourceQuota
			name := testAccResourceQuotaMigrationName()
			beforeConfig := testAccKubernetesResourceQuotaV1Config_managedMetadata(name, tc.managed, "before")
			afterConfig := testAccKubernetesResourceQuotaV1Config_managedMetadata(name, tc.managed, "after")

			resource.ParallelTest(t, resource.TestCase{
				PreCheck:               func() { testAccPreCheck(t) },
				TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_8_0)},
				CheckDestroy:           testAccCheckKubernetesResourceQuotaV1Destroy,
				Steps: []resource.TestStep{
					{
						ExternalProviders: testAccResourceQuotaExternalProvider(resourceQuotaSDKv2Version),
						Config:            testAccResourceQuotaMoveSource(beforeConfig),
						Check:             testAccStoreResourceQuota(&before),
					},
					{
						// Another controller writes to the map Terraform does not manage,
						// before the move.
						PreConfig:                func() { testAccInjectIgnoredQuotaMetadata(t, name, tc.ignored) },
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Config:                   testAccResourceQuotaMoveTarget(beforeConfig),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							// The move itself must not recreate the quota.
							testAccCheckKubernetesResourceQuotaV1Exists(resourceQuotaResourceName, &after),
							testAccCheckResourceQuotaNotRecreated(&before, &after),
							testAccCheckIgnoredQuotaMetadata(name, tc.ignored),
						),
					},
					{
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Config:                   afterConfig,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceQuotaResourceName, plancheck.ResourceActionUpdate)},
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							testAccCheckKubernetesResourceQuotaV1Exists(resourceQuotaResourceName, &after),
							testAccCheckResourceQuotaNotRecreated(&before, &after),
							resource.TestCheckResourceAttr(resourceQuotaResourceName, "metadata.0."+tc.managed+".managed", "after"),
							testAccCheckIgnoredQuotaMetadata(name, tc.ignored),
						),
					},
				},
			})
		})
	}
}

// Null map values: see testAccResourceQuotaNullInput for why rejecting them is the intended fix.
func TestAccKubernetesResourceQuotaV1_MoveStateFromUnversioned_nullInput(t *testing.T) {
	testAccResourceQuotaNullInput(t, "kubernetes_resource_quota", testAccResourceQuotaMoveTarget)
}

// testAccStoreResourceQuota records the live quota while it is still managed as the deprecated
// alias, where the usual Exists check on the versioned address cannot run.
func testAccStoreResourceQuota(out *k8sv1.ResourceQuota) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["kubernetes_resource_quota.test"]
		if !ok {
			return fmt.Errorf("kubernetes_resource_quota.test not in state")
		}
		conn, err := mainClientset()
		if err != nil {
			return err
		}
		namespace, name, _ := strings.Cut(rs.Primary.ID, "/")
		quota, err := conn.CoreV1().ResourceQuotas(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		*out = *quota
		return nil
	}
}
