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
)

var (
	_ resource.Resource                    = (*IngressClassV1)(nil)
	_ resource.ResourceWithConfigure       = (*IngressClassV1)(nil)
	_ resource.ResourceWithImportState     = (*IngressClassV1)(nil)
	_ resource.ResourceWithIdentity        = (*IngressClassV1)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*IngressClassV1)(nil)
	_ resource.ResourceWithMoveState       = (*IngressClassV1)(nil)
)

type IngressClassV1 struct {
	SDKv2Meta func() any
}

func NewIngressClassV1() resource.Resource {
	return &IngressClassV1{}
}

func (r *IngressClassV1) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ingress_class_v1"
}

func (r *IngressClassV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	meta, ok := req.ProviderData.(func() any)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected func() any, got %T.", req.ProviderData))
		return
	}
	r.SDKv2Meta = meta
}

func (r *IngressClassV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughWithIdentity(ctx, path.Root("id"), path.Root("name"), req, resp)
}

func (r *IngressClassV1) UpgradeIdentity(context.Context) map[int64]resource.IdentityUpgrader {
	return common.UpgradeIdentity("IngressClass", "networking.k8s.io/v1")
}

func ingressClassIdentity(name string) common.ResourceIdentity {
	return common.ResourceIdentity{
		Name: types.StringValue(name), Kind: types.StringValue("IngressClass"),
		APIVersion: types.StringValue("networking.k8s.io/v1"),
	}
}
