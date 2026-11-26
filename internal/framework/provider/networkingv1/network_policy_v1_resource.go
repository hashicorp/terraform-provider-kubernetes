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

const (
	networkPolicyAPIVersion = "networking.k8s.io/v1"
	networkPolicyKind       = "NetworkPolicy"
)

var (
	_ resource.Resource                    = (*NetworkPolicyV1)(nil)
	_ resource.ResourceWithConfigure       = (*NetworkPolicyV1)(nil)
	_ resource.ResourceWithIdentity        = (*NetworkPolicyV1)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*NetworkPolicyV1)(nil)
	_ resource.ResourceWithImportState     = (*NetworkPolicyV1)(nil)
	_ resource.ResourceWithMoveState       = (*NetworkPolicyV1)(nil)
)

type NetworkPolicyV1 struct {
	SDKv2Meta func() any
}

func NewNetworkPolicyV1() resource.Resource {
	return &NetworkPolicyV1{}
}

func (r *NetworkPolicyV1) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_policy_v1"
}

func (r *NetworkPolicyV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NetworkPolicyV1) UpgradeIdentity(context.Context) map[int64]resource.IdentityUpgrader {
	return common.UpgradeNamespacedIdentity(networkPolicyKind, networkPolicyAPIVersion)
}

func networkPolicyIdentity(namespace, name string) common.NamespacedResourceIdentity {
	return common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{
			Name:       types.StringValue(name),
			Kind:       types.StringValue(networkPolicyKind),
			APIVersion: types.StringValue(networkPolicyAPIVersion),
		},
		Namespace: types.StringValue(namespace),
	}
}

func (r *NetworkPolicyV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := req.ID
	if id == "" {
		if req.Identity == nil {
			resp.Diagnostics.AddError("Missing import identifier", "Supply the network policy's namespace/name ID or identity.")
			return
		}
		var identity common.NamespacedResourceIdentity
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		id = identity.Namespace.ValueString() + "/" + identity.Name.ValueString()
	}
	namespace, name, err := kubernetes.IdParts(id)
	if err != nil {
		resp.Diagnostics.AddError("Invalid network policy import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, networkPolicyIdentity(namespace, name))...)
	}
}
