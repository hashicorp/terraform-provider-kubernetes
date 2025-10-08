// Copyright (c) HashiCorp, Inc.

package admissionregistrationv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var (
	_ resource.Resource                = (*ValidatingAdmissionPolicy)(nil)
	_ resource.ResourceWithConfigure   = (*ValidatingAdmissionPolicy)(nil)
	_ resource.ResourceWithImportState = (*ValidatingAdmissionPolicy)(nil)
)

type ValidatingAdmissionPolicy struct {
	SDKv2Meta func() any
}

func NewValidatingAdmissionPolicy() resource.Resource {
	return &ValidatingAdmissionPolicy{}
}

func (r *ValidatingAdmissionPolicy) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_validating_admission_policy"
}

func (r *ValidatingAdmissionPolicy) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.SDKv2Meta = req.ProviderData.(func() any)
}
