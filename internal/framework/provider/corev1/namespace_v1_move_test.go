// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0
package corev1_test

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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/corev1"
	k8sv1 "k8s.io/api/core/v1"
)

// MoveState tests: moving state from the deprecated kubernetes_namespace alias, which stays
// on SDKv2, to this framework resource.
//
// Distinct from the migration tests next door, which upgrade state in place under the same
// type name. There the framework decodes SDKv2 state itself; here MoveState translates it
// field by field, so each stored shape is a separate branch to cover.
//
// Configs come from namespace_v1_migration_test.go so both families describe the same
// shapes and cannot drift apart.

// testAccNamespaceMoveStateSource rewrites a kubernetes_namespace_v1 config to the
// deprecated alias, for the step that runs under the released SDKv2 provider.
func testAccNamespaceMoveStateSource(config string) string {
	return strings.ReplaceAll(config, `"kubernetes_namespace_v1"`, `"kubernetes_namespace"`)
}

// testAccNamespaceMoveStateTarget appends the moved block. Apart from that and the type
// name, the two steps' configs are identical, so a non-empty plan can only come from the
// move itself.
func testAccNamespaceMoveStateTarget(config string) string {
	return config + `
moved {
  from = kubernetes_namespace.test
  to   = kubernetes_namespace_v1.test
}
`
}

// testAccNamespaceMoveState applies a config as kubernetes_namespace under the given SDKv2
// release, then re-plans it as kubernetes_namespace_v1 under the local build with a moved
// block, and asserts the move plans nothing.
//
// Without MoveState these fail outright: Terraform moves state across resource types only
// when the destination provider implements it. Cross-type moved needs Terraform 1.8+.
//
// CheckDestroy matches both type names, since a failure at step 2 leaves state under the
// old address and would otherwise leak a namespace silently.
func testAccNamespaceMoveState(t *testing.T, sdkv2Version, config string) {
	t.Helper()
	testAccNamespaceMoveStateWithPlanCheck(t, sdkv2Version, config, plancheck.ExpectEmptyPlan())
}

// testAccNamespaceMoveStateExpectingUpdate is testAccNamespaceMoveState for the shapes that
// cannot move cleanly: SDKv2 stored null for a config that says `{}`, so the first plan
// reconciles the two. An update is an acceptable one-off; a replacement would destroy the
// namespace, which is why the action is asserted rather than just a non-empty plan.
func testAccNamespaceMoveStateExpectingUpdate(t *testing.T, sdkv2Version, config string) {
	t.Helper()
	testAccNamespaceMoveStateWithPlanCheck(t, sdkv2Version, config,
		plancheck.ExpectResourceAction("kubernetes_namespace_v1.test", plancheck.ResourceActionUpdate))
}

func testAccNamespaceMoveStateWithPlanCheck(t *testing.T, sdkv2Version, config string, check plancheck.PlanCheck) {
	t.Helper()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		CheckDestroy: testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						VersionConstraint: sdkv2Version,
						Source:            "hashicorp/kubernetes",
					},
				},
				Config: testAccNamespaceMoveStateSource(config),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccNamespaceMoveStateTarget(config),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{check},
				},
			},
		},
	})
}

// The baseline shape: a name and nothing else, so both metadata maps are stored null.
func TestAccNamespace_MoveStateFromUnversioned(t *testing.T) {
	nsName := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMoveState(t, sdkv2ProviderVersion,
		testAccKubernetesNamespaceV1Config_basic(nsName))
}

// generate_name: metadata.name is server-assigned, and is Optional+Computed with both
// UseStateForUnknown and RequiresReplace. If the move dropped the name or left it unknown,
// the plan would be a replacement rather than empty, destroying the namespace. It is also
// the only shape where SDKv2 stores a real generate_name instead of the "" it writes when
// the field was never set.
func TestAccNamespace_MoveStateFromUnversioned_generateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMoveState(t, sdkv2ProviderVersion,
		testAccKubernetesNamespaceV1Config_generatedName(prefix))
}

// timeouts is the one attribute whose type cannot be eyeballed — SDKv2 injects it as a
// single-nested block of strings (helper/schema/core_schema.go:328) — and MoveState rebuilds
// it by hand. Dropping it leaves state null against a config that sets it, so the plan is
// not empty and this fails. The other tests declare no timeouts block.
func TestAccNamespace_MoveStateFromUnversioned_deleteTimeout(t *testing.T) {
	nsName := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMoveState(t, sdkv2ProviderVersion,
		testAccKubernetesNamespaceV1Config_timeouts(nsName))
}

