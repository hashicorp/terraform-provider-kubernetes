// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestDeploymentV1RecreateConfiguration(t *testing.T) {
	// Planning exercises Terraform's HCL decoder and the production mux, without
	// creating infrastructure or depending on a configured Kubernetes client.
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{{
			Config: strings.ReplaceAll(testAccKubernetesDeploymentV1ConfigWithDeploymentStrategy(
				"deployment-recreate-plan", "Recreate", busyboxImage), "      rolling_update = null\n", ""),
			PlanOnly:           true,
			ExpectNonEmptyPlan: true,
		}},
	})
}

func TestDeploymentV1LegacyValidationDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name    string
		config  string
		pattern string
	}{
		{
			name:    "restart-policy",
			config:  testAccKubernetesDeploymentV1Config_with_restart_policy("deployment-invalid-plan", busyboxImage, "Never"),
			pattern: `expected spec\.0\.template\.0\.spec\.0\.restart_policy to be one of \["Always"\], got Never`,
		},
		{
			name:    "empty-divisor",
			config:  testAccKubernetesDeploymentV1ConfigWithResourceFieldSelector("deployment-invalid-plan", busyboxImage, "limits.cpu", ""),
			pattern: "quantities must match the regular expression",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Steps: []resource.TestStep{{
					Config: tc.config, PlanOnly: true, ExpectError: regexp.MustCompile(tc.pattern),
				}},
			})
		})
	}
}

func TestDeploymentV1MigrationConfigurations(t *testing.T) {
	for _, strategy := range []string{"RollingUpdate", "Recreate"} {
		t.Run(strategy, func(t *testing.T) {
			resource.UnitTest(t, resource.TestCase{
				Steps: []resource.TestStep{
					{
						ExternalProviders: map[string]resource.ExternalProvider{
							"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: deploymentSDKv2ProviderVersion},
						},
						Config:             testAccDeploymentMigrationFullConfig("deployment-full-plan", strategy, true),
						PlanOnly:           true,
						ExpectNonEmptyPlan: true,
					},
					{
						ProtoV6ProviderFactories: testAccProviderFactories,
						Config:                   testAccDeploymentMigrationFullConfig("deployment-full-plan", strategy, false),
						PlanOnly:                 true,
						ExpectNonEmptyPlan:       true,
					},
				},
			})
		})
	}
	t.Run("pre-identity", func(t *testing.T) {
		resource.UnitTest(t, resource.TestCase{
			Steps: []resource.TestStep{{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: deploymentSDKv2PreIdentityVersion},
				},
				Config:             testAccDeploymentMigrationFullConfig("deployment-pre-identity-plan", "RollingUpdate", true),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			}},
		})
	})
}

func TestDeploymentV1MuxValidateConfig(t *testing.T) {
	ctx := context.Background()
	server, err := testAccProviderFactories["kubernetes"]()
	if err != nil {
		t.Fatal(err)
	}
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	typ := schemas.ResourceSchemas["kubernetes_deployment_v1"].ValueType()
	for _, tc := range []struct {
		name    string
		mutate  func(map[string]any)
		wantErr bool
	}{
		{name: "minimal"},
		{name: "missing-metadata", mutate: func(v map[string]any) { delete(v, "metadata") }, wantErr: true},
		{name: "missing-spec", mutate: func(v map[string]any) { delete(v, "spec") }, wantErr: true},
		{name: "missing-template", mutate: func(v map[string]any) { delete(deploymentProtocolSpec(v), "template") }, wantErr: true},
		{name: "missing-template-spec", mutate: func(v map[string]any) {
			delete(deploymentProtocolSpec(v)["template"].([]any)[0].(map[string]any), "spec")
		}, wantErr: true},
		{name: "invalid-replicas", mutate: func(v map[string]any) { deploymentProtocolSpec(v)["replicas"] = "one" }, wantErr: true},
		{name: "empty-replicas", mutate: func(v map[string]any) { deploymentProtocolSpec(v)["replicas"] = "" }},
		{name: "zero-replicas", mutate: func(v map[string]any) { deploymentProtocolSpec(v)["replicas"] = "0" }},
		{name: "invalid-strategy", mutate: func(v map[string]any) {
			deploymentProtocolSpec(v)["strategy"] = []any{map[string]any{"type": "NotAStrategy"}}
		}, wantErr: true},
		{name: "recreate-omits-rolling-update", mutate: func(v map[string]any) {
			deploymentProtocolSpec(v)["strategy"] = []any{map[string]any{"type": "Recreate"}}
		}},
		{name: "invalid-max-surge", mutate: func(v map[string]any) {
			deploymentProtocolSpec(v)["strategy"] = []any{map[string]any{"rolling_update": []any{map[string]any{"max_surge": "-1"}}}}
		}, wantErr: true},
		{name: "invalid-restart-policy", mutate: func(v map[string]any) {
			template := deploymentProtocolSpec(v)["template"].([]any)[0].(map[string]any)
			template["spec"].([]any)[0].(map[string]any)["restart_policy"] = "OnFailure"
		}, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values := map[string]any{
				"metadata": []any{map[string]any{"name": "deployment-validation"}},
				"spec": []any{map[string]any{
					"selector": []any{map[string]any{"match_labels": map[string]any{"app": "validation"}}},
					"template": []any{map[string]any{
						"metadata": []any{map[string]any{"labels": map[string]any{"app": "validation"}}},
						"spec": []any{map[string]any{
							"container": []any{map[string]any{"name": "main", "image": busyboxImage}},
						}},
					}},
				}},
			}
			if tc.mutate != nil {
				tc.mutate(values)
			}
			encoded, err := json.Marshal(values)
			if err != nil {
				t.Fatal(err)
			}
			value, err := (&tfprotov6.RawState{JSON: encoded}).Unmarshal(typ)
			if err != nil {
				t.Fatal(err)
			}
			config, err := tfprotov6.NewDynamicValue(typ, value)
			if err != nil {
				t.Fatal(err)
			}
			response, err := server.ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{
				TypeName: "kubernetes_deployment_v1",
				Config:   &config,
			})
			if err != nil {
				t.Fatal(err)
			}
			var errors []string
			for _, diagnostic := range response.Diagnostics {
				if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
					errors = append(errors, fmt.Sprintf("%s: %s", diagnostic.Summary, diagnostic.Detail))
				}
			}
			if (len(errors) > 0) != tc.wantErr {
				t.Fatalf("validation errors = %v, want error = %t", errors, tc.wantErr)
			}
		})
	}
}

