// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
)

// Compile-time interface assertions — ensure RoleBindingV1 implements all
// required Plugin Framework interfaces.
var (
	_ resource.Resource                = (*RoleBindingV1)(nil)
	_ resource.ResourceWithConfigure   = (*RoleBindingV1)(nil)
	_ resource.ResourceWithImportState = (*RoleBindingV1)(nil)
	_ resource.ResourceWithIdentity    = (*RoleBindingV1)(nil)
	_ resource.ResourceWithMoveState   = (*RoleBindingV1)(nil)
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

// IdentitySchema defines the identity attributes for kubernetes_role_binding_v1.
// RoleBindings are namespaced so the identity carries api_version, kind, namespace, and name.
func (r *RoleBindingV1) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		// Version must match the identity_schema_version written by the SDKv2
		// provider (hashicorp/kubernetes@3.2.1) so that TestAccRoleBindingV1_upgradeFromSDKv2
		// can read existing state without a version mismatch error.
		Version: 1,
		Attributes: map[string]identityschema.Attribute{
			"api_version": identityschema.StringAttribute{
				RequiredForImport: true,
			},
			"kind": identityschema.StringAttribute{
				RequiredForImport: true,
			},
			"namespace": identityschema.StringAttribute{
				RequiredForImport: true,
			},
			"name": identityschema.StringAttribute{
				RequiredForImport: true,
			},
		},
	}
}

// MoveState returns the StateMover handlers that enable `moved {}` block support
// for migrating from the deprecated kubernetes_role_binding (SDKv2) resource.
func (r *RoleBindingV1) MoveState(_ context.Context) []resource.StateMover {
	return moveStateHandlers()
}
