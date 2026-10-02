// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	framework "github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider"
	batchresource "github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/batchv1"
)

func cronJobPlanValue(t *testing.T, typ tftypes.Type, raw any) tftypes.Value {
	t.Helper()
	if raw == nil {
		return tftypes.NewValue(typ, nil)
	}
	switch typed := typ.(type) {
	case tftypes.Object:
		values := raw.(map[string]interface{})
		result := make(map[string]tftypes.Value, len(typed.AttributeTypes))
		for key, childType := range typed.AttributeTypes {
			result[key] = cronJobPlanValue(t, childType, values[key])
		}
		return tftypes.NewValue(typ, result)
	case tftypes.Map:
		values := raw.(map[string]interface{})
		result := make(map[string]tftypes.Value, len(values))
		for key, child := range values {
			result[key] = cronJobPlanValue(t, typed.ElementType, child)
		}
		return tftypes.NewValue(typ, result)
	case tftypes.List:
		values := raw.([]interface{})
		result := make([]tftypes.Value, len(values))
		for i, child := range values {
			result[i] = cronJobPlanValue(t, typed.ElementType, child)
		}
		return tftypes.NewValue(typ, result)
	case tftypes.Set:
		values := raw.([]interface{})
		result := make([]tftypes.Value, len(values))
		for i, child := range values {
			result[i] = cronJobPlanValue(t, typed.ElementType, child)
		}
		return tftypes.NewValue(typ, result)
	default:
		return tftypes.NewValue(typ, raw)
	}
}

func cronJobPlanBlock(t *testing.T, block *tfprotov6.SchemaBlock, raw map[string]interface{}) tftypes.Value {
	t.Helper()
	values := map[string]tftypes.Value{}
	for _, attribute := range block.Attributes {
		values[attribute.Name] = cronJobPlanValue(t, attribute.ValueType(), raw[attribute.Name])
	}
	for _, child := range block.BlockTypes {
		switch child.Nesting {
		case tfprotov6.SchemaNestedBlockNestingModeList, tfprotov6.SchemaNestedBlockNestingModeSet:
			items, _ := raw[child.TypeName].([]interface{})
			converted := make([]tftypes.Value, len(items))
			for i, item := range items {
				converted[i] = cronJobPlanBlock(t, child.Block, item.(map[string]interface{}))
			}
			values[child.TypeName] = tftypes.NewValue(child.ValueType(), converted)
		case tfprotov6.SchemaNestedBlockNestingModeSingle:
			object, _ := raw[child.TypeName].(map[string]interface{})
			if object == nil {
				values[child.TypeName] = tftypes.NewValue(child.ValueType(), nil)
			} else {
				values[child.TypeName] = cronJobPlanBlock(t, child.Block, object)
			}
		default:
			t.Fatalf("unhandled block nesting %s", child.Nesting)
		}
	}
	return tftypes.NewValue(block.ValueType(), values)
}

func cronJobPlanConfig() map[string]interface{} {
	return map[string]interface{}{
		"metadata": []interface{}{map[string]interface{}{"name": "cron-plan"}},
		"spec": []interface{}{map[string]interface{}{
			"schedule": "@hourly",
			"job_template": []interface{}{map[string]interface{}{
				"metadata": []interface{}{map[string]interface{}{}},
				"spec": []interface{}{map[string]interface{}{
					"template": []interface{}{map[string]interface{}{
						"metadata": []interface{}{map[string]interface{}{}},
						"spec": []interface{}{map[string]interface{}{
							"container": []interface{}{map[string]interface{}{"name": "task", "image": "busybox:1.36"}},
						}},
					}},
				}},
			}},
		}},
	}
}

func cronJobPlanDiagnostics(t *testing.T, diagnostics []*tfprotov6.Diagnostic) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("%s: %s (%s)", diagnostic.Summary, diagnostic.Detail, diagnostic.Attribute)
		}
	}
}

