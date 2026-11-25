// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package storagev1_test

import (
	"context"
	"strings"
	"testing"

	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	storagev1api "k8s.io/api/storage/v1"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/storagev1"
)

// Move tests: state written for the deprecated kubernetes_csi_driver alias, moved to
// kubernetes_csi_driver_v1 with a moved block. Each migration test has a twin here using
// the same configuration.
//
// The alias is served by the storage.k8s.io/v1beta1 implementation, which Kubernetes
// removed in 1.22, so the acceptance tests need an older cluster.
// TestCSIDriverV1MoveState covers the same translation without one.

func testAccCSIDriverV1MoveSource(config string) string {
	return strings.ReplaceAll(config, `"kubernetes_csi_driver_v1"`, `"kubernetes_csi_driver"`)
}

func testAccCSIDriverV1MoveTarget(config string) string {
	return config + `
moved {
  from = kubernetes_csi_driver.test
  to   = kubernetes_csi_driver_v1.test
}
`
}

// testAccCSIDriverV1MoveFrom is testAccCSIDriverV1UpgradeFrom for a move: step 1 applies
// the alias under the given SDKv2 release, and step 2 adds the moved block. Apart from that
// and the type name the configurations are identical, so a non-empty plan can only come
// from the move.
func testAccCSIDriverV1MoveFrom(t *testing.T, sdkv2Version, config string, sdkv2Drifts bool, firstPlan plancheck.PlanCheck) {
	t.Helper()
	var before, after storagev1api.CSIDriver
	importStep := testAccCSIDriverV1ImportStep()
	importStep.ProtoV6ProviderFactories = testAccProtoV6ProviderFactories

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if testAccClusterVersionAtLeast(t, "1.22.0") {
				t.Skip("kubernetes_csi_driver uses storage.k8s.io/v1beta1, which Kubernetes 1.22 removed")
			}
		},
		// Moves between resource types need 1.8; the identity checks need 1.12.
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		CheckDestroy: testAccCheckCSIDriverV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders:  testAccSDKv2Provider(sdkv2Version),
				Config:             testAccCSIDriverV1MoveSource(config),
				Check:              testAccCheckCSIDriverV1Exists("kubernetes_csi_driver.test", &before),
				ExpectNonEmptyPlan: sdkv2Drifts,
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccCSIDriverV1MoveTarget(config),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{firstPlan},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				ConfigStateChecks: csiDriverIdentityChecks(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCSIDriverV1Exists(csiDriverResourceName, &after),
					testAccCheckCSIDriverV1Recreated(&before, &after, false),
				),
			},
			importStep,
		},
	})
}

func TestAccCSIDriverV1_MoveStateFromAlias_basicName(t *testing.T) {
	testAccCSIDriverV1MoveFrom(t, csiDriverSDKv2ProviderVersion,
		testAccCSIDriverV1Config_basic(acctest.RandomWithPrefix(csiDriverNamePrefix), true),
		false, plancheck.ExpectEmptyPlan())
}

func TestAccCSIDriverV1_MoveStateFromAlias_completeName(t *testing.T) {
	testAccCSIDriverV1MoveFrom(t, csiDriverSDKv2ProviderVersion,
		testAccCSIDriverV1Config_completeName(acctest.RandomWithPrefix(csiDriverNamePrefix), "one", true),
		false, plancheck.ExpectEmptyPlan())
}

func TestAccCSIDriverV1_MoveStateFromAlias_completeGenerateName(t *testing.T) {
	testAccCSIDriverV1MoveFrom(t, csiDriverSDKv2ProviderVersion,
		testAccCSIDriverV1Config_completeGenerateName(csiDriverNamePrefix, "one", true),
		false, plancheck.ExpectEmptyPlan())
}

func TestAccCSIDriverV1_MoveStateFromAlias_apiDefaults(t *testing.T) {
	testAccCSIDriverV1MoveFrom(t, csiDriverSDKv2ProviderVersion,
		testAccCSIDriverV1Config_apiDefaults(acctest.RandomWithPrefix(csiDriverNamePrefix)),
		true, plancheck.ExpectEmptyPlan())
}

func TestAccCSIDriverV1_MoveStateFromAlias_emptyMaps(t *testing.T) {
	testAccCSIDriverV1MoveFrom(t, csiDriverSDKv2ProviderVersion,
		testAccCSIDriverV1Config_emptyMaps(acctest.RandomWithPrefix(csiDriverNamePrefix)),
		false, plancheck.ExpectResourceAction(csiDriverResourceName, plancheck.ResourceActionUpdate))
}

func TestAccCSIDriverV1_MoveStateFromAliasPreIdentity_basicName(t *testing.T) {
	testAccCSIDriverV1MoveFrom(t, csiDriverPreIdentityProviderVersion,
		testAccCSIDriverV1Config_basic(acctest.RandomWithPrefix(csiDriverNamePrefix), true),
		false, plancheck.ExpectEmptyPlan())
}

func TestAccCSIDriverV1_MoveStateFromAliasPreIdentity_completeName(t *testing.T) {
	testAccCSIDriverV1MoveFrom(t, csiDriverPreIdentityProviderVersion,
		testAccCSIDriverV1Config_completeName(acctest.RandomWithPrefix(csiDriverNamePrefix), "one", true),
		false, plancheck.ExpectEmptyPlan())
}

func TestAccCSIDriverV1_MoveStateFromAliasPreIdentity_completeGenerateName(t *testing.T) {
	testAccCSIDriverV1MoveFrom(t, csiDriverPreIdentityProviderVersion,
		testAccCSIDriverV1Config_completeGenerateName(csiDriverNamePrefix, "one", true),
		false, plancheck.ExpectEmptyPlan())
}

