package kubernetes

import (
	"k8s.io/api/batch/v1beta1"
)

func flattenCronJobSpec(in v1beta1.CronJobSpec) ([]interface{}, error) {
	att := make(map[string]interface{})

	att["concurrency_policy"] = in.ConcurrencyPolicy
	if in.FailedJobsHistoryLimit != nil {
		att["failed_jobs_history_limit"] = int(*in.FailedJobsHistoryLimit)
	} else {
		att["failed_jobs_history_limit"] = 1
	}

	att["schedule"] = in.Schedule

	jobTemplate, err := flattenJobTemplate(in.JobTemplate)
	if err != nil {
		return nil, err
	}
	att["job_template"] = jobTemplate

	if in.StartingDeadlineSeconds != nil {
		att["starting_deadline_seconds"] = int64(*in.StartingDeadlineSeconds)
	} else {
		att["starting_deadline_seconds"] = 0
	}

	if in.SuccessfulJobsHistoryLimit != nil {
		att["successful_jobs_history_limit"] = int32(*in.SuccessfulJobsHistoryLimit)
	} else {
		att["successful_jobs_history_limit"] = 3
	}

	return []interface{}{att}, nil
}

func flattenJobTemplate(in v1beta1.JobTemplateSpec) ([]interface{}, error) {
	att := make(map[string]interface{})

	meta := flattenMetadata(in.ObjectMeta)
	att["metadata"] = meta

	jobSpec, err := flattenJobSpec(in.Spec)
	if err != nil {
		return nil, err
	}
	att["spec"] = jobSpec

	return []interface{}{att}, nil
}

func expandCronJobSpec(j []interface{}) (v1beta1.CronJobSpec, error) {
	obj := v1beta1.CronJobSpec{}

	if len(j) == 0 || j[0] == nil {
		return obj, nil
	}

	in := j[0].(map[string]interface{})

	obj.ConcurrencyPolicy = v1beta1.ConcurrencyPolicy(in["concurrency_policy"].(string))

	if v, ok := in["failed_jobs_history_limit"].(int); ok && v != 1 {
		obj.FailedJobsHistoryLimit = ptrToInt32(int32(v))
	}

	obj.Schedule = in["schedule"].(string)

	jtSpec, err := expandJobTemplate(in["job_template"].([]interface{}))
	if err != nil {
		return obj, err
	}
	obj.JobTemplate = jtSpec

	if v, ok := in["starting_deadline_seconds"].(int); ok && v > 0 {
		obj.StartingDeadlineSeconds = ptrToInt64(int64(v))
	}

	if v, ok := in["successful_jobs_history_limit"].(int); ok && v != 3 {
		obj.SuccessfulJobsHistoryLimit = ptrToInt32(int32(v))
	}

	if v, ok := in["suspend"].(bool); ok {
		obj.Suspend = ptrToBool(v)
	}

	return obj, nil
}

func expandJobTemplate(in []interface{}) (v1beta1.JobTemplateSpec, error) {
	obj := v1beta1.JobTemplateSpec{}

	tpl := in[0].(map[string]interface{})

	spec, err := expandJobSpec(tpl["spec"].([]interface{}))
	if err != nil {
		return obj, err
	}
	obj.Spec = spec

	if metaCfg, ok := tpl["metadata"]; ok {
		metadata := expandMetadata(metaCfg.([]interface{}))
		obj.ObjectMeta = metadata
	}

	return obj, nil
}
