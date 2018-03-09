package kubernetes

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func cronJobSpecFields() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"schedule": {
			Type:         schema.TypeString,
			Optional:     true,
			//ValidateFunc: validate, TODO: validate cron syntax..
			Description:  "Cron format string, e.g. 0 * * * * or @hourly, as schedule time of its jobs to be created and executed.",
		},
		"job_template": {
			Type:        schema.TypeList,
			Description: "Describes the pod that will be created when executing a cron job. More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/",
			Required:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: jobSpecFields(),
			},
		},
	}

	return s
}
