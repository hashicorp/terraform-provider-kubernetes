// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	jsonpatch "gopkg.in/evanphx/json-patch.v4"
	k8sv1 "k8s.io/api/core/v1"
)

// Configuration validation and expansion unit tests.

func TestServiceAffinityValidateConfig(t *testing.T) {
	for _, config := range []any{nil, tftypes.UnknownValue, map[string]any{}, map[string]any{"client_ip": nil}, map[string]any{"client_ip": tftypes.UnknownValue}, map[string]any{"client_ip": map[string]any{}}, map[string]any{"client_ip": map[string]any{"timeout_seconds": tftypes.UnknownValue}}, map[string]any{"client_ip": map[string]any{"timeout_seconds": 300}}} {
		for _, affinity := range []any{nil, "None", "ClientIP", tftypes.UnknownValue} {
			t.Run(fmt.Sprintf("%v/%v", affinity, config), func(t *testing.T) {
				var response resource.ValidateConfigResponse
				(&ServiceV1{}).ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: serviceAffinityValidationConfig(t, []any{map[string]any{"session_affinity": affinity, "session_affinity_config": config}})}, &response)
				serviceAffinityCheckDiagnostics(t, response.Diagnostics)
			})
		}
	}
}

func TestServiceAffinityExpandDefaults(t *testing.T) {
	for _, tc := range []struct {
		name        string
		config      any
		wantConfig  bool
		wantClient  bool
		wantTimeout int32
	}{
		{name: "omitted config"},
		{name: "unknown config", config: tftypes.UnknownValue},
		{name: "unknown client", config: map[string]any{"client_ip": tftypes.UnknownValue}, wantConfig: true},
		{name: "default config", config: map[string]any{}, wantConfig: true},
		{name: "default client", config: map[string]any{"client_ip": map[string]any{}}, wantConfig: true, wantClient: true},
		{name: "unknown timeout", config: map[string]any{"client_ip": map[string]any{"timeout_seconds": tftypes.UnknownValue}}, wantConfig: true, wantClient: true},
		{name: "explicit timeout", config: map[string]any{"client_ip": map[string]any{"timeout_seconds": 300}}, wantConfig: true, wantClient: true, wantTimeout: 300},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			config := serviceAffinityValidationConfig(t, []any{map[string]any{
				"session_affinity": "ClientIP", "session_affinity_config": tc.config,
			}})
			var model ServiceV1Model
			if diagnostics := config.Get(ctx, &model); diagnostics.HasError() {
				t.Fatal(diagnostics)
			}
			expanded, diagnostics := expandServiceSpec(ctx, model.Spec[0])
			serviceAffinityCheckDiagnostics(t, diagnostics)
			affinity := expanded.SessionAffinityConfig
			if (affinity != nil) != tc.wantConfig {
				t.Fatalf("config = %#v, want present %t", affinity, tc.wantConfig)
			}
			if affinity == nil {
				return
			}
			if (affinity.ClientIP != nil) != tc.wantClient {
				t.Fatalf("client_ip = %#v, want present %t", affinity.ClientIP, tc.wantClient)
			}
			if affinity.ClientIP == nil {
				return
			}
			timeout := affinity.ClientIP.TimeoutSeconds
			if tc.wantTimeout == 0 {
				if timeout != nil {
					t.Fatalf("omitted/unknown timeout became an explicit API value: %d", *timeout)
				}
			} else if timeout == nil || *timeout != tc.wantTimeout {
				t.Fatalf("timeout = %v, want %d", timeout, tc.wantTimeout)
			}
		})
	}
}

