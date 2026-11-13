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

// MoveState tests: moving state from the deprecated kubernetes_limit_range alias (stays on
// SDKv2) to this Framework resource.
//
// Distinct from the migration tests next door, which upgrade state in place under the same
// type name. There the Framework decodes SDKv2 state itself; here MoveState translates it
// field by field, so each stored shape is a separate branch to cover.
//
// HCL configs come from limit_range_v1_migration_test.go so both families describe the
// same shapes and cannot drift apart.

// testAccLimitRangeMoveStateSource rewrites a kubernetes_limit_range_v1 config to the
// deprecated alias type, for the step that runs under the released SDKv2 provider.
func testAccLimitRangeMoveStateSource(config string) string {
	return strings.ReplaceAll(config, `"kubernetes_limit_range_v1"`, `"kubernetes_limit_range"`)
}

// testAccLimitRangeMoveStateTarget appends the moved block. Apart from that and the type
// name, the two steps' configs are identical, so a non-empty plan can only come from the
// move itself.
func testAccLimitRangeMoveStateTarget(config string) string {
	return config + `
moved {
  from = kubernetes_limit_range.test
  to   = kubernetes_limit_range_v1.test
}
`
}

// testAccLimitRangeMoveState applies a config as kubernetes_limit_range under the given
// SDKv2 release, then re-plans it as kubernetes_limit_range_v1 under the local build with
// a moved block, and asserts the move plans nothing.
//
// Without MoveState these fail outright: Terraform moves state across resource types only
// when the destination provider implements it. Cross-type moved needs Terraform 1.8+.
func testAccLimitRangeMoveState(t *testing.T, config string) {
	t.Helper()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		CheckDestroy: testAccCheckKubernetesLimitRangeV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						VersionConstraint: sdkv2LimitRangeProviderVersion,
						Source:            "hashicorp/kubernetes",
					},
				},
				Config: testAccLimitRangeMoveStateSource(config),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccLimitRangeMoveStateTarget(config),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

// ── acceptance tests ──────────────────────────────────────────────────────────

