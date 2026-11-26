// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package autoscalingv2

import (
	"context"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

// sdkv2State is kubernetes_horizontal_pod_autoscaler_v2 state as written by provider 3.5.0.
const sdkv2State = `{
  "id": "default/test",
  "metadata": [{
    "annotations": {}, "generate_name": "", "generation": 0, "labels": {},
    "name": "test", "namespace": "default", "resource_version": "1", "uid": "u"
  }],
  "spec": [{
    "max_replicas": 10,
    "min_replicas": 1,
    "target_cpu_utilization_percentage": 0,
    "scale_target_ref": [{"api_version": "", "kind": "Deployment", "name": "app"}],
    "metric": [
      {
        "type": "Resource",
        "container_resource": [], "external": [], "object": [], "pods": [],
        "resource": [{"name": "cpu", "target": [{"type": "Utilization", "average_utilization": 80, "average_value": "", "value": ""}]}]
      },
      {
        "type": "External",
        "container_resource": [], "object": [], "pods": [], "resource": [],
        "external": [{
          "metric": [{"name": "queue", "selector": [{
            "match_labels": {},
            "match_expressions": [{"key": "tier", "operator": "Exists", "values": []}]
          }]}],
          "target": [{"type": "Value", "average_utilization": 0, "average_value": "", "value": "10"}]
        }]
      },
      {
        "type": "Object",
        "container_resource": [], "external": [], "pods": [], "resource": [],
        "object": [{
          "described_object": [{"api_version": "networking.k8s.io/v1", "kind": "Ingress", "name": "main"}],
          "metric": [{"name": "rps", "selector": []}],
          "target": [{"type": "AverageValue", "average_utilization": 0, "average_value": "2k", "value": ""}]
        }]
      }
    ],
    "behavior": [{
      "scale_down": [{"policy": [{"period_seconds": 15, "type": "Percent", "value": 100}], "select_policy": "", "stabilization_window_seconds": 0}],
      "scale_up": []
    }]
  }]
}`

func TestUpgradeStateV0(t *testing.T) {
	resp := runUpgradeV0(t, sdkv2State)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}

	for p, want := range map[string]any{
		"spec[0].target_cpu_utilization_percentage":                                nil,
		"spec[0].scale_target_ref[0].api_version":                                  nil,
		"spec[0].metric[0].resource.name":                                          "cpu",
		"spec[0].metric[0].resource.target.average_utilization":                    int64(80),
		"spec[0].metric[0].resource.target.value":                                  nil,
		"spec[0].metric[0].resource.target.average_value":                          nil,
		"spec[0].metric[0].external":                                               nil,
		"spec[0].metric[1].external.target.value":                                  "10",
		"spec[0].metric[1].external.target.average_utilization":                    nil,
		"spec[0].metric[1].external.metric.selector.match_labels":                  nil,
		"spec[0].metric[1].external.metric.selector.match_expressions[0].key":      "tier",
		"spec[0].metric[1].external.metric.selector.match_expressions[0].operator": "Exists",
		"spec[0].metric[1].external.metric.selector.match_expressions[0].values":   nil,
		"spec[0].metric[2].object.described_object.kind":                           "Ingress",
		"spec[0].metric[2].object.metric.selector":                                 nil,
		"spec[0].metric[2].object.target.average_value":                            "2k",
		"spec[0].behavior.scale_up":                                                nil,
		"spec[0].behavior.scale_down.select_policy":                                nil,
		"spec[0].behavior.scale_down.stabilization_window_seconds":                 int64(0),
		"spec[0].behavior.scale_down.policy[0].type":                               "Percent",
		"metadata[0].generate_name":                                                nil,
		"metadata[0].name":                                                         "test",
	} {
		assertValue(t, resp.State.Raw, p, want)
	}
}

func TestUpgradeStateV0_errors(t *testing.T) {
	for name, tc := range map[string]struct {
		state string
		err   string
	}{
		"multipleSources": {
			state: strings.Replace(sdkv2State, `"resource": [{"name": "cpu"`, `"resource": [{"name": "a"}, {"name": "cpu"`, 1),
			err:   "spec.metric[0]: resource has 2 elements",
		},
		"invalidID": {
			state: strings.Replace(sdkv2State, `"id": "default/test"`, `"id": "test"`, 1),
			err:   `"test" is not namespace/name`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			resp := runUpgradeV0(t, tc.state)
			if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics[0].Detail(), tc.err) {
				t.Fatalf("expected error containing %q, got %v", tc.err, resp.Diagnostics)
			}
		})
	}
}

