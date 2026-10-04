// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	sdkterraform "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

func serviceProtocolDiagnostics(t *testing.T, diagnostics []*tfprotov6.Diagnostic) {
	t.Helper()
	for _, d := range diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("%s: %s", d.Summary, d.Detail)
		}
	}
}

func TestServiceSingletonSchema(t *testing.T) {
	var response fwresource.SchemaResponse
	serviceRegisteredFrameworkResource(t).Schema(context.Background(), fwresource.SchemaRequest{}, &response)
	if response.Schema.Version != 2 {
		t.Fatalf("schema version = %d, want 2", response.Schema.Version)
	}
	spec := response.Schema.Blocks["spec"].(schema.ListNestedBlock)
	affinity, ok := spec.NestedObject.Attributes["session_affinity_config"].(schema.SingleNestedAttribute)
	if !ok || !affinity.Optional || !affinity.Computed {
		t.Fatal("affinity must be an Optional+Computed single object")
	}
	client, ok := affinity.Attributes["client_ip"].(schema.SingleNestedAttribute)
	if !ok || !client.Optional || !client.Computed {
		t.Fatal("client_ip must be an Optional+Computed single object")
	}
}

func TestServiceReadLegacyNullIdentityNotFound(t *testing.T) {
	ctx := context.Background()
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/v1/namespaces/apps/services/svc-abc" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		http.Error(w, `{"kind":"Status","apiVersion":"v1","status":"Failure","reason":"NotFound","code":404}`, 404)
	}))
	defer api.Close()
	sdk := kubernetes.Provider()
	if diagnostics := sdk.Configure(ctx, sdkterraform.NewResourceConfigRaw(map[string]interface{}{"host": api.URL})); diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	server := providerserver.NewProtocol6(provider.New("test", sdk.Meta))()
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	serviceProtocolDiagnostics(t, schemas.Diagnostics)
	configType := schemas.Provider.ValueType().(tftypes.Object)
	configValues := map[string]tftypes.Value{}
	for name, typ := range configType.AttributeTypes {
		configValues[name] = tftypes.NewValue(typ, nil)
	}
	config, err := tfprotov6.NewDynamicValue(configType, tftypes.NewValue(configType, configValues))
	if err != nil {
		t.Fatal(err)
	}
	configured, err := server.ConfigureProvider(ctx, &tfprotov6.ConfigureProviderRequest{Config: &config})
	if err != nil {
		t.Fatal(err)
	}
	serviceProtocolDiagnostics(t, configured.Diagnostics)
	upgraded, err := server.UpgradeResourceState(ctx, &tfprotov6.UpgradeResourceStateRequest{TypeName: "kubernetes_service_v1", Version: 0, RawState: serviceStoredFixture(t, 0)})
	if err != nil {
		t.Fatal(err)
	}
	serviceProtocolDiagnostics(t, upgraded.Diagnostics)
	read, err := server.ReadResource(ctx, &tfprotov6.ReadResourceRequest{TypeName: "kubernetes_service_v1", CurrentState: upgraded.UpgradedState})
	if err != nil {
		t.Fatal(err)
	}
	serviceProtocolDiagnostics(t, read.Diagnostics)
	state, err := read.NewState.Unmarshal(schemas.ResourceSchemas["kubernetes_service_v1"].ValueType())
	if err != nil {
		t.Fatal(err)
	}
	if !state.IsNull() {
		t.Fatal("404 retained resource state")
	}
	if read.NewIdentity == nil {
		t.Fatal("404 lost identity")
	}
}