func serviceAffinityValidationConfig(t *testing.T, spec any) tfsdk.Config {
	t.Helper()
	var response resource.SchemaResponse
	NewServiceV1().Schema(context.Background(), resource.SchemaRequest{}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	return tfsdk.Config{
		Schema: response.Schema,
		Raw: serviceAffinityValidationValue(t, response.Schema.Type().TerraformType(context.Background()), map[string]any{
			"metadata": []any{map[string]any{"name": "affinity-validation", "namespace": "default"}},
			"spec":     spec,
		}),
	}
}

func serviceAffinityValidationValue(t *testing.T, typ tftypes.Type, value any) tftypes.Value {
	t.Helper()
	switch value := value.(type) {
	case map[string]any:
		objectType := typ.(tftypes.Object)
		values := make(map[string]tftypes.Value, len(objectType.AttributeTypes))
		for name, attributeType := range objectType.AttributeTypes {
			values[name] = serviceAffinityValidationValue(t, attributeType, value[name])
		}
		for name := range value {
			if _, ok := objectType.AttributeTypes[name]; !ok {
				t.Fatalf("test fixture attribute %q is absent from production schema", name)
			}
		}
		return tftypes.NewValue(typ, values)
	case []any:
		elementType := typ.(tftypes.List).ElementType
		values := make([]tftypes.Value, len(value))
		for i, element := range value {
			values[i] = serviceAffinityValidationValue(t, elementType, element)
		}
		return tftypes.NewValue(typ, values)
	default:
		return tftypes.NewValue(typ, value)
	}
}

func serviceAffinityCheckDiagnostics(t *testing.T, diagnostics diag.Diagnostics) {
	t.Helper()
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
}

func TestServiceAllocationNodePortValidation(t *testing.T) {
	for _, tc := range []struct {
		name     string
		kind     types.String
		allocate types.Bool
		port     types.Int64
		reject   bool
	}{
		{"NodePort-zero", types.StringValue("NodePort"), types.BoolNull(), types.Int64Value(0), true},
		{"NodePort-null", types.StringValue("NodePort"), types.BoolNull(), types.Int64Null(), false},
		{"NodePort-unknown", types.StringValue("NodePort"), types.BoolNull(), types.Int64Unknown(), false},
		{"NodePort-explicit", types.StringValue("NodePort"), types.BoolNull(), types.Int64Value(31000), false},
		{"LB-default-zero", types.StringValue("LoadBalancer"), types.BoolNull(), types.Int64Value(0), true},
		{"LB-enabled-zero", types.StringValue("LoadBalancer"), types.BoolValue(true), types.Int64Value(0), true},
		{"LB-disabled-zero", types.StringValue("LoadBalancer"), types.BoolValue(false), types.Int64Value(0), false},
		{"LB-unknown-allocation", types.StringValue("LoadBalancer"), types.BoolUnknown(), types.Int64Value(0), false},
		{"ClusterIP-zero", types.StringValue("ClusterIP"), types.BoolNull(), types.Int64Value(0), false},
		{"ExternalName-zero", types.StringValue("ExternalName"), types.BoolNull(), types.Int64Value(0), false},
		{"default-type-zero", types.StringNull(), types.BoolNull(), types.Int64Value(0), false},
		{"unknown-type-zero", types.StringUnknown(), types.BoolNull(), types.Int64Value(0), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			diags := validateServiceNodePortAllocation(tc.kind, tc.allocate, tc.port, path.Root("node_port"))
			if diags.HasError() != tc.reject {
				t.Fatalf("reject=%t, diagnostics=%v", tc.reject, diags)
			}
		})
	}
}

