// Copyright IBM Corp. 2017, 2026

package admissionregistrationv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
)

var (
	_ resource.Resource                = (*ValidatingAdmissionPolicy)(nil)
	_ resource.ResourceWithConfigure   = (*ValidatingAdmissionPolicy)(nil)
	_ resource.ResourceWithImportState = (*ValidatingAdmissionPolicy)(nil)
	_ resource.ResourceWithIdentity    = (*ValidatingAdmissionPolicy)(nil)
)

type ValidatingAdmissionPolicy struct {
	SDKv2Meta func() any
}

func NewValidatingAdmissionPolicy() resource.Resource {
	return &ValidatingAdmissionPolicy{}
}

func (r *ValidatingAdmissionPolicy) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_validating_admission_policy_v1"
}

func (r *ValidatingAdmissionPolicy) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.SDKv2Meta = req.ProviderData.(func() any)
}

func (r *ValidatingAdmissionPolicy) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
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