// These requests deliberately skip ConfigureProvider and ReadResource: upgrading
// stored state must be sufficient to plan safely while the API is unavailable.
func TestServiceSingletonUpgradeAndMoveWithoutRead(t *testing.T) {
	ctx := context.Background()
	for _, version := range []int64{0, 1} {
		for _, move := range []bool{false, true} {
			for _, affinity := range []string{`null`, `[]`, `[{}]`, `[{"client_ip":[]}]`, `[{"client_ip":[{}]}]`, `[{"client_ip":[{"timeout_seconds":300}]}]`} {
				t.Run(fmt.Sprintf("v%d/move-%t/%s", version, move, affinity), func(t *testing.T) {
					server := providerserver.NewProtocol6(provider.New("test", nil))()
					schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
					if err != nil {
						t.Fatal(err)
					}
					serviceProtocolDiagnostics(t, schemas.Diagnostics)
					typ := schemas.ResourceSchemas["kubernetes_service_v1"].ValueType()
					fixture := serviceStoredFixture(t, version)
					var stored map[string]any
					if err := json.Unmarshal(fixture.JSON, &stored); err != nil {
						t.Fatal(err)
					}
					stored["metadata"].([]any)[0].(map[string]any)["generate_name"] = ""
					spec := stored["spec"].([]any)[0].(map[string]any)

					spec["session_affinity_config"] = json.RawMessage(affinity)
					fixture.JSON, err = json.Marshal(stored)
					if err != nil {
						t.Fatal(err)
					}
					var dynamic *tfprotov6.DynamicValue
					if move {
						result, err := server.MoveResourceState(ctx, &tfprotov6.MoveResourceStateRequest{SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes", SourceTypeName: "kubernetes_service", SourceSchemaVersion: version, SourceState: fixture, TargetTypeName: "kubernetes_service_v1"})
						if err != nil {
							t.Fatal(err)
						}
						serviceProtocolDiagnostics(t, result.Diagnostics)
						dynamic = result.TargetState
						if result.TargetIdentity == nil {
							t.Fatal("move omitted identity")
						}
					} else {
						result, err := server.UpgradeResourceState(ctx, &tfprotov6.UpgradeResourceStateRequest{TypeName: "kubernetes_service_v1", Version: version, RawState: fixture})
						if err != nil {
							t.Fatal(err)
						}
						serviceProtocolDiagnostics(t, result.Diagnostics)
						dynamic = result.UpgradedState
					}
					if dynamic == nil {
						t.Fatal("conversion omitted state")
					}
					prior, err := dynamic.Unmarshal(typ)
					if err != nil {
						t.Fatal(err)
					}
					var schemaResponse fwresource.SchemaResponse
					serviceRegisteredFrameworkResource(t).Schema(ctx, fwresource.SchemaRequest{}, &schemaResponse)
					current := tfsdk.State{Schema: schemaResponse.Schema, Raw: prior}
					for _, tc := range []struct {
						name    string
						change  path.Path
						value   any
						replace bool
					}{
						{name: "unchanged"},
						{name: "omitted-affinity"},
						{name: "metadata-only", change: path.Root("metadata").AtListIndex(0).AtName("labels"), value: map[string]string{"app": "changed"}},
						{name: "immutable-name", change: path.Root("metadata").AtListIndex(0).AtName("name"), value: "changed", replace: true},
						{name: "immutable-cluster-ip", change: path.Root("spec").AtListIndex(0).AtName("cluster_ip"), value: "10.96.0.99", replace: true},
						{name: "immutable-class", change: path.Root("spec").AtListIndex(0).AtName("load_balancer_class"), value: "example.com/other", replace: true},
					} {
						t.Run(tc.name, func(t *testing.T) {
							// Core merges prior values into Optional+Computed fields. The legacy
							// empty generate_name is tested against omitted configuration.
							config := tfsdk.State{Schema: current.Schema, Raw: prior}
							proposed := tfsdk.State{Schema: current.Schema, Raw: prior}
							emptyName := path.Root("metadata").AtListIndex(0).AtName("generate_name")
							if d := config.SetAttribute(ctx, emptyName, types.StringNull()); d.HasError() {
								t.Fatal(d)
							}
							if d := proposed.SetAttribute(ctx, emptyName, types.StringNull()); d.HasError() {
								t.Fatal(d)
							}
							config.Raw, err = tftypes.Transform(config.Raw, func(p *tftypes.AttributePath, v tftypes.Value) (tftypes.Value, error) {
								steps := p.Steps()
								computed := len(steps) == 1 && (steps[0] == tftypes.AttributeName("id") || steps[0] == tftypes.AttributeName("status"))
								if len(steps) == 3 && steps[0] == tftypes.AttributeName("metadata") {
									computed = steps[2] == tftypes.AttributeName("uid") || steps[2] == tftypes.AttributeName("generation") || steps[2] == tftypes.AttributeName("resource_version")
								}
								if computed {
									return tftypes.NewValue(v.Type(), nil), nil
								}
								return v, nil
							})
							if err != nil {
								t.Fatal(err)
							}

							if tc.name == "omitted-affinity" {
								var affinity types.Object
								at := path.Root("spec").AtListIndex(0).AtName("session_affinity_config")
								if d := current.GetAttribute(ctx, at, &affinity); d.HasError() {
									t.Fatal(d)
								}
								if d := config.SetAttribute(ctx, at, types.ObjectNull(affinity.AttributeTypes(ctx))); d.HasError() {
									t.Fatal(d)
								}
							}

							if tc.value != nil {
								if d := config.SetAttribute(ctx, tc.change, tc.value); d.HasError() {
									t.Fatal(d)
								}
								if d := proposed.SetAttribute(ctx, tc.change, tc.value); d.HasError() {
									t.Fatal(d)
								}
							}
							cfg, err := tfprotov6.NewDynamicValue(typ, config.Raw)
							if err != nil {
								t.Fatal(err)
							}
							proposal, err := tfprotov6.NewDynamicValue(typ, proposed.Raw)
							if err != nil {
								t.Fatal(err)
							}
							planned, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{TypeName: "kubernetes_service_v1", PriorState: dynamic, Config: &cfg, ProposedNewState: &proposal})
							if err != nil {
								t.Fatal(err)
							}
							serviceProtocolDiagnostics(t, planned.Diagnostics)
							if (len(planned.RequiresReplace) != 0) != tc.replace {
								t.Fatalf("replacement paths = %v, want replacement %t", planned.RequiresReplace, tc.replace)
							}
							if tc.replace {
								parent, field := "metadata", "name"
								if tc.name == "immutable-cluster-ip" {
									parent, field = "spec", "cluster_ip"
								}
								if tc.name == "immutable-class" {
									parent, field = "spec", "load_balancer_class"
								}
								want := tftypes.NewAttributePath().WithAttributeName(parent).WithElementKeyInt(0).WithAttributeName(field)
								if len(planned.RequiresReplace) != 1 || !planned.RequiresReplace[0].Equal(want) {
									t.Fatalf("replacement paths = %v, want %s", planned.RequiresReplace, want)
								}
							}
							if tc.name == "unchanged" || tc.name == "omitted-affinity" {
								got, err := planned.PlannedState.Unmarshal(typ)
								if err != nil {
									t.Fatal(err)
								}
								if !got.Equal(prior) {
									t.Fatalf("unchanged plan differs from upgraded state:\n%s\n%s", prior, got)
								}
							}
						})
					}
				})
			}
		}
	}
}

func TestServiceNameAndGenerateNameConfiguration(t *testing.T) {
	ctx := context.Background()
	server, err := mux.MuxServer(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"kubernetes_service", "kubernetes_service_v1"} {
		t.Run(name, func(t *testing.T) {
			typ := schemas.ResourceSchemas[name].ValueType()
			raw := &tfprotov6.RawState{JSON: []byte(`{"metadata":[{"name":"named","namespace":"default","generate_name":"prefix-"}],"spec":[{"port":[{"port":80}]}]}`)}
			value, err := raw.Unmarshal(typ)
			if err != nil {
				t.Fatal(err)
			}
			config, err := tfprotov6.NewDynamicValue(typ, value)
			if err != nil {
				t.Fatal(err)
			}
			result, err := server.ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{TypeName: name, Config: &config})
			if err != nil {
				t.Fatal(err)
			}
			serviceProtocolDiagnostics(t, result.Diagnostics)
		})
	}
}
