// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package networkingv1

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	networking "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	k8sclient "k8s.io/client-go/kubernetes"
)

const ingressDefaultTimeout = 20 * time.Minute

func (r *IngressV1) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan IngressV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(ingressValidateModel(plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	timeout, d := plan.Timeouts.Create(ctx, ingressDefaultTimeout)
	resp.Diagnostics.Append(d...)
	conn, _, d := networkingClient(r.SDKv2Meta)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	metadata, d := common.ExpandNamespacedMetadata(ctx, plan.Metadata)
	resp.Diagnostics.Append(d...)
	spec, d := ingressExpandSpec(ctx, plan.Spec)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := conn.NetworkingV1().Ingresses(metadata.Namespace).Create(ctx, &networking.Ingress{
		ObjectMeta: metadata, Spec: spec,
	}, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Error creating ingress", err.Error())
		return
	}
	if out == nil {
		resp.Diagnostics.AddError("Error creating ingress", "The Kubernetes API returned no ingress.")
		return
	}

	// Persist the server-assigned name and computed fields before any fallible waiter.
	resp.Diagnostics.Append(ingressApplyResult(ctx, &plan, out)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, ingressIdentity(out.Namespace, out.Name))...)
	}
	if resp.Diagnostics.HasError() || !plan.WaitForLoadBalancer.ValueBool() {
		return
	}
	out, err = ingressWaitForLoadBalancer(ctx, conn, out.Namespace, out.Name, timeout)
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for ingress load balancer",
			fmt.Sprintf("Ingress %q was created, but its load balancer did not become ready: %s", plan.ID.ValueString(), err))
		return
	}
	resp.Diagnostics.Append(ingressApplyResult(ctx, &plan, out)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *IngressV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	ctx, cancel := context.WithTimeout(ctx, ingressDefaultTimeout)
	defer cancel()

	var state IngressV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	namespace, name, err := ingressIDParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ingress ID", err.Error())
		return
	}
	conn, filters, d := networkingClient(r.SDKv2Meta)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, ingressIdentity(namespace, name))...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	out, err := conn.NetworkingV1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading ingress", fmt.Sprintf("Could not read ingress %q: %s", state.ID.ValueString(), err))
		return
	}
	if out == nil {
		resp.Diagnostics.AddError("Error reading ingress", "The Kubernetes API returned no ingress.")
		return
	}
	state.Metadata, d = flattenNetworkingNamespacedMetadata(ctx, out.ObjectMeta, state.Metadata, filters.GetIgnoreAnnotations(), filters.GetIgnoreLabels())
	resp.Diagnostics.Append(d...)
	state.Spec, d = ingressFlattenSpec(ctx, out.Spec, state.Spec)
	resp.Diagnostics.Append(d...)
	state.Status, d = ingressFlattenStatus(ctx, out.Status.LoadBalancer)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, ingressIdentity(out.Namespace, out.Name))...)
	}
}

func (r *IngressV1) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	ctx, cancel := context.WithTimeout(ctx, ingressDefaultTimeout)
	defer cancel()

	var plan, prior IngressV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	resp.Diagnostics.Append(ingressValidateModel(plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(prior.Metadata) != 1 || len(prior.Spec) != 1 {
		resp.Diagnostics.AddError("Invalid ingress state", "Stored ingress state must contain one metadata and one spec block.")
		return
	}
	namespace, name, err := ingressIDParts(prior.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ingress ID", err.Error())
		return
	}
	conn, _, d := networkingClient(r.SDKv2Meta)
	resp.Diagnostics.Append(d...)
	spec, d := ingressExpandSpec(ctx, plan.Spec)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := conn.NetworkingV1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Error reading ingress before update", err.Error())
		return
	}
	if out == nil {
		resp.Diagnostics.AddError("Error reading ingress before update", "The Kubernetes API returned no ingress.")
		return
	}
	ops, d := networkingMetadataPatch(ctx, out.ObjectMeta, prior.Metadata[0].MetadataBase, plan.Metadata[0].MetadataBase)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ops = append(ops, ingressSpecPatch(prior.Spec[0], plan.Spec[0], spec)...)
	if len(ops) > 0 {
		patch, err := ops.MarshalJSON()
		if err != nil {
			resp.Diagnostics.AddError("Error building ingress patch", err.Error())
			return
		}
		out, err = conn.NetworkingV1().Ingresses(namespace).Patch(ctx, name, k8stypes.JSONPatchType, patch, metav1.PatchOptions{})
		if err != nil {
			resp.Diagnostics.AddError("Error updating ingress", err.Error())
			return
		}
		if out == nil {
			resp.Diagnostics.AddError("Error updating ingress", "The Kubernetes API returned no ingress.")
			return
		}
	}
	// SDKv2 only waits on Create, even when this flag remains true during Update.
	resp.Diagnostics.Append(ingressApplyResult(ctx, &plan, out)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, ingressIdentity(out.Namespace, out.Name))...)
	}
}

