// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
)

const workloadReplacementDescription = "Requires replacement when a configured workload value changes."

func workloadStringRequiresReplace() planmodifier.String {
	return stringplanmodifier.RequiresReplaceIf(func(ctx context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
		resp.RequiresReplace = !(req.ConfigValue.IsNull() && req.PlanValue.IsUnknown()) &&
			!workloadTemplateEmptyNormalization(ctx, req.Path, req.Config, req.Plan, req.State)
	}, workloadReplacementDescription, workloadReplacementDescription)
}

func workloadInt64RequiresReplace() planmodifier.Int64 {
	return int64planmodifier.RequiresReplaceIf(func(ctx context.Context, req planmodifier.Int64Request, resp *int64planmodifier.RequiresReplaceIfFuncResponse) {
		resp.RequiresReplace = !(req.ConfigValue.IsNull() && req.PlanValue.IsUnknown()) &&
			!workloadTemplateEmptyNormalization(ctx, req.Path, req.Config, req.Plan, req.State)
	}, workloadReplacementDescription, workloadReplacementDescription)
}

func workloadBoolRequiresReplace() planmodifier.Bool {
	return boolplanmodifier.RequiresReplaceIf(func(ctx context.Context, req planmodifier.BoolRequest, resp *boolplanmodifier.RequiresReplaceIfFuncResponse) {
		resp.RequiresReplace = !(req.ConfigValue.IsNull() && req.PlanValue.IsUnknown()) &&
			!workloadTemplateEmptyNormalization(ctx, req.Path, req.Config, req.Plan, req.State)
	}, workloadReplacementDescription, workloadReplacementDescription)
}

func workloadListRequiresReplace() planmodifier.List {
	return listplanmodifier.RequiresReplaceIf(func(ctx context.Context, req planmodifier.ListRequest, resp *listplanmodifier.RequiresReplaceIfFuncResponse) {
		resp.RequiresReplace = workloadPlannedChange(ctx, req.PlanValue, req.StateValue, req.ConfigValue) &&
			!workloadTemplateEmptyNormalization(ctx, req.Path, req.Config, req.Plan, req.State)
	}, workloadReplacementDescription, workloadReplacementDescription)
}

// SDKv2 diffList applies an object-list's ForceNew to its count, not its
// descendants. Child fields retain their own replacement rules.
func workloadObjectListRequiresReplace() planmodifier.List {
	return listplanmodifier.RequiresReplaceIf(func(_ context.Context, req planmodifier.ListRequest, resp *listplanmodifier.RequiresReplaceIfFuncResponse) {
		if req.PlanValue.IsUnknown() {
			resp.RequiresReplace = !req.ConfigValue.IsNull()
			return
		}
		resp.RequiresReplace = len(req.PlanValue.Elements()) != len(req.StateValue.Elements())
	}, workloadReplacementDescription, workloadReplacementDescription)
}

func workloadTemplateListRequiresReplace(updatable bool) planmodifier.List {
	if !updatable {
		return workloadListRequiresReplace()
	}
	return workloadObjectListRequiresReplace()
}

func workloadSetRequiresReplace() planmodifier.Set {
	return setplanmodifier.RequiresReplaceIf(func(ctx context.Context, req planmodifier.SetRequest, resp *setplanmodifier.RequiresReplaceIfFuncResponse) {
		resp.RequiresReplace = !(req.ConfigValue.IsNull() && req.PlanValue.IsUnknown()) &&
			!workloadTemplateEmptyNormalization(ctx, req.Path, req.Config, req.Plan, req.State)
	}, workloadReplacementDescription, workloadReplacementDescription)
}

func workloadMapRequiresReplace() planmodifier.Map {
	return mapplanmodifier.RequiresReplaceIf(func(ctx context.Context, req planmodifier.MapRequest, resp *mapplanmodifier.RequiresReplaceIfFuncResponse) {
		resp.RequiresReplace = workloadPlannedChange(ctx, req.PlanValue, req.StateValue, req.ConfigValue) &&
			!workloadTemplateEmptyNormalization(ctx, req.Path, req.Config, req.Plan, req.State)
	}, workloadReplacementDescription, workloadReplacementDescription)
}

func workloadStringPlanModifiers(updatable bool) []planmodifier.String {
	if updatable {
		return nil
	}
	return []planmodifier.String{workloadStringRequiresReplace()}
}

func workloadInt64PlanModifiers(updatable bool) []planmodifier.Int64 {
	if updatable {
		return nil
	}
	return []planmodifier.Int64{workloadInt64RequiresReplace()}
}

func workloadBoolPlanModifiers(updatable bool) []planmodifier.Bool {
	if updatable {
		return nil
	}
	return []planmodifier.Bool{workloadBoolRequiresReplace()}
}