func TestServiceAllocationIPValidation(t *testing.T) {
	empty := types.ListValueMust(types.StringType, []attr.Value{})
	assigned := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("IPv4")})
	for _, tc := range []struct {
		name   string
		kind   types.String
		value  types.List
		reject bool
	}{
		{"ClusterIP-empty", types.StringValue("ClusterIP"), empty, true},
		{"NodePort-empty", types.StringValue("NodePort"), empty, true},
		{"LB-empty", types.StringValue("LoadBalancer"), empty, true},
		{"default-type-empty", types.StringNull(), empty, true},
		{"ExternalName-empty", types.StringValue("ExternalName"), empty, false},
		{"unknown-type-empty", types.StringUnknown(), empty, false},
		{"omitted", types.StringValue("ClusterIP"), types.ListNull(types.StringType), false},
		{"unknown-list", types.StringValue("ClusterIP"), types.ListUnknown(types.StringType), false},
		{"assigned-list", types.StringValue("ClusterIP"), assigned, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			diags := validateServiceIPAllocations(tc.kind, tc.value, tc.value, path.Root("spec").AtListIndex(0))
			if diags.HasError() != tc.reject {
				t.Fatalf("reject=%t, diagnostics=%v", tc.reject, diags)
			}
		})
	}
}

func TestServiceAllocationTargetPortValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		value  types.String
		reject bool
	}{
		{"empty", types.StringValue(""), true},
		{"zero", types.StringValue("0"), true},
		{"padded-zero", types.StringValue("000"), true},
		{"signed-zero", types.StringValue("+0"), true},
		{"number", types.StringValue("80"), false},
		{"name", types.StringValue("http"), false},
		{"null", types.StringNull(), false},
		{"unknown", types.StringUnknown(), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			diags := validateServiceTargetPort(tc.value, path.Root("target_port"))
			if diags.HasError() != tc.reject {
				t.Fatalf("reject=%t, diagnostics=%v", tc.reject, diags)
			}
		})
	}
}

// Allocation and traffic-policy patch unit tests.

func TestServicePortCombinedEditPreservesAllocation(t *testing.T) {
	prior := ServiceV1PortModel{
		Name: types.StringValue(""), Port: types.Int64Value(80),
		Protocol: types.StringValue("TCP"), AppProtocol: types.StringValue(""),
		NodePort: types.Int64Value(31000), TargetPort: types.StringValue("80"),
	}
	planned := prior
	planned.Name = types.StringValue("http")
	planned.Port = types.Int64Value(81)
	planned.NodePort = types.Int64Unknown()
	planned.TargetPort = types.StringUnknown()
	metrics := ServiceV1PortModel{
		Name: types.StringValue("metrics"), Port: types.Int64Value(9100),
		Protocol: types.StringValue("TCP"), AppProtocol: types.StringValue(""),
		NodePort: types.Int64Value(31001), TargetPort: types.StringValue("9100"),
	}
	plannedMetrics := metrics
	plannedMetrics.NodePort = types.Int64Unknown()
	plannedMetrics.TargetPort = types.StringUnknown()
	namedPrior := prior
	namedPrior.Name = types.StringValue("web")
	for _, tc := range []struct {
		name                   string
		state, plan            []ServiceV1PortModel
		nodePorts, targetPorts []int64
	}{
		{"single", []ServiceV1PortModel{prior}, []ServiceV1PortModel{planned}, []int64{31000}, []int64{80}},
		{"multiple", []ServiceV1PortModel{namedPrior, metrics}, []ServiceV1PortModel{planned, plannedMetrics}, []int64{31000, 31001}, []int64{80, 9100}},
		{"reordered", []ServiceV1PortModel{namedPrior, metrics}, []ServiceV1PortModel{plannedMetrics, planned}, []int64{31001, 31000}, []int64{9100, 80}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var live []k8sv1.ServicePort
			var rawPorts []map[string]any
			for _, p := range tc.state {
				live = append(live, expandServicePort(p))
				rawPorts = append(rawPorts, map[string]any{
					"name": p.Name.ValueString(), "port": p.Port.ValueInt64(), "protocol": "TCP",
					"nodePort": p.NodePort.ValueInt64(), "targetPort": expandServicePort(p).TargetPort.IntVal,
					"futureControllerField": fmt.Sprintf("port-%d", p.NodePort.ValueInt64()),
				})
			}
			ops := servicePortsPatchOps(tc.state, tc.plan, live, false)
			body, err := json.Marshal(ops)
			if err != nil {
				t.Fatal(err)
			}
			patch, err := jsonpatch.DecodePatch(body)
			if err != nil {
				t.Fatal(err)
			}
			initial, err := json.Marshal(map[string]any{"spec": map[string]any{"ports": rawPorts}})
			if err != nil {
				t.Fatal(err)
			}
			result, err := patch.Apply(initial)
			if err != nil {
				t.Fatal(err)
			}
			var actual struct {
				Spec struct {
					Ports []map[string]any `json:"ports"`
				} `json:"spec"`
			}
			if err := json.Unmarshal(result, &actual); err != nil {
				t.Fatal(err)
			}
			if len(actual.Spec.Ports) != len(tc.plan) {
				t.Fatalf("unexpected ports: %s; patch %s", result, body)
			}
			for i, port := range actual.Spec.Ports {
				if port["name"] != tc.plan[i].Name.ValueString() || port["port"] != float64(tc.plan[i].Port.ValueInt64()) {
					t.Errorf("requested edit missing: %s", result)
				}
				if port["nodePort"] != float64(tc.nodePorts[i]) || port["targetPort"] != float64(tc.targetPorts[i]) ||
					port["futureControllerField"] != fmt.Sprintf("port-%d", tc.nodePorts[i]) {
					t.Errorf("combined edit lost allocations or controller fields: %s; patch %s", result, body)
				}
			}
		})
	}
}

