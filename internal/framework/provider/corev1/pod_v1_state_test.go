// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	corev1 "k8s.io/api/core/v1"
)

func TestPodV1ImportState(t *testing.T) {
	ctx := context.Background()
	pod := &PodV1{}
	_, identityResp := podV1Schemas(t, pod)
	checkTargetState := func(t *testing.T, state tfsdk.State) {
		t.Helper()
		var target types.List
		if diags := state.GetAttribute(ctx, path.Root("target_state"), &target); diags.HasError() {
			t.Fatal(diags)
		}
		if !target.Equal(types.ListValueMust(types.StringType, nil)) {
			t.Fatalf("target_state = %v, want the empty-list default", target)
		}
	}

	t.Run("id", func(t *testing.T) {
		resp := resource.ImportStateResponse{
			State:    podV1ImportStateSchema(t),
			Identity: &tfsdk.ResourceIdentity{Schema: identityResp.IdentitySchema},
		}
		pod.ImportState(ctx, resource.ImportStateRequest{ID: "team-a/demo"}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}

		var id types.String
		if diags := resp.State.GetAttribute(ctx, path.Root("id"), &id); diags.HasError() {
			t.Fatal(diags)
		}
		if !id.Equal(types.StringValue("team-a/demo")) {
			t.Fatalf("id = %v, want team-a/demo", id)
		}
		assertPodIdentity(t, ctx, resp.Identity, "team-a", "demo")
		checkTargetState(t, resp.State)
	})

	t.Run("identity", func(t *testing.T) {
		importIdentity := &tfsdk.ResourceIdentity{Schema: identityResp.IdentitySchema}
		if diags := importIdentity.Set(ctx, common.NamespacedResourceIdentity{
			ResourceIdentity: common.ResourceIdentity{
				APIVersion: types.StringValue(podAPIVersion),
				Kind:       types.StringValue(podKind),
				Name:       types.StringValue("demo"),
			},
			Namespace: types.StringValue("team-b"),
		}); diags.HasError() {
			t.Fatal(diags)
		}

		resp := resource.ImportStateResponse{
			State:    podV1ImportStateSchema(t),
			Identity: &tfsdk.ResourceIdentity{Schema: identityResp.IdentitySchema},
		}
		pod.ImportState(ctx, resource.ImportStateRequest{Identity: importIdentity}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}

		var id types.String
		if diags := resp.State.GetAttribute(ctx, path.Root("id"), &id); diags.HasError() {
			t.Fatal(diags)
		}
		if !id.Equal(types.StringValue("team-b/demo")) {
			t.Fatalf("id = %v, want team-b/demo", id)
		}
		assertPodIdentity(t, ctx, resp.Identity, "team-b", "demo")
		checkTargetState(t, resp.State)
	})

	t.Run("identity namespace defaults", func(t *testing.T) {
		importIdentity := &tfsdk.ResourceIdentity{Schema: identityResp.IdentitySchema}
		if diags := importIdentity.Set(ctx, common.NamespacedResourceIdentity{
			ResourceIdentity: common.ResourceIdentity{
				APIVersion: types.StringValue(podAPIVersion),
				Kind:       types.StringValue(podKind),
				Name:       types.StringValue("demo-default"),
			},
			Namespace: types.StringNull(),
		}); diags.HasError() {
			t.Fatal(diags)
		}

		resp := resource.ImportStateResponse{
			State:    podV1ImportStateSchema(t),
			Identity: &tfsdk.ResourceIdentity{Schema: identityResp.IdentitySchema},
		}
		pod.ImportState(ctx, resource.ImportStateRequest{Identity: importIdentity}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}

		var id types.String
		if diags := resp.State.GetAttribute(ctx, path.Root("id"), &id); diags.HasError() {
			t.Fatal(diags)
		}
		if !id.Equal(types.StringValue(corev1.NamespaceDefault + "/demo-default")) {
			t.Fatalf("id = %v, want %s/demo-default", id, corev1.NamespaceDefault)
		}
		assertPodIdentity(t, ctx, resp.Identity, corev1.NamespaceDefault, "demo-default")
	})

	t.Run("malformed id", func(t *testing.T) {
		for _, id := range []string{"malformed", "/demo", "team-a/"} {
			resp := resource.ImportStateResponse{
				State:    podV1ImportStateSchema(t),
				Identity: &tfsdk.ResourceIdentity{Schema: identityResp.IdentitySchema},
			}
			pod.ImportState(ctx, resource.ImportStateRequest{ID: id}, &resp)
			if !resp.Diagnostics.HasError() {
				t.Fatalf("expected diagnostics for malformed id %q", id)
			}
		}
	})

	t.Run("identity empty namespace", func(t *testing.T) {
		importIdentity := &tfsdk.ResourceIdentity{Schema: identityResp.IdentitySchema}
		if diags := importIdentity.Set(ctx, common.NamespacedResourceIdentity{
			ResourceIdentity: common.ResourceIdentity{
				APIVersion: types.StringValue(podAPIVersion),
				Kind:       types.StringValue(podKind),
				Name:       types.StringValue("demo"),
			},
			Namespace: types.StringValue(""),
		}); diags.HasError() {
			t.Fatal(diags)
		}
		resp := resource.ImportStateResponse{
			State:    podV1ImportStateSchema(t),
			Identity: &tfsdk.ResourceIdentity{Schema: identityResp.IdentitySchema},
		}
		pod.ImportState(ctx, resource.ImportStateRequest{Identity: importIdentity}, &resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected diagnostics for empty namespace")
		}
	})

	t.Run("identity wrong kind apiVersion", func(t *testing.T) {
		for _, tc := range []common.NamespacedResourceIdentity{
			{
				ResourceIdentity: common.ResourceIdentity{
					APIVersion: types.StringValue("apps/v1"),
					Kind:       types.StringValue(podKind),
					Name:       types.StringValue("demo"),
				},
				Namespace: types.StringValue("team-a"),
			},
			{
				ResourceIdentity: common.ResourceIdentity{
					APIVersion: types.StringValue(podAPIVersion),
					Kind:       types.StringValue("Deployment"),
					Name:       types.StringValue("demo"),
				},
				Namespace: types.StringValue("team-a"),
			},
		} {
			importIdentity := &tfsdk.ResourceIdentity{Schema: identityResp.IdentitySchema}
			if diags := importIdentity.Set(ctx, tc); diags.HasError() {
				t.Fatal(diags)
			}
			resp := resource.ImportStateResponse{
				State:    podV1ImportStateSchema(t),
				Identity: &tfsdk.ResourceIdentity{Schema: identityResp.IdentitySchema},
			}
			pod.ImportState(ctx, resource.ImportStateRequest{Identity: importIdentity}, &resp)
			if !resp.Diagnostics.HasError() {
				t.Fatal("expected diagnostics for identity with invalid api_version/kind")
			}
		}
	})
}

