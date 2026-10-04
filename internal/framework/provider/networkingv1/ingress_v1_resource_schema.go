// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package networkingv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	networking "k8s.io/api/networking/v1"
)

func (r *IngressV1) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "Ingress is a collection of rules that allow inbound connections to reach the endpoints defined by a backend. An Ingress can be configured to give services externally-reachable urls, load balance traffic, terminate SSL, offer name based virtual hosting etc.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"wait_for_load_balancer": schema.BoolAttribute{
				Optional:    true,
				Description: "Terraform will wait for the load balancer to have at least 1 endpoint before considering the resource created.",
			},
			"status": schema.ListAttribute{
				Computed:    true,
				ElementType: ingressStatusType(),
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": networkingMetadataSchema("ingress", true),
			"spec":     ingressSpecSchema(),
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Delete: true}),
		},
	}
}

func ingressSpecSchema() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: networking.Ingress{}.SwaggerDoc()["spec"],
		Validators: []validator.List{
			listvalidator.IsRequired(),
			listvalidator.SizeBetween(1, 1),
		},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"ingress_class_name": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: networking.IngressSpec{}.SwaggerDoc()["ingressClassName"],
				},
				"default_backend": ingressBackendSchema("A default backend capable of servicing requests that don't match any rule. At least one of 'backend' or 'rules' must be specified. This field is optional to allow the loadbalancer controller or defaulting logic to specify a global default."),
			},
			Blocks: map[string]schema.Block{
				"rule": schema.ListNestedBlock{
					Description: networking.IngressSpec{}.SwaggerDoc()["rules"],
					NestedObject: schema.NestedBlockObject{
						Attributes: map[string]schema.Attribute{
							"host": ingressOptionalString(networking.IngressRule{}.SwaggerDoc()["host"]),
							"http": schema.SingleNestedAttribute{
								Description: "HTTP routing paths and their backends.",
								Optional:    true,
								Attributes: map[string]schema.Attribute{
									"path": ingressPathSchema(),
								},
							},
						},
					},
				},
				"tls": schema.ListNestedBlock{
					Description: networking.IngressSpec{}.SwaggerDoc()["tls"],
					NestedObject: schema.NestedBlockObject{
						Attributes: map[string]schema.Attribute{
							"hosts": schema.ListAttribute{
								Optional:    true,
								Computed:    true,
								ElementType: types.StringType,
								Description: networking.IngressTLS{}.SwaggerDoc()["hosts"],
								PlanModifiers: []planmodifier.List{
									networkingEmptyCollectionPlanModifier{},
								},
							},
							"secret_name": ingressOptionalString(networking.IngressTLS{}.SwaggerDoc()["secretName"]),
						},
					},
				},
			},
		},
	}
}

func ingressPathSchema() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Description: networking.HTTPIngressRuleValue{}.SwaggerDoc()["paths"],
		Required:    true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"path": ingressOptionalString(networking.HTTPIngressPath{}.SwaggerDoc()["path"]),
				"path_type": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Default:     stringdefault.StaticString(string(networking.PathTypeImplementationSpecific)),
					Description: networking.HTTPIngressPath{}.SwaggerDoc()["pathType"],
					Validators: []validator.String{
						stringvalidator.OneOf(
							string(networking.PathTypeImplementationSpecific), string(networking.PathTypePrefix), string(networking.PathTypeExact),
						),
					},
				},
				"backend": ingressBackendSchema("Backend defines the referenced service endpoint to which the traffic will be forwarded to."),
			},
		},
		Validators: []validator.List{
			listvalidator.SizeAtLeast(1),
		},
	}
}

func ingressBackendSchema(description string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Description: description,
		Optional:    true,
		Attributes: map[string]schema.Attribute{
			"resource": schema.SingleNestedAttribute{
				Description: "Resource is an ObjectRef to another Kubernetes resource in the namespace of the Ingress object. If resource is specified, a service.Name and service.Port must not be specified.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"api_group": schema.StringAttribute{
						Required:    true,
						Description: "APIGroup is the group for the resource being referenced.",
					},
					"kind": schema.StringAttribute{
						Required:    true,
						Description: "The kind of resource.",
					},
					"name": schema.StringAttribute{
						Required:    true,
						Description: "The name of the referenced resource.",
					},
				},
			},
			"service": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required:    true,
						Description: "Specifies the name of the referenced service.",
					},
					"port": schema.SingleNestedAttribute{
						Description: "Specifies the port of the referenced service.",
						Required:    true,
						Attributes: map[string]schema.Attribute{
							"name": ingressOptionalString("Specifies the name of the port of the referenced service."),
							"number": schema.Int64Attribute{
								Optional:    true,
								Computed:    true,
								Default:     int64default.StaticInt64(0),
								Description: "Specifies the numerical port of the referenced service.",
							},
						},
					},
				},
			},
		},
	}
}

// SDKv2's implicit scalar zero is observable even when an optional field is omitted.
func ingressOptionalString(description string) schema.StringAttribute {
	return schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Default:     stringdefault.StaticString(""),
		Description: description,
	}
}

func ingressStatusType() types.ObjectType {
	endpoint := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"ip":       types.StringType,
			"hostname": types.StringType,
		},
	}
	loadBalancer := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"ingress": types.ListType{
				ElemType: endpoint,
			},
		},
	}
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"load_balancer": types.ListType{
				ElemType: loadBalancer,
			},
		},
	}
}
