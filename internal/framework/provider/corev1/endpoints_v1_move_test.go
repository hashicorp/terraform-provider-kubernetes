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
)

func testAccEndpointsMoveStateSource(config string) string {
	return strings.ReplaceAll(config, `"kubernetes_endpoints_v1"`, `"kubernetes_endpoints"`)
}

func testAccEndpointsMoveStateTarget(config string) string {
	return config + `
moved {
  from = kubernetes_endpoints.test
  to   = kubernetes_endpoints_v1.test
}
`
}

func testAccEndpointsMoveState(t *testing.T, sdkv2Version, config string) {
	t.Helper()
	testAccEndpointsMoveStateWithPlanCheck(t, sdkv2Version, config, plancheck.ExpectEmptyPlan())
}

func testAccEndpointsMoveStateWithPlanCheck(t *testing.T, sdkv2Version, config string, check plancheck.PlanCheck) {
	t.Helper()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		CheckDestroy: testAccCheckKubernetesEndpointV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						VersionConstraint: sdkv2Version,
						Source:            "hashicorp/kubernetes",
					},
				},
				Config: testAccEndpointsMoveStateSource(config),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccEndpointsMoveStateTarget(config),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{check},
				},
			},
		},
	})
}

func TestAccEndpoints_MoveStateFromUnversioned(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccEndpointsMoveState(t, sdkv2ProviderVersion, testAccKubernetesEndpointsV1_basic(name))
}

