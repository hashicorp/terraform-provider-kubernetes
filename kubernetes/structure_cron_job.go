package kubernetes

import (
	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/api/batch/v1beta1"
	"errors"
)

func flattenCronJobSpec(in v1beta1.CronJobSpec) ([]interface{}, error) {
	att := make(map[string]interface{})

	if in.Schedule != "" {
		att["schedule"] = in.Schedule
	} else {
		return nil, errors.New("You need to define a schedule.")
	}

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

	if v, ok := in["schedule"].(string); ok && len(v) > 0 {
		obj.Schedule = *ptrToString(string(v))
	} else {
		return obj, errors.New("You need to define a schedule.")
	}

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
