// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package nodev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (r *RuntimeClassV1) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: 1, // SDKv2 was version 0 (default); Framework is version 1
		MarkdownDescription: "A RuntimeClass is used to determine which container runtime is used to run " +
			"all containers in a pod. RuntimeClass objects in the `node.k8s.io` API group select a " +
			"specific handler (e.g. `runc`, `kata`, `gvisor`). " +
			"More info: https://kubernetes.io/docs/concepts/containers/runtime-class/",

		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier for this resource (the RuntimeClass name).",
				Computed:            true,
			},

			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Standard object metadata. " +
					"More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata",
				Required: true,
				Attributes: map[string]schema.Attribute{
					"annotations": schema.MapAttribute{
						MarkdownDescription: "An unstructured key value map stored with the RuntimeClass. " +
							"Keys under *.kubernetes.io/ are managed by the cluster and filtered from state. " +
							"More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/",
						ElementType: types.StringType,
						Optional:    true,
					},

					"generate_name": schema.StringAttribute{
						MarkdownDescription: "Prefix used by the server to generate a unique name when `name` is not provided. " +
							"More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#idempotency",
						Optional: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},

					"generation": schema.Int64Attribute{
						MarkdownDescription: "A sequence number representing a specific generation of the desired state. Read-only.",
						Computed:            true,
					},

					"labels": schema.MapAttribute{
						MarkdownDescription: "Map of string keys and values that can be used to organize and categorize the RuntimeClass. " +
							"Keys under *.kubernetes.io/ are filtered from state. " +
							"More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/",
						ElementType: types.StringType,
						Optional:    true,
					},

					"name": schema.StringAttribute{
						MarkdownDescription: "Name of the RuntimeClass, must be unique. Cannot be updated. " +
							"More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
						Optional: true,
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},

					"resource_version": schema.StringAttribute{
						MarkdownDescription: "An opaque value representing the internal version of this object. Read-only.",
						Computed:            true,
					},

					"uid": schema.StringAttribute{
						MarkdownDescription: "The unique in time and space value for this RuntimeClass. Read-only. " +
							"More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids",
						Computed: true,
					},
				},
			},

			"handler": schema.StringAttribute{
				MarkdownDescription: "Specifies the underlying runtime and configuration that the CRI " +
					"implementation will use to handle pods of this class. " +
					"Must match a handler registered on every node that is expected to run pods of this RuntimeClass. " +
					"Must be a DNS label (lowercase alphanumeric + hyphens, cannot start/end with hyphen).",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						dnsLabelRegexp,
						"must be a valid DNS label: lowercase alphanumeric characters or '-', "+
							"must start and end with an alphanumeric character",
					),
				},
			},
		},
	}
}
