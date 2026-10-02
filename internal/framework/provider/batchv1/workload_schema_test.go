// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	sdkschema "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

func legacyWorkloadTemplate(t *testing.T, updatable bool) *sdkschema.Schema {
	t.Helper()
	resources := kubernetes.Provider().ResourcesMap
	if updatable {
		fields := resources["kubernetes_cron_job"].Schema
		for _, name := range []string{"spec", "job_template", "spec"} {
			fields = fields[name].Elem.(*sdkschema.Resource).Schema
		}
		return fields["template"]
	}
	return resources["kubernetes_job"].Schema["spec"].Elem.(*sdkschema.Resource).Schema["template"]
}

func TestWorkloadSchemaLegacyContract(t *testing.T) {
	for _, updatable := range []bool{false, true} {
		t.Run(fmt.Sprintf("updatable=%t", updatable), func(t *testing.T) {
			ctx := context.Background()
			actual := schema.Schema{
				Attributes: map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}},
				Blocks:     map[string]schema.Block{"template": podTemplateBlock(updatable)},
			}
			if diags := actual.ValidateImplementation(ctx); diags.HasError() {
				t.Fatalf("invalid Framework schema: %s", diags)
			}
			legacy := legacyWorkloadTemplate(t, updatable)
			oldSchema := &sdkschema.Resource{Schema: map[string]*sdkschema.Schema{"template": legacy}}
			oldJSON, err := oldSchema.CoreConfigSchema().ImpliedType().MarshalJSON()
			if err != nil {
				t.Fatal(err)
			}
			oldType, err := tftypes.ParseJSONType(oldJSON)
			if err != nil {
				t.Fatal(err)
			}
			if newType := actual.Type().TerraformType(ctx); !newType.Equal(oldType) {
				t.Error("persisted workload type changed")
			}
			compareWorkloadField(t, "template", legacy, podTemplateBlock(updatable), false)
		})
	}
}

func compareWorkloadField(t *testing.T, fieldPath string, legacy *sdkschema.Schema, actual any, attributesOnly bool) {
	t.Helper()
	if actual == nil {
		t.Errorf("%s missing from Framework schema", fieldPath)
		return
	}
	value := reflect.ValueOf(actual)
	modifiers := value.FieldByName("PlanModifiers")
	forceNew := legacy.ForceNew
	hasReplacement := modifiers.IsValid() && modifiers.Len() > 0
	if hasReplacement {
		switch modifier := modifiers.Index(0).Interface().(type) {
		case workloadQuantityStringPlanModifier:
			hasReplacement = modifier.requiresReplace
		case workloadQuantityMapPlanModifier:
			hasReplacement = modifier.requiresReplace
		}
	}
	if got := hasReplacement; got != forceNew {
		t.Errorf("%s replacement = %t, want %t", fieldPath, got, forceNew)
	}
	var attrs map[string]schema.Attribute
	var blocks map[string]schema.Block
	switch modern := actual.(type) {
	case schema.ListNestedBlock:
		if legacy.Type != sdkschema.TypeList || legacy.Computed || attributesOnly {
			t.Errorf("%s unexpectedly remains a list block", fieldPath)
		}
		attrs, blocks = modern.NestedObject.Attributes, modern.NestedObject.Blocks
	case schema.SetNestedBlock:
		if legacy.Type != sdkschema.TypeSet || legacy.Computed || attributesOnly {
			t.Errorf("%s unexpectedly remains a set block", fieldPath)
		}
		attrs, blocks = modern.NestedObject.Attributes, modern.NestedObject.Blocks
	case schema.ListNestedAttribute:
		if legacy.Type != sdkschema.TypeList || (!legacy.Computed && !attributesOnly) {
			t.Errorf("%s unnecessarily changed from a block to an attribute", fieldPath)
		}
		attrs, attributesOnly = modern.NestedObject.Attributes, true
	case schema.SetNestedAttribute:
		if legacy.Type != sdkschema.TypeSet || (!legacy.Computed && !attributesOnly) {
			t.Errorf("%s unnecessarily changed from a block to an attribute", fieldPath)
		}
		attrs, attributesOnly = modern.NestedObject.Attributes, true
	}
	if modern, ok := actual.(schema.Attribute); ok {
		if modern.IsRequired() != legacy.Required || modern.IsOptional() != legacy.Optional ||
			modern.IsComputed() != (legacy.Computed || legacy.Default != nil) || modern.IsSensitive() != legacy.Sensitive {
			t.Errorf("%s flags differ: required=%t optional=%t computed=%t sensitive=%t", fieldPath,
				modern.IsRequired(), modern.IsOptional(), modern.IsComputed(), modern.IsSensitive())
		}
		compareWorkloadDefault(t, fieldPath, legacy, actual)
		compareWorkloadScalarValidators(t, fieldPath, legacy, actual)
	}
	validators := value.FieldByName("Validators")
	if legacy.ValidateFunc != nil && (!validators.IsValid() || validators.Len() == 0) {
		t.Errorf("%s lost its validator", fieldPath)
	}
	if child, ok := legacy.Elem.(*sdkschema.Resource); ok {
		if len(attrs)+len(blocks) != len(child.Schema) {
			t.Errorf("%s field count = %d, want %d", fieldPath, len(attrs)+len(blocks), len(child.Schema))
		}
		for name, field := range child.Schema {
			var next any = attrs[name]
			if next == nil {
				next = blocks[name]
			}
			compareWorkloadField(t, fieldPath+"."+name, field, next, attributesOnly)
		}
	}
	if legacy.Type == sdkschema.TypeList || legacy.Type == sdkschema.TypeSet {
		compareWorkloadCardinality(t, fieldPath, legacy, actual)
		compareWorkloadElementValidators(t, fieldPath, legacy, actual)
	}
	if legacy.DiffSuppressFunc != nil {
		switch modern := actual.(type) {
		case schema.StringAttribute:
			if _, ok := modern.CustomType.(workloadQuantityType); !ok {
				t.Errorf("%s lost quantity semantic equality", fieldPath)
			}
		case schema.MapAttribute:
			if _, ok := modern.ElementType.(workloadQuantityType); !ok {
				t.Errorf("%s lost quantity map element semantic equality", fieldPath)
			}
		default:
			t.Errorf("%s unsupported DiffSuppress mapping %T", fieldPath, actual)
		}
	}
}

