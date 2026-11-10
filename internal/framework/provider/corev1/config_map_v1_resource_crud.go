// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8Types "k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

// Create implements [resource.Resource].
func (c *ConfigMapV1) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ConfigMapV1Model
	// Plan, not Config: Config has not had computed values resolved, so immutable would be
	// null instead of false and metadata.namespace null instead of "default".
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clients, _, metaDiags := c.sdkv2Meta()
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	metadata, diags := common.ExpandNamespacedMetadata(ctx, plan.Metadata)
	resp.Diagnostics.Append(diags...)
	data, diags := expandStringMap(ctx, plan.Data)
	resp.Diagnostics.Append(diags...)
	binaryData, diags := expandBase64Map(ctx, plan.BinaryData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfgMap := v1.ConfigMap{
		ObjectMeta: metadata,
		BinaryData: binaryData,
		Data:       data,
		// SDKv2 always sent a pointer, so an unset immutable reached the API as false.
		Immutable: ptr.To(plan.Immutable.ValueBool()),
	}

	tflog.Info(ctx, "Creating config map", map[string]any{"name": metadata.Name, "namespace": metadata.Namespace})
	out, err := conn.CoreV1().ConfigMaps(metadata.Namespace).Create(ctx, &cfgMap, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Error creating config map", err.Error())
		return
	}

	populateConfigMapFromResponse(&plan, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, configMapIdentity(out.Namespace, out.Name))...)
}

// Read implements [resource.Resource].
func (c *ConfigMapV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ConfigMapV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// kubernetes.IdParts permits an empty namespace or name; reject both.
	namespace, name, err := parseConfigMapID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID", err.Error())
		return
	}

	clients, filters, metaDiags := c.sdkv2Meta()
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	cfgMap, err := conn.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		// Not found means it was removed outside Terraform. SDKv2 signalled this through
		// its Exists hook and d.SetId(""); RemoveResource lets the next plan recreate it.
		if apierrors.IsNotFound(err) {
			tflog.Info(ctx, "Config map no longer exists, removing from state",
				map[string]any{"name": name, "namespace": namespace})
			// Terraform before 1.12 does not send identity on refresh, but the framework
			// still requires it when Read removes a resource.
			if resp.Identity != nil && resp.Identity.Raw.IsNull() {
				resp.Diagnostics.Append(resp.Identity.Set(ctx, configMapIdentity(namespace, name))...)
				if resp.Diagnostics.HasError() {
					return
				}
			}
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Failed to read config map %q in namespace %q", name, namespace), err.Error())
		return
	}

	// Prior state is the filtering reference. On import it is empty, which is the correct
	// baseline: nothing was declared, so nothing is exempt from filtering.
	metadata, diags := common.FlattenNamespacedMetadata(ctx, cfgMap.ObjectMeta, state.Metadata,
		filters.GetIgnoreAnnotations(), filters.GetIgnoreLabels())
	resp.Diagnostics.Append(diags...)
	data, diags := flattenStringMap(ctx, cfgMap.Data, state.Data)
	resp.Diagnostics.Append(diags...)
	binaryData, diags := flattenBase64Map(ctx, cfgMap.BinaryData, state.BinaryData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.ID = types.StringValue(kubernetes.BuildId(cfgMap.ObjectMeta))
	state.Metadata = metadata
	state.Data = data
	state.BinaryData = binaryData
	// A nil pointer is what SDKv2 stored as false.
	state.Immutable = types.BoolValue(cfgMap.Immutable != nil && *cfgMap.Immutable)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, configMapIdentity(cfgMap.Namespace, cfgMap.Name))...)
}

// Update implements [resource.Resource].
func (c *ConfigMapV1) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ConfigMapV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	namespace, name, err := parseConfigMapID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID", err.Error())
		return
	}

	clients, _, metaDiags := c.sdkv2Meta()
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	ops := common.BaseMetadataPatchOps("/metadata/", state.Metadata[0].MetadataBase, plan.Metadata[0].MetadataBase)

	if !mapsEquivalent(state.BinaryData, plan.BinaryData) {
		ops = append(ops, kubernetes.DiffStringMap("/binaryData/",
			common.ExpandMapForPatch(state.BinaryData), common.ExpandMapForPatch(plan.BinaryData))...)
	}
	if !mapsEquivalent(state.Data, plan.Data) {
		ops = append(ops, kubernetes.DiffStringMap("/data/",
			common.ExpandMapForPatch(state.Data), common.ExpandMapForPatch(plan.Data))...)
	}
	if plan.Immutable.ValueBool() != state.Immutable.ValueBool() {
		// "add" rather than SDKv2's "replace": identical when the field exists, and it also
		// works on an imported ConfigMap whose immutable is unset, where "replace" fails.
		ops = append(ops, &kubernetes.AddOperation{
			Path:  "/immutable",
			Value: plan.Immutable.ValueBool(),
		})
	}

	var out *v1.ConfigMap
	if len(ops) == 0 {
		// Only provider-level or equivalent-value changes: nothing to patch, but the
		// computed metadata must still be resolved.
		out, err = conn.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			resp.Diagnostics.AddError("Failed to read config map during update", err.Error())
			return
		}
	} else {
		patch, err := ops.MarshalJSON()
		if err != nil {
			resp.Diagnostics.AddError("Failed to marshal update operations", err.Error())
			return
		}
		tflog.Info(ctx, "Updating config map", map[string]any{"name": name, "namespace": namespace, "patch": string(patch)})
		out, err = conn.CoreV1().ConfigMaps(namespace).Patch(ctx, name, k8Types.JSONPatchType, patch, metav1.PatchOptions{})
		if err != nil {
			resp.Diagnostics.AddError("Failed to update config map", err.Error())
			return
		}
	}

	populateConfigMapFromResponse(&plan, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, configMapIdentity(out.Namespace, out.Name))...)
}

// Delete implements [resource.Resource].
func (c *ConfigMapV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ConfigMapV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	namespace, name, err := parseConfigMapID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID", err.Error())
		return
	}

	clients, _, metaDiags := c.sdkv2Meta()
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	tflog.Info(ctx, "Deleting config map", map[string]any{"name": name, "namespace": namespace})
	err = conn.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	// Already gone is success; erroring would leave the object undestroyable.
	if err != nil && !apierrors.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting config map", err.Error())
	}
}

// populateConfigMapFromResponse writes server-assigned fields into a plan-derived model.
// data, binary_data, labels and annotations are deliberately left as planned: the API
// server and admission webhooks can add keys of their own, and echoing them back would not
// match the plan, which Terraform rejects as an inconsistent result. Filtering them is
// Read's job.
func populateConfigMapFromResponse(model *ConfigMapV1Model, out *v1.ConfigMap) {
	model.ID = types.StringValue(kubernetes.BuildId(out.ObjectMeta))
	common.PopulateNamespacedMetadataFromResponse(&model.Metadata[0], out.ObjectMeta)
}
