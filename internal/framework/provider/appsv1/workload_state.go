// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

type workloadTemplateModel struct {
	Metadata []common.NamespacedMetadataModel `tfsdk:"metadata"`
	Spec     types.List                       `tfsdk:"spec"`
}

// workloadStateModel is workload state whose spec is already a Framework value,
// which avoids reflecting over the large pod spec.
type workloadStateModel struct {
	ID             types.String                     `tfsdk:"id"`
	Metadata       []common.NamespacedMetadataModel `tfsdk:"metadata"`
	Spec           types.List                       `tfsdk:"spec"`
	WaitForRollout types.Bool                       `tfsdk:"wait_for_rollout"`
	Timeouts       timeouts.Value                   `tfsdk:"timeouts"`
}

// workloadSpecListType returns the type of the spec block of a workload schema.
func workloadSpecListType(schema common.SchemaFunc) func() types.ListType {
	return sync.OnceValue(func() types.ListType {
		var resp resource.SchemaResponse
		schema(context.Background(), resource.SchemaRequest{}, &resp)
		return resp.Schema.Blocks["spec"].Type().(types.ListType)
	})
}

// workloadListValue is types.ListValueFrom with each element built by object.
func workloadListValue[T any](listType types.ListType, items []T, object func(T, types.ObjectType) (attr.Value, diag.Diagnostics)) (types.List, diag.Diagnostics) {
	elemType := listType.ElemType.(types.ObjectType)
	if items == nil {
		return types.ListNull(elemType), nil
	}
	var diags diag.Diagnostics
	elements := make([]attr.Value, 0, len(items))
	for _, item := range items {
		element, d := object(item, elemType)
		diags.Append(d...)
		elements = append(elements, element)
	}
	if diags.HasError() {
		return types.ListNull(elemType), diags
	}
	return types.ListValue(elemType, elements)
}

// workloadTemplateListValue builds a template block without reflecting over
// its pod spec.
func workloadTemplateListValue(ctx context.Context, listType types.ListType, templates []workloadTemplateModel) (types.List, diag.Diagnostics) {
	return workloadListValue(listType, templates, func(t workloadTemplateModel, typ types.ObjectType) (attr.Value, diag.Diagnostics) {
		metadata, diags := types.ListValueFrom(ctx, typ.AttrTypes["metadata"].(types.ListType).ElemType, t.Metadata)
		if diags.HasError() {
			return nil, diags
		}
		object, d := types.ObjectValue(typ.AttrTypes, map[string]attr.Value{
			"metadata": metadata,
			"spec":     t.Spec,
		})
		diags.Append(d...)
		return object, diags
	})
}

// workloadModels decodes the objects of a list block like ElementsAs, with
// model building each element from its attributes.
func workloadModels[T any](value types.List, model func(map[string]attr.Value, *diag.Diagnostics) T) ([]T, diag.Diagnostics) {
	var diags diag.Diagnostics
	if value.IsNull() {
		return nil, diags
	}
	if value.IsUnknown() {
		diags.AddError("Value Conversion Error", "Received an unknown list where a known list block was expected.")
		return nil, diags
	}
	out := make([]T, 0, len(value.Elements()))
	for _, element := range value.Elements() {
		object, ok := element.(types.Object)
		if !ok || object.IsNull() || object.IsUnknown() {
			diags.AddError("Value Conversion Error", fmt.Sprintf("Received %s where a known block object was expected.", element))
			return nil, diags
		}
		out = append(out, model(object.Attributes(), &diags))
		if diags.HasError() {
			return nil, diags
		}
	}
	return out, diags
}

// workloadTemplateModels decodes a template block without reflecting over its
// pod spec.
func workloadTemplateModels(ctx context.Context, value types.List) ([]workloadTemplateModel, diag.Diagnostics) {
	return workloadModels(value, func(attrs map[string]attr.Value, diags *diag.Diagnostics) workloadTemplateModel {
		model := workloadTemplateModel{Spec: attributeAs[types.List](attrs, "spec", diags)}
		attributeElementsAs(ctx, attrs, "metadata", &model.Metadata, diags)
		return model
	})
}

func attributeAs[T attr.Value](attrs map[string]attr.Value, name string, diags *diag.Diagnostics) T {
	value, ok := attrs[name].(T)
	if !ok {
		var want T
		diags.AddError("Value Conversion Error", fmt.Sprintf("Expected %T for %q, got %T.", want, name, attrs[name]))
	}
	return value
}

func attributeElementsAs(ctx context.Context, attrs map[string]attr.Value, name string, target any, diags *diag.Diagnostics) {
	if list, ok := attrs[name].(types.List); ok {
		diags.Append(list.ElementsAs(ctx, target, false)...)
		return
	}
	diags.AddError("Value Conversion Error", fmt.Sprintf("Expected a list for %q, got %T.", name, attrs[name]))
}
