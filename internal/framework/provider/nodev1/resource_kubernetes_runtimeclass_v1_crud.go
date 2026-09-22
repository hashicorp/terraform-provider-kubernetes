// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package nodev1

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *RuntimeClassV1) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RuntimeClassV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := r.SDKv2Meta().(kubernetes.KubeClientsets).MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	obj := buildRuntimeClassObject(plan)

	out, err := conn.NodeV1().RuntimeClasses().Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"error creating RuntimeClass",
			fmt.Sprintf("Failed to create RuntimeClass %q: %s", obj.Name, err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(out.Name)
	plan.Metadata[0].Name = types.StringValue(out.Name)
	plan.Metadata[0].UID = types.StringValue(string(out.UID))
	plan.Metadata[0].ResourceVersion = types.StringValue(out.ResourceVersion)
	plan.Metadata[0].Generation = types.Int64Value(out.Generation)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	identity := RuntimeClassV1IdentityModel{
		APIVersion: types.StringValue("node.k8s.io/v1"),
		Kind:       types.StringValue("RuntimeClass"),
		Name:       types.StringValue(out.Name),
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
}

func (r *RuntimeClassV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RuntimeClassV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := r.SDKv2Meta().(kubernetes.KubeClientsets).MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	name := state.Metadata[0].Name.ValueString()
	out, err := conn.NodeV1().RuntimeClasses().Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"error reading RuntimeClass",
			fmt.Sprintf("Failed to read RuntimeClass %q: %s", name, err.Error()),
		)
		return
	}

	state.Metadata = []MetadataModel{
		flattenMetadata(out.ObjectMeta, state.Metadata[0].Annotations, state.Metadata[0].Labels, r.IgnoreAnnotations, r.IgnoreLabels),
	}
	state.Handler = types.StringValue(out.Handler)
	state.ID = types.StringValue(out.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	identity := RuntimeClassV1IdentityModel{
		APIVersion: types.StringValue("node.k8s.io/v1"),
		Kind:       types.StringValue("RuntimeClass"),
		Name:       types.StringValue(out.Name),
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
}

func (r *RuntimeClassV1) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RuntimeClassV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// ResourceVersion is a computed server field not present in the plan.
	// Read it from prior state so the Kubernetes API accepts the PUT.
	var state RuntimeClassV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Metadata[0].ResourceVersion = state.Metadata[0].ResourceVersion

	conn, err := r.SDKv2Meta().(kubernetes.KubeClientsets).MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	obj := buildRuntimeClassObject(plan)
	out, err := conn.NodeV1().RuntimeClasses().Update(ctx, obj, metav1.UpdateOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"error updating RuntimeClass",
			fmt.Sprintf("Failed to update RuntimeClass %q: %s", obj.Name, err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(out.Name)
	plan.Metadata[0].Name = types.StringValue(out.Name)
	plan.Metadata[0].UID = types.StringValue(string(out.UID))
	plan.Metadata[0].ResourceVersion = types.StringValue(out.ResourceVersion)
	plan.Metadata[0].Generation = types.Int64Value(out.Generation)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	identity := RuntimeClassV1IdentityModel{
		APIVersion: types.StringValue("node.k8s.io/v1"),
		Kind:       types.StringValue("RuntimeClass"),
		Name:       types.StringValue(out.Name),
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
}

func (r *RuntimeClassV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RuntimeClassV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := r.SDKv2Meta().(kubernetes.KubeClientsets).MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	name := state.Metadata[0].Name.ValueString()
	err = conn.NodeV1().RuntimeClasses().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		resp.Diagnostics.AddError(
			"error deleting RuntimeClass",
			fmt.Sprintf("Failed to delete RuntimeClass %q: %s", name, err.Error()),
		)
		return
	}
}

func (r *RuntimeClassV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var name string

	if req.ID != "" {
		name = req.ID
	} else {
		var identityData RuntimeClassV1IdentityModel
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identityData)...)
		if resp.Diagnostics.HasError() {
			return
		}
		name = identityData.Name.ValueString()
	}

	conn, err := r.SDKv2Meta().(kubernetes.KubeClientsets).MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	out, err := conn.NodeV1().RuntimeClasses().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"error importing RuntimeClass",
			fmt.Sprintf("Failed to import RuntimeClass %q: %s", name, err.Error()),
		)
		return
	}

	var state RuntimeClassV1Model
	state.ID = types.StringValue(out.Name)
	state.Handler = types.StringValue(out.Handler)
	state.Metadata = []MetadataModel{flattenMetadata(out.ObjectMeta, nil, nil, r.IgnoreAnnotations, r.IgnoreLabels)}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	identity := RuntimeClassV1IdentityModel{
		APIVersion: types.StringValue("node.k8s.io/v1"),
		Kind:       types.StringValue("RuntimeClass"),
		Name:       types.StringValue(out.Name),
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
}