func cronJobKnownPlan(t *testing.T, value tftypes.Value) tftypes.Value {
	t.Helper()
	if value.IsNull() {
		return value
	}
	if !value.IsKnown() {
		switch value.Type().(type) {
		case tftypes.List, tftypes.Set:
			return tftypes.NewValue(value.Type(), []tftypes.Value{})
		case tftypes.Map:
			return tftypes.NewValue(value.Type(), map[string]tftypes.Value{})
		}
		switch {
		case value.Type().Is(tftypes.String):
			return tftypes.NewValue(value.Type(), "")
		case value.Type().Is(tftypes.Number):
			return tftypes.NewValue(value.Type(), int64(0))
		case value.Type().Is(tftypes.Bool):
			return tftypes.NewValue(value.Type(), false)
		default:
			return tftypes.NewValue(value.Type(), nil)
		}
	}
	switch value.Type().(type) {
	case tftypes.Object, tftypes.Map:
		var fields map[string]tftypes.Value
		if err := value.As(&fields); err != nil {
			t.Fatal(err)
		}
		for key, child := range fields {
			fields[key] = cronJobKnownPlan(t, child)
		}
		return tftypes.NewValue(value.Type(), fields)
	case tftypes.List, tftypes.Set:
		var elements []tftypes.Value
		if err := value.As(&elements); err != nil {
			t.Fatal(err)
		}
		for i, child := range elements {
			elements[i] = cronJobKnownPlan(t, child)
		}
		return tftypes.NewValue(value.Type(), elements)
	default:
		return value
	}
}

func cronJobProposedPlan(t *testing.T, block *tfprotov6.SchemaBlock, config, prior tftypes.Value) tftypes.Value {
	t.Helper()
	var configured, previous map[string]tftypes.Value
	if err := config.As(&configured); err != nil {
		t.Fatal(err)
	}
	if err := prior.As(&previous); err != nil {
		t.Fatal(err)
	}
	for _, attribute := range block.Attributes {
		if attribute.Computed && configured[attribute.Name].IsNull() {
			configured[attribute.Name] = previous[attribute.Name]
		}
	}
	for _, child := range block.BlockTypes {
		if child.Nesting != tfprotov6.SchemaNestedBlockNestingModeList {
			continue
		}
		var configuredItems, previousItems []tftypes.Value
		if err := configured[child.TypeName].As(&configuredItems); err != nil {
			t.Fatal(err)
		}
		if err := previous[child.TypeName].As(&previousItems); err != nil {
			t.Fatal(err)
		}
		for i, item := range configuredItems {
			if i < len(previousItems) {
				configuredItems[i] = cronJobProposedPlan(t, child.Block, item, previousItems[i])
			}
		}
		configured[child.TypeName] = tftypes.NewValue(child.ValueType(), configuredItems)
	}
	return tftypes.NewValue(block.ValueType(), configured)
}

