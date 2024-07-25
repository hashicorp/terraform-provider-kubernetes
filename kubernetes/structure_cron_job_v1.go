// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	batch "k8s.io/api/batch/v1"
	"k8s.io/utils/ptr"
)

func flattenCronJobSpecV1(in batch.CronJobSpec, d *schema.ResourceData, meta interface{}) ([]interface{}, error) {
	att := make(map[string]interface{})

	att["concurrency_policy"] = in.ConcurrencyPolicy

	if in.FailedJobsHistoryLimit != nil {
		att["failed_jobs_history_limit"] = int(*in.FailedJobsHistoryLimit)
	}

	att["schedule"] = in.Schedule

	att["timezone"] = in.TimeZone

	jobTemplate, err := flattenJobTemplateV1(in.JobTemplate, d, meta)
	if err != nil {
		return nil, err
	}
	att["job_template"] = jobTemplate

	if in.StartingDeadlineSeconds != nil {
		att["starting_deadline_seconds"] = int(*in.StartingDeadlineSeconds)
	}

	if in.SuccessfulJobsHistoryLimit != nil {
		att["successful_jobs_history_limit"] = int(*in.SuccessfulJobsHistoryLimit)
	}

	att["suspend"] = in.Suspend

	return []interface{}{att}, nil
}

func flattenJobTemplateV1(in batch.JobTemplateSpec, d *schema.ResourceData, meta interface{}) ([]interface{}, error) {
	att := make(map[string]interface{})

	att["metadata"] = flattenMetadataFields(in.ObjectMeta)

	jobSpec, err := flattenJobV1Spec(in.Spec, d, meta, "spec.0.job_template.0.spec.0.template.0.")
	if err != nil {
		return nil, err
	}
	att["spec"] = jobSpec

	return []interface{}{att}, nil
}

func expandCronJobSpecV1(j []interface{}) (batch.CronJobSpec, error) {
	obj := batch.CronJobSpec{}

	if len(j) == 0 || j[0] == nil {
		return obj, nil
	}

	in := j[0].(map[string]interface{})

	if v, ok := in["concurrency_policy"].(string); ok && v != "" {
		obj.ConcurrencyPolicy = batch.ConcurrencyPolicy(v)
	}

	if v, ok := in["failed_jobs_history_limit"].(int); ok && v != 1 {
		obj.FailedJobsHistoryLimit = ptr.To(int32(v))
	}

	if v, ok := in["schedule"].(string); ok && v != "" {
		obj.Schedule = v
	}

	if v, ok := in["timezone"].(string); ok && v != "" {
		obj.TimeZone = &v
	}

	jtSpec, err := expandJobTemplateV1(in["job_template"].([]interface{}))
	if err != nil {
		return obj, err
	}
	obj.JobTemplate = jtSpec

	if v, ok := in["starting_deadline_seconds"].(int); ok && v > 0 {
		obj.StartingDeadlineSeconds = ptr.To(int64(v))
	}

	if v, ok := in["successful_jobs_history_limit"].(int); ok && v != 3 {
		obj.SuccessfulJobsHistoryLimit = ptr.To(int32(v))
	}

	if v, ok := in["suspend"].(bool); ok {
		obj.Suspend = ptr.To(v)
	}

	return obj, nil
}

func expandJobTemplateV1(in []interface{}) (batch.JobTemplateSpec, error) {
	obj := batch.JobTemplateSpec{}

	if len(in) == 0 || in[0] == nil {
		return obj, nil
	}

	tpl := in[0].(map[string]interface{})

	spec, err := expandJobV1Spec(tpl["spec"].([]interface{}))
	if err != nil {
		return obj, err
	}
	obj.Spec = spec

	if metaCfg, ok := tpl["metadata"].([]interface{}); ok {
		metadata := expandMetadata(metaCfg)
		obj.ObjectMeta = metadata
	}

	return obj, nil
}
