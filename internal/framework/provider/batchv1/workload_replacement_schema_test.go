// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	sdkschema "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestWorkloadObjectListForceNewMatchesSDK(t *testing.T) {
	ctx := context.Background()
	objectType := types.ObjectType{AttrTypes: map[string]attr.Type{"name": types.StringType}}
	modern := func(names []string) types.List {
		elements := make([]attr.Value, 0, len(names))
		for _, name := range names {
			elements = append(elements, types.ObjectValueMust(objectType.AttrTypes, map[string]attr.Value{"name": types.StringValue(name)}))
		}
		return types.ListValueMust(objectType, elements)
	}
	legacy := func(names []string) map[string]interface{} {
		elements := make([]interface{}, 0, len(names))
		for _, name := range names {
			elements = append(elements, map[string]interface{}{"name": name})
		}
		return map[string]interface{}{"template": elements}
	}
	fields := map[string]*sdkschema.Schema{
		"template": {
			Type: sdkschema.TypeList, Optional: true, ForceNew: true,
			Elem: &sdkschema.Resource{Schema: map[string]*sdkschema.Schema{
				"name": {Type: sdkschema.TypeString, Optional: true},
			}},
		},
	}
	for _, tc := range []struct {
		name          string
		prior, plan   []string
		shouldReplace bool
	}{
		{"unchanged", []string{"before"}, []string{"before"}, false},
		{"child-updated", []string{"before"}, []string{"after"}, false},
		{"element-added", []string{"before"}, []string{"before", "after"}, true},
		{"element-removed", []string{"before"}, nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := sdkschema.TestResourceDataRaw(t, fields, legacy(tc.prior))
			data.SetId("existing")
			diff, err := sdkschema.InternalMap(fields).Diff(ctx, data.State(), terraform.NewResourceConfigRaw(legacy(tc.plan)), nil, nil, false)
			if err != nil {
				t.Fatal(err)
			}
			legacyRequiresReplace := diff != nil && diff.RequiresNew()
			if legacyRequiresReplace != tc.shouldReplace {
				t.Fatalf("SDK requires replacement=%t, want %t", legacyRequiresReplace, tc.shouldReplace)
			}
			request := planmodifier.ListRequest{
				State:      tfsdk.State{Raw: tftypes.NewValue(tftypes.String, "exists")},
				Plan:       tfsdk.Plan{Raw: tftypes.NewValue(tftypes.String, "exists")},
				StateValue: modern(tc.prior), PlanValue: modern(tc.plan), ConfigValue: modern(tc.plan),
			}
			response := planmodifier.ListResponse{PlanValue: request.PlanValue}
			for _, modifier := range podTemplateBlock(true).PlanModifiers {
				modifier.PlanModifyList(ctx, request, &response)
			}
			if response.RequiresReplace != legacyRequiresReplace || response.Diagnostics.HasError() {
				t.Fatalf("Framework requires replacement=%t, SDK=%t; %s",
					response.RequiresReplace, legacyRequiresReplace, response.Diagnostics)
			}
		})
	}
}

func TestWorkloadPrimitiveListForceNewOverridesElementFlag(t *testing.T) {
	fields := map[string]*sdkschema.Schema{
		"searches": {
			Type: sdkschema.TypeList, Optional: true,
			Elem: &sdkschema.Schema{Type: sdkschema.TypeString, ForceNew: true},
		},
	}
	data := sdkschema.TestResourceDataRaw(t, fields, map[string]interface{}{"searches": []interface{}{"before.example"}})
	data.SetId("existing")
	diff, err := sdkschema.InternalMap(fields).Diff(context.Background(), data.State(),
		terraform.NewResourceConfigRaw(map[string]interface{}{"searches": []interface{}{"after.example"}}), nil, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if diff == nil || diff.RequiresNew() {
		t.Fatal("SDK primitive list must use its parent's ForceNew, not the element's flag")
	}
}

func TestWorkloadTemplateReplacementByResource(t *testing.T) {
	ctx := context.Background()
	objectType := types.ObjectType{AttrTypes: map[string]attr.Type{"env": types.ListType{ElemType: types.StringType}}}
	value := func(env []attr.Value) types.List {
		return types.ListValueMust(objectType, []attr.Value{
			types.ObjectValueMust(objectType.AttrTypes, map[string]attr.Value{
				"env": types.ListValueMust(types.StringType, env),
			}),
		})
	}
	for _, tc := range []struct {
		name      string
		updatable bool
	}{
		{"job", false},
		{"cronjob", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := planmodifier.ListRequest{
				State:      tfsdk.State{Raw: tftypes.NewValue(tftypes.String, "exists")},
				Plan:       tfsdk.Plan{Raw: tftypes.NewValue(tftypes.String, "exists")},
				StateValue: value([]attr.Value{types.StringValue("configured")}),
				PlanValue:  value(nil), ConfigValue: value(nil),
			}
			template := podTemplateBlock(tc.updatable)
			for name, modifiers := range map[string][]planmodifier.List{
				"template": template.PlanModifiers,
				"spec":     template.NestedObject.Blocks["spec"].(schema.ListNestedBlock).PlanModifiers,
			} {
				response := planmodifier.ListResponse{PlanValue: request.PlanValue}
				for _, modifier := range modifiers {
					modifier.PlanModifyList(ctx, request, &response)
				}
				if response.Diagnostics.HasError() || response.RequiresReplace == tc.updatable {
					t.Fatalf("%s replacement=%t, want %t; %s", name, response.RequiresReplace, !tc.updatable, response.Diagnostics)
				}
			}
		})
	}
}
