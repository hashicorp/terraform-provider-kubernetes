// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1_test

import (
	"context"
	"encoding/json"
	"testing"

	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/rbacv1"
)

// TestClusterRoleBindingMoveState exercises the ClusterRoleBinding StateMover
// without a running Kubernetes cluster.
func TestClusterRoleBindingMoveState(t *testing.T) {
	ctx := context.Background()

	crb := &rbacv1.ClusterRoleBinding{}

	var schemaResp frameworkresource.SchemaResponse
	crb.Schema(ctx, frameworkresource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatal(schemaResp.Diagnostics)
	}

	var identitySchemaResp frameworkresource.IdentitySchemaResponse
	crb.IdentitySchema(ctx, frameworkresource.IdentitySchemaRequest{}, &identitySchemaResp)
	if identitySchemaResp.Diagnostics.HasError() {
		t.Fatal(identitySchemaResp.Diagnostics)
	}

	mover := crb.MoveState(ctx)[0].StateMover

	// validSourceJSON returns a minimal but structurally valid JSON payload
	// that matches the SDKv2 kubernetes_cluster_role_binding schema layout.
	// ClusterRoleBinding is cluster-scoped so metadata has no namespace field.
	validSourceJSON := func() []byte {
		src := map[string]any{
			"id": "my-binding",
			"metadata": []map[string]any{{
				"name":             "my-binding",
				"generate_name":    "",
				"annotations":      nil,
				"labels":           nil,
				"generation":       0,
				"resource_version": "",
				"uid":              "",
			}},
			"role_ref": []map[string]any{{
				"api_group": "rbac.authorization.k8s.io",
				"kind":      "ClusterRole",
				"name":      "cluster-admin",
			}},
			"subject": []map[string]any{{
				"api_group": "rbac.authorization.k8s.io",
				"kind":      "User",
				"name":      "notauser",
				"namespace": "",
			}},
		}
		b, err := json.Marshal(src)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}

	// buildSourceState constructs a *tfsdk.State pre-populated with validSourceJSON
	// using the same RawState → tftypes.Value unmarshal path the framework itself uses.
	buildSourceState := func() *tfsdk.State {
		raw := &tfprotov6.RawState{JSON: validSourceJSON()}
		val, err := raw.Unmarshal(schemaResp.Schema.Type().TerraformType(ctx))
		if err != nil {
			t.Fatalf("could not unmarshal source state: %v", err)
		}
		s := tfsdk.State{Schema: schemaResp.Schema, Raw: val}
		return &s
	}

	// ---- valid cases: mover must populate TargetState and TargetIdentity ----
	for _, tc := range []struct {
		name            string
		providerAddress string
	}{
		{name: "canonical provider address", providerAddress: "registry.terraform.io/hashicorp/kubernetes"},
		{name: "mirror provider address", providerAddress: "mirror.example.com/hashicorp/kubernetes"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := frameworkresource.MoveStateResponse{
				TargetState:    tfsdk.State{Schema: schemaResp.Schema},
				TargetIdentity: &tfsdk.ResourceIdentity{Schema: identitySchemaResp.IdentitySchema},
			}
			mover(ctx, frameworkresource.MoveStateRequest{
				SourceTypeName:        "kubernetes_cluster_role_binding",
				SourceSchemaVersion:   0,
				SourceProviderAddress: tc.providerAddress,
				SourceState:           buildSourceState(),
			}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}
			if resp.TargetState.Raw.IsNull() {
				t.Fatal("expected TargetState to be populated, got null")
			}
			var identity common.ResourceIdentity
			if diags := resp.TargetIdentity.Get(ctx, &identity); diags.HasError() {
				t.Fatal(diags)
			}
			if identity.Name.ValueString() != "my-binding" {
				t.Errorf("unexpected identity name %q", identity.Name.ValueString())
			}
		})
	}

	// ---- guard cases: unrecognised source must produce no state/identity ----
	for _, tc := range []struct {
		name            string
		sourceType      string
		sourceVersion   int64
		providerAddress string
	}{
		{name: "wrong resource type", sourceType: "kubernetes_role_binding", sourceVersion: 0, providerAddress: "registry.terraform.io/hashicorp/kubernetes"},
		{name: "v1 suffix skipped", sourceType: "kubernetes_cluster_role_binding_v1", sourceVersion: 0, providerAddress: "registry.terraform.io/hashicorp/kubernetes"},
		{name: "wrong schema version", sourceType: "kubernetes_cluster_role_binding", sourceVersion: 1, providerAddress: "registry.terraform.io/hashicorp/kubernetes"},
		{name: "wrong provider address", sourceType: "kubernetes_cluster_role_binding", sourceVersion: 0, providerAddress: "registry.terraform.io/other/kubernetes"},
		{name: "forked provider namespace", sourceType: "kubernetes_cluster_role_binding", sourceVersion: 0, providerAddress: "registry.terraform.io/nothashicorp/kubernetes"},
		{name: "empty provider address", sourceType: "kubernetes_cluster_role_binding", sourceVersion: 0, providerAddress: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := frameworkresource.MoveStateResponse{
				TargetState:    tfsdk.State{Schema: schemaResp.Schema},
				TargetIdentity: &tfsdk.ResourceIdentity{Schema: identitySchemaResp.IdentitySchema},
			}
			mover(ctx, frameworkresource.MoveStateRequest{
				SourceTypeName:        tc.sourceType,
				SourceSchemaVersion:   tc.sourceVersion,
				SourceProviderAddress: tc.providerAddress,
			}, &resp)
			if resp.Diagnostics.HasError() {
				t.Errorf("unexpected diagnostics: %v", resp.Diagnostics)
			}
			if !resp.TargetState.Raw.IsNull() || !resp.TargetIdentity.Raw.IsNull() {
				t.Fatal("unrecognised source must not produce state or identity")
			}
		})
	}

	// ---- error cases: matched source but absent or undecodable SourceState ----
	//
	// The framework leaves SourceState nil when it cannot decode the source
	// state. A non-nil state with a null Raw value cannot be decoded into the
	// model, so SourceState.Get() itself fails.
	for _, tc := range []struct {
		name        string
		sourceState *tfsdk.State
	}{
		{name: "nil SourceState", sourceState: nil},
		{name: "undecodable SourceState", sourceState: &tfsdk.State{Schema: schemaResp.Schema}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := frameworkresource.MoveStateResponse{
				TargetState:    tfsdk.State{Schema: schemaResp.Schema},
				TargetIdentity: &tfsdk.ResourceIdentity{Schema: identitySchemaResp.IdentitySchema},
			}
			mover(ctx, frameworkresource.MoveStateRequest{
				SourceTypeName:        "kubernetes_cluster_role_binding",
				SourceSchemaVersion:   0,
				SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes",
				SourceState:           tc.sourceState,
			}, &resp)
			if !resp.Diagnostics.HasError() {
				t.Error("expected an error diagnostic, got none")
			}
			if !resp.TargetState.Raw.IsNull() {
				t.Error("TargetState must remain null when source is absent or undecodable")
			}
			if !resp.TargetIdentity.Raw.IsNull() {
				t.Error("TargetIdentity must remain null when source is absent or undecodable")
			}
		})
	}
}
