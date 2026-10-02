// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

// The schema is built and frozen once per process; see common.FrozenSchema.
var jobFrozenSchema = common.FrozenSchema(buildJobSchema)

func (r *JobV1) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	jobFrozenSchema(ctx, req, resp)
}

func buildJobSchema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	metadata := common.WithEmptyMetadataCompatibility(common.NamespacedMetadataSchema("job", true))
	// Kubernetes copies the pod template's labels to a Job created without any.
	labels := metadata.NestedObject.Attributes["labels"].(schema.MapAttribute)
	labels.Default, labels.PlanModifiers = nil, nil
	metadata.NestedObject.Attributes["labels"] = labels
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "A Job creates one or more Pods and ensures that a specified number successfully terminate.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"wait_for_completion": schema.BoolAttribute{
				Optional: true, Computed: true, Default: booldefault.StaticBool(true),
				Description: "Wait until the job completes successfully. A failed job returns an error.",
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": metadata,
			"spec":     jobSpecBlock(true),
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Update: true, Delete: true}),
		},
	}
}

func (r *JobV1) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = common.NamespacedIdentitySchema()
}

func (r *JobV1) UpgradeIdentity(context.Context) map[int64]resource.IdentityUpgrader {
	return common.UpgradeNamespacedIdentity("Job", "batch/v1")
}
