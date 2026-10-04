// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package networkingv1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func networkingUpgradeFixture(t *testing.T, kind string) *tfprotov6.RawState {
	t.Helper()
	metadata := map[string]any{"name": "example", "uid": "kept-uid", "resource_version": "10", "generation": 1, "generate_name": "", "labels": nil, "annotations": nil}
	id := "example"
	if kind != "ingress_class" {
		metadata["namespace"] = "default"
		id = "default/example"
	}
	var spec map[string]any
	out := map[string]any{"id": id, "metadata": []any{metadata}}
	switch kind {
	case "ingress_class":
		spec = map[string]any{"controller": "example.com/controller", "parameters": []any{map[string]any{"name": "params", "kind": "Params", "api_group": "example.com", "namespace": "", "scope": "Cluster"}}}
	case "ingress":
		backend := func() []any {
			return []any{map[string]any{"resource": []any{}, "service": []any{map[string]any{"name": "app", "port": []any{map[string]any{"name": "", "number": 80}}}}}}
		}
		spec = map[string]any{"ingress_class_name": "example", "default_backend": backend(), "rule": []any{map[string]any{"host": "example.com", "http": []any{map[string]any{"path": []any{map[string]any{"path": "/", "path_type": "Prefix", "backend": backend()}}}}}}, "tls": []any{}}
		out["wait_for_load_balancer"] = false
		out["timeouts"] = map[string]any{"create": "17m", "delete": "19m"}
		out["status"] = []any{map[string]any{"load_balancer": []any{map[string]any{"ingress": []any{}}}}}
	case "network_policy":
		selector := func() []any { return []any{map[string]any{"match_labels": nil, "match_expressions": []any{}}} }
		peer := func() map[string]any {
			return map[string]any{"ip_block": []any{}, "namespace_selector": selector(), "pod_selector": []any{}}
		}
		spec = map[string]any{"pod_selector": selector(), "policy_types": []any{"Ingress", "Egress"}, "ingress": []any{map[string]any{"ports": []any{}, "from": []any{peer()}}}, "egress": []any{map[string]any{"ports": []any{}, "to": []any{peer()}}}}
	default:
		t.Fatal(kind)
	}
	out["spec"] = []any{spec}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	return &tfprotov6.RawState{JSON: data}
}

func networkingUpgradeProvider(kind string, meta func() any) provider.Provider {
	switch kind {
	case "ingress":
		return ingressInternalProvider{meta: meta}
	case "ingress_class":
		return ingressClassTestProvider{meta: meta}
	default:
		return networkPolicyTestProvider{meta: meta}
	}
}

func networkingUpgradeResource(kind string) resource.Resource {
	switch kind {
	case "ingress":
		return NewIngressV1()
	case "ingress_class":
		return NewIngressClassV1()
	default:
		return NewNetworkPolicyV1()
	}
}

