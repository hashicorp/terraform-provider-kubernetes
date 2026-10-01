// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package nodev1

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

const (
	runtimeClassAPIVersion = "node.k8s.io/v1"
	runtimeClassKind       = "RuntimeClass"
)

var (
	_ resource.Resource                    = (*RuntimeClassV1)(nil)
	_ resource.ResourceWithConfigure       = (*RuntimeClassV1)(nil)
	_ resource.ResourceWithIdentity        = (*RuntimeClassV1)(nil)
	_ resource.ResourceWithImportState     = (*RuntimeClassV1)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*RuntimeClassV1)(nil)
)

type RuntimeClassV1 struct {
	// SDKv2Meta must stay func() any: that is the concrete type stored in ProviderData.
	// Go function types are invariant, so assert the *result* instead — see sdkv2Meta().
	SDKv2Meta func() any
}

func NewRuntimeClassV1() resource.Resource {
	return &RuntimeClassV1{}
}

// Metadata implements [resource.Resource].
func (r *RuntimeClassV1) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_runtime_class_v1"
}

// Configure implements [resource.ResourceWithConfigure].
func (r *RuntimeClassV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	sdkv2Meta, ok := req.ProviderData.(func() any)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data",
			fmt.Sprintf("Expected func() any, got %T. This is a bug in the provider.", req.ProviderData),
		)
		return
	}
	r.SDKv2Meta = sdkv2Meta
}

// sdkv2Meta resolves the SDKv2 provider metadata per request rather than in Configure,
// because the mux configures the SDKv2 provider independently and its meta may not be
// populated when Configure runs.
func (r *RuntimeClassV1) sdkv2Meta() (kubernetes.KubeClientsets, kubernetes.MetadataFilters, diag.Diagnostics) {
	var diags diag.Diagnostics

	if r.SDKv2Meta == nil {
		diags.AddError("Provider not configured",
			"The SDKv2 provider metadata is unavailable. This is a bug in the provider.")
		return nil, nil, diags
	}

	meta := r.SDKv2Meta()
	clients, ok := meta.(kubernetes.KubeClientsets)
	if !ok {
		diags.AddError("Unexpected provider data",
			fmt.Sprintf("Expected kubernetes.KubeClientsets, got %T. This is a bug in the provider.", meta))
		return nil, nil, diags
	}

	filters, ok := meta.(kubernetes.MetadataFilters)
	if !ok {
		diags.AddError("Unexpected provider data",
			fmt.Sprintf("Expected kubernetes.MetadataFilters, got %T. This is a bug in the provider.", meta))
		return nil, nil, diags
	}

	return clients, filters, diags
}

// IdentitySchema implements [resource.ResourceWithIdentity].
//
// The SDKv2 resource never declared an identity, so this is new with the migration.
// common.IdentitySchema is used anyway, at Version 1, so this resource matches every other
// cluster-scoped resource in the provider.
func (r *RuntimeClassV1) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = common.IdentitySchema()
}

// ImportState implements [resource.ResourceWithImportState]. It mirrors SDKv2's
// ImportStatePassthroughContext: only the ID is set, and Read populates everything else.
func (r *RuntimeClassV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughWithIdentity(ctx, path.Root("id"), path.Root("name"), req, resp)
}

// UpgradeIdentity implements [resource.ResourceWithUpgradeIdentity].
//
// No released SDKv2 version of this resource wrote an identity: state from 2.37.1 through
// 3.2.1 carries identity_schema_version 0 and a null identity. Because the declared identity
// is Version 1, Terraform asks for an upgrade on the first plan after the provider upgrade,
// and without this every existing object fails with "Unable to Upgrade Resource Identity".
func (r *RuntimeClassV1) UpgradeIdentity(_ context.Context) map[int64]resource.IdentityUpgrader {
	return common.UpgradeIdentity(runtimeClassKind, runtimeClassAPIVersion)
}
