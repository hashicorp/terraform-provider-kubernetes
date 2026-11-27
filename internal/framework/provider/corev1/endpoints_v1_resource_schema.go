// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

// Schema implements [resource.Resource].
func (e *EndpointsV1) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An Endpoints resource is an abstraction, linked to a Service, which defines the list of endpoints that actually implement the service.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The unique ID for this terraform resource in the form `namespace/name`.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": common.NamespacedMetadataSchema("endpoints", true),
			"subset": schema.SetNestedBlock{
				Description: "Set of addresses and ports that comprise a service. More info: https://kubernetes.io/docs/concepts/services-networking/service/#services-without-selectors",
				NestedObject: schema.NestedBlockObject{
					Blocks: map[string]schema.Block{
						"address": schema.SetNestedBlock{
							Description: "IP address which offers the related ports that are marked as ready. These endpoints should be considered safe for load balancers and clients to utilize.",
							Validators: []validator.Set{
								setvalidator.SizeAtLeast(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"hostname": schema.StringAttribute{
										Description: "The Hostname of this endpoint.",
										Optional:    true,
									},
									"ip": schema.StringAttribute{
										Description: "The IP of this endpoint. May not be loopback (127.0.0.0/8), link-local (169.254.0.0/16), or link-local multicast ((224.0.0.0/24).",
										Required:    true,
									},
									"node_name": schema.StringAttribute{
										Description: "Node hosting this endpoint. This can be used to determine endpoints local to a node.",
										Optional:    true,
									},
								},
							},
						},
						"not_ready_address": schema.SetNestedBlock{
							Description: "IP address which offers the related ports but is not currently marked as ready because it have not yet finished starting, have recently failed a readiness check, or have recently failed a liveness check.",
							Validators: []validator.Set{
								setvalidator.SizeAtLeast(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"hostname": schema.StringAttribute{
										Description: "The Hostname of this endpoint.",
										Optional:    true,
									},
									"ip": schema.StringAttribute{
										Description: "The IP of this endpoint. May not be loopback (127.0.0.0/8), link-local (169.254.0.0/16), or link-local multicast ((224.0.0.0/24).",
										Required:    true,
									},
									"node_name": schema.StringAttribute{
										Description: "Node hosting this endpoint. This can be used to determine endpoints local to a node.",
										Optional:    true,
									},
								},
							},
						},
						"port": schema.SetNestedBlock{
							Description: "Port number available on the related IP addresses.",
							Validators: []validator.Set{
								setvalidator.SizeAtLeast(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{
										Description: "The name of this port within the endpoint. Must be a DNS_LABEL. Optional if only one Port is defined on this endpoint.",
										Optional:    true,
									},
									"port": schema.Int64Attribute{
										Description: "The port that will be exposed by this endpoint.",
										Required:    true,
									},
									"protocol": schema.StringAttribute{
										Description: "The IP protocol for this port. Supports `TCP` and `UDP`. Default is `TCP`.",
										Optional:    true,
										Computed:    true,
										Default:     stringdefault.StaticString("TCP"),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// IdentitySchema implements [resource.ResourceWithIdentity].
func (e *EndpointsV1) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = common.NamespacedIdentitySchema()
}