func deploymentProtocolSpec(value map[string]any) map[string]any {
	return value["spec"].([]any)[0].(map[string]any)
}

func TestDeploymentV1MuxMigrationState(t *testing.T) {
	// Schema v0 was last released under the unversioned alias in v1.13.3.
	// Versions >=2.0.0, including the pre-identity v2.23.0 baseline, use v1.
	ctx := context.Background()
	server, err := testAccProviderFactories["kubernetes"]()
	if err != nil {
		t.Fatal(err)
	}
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	identities, err := server.GetResourceIdentitySchemas(ctx, &tfprotov6.GetResourceIdentitySchemasRequest{})
	if err != nil {
		t.Fatal(err)
	}
	typ := schemas.ResourceSchemas["kubernetes_deployment_v1"].ValueType()
	expectedJSON := []byte(`{
  "id": "default/migrated",
  "metadata": [{"name":"migrated","namespace":"default","uid":"unchanged-uid","generation":3,"resource_version":"41","labels":{"app":"migrated"}}],
  "spec": [{
    "replicas":"2",
    "template":[{
      "metadata":[{"labels":{"app":"migrated"}}],
      "spec":[{
        "container":[{"name":"main","image":"busybox:1.36","resources":[{"limits":{"cpu":"250m"},"requests":{"cpu":"100m"}}]}],
        "init_container":[{"name":"init","image":"busybox:1.36","resources":[{"limits":{"cpu":"500m"},"requests":{"cpu":"100m"}}]}]
      }]
    }]
  }],
  "wait_for_rollout":false
}`)
	var values map[string]any
	if err := json.Unmarshal(expectedJSON, &values); err != nil {
		t.Fatal(err)
	}
	deploymentProtocolPersistedBlocks(schemas.ResourceSchemas["kubernetes_deployment_v1"].Block, values)
	expectedJSON, err = json.Marshal(values)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := (&tfprotov6.RawState{JSON: expectedJSON}).Unmarshal(typ)
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []int64{0, 1} {
		t.Run(fmt.Sprintf("schema-v%d", version), func(t *testing.T) {
			source := string(expectedJSON)
			if version == 0 {
				source = strings.ReplaceAll(source, `"limits":{"cpu":"250m"}`, `"limits":[{"cpu":"250m"}]`)
				source = strings.ReplaceAll(source, `"limits":{"cpu":"500m"}`, `"limits":[{"cpu":"500m"}]`)
				source = strings.ReplaceAll(source, `"requests":{"cpu":"100m"}`, `"requests":[{"cpu":"100m"}]`)
			}
			raw := &tfprotov6.RawState{JSON: []byte(source)}
			upgraded, err := server.UpgradeResourceState(ctx, &tfprotov6.UpgradeResourceStateRequest{
				TypeName: "kubernetes_deployment_v1", Version: version, RawState: raw,
			})
			if err != nil {
				t.Fatal(err)
			}
			deploymentProtocolNoErrors(t, upgraded.Diagnostics)
			if upgraded.UpgradedState == nil {
				t.Fatal("upgrade returned no state")
			}
			upgradedValue, err := upgraded.UpgradedState.Unmarshal(typ)
			if err != nil || !upgradedValue.Equal(expected) {
				diff, _ := upgradedValue.Diff(expected)
				t.Fatalf("upgrade changed state outside the historical resource-map conversion: %v; differences: %v", err, diff)
			}
			moved, err := server.MoveResourceState(ctx, &tfprotov6.MoveResourceStateRequest{
				SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes",
				SourceTypeName:        "kubernetes_deployment",
				SourceSchemaVersion:   version,
				SourceState:           raw,
				TargetTypeName:        "kubernetes_deployment_v1",
			})
			if err != nil {
				t.Fatal(err)
			}
			deploymentProtocolNoErrors(t, moved.Diagnostics)
			if moved.TargetState == nil || moved.TargetIdentity == nil || moved.TargetIdentity.IdentityData == nil {
				t.Fatal("alias move must return complete state and provider identity")
			}
			movedValue, err := moved.TargetState.Unmarshal(typ)
			if err != nil || !movedValue.Equal(expected) {
				t.Fatalf("alias move changed configured state: %v", err)
			}
			identityType := identities.IdentitySchemas["kubernetes_deployment_v1"].ValueType()
			identity, err := moved.TargetIdentity.IdentityData.Unmarshal(identityType)
			if err != nil {
				t.Fatal(err)
			}
			expectedIdentity, err := (&tfprotov6.RawState{JSON: []byte(`{"api_version":"apps/v1","kind":"Deployment","name":"migrated","namespace":"default"}`)}).Unmarshal(identityType)
			if err != nil || !identity.Equal(expectedIdentity) {
				t.Fatalf("alias move did not preserve identity: %v", err)
			}
		})
	}
}

