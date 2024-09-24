package corev1

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/hashicorp/terraform-plugin-codegen-kubernetes/autocrud"
)

func (r *Secret) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var dataModel SecretModel

	diag := req.Config.Get(ctx, &dataModel)
	resp.Diagnostics.Append(diag...)
	if diag.HasError() {
		return
	}
	r.BeforeCreate(ctx, req, resp, &dataModel)

	defaultTimeout, err := time.ParseDuration("20m")
	if err != nil {
		resp.Diagnostics.AddError("Error parsing timeout", err.Error())
		return
	}
	timeout, diag := dataModel.Timeouts.Create(ctx, defaultTimeout)
	resp.Diagnostics.Append(diag...)
	if diag.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err = autocrud.Create(ctx, r.clientGetter, r.APIVersion, r.Kind, &dataModel)
	if err != nil {
		resp.Diagnostics.AddError("Error creating resource", err.Error())
		return
	}

	r.AfterCreate(ctx, req, resp, &dataModel)

	diags := resp.State.Set(ctx, &dataModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *Secret) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var dataModel SecretModel

	diag := req.State.Get(ctx, &dataModel)
	resp.Diagnostics.Append(diag...)
	if diag.HasError() {
		return
	}

	defaultTimeout, err := time.ParseDuration("20m")
	if err != nil {
		resp.Diagnostics.AddError("Error parsing timeout", err.Error())
		return
	}
	timeout, diag := dataModel.Timeouts.Read(ctx, defaultTimeout)
	resp.Diagnostics.Append(diag...)
	if diag.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var id string
	req.State.GetAttribute(ctx, path.Root("id"), &id)
	err = autocrud.Read(ctx, r.clientGetter, r.Kind, r.APIVersion, id, &dataModel)
	if err != nil {
		resp.Diagnostics.AddError("Error reading resource", err.Error())
		return
	}

	r.AfterRead(ctx, req, resp, &dataModel)

	diags := resp.State.Set(ctx, &dataModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *Secret) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var dataModel SecretModel

	diag := req.Config.Get(ctx, &dataModel)
	resp.Diagnostics.Append(diag...)
	if diag.HasError() {
		return
	}

	r.BeforeUpdate(ctx, req, resp, &dataModel)

	defaultTimeout, err := time.ParseDuration("20m")
	if err != nil {
		resp.Diagnostics.AddError("Error parsing timeout", err.Error())
		return
	}
	timeout, diag := dataModel.Timeouts.Update(ctx, defaultTimeout)
	resp.Diagnostics.Append(diag...)
	if diag.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err = autocrud.Update(ctx, r.clientGetter, r.Kind, r.APIVersion, &dataModel)
	if err != nil {
		resp.Diagnostics.AddError("Error updating resource", err.Error())
		return
	}

	r.AfterUpdate(ctx, req, resp, &dataModel)

	diags := resp.State.Set(ctx, &dataModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *Secret) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	waitForDeletion := false

	var dataModel SecretModel

	diag := req.State.Get(ctx, &dataModel)
	resp.Diagnostics.Append(diag...)
	if diag.HasError() {
		return
	}

	defaultTimeout, err := time.ParseDuration("20m")
	if err != nil {
		resp.Diagnostics.AddError("Error parsing timeout", err.Error())
		return
	}
	timeout, diag := dataModel.Timeouts.Delete(ctx, defaultTimeout)
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

func (r *Secret) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