func compareWorkloadDefault(t *testing.T, fieldPath string, legacy *sdkschema.Schema, actual any) {
	t.Helper()
	defaultField := reflect.ValueOf(actual).FieldByName("Default")
	hasDefault := defaultField.IsValid() && !defaultField.IsNil()
	if hasDefault != (legacy.Default != nil) {
		t.Errorf("%s default presence = %t, want %t", fieldPath, hasDefault, legacy.Default != nil)
	}
	if !hasDefault {
		return
	}
	var got any
	switch modern := actual.(type) {
	case schema.StringAttribute:
		var resp defaults.StringResponse
		modern.Default.DefaultString(context.Background(), defaults.StringRequest{}, &resp)
		got = resp.PlanValue.ValueString()
	case schema.Int64Attribute:
		var resp defaults.Int64Response
		modern.Default.DefaultInt64(context.Background(), defaults.Int64Request{}, &resp)
		got = int(resp.PlanValue.ValueInt64())
	case schema.BoolAttribute:
		var resp defaults.BoolResponse
		modern.Default.DefaultBool(context.Background(), defaults.BoolRequest{}, &resp)
		got = resp.PlanValue.ValueBool()
	default:
		t.Fatalf("%s untested default type %T", fieldPath, actual)
	}
	if !reflect.DeepEqual(got, legacy.Default) {
		t.Errorf("%s default = %#v, want %#v", fieldPath, got, legacy.Default)
	}
}

func compareWorkloadScalarValidators(t *testing.T, fieldPath string, legacy *sdkschema.Schema, actual any) {
	t.Helper()
	if legacy.ValidateFunc == nil {
		return
	}
	ctx := context.Background()
	switch modern := actual.(type) {
	case schema.StringAttribute:
		for _, text := range []string{
			"", "0", "1", "-1", "0644", "0777", "01000", "0789", "2147483648", "9223372036854775808",
			"http", "80", "65536", "1234567890123456", "1.5Gi", "1024Mi", "garbage",
			"localhost", "example.com", "a-", "a.b", "a_b", "A", "127.0.0.1", "::1", "012.1.1.1",
			"Never", "Always", "OnFailure", "ClusterFirst", "ClusterFirstWithHostNet", "Default", "None",
			"HTTP", "HTTPS", "TCP", "UDP", "SCTP", "linux", "windows", "HugePages", "HugePages-2Mi",
			"Unconfined", "Localhost", "RuntimeDefault", "DirectoryOrCreate", "File", "Socket",
			"Honor", "Ignore", "ReadWriteOncePod", "Exists", "In", "DoNotSchedule", "ScheduleAnyway",
			"relative/path", "/absolute/path", "one/../two", "..hidden", "./relative", "0644 ",
		} {
			_, oldErrors := legacy.ValidateFunc(text, fieldPath)
			var response validator.StringResponse
			for _, rule := range modern.Validators {
				if strings.Contains(fmt.Sprintf("%T", rule), "ConflictsWith") {
					continue
				}
				rule.ValidateString(ctx, validator.StringRequest{Path: path.Root(fieldPath), ConfigValue: types.StringValue(text)}, &response)
			}
			if response.Diagnostics.HasError() != (len(oldErrors) > 0) {
				t.Errorf("%s validation differs for %q: old=%v new=%s", fieldPath, text, oldErrors, response.Diagnostics)
			}
		}
	case schema.Int64Attribute:
		for _, number := range []int{0, 1, -1, 599, 600, 601, 65535, 65536} {
			_, oldErrors := legacy.ValidateFunc(number, fieldPath)
			var response validator.Int64Response
			for _, rule := range modern.Validators {
				rule.ValidateInt64(ctx, validator.Int64Request{Path: path.Root(fieldPath), ConfigValue: types.Int64Value(int64(number))}, &response)
			}
			if response.Diagnostics.HasError() != (len(oldErrors) > 0) {
				t.Errorf("%s validation differs for %d: old=%v new=%s", fieldPath, number, oldErrors, response.Diagnostics)
			}
		}
	}
}

