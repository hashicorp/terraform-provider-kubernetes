package corev1

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/hashicorp/terraform-plugin-codegen-kubernetes/autocrud"
)

func (r *ConfigMap) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var dataModel ConfigMapModel

	diag := req.Config.Get(ctx, &dataModel)
	resp.Diagnostics.Append(diag...)
	if diag.HasError() {
		return
	}

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

	diags := resp.State.Set(ctx, &dataModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ConfigMap) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var dataModel ConfigMapModel

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

	err = autocrud.Read(ctx, r.clientGetter, r.Kind, r.APIVersion, req, &dataModel)
	if err != nil {
		resp.Diagnostics.AddError("Error reading resource", err.Error())
		return
	}

	diags := resp.State.Set(ctx, &dataModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ConfigMap) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var dataModel ConfigMapModel

	diag := req.Config.Get(ctx, &dataModel)
	resp.Diagnostics.Append(diag...)
	if diag.HasError() {
		return
	}

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

	diags := resp.State.Set(ctx, &dataModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *ConfigMap) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	waitForDeletion := false

	var dataModel ConfigMapModel

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

func (r *ConfigMap) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
