// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func cronJobSpecFieldsV1() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"concurrency_policy": {
			Type:         schema.TypeString,
			Optional:     true,
			Default:      "Allow",
			ValidateFunc: validation.StringInSlice([]string{"Allow", "Forbid", "Replace"}, false),
			Description:  "Specifies how to treat concurrent executions of a Job. Defaults to Allow.",
		},
		"failed_jobs_history_limit": {
			Type:        schema.TypeInt,
			Optional:    true,
			Default:     1,
			Description: "The number of failed finished jobs to retain. This is a pointer to distinguish between explicit zero and not specified. Defaults to 1.",
		},
		"job_template": {
			Type:        schema.TypeList,
			Description: "Describes the pod that will be created when executing a cron job. More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/",
			Required:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"metadata": namespacedMetadataSchemaIsTemplate("jobTemplateSpec", true, true),
					"spec": {
						Type:        schema.TypeList,
						Description: "Specification of the desired behavior of the job",
						Required:    true,
						MaxItems:    1,
						Elem: &schema.Resource{
							Schema: jobSpecFields(true),
						},
					},
				},
			},
		},
		"schedule": {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validateCronExpression,
			Description:  "Cron format string, e.g. 0 * * * * or @hourly, as schedule time of its jobs to be created and executed.",
		},
		"starting_deadline_seconds": {
			Type:        schema.TypeInt,
			Optional:    true,
			Default:     0,
			Description: "Optional deadline in seconds for starting the job if it misses scheduled time for any reason. Missed jobs executions will be counted as failed ones.",
		},
		"successful_jobs_history_limit": {
			Type:        schema.TypeInt,
			Optional:    true,
			Default:     3,
			Description: "The number of successful finished jobs to retain. Defaults to 3.",
		},
		"suspend": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "This flag tells the controller to suspend subsequent executions, it does not apply to already started executions. Defaults to false.",
		},
		"timezone": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The time zone for the given schedule. If not specified, this will rely on the time zone of the kube-controller-manager process. ",
		},
	}
}