func TestMoveState(t *testing.T) {
	ctx := context.Background()
	const address = "registry.terraform.io/hashicorp/kubernetes"
	withoutV2Fields := strings.Replace(strings.Replace(sdkv2State,
		`"behavior": [{`, `"behavior": [], "unused": [{`, 1),
		`"metric": [`, `"metric": [], "unused_metric": [`, 1)

	for name, tc := range map[string]struct {
		typeName, address, state string
		schemaVersion            int64
		moved                    bool
		err                      string
	}{
		"v2beta2":                {typeName: v2beta2HPATypeName, address: address, state: sdkv2State, moved: true},
		"unversioned":            {typeName: unversionedHPATypeName, address: address, state: sdkv2State, moved: true},
		"unversionedV1":          {typeName: unversionedHPATypeName, address: address, state: withoutV2Fields, err: "kubernetes_horizontal_pod_autoscaler_v1"},
		"v2beta2WithoutV2Fields": {typeName: v2beta2HPATypeName, address: address, state: withoutV2Fields, moved: true},
		"otherProvider":          {typeName: v2beta2HPATypeName, address: "registry.terraform.io/example/kubernetes", state: sdkv2State},
		"otherType":              {typeName: "kubernetes_deployment", address: address, state: sdkv2State},
		"otherSchemaVersion":     {typeName: v2beta2HPATypeName, address: address, state: sdkv2State, schemaVersion: 1},
	} {
		t.Run(name, func(t *testing.T) {
			r := &HorizontalPodAutoscalerV2{}
			identitySchema := common.NamespacedIdentitySchema()
			req := resource.MoveStateRequest{
				SourceTypeName:        tc.typeName,
				SourceProviderAddress: tc.address,
				SourceSchemaVersion:   tc.schemaVersion,
				SourceRawState:        &tfprotov6.RawState{JSON: []byte(tc.state)},
			}
			resp := &resource.MoveStateResponse{
				TargetState: tfsdk.State{Schema: hpaSchema(), Raw: tftypes.NewValue(hpaSchema().Type().TerraformType(ctx), nil)},
				TargetIdentity: &tfsdk.ResourceIdentity{
					Schema: identitySchema,
					Raw:    tftypes.NewValue(identitySchema.Type().TerraformType(ctx), nil),
				},
			}
			for _, mover := range r.MoveState(ctx) {
				mover.StateMover(ctx, req, resp)
				if resp.Diagnostics.HasError() || !resp.TargetState.Raw.IsNull() {
					break
				}
			}

			if tc.err != "" {
				if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics[0].Detail(), tc.err) {
					t.Fatalf("expected error containing %q, got %v", tc.err, resp.Diagnostics)
				}
				return
			}
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			if !tc.moved {
				if !resp.TargetState.Raw.IsNull() {
					t.Fatal("expected the source to be skipped")
				}
				return
			}
			if resp.TargetState.Raw.IsNull() {
				t.Fatal("expected the state to be moved")
			}
			var identity common.NamespacedResourceIdentity
			if diags := resp.TargetIdentity.Get(ctx, &identity); diags.HasError() {
				t.Fatal(diags)
			}
			if identity != hpaIdentity("default", "test") {
				t.Errorf("identity = %+v", identity)
			}
		})
	}
}

func TestQuantityPlanModifier(t *testing.T) {
	ctx := context.Background()
	for name, tc := range map[string]struct {
		state, config types.String
		want          types.String
	}{
		"equivalent":    {state: types.StringValue("1000m"), config: types.StringValue("1"), want: types.StringValue("1000m")},
		"decimal":       {state: types.StringValue("0.5"), config: types.StringValue("500m"), want: types.StringValue("0.5")},
		"changed":       {state: types.StringValue("1000m"), config: types.StringValue("2"), want: types.StringValue("2")},
		"invalid":       {state: types.StringValue("1"), config: types.StringValue("one"), want: types.StringValue("one")},
		"create":        {state: types.StringNull(), config: types.StringValue("1"), want: types.StringValue("1")},
		"removed":       {state: types.StringValue("1"), config: types.StringNull(), want: types.StringNull()},
		"configUnknown": {state: types.StringValue("1"), config: types.StringUnknown(), want: types.StringUnknown()},
	} {
		t.Run(name, func(t *testing.T) {
			resp := &planmodifier.StringResponse{PlanValue: tc.config}
			quantityPlanModifier{}.PlanModifyString(ctx, planmodifier.StringRequest{
				StateValue:  tc.state,
				ConfigValue: tc.config,
				PlanValue:   tc.config,
			}, resp)
			if !resp.PlanValue.Equal(tc.want) {
				t.Errorf("plan = %s, want %s", resp.PlanValue, tc.want)
			}
		})
	}
}

func runUpgradeV0(t *testing.T, raw string) *resource.UpgradeStateResponse {
	t.Helper()
	ctx := context.Background()
	r := &HorizontalPodAutoscalerV2{}
	resp := &resource.UpgradeStateResponse{State: tfsdk.State{Schema: hpaSchema()}}
	r.UpgradeState(ctx)[0].StateUpgrader(ctx, resource.UpgradeStateRequest{
		RawState: &tfprotov6.RawState{JSON: []byte(raw)},
	}, resp)
	return resp
}

var pathStep = regexp.MustCompile(`^([a-z_]+)(?:\[(\d+)\])?$`)

// assertValue compares the value at a dotted path such as "spec[0].metric[1].type" with want,
// where nil means null.
func assertValue(t *testing.T, value tftypes.Value, p string, want any) {
	t.Helper()
	at := tftypes.NewAttributePath()
	for _, step := range strings.Split(p, ".") {
		m := pathStep.FindStringSubmatch(step)
		if m == nil {
			t.Fatalf("invalid path %q", p)
		}
		at = at.WithAttributeName(m[1])
		if m[2] != "" {
			i, _ := strconv.Atoi(m[2])
			at = at.WithElementKeyInt(i)
		}
	}
	found, _, err := tftypes.WalkAttributePath(value, at)
	if err != nil {
		t.Errorf("%s: %s", p, err)
		return
	}
	got := found.(tftypes.Value)
	switch want := want.(type) {
	case nil:
		if !got.IsNull() {
			t.Errorf("%s = %s, want null", p, got)
		}
	case string:
		var s string
		if err := got.As(&s); err != nil || s != want || got.IsNull() {
			t.Errorf("%s = %s, want %q", p, got, want)
		}
	case int64:
		var n big.Float
		if err := got.As(&n); err != nil || got.IsNull() {
			t.Errorf("%s = %s, want %d", p, got, want)
			return
		}
		if i, _ := n.Int64(); i != want {
			t.Errorf("%s = %s, want %d", p, got, want)
		}
	}
}
