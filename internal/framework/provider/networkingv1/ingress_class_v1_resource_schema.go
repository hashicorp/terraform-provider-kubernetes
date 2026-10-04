// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package networkingv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	networking "k8s.io/api/networking/v1"
)

func (r *IngressClassV1) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	specDoc := networking.IngressClassSpec{}.SwaggerDoc()
	parametersDoc := networking.IngressClassParametersReference{}.SwaggerDoc()
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "Ingresses can be implemented by different controllers, often with different configuration. Each Ingress should specify a class, a reference to an IngressClass resource that contains additional configuration including the name of the controller that should implement the class.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": networkingMetadataSchema("ingress_class_v1", false),
			"spec": schema.ListNestedBlock{
				Description: "Specification of the IngressClass. Exactly one spec block is required.",
				Validators: []validator.List{
					listvalidator.IsRequired(),
					listvalidator.SizeBetween(1, 1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"controller": schema.StringAttribute{
							Description: specDoc["controller"],
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString(""),
						},
						"parameters": schema.SingleNestedAttribute{
							Description: specDoc["parameters"],
							Optional:    true,
							Attributes: map[string]schema.Attribute{
								"api_group": schema.StringAttribute{
									Description: parametersDoc["apiGroup"],
									Optional:    true,
									Computed:    true,
									Default:     stringdefault.StaticString(""),
								},
								"kind": schema.StringAttribute{
									Description: parametersDoc["kind"],
									Required:    true,
								},
								"name": schema.StringAttribute{
									Description: parametersDoc["name"],
									Required:    true,
								},
								"scope": schema.StringAttribute{
									Description: parametersDoc["scope"],
									Optional:    true,
									Computed:    true,
									Validators: []validator.String{
										stringvalidator.OneOf("Cluster", "Namespace"),
									},
								},
								"namespace": schema.StringAttribute{
									Description: parametersDoc["namespace"],
									Optional:    true,
									Computed:    true,
									Default:     stringdefault.StaticString(""),
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *IngressClassV1) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = common.IdentitySchema()
}