// UpgradeResourceState and MoveResourceState must independently reach the final
// schema. Deliberately skip Read before planning so refresh cannot hide a bad conversion.
func TestNetworkingSingletonUpgradeAndMoveWithoutRead(t *testing.T) {
	ctx := context.Background()
	for _, kind := range []string{"ingress", "ingress_class", "network_policy"} {
		t.Run(kind, func(t *testing.T) {
			server := providerserver.NewProtocol6(networkingUpgradeProvider(kind, nil))()
			typeName := "kubernetes_" + kind + "_v1"
			schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
			if err != nil {
				t.Fatal(err)
			}
			ingressClassCheckDiagnostics(t, schemas.Diagnostics)
			schema := schemas.ResourceSchemas[typeName]
			if schema.Version != 1 {
				t.Fatalf("schema version=%d", schema.Version)
			}
			typ := schema.ValueType()
			raw := networkingUpgradeFixture(t, kind)
			upgraded, err := server.UpgradeResourceState(ctx, &tfprotov6.UpgradeResourceStateRequest{TypeName: typeName, Version: 0, RawState: raw})
			if err != nil {
				t.Fatal(err)
			}
			ingressClassCheckDiagnostics(t, upgraded.Diagnostics)
			expected, err := upgraded.UpgradedState.Unmarshal(typ)
			if err != nil {
				t.Fatal(err)
			}
			routes := map[string]*tfprotov6.DynamicValue{"same_type": upgraded.UpgradedState}
			if kind != "ingress" {
				moved, err := server.MoveResourceState(ctx, &tfprotov6.MoveResourceStateRequest{SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes", SourceTypeName: "kubernetes_" + kind, SourceSchemaVersion: 0, SourceState: raw, TargetTypeName: typeName})
				if err != nil {
					t.Fatal(err)
				}
				ingressClassCheckDiagnostics(t, moved.Diagnostics)
				routes["alias_move"] = moved.TargetState
			}
			for route, prior := range routes {
				t.Run(route, func(t *testing.T) {
					value, err := prior.Unmarshal(typ)
					if err != nil {
						t.Fatal(err)
					}
					if !value.Equal(expected) {
						t.Fatal("same-type and alias conversion differ")
					}
					config, err := tftypes.Transform(value, func(at *tftypes.AttributePath, v tftypes.Value) (tftypes.Value, error) {
						steps := at.Steps()
						if len(steps) == 0 {
							return v, nil
						}
						last := steps[len(steps)-1]
						if last == tftypes.AttributeName("id") || last == tftypes.AttributeName("status") || last == tftypes.AttributeName("generate_name") || last == tftypes.AttributeName("generation") || last == tftypes.AttributeName("resource_version") || last == tftypes.AttributeName("uid") {
							return tftypes.NewValue(v.Type(), nil), nil
						}
						return v, nil
					})
					if err != nil {
						t.Fatal(err)
					}
					plan, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{TypeName: typeName, Config: ingressInternalDynamic(t, config), PriorState: prior, ProposedNewState: prior})
					if err != nil {
						t.Fatal(err)
					}
					ingressClassCheckDiagnostics(t, plan.Diagnostics)
					if len(plan.RequiresReplace) != 0 {
						t.Fatalf("unchanged migration replaces %v", plan.RequiresReplace)
					}
					planned, err := plan.PlannedState.Unmarshal(typ)
					if err != nil {
						t.Fatal(err)
					}
					if !planned.Equal(value) {
						t.Fatalf("unchanged migration changed state:\n%s\n%s", value, planned)
					}
					changed, err := tftypes.Transform(config, func(at *tftypes.AttributePath, v tftypes.Value) (tftypes.Value, error) {
						steps := at.Steps()
						if len(steps) == 3 && steps[0] == tftypes.AttributeName("metadata") && steps[2] == tftypes.AttributeName("name") {
							return tftypes.NewValue(tftypes.String, "changed"), nil
						}
						return v, nil
					})
					if err != nil {
						t.Fatal(err)
					}
					// A real name edit must retain its replacement contract.
					renamed, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{TypeName: typeName, Config: ingressInternalDynamic(t, changed), PriorState: prior, ProposedNewState: ingressInternalDynamic(t, changed)})
					if err != nil {
						t.Fatal(err)
					}
					ingressClassCheckDiagnostics(t, renamed.Diagnostics)
					if len(renamed.RequiresReplace) == 0 {
						t.Fatal("name edit lost required replacement")
					}
				})
			}
		})
	}
}