func TestPodV1UpgradeStateV0(t *testing.T) {
	ctx := context.Background()
	pod := &PodV1{}
	schemaResp, _ := podV1Schemas(t, pod)
	upgrader := pod.UpgradeState(ctx)[0]

	source := map[string]any{
		"id": "team-a/demo",
		"metadata": []map[string]any{{
			"name": "demo", "namespace": "team-a", "uid": "uid-1",
			"generation": 3, "resource_version": "rv-1", "generate_name": "demo-",
			"annotations": map[string]any{"a": "b"}, "labels": map[string]any{},
		}},
		"target_state": []any{"Pending"},
		"timeouts":     map[string]any{"create": "5m", "delete": ""},
		"spec": []map[string]any{{
			"container": []map[string]any{
				{
					"name": "work", "image": "busybox",
					"resources": []map[string]any{{
						"limits":   []map[string]any{{"cpu": "20m"}},
						"requests": []map[string]any{{"memory": "16Mi"}},
					}},
				},
			},
			"init_container": []map[string]any{
				{
					"name": "init", "image": "busybox",
					"resources": []map[string]any{{"limits": []map[string]any{{"cpu": "5m"}}}},
				},
			},
		}},
	}
	expected := map[string]any{
		"id": "team-a/demo",
		"metadata": []map[string]any{{
			"name": "demo", "namespace": "team-a", "uid": "uid-1",
			"generation": 3, "resource_version": "rv-1", "generate_name": "demo-",
			"annotations": map[string]any{"a": "b"}, "labels": map[string]any{},
		}},
		"target_state": []any{"Pending"},
		"timeouts":     map[string]any{"create": "5m", "delete": nil},
		"spec": []map[string]any{{
			"container": []map[string]any{
				{
					"name": "work", "image": "busybox",
					"resources": []map[string]any{{
						"limits":   map[string]any{"cpu": "20m"},
						"requests": map[string]any{"memory": "16Mi"},
					}},
				},
			},
			"init_container": []map[string]any{
				{
					"name": "init", "image": "busybox",
					"resources": []map[string]any{{
						"limits":   map[string]any{"cpu": "5m"},
						"requests": map[string]any{},
					}},
				},
			},
		}},
	}

	resp := resource.UpgradeStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	upgrader.StateUpgrader(ctx, resource.UpgradeStateRequest{
		RawState: &tfprotov6.RawState{JSON: mustJSON(t, source)},
	}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}

	want := podV1TerraformValue(t, schemaResp, expected)
	if !resp.State.Raw.Equal(want) {
		t.Fatalf("upgraded state mismatch\nwant: %s\ngot:  %s", want, resp.State.Raw)
	}

	t.Run("timeouts unknown attr", func(t *testing.T) {
		resp := resource.UpgradeStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		upgrader.StateUpgrader(ctx, resource.UpgradeStateRequest{
			RawState: &tfprotov6.RawState{JSON: mustJSON(t, map[string]any{
				"id":       "team-a/demo",
				"metadata": []map[string]any{{"name": "demo", "namespace": "team-a"}},
				"spec":     []map[string]any{{"container": []map[string]any{{"name": "work", "image": "busybox"}}}},
				"timeouts": map[string]any{"create": "5m", "update": "1m"},
			})},
		}, &resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected diagnostics for unsupported timeout attribute")
		}
	})

	t.Run("resources multiplicity rejected", func(t *testing.T) {
		resp := resource.UpgradeStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		upgrader.StateUpgrader(ctx, resource.UpgradeStateRequest{
			RawState: &tfprotov6.RawState{JSON: mustJSON(t, map[string]any{
				"id":       "team-a/demo",
				"metadata": []map[string]any{{"name": "demo", "namespace": "team-a"}},
				"spec": []map[string]any{{
					"container": []map[string]any{{
						"name": "work", "image": "busybox",
						"resources": []map[string]any{
							{"limits": []map[string]any{{"cpu": "20m"}}},
							{"limits": []map[string]any{{"memory": "256Mi"}}},
						},
					}},
				}},
			})},
		}, &resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected diagnostics for resources list multiplicity")
		}
	})

	t.Run("requests multiplicity rejected", func(t *testing.T) {
		resp := resource.UpgradeStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		upgrader.StateUpgrader(ctx, resource.UpgradeStateRequest{
			RawState: &tfprotov6.RawState{JSON: mustJSON(t, map[string]any{
				"id":       "team-a/demo",
				"metadata": []map[string]any{{"name": "demo", "namespace": "team-a"}},
				"spec": []map[string]any{{
					"container": []map[string]any{{
						"name": "work", "image": "busybox",
						"resources": []map[string]any{{
							"requests": []map[string]any{
								{"cpu": "10m"},
								{"memory": "16Mi"},
							},
						}},
					}},
				}},
			})},
		}, &resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected diagnostics for requests list multiplicity")
		}
	})

	t.Run("spec cardinality rejected", func(t *testing.T) {
		for _, source := range []map[string]any{
			{
				"id":       "team-a/demo",
				"metadata": []map[string]any{{"name": "demo", "namespace": "team-a"}},
				"spec":     []map[string]any{},
			},
			{
				"id":       "team-a/demo",
				"metadata": []map[string]any{{"name": "demo", "namespace": "team-a"}},
				"spec": []map[string]any{
					{"container": []map[string]any{{"name": "one", "image": "busybox"}}},
					{"container": []map[string]any{{"name": "two", "image": "busybox"}}},
				},
			},
		} {
			resp := resource.UpgradeStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			upgrader.StateUpgrader(ctx, resource.UpgradeStateRequest{
				RawState: &tfprotov6.RawState{JSON: mustJSON(t, source)},
			}, &resp)
			if !resp.Diagnostics.HasError() {
				t.Fatal("expected diagnostics for malformed spec cardinality")
			}
		}
	})
}

