// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/rbacv1"
)

// TestClusterRoleBindingMoveState exercises the ClusterRoleBinding StateMover
// with SDKv2-shaped raw JSON, without a running cluster. It mirrors
// TestRoleMoveState: matching sources are moved field by field, other sources
// are skipped, and absent or malformed source state returns an error with no
// target state or identity (never a panic).
func TestClusterRoleBindingMoveState(t *testing.T) {
	ctx := context.Background()
	r := &rbacv1.ClusterRoleBinding{}
	var schema frameworkresource.SchemaResponse
	r.Schema(ctx, frameworkresource.SchemaRequest{}, &schema)
	identitySchema := common.IdentitySchema()

	// SDKv2 state as written by kubernetes_cluster_role_binding at schema version 0.
	valid := `{"id":"my-binding","metadata":[{"name":"my-binding","uid":"retained","resource_version":"7","generation":0,"labels":{},"annotations":null,"generate_name":""}],` +
		`"role_ref":[{"api_group":"rbac.authorization.k8s.io","kind":"ClusterRole","name":"cluster-admin"}],` +
		`"subject":[{"api_group":"rbac.authorization.k8s.io","kind":"User","name":"notauser","namespace":"default"},` +
		`{"api_group":"","kind":"ServiceAccount","name":"default","namespace":"kube-system"}]}`
	for _, tc := range []struct {
		name, sourceType, provider, raw string
		version                         int64
		wantError, wantSkip, noIdentity bool
		wantDetail                      string
		nullLabels, emptyAnnotations    bool
	}{
		{name: "valid", raw: valid},
		{name: "bothNullMaps", raw: strings.Replace(valid, `"labels":{}`, `"labels":null`, 1), nullLabels: true},
		{name: "bothEmptyMaps", raw: strings.Replace(valid, `"annotations":null`, `"annotations":{}`, 1), emptyAnnotations: true},
		{name: "omittedMaps", raw: strings.Replace(strings.Replace(valid, `"labels":{},`, "", 1), `"annotations":null,`, "", 1), nullLabels: true},
		{name: "generatedName", raw: strings.Replace(valid, `"generate_name":""`, `"generate_name":"my-"`, 1)},
		{name: "withoutTargetIdentity", raw: valid, noIdentity: true},
		{name: "privateRegistry", provider: "registry.example.com/hashicorp/kubernetes", raw: valid},
		{name: "wrongType", sourceType: "kubernetes_role_binding", raw: valid, wantSkip: true},
		{name: "v1Type", sourceType: "kubernetes_cluster_role_binding_v1", raw: valid, wantSkip: true},
		{name: "prefixedType", sourceType: "other_kubernetes_cluster_role_binding", raw: valid, wantSkip: true},
		{name: "wrongProvider", provider: "registry.terraform.io/example/kubernetes", raw: valid, wantSkip: true},
		{name: "lookalikeProvider", provider: "registry.example.com/nothashicorp/kubernetes", raw: valid, wantSkip: true},
		{name: "emptyProvider", provider: "-", raw: valid, wantSkip: true},
		{name: "wrongVersion", version: 1, raw: valid, wantSkip: true},
		{name: "missingJSON", wantError: true, wantDetail: "The source state has no JSON data. Flatmap state is not supported."},
		{name: "malformedJSON", raw: "{", wantError: true},
		{name: "emptyID", raw: `{"metadata":[{}],"role_ref":[{}]}`, wantError: true, wantDetail: "The source state must have a non-empty id."},
		{name: "missingMetadata", raw: `{"id":"my-binding","role_ref":[{}]}`, wantError: true, wantDetail: "Expected exactly 1 metadata element in the source state, got 0."},
		{name: "multipleMetadata", raw: `{"id":"my-binding","metadata":[{},{}],"role_ref":[{}]}`, wantError: true, wantDetail: "Expected exactly 1 metadata element in the source state, got 2."},
		{name: "missingRoleRef", raw: `{"id":"my-binding","metadata":[{}]}`, wantError: true, wantDetail: "Expected exactly 1 role_ref element in the source state, got 0."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := frameworkresource.MoveStateRequest{
				SourceTypeName:        "kubernetes_cluster_role_binding",
				SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes",
				SourceSchemaVersion:   tc.version,
			}
			if tc.sourceType != "" {
				req.SourceTypeName = tc.sourceType
			}
			if tc.provider == "-" {
				req.SourceProviderAddress = ""
			} else if tc.provider != "" {
				req.SourceProviderAddress = tc.provider
			}
			if tc.raw != "" {
				req.SourceRawState = &tfprotov6.RawState{JSON: []byte(tc.raw)}
			}
			resp := frameworkresource.MoveStateResponse{
				TargetState: tfsdk.State{Schema: schema.Schema, Raw: tftypes.NewValue(schema.Schema.Type().TerraformType(ctx), nil)},
			}
			if !tc.noIdentity {
				resp.TargetIdentity = &tfsdk.ResourceIdentity{Schema: identitySchema, Raw: tftypes.NewValue(identitySchema.Type().TerraformType(ctx), nil)}
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
				if resp.TargetIdentity != nil && !resp.TargetIdentity.Raw.IsNull() {
					t.Fatal("unexpected target identity")
				}
				return
			}

			var state rbacv1.ClusterRoleBindingModel
			if diags := resp.TargetState.Get(ctx, &state); diags.HasError() {
				t.Fatal(diags)
			}
			m := state.Metadata[0]
			if state.ID.ValueString() != "my-binding" || m.Name.ValueString() != "my-binding" ||
				m.UID.ValueString() != "retained" || m.ResourceVersion.ValueString() != "7" ||
				m.Annotations.IsNull() == tc.emptyAnnotations || m.Labels.IsNull() != tc.nullLabels {
				t.Fatalf("incorrect metadata: %#v", state)
			}
			if len(state.RoleRef) != 1 || state.RoleRef[0].APIGroup.ValueString() != "rbac.authorization.k8s.io" ||
				state.RoleRef[0].Kind.ValueString() != "ClusterRole" || state.RoleRef[0].Name.ValueString() != "cluster-admin" {
				t.Fatalf("incorrect role_ref: %#v", state.RoleRef)
			}
			if len(state.Subject) != 2 ||
				state.Subject[0].Kind.ValueString() != "User" || state.Subject[0].APIGroup.ValueString() != "rbac.authorization.k8s.io" ||
				state.Subject[0].Namespace.ValueString() != "default" ||
				state.Subject[1].Kind.ValueString() != "ServiceAccount" || state.Subject[1].APIGroup.IsNull() ||
				state.Subject[1].APIGroup.ValueString() != "" || state.Subject[1].Namespace.ValueString() != "kube-system" {
				t.Fatalf("incorrect subjects: %#v", state.Subject)
			}
			if tc.name == "generatedName" {
				if m.GenerateName.ValueString() != "my-" {
					t.Fatal("lost generate_name")
				}
			} else if !m.GenerateName.IsNull() {
				t.Fatal("unset generate_name must normalize to null")
			}

			if resp.TargetIdentity != nil {
				var identity common.ResourceIdentity
				if diags := resp.TargetIdentity.Get(ctx, &identity); diags.HasError() {
					t.Fatal(diags)
				}
				if identity.Kind.ValueString() != "ClusterRoleBinding" || identity.APIVersion.ValueString() != "rbac.authorization.k8s.io/v1" ||
					identity.Name.ValueString() != "my-binding" {
					t.Fatalf("identity: %#v", identity)
				}
			}
		})
	}
	// Ensure the fixture itself is JSON, rather than testing a decode failure accidentally.
	if !json.Valid([]byte(valid)) {
		t.Fatal("invalid fixture")
	}
}
