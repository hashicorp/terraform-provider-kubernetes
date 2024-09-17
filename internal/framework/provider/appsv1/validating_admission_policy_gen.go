package appsv1

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ValidatingAdmissionPolicy{}
var _ resource.ResourceWithImportState = &ValidatingAdmissionPolicy{}

func NewDValidatingAdmissionPolicy() resource.Resource {
	return &ValidatingAdmissionPolicy{
		Kind:       "ValidatingAdmissionPolicy",
		APIVersion: "apps/v1",
	}
}

type ValidatingAdmissionPolicy struct {
	APIVersion string
	Kind       string

	clientGetter client.KubernetesClientGetter
}

func (r *ValidatingAdmissionPolicy) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "kubernetes_deployment_v1_gen"
}

func (r *ValidatingAdmissionPolicy) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	clientGetter, ok := req.ProviderData.(client.KubernetesClientGetter)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected KubernetesClientGetter, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.clientGetter = clientGetter
}