func TestNetworkingSingletonConversionContract(t *testing.T) {
	for _, kind := range []string{"ingress", "ingress_class", "network_policy"} {
		t.Run(kind, func(t *testing.T) {
			var object map[string]any
			if err := json.Unmarshal(networkingUpgradeFixture(t, kind).JSON, &object); err != nil {
				t.Fatal(err)
			}
			beforeID := object["id"]
			beforeTimeouts, _ := json.Marshal(object["timeouts"])
			if err := upgradeNetworkingSingletons(object, kind); err != nil {
				t.Fatal(err)
			}
			first, _ := json.Marshal(object)
			if err := upgradeNetworkingSingletons(object, kind); err != nil {
				t.Fatal(err)
			}
			second, _ := json.Marshal(object)
			if string(first) != string(second) {
				t.Fatal("converter is not idempotent")
			}
			if object["id"] != beforeID {
				t.Fatal("ID changed")
			}
			afterTimeouts, _ := json.Marshal(object["timeouts"])
			if string(afterTimeouts) != string(beforeTimeouts) {
				t.Fatal("configured timeouts changed")
			}
			spec := object["spec"].([]any)[0].(map[string]any)
			switch kind {
			case "ingress_class":
				if _, ok := spec["parameters"].(map[string]any); !ok {
					t.Fatal("parameters is not an object")
				}
			case "ingress":
				backend := spec["default_backend"].(map[string]any)
				service := backend["service"].(map[string]any)
				if _, ok := service["port"].(map[string]any); !ok {
					t.Fatal("port is not an object")
				}
				if backend["resource"] != nil {
					t.Fatal("omitted resource must be null")
				}
			case "network_policy":
				peer := spec["ingress"].([]any)[0].(map[string]any)["from"].([]any)[0].(map[string]any)
				if peer["pod_selector"] != nil {
					t.Fatal("omitted selector must be null")
				}
				if _, ok := peer["namespace_selector"].(map[string]any); !ok {
					t.Fatal("present empty namespace selector must remain an object")
				}
			}
		})
	}
	for _, tc := range []struct {
		name                string
		value               any
		required, wantError bool
	}{{"absent", nil, false, false}, {"empty-list", []any{}, false, false}, {"empty-object", map[string]any{}, false, false}, {"one-empty", []any{map[string]any{}}, false, false}, {"multiple", []any{map[string]any{}, map[string]any{}}, false, true}, {"scalar", true, false, true}, {"null-element", []any{nil}, true, true}, {"optional-null-element", []any{nil}, false, true}, {"missing-required", nil, true, true}, {"empty-required", []any{}, true, true}} {
		t.Run(tc.name, func(t *testing.T) {
			parent := map[string]any{"selector": tc.value}
			_, err := networkingSingleton(parent, "selector", "spec[0]", tc.required)
			if (err != nil) != tc.wantError {
				t.Fatalf("conversion error=%v", err)
			}
			if err != nil && !strings.Contains(err.Error(), "spec[0].selector") {
				t.Fatalf("missing path: %v", err)
			}
		})
	}
}

func TestNetworkingLegacyIdentityMissingReadProtocol(t *testing.T) {
	ctx := context.Background()
	for _, kind := range []string{"ingress", "ingress_class", "network_policy"} {
		t.Run(kind, func(t *testing.T) {
			calls := 0
			api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != http.MethodGet {
					t.Errorf("unexpected mutation: %s", r.Method)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprint(w, `{"apiVersion":"v1","kind":"Status","reason":"NotFound","code":404}`)
			}))
			defer api.Close()
			client, err := kubeclient.NewForConfig(&rest.Config{Host: api.URL})
			if err != nil {
				t.Fatal(err)
			}
			server := providerserver.NewProtocol6(networkingUpgradeProvider(kind, func() any { return ingressClassTestClientsets{client: client} }))()
			typeName := "kubernetes_" + kind + "_v1"
			schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
			if err != nil {
				t.Fatal(err)
			}
			ingressClassCheckDiagnostics(t, schemas.Diagnostics)
			conf := tftypes.NewValue(schemas.Provider.ValueType(), map[string]tftypes.Value{})
			configured, err := server.ConfigureProvider(ctx, &tfprotov6.ConfigureProviderRequest{Config: ingressInternalDynamic(t, conf)})
			if err != nil {
				t.Fatal(err)
			}
			ingressClassCheckDiagnostics(t, configured.Diagnostics)
			upgraded, err := server.UpgradeResourceState(ctx, &tfprotov6.UpgradeResourceStateRequest{TypeName: typeName, Version: 0, RawState: networkingUpgradeFixture(t, kind)})
			if err != nil {
				t.Fatal(err)
			}
			ingressClassCheckDiagnostics(t, upgraded.Diagnostics)
			read, err := server.ReadResource(ctx, &tfprotov6.ReadResourceRequest{TypeName: typeName, CurrentState: upgraded.UpgradedState})
			if err != nil {
				t.Fatal(err)
			}
			ingressClassCheckDiagnostics(t, read.Diagnostics)
			state, err := read.NewState.Unmarshal(schemas.ResourceSchemas[typeName].ValueType())
			if err != nil || !state.IsNull() {
				t.Fatalf("state not removed: %s %v", state, err)
			}
			if calls != 1 {
				t.Fatalf("GET calls=%d", calls)
			}
		})
	}
}

