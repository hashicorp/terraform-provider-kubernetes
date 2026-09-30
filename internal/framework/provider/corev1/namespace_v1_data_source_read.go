// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// specElementType is the tftypes object shape for a single spec element.
// It must match the NestedObject defined in Schema.
var specElementType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"finalizers": types.ListType{ElemType: types.StringType},
	},
}

func (d *NamespaceV1DataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var model NamespaceV1DataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	meta := d.SDKv2Meta().(kubernetes.KubeClientsets)
	conn, err := meta.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	name := model.Metadata[0].Name.ValueString()

	//  Set the synthetic ID now — before the API call — so that if the namespace
	//  is not found (404) we still persist the requested name as the ID.
	//  This matches the SDKv2 behaviour.
	model.ID = types.StringValue(name)

	ns, err := conn.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Namespace does not exist. Return without an error, as SDKv2 does, and
			// write the config-decoded model: id and metadata.name are set, everything
			// else is null.
			//
			// This is the documented divergence: SDKv2 recorded empty values instead —
			// {} for the metadata maps, "" for uid and resource_version, 0 for
			// generation. An existence check written as metadata[0].uid != "" therefore
			// flips from false to true, because HCL treats null as equal only to null.
			// See the changelog entry and the migration test, which pins both shapes.
			resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
			return
		}
		resp.Diagnostics.AddError(
			"error reading Namespace",
			fmt.Sprintf("Failed to read Namespace %q: %s", name, err.Error()),
		)
		return
	}

	metadata, metaDiags := common.FlattenDataSourceMetadataFields(ctx, ns.ObjectMeta)
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	model.Metadata = []common.MetadataBase{metadata}

	// Populate the spec attribute — equivalent to flattenNamespaceV1Spec in SDKv2.
	// spec is a ListNestedAttribute so we build a types.List of object values.
	if len(ns.Spec.Finalizers) > 0 {
		finVals := make([]attr.Value, len(ns.Spec.Finalizers))
		for i, f := range ns.Spec.Finalizers {
			finVals[i] = types.StringValue(string(f))
		}
		finList := types.ListValueMust(types.StringType, finVals)
		specObj, diags := types.ObjectValue(specElementType.AttrTypes, map[string]attr.Value{
			"finalizers": finList,
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		model.Spec = types.ListValueMust(specElementType, []attr.Value{specObj})
	} else {
		model.Spec = types.ListValueMust(specElementType, []attr.Value{})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
