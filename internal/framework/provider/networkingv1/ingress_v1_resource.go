// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package networkingv1

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

var (
	_ resource.Resource                    = (*IngressV1)(nil)
	_ resource.ResourceWithConfigure       = (*IngressV1)(nil)
	_ resource.ResourceWithIdentity        = (*IngressV1)(nil)
	_ resource.ResourceWithImportState     = (*IngressV1)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*IngressV1)(nil)
)

const (
	ingressAPIVersion = "networking.k8s.io/v1"
	ingressKind       = "Ingress"
)

type IngressV1 struct {
	SDKv2Meta func() any
}

func NewIngressV1() resource.Resource {
	return &IngressV1{}
}

func (r *IngressV1) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ingress_v1"
}

func (r *IngressV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	meta, ok := req.ProviderData.(func() any)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data",
			fmt.Sprintf("Expected func() any, got %T. This is a bug in the provider.", req.ProviderData))
		return
	}
	r.SDKv2Meta = meta
}

func (r *IngressV1) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = common.NamespacedIdentitySchema()
}

func (r *IngressV1) UpgradeIdentity(context.Context) map[int64]resource.IdentityUpgrader {
	return common.UpgradeNamespacedIdentity(ingressKind, ingressAPIVersion)
}

func (r *IngressV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := req.ID
	if id == "" && req.Identity != nil {
		var identity common.NamespacedResourceIdentity
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if identity.Name.IsNull() || identity.Name.IsUnknown() || identity.Name.ValueString() == "" {
			resp.Diagnostics.AddError("Invalid ingress identity", "The ingress identity must contain a nonempty name.")
			return
		}
		if identity.Kind.ValueString() != ingressKind || identity.APIVersion.ValueString() != ingressAPIVersion {
			resp.Diagnostics.AddError("Invalid ingress identity", "Expected kind Ingress and api_version networking.k8s.io/v1.")
			return
		}
		namespace := identity.Namespace.ValueString()
		if identity.Namespace.IsUnknown() {
			resp.Diagnostics.AddError("Invalid ingress identity", "The ingress namespace must be known.")
			return
		}
		if namespace == "" {
			namespace = "default"
		}
		id = namespace + "/" + identity.Name.ValueString()
	}
	namespace, name, err := ingressIDParts(id)
	if err != nil {
		resp.Diagnostics.AddError("Invalid ingress ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, ingressIdentity(namespace, name))...)
	}
}

func ingressIDParts(id string) (string, string, error) {
	namespace, name, err := kubernetes.IdParts(id)
	if err != nil {
		return "", "", err
	}
	if namespace == "" || name == "" {
		return "", "", fmt.Errorf("expected a nonempty namespace/name ID, got %q", id)
	}
	return namespace, name, nil
}

func ingressIdentity(namespace, name string) common.NamespacedResourceIdentity {
	return common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{
			Name: types.StringValue(name), Kind: types.StringValue(ingressKind),
			APIVersion: types.StringValue(ingressAPIVersion),
		},
		Namespace: types.StringValue(namespace),
	}
}
