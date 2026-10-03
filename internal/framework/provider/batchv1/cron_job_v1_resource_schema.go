// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/robfig/cron"
)

// The schema is built and frozen once per process; see common.FrozenSchema.
var cronJobFrozenSchema = common.FrozenSchema(buildCronJobSchema)

func (r *CronJobV1) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	cronJobFrozenSchema(ctx, req, resp)
}

func buildCronJobSchema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "A CronJob creates Jobs on a time-based schedule.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": common.WithEmptyMetadataCompatibility(common.NamespacedMetadataSchema("cronjob", true)),
			"spec":     cronJobSpecBlock(),
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Delete: true}),
		},
	}
}

func (r *CronJobV1) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = common.NamespacedIdentitySchema()
}

func cronJobSpecBlock() schema.ListNestedBlock {
	templateMetadata := templateMetadataBlock(common.NamespacedMetadataSchema("jobTemplateSpec", true))
	// A template has no namespace default in Kubernetes; omission plans SDKv2's empty value.
	templateMetadata.NestedObject.Attributes["namespace"] = schema.StringAttribute{
		Optional:      true,
		Computed:      true,
		Default:       stringdefault.StaticString(""),
		PlanModifiers: []planmodifier.String{zeroEquivalentStringRequiresReplace{}},
	}
	return schema.ListNestedBlock{
		Description: "Specification of the cron job. Exactly one spec block is required.",
		Validators:  []validator.List{listvalidator.IsRequired(), listvalidator.SizeBetween(1, 1)},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"concurrency_policy": schema.StringAttribute{
					Description: "Specifies how to treat concurrent executions of a Job. Defaults to Allow.",
					Optional:    true,
					Computed:    true,
					Default:     stringdefault.StaticString("Allow"),
					Validators:  []validator.String{stringvalidator.OneOf("Allow", "Forbid", "Replace")},
				},
				"failed_jobs_history_limit": schema.Int64Attribute{
					Description: "The number of failed finished jobs to retain. Defaults to 1.",
					Optional:    true,
					Computed:    true,
					Default:     int64default.StaticInt64(1),
				},
				"schedule": schema.StringAttribute{
					Description: "Cron format string, for example 0 * * * * or @hourly.",
					Required:    true,
					Validators:  []validator.String{cronScheduleValidator{}},
				},
				"starting_deadline_seconds": schema.Int64Attribute{
					Description: "Deadline in seconds for starting a job if it misses its scheduled time.",
					Optional:    true,
					Computed:    true,
					Default:     int64default.StaticInt64(0),
				},
				"successful_jobs_history_limit": schema.Int64Attribute{
					Description: "The number of successful finished jobs to retain. Defaults to 3.",
					Optional:    true,
					Computed:    true,
					Default:     int64default.StaticInt64(3),
				},
				"suspend": schema.BoolAttribute{
					Description: "Suspend subsequent executions without affecting already started executions.",
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(false),
				},
				"timezone": schema.StringAttribute{
					Description: "The time zone for the schedule. Empty uses the kube-controller-manager time zone.",
					Optional:    true,
					Computed:    true,
					Default:     stringdefault.StaticString(""),
				},
			},
			Blocks: map[string]schema.Block{
				"job_template": schema.ListNestedBlock{
					Description: "Template for Jobs created by this CronJob.",
					Validators:  []validator.List{listvalidator.IsRequired(), listvalidator.SizeBetween(1, 1)},
					NestedObject: schema.NestedBlockObject{
						Blocks: map[string]schema.Block{
							"metadata": templateMetadata,
							"spec":     jobSpecBlock(false),
						},
					},
				},
			},
		},
	}
}

// cronJobSpecBlockType is the CronJob spec type, derived once per process.
var cronJobSpecBlockType = sync.OnceValue(func() types.ObjectType {
	return common.FreezeListNestedBlock(cronJobSpecBlock()).NestedObject.Type().(types.ObjectType)
})

type cronScheduleValidator struct{}

func (cronScheduleValidator) Description(context.Context) string {
	return "The value must be a valid Cron expression."
}

func (v cronScheduleValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (cronScheduleValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if _, err := cron.ParseStandard(req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Cron expression", err.Error())
	}
}