// TestCSIDriverV1MoveState runs the mover on SDKv2-shaped JSON. Matching sources are
// translated field by field; any other source is skipped with neither state nor
// diagnostics, so another mover can claim it; unusable source state is an error.
func TestCSIDriverV1MoveState(t *testing.T) {
	ctx := context.Background()
	r := &storagev1.CSIDriverV1{}
	var schema frameworkresource.SchemaResponse
	r.Schema(ctx, frameworkresource.SchemaRequest{}, &schema)
	mover := r.MoveState(ctx)[0].StateMover

	// As kubernetes_csi_driver writes it: generate_name "" when unset, maps null or {}.
	source := `{"id":"driver","metadata":[{"annotations":null,"generate_name":"","generation":0,"labels":{"app":"x"},"name":"driver","resource_version":"7","uid":"uid-1"}],` +
		`"spec":[{"attach_required":true,"pod_info_on_mount":false,"volume_lifecycle_modes":["Persistent"]}]}`
	moved := strings.Replace(source, `"generate_name":""`, `"generate_name":null`, 1)

	for _, tc := range []struct {
		name                  string
		sourceType, provider  string
		version               int64
		raw, want             string
		wantError, noIdentity bool
	}{
		{name: "valid", raw: source, want: moved},
		{name: "generatedName",
			raw:  strings.Replace(source, `"generate_name":""`, `"generate_name":"driver-"`, 1),
			want: strings.Replace(source, `"generate_name":""`, `"generate_name":"driver-"`, 1)},
		{name: "emptyMaps",
			raw:  strings.Replace(source, `"annotations":null`, `"annotations":{}`, 1),
			want: strings.Replace(moved, `"annotations":null`, `"annotations":{}`, 1)},
		{name: "nullVolumeLifecycleModes",
			raw:  strings.Replace(source, `["Persistent"]`, `null`, 1),
			want: strings.Replace(moved, `["Persistent"]`, `null`, 1)},
		{name: "podInfoOnMount",
			raw:  strings.Replace(source, `"pod_info_on_mount":false`, `"pod_info_on_mount":true`, 1),
			want: strings.Replace(moved, `"pod_info_on_mount":false`, `"pod_info_on_mount":true`, 1)},
		{name: "withoutSpec",
			raw:  source[:strings.Index(source, `"spec"`)] + `"spec":[]}`,
			want: moved[:strings.Index(moved, `"spec"`)] + `"spec":[]}`},
		{name: "withoutTargetIdentity", raw: source, want: moved, noIdentity: true},
		{name: "privateRegistry", provider: "registry.example.com/hashicorp/kubernetes", raw: source, want: moved},
		{name: "v1Type", sourceType: "kubernetes_csi_driver_v1", raw: source},
		{name: "otherType", sourceType: "kubernetes_storage_class", raw: source},
		{name: "otherProvider", provider: "registry.terraform.io/example/kubernetes", raw: source},
		{name: "lookalikeProvider", provider: "registry.example.com/nothashicorp/kubernetes", raw: source},
		{name: "otherSchemaVersion", version: 1, raw: source},
		{name: "missingJSON", wantError: true},
		{name: "malformedJSON", raw: "{", wantError: true},
		{name: "emptyID", raw: strings.Replace(source, `"id":"driver"`, `"id":""`, 1), wantError: true},
		{name: "withoutMetadata", raw: source[:strings.Index(source, `"metadata"`)] + `"metadata":[],"spec":[]}`, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sourceType, provider := tc.sourceType, tc.provider
			if sourceType == "" {
				sourceType = "kubernetes_csi_driver"
			}
			if provider == "" {
				provider = "registry.terraform.io/hashicorp/kubernetes"
			}
			req := frameworkresource.MoveStateRequest{
				SourceTypeName:        sourceType,
				SourceSchemaVersion:   tc.version,
				SourceProviderAddress: provider,
			}
			if tc.raw != "" {
				req.SourceRawState = &tfprotov6.RawState{JSON: []byte(tc.raw)}
			}
			resp := frameworkresource.MoveStateResponse{TargetState: tfsdk.State{Schema: schema.Schema}}
			if !tc.noIdentity {
				resp.TargetIdentity = &tfsdk.ResourceIdentity{Schema: common.IdentitySchema()}
			}

			mover(ctx, req, &resp)

			if resp.Diagnostics.HasError() != tc.wantError {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}
			if tc.want == "" {
				if !resp.TargetState.Raw.IsNull() || (resp.TargetIdentity != nil && !resp.TargetIdentity.Raw.IsNull()) {
					t.Fatal("a skipped or invalid source must produce neither state nor identity")
				}
				return
			}

			want, err := (&tfprotov6.RawState{JSON: []byte(tc.want)}).Unmarshal(schema.Schema.Type().TerraformType(ctx))
			if err != nil {
				t.Fatal(err)
			}
			if !resp.TargetState.Raw.Equal(want) {
				t.Errorf("moved state mismatch\nwant: %s\ngot:  %s", want, resp.TargetState.Raw)
			}
			if tc.noIdentity {
				return
			}
			var identity common.ResourceIdentity
			if diags := resp.TargetIdentity.Get(ctx, &identity); diags.HasError() {
				t.Fatal(diags)
			}
			wantIdentity := common.ResourceIdentity{
				Name:       types.StringValue("driver"),
				Kind:       types.StringValue("CSIDriver"),
				APIVersion: types.StringValue("storage.k8s.io/v1"),
			}
			if identity != wantIdentity {
				t.Errorf("moved identity = %+v, want %+v", identity, wantIdentity)
			}
		})
	}
}