// annotations populated, labels absent — one map stored null and one not. The API server
// also adds an internal label here, which Read has to filter back out after the move.
func TestAccNamespace_MoveStateFromUnversioned_annotations(t *testing.T) {
	nsName := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMoveState(t, sdkv2ProviderVersion,
		testAccKubernetesNamespaceV1Config_Annotations(nsName))
}

// The mirror: labels populated, annotations absent.
func TestAccNamespace_MoveStateFromUnversioned_labels(t *testing.T) {
	nsName := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMoveState(t, sdkv2ProviderVersion,
		testAccKubernetesNamespaceV1Config_Labels(nsName))
}

// A key the filtering rules would normally strip, declared in config so it must survive the
// move and the Read that follows it.
func TestAccNamespace_MoveStateFromUnversioned_declaredInternalLabel(t *testing.T) {
	nsName := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMoveState(t, sdkv2ProviderVersion,
		testAccKubernetesNamespaceV1Config_declaredInternalLabel(nsName))
}

// Every optional field declared but empty. SDKv2 stores null for `annotations = {}`, which
// MoveState carries across as null, so the first plan updates to the `{}` the config asks
// for. The same one-off update the in-place upgrade produces for this shape.
func TestAccNamespace_MoveStateFromUnversioned_emptyValuesName(t *testing.T) {
	nsName := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMoveStateExpectingUpdate(t, sdkv2ProviderVersion,
		testAccKubernetesNamespaceV1Config_emptyValuesName(nsName))
}

// The same shape named by generate_name rather than name.
func TestAccNamespace_MoveStateFromUnversioned_emptyValuesGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMoveStateExpectingUpdate(t, sdkv2ProviderVersion,
		testAccKubernetesNamespaceV1Config_emptyValuesGeneratedName(prefix))
}

// Every attribute populated at once: both maps with several keys, wait_for_default_service_account
// true, and a delete timeout. Each is translated separately, so this is the shape that
// catches a field dropped from the translation.
func TestAccNamespace_MoveStateFromUnversioned_completeName(t *testing.T) {
	nsName := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMoveState(t, sdkv2ProviderVersion,
		testAccKubernetesNamespaceV1Config_completeName(nsName))
}

func TestAccNamespace_MoveStateFromUnversioned_completeGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMoveState(t, sdkv2ProviderVersion,
		testAccKubernetesNamespaceV1Config_completeGeneratedName(prefix))
}

// The pre-identity tests below are the only move coverage of state written before resource
// identity existed (2.38.0). There req.SourceIdentity is nil, so the identity MoveState
// writes is built rather than carried across. Four shapes: the minimal one and the fully
// populated one, each with name and with generate_name, since a server-assigned name is
// what the identity carries.
func TestAccNamespace_MoveStateFromUnversionedPreIdentity_basicName(t *testing.T) {
	nsName := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMoveState(t, sdkv2PreIdentityProviderVersion,
		testAccKubernetesNamespaceV1Config_basic(nsName))
}

func TestAccNamespace_MoveStateFromUnversionedPreIdentity_basicGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMoveState(t, sdkv2PreIdentityProviderVersion,
		testAccKubernetesNamespaceV1Config_generatedName(prefix))
}

func TestAccNamespace_MoveStateFromUnversionedPreIdentity_completeName(t *testing.T) {
	nsName := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMoveState(t, sdkv2PreIdentityProviderVersion,
		testAccKubernetesNamespaceV1Config_completeName(nsName))
}

func TestAccNamespace_MoveStateFromUnversionedPreIdentity_completeGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccNamespaceMoveState(t, sdkv2PreIdentityProviderVersion,
		testAccKubernetesNamespaceV1Config_completeGeneratedName(prefix))
}