func workloadListPlanModifiers(updatable bool) []planmodifier.List {
	if updatable {
		return nil
	}
	return []planmodifier.List{workloadListRequiresReplace()}
}

func workloadObjectListPlanModifiers(updatable bool) []planmodifier.List {
	if updatable {
		return nil
	}
	return []planmodifier.List{workloadObjectListRequiresReplace()}
}

func workloadSetPlanModifiers(updatable bool) []planmodifier.Set {
	if updatable {
		return nil
	}
	return []planmodifier.Set{workloadSetRequiresReplace()}
}

func workloadMapPlanModifiers(updatable bool) []planmodifier.Map {
	if updatable {
		return nil
	}
	return []planmodifier.Map{workloadMapRequiresReplace()}
}

// Parent blocks are checked before their children. Ignore unknowns introduced
// for unconfigured Computed fields, and quantities whose spelling alone changed.
// Configured unknowns and null/empty differences still require replacement.
func workloadPlannedChange(ctx context.Context, plan, state, config attr.Value) bool {
	if plan.Equal(state) || (plan.IsUnknown() && config.IsNull()) {
		return false
	}
	if plan.IsNull() || plan.IsUnknown() || state.IsNull() || state.IsUnknown() {
		return true
	}
	switch planned := plan.(type) {
	case workloadQuantityValue:
		prior, ok := state.(workloadQuantityValue)
		if !ok {
			return true
		}
		equal, diags := planned.StringSemanticEquals(ctx, prior)
		return diags.HasError() || !equal
	case types.List:
		prior, ok := state.(types.List)
		if !ok || len(planned.Elements()) != len(prior.Elements()) {
			return true
		}
		configured, ok := config.(types.List)
		if !ok {
			return true
		}
		plannedValues, priorValues, configValues := planned.Elements(), prior.Elements(), configured.Elements()
		for i, child := range plannedValues {
			childConfig := workloadUnconfiguredChild(ctx, config, child)
			if i < len(configValues) {
				childConfig = configValues[i]
			}
			if workloadPlannedChange(ctx, child, priorValues[i], childConfig) {
				return true
			}
		}
		return false
	case types.Object:
		prior, ok := state.(types.Object)
		if !ok {
			return true
		}
		configured, ok := config.(types.Object)
		if !ok {
			return true
		}
		return workloadPlannedFieldsChange(ctx, planned.Attributes(), prior.Attributes(), configured.Attributes(), config)
	case types.Map:
		prior, ok := state.(types.Map)
		if !ok {
			return true
		}
		configured, ok := config.(types.Map)
		if !ok {
			return true
		}
		return workloadPlannedFieldsChange(ctx, planned.Elements(), prior.Elements(), configured.Elements(), config)
	}
	return true
}

func workloadPlannedFieldsChange(ctx context.Context, plan, state, config map[string]attr.Value, parentConfig attr.Value) bool {
	if len(plan) != len(state) {
		return true
	}
	for name, child := range plan {
		prior, ok := state[name]
		if !ok {
			return true
		}
		configured, ok := config[name]
		if !ok {
			configured = workloadUnconfiguredChild(ctx, parentConfig, child)
		}
		if workloadPlannedChange(ctx, child, prior, configured) {
			return true
		}
	}
	return false
}

func workloadUnconfiguredChild(ctx context.Context, parent, child attr.Value) attr.Value {
	var raw any
	if parent.IsUnknown() {
		raw = tftypes.UnknownValue
	}
	typ := child.Type(ctx)
	value, err := typ.ValueFromTerraform(ctx, tftypes.NewValue(typ.TerraformType(ctx), raw))
	if err != nil {
		return child
	}
	return value
}

// SDKv2 persisted empty values for unconfigured fields. A transition to null is
// state-only only when the complete pod template expands to the same API value.
// The comparison copy resolves computed unknowns; the real plan is never changed.
func workloadTemplateEmptyNormalization(ctx context.Context, changed path.Path, config tfsdk.Config, plan tfsdk.Plan, state tfsdk.State) bool {
	if config.Schema == nil || plan.Schema == nil || state.Schema == nil {
		return false
	}
	templatePath := changed
	for len(templatePath.Steps()) > 0 {
		last, _ := templatePath.Steps().LastStep()
		if name, ok := last.(path.PathStepAttributeName); ok && name == "template" {
			break
		}
		templatePath = templatePath.ParentPath()
	}
	if len(templatePath.Steps()) == 0 {
		return false
	}
	var configured, planned, prior types.List
	if config.GetAttribute(ctx, templatePath, &configured).HasError() ||
		plan.GetAttribute(ctx, templatePath, &planned).HasError() ||
		state.GetAttribute(ctx, templatePath, &prior).HasError() {
		return false
	}
	comparison, ok := workloadEmptyNormalizationValue(ctx, planned, prior, configured)
	if !ok {
		return false
	}
	plannedRaw, diags := legacyValue(ctx, comparison)
	if diags.HasError() {
		return false
	}
	priorRaw, diags := legacyValue(ctx, prior)
	if diags.HasError() {
		return false
	}
	plannedSpec, err := kubernetes.ExpandJobV1Spec([]interface{}{map[string]interface{}{"template": plannedRaw}})
	if err != nil {
		return false
	}
	priorSpec, err := kubernetes.ExpandJobV1Spec([]interface{}{map[string]interface{}{"template": priorRaw}})
	return err == nil && apiequality.Semantic.DeepEqual(plannedSpec.Template, priorSpec.Template)
}

