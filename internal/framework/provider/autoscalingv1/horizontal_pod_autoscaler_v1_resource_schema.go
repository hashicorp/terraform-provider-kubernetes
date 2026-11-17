// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package autoscalingv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

// singleRequiredBlock reproduces SDKv2's Required + MaxItems: 1 for a list block.
func singleRequiredBlock() []validator.List {
	return []validator.List{
		listvalidator.IsRequired(),
		listvalidator.SizeAtLeast(1),
		listvalidator.SizeAtMost(1),
	}
}

func (r *HorizontalPodAutoscalerV1) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Horizontal Pod Autoscaler automatically scales the number of pods in a replication controller, deployment or replica set based on observed CPU utilization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of this resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": common.NamespacedMetadataSchema("horizontal pod autoscaler", true),
			"spec": schema.ListNestedBlock{
				Description: "Behaviour of the autoscaler. Exactly one spec block is required. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status",
				Validators:  singleRequiredBlock(),
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"max_replicas": schema.Int64Attribute{
							Description: "Upper limit for the number of pods that can be set by the autoscaler.",
							Required:    true,
						},
						"min_replicas": schema.Int64Attribute{
							Description: "Lower limit for the number of pods that can be set by the autoscaler, defaults to `1`.",
							Optional:    true,
							Computed:    true,
							Default:     int64default.StaticInt64(1),
						},
						"target_cpu_utilization_percentage": schema.Int64Attribute{
							Description: "Target average CPU utilization (represented as a percentage of requested CPU) over all the pods. If not specified the default autoscaling policy will be used.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
					},
					Blocks: map[string]schema.Block{
						"scale_target_ref": schema.ListNestedBlock{
							Description: "Reference to scaled resource. e.g. Replication Controller. Exactly one scale_target_ref block is required.",
							Validators:  singleRequiredBlock(),
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"api_version": schema.StringAttribute{
										Description: "API version of the referent",
										Optional:    true,
									},
									"kind": schema.StringAttribute{
										Description: "Kind of the referent. e.g. `ReplicationController`. More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#types-kinds",
										Required:    true,
									},
									"name": schema.StringAttribute{
										Description: "Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
										Required:    true,
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
