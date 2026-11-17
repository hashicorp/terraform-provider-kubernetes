// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package autoscalingv1_test

import (
	"context"
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
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/autoscalingv1"
)

const hpaUnversionedResourceName = "kubernetes_horizontal_pod_autoscaler.test"

var maxReplicasPattern = regexp.MustCompile(`max_replicas\s*=\s*\d+`)

func testAccHorizontalPodAutoscalerV1MoveSource(config string) string {
	return strings.ReplaceAll(config, `"kubernetes_horizontal_pod_autoscaler_v1"`, `"kubernetes_horizontal_pod_autoscaler"`)
}

func testAccHorizontalPodAutoscalerV1MoveTarget(config string) string {
	return config + `
moved {
  from = kubernetes_horizontal_pod_autoscaler.test
  to   = kubernetes_horizontal_pod_autoscaler_v1.test
}
`
}

func TestAccKubernetesHorizontalPodAutoscalerV1_movedFromUnversioned(t *testing.T) {
	for _, sc := range hpaMigrationScenarios() {
		t.Run(sc.name, func(t *testing.T) {
			testAccHorizontalPodAutoscalerV1Move(t, sc, resource.TestStep{
				ExternalProviders: externalProvider(hpaSDKv2ProviderVersion),
			})
		})
	}
}

// TestAccKubernetesHorizontalPodAutoscalerV1_movedFromUnversionedLocal moves state written by
// this build's SDKv2 unversioned resource, which must stay compatible with the mover.
func TestAccKubernetesHorizontalPodAutoscalerV1_movedFromUnversionedLocal(t *testing.T) {
	testAccHorizontalPodAutoscalerV1Move(t, hpaMigrationScenarios()[0], resource.TestStep{
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
	})
}

func testAccHorizontalPodAutoscalerV1Move(t *testing.T, sc hpaMigrationScenario, source resource.TestStep) {
	t.Helper()
	var uid string
	name := "tf-acc-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	config := sc.config(name)
	updated := maxReplicasPattern.ReplaceAllString(config, "max_replicas = 9")

	source.Config = testAccHorizontalPodAutoscalerV1MoveSource(config)
	source.Check = testAccHorizontalPodAutoscalerCheckExists(hpaUnversionedResourceName, &uid)
	steps := []resource.TestStep{
		source,
		{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Config:                   testAccHorizontalPodAutoscalerV1MoveTarget(config),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
			},
			Check: testAccHorizontalPodAutoscalerCheckExists(hpaV1ResourceName, &uid),
		},
	}
	if sc.importable {
		steps = append(steps, resource.TestStep{
			ResourceName:             hpaV1ResourceName,
			ImportState:              true,
			ImportStateVerify:        true,
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		})
	}
	steps = append(steps,
		resource.TestStep{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Config:                   updated,
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(hpaV1ResourceName, plancheck.ResourceActionUpdate)},
			},
			Check: resource.ComposeAggregateTestCheckFunc(
				testAccHorizontalPodAutoscalerCheckExists(hpaV1ResourceName, &uid),
				resource.TestCheckResourceAttr(hpaV1ResourceName, "spec.0.max_replicas", "9"),
			),
		},
		resource.TestStep{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Config:                   updated,
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
			},
		},
	)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:               func() { testAccPreCheck(t) },
		CheckDestroy:           testAccHorizontalPodAutoscalerCheckDestroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_8_0)},
		Steps:                  steps,
	})
}

