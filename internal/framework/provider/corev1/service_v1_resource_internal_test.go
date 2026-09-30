// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
	affinityPath := path.Root("spec").AtListIndex(0).AtName("session_affinity_config")
	clientPath := affinityPath.AtListIndex(0).AtName("client_ip")
	clientConfig := func(client any) []any {
		return []any{map[string]any{"client_ip": client}}
	}
	for _, tc := range []struct {
		name      string
		affinity  any
		config    any
		omit      bool
		errorPath *path.Path
	}{
		{name: "ClientIP empty outer list", affinity: "ClientIP", config: []any{}, errorPath: &affinityPath},
		{name: "ClientIP empty client list", affinity: "ClientIP", config: clientConfig([]any{}), errorPath: &clientPath},
		{name: "ClientIP omitted outer", affinity: "ClientIP", omit: true},
		{name: "ClientIP null outer", affinity: "ClientIP"},
		{name: "ClientIP default outer object", affinity: "ClientIP", config: []any{map[string]any{}}},
		{name: "ClientIP null client list", affinity: "ClientIP", config: clientConfig(nil)},
		{name: "ClientIP default client object", affinity: "ClientIP", config: clientConfig([]any{map[string]any{}})},
		{name: "ClientIP explicit timeout", affinity: "ClientIP", config: clientConfig([]any{map[string]any{"timeout_seconds": 300}})},
		{name: "None empty outer list", affinity: "None", config: []any{}},
		{name: "None empty client list", affinity: "None", config: clientConfig([]any{})},
		{name: "unknown affinity with empty outer", affinity: tftypes.UnknownValue, config: []any{}},
		{name: "unknown affinity with empty client", affinity: tftypes.UnknownValue, config: clientConfig([]any{})},
		{name: "null affinity defers", config: []any{}},
		{name: "unknown outer list", affinity: "ClientIP", config: tftypes.UnknownValue},
		{name: "unknown outer object", affinity: "ClientIP", config: []any{tftypes.UnknownValue}},
		{name: "unknown client list", affinity: "ClientIP", config: clientConfig(tftypes.UnknownValue)},
		{name: "unknown client object", affinity: "ClientIP", config: clientConfig([]any{tftypes.UnknownValue})},
		{name: "unknown timeout", affinity: "ClientIP", config: clientConfig([]any{map[string]any{"timeout_seconds": tftypes.UnknownValue}})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := map[string]any{"session_affinity": tc.affinity}
			if !tc.omit {
				spec["session_affinity_config"] = tc.config
			}
			config := serviceAffinityValidationConfig(t, []any{spec})
			var response resource.ValidateConfigResponse
			(&ServiceV1{}).ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: config}, &response)
			serviceAffinityCheckDiagnostics(t, response.Diagnostics, tc.errorPath)
		})
	}
	for _, tc := range []struct {
		name string
		spec any
	}{
		{"unknown spec list", tftypes.UnknownValue},
		{"unknown spec object", []any{tftypes.UnknownValue}},
		{"null spec list", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := serviceAffinityValidationConfig(t, tc.spec)
			var response resource.ValidateConfigResponse
			(&ServiceV1{}).ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: config}, &response)
			serviceAffinityCheckDiagnostics(t, response.Diagnostics, nil)
		})
	}
}