func compareWorkloadCardinality(t *testing.T, fieldPath string, legacy *sdkschema.Schema, actual any) {
	t.Helper()
	minimum := legacy.MinItems
	if legacy.Required && minimum == 0 {
		minimum = 1
	}
	// Run only cardinality validators; element validators have separate coverage.
	reflected := reflect.ValueOf(actual).FieldByName("Validators")
	if !reflected.IsValid() {
		t.Fatalf("%s has no validator field", fieldPath)
	}
	for _, count := range []int{0, 1, 2, 256} {
		wantError := count < minimum || (legacy.MaxItems > 0 && count > legacy.MaxItems)
		elements := make([]attr.Value, count)
		for index := range elements {
			elements[index] = types.StringValue(fmt.Sprint(index))
		}
		gotError := false
		for i := 0; i < reflected.Len(); i++ {
			rule := reflected.Index(i).Interface()
			if !strings.Contains(fmt.Sprintf("%T", rule), "size") {
				continue
			}
			switch rule := rule.(type) {
			case validator.List:
				var response validator.ListResponse
				rule.ValidateList(context.Background(), validator.ListRequest{
					Path: path.Root(fieldPath), ConfigValue: types.ListValueMust(types.StringType, elements),
				}, &response)
				gotError = gotError || response.Diagnostics.HasError()
			case validator.Set:
				var response validator.SetResponse
				rule.ValidateSet(context.Background(), validator.SetRequest{
					Path: path.Root(fieldPath), ConfigValue: types.SetValueMust(types.StringType, elements),
				}, &response)
				gotError = gotError || response.Diagnostics.HasError()
			}
		}
		if gotError != wantError {
			t.Errorf("%s cardinality %d error=%t, want %t", fieldPath, count, gotError, wantError)
		}
	}
}

func compareWorkloadElementValidators(t *testing.T, fieldPath string, legacy *sdkschema.Schema, actual any) {
	t.Helper()
	element, ok := legacy.Elem.(*sdkschema.Schema)
	if !ok || element.ValidateFunc == nil {
		return
	}
	reflected := reflect.ValueOf(actual).FieldByName("Validators")
	for _, text := range []string{"", "localhost", "example.com", "127.0.0.1", "::1", "a_", "ReadWriteOnce", "ReadOnlyMany", "ReadWriteMany", "ReadWriteOncePod", "invalid"} {
		_, oldErrors := element.ValidateFunc(text, fieldPath)
		gotError := false
		for i := 0; i < reflected.Len(); i++ {
			rule := reflected.Index(i).Interface()
			if !strings.Contains(fmt.Sprintf("%T", rule), "valueStrings") {
				continue
			}
			switch rule := rule.(type) {
			case validator.List:
				var response validator.ListResponse
				rule.ValidateList(context.Background(), validator.ListRequest{
					Path: path.Root(fieldPath), ConfigValue: types.ListValueMust(types.StringType, []attr.Value{types.StringValue(text)}),
				}, &response)
				gotError = gotError || response.Diagnostics.HasError()
			case validator.Set:
				var response validator.SetResponse
				rule.ValidateSet(context.Background(), validator.SetRequest{
					Path: path.Root(fieldPath), ConfigValue: types.SetValueMust(types.StringType, []attr.Value{types.StringValue(text)}),
				}, &response)
				gotError = gotError || response.Diagnostics.HasError()
			}
		}
		if gotError != (len(oldErrors) > 0) {
			t.Errorf("%s element validation differs for %q: error=%t, want %t", fieldPath, text, gotError, len(oldErrors) > 0)
		}
	}
}
