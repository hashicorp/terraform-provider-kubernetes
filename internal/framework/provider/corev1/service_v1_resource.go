// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

const (
	serviceAPIVersion = "v1"
	serviceKind       = "Service"
)

var (
	_ resource.Resource                    = (*ServiceV1)(nil)
	_ resource.ResourceWithConfigure       = (*ServiceV1)(nil)
	_ resource.ResourceWithIdentity        = (*ServiceV1)(nil)
	_ resource.ResourceWithImportState     = (*ServiceV1)(nil)
	_ resource.ResourceWithMoveState       = (*ServiceV1)(nil)
	_ resource.ResourceWithModifyPlan      = (*ServiceV1)(nil)
	_ resource.ResourceWithUpgradeState    = (*ServiceV1)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*ServiceV1)(nil)
	_ resource.ResourceWithValidateConfig  = (*ServiceV1)(nil)
)

type ServiceV1 struct {
	SDKv2Meta func() any
}

func NewServiceV1() resource.Resource {
	return &ServiceV1{}
}

func (r *ServiceV1) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_v1"
}

func (r *ServiceV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	meta, ok := req.ProviderData.(func() any)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected func() any, got %T. This is a bug in the provider.", req.ProviderData))
		return
	}
	r.SDKv2Meta = meta
}

func (r *ServiceV1) sdkv2Meta() (kubernetes.KubeClientsets, kubernetes.MetadataFilters, diag.Diagnostics) {
	var diags diag.Diagnostics
	if r.SDKv2Meta == nil {
		diags.AddError("Provider not configured", "The SDKv2 provider metadata is unavailable.")
		return nil, nil, diags
	}
	meta := r.SDKv2Meta()
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
	return clients, filters, diags
}

func (r *ServiceV1) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = common.NamespacedIdentitySchema()
}

func (r *ServiceV1) UpgradeIdentity(context.Context) map[int64]resource.IdentityUpgrader {
	return common.UpgradeNamespacedIdentity(serviceKind, serviceAPIVersion)
}

func serviceIdentity(namespace, name string) common.NamespacedResourceIdentity {
	return common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{
			APIVersion: types.StringValue(serviceAPIVersion),
			Kind:       types.StringValue(serviceKind),
			Name:       types.StringValue(name),
		},
		Namespace: types.StringValue(namespace),
	}
}

func (r *ServiceV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := req.ID
	if req.Identity != nil && !req.Identity.Raw.IsNull() {
		var identity common.NamespacedResourceIdentity
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if identity.Kind.ValueString() != serviceKind || identity.APIVersion.ValueString() != serviceAPIVersion || identity.Name.IsNull() || identity.Name.IsUnknown() || identity.Name.ValueString() == "" {
			resp.Diagnostics.AddError("Invalid service import identity", "Import requires api_version = \"v1\", kind = \"Service\", and a nonempty name.")
			return
		}
		namespace := identity.Namespace.ValueString()
		if identity.Namespace.IsUnknown() {
			resp.Diagnostics.AddError("Invalid service import identity", "The namespace must be known.")
			return
		}
		if namespace == "" {
			namespace = "default"
		}
		id = namespace + "/" + identity.Name.ValueString()
	}
	namespace, name, err := kubernetes.IdParts(id)
	if err != nil || namespace == "" || name == "" {
		resp.Diagnostics.AddError("Invalid service import ID", "The import ID must have the form namespace/name.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, serviceIdentity(namespace, name))...)
	}
}
