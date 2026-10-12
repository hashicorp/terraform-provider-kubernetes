// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

var (
	_ resource.Resource                    = (*ClusterRoleBinding)(nil)
	_ resource.ResourceWithConfigure       = (*ClusterRoleBinding)(nil)
	_ resource.ResourceWithImportState     = (*ClusterRoleBinding)(nil)
	_ resource.ResourceWithIdentity        = (*ClusterRoleBinding)(nil)
	_ resource.ResourceWithMoveState       = (*ClusterRoleBinding)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*ClusterRoleBinding)(nil)
)

// ClusterRoleBinding is the Plugin Framework implementation of the
// kubernetes_cluster_role_binding_v1 resource. It is served alongside the SDKv2
// provider through the mux server. The schema intentionally uses blocks (rather
// than nested attributes) so that both the HCL configuration syntax and the
// persisted state layout remain compatible with the SDKv2 version.
type ClusterRoleBinding struct {
	// SDKv2Meta returns the provider meta struct configured by the SDKv2
	// provider (shared through the mux server).
	SDKv2Meta func() any
}

func NewClusterRoleBinding() resource.Resource {
	return &ClusterRoleBinding{}
}

func (r *ClusterRoleBinding) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_role_binding_v1"
}

func (r *ClusterRoleBinding) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	meta, ok := req.ProviderData.(func() any)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", "Expected the SDKv2 provider metadata accessor.")
		return
	}
	r.SDKv2Meta = meta
}

func (r *ClusterRoleBinding) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = common.IdentitySchema()
}

func (r *ClusterRoleBinding) UpgradeIdentity(_ context.Context) map[int64]resource.IdentityUpgrader {
	return common.UpgradeIdentity(clusterRoleBindingKind, clusterRoleBindingAPIVersion)
}