// The API expansion proves empty namespace selectors and absent pod selectors
// retain different meanings, and the flattener must converge with upgraded state.
func TestNetworkingSingletonReadConvergence(t *testing.T) {
	ctx := context.Background()
	for _, kind := range []string{"ingress", "ingress_class", "network_policy"} {
		t.Run(kind, func(t *testing.T) {
			r := networkingUpgradeResource(kind)
			var schema resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schema)
			value, _, _, err := decodeNetworkingState(ctx, networkingUpgradeFixture(t, kind), schema.Schema.Type().TerraformType(ctx), kind)
			if err != nil {
				t.Fatal(err)
			}
			state := tfsdk.State{Schema: schema.Schema, Raw: value}
			switch kind {
			case "ingress_class":
				var model IngressClassV1Model
				if d := state.Get(ctx, &model); d.HasError() {
					t.Fatal(d)
				}
				flat := flattenIngressClassSpec(expandIngressClassSpec(model.Spec[0]))
				if !reflect.DeepEqual(model.Spec, flat) {
					t.Fatalf("upgrade/read differ: %#v %#v", model.Spec, flat)
				}
			case "ingress":
				var model IngressV1Model
				if d := state.Get(ctx, &model); d.HasError() {
					t.Fatal(d)
				}
				api, d := ingressExpandSpec(ctx, model.Spec)
				if d.HasError() {
					t.Fatal(d)
				}
				flat, d := ingressFlattenSpec(ctx, api, model.Spec)
				if d.HasError() {
					t.Fatal(d)
				}
				if !reflect.DeepEqual(model.Spec, flat) {
					t.Fatalf("upgrade/read differ: %#v %#v", model.Spec, flat)
				}
			case "network_policy":
				var model NetworkPolicyV1Model
				if d := state.Get(ctx, &model); d.HasError() {
					t.Fatal(d)
				}
				api, d := networkPolicyExpandSpec(ctx, model.Spec)
				if d.HasError() {
					t.Fatal(d)
				}
				peer := api.Ingress[0].From[0]
				if peer.NamespaceSelector == nil || peer.PodSelector != nil || !reflect.DeepEqual(*peer.NamespaceSelector, metav1.LabelSelector{}) {
					t.Fatalf("selector meaning changed: %#v", peer)
				}
				flat, d := networkPolicyFlattenSpec(ctx, api, model.Spec)
				if d.HasError() {
					t.Fatal(d)
				}
				if !reflect.DeepEqual(model.Spec, flat) {
					t.Fatalf("upgrade/read differ: %#v %#v", model.Spec, flat)
				}
			}
		})
	}
}