func TestPodV1DecodeSourceStatePreservesLargeIntegers(t *testing.T) {
	source := map[string]any{
		"id": "team-a/demo",
		"metadata": []map[string]any{{
			"name": "demo", "namespace": "team-a",
		}},
		"spec": []map[string]any{{
			"security_context": []map[string]any{{
				"run_as_user": int64(9007199254740993),
			}},
		}},
	}

	state, _, _, diags := podV1DecodeSourceState(mustJSON(t, source), 1, "summary")
	if diags.HasError() {
		t.Fatal(diags)
	}

	spec := state["spec"].([]any)
	securityContext := spec[0].(map[string]any)["security_context"].([]any)
	runAsUser := securityContext[0].(map[string]any)["run_as_user"]
	number, ok := runAsUser.(json.Number)
	if !ok {
		t.Fatalf("run_as_user type = %T, want json.Number", runAsUser)
	}
	if number.String() != "9007199254740993" {
		t.Fatalf("run_as_user = %q, want %q", number.String(), "9007199254740993")
	}
}

func TestPodV1MoveState(t *testing.T) {
	ctx := context.Background()
	pod := &PodV1{}
	schemaResp, identityResp := podV1Schemas(t, pod)
	mover := pod.MoveState(ctx)[0].StateMover

	t.Run("v0 conversion and identity before 2.38", func(t *testing.T) {
		source := map[string]any{
			"id": "team-a/demo",
			"metadata": []map[string]any{{
				"name": "demo", "namespace": "team-a", "uid": "uid-1",
				"generation": 3, "resource_version": "rv-1", "generate_name": "demo-",
				"annotations": map[string]any{"a": "b"}, "labels": map[string]any{},
			}},
			"target_state": []any{"Pending"},
			"timeouts":     map[string]any{"create": "5m", "delete": ""},
			"spec": []map[string]any{{
				"container": []map[string]any{
					{
						"name": "work", "image": "busybox",
						"resources": []map[string]any{{
							"limits":   []map[string]any{{"cpu": "20m"}},
							"requests": []map[string]any{{"memory": "16Mi"}},
						}},
					},
				},
				"init_container": []map[string]any{
					{
						"name": "init", "image": "busybox",
						"resources": []map[string]any{{"limits": []map[string]any{{"cpu": "5m"}}}},
					},
				},
			}},
		}
		expected := map[string]any{
			"id": "team-a/demo",
			"metadata": []map[string]any{{
				"name": "demo", "namespace": "team-a", "uid": "uid-1",
				"generation": 3, "resource_version": "rv-1", "generate_name": "demo-",
				"annotations": map[string]any{"a": "b"}, "labels": map[string]any{},
			}},
			"target_state": []any{"Pending"},
			"timeouts":     map[string]any{"create": "5m", "delete": nil},
			"spec": []map[string]any{{
				"container": []map[string]any{
					{
						"name": "work", "image": "busybox",
						"resources": []map[string]any{{
							"limits":   map[string]any{"cpu": "20m"},
							"requests": map[string]any{"memory": "16Mi"},
						}},
					},
				},
				"init_container": []map[string]any{
					{
						"name": "init", "image": "busybox",
						"resources": []map[string]any{{
							"limits":   map[string]any{"cpu": "5m"},
							"requests": map[string]any{},
						}},
					},
				},
			}},
		}

		resp := resource.MoveStateResponse{
			TargetState:    tfsdk.State{Schema: schemaResp.Schema},
			TargetIdentity: &tfsdk.ResourceIdentity{Schema: identityResp.IdentitySchema},
		}
		mover(ctx, resource.MoveStateRequest{
			SourceTypeName:        podUnversionedTypeName,
			SourceSchemaVersion:   0,
			SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes",
			SourceRawState:        &tfprotov6.RawState{JSON: mustJSON(t, source)},
		}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}

		want := podV1TerraformValue(t, schemaResp, expected)
		if !resp.TargetState.Raw.Equal(want) {
			t.Fatalf("moved state mismatch\nwant: %s\ngot:  %s", want, resp.TargetState.Raw)
		}
		assertPodIdentity(t, ctx, resp.TargetIdentity, "team-a", "demo")
	})

	t.Run("v1 preserves null and empty maps", func(t *testing.T) {
		source := map[string]any{
			"id": "team-b/generated-123",
			"metadata": []map[string]any{{
				"name": "generated-123", "namespace": "team-b", "uid": "uid-2",
				"generation": 2, "resource_version": "rv-2", "generate_name": "generated-",
				"annotations": nil, "labels": map[string]any{},
			}},
			"timeouts": map[string]any{"create": "", "delete": "4m"},
			"spec": []map[string]any{{
				"host_network": true,
				"container": []map[string]any{
					{
						"name": "work", "image": "busybox",
						"resources": []map[string]any{{
							"limits":   map[string]any{},
							"requests": nil,
						}},
					},
				},
			}},
		}
		expected := map[string]any{
			"id": "team-b/generated-123",
			"metadata": []map[string]any{{
				"name": "generated-123", "namespace": "team-b", "uid": "uid-2",
				"generation": 2, "resource_version": "rv-2", "generate_name": "generated-",
				"annotations": nil, "labels": map[string]any{},
			}},
			"timeouts": map[string]any{"create": nil, "delete": "4m"},
			"spec": []map[string]any{{
				"host_network": true,
				"container": []map[string]any{
					{
						"name": "work", "image": "busybox",
						"resources": []map[string]any{{
							"limits":   map[string]any{},
							"requests": nil,
						}},
					},
				},
			}},
		}

		resp := resource.MoveStateResponse{
			TargetState:    tfsdk.State{Schema: schemaResp.Schema},
			TargetIdentity: &tfsdk.ResourceIdentity{Schema: identityResp.IdentitySchema},
		}
		mover(ctx, resource.MoveStateRequest{
			SourceTypeName:        podUnversionedTypeName,
			SourceSchemaVersion:   1,
			SourceProviderAddress: "mirror.example.com/hashicorp/kubernetes",
			SourceRawState:        &tfprotov6.RawState{JSON: mustJSON(t, source)},
			SourceIdentity:        &tfprotov6.RawState{JSON: []byte(`{"namespace":"team-b","name":"generated-123","api_version":"v1","kind":"Pod"}`)},
		}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}

		want := podV1TerraformValue(t, schemaResp, expected)
		if !resp.TargetState.Raw.Equal(want) {
			t.Fatalf("moved state mismatch\nwant: %s\ngot:  %s", want, resp.TargetState.Raw)
		}
		assertPodIdentity(t, ctx, resp.TargetIdentity, "team-b", "generated-123")
	})

	for _, tc := range []struct {
		name            string
		sourceType      string
		sourceVersion   int64
		providerAddress string
		rawState        *tfprotov6.RawState
		sourceIdentity  *tfprotov6.RawState
		wantSkip        bool
		wantError       bool
	}{
		{name: "wrong type", sourceType: "kubernetes_namespace", sourceVersion: 1, providerAddress: "registry.terraform.io/hashicorp/kubernetes", wantSkip: true},
		{name: "wrong version", sourceType: podUnversionedTypeName, sourceVersion: 2, providerAddress: "registry.terraform.io/hashicorp/kubernetes", wantSkip: true},
		{name: "wrong provider", sourceType: podUnversionedTypeName, sourceVersion: 1, providerAddress: "registry.terraform.io/example/kubernetes", wantSkip: true},
		{name: "missing raw state", sourceType: podUnversionedTypeName, sourceVersion: 1, providerAddress: "registry.terraform.io/hashicorp/kubernetes", wantError: true},
		{name: "malformed json", sourceType: podUnversionedTypeName, sourceVersion: 1, providerAddress: "registry.terraform.io/hashicorp/kubernetes", rawState: &tfprotov6.RawState{JSON: []byte(`{`)}, wantError: true},
		{name: "malformed id", sourceType: podUnversionedTypeName, sourceVersion: 1, providerAddress: "registry.terraform.io/hashicorp/kubernetes", rawState: &tfprotov6.RawState{JSON: mustJSON(t, map[string]any{
			"id":       "bad",
			"metadata": []map[string]any{{"name": "demo", "namespace": "team-a"}},
			"spec":     []map[string]any{{"container": []map[string]any{{"name": "work", "image": "busybox"}}}},
		})}, wantError: true},
		{name: "metadata mismatch", sourceType: podUnversionedTypeName, sourceVersion: 1, providerAddress: "registry.terraform.io/hashicorp/kubernetes", rawState: &tfprotov6.RawState{JSON: mustJSON(t, map[string]any{
			"id":       "team-a/demo",
			"metadata": []map[string]any{{"name": "demo", "namespace": "team-b"}},
			"spec":     []map[string]any{{"container": []map[string]any{{"name": "work", "image": "busybox"}}}},
		})}, wantError: true},
		{name: "timeouts unknown attr", sourceType: podUnversionedTypeName, sourceVersion: 1, providerAddress: "registry.terraform.io/hashicorp/kubernetes", rawState: &tfprotov6.RawState{JSON: mustJSON(t, map[string]any{
			"id":       "team-a/demo",
			"metadata": []map[string]any{{"name": "demo", "namespace": "team-a"}},
			"spec":     []map[string]any{{"container": []map[string]any{{"name": "work", "image": "busybox"}}}},
			"timeouts": map[string]any{"create": "5m", "update": "1m"},
		})}, wantError: true},
		{name: "identity conflict", sourceType: podUnversionedTypeName, sourceVersion: 1, providerAddress: "registry.terraform.io/hashicorp/kubernetes", rawState: &tfprotov6.RawState{JSON: mustJSON(t, map[string]any{
			"id":       "team-a/demo",
			"metadata": []map[string]any{{"name": "demo", "namespace": "team-a"}},
			"spec":     []map[string]any{{"container": []map[string]any{{"name": "work", "image": "busybox"}}}},
		})}, sourceIdentity: &tfprotov6.RawState{JSON: []byte(`{"namespace":"other","name":"demo","api_version":"v1","kind":"Pod"}`)}, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := resource.MoveStateResponse{
				TargetState:    tfsdk.State{Schema: schemaResp.Schema},
				TargetIdentity: &tfsdk.ResourceIdentity{Schema: identityResp.IdentitySchema},
			}
			mover(ctx, resource.MoveStateRequest{
				SourceTypeName:        tc.sourceType,
				SourceSchemaVersion:   tc.sourceVersion,
				SourceProviderAddress: tc.providerAddress,
				SourceRawState:        tc.rawState,
				SourceIdentity:        tc.sourceIdentity,
			}, &resp)
			if resp.Diagnostics.HasError() != tc.wantError {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}
			if tc.wantSkip || tc.wantError {
				if !resp.TargetState.Raw.IsNull() || !resp.TargetIdentity.Raw.IsNull() {
					t.Fatal("unsupported source must not produce state or identity")
				}
			}
		})
	}
}

