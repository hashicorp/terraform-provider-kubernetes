// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

func (r *NamespaceV1) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NamespaceV1Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, 5*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	meta, ok := r.SDKv2Meta().(kubernetes.KubeClientsets)
	if !ok {
		resp.Diagnostics.AddError("provider configuration error", "unexpected meta type")
		return
	}
	conn, err := meta.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	ns := expandNamespace(plan)
	out, err := conn.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"error creating namespace",
			fmt.Sprintf("Failed to create namespace %q: %s", plan.Metadata.Name.ValueString(), err.Error()),
		)
		return
	}

	if plan.WaitForDefaultServiceAccount.ValueBool() {
		err = retry.RetryContext(ctx, createTimeout, func() *retry.RetryError {
			_, err := conn.CoreV1().ServiceAccounts(out.Name).Get(ctx, "default", metav1.GetOptions{})
			if err == nil {
				return nil
			}
			if apierrors.IsNotFound(err) || apierrors.IsServerTimeout(err) || apierrors.IsServiceUnavailable(err) {
				return retry.RetryableError(err)
			}
			return retry.NonRetryableError(err)
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"error waiting for default service account",
				fmt.Sprintf("Failed to wait for default service account in namespace %q: %s", out.Name, err.Error()),
			)
			return
		}
	}

	plan.ID = types.StringValue(out.Name)
	plan.Metadata = flattenNamespaceMetadata(out.ObjectMeta, plan.Metadata, meta.GetIgnoreAnnotations(), meta.GetIgnoreLabels())

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	identity := NamespaceV1IdentityModel{
		APIVersion: types.StringValue("v1"),
		Kind:       types.StringValue("Namespace"),
		Name:       types.StringValue(out.Name),
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
}

func (r *NamespaceV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NamespaceV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	meta, ok := r.SDKv2Meta().(kubernetes.KubeClientsets)
	if !ok {
		resp.Diagnostics.AddError("provider configuration error", "unexpected meta type")
		return
	}
	conn, err := meta.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	name := state.ID.ValueString()
	out, err := conn.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"error reading namespace",
			fmt.Sprintf("Failed to read namespace %q: %s", name, err.Error()),
		)
		return
	}

	state.ID = types.StringValue(out.Name)
	state.Metadata = flattenNamespaceMetadata(out.ObjectMeta, state.Metadata, meta.GetIgnoreAnnotations(), meta.GetIgnoreLabels())

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	identity := NamespaceV1IdentityModel{
		APIVersion: types.StringValue("v1"),
		Kind:       types.StringValue("Namespace"),
		Name:       types.StringValue(out.Name),
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
}

func (r *NamespaceV1) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state NamespaceV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	meta, ok := r.SDKv2Meta().(kubernetes.KubeClientsets)
	if !ok {
		resp.Diagnostics.AddError("provider configuration error", "unexpected meta type")
		return
	}
	conn, err := meta.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	name := state.ID.ValueString()
	data, err := diffMetadataPatch(state.Metadata, plan.Metadata)
	if err != nil {
		resp.Diagnostics.AddError("error building patch", fmt.Sprintf("Failed to build JSON patch for namespace %q: %s", name, err.Error()))
		return
	}

	out, err := conn.CoreV1().Namespaces().Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"error updating namespace",
			fmt.Sprintf("Failed to patch namespace %q: %s", name, err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(out.Name)
	plan.Metadata = flattenNamespaceMetadata(out.ObjectMeta, plan.Metadata, meta.GetIgnoreAnnotations(), meta.GetIgnoreLabels())

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	identity := NamespaceV1IdentityModel{
		APIVersion: types.StringValue("v1"),
		Kind:       types.StringValue("Namespace"),
		Name:       types.StringValue(out.Name),
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
}

func (r *NamespaceV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NamespaceV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := state.Timeouts.Delete(ctx, 5*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	meta, ok := r.SDKv2Meta().(kubernetes.KubeClientsets)
	if !ok {
		resp.Diagnostics.AddError("provider configuration error", "unexpected meta type")
		return
	}
	conn, err := meta.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	name := state.ID.ValueString()
	err = conn.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		resp.Diagnostics.AddError(
			"error deleting namespace",
			fmt.Sprintf("Failed to delete namespace %q: %s", name, err.Error()),
		)
		return
	}

	stateConf := &retry.StateChangeConf{
		Pending:      []string{"Terminating"},
		Target:       []string{},
		Timeout:      deleteTimeout,
		Delay:        5 * time.Second,
		PollInterval: 5 * time.Second,
		Refresh: func() (interface{}, string, error) {
			out, err := conn.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				return nil, "", nil
			}
			if err != nil {
				return nil, "", err
			}
			return out, string(out.Status.Phase), nil
		},
	}
	_, err = stateConf.WaitForStateContext(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"error waiting for namespace deletion",
			fmt.Sprintf("Namespace %q did not finish terminating: %s", name, err.Error()),
		)
	}
}

func (r *NamespaceV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var name string

	if req.ID != "" {
		name = req.ID
	} else {
		var identityData NamespaceV1IdentityModel
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identityData)...)
		if resp.Diagnostics.HasError() {
			return
		}
		name = identityData.Name.ValueString()
	}

	meta, ok := r.SDKv2Meta().(kubernetes.KubeClientsets)
	if !ok {
		resp.Diagnostics.AddError("provider configuration error", "unexpected meta type")
		return
	}
	conn, err := meta.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	out, err := conn.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"error importing namespace",
			fmt.Sprintf("Failed to import namespace %q: %s", name, err.Error()),
		)
		return
	}

	var state NamespaceV1Model
	state.ID = types.StringValue(out.Name)
	state.WaitForDefaultServiceAccount = types.BoolValue(false)

	timeoutsAttrTypes := map[string]attr.Type{
		"create": types.StringType,
		"delete": types.StringType,
	}
	state.Timeouts = timeouts.Value{
		Object: types.ObjectNull(timeoutsAttrTypes),
	}

	state.Metadata = flattenNamespaceMetadata(out.ObjectMeta, NamespaceMetadataModel{}, meta.GetIgnoreAnnotations(), meta.GetIgnoreLabels())

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	identity := NamespaceV1IdentityModel{
		APIVersion: types.StringValue("v1"),
		Kind:       types.StringValue("Namespace"),
		Name:       types.StringValue(out.Name),
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
}