func TestServiceClusterIPExternalTrafficPolicyTransition(t *testing.T) {
	for _, sourceType := range []string{"NodePort", "LoadBalancer"} {
		for _, externalIPs := range []bool{false, true} {
			t.Run(sourceType+map[bool]string{false: "/internal", true: "/external"}[externalIPs], func(t *testing.T) {
				state := ServiceV1SpecModel{
					Type: types.StringValue(sourceType), SessionAffinity: types.StringValue("None"),
					AllocateLoadBalancerNodePorts: types.BoolValue(true),
					ExternalTrafficPolicy:         types.StringValue("Local"),
					ExternalIPs:                   types.SetNull(types.StringType),
				}
				live := k8sv1.ServiceSpec{
					Type: k8sv1.ServiceType(sourceType), ExternalTrafficPolicy: k8sv1.ServiceExternalTrafficPolicyLocal,
				}
				if externalIPs {
					state.ExternalIPs = types.SetValueMust(types.StringType, []attr.Value{types.StringValue("192.0.2.1")})
					live.ExternalIPs = []string{"192.0.2.1"}
				}
				plan := state
				plan.Type = types.StringValue("ClusterIP")
				ops, diags := serviceSpecPatchOps(context.Background(), state, plan, live)
				if diags.HasError() {
					t.Fatal(diags)
				}
				removedPolicy := false
				for _, op := range ops {
					if op.Path == "/spec/externalTrafficPolicy" && op.Op == "remove" {
						removedPolicy = true
					}
				}
				if removedPolicy == externalIPs {
					t.Errorf("external IPs=%t: policy removal=%t, want %t; ops %#v", externalIPs, removedPolicy, !externalIPs, ops)
				}
			})
		}
	}
}

