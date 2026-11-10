// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/corev1"
)

// MoveState tests: moving state from the deprecated kubernetes_config_map alias, which
// stays on SDKv2, to kubernetes_config_map_v1. Cross-type moved blocks need Terraform 1.8+.
//
// Configs come from config_map_v1_resource_test.go so both families describe the same shapes.

// testAccConfigMapMoveStateSource rewrites a kubernetes_config_map_v1 config to the
// deprecated alias for the step that runs under the released SDKv2 provider.
func testAccConfigMapMoveStateSource(config string) string {
	return strings.ReplaceAll(config, `"kubernetes_config_map_v1"`, `"kubernetes_config_map"`)
}

// testAccConfigMapMoveStateTarget appends the moved block. Apart from that and the type
// name the two configs are identical, so a non-empty plan can only come from the move.
func testAccConfigMapMoveStateTarget(config string) string {
	return config + `
moved {
  from = kubernetes_config_map.test
  to   = kubernetes_config_map_v1.test
}
`
}

func testAccConfigMapMoveState(t *testing.T, sdkv2Version, config string) {
	t.Helper()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		CheckDestroy: testAccCheckKubernetesConfigMapV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {VersionConstraint: sdkv2Version, Source: "hashicorp/kubernetes"},
				},
				Config: testAccConfigMapMoveStateSource(config),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccConfigMapMoveStateTarget(config),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestAccConfigMap_MoveStateFromUnversioned(t *testing.T) {
	testAccConfigMapMoveState(t, sdkv2ProviderVersion, testAccConfigMapV1Config_basic(testAccConfigMapMigrationName()))
}

func TestAccConfigMap_MoveStateFromUnversioned_noData(t *testing.T) {
	testAccConfigMapMoveState(t, sdkv2ProviderVersion, testAccConfigMapV1Config_noData(testAccConfigMapMigrationName()))
}

// A server-assigned name must survive the move; if it were dropped the plan would replace.
func TestAccConfigMap_MoveStateFromUnversioned_generateName(t *testing.T) {
	testAccConfigMapMoveState(t, sdkv2ProviderVersion,
		testAccConfigMapV1Config_generatedName("tf-migration-test-gen-"))
}

func TestAccConfigMap_MoveStateFromUnversioned_binaryData(t *testing.T) {
	testAccConfigMapMoveState(t, sdkv2ProviderVersion,
		testAccConfigMapV1Config_binaryData2("tf-migration-test-gen-", testAccConfigMapFixtureDir(t)))
}

func TestAccConfigMap_MoveStateFromUnversioned_immutable(t *testing.T) {
	testAccConfigMapMoveState(t, sdkv2ProviderVersion,
		testAccConfigMapV1Config_immutable(testAccConfigMapMigrationName(), true, "second"))
}

func TestAccConfigMap_MoveStateFromUnversionedPreIdentity_basic(t *testing.T) {
	testAccConfigMapMoveState(t, sdkv2PreIdentityProviderVersion,
		testAccConfigMapV1Config_basic(testAccConfigMapMigrationName()))
}

