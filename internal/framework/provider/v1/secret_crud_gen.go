package v1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/autocrud"
)

func (r *Secret) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var dataModel SecretModel

	diag := req.Config.Get(ctx, &dataModel)
	resp.Diagnostics.Append(diag...)
	if diag.HasError() {
		return
	}

	err := autocrud.Create(ctx, r.clientGetter, r.APIVersion, r.Kind, &dataModel)
	if err != nil {
		resp.Diagnostics.AddError("Error creating resource", err.Error())
	}
	diags := resp.State.Set(ctx, &dataModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *Secret) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var dataModel SecretModel

	err := autocrud.Read(ctx, r.clientGetter, r.Kind, r.APIVersion, req, &dataModel)
	if err != nil {
		resp.Diagnostics.AddError("Error reading resource", err.Error())
	}
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

	err := autocrud.Update(ctx, r.clientGetter, r.Kind, r.APIVersion, req, &dataModel)
	if err != nil {
		resp.Diagnostics.AddError("Error updating resource", err.Error())
	}
	diags := resp.State.Set(ctx, &dataModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *Secret) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	err := autocrud.Delete(ctx, r.clientGetter, r.Kind, r.APIVersion, req)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting resource", err.Error())
	}
}

func (r *Secret) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
