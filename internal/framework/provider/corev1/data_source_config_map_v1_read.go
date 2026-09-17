// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (d *ConfigMapV1DataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model ConfigMapV1Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(model.Metadata) == 0 {
		return
	}

	name := model.Metadata[0].Name.ValueString()
	namespace := model.Metadata[0].Namespace.ValueString()
	if namespace == "" {
		namespace = "default"
	}

	conn, err := d.SDKv2Meta().(kubernetes.KubeClientsets).MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	log.Printf("[INFO] Reading config map %s", name)
	cfgMap, err := conn.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Printf("[DEBUG] Received error: %#v", err)
			model.Data = types.MapValueMust(types.StringType, map[string]attr.Value{})
			model.BinaryData = types.MapValueMust(types.StringType, map[string]attr.Value{})
			resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
			return
		}
		log.Printf("[DEBUG] Received error: %#v", err)
		resp.Diagnostics.AddError(
			"error reading ConfigMap",
			err.Error(),
		)
		return
	}
	log.Printf("[INFO] Received config map: %#v", cfgMap)

	// Populate server-set metadata fields
	model.Metadata[0].UID = types.StringValue(string(cfgMap.UID))
	model.Metadata[0].ResourceVersion = types.StringValue(cfgMap.ResourceVersion)
	model.Metadata[0].Generation = types.Int64Value(cfgMap.Generation)

	if len(cfgMap.Annotations) > 0 {
		model.Metadata[0].Annotations = flattenStringMap(cfgMap.Annotations)
	}
	if len(cfgMap.Labels) > 0 {
		model.Metadata[0].Labels = flattenStringMap(cfgMap.Labels)
	}

	// Populate data
	dataMap, diags := stringMapToAttrMap(ctx, cfgMap.Data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	model.Data = dataMap

	// Populate binary_data (base64-encoded)
	b64Map := flattenByteMapToBase64Map(cfgMap.BinaryData)
	binaryDataMap, diags := typedStringMapToAttrMap(ctx, b64Map)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	model.BinaryData = binaryDataMap

	// Populate immutable — safely dereference *bool
	immutable := false
	if cfgMap.Immutable != nil {
		immutable = *cfgMap.Immutable
	}
	model.Immutable = types.BoolValue(immutable)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