func TestConfigMapMoveState(t *testing.T) {
	ctx := context.Background()
	configMap := &corev1.ConfigMapV1{}
	var schemaResponse frameworkresource.SchemaResponse
	configMap.Schema(ctx, frameworkresource.SchemaRequest{}, &schemaResponse)
	if schemaResponse.Diagnostics.HasError() {
		t.Fatal(schemaResponse.Diagnostics)
	}
	var identitySchemaResponse frameworkresource.IdentitySchemaResponse
	configMap.IdentitySchema(ctx, frameworkresource.IdentitySchemaRequest{}, &identitySchemaResponse)
	if identitySchemaResponse.Diagnostics.HasError() {
		t.Fatal(identitySchemaResponse.Diagnostics)
	}
	mover := configMap.MoveState(ctx)[0].StateMover

	metadata := func(generateName any, annotations, labels any) []map[string]any {
		return []map[string]any{{
			"name": "cfg", "namespace": "team", "generate_name": generateName,
			"annotations": annotations, "labels": labels,
			"uid": "original-uid", "generation": 7, "resource_version": "1234",
		}}
	}

	for _, testCase := range []struct {
		name             string
		providerAddress  string
		generateName     string
		wantGenerateName any
		annotations      any
		labels           any
		data             any
		binaryData       any
		immutable        bool
		noTargetIdentity bool
	}{
		{
			name:            "complete",
			providerAddress: "registry.terraform.io/hashicorp/kubernetes",
			annotations:     map[string]string{"example.com/note": "retained"},
			labels:          map[string]string{"app": "demo"},
			data:            map[string]string{"one": "first"},
			binaryData:      map[string]string{"bin": "UmF3"},
			immutable:       true,
		},
		{
			name:            "generated name and null maps",
			providerAddress: "mirror.example.com/hashicorp/kubernetes",
			generateName:    "generated-", wantGenerateName: "generated-",
		},
		{
			name:            "empty maps stay empty",
			providerAddress: "registry.terraform.io/hashicorp/kubernetes",
			annotations:     map[string]string{}, labels: map[string]string{},
			data: map[string]string{}, binaryData: map[string]string{},
		},
		{
			name:             "without target identity",
			providerAddress:  "registry.terraform.io/hashicorp/kubernetes",
			noTargetIdentity: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			source := map[string]any{
				"id": "team/cfg", "immutable": testCase.immutable,
				"data": testCase.data, "binary_data": testCase.binaryData,
				"metadata": metadata(testCase.generateName, testCase.annotations, testCase.labels),
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
				SourceTypeName: "kubernetes_config_map", SourceSchemaVersion: 0,
				SourceProviderAddress: testCase.providerAddress,
				SourceRawState:        &tfprotov6.RawState{JSON: sourceJSON},
			}, &response)
			if response.Diagnostics.HasError() {
				t.Fatal(response.Diagnostics)
			}

			// "" becomes null; every other value, including {} and null maps, is kept.
			expectedJSON, err := json.Marshal(map[string]any{
				"id": "team/cfg", "immutable": testCase.immutable,
				"data": testCase.data, "binary_data": testCase.binaryData,
				"metadata": metadata(testCase.wantGenerateName, testCase.annotations, testCase.labels),
			})
			if err != nil {
				t.Fatal(err)
			}
			expected, err := (&tfprotov6.RawState{JSON: expectedJSON}).Unmarshal(schemaResponse.Schema.Type().TerraformType(ctx))
			if err != nil {
				t.Fatal(err)
			}
			if !response.TargetState.Raw.Equal(expected) {
				t.Errorf("moved state mismatch\nwant: %s\ngot:  %s", expected, response.TargetState.Raw)
			}
			if response.TargetIdentity != nil {
				var identity common.NamespacedResourceIdentity
				if diags := response.TargetIdentity.Get(ctx, &identity); diags.HasError() {
					t.Fatal(diags)
				}
				if identity.Name != types.StringValue("cfg") || identity.Namespace != types.StringValue("team") ||
					identity.APIVersion != types.StringValue("v1") || identity.Kind != types.StringValue("ConfigMap") {
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
		{name: "wrong schema version", sourceType: "kubernetes_config_map", sourceVersion: 1, providerAddress: "registry.terraform.io/hashicorp/kubernetes"},
		{name: "wrong provider", sourceType: "kubernetes_config_map", providerAddress: "registry.terraform.io/other/kubernetes"},
		{name: "missing raw state", sourceType: "kubernetes_config_map", providerAddress: "registry.terraform.io/hashicorp/kubernetes", wantError: true},
		{name: "malformed raw state", sourceType: "kubernetes_config_map", providerAddress: "registry.terraform.io/hashicorp/kubernetes", rawState: &tfprotov6.RawState{JSON: []byte(`{`)}, wantError: true},
		{name: "empty id", sourceType: "kubernetes_config_map", providerAddress: "registry.terraform.io/hashicorp/kubernetes", rawState: &tfprotov6.RawState{JSON: []byte(`{"id":"","metadata":[{}]}`)}, wantError: true},
		{name: "no metadata", sourceType: "kubernetes_config_map", providerAddress: "registry.terraform.io/hashicorp/kubernetes", rawState: &tfprotov6.RawState{JSON: []byte(`{"id":"a/b","metadata":[]}`)}, wantError: true},
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
