// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

const (
	serviceCreateTimeout = 10 * time.Minute
	serviceDeleteTimeout = 20 * time.Minute
)

func (r *ServiceV1) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ServiceV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(plan.Metadata) != 1 || len(plan.Spec) != 1 {
		resp.Diagnostics.AddError("Invalid service plan", "Exactly one metadata and spec block are required.")
		return
	}
	resp.Diagnostics.Append(validateServiceWritePlan(plan.Spec[0])...)
	if resp.Diagnostics.HasError() {
		return
	}
	timeout, d := plan.Timeouts.Create(ctx, serviceCreateTimeout)
	resp.Diagnostics.Append(d...)
	clients, filters, d := r.sdkv2Meta()
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}
	meta, d := common.ExpandNamespacedMetadata(ctx, plan.Metadata)
	resp.Diagnostics.Append(d...)
	spec, d := expandServiceSpec(ctx, plan.Spec[0])
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	out, err := conn.CoreV1().Services(meta.Namespace).Create(ctx, &corev1.Service{ObjectMeta: meta, Spec: spec}, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Error creating service", err.Error())
		return
	}
	// Retain the server identity even if conversion or the subsequent wait fails.
	resp.State.Raw = req.Plan.Raw
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), out.Namespace+"/"+out.Name)...)
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, serviceIdentity(out.Namespace, out.Name))...)
	}
	resp.Diagnostics.Append(flattenService(ctx, out, &plan, filters, true)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if out.Spec.Type != corev1.ServiceTypeLoadBalancer || !plan.WaitForLoadBalancer.ValueBool() {
		return
	}
	err = retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		current, err := conn.CoreV1().Services(out.Namespace).Get(ctx, out.Name, metav1.GetOptions{})
		if err != nil {
			return retry.NonRetryableError(err)
		}
		out = current
		if len(current.Status.LoadBalancer.Ingress) > 0 {
			return nil
		}
		return retry.RetryableError(fmt.Errorf("waiting for Service %q to assign an IP or hostname to its load balancer", plan.ID.ValueString()))
	})
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for service load balancer", fmt.Sprintf("Service %q was created, but its load balancer did not become ready: %s", plan.ID.ValueString(), err))
		return
	}
	resp.Diagnostics.Append(flattenService(ctx, out, &plan, filters, true)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	}
}

func (r *ServiceV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ServiceV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid service ID", err.Error())
		return
	}
	clients, filters, d := r.sdkv2Meta()
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}
	out, err := conn.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading service", err.Error())
		return
	}
	resp.Diagnostics.Append(flattenService(ctx, out, &state, filters, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, serviceIdentity(out.Namespace, out.Name))...)
	}
}

func (r *ServiceV1) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, plan ServiceV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(state.Metadata) != 1 || len(plan.Metadata) != 1 || len(state.Spec) != 1 || len(plan.Spec) != 1 {
		resp.Diagnostics.AddError("Invalid service state or plan", "Exactly one metadata and spec block are required.")
		return
	}
	resp.Diagnostics.Append(validateServiceWritePlan(plan.Spec[0])...)
	if resp.Diagnostics.HasError() {
		return
	}
	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid service ID", err.Error())
		return
	}
	clients, filters, d := r.sdkv2Meta()
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}
	out, err := conn.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Error reading service during update", err.Error())
		return
	}
	ops := serviceMetadataPatchOps(state.Metadata[0], plan.Metadata[0], out.ObjectMeta)
	specOps, d := serviceSpecPatchOps(ctx, state.Spec[0], plan.Spec[0], out.Spec)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ops = append(ops, specOps...)
	if len(ops) > 0 {
		if out.ResourceVersion != "" {
			ops = append([]servicePatchOperation{{Op: "test", Path: "/metadata/resourceVersion", Value: out.ResourceVersion}}, ops...)
		}
		data, err := json.Marshal(ops)
		if err != nil {
			resp.Diagnostics.AddError("Error preparing service update", err.Error())
			return
		}
		out, err = conn.CoreV1().Services(namespace).Patch(ctx, name, k8stypes.JSONPatchType, data, metav1.PatchOptions{})
		if err != nil {
			resp.Diagnostics.AddError("Error updating service", err.Error())
			return
		}
	}
	resp.Diagnostics.Append(flattenService(ctx, out, &plan, filters, true)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, serviceIdentity(out.Namespace, out.Name))...)
	}
}

func (r *ServiceV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ServiceV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid service ID", err.Error())
		return
	}
	clients, _, d := r.sdkv2Meta()
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(ctx, serviceDeleteTimeout)
	defer cancel()
	err = conn.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error deleting service", err.Error())
		return
	}
	err = retry.RetryContext(ctx, serviceDeleteTimeout, func() *retry.RetryError {
		_, err := conn.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return retry.NonRetryableError(err)
		}
		return retry.RetryableError(fmt.Errorf("Service %q still exists", state.ID.ValueString()))
	})
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for service deletion", err.Error())
	}
}
