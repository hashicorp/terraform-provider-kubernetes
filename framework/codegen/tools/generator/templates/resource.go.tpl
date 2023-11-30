package {{ .ResourceConfig.Package }}

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &{{ .ResourceConfig.Kind }}{}
var _ resource.ResourceWithImportState = &{{ .ResourceConfig.Kind }}{}

func New{{ .ResourceConfig.Kind }}() resource.Resource {
	return &{{ .ResourceConfig.Kind }}{
		Kind: "{{ .ResourceConfig.Kind }}",
		APIVersion: "{{ .ResourceConfig.APIVersion }}",
    }
}

type {{ .ResourceConfig.Kind }} struct {
	APIVersion string
	Kind       string

	clientGetter KubernetesClientGetter
}


// FIXME move this
type KubernetesClientGetter interface {
	DynamicClient() (dynamic.Interface, error)
	DiscoveryClient() (discovery.DiscoveryInterface, error)
}

func (r *{{ .ResourceConfig.Kind }}) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "{{ .ResourceConfig.Name }}"
}

func (r *{{ .ResourceConfig.Kind }}) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	clientGetter, ok := req.ProviderData.(KubernetesClientGetter)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected KubernetesClientGetter, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.clientGetter = clientGetter
}