func podV1Schemas(t *testing.T, pod *PodV1) (resource.SchemaResponse, resource.IdentitySchemaResponse) {
	t.Helper()
	ctx := context.Background()

	var schemaResp resource.SchemaResponse
	pod.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatal(schemaResp.Diagnostics)
	}

	var identityResp resource.IdentitySchemaResponse
	pod.IdentitySchema(ctx, resource.IdentitySchemaRequest{}, &identityResp)
	if identityResp.Diagnostics.HasError() {
		t.Fatal(identityResp.Diagnostics)
	}

	return schemaResp, identityResp
}

func podV1ImportStateSchema(t *testing.T) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	schemaResponse, _ := podV1Schemas(t, &PodV1{})
	s := schemaResponse.Schema
	raw := tfprotov6.RawState{JSON: mustJSON(t, map[string]any{"id": nil})}
	value, err := raw.Unmarshal(s.Type().TerraformType(ctx))
	if err != nil {
		t.Fatalf("failed to initialize import state: %s", err)
	}
	return tfsdk.State{
		Schema: s,
		Raw:    value,
	}
}

func podV1TerraformValue(t *testing.T, schemaResp resource.SchemaResponse, state map[string]any) tftypes.Value {
	t.Helper()
	raw := tfprotov6.RawState{JSON: mustJSON(t, state)}
	value, err := raw.Unmarshal(schemaResp.Schema.Type().TerraformType(context.Background()))
	if err != nil {
		t.Fatalf("failed to decode expected state: %s", err)
	}
	return value
}

func assertPodIdentity(t *testing.T, ctx context.Context, identity *tfsdk.ResourceIdentity, namespace, name string) {
	t.Helper()
	if identity == nil {
		t.Fatal("identity is nil")
		return
	}
	var got common.NamespacedResourceIdentity
	if diags := identity.Get(ctx, &got); diags.HasError() {
		t.Fatal(diags)
	}
	want := podV1Identity(namespace, name)
	if !got.APIVersion.Equal(want.APIVersion) ||
		!got.Kind.Equal(want.Kind) ||
		!got.Name.Equal(want.Name) ||
		!got.Namespace.Equal(want.Namespace) {
		t.Fatalf("identity mismatch\nwant: %#v\ngot:  %#v", want, got)
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json: %s", err)
	}
	return data
}
