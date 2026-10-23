// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestRoleBindingSubjectAPIGroupPlan(t *testing.T) {
	ctx := context.Background()
	s := roleBindingV1Schema()
	modifier := s.Blocks["subject"].(schema.ListNestedBlock).NestedObject.Attributes["api_group"].(schema.StringAttribute).PlanModifiers[0]
	for _, tc := range []struct {
		name                                    string
		config, state, planned                  types.String
		priorKind, kind, priorName, subjectName types.String
		reuse                                   bool
	}{
		{"unchanged", types.StringNull(), types.StringValue(""), types.StringUnknown(), types.StringValue("ServiceAccount"), types.StringValue("ServiceAccount"), types.StringValue("builder"), types.StringValue("builder"), true},
		{"preserveEmptyUser", types.StringNull(), types.StringValue(""), types.StringUnknown(), types.StringValue("User"), types.StringValue("User"), types.StringValue("alice"), types.StringValue("alice"), true},
		{"changedKind", types.StringNull(), types.StringValue("rbac.authorization.k8s.io"), types.StringUnknown(), types.StringValue("User"), types.StringValue("ServiceAccount"), types.StringValue("builder"), types.StringValue("builder"), false},
		{"reordered", types.StringNull(), types.StringValue(""), types.StringUnknown(), types.StringValue("ServiceAccount"), types.StringValue("ServiceAccount"), types.StringValue("builder"), types.StringValue("other"), false},
		{"unknownKind", types.StringNull(), types.StringValue(""), types.StringUnknown(), types.StringValue("ServiceAccount"), types.StringUnknown(), types.StringValue("builder"), types.StringValue("builder"), false},
		{"unknownName", types.StringNull(), types.StringValue(""), types.StringUnknown(), types.StringValue("ServiceAccount"), types.StringValue("ServiceAccount"), types.StringValue("builder"), types.StringUnknown(), false},
		{"unknownConfig", types.StringUnknown(), types.StringValue(""), types.StringUnknown(), types.StringValue("ServiceAccount"), types.StringValue("ServiceAccount"), types.StringValue("builder"), types.StringValue("builder"), false},
		{"configuredEmpty", types.StringValue(""), types.StringValue("rbac.authorization.k8s.io"), types.StringValue(""), types.StringValue("User"), types.StringValue("User"), types.StringValue("alice"), types.StringValue("alice"), false},
		{"newSubject", types.StringNull(), types.StringNull(), types.StringUnknown(), types.StringNull(), types.StringValue("ServiceAccount"), types.StringNull(), types.StringValue("builder"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prior := RoleBindingModel{Subject: []SubjectModel{{Kind: tc.priorKind, Name: tc.priorName, APIGroup: tc.state, Namespace: types.StringValue("default")}}}
			planned := RoleBindingModel{Subject: []SubjectModel{{Kind: tc.kind, Name: tc.subjectName, APIGroup: tc.planned, Namespace: types.StringValue("default")}}}
			state := tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
			plan := tfsdk.Plan{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
			if diags := state.Set(ctx, &prior); diags.HasError() {
				t.Fatal(diags)
			}
			if diags := plan.Set(ctx, &planned); diags.HasError() {
				t.Fatal(diags)
			}
			resp := planmodifier.StringResponse{PlanValue: tc.planned}
			modifier.PlanModifyString(ctx, planmodifier.StringRequest{
				Path:        path.Root("subject").AtListIndex(0).AtName("api_group"),
				ConfigValue: tc.config, StateValue: tc.state, PlanValue: tc.planned, State: state, Plan: plan,
			}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			want := tc.planned
			if tc.reuse {
				want = tc.state
			}
			if !resp.PlanValue.Equal(want) {
				t.Fatalf("plan = %v, want %v", resp.PlanValue, want)
			}
		})
	}
}

func TestRoleBindingConfigure(t *testing.T) {
	for _, data := range []any{nil, "invalid", func() any { return nil }} {
		r := &RoleBindingV1{}
		resp := resource.ConfigureResponse{}
		r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: data}, &resp)
		_, invalid := data.(string)
		if resp.Diagnostics.HasError() != invalid {
			t.Fatalf("data %T: diagnostics %v", data, resp.Diagnostics)
		}
	}
}
