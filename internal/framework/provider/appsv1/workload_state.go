// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
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

// workloadStateModel is the state of a workload whose spec is already a
// Framework value. Framework reflection converts every nested model slice
// back and forth, so the large pod spec is cheaper to set this way.
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
