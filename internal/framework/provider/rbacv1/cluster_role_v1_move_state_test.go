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
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/rbacv1"
)

// TestClusterRoleMoveState exercises the ClusterRole StateMover with SDKv2-shaped raw
// JSON, without a running cluster. Matching sources are moved field by field, other
// sources are skipped, and absent or malformed source state returns an error with no
// target state or identity (never a panic).
func TestClusterRoleMoveState(t *testing.T) {
	ctx := context.Background()
	r := &rbacv1.ClusterRoleV1{}
	var schema frameworkresource.SchemaResponse
	r.Schema(ctx, frameworkresource.SchemaRequest{}, &schema)
	identitySchema := common.IdentitySchema()

	meta := `"metadata":[{"name":"my-role","uid":"retained","resource_version":"7","generation":0,"labels":{},"annotations":null,"generate_name":""}]`
	// SDKv2 state as written by kubernetes_cluster_role at schema version 0: omitted
	// list attributes are null after create and [] after a refresh.
	valid := `{"id":"my-role",` + meta + `,"aggregation_rule":[],` +
		`"rule":[{"api_groups":[""],"non_resource_urls":null,"resource_names":[],"resources":["pods","pods/log"],"verbs":["get","list"]},` +
		`{"api_groups":null,"non_resource_urls":["/metrics"],"resource_names":null,"resources":null,"verbs":["get"]}]}`
	aggregated := `{"id":"my-role",` + meta + `,"rule":[{"api_groups":[""],"non_resource_urls":[],"resource_names":[],"resources":["configmaps"],"verbs":["get"]}],` +
		`"aggregation_rule":[{"cluster_role_selectors":[` +
		`{"match_labels":{"rbac.example.com/aggregate-to-x":"true"},"match_expressions":[]},` +
		`{"match_labels":null,"match_expressions":[{"key":"env","operator":"Exists","values":null},{"key":"tier","operator":"In","values":["a","b"]}]}]}]}`

	for _, tc := range []struct {
		name, sourceType, provider, raw string
		version                         int64
		wantError, wantSkip, noIdentity bool
		wantDetail                      string
		nullLabels, emptyAnnotations    bool
		aggregated                      bool
	}{
		{name: "valid", raw: valid},
		{name: "aggregated", raw: aggregated, aggregated: true},
		{name: "bothNullMaps", raw: strings.Replace(valid, `"labels":{}`, `"labels":null`, 1), nullLabels: true},
		{name: "bothEmptyMaps", raw: strings.Replace(valid, `"annotations":null`, `"annotations":{}`, 1), emptyAnnotations: true},
		{name: "omittedMaps", raw: strings.Replace(strings.Replace(valid, `"labels":{},`, "", 1), `"annotations":null,`, "", 1), nullLabels: true},
		{name: "generatedName", raw: strings.Replace(valid, `"generate_name":""`, `"generate_name":"my-"`, 1)},
		{name: "withoutTargetIdentity", raw: valid, noIdentity: true},
		{name: "privateRegistry", provider: "registry.example.com/hashicorp/kubernetes", raw: valid},
		{name: "wrongType", sourceType: "kubernetes_role", raw: valid, wantSkip: true},
		{name: "v1Type", sourceType: "kubernetes_cluster_role_v1", raw: valid, wantSkip: true},
		{name: "prefixedType", sourceType: "other_kubernetes_cluster_role", raw: valid, wantSkip: true},
		{name: "wrongProvider", provider: "registry.terraform.io/example/kubernetes", raw: valid, wantSkip: true},
		{name: "lookalikeProvider", provider: "registry.example.com/nothashicorp/kubernetes", raw: valid, wantSkip: true},
		{name: "emptyProvider", provider: "-", raw: valid, wantSkip: true},
		{name: "wrongVersion", version: 1, raw: valid, wantSkip: true},
		{name: "missingJSON", wantError: true, wantDetail: "The source state has no JSON data. Flatmap state is not supported."},
		{name: "malformedJSON", raw: "{", wantError: true},
		{name: "emptyID", raw: `{"metadata":[{}]}`, wantError: true, wantDetail: "The source state must have a non-empty id."},
		{name: "missingMetadata", raw: `{"id":"my-role"}`, wantError: true, wantDetail: "Expected exactly 1 metadata element in the source state, got 0."},
		{name: "multipleMetadata", raw: `{"id":"my-role","metadata":[{},{}]}`, wantError: true, wantDetail: "Expected exactly 1 metadata element in the source state, got 2."},
		{name: "multipleAggregationRule", raw: `{"id":"my-role","metadata":[{}],"aggregation_rule":[{},{}]}`, wantError: true, wantDetail: "Expected at most 1 aggregation_rule element in the source state, got 2."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := frameworkresource.MoveStateRequest{
				SourceTypeName:        "kubernetes_cluster_role",
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

			var state rbacv1.ClusterRoleModel
			if diags := resp.TargetState.Get(ctx, &state); diags.HasError() {
				t.Fatal(diags)
			}
			m := state.Metadata[0]
			if state.ID.ValueString() != "my-role" || m.Name.ValueString() != "my-role" ||
				m.UID.ValueString() != "retained" || m.ResourceVersion.ValueString() != "7" ||
				m.Annotations.IsNull() == tc.emptyAnnotations || m.Labels.IsNull() != tc.nullLabels {
				t.Fatalf("incorrect metadata: %#v", state)
			}
			if tc.name == "generatedName" {
				if m.GenerateName.ValueString() != "my-" {
					t.Fatal("lost generate_name")
				}
			} else if !m.GenerateName.IsNull() {
				t.Fatal("unset generate_name must normalize to null")
			}

			if tc.aggregated {
				if len(state.Rule) != 1 || len(state.AggregationRule) != 1 {
					t.Fatalf("incorrect rule/aggregation_rule counts: %#v", state)
				}
				sels := state.AggregationRule[0].ClusterRoleSelectors
				if len(sels) != 2 {
					t.Fatalf("incorrect selectors: %#v", sels)
				}
				if sels[0].MatchLabels.IsNull() || len(sels[0].MatchLabels.Elements()) != 1 || len(sels[0].MatchExpressions) != 0 {
					t.Fatalf("incorrect first selector: %#v", sels[0])
				}
				if !sels[1].MatchLabels.IsNull() || len(sels[1].MatchExpressions) != 2 {
					t.Fatalf("incorrect second selector: %#v", sels[1])
				}
				exists, in := sels[1].MatchExpressions[0], sels[1].MatchExpressions[1]
				if exists.Key.ValueString() != "env" || exists.Operator.ValueString() != "Exists" || !exists.Values.IsNull() ||
					in.Key.ValueString() != "tier" || len(in.Values.Elements()) != 2 {
					t.Fatalf("incorrect expressions: %#v", sels[1].MatchExpressions)
				}
			} else {
				if len(state.AggregationRule) != 0 {
					t.Fatalf("unexpected aggregation_rule: %#v", state.AggregationRule)
				}
				if len(state.Rule) != 2 {
					t.Fatalf("incorrect rule count: %#v", state.Rule)
				}
				// The schema defaults an omitted list to an empty set, so SDKv2's null and
				// [] both become a known empty set. Keeping null would plan an in-place
				// update on the first plan after the move when refresh is skipped.
				first, second := state.Rule[0], state.Rule[1]
				for name, set := range map[string]types.Set{
					"first.non_resource_urls": first.NonResourceURLs, "first.resource_names": first.ResourceNames,
					"second.api_groups": second.APIGroups, "second.resources": second.Resources, "second.resource_names": second.ResourceNames,
				} {
					if set.IsNull() || set.IsUnknown() || len(set.Elements()) != 0 {
						t.Fatalf("%s must be a known empty set, got %#v", name, set)
					}
				}
				if len(first.Resources.Elements()) != 2 || len(first.Verbs.Elements()) != 2 || len(first.APIGroups.Elements()) != 1 ||
					len(second.NonResourceURLs.Elements()) != 1 || len(second.Verbs.Elements()) != 1 {
					t.Fatalf("incorrect rules: %#v", state.Rule)
				}
			}

			if resp.TargetIdentity != nil {
				var identity common.ResourceIdentity
				if diags := resp.TargetIdentity.Get(ctx, &identity); diags.HasError() {
					t.Fatal(diags)
				}
				if identity.Kind.ValueString() != "ClusterRole" || identity.APIVersion.ValueString() != "rbac.authorization.k8s.io/v1" ||
					identity.Name.ValueString() != "my-role" {
					t.Fatalf("identity: %#v", identity)
				}
			}
		})
	}
	// Ensure the fixtures are JSON, rather than testing a decode failure accidentally.
	for _, fixture := range []string{valid, aggregated} {
		if !json.Valid([]byte(fixture)) {
			t.Fatal("invalid fixture")
		}
	}
}
