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
		Version: 2,
		Description: "A Job creates one or more Pods and ensures that a specified number of them successfully terminate. " +
			"As pods successfully complete, the Job tracks the successful completions. " +
			"When a specified number of successful completions is reached, the task (i.e. the Job) is complete. " +
			"Deleting a Job will clean up the Pods it created. " +
			"A simple case is to create one Job object in order to reliably run one Pod to completion. " +
			"The Job object will start a new Pod if the first Pod fails or is deleted (for example due to a node hardware failure or a node reboot). " +
			"You can also use a Job to run multiple Pods in parallel.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"wait_for_completion": schema.BoolAttribute{
				Optional: true, Computed: true, Default: booldefault.StaticBool(true),
				Description: "Wait for the Job to complete successfully during create and update operations. Defaults to true. A failed Job returns an error.",
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
