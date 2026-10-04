// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package networkingv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	networking "k8s.io/api/networking/v1"
)

func (r *NetworkPolicyV1) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	specDoc := networking.NetworkPolicySpec{}.SwaggerDoc()
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "Kubernetes supports network policies to specify how groups of pods are allowed to communicate with each other and with other network endpoints. NetworkPolicy resources use labels to select pods and define rules which specify what traffic is allowed to the selected pods. Read more about network policies at https://kubernetes.io/docs/concepts/services-networking/network-policies/",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": networkingMetadataSchema("network policy", true),
			"spec": schema.ListNestedBlock{
				Description: networking.NetworkPolicy{}.SwaggerDoc()["spec"],
				Validators:  networkPolicyOneBlock(true),
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"policy_types": schema.ListAttribute{
							Description: specDoc["policyTypes"],
							Required:    true,
							ElementType: types.StringType,
							Validators: []validator.List{
								listvalidator.SizeBetween(1, 2),
							},
						},
						"pod_selector": networkPolicySelectorBlock(specDoc["podSelector"], true),
					},
					Blocks: map[string]schema.Block{
						"ingress": schema.ListNestedBlock{
							Description: specDoc["ingress"],
							NestedObject: schema.NestedBlockObject{
								Blocks: map[string]schema.Block{
									"ports": networkPolicyPortsBlock(networking.NetworkPolicyIngressRule{}.SwaggerDoc()["ports"]),
									"from":  networkPolicyPeersBlock(networking.NetworkPolicyIngressRule{}.SwaggerDoc()["from"]),
								},
							},
						},
						"egress": schema.ListNestedBlock{
							Description: specDoc["egress"],
							NestedObject: schema.NestedBlockObject{
								Blocks: map[string]schema.Block{
									"ports": networkPolicyPortsBlock(networking.NetworkPolicyEgressRule{}.SwaggerDoc()["ports"]),
									"to":    networkPolicyPeersBlock(networking.NetworkPolicyEgressRule{}.SwaggerDoc()["to"]),
								},
							},
						},
					},
				},
			},
		},
	}
}

func networkPolicyOneBlock(required bool) []validator.List {
	if required {
		return []validator.List{
			listvalidator.IsRequired(),
			listvalidator.SizeBetween(1, 1),
		}
	}
	return []validator.List{
		listvalidator.SizeAtMost(1),
	}
}

// SDKv2 writes zero values for omitted optional scalars, even without a Default.
func networkPolicyOptionalString(description string) schema.StringAttribute {
	return schema.StringAttribute{
		Description: description,
		Optional:    true,
		Computed:    true,
		Default:     stringdefault.StaticString(""),
	}
}

func networkPolicyPortsBlock(description string) schema.ListNestedBlock {
	doc := networking.NetworkPolicyPort{}.SwaggerDoc()
	return schema.ListNestedBlock{
		Description: description,
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"port": networkPolicyOptionalString(doc["port"]),
				"end_port": schema.Int64Attribute{
					Description: doc["endPort"],
					Optional:    true,
					Computed:    true,
					Default:     int64default.StaticInt64(0),
				},
				"protocol": schema.StringAttribute{
					Description: doc["protocol"],
					Optional:    true,
					Computed:    true,
					Default:     stringdefault.StaticString("TCP"),
				},
			},
		},
	}
}

func networkPolicyPeersBlock(description string) schema.ListNestedBlock {
	doc := networking.NetworkPolicyPeer{}.SwaggerDoc()
	return schema.ListNestedBlock{
		Description: description,
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"ip_block": schema.SingleNestedAttribute{
					Description: doc["ipBlock"],
					Optional:    true,
					Attributes: map[string]schema.Attribute{
						"cidr": networkPolicyOptionalString(networking.IPBlock{}.SwaggerDoc()["cidr"]),
						"except": schema.ListAttribute{
							Description: networking.IPBlock{}.SwaggerDoc()["except"],
							Optional:    true,
							Computed:    true,
							ElementType: types.StringType,
							PlanModifiers: []planmodifier.List{
								networkingEmptyCollectionPlanModifier{},
							},
						},
					},
				},
				"namespace_selector": networkPolicySelectorBlock(doc["namespaceSelector"], false),
				"pod_selector":       networkPolicySelectorBlock(doc["podSelector"], false),
			},
		},
	}
}

func networkPolicySelectorBlock(description string, required bool) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Description: description,
		Required:    required,
		Optional:    !required,
		Attributes: map[string]schema.Attribute{
			"match_labels": schema.MapAttribute{
				Description: "A map of {key,value} pairs. The requirements are ANDed.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					networkingEmptyCollectionPlanModifier{},
				},
			},
			"match_expressions": schema.ListNestedAttribute{
				Description: "A list of label selector requirements. The requirements are ANDed.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key":      networkPolicyOptionalString("The label key that the selector applies to."),
						"operator": networkPolicyOptionalString("A key's relationship to a set of values. Valid operators are `In`, `NotIn`, `Exists` and `DoesNotExist`."),
						"values": schema.SetAttribute{
							Description: "An array of string values. If the operator is `In` or `NotIn`, the values array must be non-empty. If the operator is `Exists` or `DoesNotExist`, the values array must be empty.",
							Optional:    true,
							Computed:    true,
							ElementType: types.StringType,
							PlanModifiers: []planmodifier.Set{
								networkingEmptyCollectionPlanModifier{},
							},
						},
					},
				},
			},
		},
	}
}

func (r *NetworkPolicyV1) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = common.NamespacedIdentitySchema()
}
