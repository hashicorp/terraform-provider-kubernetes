// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"
	"encoding/json"
	"maps"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// The Framework checks identity after Read returns, including when Read removes
// missing state. Calling the resource method directly misses this regression.
func TestServiceAccountLegacyMissingReadProtocol(t *testing.T) {
	ctx := context.Background()
	for _, kind := range serviceAccountTypes {
		t.Run(kind.name, func(t *testing.T) {
			api := newServiceAccountAPI(t, kind.adopt)
			api.object = nil
			api.setPhase("migration")
			server, schemas := serviceAccountServer(t)
			providerType := schemas.Provider.ValueType()
			config := serviceAccountDynamic(t, serviceAccountValue(providerType, map[string]tftypes.Value{
				"host": tftypes.NewValue(tftypes.String, api.server.URL),
			}))
			configured, err := server.ConfigureProvider(ctx, &tfprotov6.ConfigureProviderRequest{Config: config})
			if err != nil || serviceAccountDiagnosticError(configured.Diagnostics) {
				t.Fatalf("configure: %v, %#v", err, configured)
			}
			name := "account"
			if kind.adopt {
				name = "default"
			}
			typ := schemas.ResourceSchemas[kind.name].ValueType()
			prior := serviceAccountLegacyState(t, typ, name, false)
			for _, withIdentity := range []bool{false, true} {
				t.Run(map[bool]string{false: "first-pre-identity-refresh", true: "existing-identity"}[withIdentity], func(t *testing.T) {
					var identity *tfprotov6.ResourceIdentityData
					if withIdentity {
						identities, err := server.GetResourceIdentitySchemas(ctx, &tfprotov6.GetResourceIdentitySchemasRequest{})
						if err != nil || serviceAccountDiagnosticError(identities.Diagnostics) {
							t.Fatalf("identity schemas: %v, %#v", err, identities)
						}
						identity = &tfprotov6.ResourceIdentityData{IdentityData: serviceAccountDynamic(t, serviceAccountValue(identities.IdentitySchemas[kind.name].ValueType(), map[string]tftypes.Value{
							"api_version": tftypes.NewValue(tftypes.String, "v1"), "kind": tftypes.NewValue(tftypes.String, "ServiceAccount"),
							"namespace": tftypes.NewValue(tftypes.String, serviceAccountCLINamespace), "name": tftypes.NewValue(tftypes.String, name),
						}))}
					}
					read, err := server.ReadResource(ctx, &tfprotov6.ReadResourceRequest{
						TypeName: kind.name, CurrentState: serviceAccountDynamic(t, prior), CurrentIdentity: identity,
					})
					if err != nil || serviceAccountDiagnosticError(read.Diagnostics) || read.NewState == nil {
						for _, diagnostic := range read.Diagnostics {
							t.Logf("%s: %s", diagnostic.Summary, diagnostic.Detail)
						}
						t.Fatalf("missing resource read: %v, %#v", err, read)
					}
					state, err := read.NewState.Unmarshal(typ)
					if err != nil || !state.IsNull() {
						t.Fatalf("404 must remove state: %v, %s", err, state)
					}
				})
			}
			api.mu.Lock()
			defer api.mu.Unlock()
			if len(api.calls["migration"]) != 1 || api.calls["migration"]["GET /api/v1/namespaces/"+serviceAccountCLINamespace+"/serviceaccounts/"+name] != 2 {
				t.Fatalf("missing Read must only GET the existing identifier: %v", api.calls)
			}
		})
	}
}

func serviceAccountDynamic(t *testing.T, value tftypes.Value) *tfprotov6.DynamicValue {
	t.Helper()
	dynamic, err := tfprotov6.NewDynamicValue(value.Type(), value)
	if err != nil {
		t.Fatal(err)
	}
	return &dynamic
}

