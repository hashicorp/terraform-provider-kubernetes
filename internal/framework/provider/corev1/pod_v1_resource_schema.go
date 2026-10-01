// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

func (p *PodV1) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	spec := podBlock(podSpecWithDescriptions(podSpecObject()), 1, 1, false)
	spec.Description = "Specification of the desired behavior of the pod. Exactly one spec block is required."
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "A pod is a group of one or more containers, the shared storage for those containers, and options about how to run the containers. Pods are always co-located and co-scheduled, and run in a shared context. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod/.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"target_state": schema.ListAttribute{
				Description: "A list of the pod phases that indicate whether it was successfully created. Options: Pending, Running, Succeeded, Failed, Unknown. Defaults to Running.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.ValueStringsAre(stringvalidator.OneOf("Pending", "Running", "Succeeded", "Failed", "Unknown")),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": common.NamespacedMetadataSchema("pod", true),
			"spec":     spec,
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Delete: true}),
		},
	}
}