// Minimal shape: name + namespace, no spec.
func TestAccLimitRange_MoveStateFromDeprecatedAlias_empty(t *testing.T) {
	name := fmt.Sprintf("tf-move-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccLimitRangeMoveState(t, testAccKubernetesLimitRangeV1MigrationConfig_empty(name))
}

// Container limit with default and default_request — the most common production shape.
func TestAccLimitRange_MoveStateFromDeprecatedAlias_basic(t *testing.T) {
	name := fmt.Sprintf("tf-move-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccLimitRangeMoveState(t, testAccKubernetesLimitRangeV1MigrationConfig_basic(name))
}

// generate_name: the server assigns a name, so this is the shape where metadata.name is
// unknown at plan time and exercises the RequiresReplace / UseStateForUnknown contract.
func TestAccLimitRange_MoveStateFromDeprecatedAlias_generatedName(t *testing.T) {
	prefix := fmt.Sprintf("tf-move-test-%s-", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	ns := fmt.Sprintf("tf-move-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccLimitRangeMoveState(t, testAccKubernetesLimitRangeV1MigrationConfig_generatedName(prefix, ns))
}

// Multiple limits in one spec: each item is translated separately, so this catches a
// field dropped from sdkv2LimitRangeSpecToFramework for any single item type.
func TestAccLimitRange_MoveStateFromDeprecatedAlias_multipleLimits(t *testing.T) {
	name := fmt.Sprintf("tf-move-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccLimitRangeMoveState(t, testAccKubernetesLimitRangeV1MigrationConfig_multipleLimits(name))
}

// Pre-identity: state from before 2.38.0 carries no identity at all. MoveState builds it
// from the moved state rather than carrying it across, so this exercises that branch.
func TestAccLimitRange_MoveStateFromDeprecatedAliasPreIdentity_basic(t *testing.T) {
	name := fmt.Sprintf("tf-move-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		CheckDestroy: testAccCheckKubernetesLimitRangeV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {
						VersionConstraint: sdkv2PreIdentityLimitRangeProviderVersion,
						Source:            "hashicorp/kubernetes",
					},
				},
				Config: testAccLimitRangeMoveStateSource(testAccKubernetesLimitRangeV1MigrationConfig_basic(name)),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccLimitRangeMoveStateTarget(testAccKubernetesLimitRangeV1MigrationConfig_basic(name)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

// ── unit test: StateMover without a cluster ───────────────────────────────────

// TestLimitRangeMoveState exercises the StateMover function directly using crafted JSON,
// without needing TF_ACC or a live cluster. It verifies field-by-field translation and
// every guard condition in one fast test.
func TestLimitRangeMoveState(t *testing.T) {
	ctx := context.Background()

	lr := &corev1.LimitRangeV1{}
	var schemaResp frameworkresource.SchemaResponse
	lr.Schema(ctx, frameworkresource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatal(schemaResp.Diagnostics)
	}
	var identitySchemaResp frameworkresource.IdentitySchemaResponse
	lr.IdentitySchema(ctx, frameworkresource.IdentitySchemaRequest{}, &identitySchemaResp)
	if identitySchemaResp.Diagnostics.HasError() {
		t.Fatal(identitySchemaResp.Diagnostics)
	}
	mover := lr.MoveState(ctx)[0].StateMover

	// ── happy-path cases ──────────────────────────────────────────────────────

	for _, tc := range []struct {
		name         string
		providerAddr string
		annotations  map[string]string
		labels       map[string]string
		generateName string
		wantGenName  any // nil means expect null in state
		spec         []map[string]any
		wantSpec     []map[string]any
	}{
		{
			name:         "empty spec, both maps null",
			providerAddr: "registry.terraform.io/hashicorp/kubernetes",
		},
		{
			name:         "basic container limit, both maps populated",
			providerAddr: "registry.terraform.io/hashicorp/kubernetes",
			annotations:  map[string]string{"example.com/note": "retained"},
			labels:       map[string]string{"app": "demo"},
			spec: []map[string]any{{
				"limit": []map[string]any{{
					"type":                    "Container",
					"default":                 map[string]string{"cpu": "200m", "memory": "512M"},
					"default_request":         map[string]string{"cpu": "100m", "memory": "256M"},
					"max":                     map[string]string(nil),
					"max_limit_request_ratio": map[string]string(nil),
					"min":                     map[string]string(nil),
				}},
			}},
			wantSpec: []map[string]any{{
				"limit": []map[string]any{{
					"type":                    "Container",
					"default":                 map[string]string{"cpu": "200m", "memory": "512M"},
					"default_request":         map[string]string{"cpu": "100m", "memory": "256M"},
					"max":                     map[string]string(nil),
					"max_limit_request_ratio": map[string]string(nil),
					"min":                     map[string]string(nil),
				}},
			}},
		},
		{
			name:         "generate_name stored as real value",
			providerAddr: "mirror.example.com/hashicorp/kubernetes",
			generateName: "generated-",
			wantGenName:  "generated-",
		},
		{
			name:         "generate_name stored as empty string (SDKv2 unset) becomes null",
			providerAddr: "registry.terraform.io/hashicorp/kubernetes",
			generateName: "", // SDKv2 unset: writes ""
			wantGenName:  nil,
		},
		{
			name:         "multiple limits",
			providerAddr: "registry.terraform.io/hashicorp/kubernetes",
			spec: []map[string]any{{
				"limit": []map[string]any{
					{
						"type":    "Pod",
						"max":     map[string]string{"cpu": "200m", "memory": "1024M"},
						"default": map[string]string(nil), "default_request": map[string]string(nil),
						"max_limit_request_ratio": map[string]string(nil), "min": map[string]string(nil),
					},
					{
						"type":    "PersistentVolumeClaim",
						"min":     map[string]string{"storage": "24M"},
						"default": map[string]string(nil), "default_request": map[string]string(nil),
						"max": map[string]string(nil), "max_limit_request_ratio": map[string]string(nil),
					},
				},
			}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := map[string]any{
				"id":   "default/my-lr",
				"spec": tc.spec,
				"metadata": []map[string]any{{
					"name": "my-lr", "namespace": "default",
					"generate_name":    tc.generateName,
					"annotations":      tc.annotations,
					"labels":           tc.labels,
					"uid":              "original-uid",
					"generation":       int64(1),
					"resource_version": "42",
				}},
			}
			sourceJSON, err := json.Marshal(source)
			if err != nil {
				t.Fatal(err)
			}

			resp := frameworkresource.MoveStateResponse{
				TargetState:    tfsdk.State{Schema: schemaResp.Schema},
				TargetIdentity: &tfsdk.ResourceIdentity{Schema: identitySchemaResp.IdentitySchema},
			}
			mover(ctx, frameworkresource.MoveStateRequest{
				SourceTypeName:        "kubernetes_limit_range",
				SourceSchemaVersion:   0,
				SourceProviderAddress: tc.providerAddr,
				SourceRawState:        &tfprotov6.RawState{JSON: sourceJSON},
			}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}

			// Assert identity — always built from the moved state.
			var identity common.NamespacedResourceIdentity
			if diags := resp.TargetIdentity.Get(ctx, &identity); diags.HasError() {
				t.Fatal(diags)
			}
			if identity.Name != types.StringValue("my-lr") {
				t.Errorf("identity.Name = %s, want my-lr", identity.Name)
			}
			if identity.Namespace != types.StringValue("default") {
				t.Errorf("identity.Namespace = %s, want default", identity.Namespace)
			}
			if identity.APIVersion != types.StringValue("v1") {
				t.Errorf("identity.APIVersion = %s, want v1", identity.APIVersion)
			}
			if identity.Kind != types.StringValue("LimitRange") {
				t.Errorf("identity.Kind = %s, want LimitRange", identity.Kind)
			}

			// Assert generate_name: null when wantGenName is nil, value otherwise.
			wantSpec := tc.wantSpec
			if wantSpec == nil {
				wantSpec = tc.spec // spec is copied unchanged
			}
			expected := map[string]any{
				"id":   "default/my-lr",
				"spec": wantSpec,
				"metadata": []map[string]any{{
					"name": "my-lr", "namespace": "default",
					"generate_name":    tc.wantGenName,
					"annotations":      tc.annotations,
					"labels":           tc.labels,
					"uid":              "original-uid",
					"generation":       int64(1),
					"resource_version": "42",
				}},
			}
			expectedJSON, err := json.Marshal(expected)
			if err != nil {
				t.Fatal(err)
			}
			expectedRaw := &tfprotov6.RawState{JSON: expectedJSON}
			expectedVal, err := expectedRaw.Unmarshal(schemaResp.Schema.Type().TerraformType(ctx))
			if err != nil {
				t.Fatal(err)
			}
			if !resp.TargetState.Raw.Equal(expectedVal) {
				t.Errorf("moved state mismatch\nwant: %s\ngot:  %s", expectedVal, resp.TargetState.Raw)
			}
		})
	}

	// ── guard / error cases ───────────────────────────────────────────────────

	for _, tc := range []struct {
		name          string
		sourceType    string
		sourceVersion int64
		providerAddr  string
		rawState      *tfprotov6.RawState
		wantError     bool
	}{
		{
			name:       "wrong resource type — no state, no error",
			sourceType: "kubernetes_pod", providerAddr: "registry.terraform.io/hashicorp/kubernetes",
		},
		{
			name:       "wrong schema version — no state, no error",
			sourceType: "kubernetes_limit_range", sourceVersion: 1,
			providerAddr: "registry.terraform.io/hashicorp/kubernetes",
		},
		{
			name:       "wrong provider — no state, no error",
			sourceType: "kubernetes_limit_range", providerAddr: "registry.terraform.io/other/kubernetes",
		},
		{
			name:       "nil raw state — error",
			sourceType: "kubernetes_limit_range", providerAddr: "registry.terraform.io/hashicorp/kubernetes",
			wantError: true,
		},
		{
			name:       "malformed JSON — error",
			sourceType: "kubernetes_limit_range", providerAddr: "registry.terraform.io/hashicorp/kubernetes",
			rawState:  &tfprotov6.RawState{JSON: []byte(`{`)},
			wantError: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := frameworkresource.MoveStateResponse{
				TargetState:    tfsdk.State{Schema: schemaResp.Schema},
				TargetIdentity: &tfsdk.ResourceIdentity{Schema: identitySchemaResp.IdentitySchema},
			}
			mover(ctx, frameworkresource.MoveStateRequest{
				SourceTypeName:        tc.sourceType,
				SourceSchemaVersion:   tc.sourceVersion,
				SourceProviderAddress: tc.providerAddr,
				SourceRawState:        tc.rawState,
			}, &resp)
			if resp.Diagnostics.HasError() != tc.wantError {
				t.Errorf("HasError = %v, want %v; diags: %v", resp.Diagnostics.HasError(), tc.wantError, resp.Diagnostics)
			}
			if !resp.TargetState.Raw.IsNull() || !resp.TargetIdentity.Raw.IsNull() {
				t.Fatal("unsupported or invalid source must not produce state or identity")
			}
		})
	}
}