func deploymentProtocolNoErrors(t *testing.T, diagnostics []*tfprotov6.Diagnostic) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("%s: %s", diagnostic.Summary, diagnostic.Detail)
		}
	}

}

// SDKv2 persists absent list/set blocks as empty collections, not JSON null.
func deploymentProtocolPersistedBlocks(schema *tfprotov6.SchemaBlock, values map[string]any) {
	for _, block := range schema.BlockTypes {
		switch block.Nesting {
		case tfprotov6.SchemaNestedBlockNestingModeList, tfprotov6.SchemaNestedBlockNestingModeSet:
			if values[block.TypeName] == nil {
				values[block.TypeName] = []any{}
			}
			for _, value := range values[block.TypeName].([]any) {
				deploymentProtocolPersistedBlocks(block.Block, value.(map[string]any))
			}
		}

	}
}

func TestDeploymentV1MuxRejectsInvalidMoves(t *testing.T) {
	server, err := testAccProviderFactories["kubernetes"]()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		version int64
		source  string
		address string
		state   string
	}{
		{name: "unsupported-schema", version: 2, source: "kubernetes_deployment", address: "registry.terraform.io/hashicorp/kubernetes", state: `{}`},
		{name: "unrelated-source", version: 1, source: "kubernetes_pod", address: "registry.terraform.io/hashicorp/kubernetes", state: `{}`},
		{name: "unrelated-provider", version: 1, source: "kubernetes_deployment", address: "registry.terraform.io/example/kubernetes", state: `{}`},
		{name: "malformed-v0", version: 0, source: "kubernetes_deployment", address: "registry.terraform.io/hashicorp/kubernetes", state: `{"spec":"invalid"}`},
		{name: "missing-id", version: 1, source: "kubernetes_deployment", address: "registry.terraform.io/hashicorp/kubernetes", state: `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response, err := server.MoveResourceState(context.Background(), &tfprotov6.MoveResourceStateRequest{
				SourceProviderAddress: tc.address,
				SourceTypeName:        tc.source,
				SourceSchemaVersion:   tc.version,
				SourceState:           &tfprotov6.RawState{JSON: []byte(tc.state)},
				TargetTypeName:        "kubernetes_deployment_v1",
			})
			if err != nil {
				t.Fatal(err)
			}
			hasError := false
			for _, diagnostic := range response.Diagnostics {
				hasError = hasError || diagnostic.Severity == tfprotov6.DiagnosticSeverityError
			}
			if !hasError || response.TargetState != nil || response.TargetIdentity != nil {
				t.Fatalf("invalid move produced partial success: %#v", response)
			}
		})
	}
}