func TestHorizontalPodAutoscalerV1MoveState(t *testing.T) {
	ctx := context.Background()
	r := &autoscalingv1.HorizontalPodAutoscalerV1{}
	var schema frameworkresource.SchemaResponse
	r.Schema(ctx, frameworkresource.SchemaRequest{}, &schema)

	valid := `{"id":"team/web","metadata":[{"name":"web","namespace":"team","uid":"retained","generation":1,"resource_version":"7",` +
		`"labels":{},"annotations":null,"generate_name":""}],` +
		`"spec":[{"max_replicas":5,"min_replicas":1,"target_cpu_utilization_percentage":0,` +
		`"scale_target_ref":[{"api_version":"","kind":"Deployment","name":"web"}],"metric":[],"behavior":[]}]}`
	withCPU := strings.Replace(valid, `"target_cpu_utilization_percentage":0`, `"target_cpu_utilization_percentage":60`, 1)
	withRefVersion := strings.Replace(valid, `"api_version":""`, `"api_version":"apps/v1"`, 1)
	for _, tc := range []struct {
		name, sourceType, provider, raw string
		version                         int64
		wantError, wantSkip             bool
		wantDetail                      string
	}{
		{name: "valid", raw: valid},
		{name: "targetCPU", raw: withCPU},
		{name: "apiVersion", raw: withRefVersion},
		{name: "generatedName", raw: strings.Replace(valid, `"generate_name":""`, `"generate_name":"web-"`, 1)},
		{name: "omittedMetricAndBehavior", raw: strings.Replace(valid, `,"metric":[],"behavior":[]`, "", 1)},
		{name: "privateRegistry", provider: "registry.example.com/hashicorp/kubernetes", raw: valid},
		{name: "wrongType", sourceType: "kubernetes_horizontal_pod_autoscaler_v2", raw: valid, wantSkip: true},
		{name: "wrongProvider", provider: "registry.terraform.io/example/kubernetes", raw: valid, wantSkip: true},
		{name: "lookalikeProvider", provider: "registry.example.com/nothashicorp/kubernetes", raw: valid, wantSkip: true},
		{name: "wrongVersion", version: 1, raw: valid, wantSkip: true},
		{name: "missingJSON", wantError: true},
		{name: "malformedJSON", raw: "{", wantError: true},
		{name: "emptyID", raw: `{"metadata":[{}]}`, wantError: true, wantDetail: "The source state must have a non-empty id."},
		{name: "missingMetadata", raw: `{"id":"team/web"}`, wantError: true, wantDetail: "Expected exactly 1 metadata element in the source state, got 0."},
		{name: "missingSpec", raw: `{"id":"team/web","metadata":[{}]}`, wantError: true, wantDetail: "Expected exactly 1 spec element in the source state, got 0."},
		{name: "missingScaleTargetRef", raw: strings.Replace(valid, `[{"api_version":"","kind":"Deployment","name":"web"}]`, `[]`, 1),
			wantError: true, wantDetail: "Expected exactly 1 spec.scale_target_ref element in the source state, got 0."},
		{name: "metric", raw: strings.Replace(valid, `"metric":[]`, `"metric":[{"type":"Resource"}]`, 1), wantError: true,
			wantDetail: "The source state sets spec.metric or spec.behavior, which kubernetes_horizontal_pod_autoscaler_v1 does not support. " +
				"Manage this autoscaler with kubernetes_horizontal_pod_autoscaler_v2 instead."},
		{name: "behavior", raw: strings.Replace(valid, `"behavior":[]`, `"behavior":[{"scale_up":[]}]`, 1), wantError: true,
			wantDetail: "The source state sets spec.metric or spec.behavior, which kubernetes_horizontal_pod_autoscaler_v1 does not support. " +
				"Manage this autoscaler with kubernetes_horizontal_pod_autoscaler_v2 instead."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := frameworkresource.MoveStateRequest{
				SourceTypeName:        "kubernetes_horizontal_pod_autoscaler",
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
			resp := frameworkresource.MoveStateResponse{
				TargetState: tfsdk.State{Schema: schema.Schema, Raw: tftypes.NewValue(schema.Schema.Type().TerraformType(ctx), nil)},
			}
			r.MoveState(ctx)[0].StateMover(ctx, req, &resp)
			if resp.Diagnostics.HasError() != tc.wantError {
				t.Fatalf("diagnostics: %v", resp.Diagnostics)
			}
			if tc.wantDetail != "" && (len(resp.Diagnostics) != 1 || resp.Diagnostics[0].Detail() != tc.wantDetail) {
				t.Fatalf("expected diagnostic detail %q, got %v", tc.wantDetail, resp.Diagnostics)
			}
			if tc.wantError || tc.wantSkip {
				if !resp.TargetState.Raw.IsNull() {
					t.Fatal("unexpected target state")
				}
				return
			}

			var state autoscalingv1.HorizontalPodAutoscalerV1Model
			if diags := resp.TargetState.Get(ctx, &state); diags.HasError() {
				t.Fatal(diags)
			}
			md, spec := state.Metadata[0], state.Spec[0]
			if state.ID.ValueString() != "team/web" || md.Namespace.ValueString() != "team" || md.Name.ValueString() != "web" ||
				md.UID.ValueString() != "retained" || md.ResourceVersion.ValueString() != "7" || md.Generation.ValueInt64() != 1 ||
				spec.MaxReplicas.ValueInt64() != 5 || spec.MinReplicas.ValueInt64() != 1 ||
				spec.ScaleTargetRef[0].Kind.ValueString() != "Deployment" || spec.ScaleTargetRef[0].Name.ValueString() != "web" {
				t.Fatalf("incorrect state: %#v", state)
			}
			if md.Labels.IsNull() || !md.Annotations.IsNull() {
				t.Fatalf("map nullness not preserved: labels=%v annotations=%v", md.Labels, md.Annotations)
			}
			if got, want := md.GenerateName.ValueString(), map[bool]string{true: "web-"}[tc.name == "generatedName"]; got != want ||
				md.GenerateName.IsNull() != (want == "") {
				t.Fatalf("generate_name = %v", md.GenerateName)
			}
			if got, want := spec.TargetCPUUtilizationPercentage, int64(60); tc.name == "targetCPU" {
				if got.ValueInt64() != want {
					t.Fatalf("target_cpu_utilization_percentage = %v", got)
				}
			} else if !got.IsNull() {
				t.Fatalf("unset target_cpu_utilization_percentage must normalize to null, got %v", got)
			}
			if got := spec.ScaleTargetRef[0].APIVersion; tc.name == "apiVersion" {
				if got.ValueString() != "apps/v1" {
					t.Fatalf("api_version = %v", got)
				}
			} else if !got.IsNull() {
				t.Fatalf("unset api_version must normalize to null, got %v", got)
			}
		})
	}
}