func TestServiceAffinityLateKnownEmpty(t *testing.T) {
	affinityPath := path.Root("spec").AtListIndex(0).AtName("session_affinity_config")
	clientPath := affinityPath.AtListIndex(0).AtName("client_ip")
	for _, tc := range []struct {
		name          string
		before, after map[string]any
		errorPath     path.Path
	}{
		{
			name:      "outer list resolves empty",
			before:    map[string]any{"session_affinity": "ClientIP", "session_affinity_config": tftypes.UnknownValue},
			after:     map[string]any{"session_affinity": "ClientIP", "session_affinity_config": []any{}},
			errorPath: affinityPath,
		},
		{
			name:      "client list resolves empty",
			before:    map[string]any{"session_affinity": "ClientIP", "session_affinity_config": []any{map[string]any{"client_ip": tftypes.UnknownValue}}},
			after:     map[string]any{"session_affinity": "ClientIP", "session_affinity_config": []any{map[string]any{"client_ip": []any{}}}},
			errorPath: clientPath,
		},
		{
			name:      "affinity resolves ClientIP with empty outer",
			before:    map[string]any{"session_affinity": tftypes.UnknownValue, "session_affinity_config": []any{}},
			after:     map[string]any{"session_affinity": "ClientIP", "session_affinity_config": []any{}},
			errorPath: affinityPath,
		},
		{
			name:      "affinity resolves ClientIP with empty client",
			before:    map[string]any{"session_affinity": tftypes.UnknownValue, "session_affinity_config": []any{map[string]any{"client_ip": []any{}}}},
			after:     map[string]any{"session_affinity": "ClientIP", "session_affinity_config": []any{map[string]any{"client_ip": []any{}}}},
			errorPath: clientPath,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			r := &ServiceV1{SDKv2Meta: func() any {
				t.Error("invalid affinity reached provider client configuration")
				return nil
			}}
			var validation resource.ValidateConfigResponse
			r.ValidateConfig(ctx, resource.ValidateConfigRequest{
				Config: serviceAffinityValidationConfig(t, []any{tc.before}),
			}, &validation)
			serviceAffinityCheckDiagnostics(t, validation.Diagnostics, nil)

			resolved := serviceAffinityValidationConfig(t, []any{tc.after})
			plan := tfsdk.Plan{Schema: resolved.Schema, Raw: resolved.Raw}
			t.Run("Create", func(t *testing.T) {
				response := resource.CreateResponse{State: tfsdk.State{Schema: resolved.Schema}}
				r.Create(ctx, resource.CreateRequest{Plan: plan}, &response)
				serviceAffinityCheckDiagnostics(t, response.Diagnostics, &tc.errorPath)
				if !response.State.Raw.IsNull() {
					t.Fatal("rejected create produced resource state")
				}
			})
			t.Run("Update", func(t *testing.T) {
				prior := serviceAffinityValidationConfig(t, []any{map[string]any{
					"session_affinity": "ClientIP",
					"session_affinity_config": []any{map[string]any{
						"client_ip": []any{map[string]any{"timeout_seconds": 300}},
					}},
				}})
				state := tfsdk.State{Schema: prior.Schema, Raw: prior.Raw}
				if diagnostics := state.SetAttribute(ctx, path.Root("id"), "default/affinity-validation"); diagnostics.HasError() {
					t.Fatal(diagnostics)
				}
				response := resource.UpdateResponse{State: state}
				r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, &response)
				serviceAffinityCheckDiagnostics(t, response.Diagnostics, &tc.errorPath)
				if !response.State.Raw.Equal(state.Raw) {
					t.Fatal("rejected update changed prior resource state")
				}
			})
		})
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
		{name: "default config", config: []any{map[string]any{}}, wantConfig: true},
		{name: "default client", config: []any{map[string]any{"client_ip": []any{map[string]any{}}}}, wantConfig: true, wantClient: true},
		{name: "unknown timeout", config: []any{map[string]any{"client_ip": []any{map[string]any{"timeout_seconds": tftypes.UnknownValue}}}}, wantConfig: true, wantClient: true},
		{name: "explicit timeout", config: []any{map[string]any{"client_ip": []any{map[string]any{"timeout_seconds": 300}}}}, wantConfig: true, wantClient: true, wantTimeout: 300},
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
			serviceAffinityCheckDiagnostics(t, diagnostics, nil)
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

func serviceAffinityCheckDiagnostics(t *testing.T, diagnostics diag.Diagnostics, expectedPath *path.Path) {
	t.Helper()
	if expectedPath == nil {
		if len(diagnostics) != 0 {
			t.Fatalf("unexpected diagnostics: %v", diagnostics)
		}
		return
	}
	if len(diagnostics) != 1 || diagnostics[0].Severity() != diag.SeverityError {
		t.Fatalf("expected one affinity error at %s, got: %v", expectedPath, diagnostics)
	}
	diagnostic := diagnostics[0]
	if diagnostic.Summary() != "Empty ClientIP configuration" {
		t.Fatalf("unexpected diagnostic: %s: %s", diagnostic.Summary(), diagnostic.Detail())
	}
	withPath, ok := diagnostic.(diag.DiagnosticWithPath)
	if !ok || !withPath.Path().Equal(*expectedPath) {
		t.Fatalf("expected diagnostic at %s, got: %v", expectedPath, diagnostic)
	}
	for _, guidance := range []string{"omit", "null", "[{}]"} {
		if !strings.Contains(diagnostic.Detail(), guidance) {
			t.Errorf("diagnostic lacks %q remediation: %s", guidance, diagnostic.Detail())
		}
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