func serviceAccountLegacyState(t *testing.T, typ tftypes.Type, name string, populated bool) tftypes.Value {
	t.Helper()
	value, err := serviceAccountLegacyRawState(t, typ, name, populated).Unmarshal(typ)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func serviceAccountLegacyRawState(t *testing.T, typ tftypes.Type, name string, populated bool) *tfprotov6.RawState {
	t.Helper()
	metadata := map[string]any{
		"name": name, "namespace": serviceAccountCLINamespace, "uid": "original-uid",
		"resource_version": "100", "generation": 0, "annotations": nil, "labels": nil,
	}
	metadataType := typ.(tftypes.Object).AttributeTypes["metadata"].(tftypes.List).ElementType.(tftypes.Object)
	if _, ok := metadataType.AttributeTypes["generate_name"]; ok {
		metadata["generate_name"] = ""
	}
	state := map[string]any{
		"id": serviceAccountCLINamespace + "/" + name, "metadata": []any{metadata},
		"automount_service_account_token": true, "default_secret_name": "",
		"secret": []any{}, "image_pull_secret": []any{}, "timeouts": nil,
	}
	if populated {
		metadata["annotations"] = map[string]string{"managed": "annotation"}
		metadata["labels"] = map[string]string{"managed": "label"}
		state["secret"] = []any{map[string]string{"name": "user-secret"}}
		state["image_pull_secret"] = []any{map[string]string{"name": "pull-secret"}}
		state["automount_service_account_token"] = false
		state["timeouts"] = map[string]string{"create": "45s"}
	}
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	return &tfprotov6.RawState{JSON: raw}
}

func TestServiceAccountNameAndGenerateNameProtocol(t *testing.T) {
	server, schemas := serviceAccountServer(t)
	typ := schemas.ResourceSchemas["kubernetes_service_account_v1"].ValueType()
	metadataType := typ.(tftypes.Object).AttributeTypes["metadata"].(tftypes.List)
	config := serviceAccountValue(typ, map[string]tftypes.Value{
		"metadata": tftypes.NewValue(metadataType, []tftypes.Value{serviceAccountValue(metadataType.ElementType, map[string]tftypes.Value{
			"name": tftypes.NewValue(tftypes.String, "account"), "generate_name": tftypes.NewValue(tftypes.String, "prefix-"),
		})}),
	})
	response, err := server.ValidateResourceConfig(context.Background(), &tfprotov6.ValidateResourceConfigRequest{
		TypeName: "kubernetes_service_account_v1", Config: serviceAccountDynamic(t, config),
	})
	if err != nil || serviceAccountDiagnosticError(response.Diagnostics) {
		t.Fatalf("SDK-compatible name plus generate_name rejected: %v, %#v", err, response)
	}
	for _, diagnostic := range response.Diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityWarning && strings.Contains(diagnostic.Summary, "generate_name is ignored") {
			return
		}
	}
	t.Fatalf("missing ignored generate_name warning: %#v", response.Diagnostics)
}

