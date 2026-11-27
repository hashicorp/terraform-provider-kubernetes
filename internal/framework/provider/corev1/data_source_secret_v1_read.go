// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

// Read fetches the Secret from the Kubernetes API and maps it into state.
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
	model.ID = types.StringValue(kubernetes.BuildId(metav1.ObjectMeta{
		Namespace: namespace,
		Name:      name,
	}))

	if d.SDKv2Meta == nil {
		resp.Diagnostics.AddError("Provider not configured",
			"The SDKv2 provider metadata is unavailable. This is a bug in the provider.")
		return
	}
	sdkMeta := d.SDKv2Meta()
	conn, ok := sdkMeta.(kubernetes.KubeClientsets)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data",
			fmt.Sprintf("Expected kubernetes.KubeClientsets, got %T. This is a bug in the provider.", sdkMeta))
		return
	}
	client, err := conn.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("failed to create Kubernetes client", err.Error())
		return
	}

	log.Printf("[INFO] Reading secret %s/%s", namespace, name)
	secret, err := client.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Secret does not exist — return empty state with no error.
			resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
			return
		}
		resp.Diagnostics.AddError("failed to read secret", err.Error())
		return
	}
	log.Printf("[INFO] Received secret: %#v", secret.ObjectMeta)

	model.ID = types.StringValue(kubernetes.BuildId(secret.ObjectMeta))

	// Flatten metadata unfiltered: ignore_annotations / ignore_labels do not apply to data sources.
	metadata, diags := flattenSecretV1Metadata(ctx, secret.ObjectMeta)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	model.Metadata[0] = metadata

	// binary_data selective extraction: declared keys are base64-encoded and removed
	// from the working copy; remaining entries become model.Data (UTF-8 strings).
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

		binaryData, binDiags := types.MapValueFrom(ctx, types.StringType, base64EncodeByteMap(binaryRaw))
		resp.Diagnostics.Append(binDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		model.BinaryData = binaryData
	} else {
		model.BinaryData = types.MapNull(types.StringType)
	}

	dataMap, dataDiags := types.MapValueFrom(ctx, types.StringType, flattenByteMapToStringMap(workingData))
	resp.Diagnostics.Append(dataDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	model.Data = dataMap

	model.Type = types.StringValue(string(secret.Type))

	immutable := false
	if secret.Immutable != nil {
		immutable = *secret.Immutable
	}
	model.Immutable = types.BoolValue(immutable)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
