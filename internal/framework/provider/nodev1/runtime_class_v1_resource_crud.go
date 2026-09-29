// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package nodev1

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"

	nodev1 "k8s.io/api/node/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

// Create implements [resource.Resource].
func (r *RuntimeClassV1) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RuntimeClassV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clients, _, diags := r.sdkv2Meta()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	metadata, diags := common.ExpandMetadata(ctx, plan.Metadata)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	runtimeClass := nodev1.RuntimeClass{
		ObjectMeta: metadata,
		Handler:    plan.Handler.ValueString(),
	}
	tflog.Info(ctx, "Creating runtime class", map[string]any{"name": metadata.Name, "generate_name": metadata.GenerateName})

	out, err := conn.NodeV1().RuntimeClasses().Create(ctx, &runtimeClass, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Error creating runtime class", err.Error())
		return
	}

	// Echo the plan; only server-assigned fields come from the response. Filtering the
	// API's own labels and annotations is Read's job.
	setServerAssignedFields(&plan, out)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, runtimeClassIdentity(out.Name))...)
}

// Read implements [resource.Resource].
func (r *RuntimeClassV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RuntimeClassV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clients, metadataFilters, diags := r.sdkv2Meta()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	// SDKv2 read the object by d.Id(), which is the name.
	name := state.ID.ValueString()
	out, err := conn.NodeV1().RuntimeClasses().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			tflog.Info(ctx, "Runtime class no longer exists, removing from state", map[string]any{"name": name})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to read runtime class %q", name), err.Error())
		return
	}

	// Prior state is the filtering reference. On import it is empty, which is correct:
	// nothing was declared, so nothing is exempt from filtering.
	metadata, diags := common.FlattenMetadata(ctx, out.ObjectMeta, state.Metadata,
		metadataFilters.GetIgnoreAnnotations(), metadataFilters.GetIgnoreLabels())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.ID = types.StringValue(out.Name)
	state.Metadata = metadata
	state.Handler = types.StringValue(out.Handler)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, runtimeClassIdentity(out.Name))...)
}

// Update implements [resource.Resource]. Only labels and annotations can reach here:
// handler, name and generate_name all require replacement.
func (r *RuntimeClassV1) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state RuntimeClassV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clients, _, diags := r.sdkv2Meta()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	// JSON Patch of the managed maps only, as SDKv2's patchMetadata did, so keys owned by
	// other controllers survive. A full-object PUT would drop them.
	ops := common.MetadataPatchOps("/metadata/", state.Metadata[0], plan.Metadata[0])
	name := state.ID.ValueString()

	var out *nodev1.RuntimeClass
	if len(ops) == 0 {
		// e.g. null -> {} after upgrading from SDKv2: a Terraform-only change with no key to
		// patch. Read the object to refresh server-assigned fields instead.
		out, err = conn.NodeV1().RuntimeClasses().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			resp.Diagnostics.AddError("Failed to read runtime class during update", err.Error())
			return
		}
	} else {
		data, err := ops.MarshalJSON()
		if err != nil {
			resp.Diagnostics.AddError("Failed to marshal update operations", err.Error())
			return
		}
		tflog.Info(ctx, "Updating runtime class", map[string]any{"name": name, "patch": string(data)})
		out, err = conn.NodeV1().RuntimeClasses().Patch(ctx, name, k8stypes.JSONPatchType, data, metav1.PatchOptions{})
		if err != nil {
			resp.Diagnostics.AddError("Failed to update runtime class", err.Error())
			return
		}
	}

	setServerAssignedFields(&plan, out)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, runtimeClassIdentity(out.Name))...)
}

// Delete implements [resource.Resource]. Like SDKv2, it does not wait: RuntimeClass has no
// finalizers or terminating phase.
func (r *RuntimeClassV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RuntimeClassV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clients, _, diags := r.sdkv2Meta()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	name := state.ID.ValueString()
	err = conn.NodeV1().RuntimeClasses().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete runtime class", err.Error())
	}
}

// setServerAssignedFields overwrites only the fields the API server assigns. It takes the
// model by pointer: with a value receiver the ID assignment would be silently discarded.
func setServerAssignedFields(m *RuntimeClassV1Model, out *nodev1.RuntimeClass) {
	m.ID = types.StringValue(out.Name)
	m.Metadata[0].Name = types.StringValue(out.Name)
	m.Metadata[0].UID = types.StringValue(string(out.UID))
	m.Metadata[0].ResourceVersion = types.StringValue(out.ResourceVersion)
	m.Metadata[0].Generation = types.Int64Value(out.Generation)
}

func runtimeClassIdentity(name string) common.ResourceIdentity {
	return common.ResourceIdentity{
		APIVersion: types.StringValue(runtimeClassAPIVersion),
		Kind:       types.StringValue(runtimeClassKind),
		Name:       types.StringValue(name),
	}
}
