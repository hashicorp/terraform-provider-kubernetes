// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

// Compile-time interface assertions — ensure RoleBindingV1 implements all
// required Plugin Framework interfaces.
var (
	_ resource.Resource                    = (*RoleBindingV1)(nil)
	_ resource.ResourceWithConfigure       = (*RoleBindingV1)(nil)
	_ resource.ResourceWithImportState     = (*RoleBindingV1)(nil)
	_ resource.ResourceWithIdentity        = (*RoleBindingV1)(nil)
	_ resource.ResourceWithMoveState       = (*RoleBindingV1)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*RoleBindingV1)(nil)
)

// RoleBindingV1 is the Plugin Framework resource for kubernetes_role_binding_v1.
type RoleBindingV1 struct {
	// SDKv2Meta is a function that returns the provider metadata (KubeClientsets).
	// It bridges the mux setup where the framework provider delegates client
	// creation to the SDKv2 provider configuration.
	SDKv2Meta func() any
}

// NewRoleBindingV1 returns a new instance of RoleBindingV1 as a resource.Resource.
// Registered in the framework provider's Resources() method.
func NewRoleBindingV1() resource.Resource {
	return &RoleBindingV1{}
}

// Metadata sets the Terraform resource type name to "kubernetes_role_binding_v1".
func (r *RoleBindingV1) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_binding_v1"
}

// Configure stores the SDKv2Meta function passed through from the mux provider.
func (r *RoleBindingV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.SDKv2Meta = req.ProviderData.(func() any)
}

// sdkv2Meta resolves the SDKv2 provider metadata into the two interfaces this resource
// needs. The call is deferred until the request rather than made in Configure, because the
// mux server configures the SDKv2 provider independently and its meta is not populated
// until that happens.
func (r *RoleBindingV1) sdkv2Meta() (kubernetes.KubeClientsets, kubernetes.MetadataFilters, diag.Diagnostics) {
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

// IdentitySchema defines the identity schema for kubernetes_role_binding_v1.
// RoleBindings are namespaced; delegates to common.NamespacedIdentitySchema which mirrors
// the SDKv2 resourceIdentitySchemaNamespaced contract (Version: 1, namespace OptionalForImport,
// others RequiredForImport).
func (r *RoleBindingV1) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = common.NamespacedIdentitySchema()
}

// UpgradeIdentity implements [resource.ResourceWithUpgradeIdentity].
//
// Without this, any object created by provider 2.37.x or older fails its first plan with
// "Unable to Upgrade Resource Identity": identity shipped in 2.38.0, so older state carries
// identity_schema_version 0 and no identity, and Terraform asks for an upgrade whenever the
// stored version differs from the declared one. SDKv2 answers that generically in its gRPC
// server; the framework requires each resource to supply it.
func (r *RoleBindingV1) UpgradeIdentity(ctx context.Context) map[int64]resource.IdentityUpgrader {
	return common.UpgradeNamespacedIdentity(roleBindingKind, roleBindingAPIVersion)
}

// MoveState returns the StateMover handlers that enable `moved {}` block support
// for migrating from the deprecated kubernetes_role_binding (SDKv2) resource.
func (r *RoleBindingV1) MoveState(_ context.Context) []resource.StateMover {
	return moveStateHandlers()
}
