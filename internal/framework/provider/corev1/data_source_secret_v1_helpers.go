// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"encoding/base64"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

// flattenSecretV1Metadata converts a Kubernetes ObjectMeta into a
// common.NamespacedMetadataModel.
//
// It is the Framework equivalent of the SDKv2 flattenMetadataFields (used by the
// SDKv2 secret data source), and is intentionally UNFILTERED: unlike
// common.FlattenNamespacedMetadata — which runs RemoveInternalKeys / RemoveKeys to
// drop control-plane and ignore-listed keys for the resource Read path — the secret
// data source surfaces every annotation and label exactly as the API returns them.
// This matches the SDKv2 data source, which never applied ignore_annotations /
// ignore_labels.
//
// nil maps become null (not {}), matching flattenMetadataFields, which stored the raw
// map and left SDKv2 to serialise a nil map as null.
func flattenSecretV1Metadata(ctx context.Context, meta metav1.ObjectMeta) (common.NamespacedMetadataModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	annotations, d := stringMapToTypesMap(ctx, meta.Annotations)
	diags.Append(d...)
	labels, d := stringMapToTypesMap(ctx, meta.Labels)
	diags.Append(d...)

	// generate_name is omitted from state unless the server actually set it, matching
	// flattenMetadataFields, which only wrote the key when meta.GenerateName != "".
	generateName := types.StringNull()
	if meta.GenerateName != "" {
		generateName = types.StringValue(meta.GenerateName)
	}

	return common.NamespacedMetadataModel{
		MetadataModel: common.MetadataModel{
			MetadataBase: common.MetadataBase{
				Name:            types.StringValue(meta.Name),
				Generation:      types.Int64Value(meta.Generation),
				ResourceVersion: types.StringValue(meta.ResourceVersion),
				UID:             types.StringValue(string(meta.UID)),
				Annotations:     annotations,
				Labels:          labels,
			},
			GenerateName: generateName,
		},
		Namespace: types.StringValue(meta.Namespace),
	}, diags
}

// stringMapToTypesMap converts a plain Go map into a types.Map of strings. A nil map
// yields a null Map so the state matches the SDKv2 data source, which stored null for
// absent annotations / labels.
func stringMapToTypesMap(ctx context.Context, m map[string]string) (types.Map, diag.Diagnostics) {
	if m == nil {
		return types.MapNull(types.StringType), nil
	}
	return types.MapValueFrom(ctx, types.StringType, m)
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