func TestNamespaceMoveState(t *testing.T) {
	ctx := context.Background()
	namespace := &corev1.NamespaceV1{}
	var schemaResponse frameworkresource.SchemaResponse
	namespace.Schema(ctx, frameworkresource.SchemaRequest{}, &schemaResponse)
	if schemaResponse.Diagnostics.HasError() {
		t.Fatal(schemaResponse.Diagnostics)
	}
	var identitySchemaResponse frameworkresource.IdentitySchemaResponse
	namespace.IdentitySchema(ctx, frameworkresource.IdentitySchemaRequest{}, &identitySchemaResponse)
	if identitySchemaResponse.Diagnostics.HasError() {
		t.Fatal(identitySchemaResponse.Diagnostics)
	}
	mover := namespace.MoveState(ctx)[0].StateMover

	for _, testCase := range []struct {
		name             string
		providerAddress  string
		annotations      map[string]string
		labels           map[string]string
		generateName     string
		wantGenerateName any
		wait             bool
		timeouts         any
		wantTimeouts     any
		noTargetIdentity bool
	}{
		{
			name:            "complete",
			providerAddress: "registry.terraform.io/hashicorp/kubernetes",
			annotations:     map[string]string{"example.com/note": "retained"},
			labels:          map[string]string{"app": "demo"},
			wait:            true,
			timeouts:        map[string]any{"delete": "10m"}, wantTimeouts: map[string]any{"delete": "10m"},
		},
		{
			name:            "generated name and null maps",
			providerAddress: "mirror.example.com/hashicorp/kubernetes",
			generateName:    "generated-", wantGenerateName: "generated-",
		},
		{
			name:            "empty maps and null timeout",
			providerAddress: "registry.terraform.io/hashicorp/kubernetes",
			annotations:     map[string]string{}, labels: map[string]string{},
			timeouts: map[string]any{"delete": nil}, wantTimeouts: map[string]any{"delete": nil},
		},
		{
			name:            "empty timeout without target identity",
			providerAddress: "registry.terraform.io/hashicorp/kubernetes",
			timeouts:        map[string]any{"delete": ""}, wantTimeouts: map[string]any{"delete": nil},
			noTargetIdentity: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			source := map[string]any{
				"id":                               "namespace",
				"wait_for_default_service_account": testCase.wait,
				"timeouts":                         testCase.timeouts,
				"metadata": []map[string]any{{
					"name": "namespace", "generate_name": testCase.generateName,
					"annotations": testCase.annotations, "labels": testCase.labels,
					"uid": "original-uid", "generation": 7, "resource_version": "1234",
				}},
			}
			sourceJSON, err := json.Marshal(source)
			if err != nil {
				t.Fatal(err)
			}
			response := frameworkresource.MoveStateResponse{
				TargetState: tfsdk.State{Schema: schemaResponse.Schema},
			}
			if !testCase.noTargetIdentity {
				response.TargetIdentity = &tfsdk.ResourceIdentity{Schema: identitySchemaResponse.IdentitySchema}
			}
			mover(ctx, frameworkresource.MoveStateRequest{
				SourceTypeName: "kubernetes_namespace", SourceSchemaVersion: 0,
				SourceProviderAddress: testCase.providerAddress,
				SourceRawState:        &tfprotov6.RawState{JSON: sourceJSON},
			}, &response)
			if response.Diagnostics.HasError() {
				t.Fatal(response.Diagnostics)
			}

			expectedJSON, err := json.Marshal(map[string]any{
				"id":                               "namespace",
				"wait_for_default_service_account": testCase.wait,
				"timeouts":                         testCase.wantTimeouts,
				"metadata": []map[string]any{{
					"name": "namespace", "generate_name": testCase.wantGenerateName,
					"annotations": testCase.annotations, "labels": testCase.labels,
					"uid": "original-uid", "generation": 7, "resource_version": "1234",
				}},
			})
			if err != nil {
				t.Fatal(err)
			}
			expectedRaw := &tfprotov6.RawState{JSON: expectedJSON}
			expected, err := expectedRaw.Unmarshal(schemaResponse.Schema.Type().TerraformType(ctx))
			if err != nil {
				t.Fatal(err)
			}
			if !response.TargetState.Raw.Equal(expected) {
				t.Errorf("moved state mismatch\nwant: %s\ngot:  %s", expected, response.TargetState.Raw)
			}
			if response.TargetIdentity != nil {
				var identity common.ResourceIdentity
				if diags := response.TargetIdentity.Get(ctx, &identity); diags.HasError() {
					t.Fatal(diags)
				}
				if identity.Name != types.StringValue("namespace") || identity.APIVersion != types.StringValue("v1") || identity.Kind != types.StringValue("Namespace") {
					t.Errorf("unexpected moved identity: %#v", identity)
				}
			}
		})
	}

	for _, testCase := range []struct {
		name            string
		sourceType      string
		sourceVersion   int64
		providerAddress string
		rawState        *tfprotov6.RawState
		wantError       bool
	}{
		{name: "wrong resource", sourceType: "kubernetes_pod", providerAddress: "registry.terraform.io/hashicorp/kubernetes"},
		{name: "wrong schema version", sourceType: "kubernetes_namespace", sourceVersion: 1, providerAddress: "registry.terraform.io/hashicorp/kubernetes"},
		{name: "wrong provider", sourceType: "kubernetes_namespace", providerAddress: "registry.terraform.io/other/kubernetes"},
		{name: "missing raw state", sourceType: "kubernetes_namespace", providerAddress: "registry.terraform.io/hashicorp/kubernetes", wantError: true},
		{name: "malformed raw state", sourceType: "kubernetes_namespace", providerAddress: "registry.terraform.io/hashicorp/kubernetes", rawState: &tfprotov6.RawState{JSON: []byte(`{`)}, wantError: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			response := frameworkresource.MoveStateResponse{
				TargetState:    tfsdk.State{Schema: schemaResponse.Schema},
				TargetIdentity: &tfsdk.ResourceIdentity{Schema: identitySchemaResponse.IdentitySchema},
			}
			mover(ctx, frameworkresource.MoveStateRequest{
				SourceTypeName: testCase.sourceType, SourceSchemaVersion: testCase.sourceVersion,
				SourceProviderAddress: testCase.providerAddress, SourceRawState: testCase.rawState,
			}, &response)
			if response.Diagnostics.HasError() != testCase.wantError {
				t.Errorf("unexpected diagnostics: %v", response.Diagnostics)
			}
			if !response.TargetState.Raw.IsNull() || !response.TargetIdentity.Raw.IsNull() {
				t.Fatal("unsupported or invalid source must not produce state or identity")
			}
		})
	}
}