func TestAccEndpoints_MoveStateFromUnversioned_generatedName(t *testing.T) {
	prefix := fmt.Sprintf("tf-migration-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccEndpointsMoveState(t, sdkv2ProviderVersion, testAccKubernetesEndpointsV1_generatedName(prefix))
}

func TestAccEndpoints_MoveStateFromUnversioned_modified(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccEndpointsMoveState(t, sdkv2ProviderVersion, testAccKubernetesEndpointsV1_modified(name))
}

func TestAccEndpoints_MoveStateFromUnversionedPreIdentity_basic(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccEndpointsMoveState(t, sdkv2PreIdentityProviderVersion, testAccKubernetesEndpointsV1_basic(name))
}

func TestAccEndpoints_MoveStateFromUnversionedPreIdentity_modified(t *testing.T) {
	name := fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccEndpointsMoveState(t, sdkv2PreIdentityProviderVersion, testAccKubernetesEndpointsV1_modified(name))
}

func TestEndpointsMoveState(t *testing.T) {
	ctx := context.Background()
	endpoints := &corev1.EndpointsV1{}
	var schemaResponse frameworkresource.SchemaResponse
	endpoints.Schema(ctx, frameworkresource.SchemaRequest{}, &schemaResponse)
	if schemaResponse.Diagnostics.HasError() {
		t.Fatal(schemaResponse.Diagnostics)
	}
	var identitySchemaResponse frameworkresource.IdentitySchemaResponse
	endpoints.IdentitySchema(ctx, frameworkresource.IdentitySchemaRequest{}, &identitySchemaResponse)
	if identitySchemaResponse.Diagnostics.HasError() {
		t.Fatal(identitySchemaResponse.Diagnostics)
	}
	mover := endpoints.MoveState(ctx)[0].StateMover

	for _, testCase := range []struct {
		name             string
		providerAddress  string
		annotations      map[string]string
		labels           map[string]string
		generateName     string
		wantGenerateName any
		subset           []map[string]any
		wantSubset       []map[string]any
		noTargetIdentity bool
	}{
		{
			name:            "complete",
			providerAddress: "registry.terraform.io/hashicorp/kubernetes",
			annotations:     map[string]string{"example.com/note": "retained"},
			labels:          map[string]string{"app": "demo"},
			subset: []map[string]any{
				{
					"address": []map[string]any{
						{"ip": "10.0.0.1", "hostname": "host1", "node_name": "node1"},
					},
					"not_ready_address": []map[string]any{
						{"ip": "10.0.0.2", "hostname": "host2", "node_name": "node2"},
					},
					"port": []map[string]any{
						{"name": "http", "port": int64(80), "protocol": "TCP"},
					},
				},
			},
			wantSubset: []map[string]any{
				{
					"address": []map[string]any{
						{"ip": "10.0.0.1", "hostname": "host1", "node_name": "node1"},
					},
					"not_ready_address": []map[string]any{
						{"ip": "10.0.0.2", "hostname": "host2", "node_name": "node2"},
					},
					"port": []map[string]any{
						{"name": "http", "port": int64(80), "protocol": "TCP"},
					},
				},
			},
		},
		{
			name:            "generated name and null subsets",
			providerAddress: "mirror.example.com/hashicorp/kubernetes",
			generateName:    "generated-", wantGenerateName: "generated-",
		},
		{
			name:            "empty maps and empty strings in subset fields become null",
			providerAddress: "registry.terraform.io/hashicorp/kubernetes",
			annotations:     map[string]string{}, labels: map[string]string{},
			subset: []map[string]any{
				{
					"address": []map[string]any{
						{"ip": "10.0.0.1", "hostname": "", "node_name": ""},
					},
					"port": []map[string]any{
						{"name": "", "port": int64(80), "protocol": "TCP"},
					},
				},
			},
			wantSubset: []map[string]any{
				{
					"address": []map[string]any{
						{"ip": "10.0.0.1", "hostname": nil, "node_name": nil},
					},
					"not_ready_address": nil,
					"port": []map[string]any{
						{"name": nil, "port": int64(80), "protocol": "TCP"},
					},
				},
			},
		},
		{
			name:             "without target identity",
			providerAddress:  "registry.terraform.io/hashicorp/kubernetes",
			noTargetIdentity: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			source := map[string]any{
				"id": "default/endpoints-test",
				"metadata": []map[string]any{{
					"name": "endpoints-test", "namespace": "default", "generate_name": testCase.generateName,
					"annotations": testCase.annotations, "labels": testCase.labels,
					"uid": "original-uid", "generation": 1, "resource_version": "1234",
				}},
				"subset": testCase.subset,
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
				SourceTypeName: "kubernetes_endpoints", SourceSchemaVersion: 0,
				SourceProviderAddress: testCase.providerAddress,
				SourceRawState:        &tfprotov6.RawState{JSON: sourceJSON},
			}, &response)
			if response.Diagnostics.HasError() {
				t.Fatal(response.Diagnostics)
			}

			expectedJSON, err := json.Marshal(map[string]any{
				"id": "default/endpoints-test",
				"metadata": []map[string]any{{
					"name": "endpoints-test", "namespace": "default", "generate_name": testCase.wantGenerateName,
					"annotations": testCase.annotations, "labels": testCase.labels,
					"uid": "original-uid", "generation": 1, "resource_version": "1234",
				}},
				"subset": testCase.wantSubset,
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
				var identity common.NamespacedResourceIdentity
				if diags := response.TargetIdentity.Get(ctx, &identity); diags.HasError() {
					t.Fatal(diags)
				}
				if identity.Name != types.StringValue("endpoints-test") || identity.Namespace != types.StringValue("default") || identity.APIVersion != types.StringValue("v1") || identity.Kind != types.StringValue("Endpoints") {
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
		{name: "wrong resource", sourceType: "kubernetes_namespace", providerAddress: "registry.terraform.io/hashicorp/kubernetes"},
		{name: "wrong schema version", sourceType: "kubernetes_endpoints", sourceVersion: 1, providerAddress: "registry.terraform.io/hashicorp/kubernetes"},
		{name: "wrong provider", sourceType: "kubernetes_endpoints", providerAddress: "registry.terraform.io/other/kubernetes"},
		{name: "missing raw state", sourceType: "kubernetes_endpoints", providerAddress: "registry.terraform.io/hashicorp/kubernetes", wantError: true},
		{name: "malformed raw state", sourceType: "kubernetes_endpoints", providerAddress: "registry.terraform.io/hashicorp/kubernetes", rawState: &tfprotov6.RawState{JSON: []byte(`{`)}, wantError: true},
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
