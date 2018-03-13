package kubernetes

import (
	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/api/batch/v1beta1"
	"errors"
)

func flattenCronJobSpec(in v1beta1.CronJobSpec) ([]interface{}, error) {
	att := make(map[string]interface{})

	att["schedule"] = in.Schedule

	jobSpec, err := flattenJobSpec(in.JobTemplate.Spec)
	if err != nil {
		return nil, err
	}
	att["job_template"] = jobSpec

	return []interface{}{att}, nil
}

func expandCronJobSpec(j []interface{}) (v1beta1.CronJobSpec, error) {
	obj := v1beta1.CronJobSpec{}

	if len(j) == 0 || j[0] == nil {
		return obj, nil
	}

	in := j[0].(map[string]interface{})

	obj.Schedule = in["schedule"].(string)

	podSpec, err := expandJobSpec(in["job_template"].([]interface{}))
	if err != nil {
		return obj, err
	}


	obj.JobTemplate = v1beta1.JobTemplateSpec {
		Spec: podSpec,
	}

	return obj, nil
}

func patchCronJobSpec(pathPrefix, prefix string, d *schema.ResourceData) (PatchOperations, error) {
	ops := make([]PatchOperation, 0)

	return ops, nil
}
