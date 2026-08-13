// Copyright (c) HashiCorp, Inc.

package admissionregistrationv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
)

var (
	_ resource.Resource                = (*ValidatingAdmissionPolicyBinding)(nil)
	_ resource.ResourceWithConfigure   = (*ValidatingAdmissionPolicyBinding)(nil)
	_ resource.ResourceWithImportState = (*ValidatingAdmissionPolicyBinding)(nil)
	_ resource.ResourceWithIdentity    = (*ValidatingAdmissionPolicyBinding)(nil)
)

type ValidatingAdmissionPolicyBinding struct {
	SDKv2Meta func() any
}

func NewValidatingAdmissionPolicyBinding() resource.Resource {
	return &ValidatingAdmissionPolicyBinding{}
}

func (r *ValidatingAdmissionPolicyBinding) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_validating_admission_policy_binding_v1"
}

func (r *ValidatingAdmissionPolicyBinding) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.SDKv2Meta = req.ProviderData.(func() any)
}

func (r *ValidatingAdmissionPolicyBinding) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"api_version": identityschema.StringAttribute{
				RequiredForImport: true,
			},
			"kind": identityschema.StringAttribute{
				RequiredForImport: true,
			},
			"name": identityschema.StringAttribute{
				RequiredForImport: true,
			},
		},
	}
}
