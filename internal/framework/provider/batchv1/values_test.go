// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/types"
	sdkschema "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestLegacyBatchValue(t *testing.T) {
	ctx := context.Background()
	objectTypes := map[string]attr.Type{
		"text": types.StringType, "integer": types.Int64Type, "flag": types.BoolType,
		"list": types.ListType{ElemType: types.StringType},
		"set":  types.SetType{ElemType: types.StringType},
		"map":  types.MapType{ElemType: types.StringType},
	}
	value := types.ObjectValueMust(objectTypes, map[string]attr.Value{
		"text": types.StringUnknown(), "integer": types.Int64Null(), "flag": types.BoolNull(),
		"list": types.ListNull(types.StringType),
		"set":  types.SetValueMust(types.StringType, []attr.Value{types.StringValue("b"), types.StringValue("a")}),
		"map":  types.MapValueMust(types.StringType, map[string]attr.Value{"key": types.StringValue("value")}),
	})
	converted, diagnostics := legacyValue(ctx, value)
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	actual := converted.(map[string]interface{})
	if actual["text"] != "" || actual["integer"] != 0 || actual["flag"] != false {
		t.Fatalf("incorrect SDK primitive representation: %#v", actual)
	}
	if !reflect.DeepEqual(actual["list"], []interface{}{}) {
		t.Fatalf("incorrect omitted list representation: %#v", actual["list"])
	}
	if actual["set"].(*sdkschema.Set).Len() != 2 {
		t.Fatal("set elements lost")
	}
	if !reflect.DeepEqual(actual["map"], map[string]interface{}{"key": "value"}) {
		t.Fatal("map values lost")
	}
}

func batchValueTestBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{NestedObject: schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"text":  schema.StringAttribute{Optional: true},
			"count": schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(6)},
			"resources": schema.ListNestedAttribute{
				Optional: true, Computed: true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"limits":   schema.MapAttribute{Optional: true, Computed: true, ElementType: types.StringType},
					"requests": schema.MapAttribute{Optional: true, Computed: true, ElementType: types.StringType},
				}},
			},
		},
		Blocks: map[string]schema.Block{
			"nested": schema.ListNestedBlock{NestedObject: schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{"name": schema.StringAttribute{Required: true}},
			}},
		},
	}}
}

func TestBatchValueFromAPI(t *testing.T) {
	ctx := context.Background()
	block := batchValueTestBlock()
	value, diagnostics := valueFromAPI(ctx, block, []interface{}{map[string]interface{}{
		"text": "",
		"resources": []interface{}{map[string]interface{}{
			"limits": map[string]string{}, "requests": map[string]string{"memory": "64Mi"},
		}},
	}}, types.ListNull(block.NestedObject.Type()))
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	object := value.Elements()[0].(types.Object).Attributes()
	if !object["text"].IsNull() {
		t.Fatal("omitted optional string must not become an empty string")
	}
	if !object["count"].Equal(types.Int64Value(6)) {
		t.Fatal("API omission must preserve the schema's default")
	}
	if object["nested"].IsNull() || len(object["nested"].(types.List).Elements()) != 0 {
		t.Fatal("omitted block must have zero elements")
	}
	resources := object["resources"].(types.List).Elements()[0].(types.Object).Attributes()
	if !resources["limits"].IsNull() {
		t.Fatal("API empty map without a prior explicit map should stay null")
	}
	if resources["requests"].(types.Map).Elements()["memory"] != types.StringValue("64Mi") {
		t.Fatal("API-computed resource requests were lost")
	}
}

func TestBatchValuePreservesExplicitEmptyAndDetectsDrift(t *testing.T) {
	ctx := context.Background()
	block := batchValueTestBlock()
	prior, diagnostics := valueFromAPI(ctx, block, []interface{}{map[string]interface{}{"text": "before"}},
		types.ListNull(block.NestedObject.Type()))
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	updated, diagnostics := valueFromAPI(ctx, block, []interface{}{map[string]interface{}{"text": "after"}}, prior)
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	if updated.Elements()[0].(types.Object).Attributes()["text"] != types.StringValue("after") {
		t.Fatal("Read retained a stale configured value instead of detecting drift")
	}
	empty, diagnostics := valueFromAPI(ctx, block, []interface{}{map[string]interface{}{"text": ""}}, updated)
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	if empty.Elements()[0].(types.Object).Attributes()["text"] != types.StringValue("") {
		t.Fatal("explicit empty string was collapsed")
	}
}

