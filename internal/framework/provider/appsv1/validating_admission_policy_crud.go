package appsv1

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-codegen-kubernetes/autocrud"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func (r *ValidatingAdmissionPolicy) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var validatingAdmissionPolicyModel ValidatingAdmissionPolicyModel

	diag := req.Config.Get(ctx, &validatingAdmissionPolicyModel)
	resp.Diagnostics.Append(diag...)
	if diag.HasError() {
		return
	}

	defaultTimeout, err := time.ParseDuration("20m")
	if err != nil {
		resp.Diagnostics.AddError("Error parsing timeout", err.Error())
		return
	}
	timeout, diag := validatingAdmissionPolicyModel.Timeouts.Create(ctx, defaultTimeout)
	resp.Diagnostics.Append(diag...)
	if diag.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err = autocrud.Create(ctx, r.clientGetter, r.APIVersion, r.Kind, &validatingAdmissionPolicyModel)
	if err != nil {
		resp.Diagnostics.AddError("Error creating resource", err.Error())
		return
	}

	diags := resp.State.Set(ctx, &validatingAdmissionPolicyModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ValidatingAdmissionPolicy) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var validatingAdmissionPolicyModel ValidatingAdmissionPolicyModel

	diag := req.State.Get(ctx, &validatingAdmissionPolicyModel)
	resp.Diagnostics.Append(diag...)
	if diag.HasError() {
		return
	}

	defaultTimeout, err := time.ParseDuration("20m")
	if err != nil {
		resp.Diagnostics.AddError("Error parsing timeout", err.Error())
		return
	}
	timeout, diag := validatingAdmissionPolicyModel.Timeouts.Read(ctx, defaultTimeout)
	resp.Diagnostics.Append(diag...)
	if diag.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err = autocrud.Read(ctx, r.clientGetter, r.Kind, r.APIVersion, req, &validatingAdmissionPolicyModel)
	if err != nil {
		resp.Diagnostics.AddError("Error reading resource", err.Error())
		return
	}

	diags := resp.State.Set(ctx, &validatingAdmissionPolicyModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ValidatingAdmissionPolicy) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var validatingAdmissionPolicyModel ValidatingAdmissionPolicyModel

	diag := req.Config.Get(ctx, &validatingAdmissionPolicyModel)
	resp.Diagnostics.Append(diag...)
	if diag.HasError() {
		return
	}

	defaultTimeout, err := time.ParseDuration("20m")
	if err != nil {
		resp.Diagnostics.AddError("Error parsing timeout", err.Error())
		return
	}
	timeout, diag := validatingAdmissionPolicyModel.Timeouts.Update(ctx, defaultTimeout)
	resp.Diagnostics.Append(diag...)
	if diag.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err = autocrud.Update(ctx, r.clientGetter, r.Kind, r.APIVersion, &validatingAdmissionPolicyModel)
	if err != nil {
		resp.Diagnostics.AddError("Error updating resource", err.Error())
		return
	}

	diags := resp.State.Set(ctx, &validatingAdmissionPolicyModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ValidatingAdmissionPolicy) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	waitForDeletion := false

	var validatingAdmissionPolicyModel ValidatingAdmissionPolicyModel

	diag := req.State.Get(ctx, &validatingAdmissionPolicyModel)
	resp.Diagnostics.Append(diag...)
	if diag.HasError() {
		return
	}

	defaultTimeout, err := time.ParseDuration("20m")
	if err != nil {
		resp.Diagnostics.AddError("Error parsing timeout", err.Error())
		return
	}
	timeout, diag := validatingAdmissionPolicyModel.Timeouts.Delete(ctx, defaultTimeout)
	resp.Diagnostics.Append(diag...)
	if diag.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err = autocrud.Delete(ctx, r.clientGetter, r.Kind, r.APIVersion, req, waitForDeletion)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting resource", err.Error())
		return
	}

}

func (r *ValidatingAdmissionPolicy) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