func TestCronJobProtocolUpdatePlans(t *testing.T) {
	ctx := context.Background()
	var resourceSchema fwresource.SchemaResponse
	batchresource.NewCronJobV1().Schema(ctx, fwresource.SchemaRequest{}, &resourceSchema)
	server := providerserver.NewProtocol6(framework.New("test", nil))()
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	cronJobPlanDiagnostics(t, schemas.Diagnostics)
	s := schemas.ResourceSchemas["kubernetes_cron_job_v1"]
	oldConfig := cronJobPlanConfig()
	oldSpec := oldConfig["spec"].([]interface{})[0].(map[string]interface{})
	oldSpec["failed_jobs_history_limit"] = int64(5)
	oldSpec["successful_jobs_history_limit"] = int64(10)
	oldSpec["starting_deadline_seconds"] = int64(20)
	oldSpec["timezone"] = "Etc/UTC"
	oldSpec["suspend"] = true
	configValue := cronJobPlanBlock(t, s.Block, oldConfig)
	oldDynamic, err := tfprotov6.NewDynamicValue(s.ValueType(), configValue)
	if err != nil {
		t.Fatal(err)
	}
	nullDynamic, err := tfprotov6.NewDynamicValue(s.ValueType(), tftypes.NewValue(s.ValueType(), nil))
	if err != nil {
		t.Fatal(err)
	}
	create, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
		TypeName: "kubernetes_cron_job_v1", Config: &oldDynamic,
		PriorState: &nullDynamic, ProposedNewState: &oldDynamic,
	})
	if err != nil {
		t.Fatal(err)
	}
	cronJobPlanDiagnostics(t, create.Diagnostics)
	unresolved, err := create.PlannedState.Unmarshal(s.ValueType())
	if err != nil {
		t.Fatal(err)
	}
	previous := cronJobKnownPlan(t, unresolved)
	previousDynamic, err := tfprotov6.NewDynamicValue(s.ValueType(), previous)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name        string
		change      func(map[string]interface{})
		wantReplace bool
	}{
		{name: "remove defaults"},
		{name: "schedule", change: func(spec map[string]interface{}) { spec["schedule"] = "@daily" }},
		{name: "mutable container image", change: func(spec map[string]interface{}) {
			template := spec["job_template"].([]interface{})[0].(map[string]interface{})
			jobSpec := template["spec"].([]interface{})[0].(map[string]interface{})
			podTemplate := jobSpec["template"].([]interface{})[0].(map[string]interface{})
			podSpec := podTemplate["spec"].([]interface{})[0].(map[string]interface{})
			container := podSpec["container"].([]interface{})[0].(map[string]interface{})
			container["image"] = "busybox:1.37"
		}},
		{name: "mutable container environment", change: func(spec map[string]interface{}) {
			template := spec["job_template"].([]interface{})[0].(map[string]interface{})
			jobSpec := template["spec"].([]interface{})[0].(map[string]interface{})
			podTemplate := jobSpec["template"].([]interface{})[0].(map[string]interface{})
			podSpec := podTemplate["spec"].([]interface{})[0].(map[string]interface{})
			container := podSpec["container"].([]interface{})[0].(map[string]interface{})
			container["env"] = []interface{}{map[string]interface{}{"name": "MODE", "value": "updated"}}
		}},
		{name: "immutable completions", wantReplace: true, change: func(spec map[string]interface{}) {
			template := spec["job_template"].([]interface{})[0].(map[string]interface{})
			jobSpec := template["spec"].([]interface{})[0].(map[string]interface{})
			jobSpec["completions"] = int64(2)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := cronJobPlanConfig()
			spec := config["spec"].([]interface{})[0].(map[string]interface{})
			if test.change != nil {
				test.change(spec)
			}
			current := cronJobPlanBlock(t, s.Block, config)
			currentDynamic, err := tfprotov6.NewDynamicValue(s.ValueType(), current)
			if err != nil {
				t.Fatal(err)
			}
			proposed := cronJobProposedPlan(t, s.Block, current, previous)
			proposedDynamic, err := tfprotov6.NewDynamicValue(s.ValueType(), proposed)
			if err != nil {
				t.Fatal(err)
			}
			response, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
				TypeName: "kubernetes_cron_job_v1", Config: &currentDynamic,
				PriorState: &previousDynamic, ProposedNewState: &proposedDynamic,
			})
			if err != nil {
				t.Fatal(err)
			}
			cronJobPlanDiagnostics(t, response.Diagnostics)
			if (len(response.RequiresReplace) > 0) != test.wantReplace {
				t.Fatalf("replacement paths: %v; want replacement=%t", response.RequiresReplace, test.wantReplace)
			}
			planned, err := response.PlannedState.Unmarshal(s.ValueType())
			if err != nil {
				t.Fatal(err)
			}
			jobTemplatePath := path.Root("spec").AtListIndex(0).AtName("job_template").AtListIndex(0)
			for _, metadataPath := range []path.Path{
				path.Root("metadata"),
				jobTemplatePath.AtName("metadata"),
				jobTemplatePath.AtName("spec").AtListIndex(0).AtName("template").AtListIndex(0).AtName("metadata"),
			} {
				var labels types.Map
				plannedState := tfsdk.State{Schema: resourceSchema.Schema, Raw: planned}
				if d := plannedState.GetAttribute(ctx, metadataPath.AtListIndex(0).AtName("labels"), &labels); d.HasError() {
					t.Fatal(d)
				}
				if !labels.IsNull() {
					t.Errorf("omitted %s labels must stay null, not computed unknown: %s", metadataPath, labels)
				}
			}
			var root map[string]tftypes.Value
			if err := planned.As(&root); err != nil {
				t.Fatal(err)
			}
			var specs []tftypes.Value
			if err := root["spec"].As(&specs); err != nil {
				t.Fatal(err)
			}
			var fields map[string]tftypes.Value
			if err := specs[0].As(&fields); err != nil {
				t.Fatal(err)
			}
			for key, want := range map[string]interface{}{
				"failed_jobs_history_limit": int64(1), "successful_jobs_history_limit": int64(3),
				"starting_deadline_seconds": int64(0), "timezone": "", "suspend": false,
			} {
				if !fields[key].Equal(tftypes.NewValue(fields[key].Type(), want)) {
					t.Errorf("removed %s planned %s, want %v", key, fields[key], want)
				}
			}
		})
	}
}

