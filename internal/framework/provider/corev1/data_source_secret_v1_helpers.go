// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"encoding/base64"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// flattenSecretV1Metadata converts a Kubernetes ObjectMeta into a
// SecretV1MetadataModel. It is the Framework equivalent of the SDKv2
// flattenMetadataFields — intentionally unfiltered so that provider-level
// ignore_annotations / ignore_labels do NOT suppress any annotations or labels.
func flattenSecretV1Metadata(meta metav1.ObjectMeta) SecretV1MetadataModel {
	m := SecretV1MetadataModel{
		Name:            types.StringValue(meta.Name),
		Namespace:       types.StringValue(meta.Namespace),
		GenerateName:    types.StringValue(meta.GenerateName),
		Generation:      types.Int64Value(meta.Generation),
		ResourceVersion: types.StringValue(meta.ResourceVersion),
		UID:             types.StringValue(string(meta.UID)),
		Annotations:     flattenStringMap(meta.Annotations),
		Labels:          flattenStringMap(meta.Labels),
	}
	return m
}

// flattenStringMap converts a plain Go map[string]string into a
// map[string]types.String suitable for Framework model fields.
func flattenStringMap(m map[string]string) map[string]types.String {
	if m == nil {
		return map[string]types.String{}
	}
	result := make(map[string]types.String, len(m))
	for k, v := range m {
		result[k] = types.StringValue(v)
	}
	return result
}

// flattenByteMapToStringMap decodes each byte slice as a UTF-8 string.
// This mirrors the SDKv2 flattenByteMapToStringMap in kubernetes/structures.go.
func flattenByteMapToStringMap(m map[string][]byte) map[string]string {
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = string(v)
	}
	return result
}

// base64EncodeByteMap base64-encodes each byte slice value.
// This mirrors the SDKv2 base64EncodeByteMap in kubernetes/structures.go.
func base64EncodeByteMap(m map[string][]byte) map[string]string {
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = base64.StdEncoding.EncodeToString(v)
	}
	return result
}

// flattenTypesMap converts a plain Go map[string]string into a types.Map
// with StringType element type.
func flattenTypesMap(m map[string]string) types.Map {
	elems := make(map[string]attr.Value, len(m))
	for k, v := range m {
		elems[k] = types.StringValue(v)
	}
	return types.MapValueMust(types.StringType, elems)
}