func (r *IngressV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state IngressV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	timeout, d := state.Timeouts.Delete(ctx, ingressDefaultTimeout)
	resp.Diagnostics.Append(d...)
	conn, _, d := networkingClient(r.SDKv2Meta)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	namespace, name, err := ingressIDParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ingress ID", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	err = conn.NetworkingV1().Ingresses(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error deleting ingress", err.Error())
		return
	}
	err = retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		_, err := conn.NetworkingV1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return retry.NonRetryableError(err)
		}
		return retry.RetryableError(fmt.Errorf("ingress %q still exists", state.ID.ValueString()))
	})
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for ingress deletion", err.Error())
	}
}

func ingressValidateModel(model IngressV1Model) diag.Diagnostics {
	var diags diag.Diagnostics
	if len(model.Metadata) != 1 || len(model.Spec) != 1 {
		diags.AddError("Invalid ingress configuration", "Exactly one metadata and one spec block are required.")
		return diags
	}
	meta := model.Metadata[0]
	ingressKnown(&diags, "metadata.namespace", meta.Namespace)
	ingressKnown(&diags, "metadata.generate_name", meta.GenerateName)
	ingressKnown(&diags, "metadata.labels", meta.Labels)
	ingressKnown(&diags, "metadata.annotations", meta.Annotations)
	ingressKnown(&diags, "wait_for_load_balancer", model.WaitForLoadBalancer)
	if (meta.Name.IsNull() || meta.Name.IsUnknown() || meta.Name.ValueString() == "") && meta.GenerateName.ValueString() == "" {
		diags.AddError("Invalid ingress name", "A known name or generate_name is required before applying the ingress.")
	}
	return diags
}

func ingressApplyResult(ctx context.Context, plan *IngressV1Model, out *networking.Ingress) diag.Diagnostics {
	plan.ID = types.StringValue(kubernetes.BuildId(out.ObjectMeta))
	meta := &plan.Metadata[0]
	meta.Name = types.StringValue(out.Name)
	meta.UID = types.StringValue(string(out.UID))
	meta.ResourceVersion = types.StringValue(out.ResourceVersion)
	meta.Generation = types.Int64Value(out.Generation)
	if plan.Spec[0].IngressClassName.IsUnknown() || plan.Spec[0].IngressClassName.IsNull() {
		class := ""
		if out.Spec.IngressClassName != nil {
			class = *out.Spec.IngressClassName
		}
		plan.Spec[0].IngressClassName = types.StringValue(class)
	}
	var diags diag.Diagnostics
	plan.Status, diags = ingressFlattenStatus(ctx, out.Status.LoadBalancer)
	return diags
}

func ingressSpecPatch(prior, plan IngressV1SpecModel, expanded networking.IngressSpec) kubernetes.PatchOperations {
	ops := kubernetes.PatchOperations{}
	if !plan.IngressClassName.IsUnknown() && !plan.IngressClassName.Equal(prior.IngressClassName) {
		ops = append(ops, &kubernetes.AddOperation{Path: "/spec/ingressClassName", Value: expanded.IngressClassName})
	}
	if !reflect.DeepEqual(prior.DefaultBackend, plan.DefaultBackend) {
		ops = append(ops, &kubernetes.AddOperation{Path: "/spec/defaultBackend", Value: expanded.DefaultBackend})
	}
	if !reflect.DeepEqual(prior.Rule, plan.Rule) {
		ops = append(ops, &kubernetes.AddOperation{Path: "/spec/rules", Value: expanded.Rules})
	}
	if !reflect.DeepEqual(prior.TLS, plan.TLS) {
		ops = append(ops, &kubernetes.AddOperation{Path: "/spec/tls", Value: expanded.TLS})
	}
	return ops
}

func ingressWaitForLoadBalancer(ctx context.Context, conn k8sclient.Interface, namespace, name string, timeout time.Duration) (*networking.Ingress, error) {
	var out *networking.Ingress
	err := retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		value, err := conn.NetworkingV1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return retry.RetryableError(err)
		}
		if err != nil {
			return retry.NonRetryableError(err)
		}
		if value == nil {
			return retry.NonRetryableError(fmt.Errorf("the Kubernetes API returned no ingress"))
		}
		out = value
		if len(value.Status.LoadBalancer.Ingress) == 0 {
			return retry.RetryableError(fmt.Errorf("ingress %q load balancer is not ready", namespace+"/"+name))
		}
		return nil
	})
	return out, err
}