func workloadEmptyNormalizationValue(ctx context.Context, plan, state, config attr.Value) (attr.Value, bool) {
	if plan.Equal(state) {
		return plan, true
	}
	if plan.IsUnknown() && config.IsNull() && !state.IsUnknown() {
		return state, true
	}
	if plan.IsNull() && config.IsNull() && workloadEmptyValue(state) {
		return plan, true
	}
	if plan.IsNull() || plan.IsUnknown() || state.IsNull() || state.IsUnknown() {
		return nil, false
	}
	switch planned := plan.(type) {
	case workloadQuantityValue:
		prior, ok := state.(workloadQuantityValue)
		if !ok {
			return nil, false
		}
		equal, diags := planned.StringSemanticEquals(ctx, prior)
		return plan, equal && !diags.HasError()
	case types.List:
		prior, ok := state.(types.List)
		if !ok || len(planned.Elements()) != len(prior.Elements()) {
			return nil, false
		}
		configured, ok := config.(types.List)
		if !ok {
			return nil, false
		}
		elements, oldElements, configElements := planned.Elements(), prior.Elements(), configured.Elements()
		for index, child := range elements {
			childConfig := workloadUnconfiguredChild(ctx, config, child)
			if index < len(configElements) {
				childConfig = configElements[index]
			}
			normalized, ok := workloadEmptyNormalizationValue(ctx, child, oldElements[index], childConfig)
			if !ok {
				return nil, false
			}
			elements[index] = normalized
		}
		value, diags := types.ListValue(planned.ElementType(ctx), elements)
		return value, !diags.HasError()
	case types.Object:
		prior, ok := state.(types.Object)
		if !ok {
			return nil, false
		}
		configured, ok := config.(types.Object)
		if !ok {
			return nil, false
		}
		fields, ok := workloadEmptyNormalizationFields(ctx, planned.Attributes(), prior.Attributes(), configured.Attributes(), config)
		if !ok {
			return nil, false
		}
		value, diags := types.ObjectValue(planned.AttributeTypes(ctx), fields)
		return value, !diags.HasError()
	case types.Map:
		prior, ok := state.(types.Map)
		if !ok {
			return nil, false
		}
		configured, ok := config.(types.Map)
		if !ok {
			return nil, false
		}
		fields, ok := workloadEmptyNormalizationFields(ctx, planned.Elements(), prior.Elements(), configured.Elements(), config)
		if !ok {
			return nil, false
		}
		value, diags := types.MapValue(planned.ElementType(ctx), fields)
		return value, !diags.HasError()
	}
	return nil, false
}

func workloadEmptyNormalizationFields(ctx context.Context, plan, state, config map[string]attr.Value, parentConfig attr.Value) (map[string]attr.Value, bool) {
	if len(plan) != len(state) {
		return nil, false
	}
	for name, child := range plan {
		prior, ok := state[name]
		if !ok {
			return nil, false
		}
		configured, ok := config[name]
		if !ok {
			configured = workloadUnconfiguredChild(ctx, parentConfig, child)
		}
		normalized, ok := workloadEmptyNormalizationValue(ctx, child, prior, configured)
		if !ok {
			return nil, false
		}
		plan[name] = normalized
	}
	return plan, true
}

func workloadEmptyValue(value attr.Value) bool {
	if value.IsNull() || value.IsUnknown() {
		return false
	}
	switch value := value.(type) {
	case basetypes.StringValuable:
		text, diags := value.ToStringValue(context.Background())
		return !diags.HasError() && text.ValueString() == ""
	case types.Int64:
		return value.ValueInt64() == 0
	case types.Bool:
		return !value.ValueBool()
	case types.Map:
		return len(value.Elements()) == 0
	case types.List:
		return len(value.Elements()) == 0
	case types.Set:
		return len(value.Elements()) == 0
	}
	return false
}