// The move must preserve metadata written by another controller, and so must the first
// update after it.
//
// Same hazard as the in-place upgrade twin in namespace_v1_migration_test.go, reached by the
// other route: state arrives through MoveState rather than UpgradeResourceState, but it is
// equally free of the ignored key, so the update that follows is where an unguarded diff
// would replace the live map. Both directions are covered for the same reason as there.
func TestAccNamespace_MoveStateFromUnversioned_ignoredMetadataSurvivesUpdate(t *testing.T) {
	for _, tc := range []struct{ ignored, managed string }{
		{ignored: "annotations", managed: "labels"},
		{ignored: "labels", managed: "annotations"},
	} {
		t.Run(tc.ignored, func(t *testing.T) {
			var before, after k8sv1.Namespace
			nsName := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
			beforeConfig := testAccKubernetesNamespaceV1Config_managedMetadata(nsName, tc.managed, "before")
			afterConfig := testAccKubernetesNamespaceV1Config_managedMetadata(nsName, tc.managed, "after")

			resource.ParallelTest(t, resource.TestCase{
				PreCheck: func() { testAccPreCheck(t) },
				TerraformVersionChecks: []tfversion.TerraformVersionCheck{
					tfversion.SkipBelow(tfversion.Version1_8_0),
				},
				CheckDestroy: testAccCheckKubernetesNamespaceV1Destroy,
				Steps: []resource.TestStep{
					{
						ExternalProviders: map[string]resource.ExternalProvider{
							"kubernetes": {VersionConstraint: sdkv2ProviderVersion, Source: "hashicorp/kubernetes"},
						},
						Config: testAccNamespaceMoveStateSource(beforeConfig),
						// Captured by name: at this point the namespace is managed as
						// kubernetes_namespace, so it has no state at the address under test.
						Check: testAccStoreNamespace(nsName, &before),
					},
					{
						// Another controller writes to the map Terraform does not manage,
						// before the move.
						PreConfig:                func() { testAccInjectIgnoredNamespaceMetadata(t, nsName, tc.ignored, ignoredMetadataKey) },
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Config:                   testAccNamespaceMoveStateTarget(beforeConfig),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							// The move itself must not recreate the namespace.
							testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &after),
							testAccCheckNamespaceNotRecreated(&before, &after),
							testAccCheckIgnoredNamespaceMetadata(nsName, tc.ignored, ignoredMetadataKey),
						),
					},
					{
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Config:                   afterConfig,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(namespaceResourceName, plancheck.ResourceActionUpdate)},
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							testAccCheckKubernetesNamespaceV1Exists(namespaceResourceName, &after),
							testAccCheckNamespaceNotRecreated(&before, &after),
							resource.TestCheckResourceAttr(namespaceResourceName, "metadata.0."+tc.managed+".managed", "after"),
							testAccCheckIgnoredNamespaceMetadata(nsName, tc.ignored, ignoredMetadataKey),
						),
					},
				},
			})
		})
	}
}