func TestCronJobProtocolPlanDefaultsAndValidation(t *testing.T) {
	ctx := context.Background()
	server := providerserver.NewProtocol6(framework.New("test", nil))()
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	cronJobPlanDiagnostics(t, schemas.Diagnostics)
	resourceSchema := schemas.ResourceSchemas["kubernetes_cron_job_v1"]
	if resourceSchema == nil {
		t.Fatal("CronJob not registered")
	}
	for _, test := range []struct {
		name    string
		change  func(map[string]interface{})
		valid   bool
		unknown bool
	}{
		{name: "defaults", valid: true},
		{name: "zero limits", valid: true, change: func(spec map[string]interface{}) {
			spec["failed_jobs_history_limit"] = int64(0)
			spec["successful_jobs_history_limit"] = int64(0)
		}},
		{name: "unknown schedule", valid: true, unknown: true, change: func(spec map[string]interface{}) {
			spec["schedule"] = tftypes.UnknownValue
		}},
		{name: "invalid schedule", change: func(spec map[string]interface{}) { spec["schedule"] = "not a cron" }},
		{name: "missing template", change: func(spec map[string]interface{}) { delete(spec, "job_template") }},
		{name: "invalid concurrency", change: func(spec map[string]interface{}) { spec["concurrency_policy"] = "Other" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := cronJobPlanConfig()
			spec := config["spec"].([]interface{})[0].(map[string]interface{})
			if test.change != nil {
				test.change(spec)
			}
			value := cronJobPlanBlock(t, resourceSchema.Block, config)
			dynamic, err := tfprotov6.NewDynamicValue(value.Type(), value)
			if err != nil {
				t.Fatal(err)
			}
			validation, err := server.ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{
				TypeName: "kubernetes_cron_job_v1", Config: &dynamic,
			})
			if err != nil {
				t.Fatal(err)
			}
			hasError := false
			for _, diagnostic := range validation.Diagnostics {
				hasError = hasError || diagnostic.Severity == tfprotov6.DiagnosticSeverityError
			}
			if hasError != !test.valid {
				t.Fatalf("validation diagnostics: %v; valid=%t", validation.Diagnostics, test.valid)
			}
			if !test.valid {
				return
			}
			prior, err := tfprotov6.NewDynamicValue(value.Type(), tftypes.NewValue(value.Type(), nil))
			if err != nil {
				t.Fatal(err)
			}
			planned, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
				TypeName: "kubernetes_cron_job_v1", Config: &dynamic,
				PriorState: &prior, ProposedNewState: &dynamic,
			})
			if err != nil {
				t.Fatal(err)
			}
			cronJobPlanDiagnostics(t, planned.Diagnostics)
			plan, err := planned.PlannedState.Unmarshal(value.Type())
			if err != nil {
				t.Fatal(err)
			}
			var root map[string]tftypes.Value
			if err := plan.As(&root); err != nil {
				t.Fatal(err)
			}
			var specs []tftypes.Value
			if err := root["spec"].As(&specs); err != nil {
				t.Fatal(err)
			}
			var fields map[string]tftypes.Value
			if err := specs[0].As(&fields); err != nil {
				t.Fatal(err)
			}
			expected := map[string]interface{}{
				"concurrency_policy": "Allow", "failed_jobs_history_limit": int64(1),
				"successful_jobs_history_limit": int64(3), "starting_deadline_seconds": int64(0),
				"timezone": "", "suspend": false,
			}
			if test.name == "zero limits" {
				expected["failed_jobs_history_limit"] = int64(0)
				expected["successful_jobs_history_limit"] = int64(0)
			}
			for key, want := range expected {
				if !fields[key].Equal(tftypes.NewValue(fields[key].Type(), want)) {
					t.Errorf("%s plan = %s, want %v", key, fields[key], want)
				}
			}
			if test.unknown && fields["schedule"].IsKnown() {
				t.Fatal("unknown schedule was prematurely evaluated")
			}
		})
	}
}
