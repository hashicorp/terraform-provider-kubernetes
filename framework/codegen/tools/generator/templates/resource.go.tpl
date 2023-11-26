package {{ .ResourceConfig.Package }}

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &{{ .ResourceConfig.Kind }}{}
var _ resource.ResourceWithImportState = &{{ .ResourceConfig.Kind }}{}

func New{{ .ResourceConfig.Kind }}() resource.Resource {
	return &{{ .ResourceConfig.Kind }}{}
}

type {{ .ResourceConfig.Kind }} struct {
}

func (r *{{ .ResourceConfig.Kind }}) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "{{ .ResourceConfig.Name }}"
}

func (r *{{ .ResourceConfig.Kind }}) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
}