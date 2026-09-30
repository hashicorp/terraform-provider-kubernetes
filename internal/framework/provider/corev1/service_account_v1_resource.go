// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	clientset "k8s.io/client-go/kubernetes"
)

var (
	_ resource.Resource                    = (*ServiceAccountV1)(nil)
	_ resource.ResourceWithConfigure       = (*ServiceAccountV1)(nil)
	_ resource.ResourceWithIdentity        = (*ServiceAccountV1)(nil)
	_ resource.ResourceWithImportState     = (*ServiceAccountV1)(nil)
	_ resource.ResourceWithMoveState       = (*ServiceAccountV1)(nil)
	_ resource.ResourceWithModifyPlan      = (*ServiceAccountV1)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*ServiceAccountV1)(nil)
)

type ServiceAccountV1 struct {
	SDKv2Meta func() any
}

func NewServiceAccountV1() resource.Resource {
	return &ServiceAccountV1{}
}

func (r *ServiceAccountV1) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account_v1"
}

func (r *ServiceAccountV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	serviceAccountConfigure(req, resp, &r.SDKv2Meta)
}

func serviceAccountConfigure(req resource.ConfigureRequest, resp *resource.ConfigureResponse, target *func() any) {
	if req.ProviderData == nil {
		return
	}
	callback, ok := req.ProviderData.(func() any)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected func() any, got %T. This is a bug in the provider.", req.ProviderData))
		return
	}
	*target = callback
}

func serviceAccountClient(callback func() any) (*clientset.Clientset, kubernetes.MetadataFilters, diag.Diagnostics) {
	var diags diag.Diagnostics
	if callback == nil {
		diags.AddError("Provider not configured", "The SDKv2 provider metadata is unavailable.")
		return nil, nil, diags
	}
	meta := callback()
	if meta == nil || (reflect.ValueOf(meta).Kind() == reflect.Ptr && reflect.ValueOf(meta).IsNil()) {
		diags.AddError("Provider not configured", "The SDKv2 provider metadata is unavailable.")
		return nil, nil, diags
	}
	clients, ok := meta.(kubernetes.KubeClientsets)
	if !ok {
		diags.AddError("Unexpected provider data", fmt.Sprintf("Expected kubernetes.KubeClientsets, got %T.", meta))
		return nil, nil, diags
	}
	filters, ok := meta.(kubernetes.MetadataFilters)
	if !ok {
		diags.AddError("Unexpected provider data", fmt.Sprintf("Expected kubernetes.MetadataFilters, got %T.", meta))
		return nil, nil, diags
	}
	conn, err := clients.MainClientset()
	if err != nil {
		diags.AddError("Kubernetes client error", err.Error())
	} else if conn == nil {
		diags.AddError("Kubernetes client error", "The main clientset is unavailable.")
	}
	return conn, filters, diags
}

func (r *ServiceAccountV1) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = common.NamespacedIdentitySchema()
}

func (r *ServiceAccountV1) UpgradeIdentity(context.Context) map[int64]resource.IdentityUpgrader {
	return common.UpgradeNamespacedIdentity(serviceAccountKind, serviceAccountAPIVersion)
}

func (r *ServiceAccountV1) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	serviceAccountCreate(ctx, req, resp, r.SDKv2Meta, false)
}

func (r *ServiceAccountV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	serviceAccountRead(ctx, req, resp, r.SDKv2Meta, false)
}

func (r *ServiceAccountV1) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	serviceAccountUpdate(ctx, req, resp, r.SDKv2Meta, false)
}

func (r *ServiceAccountV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	serviceAccountDelete(ctx, req, resp, r.SDKv2Meta, false)
}

func (r *ServiceAccountV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	serviceAccountImport(ctx, req, resp, r.SDKv2Meta, false)
}

func (r *ServiceAccountV1) MoveState(ctx context.Context) []resource.StateMover {
	return serviceAccountMoveState(ctx, false)
}

func (r *ServiceAccountV1) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	serviceAccountModifyPlan(ctx, req, resp)
}