func networkingUpgradeConfig(t *testing.T, state tftypes.Value) tftypes.Value {
	t.Helper()
	config, err := tftypes.Transform(state, func(at *tftypes.AttributePath, v tftypes.Value) (tftypes.Value, error) {
		steps := at.Steps()
		if len(steps) == 0 {
			return v, nil
		}
		switch steps[len(steps)-1] {
		case tftypes.AttributeName("id"), tftypes.AttributeName("status"), tftypes.AttributeName("generate_name"), tftypes.AttributeName("generation"), tftypes.AttributeName("resource_version"), tftypes.AttributeName("uid"):
			return tftypes.NewValue(v.Type(), nil), nil
		}
		return v, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return config
}

func TestNetworkingSingletonUnknownPlan(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct{ kind, field string }{{"ingress", "default_backend"}, {"ingress", "http"}, {"ingress_class", "parameters"}, {"network_policy", "pod_selector"}, {"network_policy", "namespace_selector"}} {
		t.Run(tc.kind+"/"+tc.field, func(t *testing.T) {
			var s resource.SchemaResponse
			networkingUpgradeResource(tc.kind).Schema(ctx, resource.SchemaRequest{}, &s)
			state, _, _, err := decodeNetworkingState(ctx, networkingUpgradeFixture(t, tc.kind), s.Schema.Type().TerraformType(ctx), tc.kind)
			if err != nil {
				t.Fatal(err)
			}
			config, err := tftypes.Transform(networkingUpgradeConfig(t, state), func(at *tftypes.AttributePath, v tftypes.Value) (tftypes.Value, error) {
				steps := at.Steps()
				if len(steps) > 0 && steps[len(steps)-1] == tftypes.AttributeName(tc.field) {
					return tftypes.NewValue(v.Type(), tftypes.UnknownValue), nil
				}
				return v, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			server := providerserver.NewProtocol6(networkingUpgradeProvider(tc.kind, nil))()
			typeName := "kubernetes_" + tc.kind + "_v1"
			validated, err := server.ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{TypeName: typeName, Config: ingressInternalDynamic(t, config)})
			if err != nil {
				t.Fatal(err)
			}
			ingressClassCheckDiagnostics(t, validated.Diagnostics)
			plan, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{TypeName: typeName, Config: ingressInternalDynamic(t, config), PriorState: ingressInternalDynamic(t, tftypes.NewValue(state.Type(), nil)), ProposedNewState: ingressInternalDynamic(t, config)})
			if err != nil {
				t.Fatal(err)
			}
			ingressClassCheckDiagnostics(t, plan.Diagnostics)
		})
	}
}

func TestNetworkingNamespacedNameAndGenerateName(t *testing.T) {
	ctx := context.Background()
	for _, kind := range []string{"ingress", "network_policy"} {
		t.Run(kind, func(t *testing.T) {
			var s resource.SchemaResponse
			networkingUpgradeResource(kind).Schema(ctx, resource.SchemaRequest{}, &s)
			state, _, _, err := decodeNetworkingState(ctx, networkingUpgradeFixture(t, kind), s.Schema.Type().TerraformType(ctx), kind)
			if err != nil {
				t.Fatal(err)
			}
			config, err := tftypes.Transform(networkingUpgradeConfig(t, state), func(at *tftypes.AttributePath, v tftypes.Value) (tftypes.Value, error) {
				steps := at.Steps()
				if len(steps) == 3 && steps[0] == tftypes.AttributeName("metadata") && steps[2] == tftypes.AttributeName("generate_name") {
					return tftypes.NewValue(tftypes.String, "ignored-prefix-"), nil
				}
				return v, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			server := providerserver.NewProtocol6(networkingUpgradeProvider(kind, nil))()
			result, err := server.ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{TypeName: "kubernetes_" + kind + "_v1", Config: ingressInternalDynamic(t, config)})
			if err != nil {
				t.Fatal(err)
			}
			ingressClassCheckDiagnostics(t, result.Diagnostics)
			for _, d := range result.Diagnostics {
				if d.Severity == tfprotov6.DiagnosticSeverityWarning && strings.Contains(d.Summary, "generate_name is ignored") {
					return
				}
			}
			t.Fatal("missing ignored generate_name warning")
		})
	}
}
