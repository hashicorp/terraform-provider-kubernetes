package admissionregistrationv1

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	arv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *ValidatingAdmissionPolicy) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ValidatingAdmissionPolicyModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	defaultTimeout, _ := time.ParseDuration("20m")
	timeout, d := plan.Timeouts.Create(ctx, defaultTimeout)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	conn, err := r.SDKv2Meta().(kubernetes.KubeClientsets).MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	obj := &arv1.ValidatingAdmissionPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        plan.Metadata.Name.ValueString(),
			Labels:      expandStringMap(plan.Metadata.Labels),
			Annotations: expandStringMap(plan.Metadata.Annotations),
		},
		Spec: expandVAPSpec(plan.Spec),
	}

	out, err := conn.AdmissionregistrationV1().ValidatingAdmissionPolicies().Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"error creating ValidatingAdmissionPolicy",
			fmt.Sprintf("Failed to create policy %q: %s", plan.Metadata.Name.ValueString(), err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(out.Name)
	flattenVAP(out, &plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ValidatingAdmissionPolicy) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ValidatingAdmissionPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	defaultTimeout, _ := time.ParseDuration("20m")
	timeout, d := state.Timeouts.Read(ctx, defaultTimeout)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	conn, err := r.SDKv2Meta().(kubernetes.KubeClientsets).MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	name := state.Metadata.Name.ValueString()
	out, err := conn.AdmissionregistrationV1().ValidatingAdmissionPolicies().Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"error reading ValidatingAdmissionPolicy",
			fmt.Sprintf("Failed to read policy %q: %s", name, err.Error()),
		)
		return
	}

	flattenVAP(out, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ValidatingAdmissionPolicy) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ValidatingAdmissionPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	defaultTimeout, _ := time.ParseDuration("20m")
	timeout, d := plan.Timeouts.Update(ctx, defaultTimeout)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	conn, err := r.SDKv2Meta().(kubernetes.KubeClientsets).MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	name := plan.Metadata.Name.ValueString()
	cur, err := conn.AdmissionregistrationV1().ValidatingAdmissionPolicies().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"read before update failed",
			fmt.Sprintf("Failed to read policy %q before update: %s", name, err.Error()),
		)
		return
	}

	cur.Spec = expandVAPSpec(plan.Spec)

	if cur.ObjectMeta.Labels == nil {
		cur.ObjectMeta.Labels = make(map[string]string)
	}
	if cur.ObjectMeta.Annotations == nil {
		cur.ObjectMeta.Annotations = make(map[string]string)
	}
	cur.ObjectMeta.Labels = expandStringMap(plan.Metadata.Labels)
	cur.ObjectMeta.Annotations = expandStringMap(plan.Metadata.Annotations)

	out, err := conn.AdmissionregistrationV1().ValidatingAdmissionPolicies().Update(ctx, cur, metav1.UpdateOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"error updating ValidatingAdmissionPolicy",
			fmt.Sprintf("Failed to update policy %q: %s", name, err.Error()),
		)
		return
	}

	flattenVAP(out, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ValidatingAdmissionPolicy) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ValidatingAdmissionPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	defaultTimeout, _ := time.ParseDuration("20m")
	timeout, d := state.Timeouts.Delete(ctx, defaultTimeout)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	conn, err := r.SDKv2Meta().(kubernetes.KubeClientsets).MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	name := state.Metadata.Name.ValueString()
	err = conn.AdmissionregistrationV1().ValidatingAdmissionPolicies().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		resp.Diagnostics.AddError(
			"error deleting ValidatingAdmissionPolicy",
			fmt.Sprintf("Failed to delete policy %q: %s", name, err.Error()),
		)
		return
	}
}

func (r *ValidatingAdmissionPolicy) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("metadata").AtName("name"), req, resp)
}
