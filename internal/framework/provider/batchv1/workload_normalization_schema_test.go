// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestWorkloadTemplateLegacyEmptyNormalization(t *testing.T) {
	ctx := context.Background()
	block := podTemplateBlock(false)
	testSchema := schema.Schema{Blocks: map[string]schema.Block{"template": block}}
	for _, tc := range []struct {
		name                string
		prior, plan, config types.String
		expectReplacement   bool
	}{
		{"legacy-empty-to-null", types.StringValue(""), types.StringNull(), types.StringNull(), false},
		{"real-name-removed", types.StringValue("prefix-"), types.StringNull(), types.StringNull(), true},
		{"real-name-added", types.StringValue(""), types.StringValue("prefix-"), types.StringValue("prefix-"), true},
		{"explicit-empty-added", types.StringNull(), types.StringValue(""), types.StringValue(""), true},
		{"configured-unknown", types.StringValue(""), types.StringUnknown(), types.StringUnknown(), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prior := workloadTemplateNameFixture(t, block, tc.prior)
			planned := workloadTemplateNameFixture(t, block, tc.plan)
			configured := workloadTemplateNameFixture(t, block, tc.config)
			raw := func(value types.List) tftypes.Value {
				object := types.ObjectValueMust(testSchema.Type().(types.ObjectType).AttrTypes, map[string]attr.Value{"template": value})
				result, err := object.ToTerraformValue(ctx)
				if err != nil {
					t.Fatal(err)
				}
				return result
			}
			config := tfsdk.Config{Schema: testSchema, Raw: raw(configured)}
			plan := tfsdk.Plan{Schema: testSchema, Raw: raw(planned)}
			state := tfsdk.State{Schema: testSchema, Raw: raw(prior)}
			listRequest := planmodifier.ListRequest{
				Path: path.Root("template"), Config: config, Plan: plan, State: state,
				ConfigValue: configured, PlanValue: planned, StateValue: prior,
			}
			listResponse := planmodifier.ListResponse{PlanValue: planned}
			for _, modifier := range block.PlanModifiers {
				modifier.PlanModifyList(ctx, listRequest, &listResponse)
			}
			if listResponse.Diagnostics.HasError() || listResponse.RequiresReplace != tc.expectReplacement {
				t.Fatalf("template replacement=%t, want %t; %s", listResponse.RequiresReplace, tc.expectReplacement, listResponse.Diagnostics)
			}
			if !listResponse.PlanValue.Equal(planned) {
				t.Fatal("replacement comparison rewrote the real plan")
			}
			metadata := block.NestedObject.Blocks["metadata"].(schema.ListNestedBlock)
			name := metadata.NestedObject.Attributes["generate_name"].(schema.StringAttribute)
			stringRequest := planmodifier.StringRequest{
				Path:   path.Root("template").AtListIndex(0).AtName("metadata").AtListIndex(0).AtName("generate_name"),
				Config: config, Plan: plan, State: state,
				ConfigValue: tc.config, PlanValue: tc.plan, StateValue: tc.prior,
			}
			stringResponse := planmodifier.StringResponse{PlanValue: tc.plan}
			for _, modifier := range name.PlanModifiers {
				modifier.PlanModifyString(ctx, stringRequest, &stringResponse)
			}
			if stringResponse.Diagnostics.HasError() || stringResponse.RequiresReplace != tc.expectReplacement {
				t.Fatalf("name replacement=%t, want %t; %s", stringResponse.RequiresReplace, tc.expectReplacement, stringResponse.Diagnostics)
			}
			if !stringResponse.PlanValue.Equal(tc.plan) {
				t.Fatal("leaf replacement comparison rewrote the real plan")
			}
		})
	}
}

func workloadTemplateNameFixture(t *testing.T, block schema.ListNestedBlock, name types.String) types.List {
	t.Helper()
	ctx := context.Background()
	metadata := block.NestedObject.Blocks["metadata"].(schema.ListNestedBlock)
	metaType := metadata.NestedObject.Type().(types.ObjectType)
	metaFields := make(map[string]attr.Value, len(metaType.AttrTypes))
	for key, typ := range metaType.AttrTypes {
		value, err := typ.ValueFromTerraform(ctx, tftypes.NewValue(typ.TerraformType(ctx), nil))
		if err != nil {
			t.Fatal(err)
		}
		metaFields[key] = value
	}
	metaFields["generate_name"] = name
	podSpec := block.NestedObject.Blocks["spec"].(schema.ListNestedBlock)
	templateType := block.NestedObject.Type().(types.ObjectType)
	template := types.ObjectValueMust(templateType.AttrTypes, map[string]attr.Value{
		"metadata": types.ListValueMust(metaType, []attr.Value{types.ObjectValueMust(metaType.AttrTypes, metaFields)}),
		"spec":     types.ListValueMust(podSpec.NestedObject.Type(), nil),
	})
	return types.ListValueMust(templateType, []attr.Value{template})
}

func TestWorkloadEmptyNormalizationRejectsCollectionChanges(t *testing.T) {
	for _, pair := range []struct {
		plan, prior attr.Value
	}{
		{types.ListValueMust(types.StringType, nil), types.ListValueMust(types.StringType, []attr.Value{types.StringValue("")})},
		{types.MapNull(types.StringType), types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("")})},
		{types.SetNull(types.StringType), types.SetValueMust(types.StringType, []attr.Value{types.StringValue("")})},
	} {
		if _, ok := workloadEmptyNormalizationValue(context.Background(), pair.plan, pair.prior, pair.plan); ok {
			t.Fatalf("must not discard known collection elements: %s -> %s", pair.prior, pair.plan)
		}
	}
}