func TestBatchValueRejectsInvalidAPIShape(t *testing.T) {
	ctx := context.Background()
	block := batchValueTestBlock()
	for _, raw := range []interface{}{
		map[string]interface{}{"count": "not-an-integer"},
		map[string]interface{}{"resources": "not-a-list"},
		map[string]interface{}{"resources": []interface{}{"not-an-object"}},
	} {
		_, diagnostics := valueFromAPI(ctx, block, []interface{}{raw}, types.ListNull(block.NestedObject.Type()))
		if !diagnostics.HasError() {
			t.Fatalf("invalid API shape accepted: %#v", raw)
		}
	}
}

func TestBatchNestedLegacyCollections(t *testing.T) {
	raw := []map[string]interface{}{{
		"values": sdkschema.NewSet(sdkschema.HashString, []interface{}{"zone-a", "zone-b"}),
	}}
	normalized := normalizeLegacyCollections(raw).([]interface{})
	values := normalized[0].(map[string]interface{})["values"].([]interface{})
	if len(values) != 2 {
		t.Fatalf("nested SDK set values lost: %#v", normalized)
	}
}

func TestBatchValuePreservesZeroCollectionElements(t *testing.T) {
	ctx := context.Background()
	block := schema.ListNestedBlock{NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
		"args":   schema.ListAttribute{Optional: true, ElementType: types.StringType},
		"groups": schema.SetAttribute{Optional: true, ElementType: types.Int64Type},
	}}}
	actual, diagnostics := valueFromAPI(ctx, block, []interface{}{map[string]interface{}{
		"args": []string{"", "value"}, "groups": []int{0, 1000},
	}}, types.ListNull(block.NestedObject.Type()))
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	fields := actual.Elements()[0].(types.Object).Attributes()
	if fields["args"].(types.List).Elements()[0] != types.StringValue("") {
		t.Fatal("empty argument must not be converted to null")
	}
	for _, group := range fields["groups"].(types.Set).Elements() {
		if group.IsNull() {
			t.Fatal("group zero must not be converted to null")
		}
	}
}

func TestBatchPlannedValueRejectsAPIDifferences(t *testing.T) {
	ctx := context.Background()
	block := batchValueTestBlock()
	plan, diagnostics := valueFromAPI(ctx, block, []interface{}{map[string]interface{}{
		"text": "configured",
	}}, types.ListNull(block.NestedObject.Type()))
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	api, diagnostics := valueFromAPI(ctx, block, []interface{}{map[string]interface{}{
		"text": "changed-by-admission",
	}}, plan)
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	if _, diagnostics := preservePlannedValue(ctx, block, plan, api); !diagnostics.HasError() {
		t.Fatal("genuine API changes must not be hidden by planned state")
	}
}

func TestBatchPlannedValueResolvesComputedParents(t *testing.T) {
	ctx := context.Background()
	block := batchValueTestBlock()
	actual, diagnostics := valueFromAPI(ctx, block, []interface{}{map[string]interface{}{
		"text":      "configured",
		"resources": []interface{}{map[string]interface{}{"limits": map[string]string{}, "requests": map[string]string{}}},
	}}, types.ListNull(block.NestedObject.Type()))
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	object := actual.Elements()[0].(types.Object)
	fields := object.Attributes()
	fields["resources"] = types.ListUnknown(fields["resources"].(types.List).ElementType(ctx))
	plan := types.ListValueMust(block.NestedObject.Type(), []attr.Value{
		types.ObjectValueMust(object.AttributeTypes(ctx), fields),
	})
	result, diagnostics := preservePlannedValue(ctx, block, plan, actual)
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	if !result.Equal(actual) {
		t.Fatalf("computed parent was not resolved: %s", result)
	}
}