func TestServiceStoredAffinityConversion(t *testing.T) {
	for _, tc := range []struct{ source, current string }{
		{`null`, `null`}, {`[]`, `null`}, {`{}`, `{}`}, {`[{}]`, `{}`},
		{`[{"client_ip":[]}]`, `{"client_ip":null}`},
		{`[{"client_ip":[{}]}]`, `{"client_ip":{}}`},
		{`[{"client_ip":[{"timeout_seconds":null}]}]`, `{"client_ip":{"timeout_seconds":null}}`},
		{`[{"client_ip":[{"timeout_seconds":300}]}]`, `{"client_ip":{"timeout_seconds":300}}`},
	} {
		t.Run(tc.source, func(t *testing.T) {
			converted, diagnostics := serviceStoredAffinity([]byte(tc.source))
			if diagnostics.HasError() {
				t.Fatal(diagnostics)
			}
			current, diagnostics := serviceStoredAffinity([]byte(tc.current))
			if diagnostics.HasError() {
				t.Fatal(diagnostics)
			}
			if !converted.Equal(current) {
				t.Fatalf("historical/current state mismatch: %s != %s", converted, current)
			}
			expanded, diagnostics := expandServiceAffinity(context.Background(), converted, nil)
			if diagnostics.HasError() {
				t.Fatal(diagnostics)
			}
			flattened, diagnostics := flattenServiceAffinity(context.Background(), expanded)
			if diagnostics.HasError() {
				t.Fatal(diagnostics)
			}
			if !flattened.Equal(converted) {
				t.Fatalf("Read conversion does not converge: %s != %s", converted, flattened)
			}
		})
	}
	for _, malformed := range []string{`[{},{}]`, `[null]`, `[1]`, `1`, `"invalid"`, `{"client_ip":[{},{}]}`, `{"client_ip":[null]}`, `{"client_ip":true}`, `{"client_ip":{"timeout_seconds":"300"}}`} {
		t.Run(malformed, func(t *testing.T) {
			_, diagnostics := serviceStoredAffinity([]byte(malformed))
			if !diagnostics.HasError() {
				t.Fatal("malformed historical singleton accepted")
			}
		})
	}
}

func TestServiceAffinityResolveComputedObjects(t *testing.T) {
	ctx := context.Background()
	actual, d := serviceStoredAffinity([]byte(`{"client_ip":{"timeout_seconds":10800}}`))
	if d.HasError() {
		t.Fatal(d)
	}
	configured, d := serviceStoredAffinity([]byte(`{"client_ip":{"timeout_seconds":300}}`))
	if d.HasError() {
		t.Fatal(d)
	}
	for _, tc := range []struct {
		name       string
		plan, want types.Object
	}{
		{"omitted parent", types.ObjectNull(serviceSessionAffinityConfigType.AttrTypes), actual},
		{"unknown parent", types.ObjectUnknown(serviceSessionAffinityConfigType.AttrTypes), actual},
		{"omitted child", types.ObjectValueMust(serviceSessionAffinityConfigType.AttrTypes, map[string]attr.Value{"client_ip": types.ObjectNull(serviceClientIPType.AttrTypes)}), actual},
		{"unknown child", types.ObjectValueMust(serviceSessionAffinityConfigType.AttrTypes, map[string]attr.Value{"client_ip": types.ObjectUnknown(serviceClientIPType.AttrTypes)}), actual},
		{"unknown timeout", types.ObjectValueMust(serviceSessionAffinityConfigType.AttrTypes, map[string]attr.Value{"client_ip": types.ObjectValueMust(serviceClientIPType.AttrTypes, map[string]attr.Value{"timeout_seconds": types.Int64Unknown()})}), actual},
		{"known timeout", configured, configured},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, d := serviceResolveAffinityValue(ctx, tc.plan, actual)
			if d.HasError() || !got.Equal(tc.want) {
				t.Fatalf("resolved=%s want=%s diagnostics=%v", got, tc.want, d)
			}
		})
	}
	// A missing API object must still resolve unknown nested objects to typed null.
	planned := types.ObjectValueMust(serviceSessionAffinityConfigType.AttrTypes, map[string]attr.Value{"client_ip": types.ObjectUnknown(serviceClientIPType.AttrTypes)})
	got, d := serviceResolveAffinityValue(ctx, planned, types.ObjectNull(serviceSessionAffinityConfigType.AttrTypes))
	want := types.ObjectValueMust(serviceSessionAffinityConfigType.AttrTypes, map[string]attr.Value{"client_ip": types.ObjectNull(serviceClientIPType.AttrTypes)})
	if d.HasError() || !got.Equal(want) {
		t.Fatalf("missing API object resolved=%s diagnostics=%v", got, d)
	}
}