// Skip ReadResource deliberately: a refresh can normalize a legacy value before
// the plan modifiers see it and conceal unintended replacement on upgrade.
func TestServiceAccountLegacyPlanWithoutRefresh(t *testing.T) {
	ctx := context.Background()
	server, schemas := serviceAccountServer(t)
	for _, kind := range serviceAccountTypes {
		for _, populated := range []bool{false, true} {
			for _, move := range []bool{false, true} {
				prefix := kind.name + "/" + map[bool]string{false: "minimum", true: "non-default"}[populated] + "/" + map[bool]string{false: "same-type", true: "alias-move"}[move]
				t.Run(prefix, func(t *testing.T) {
					typ := schemas.ResourceSchemas[kind.name].ValueType()
					name := "account"
					if kind.adopt {
						name = "default"
					}
					raw := serviceAccountLegacyRawState(t, typ, name, populated)
					var prior *tfprotov6.DynamicValue
					var identity *tfprotov6.ResourceIdentityData
					if move {
						response, err := server.MoveResourceState(ctx, &tfprotov6.MoveResourceStateRequest{
							SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes", SourceTypeName: kind.alias,
							TargetTypeName: kind.name, SourceSchemaVersion: 0, SourceState: raw,
						})
						if err != nil || serviceAccountDiagnosticError(response.Diagnostics) || response.TargetState == nil || response.TargetIdentity == nil {
							t.Fatalf("move: %v, %#v", err, response)
						}
						prior, identity = response.TargetState, response.TargetIdentity
					} else {
						response, err := server.UpgradeResourceState(ctx, &tfprotov6.UpgradeResourceStateRequest{
							TypeName: kind.name, Version: 0, RawState: raw,
						})
						if err != nil || serviceAccountDiagnosticError(response.Diagnostics) || response.UpgradedState == nil {
							t.Fatalf("upgrade: %v, %#v", err, response)
						}
						prior = response.UpgradedState
					}
					value, err := prior.Unmarshal(typ)
					if err != nil {
						t.Fatal(err)
					}
					want, err := raw.Unmarshal(typ)
					if err != nil || !want.Equal(value) {
						t.Fatalf("conversion lost state: %v\nwant: %s\ngot: %s", err, want, value)
					}
					for _, change := range []string{"unchanged", "labels", "namespace", "name", "prefix-add", "prefix-change", "prefix-remove", "null-prefix"} {
						if kind.adopt && (change == "name" || strings.Contains(change, "prefix")) {
							continue
						}
						t.Run(change, func(t *testing.T) {
							state := value
							if change == "prefix-change" || change == "prefix-remove" {
								state = serviceAccountReplaceMetadata(t, state, "generate_name", tftypes.NewValue(tftypes.String, "old-"))
							} else if change == "null-prefix" {
								state = serviceAccountReplaceMetadata(t, state, "generate_name", tftypes.NewValue(tftypes.String, nil))
							}
							config := serviceAccountPlanConfig(t, state)
							proposal := state
							field, replacement := "", false
							var changed tftypes.Value
							switch change {
							case "labels":
								field, changed = "labels", tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{"changed": tftypes.NewValue(tftypes.String, "yes")})
							case "namespace", "name":
								field, changed, replacement = change, tftypes.NewValue(tftypes.String, "changed"), true
							case "prefix-add", "prefix-change":
								field, changed, replacement = "generate_name", tftypes.NewValue(tftypes.String, "new-"), true
							case "prefix-remove":
								replacement = true
							}
							if field != "" {
								config = serviceAccountReplaceMetadata(t, config, field, changed)
								proposal = serviceAccountReplaceMetadata(t, proposal, field, changed)
							}
							response, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
								TypeName: kind.name, PriorState: serviceAccountDynamic(t, state), PriorIdentity: identity,
								ProposedNewState: serviceAccountDynamic(t, proposal), Config: serviceAccountDynamic(t, config),
							})
							if err != nil || serviceAccountDiagnosticError(response.Diagnostics) || response.PlannedState == nil {
								t.Fatalf("plan: %v, %#v", err, response)
							}
							if (len(response.RequiresReplace) != 0) != replacement {
								t.Fatalf("replacement paths = %v, want replacement=%t", response.RequiresReplace, replacement)
							}
							if replacement {
								if field == "" {
									field = "generate_name"
								}
								wantPath := tftypes.NewAttributePath().WithAttributeName("metadata").WithElementKeyInt(0).WithAttributeName(field)
								if len(response.RequiresReplace) != 1 || !response.RequiresReplace[0].Equal(wantPath) {
									t.Fatalf("replacement paths = %v, want %s", response.RequiresReplace, wantPath)
								}
							}
							planned, err := response.PlannedState.Unmarshal(typ)
							if err != nil {
								t.Fatal(err)
							}
							if (change == "unchanged" || change == "null-prefix") && !planned.Equal(state) {
								t.Fatalf("unchanged migration must retain the complete state:\nprior: %s\nplan: %s", state, planned)
							}
							if identity != nil {
								identities, err := server.GetResourceIdentitySchemas(ctx, &tfprotov6.GetResourceIdentitySchemasRequest{})
								if err != nil || response.PlannedIdentity == nil {
									t.Fatalf("planned identity missing: %v", err)
								}
								identityType := identities.IdentitySchemas[kind.name].ValueType()
								before, err := identity.IdentityData.Unmarshal(identityType)
								if err != nil {
									t.Fatal(err)
								}
								after, err := response.PlannedIdentity.IdentityData.Unmarshal(identityType)
								if err != nil || !before.Equal(after) {
									t.Fatalf("plan changed identity: %v\nbefore: %s\nafter: %s", err, before, after)
								}
							}
						})
					}
				})
			}
		}
	}
}

func serviceAccountReplaceMetadata(t *testing.T, value tftypes.Value, field string, changed tftypes.Value) tftypes.Value {
	t.Helper()
	var root map[string]tftypes.Value
	if err := value.As(&root); err != nil {
		t.Fatal(err)
	}
	root = maps.Clone(root)
	var metadata []tftypes.Value
	if err := root["metadata"].As(&metadata); err != nil {
		t.Fatal(err)
	}
	var fields map[string]tftypes.Value
	if err := metadata[0].As(&fields); err != nil {
		t.Fatal(err)
	}
	fields = maps.Clone(fields)
	fields[field] = changed
	root["metadata"] = tftypes.NewValue(root["metadata"].Type(), []tftypes.Value{tftypes.NewValue(metadata[0].Type(), fields)})
	return tftypes.NewValue(value.Type(), root)
}

func serviceAccountPlanConfig(t *testing.T, prior tftypes.Value) tftypes.Value {
	t.Helper()
	var fields map[string]tftypes.Value
	if err := prior.As(&fields); err != nil {
		t.Fatal(err)
	}
	fields = maps.Clone(fields)
	for _, field := range []string{"id", "default_secret_name"} {
		fields[field] = tftypes.NewValue(fields[field].Type(), nil)
	}
	config := tftypes.NewValue(prior.Type(), fields)
	metadataType := fields["metadata"].Type().(tftypes.List).ElementType.(tftypes.Object)
	for _, field := range []string{"uid", "generation", "resource_version", "generate_name"} {
		if typ, ok := metadataType.AttributeTypes[field]; ok {
			config = serviceAccountReplaceMetadata(t, config, field, tftypes.NewValue(typ, nil))
		}
	}
	return config
}
