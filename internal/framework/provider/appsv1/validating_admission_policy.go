package appsv1

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-codegen-kubernetes/autocrud"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ValidatingAdmissionPolicy{}
var _ resource.ResourceWithImportState = &ValidatingAdmissionPolicy{}

func NewValidatingAdmissionPolicy() resource.Resource {
	return &ValidatingAdmissionPolicy{
		Kind:       "ValidatingAdmissionPolicy",
		APIVersion: "apps/v1",
	}
}

type ValidatingAdmissionPolicy struct {
	APIVersion string
	Kind       string

	clientGetter autocrud.KubernetesClientGetter
}

func (r *ValidatingAdmissionPolicy) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kubernetes_validating_admission_policy_v1_gen"
}

func (r *ValidatingAdmissionPolicy) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	clientGetter, ok := req.ProviderData.(autocrud.KubernetesClientGetter)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected KubernetesClientGetter, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.clientGetter = clientGetter
}
