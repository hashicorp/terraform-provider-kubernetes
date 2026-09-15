// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

// Read fetches the Secret from the Kubernetes API and maps it into state.
//
// Behavior is identical to the SDKv2 dataSourceKubernetesSecretV1Read:
//   - Not-found is treated as empty state (no error), matching SDKv2.
//   - binary_data keys declared by the user are extracted from secret.Data as
//     base64-encoded strings; those keys are then removed from the data map.
//   - Remaining secret.Data entries are decoded as UTF-8 strings into data.
//   - Metadata is flattened via flattenSecretV1Metadata (local, unfiltered) so
//     that provider-level ignore_annotations / ignore_labels do NOT suppress
//     any annotations or labels — matching flattenMetadataFields behavior.
func (d *SecretV1DataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model SecretV1Model

	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(model.Metadata) == 0 {
		resp.Diagnostics.AddError("metadata block required", "at least one metadata block must be specified")
		return
	}
	meta := model.Metadata[0]

	namespace := meta.Namespace.ValueString()
	if namespace == "" {
		namespace = "default"
	}
	name := meta.Name.ValueString()

	conn, err := d.SDKv2Meta().(kubernetes.KubeClientsets).MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("failed to create Kubernetes client", err.Error())
		return
	}

	log.Printf("[INFO] Reading secret %s/%s", namespace, name)
	secret, err := conn.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Secret does not exist — return empty state with no error, matching SDKv2.
			resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
			return
		}
		resp.Diagnostics.AddError("failed to read secret", err.Error())
		return
	}
	log.Printf("[INFO] Received secret: %#v", secret.ObjectMeta)

	// Flatten metadata — use the local, unfiltered helper (not the provider-aware
	// flattenMetadata) so ignore_annotations / ignore_labels are not applied.
	model.Metadata[0] = flattenSecretV1Metadata(secret.ObjectMeta)

	// binary_data selective extraction — replicate SDKv2 logic exactly:
	// 1. If the user declared keys in binary_data, extract their raw bytes from
	//    secret.Data, base64-encode them, and remove those keys from a working copy.
	// 2. Remaining entries become model.Data (UTF-8 decoded strings).
	workingData := make(map[string][]byte, len(secret.Data))
	for k, v := range secret.Data {
		workingData[k] = v
	}

	if !model.BinaryData.IsNull() && len(model.BinaryData.Elements()) > 0 {
		// Collect keys the user declared in binary_data.
		declaredBinaryKeys := make(map[string]types.String)
		resp.Diagnostics.Append(model.BinaryData.ElementsAs(ctx, &declaredBinaryKeys, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		binaryRaw := make(map[string][]byte, len(declaredBinaryKeys))
		for k := range declaredBinaryKeys {
			binaryRaw[k] = workingData[k]
			delete(workingData, k)
		}
		model.BinaryData = flattenTypesMap(base64EncodeByteMap(binaryRaw))
	} else {
		model.BinaryData = types.MapValueMust(types.StringType, map[string]attr.Value{})
	}

	model.Data = flattenTypesMap(flattenByteMapToStringMap(workingData))
	model.Type = types.StringValue(string(secret.Type))

	immutable := false
	if secret.Immutable != nil {
		immutable = *secret.Immutable
	}
	model.Immutable = types.BoolValue(immutable)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
