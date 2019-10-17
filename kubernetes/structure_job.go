package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	batchv1 "k8s.io/api/batch/v1"
)

func flattenJobSpec(in batchv1.JobSpec, d *schema.ResourceData, prefix ...string) ([]interface{}, error) {
	att := make(map[string]interface{})

	if in.ActiveDeadlineSeconds != nil {
		att["active_deadline_seconds"] = *in.ActiveDeadlineSeconds
	}

	if in.BackoffLimit != nil {
		att["backoff_limit"] = *in.BackoffLimit
	}

	if in.Completions != nil {
		att["completions"] = *in.Completions
	}

	if in.ManualSelector != nil {
		att["manual_selector"] = *in.ManualSelector
	}

	if in.Parallelism != nil {
		att["parallelism"] = *in.Parallelism
	}

	if in.Selector != nil {
		att["selector"] = flattenLabelSelector(in.Selector)
	}
	// Remove server-generated labels
	labels := in.Template.ObjectMeta.Labels

	if _, ok := labels["controller-uid"]; ok {
		delete(labels, "controller-uid")
	}

	if _, ok := labels["job-name"]; ok {
		delete(labels, "job-name")
	}

	podSpec, err := flattenPodTemplateSpec(in.Template, d, prefix...)
	if err != nil {
		return nil, err
	}
	att["template"] = podSpec

	return []interface{}{att}, nil
}

func expandJobSpec(j []interface{}) (batchv1.JobSpec, error) {
	obj := batchv1.JobSpec{}

	if len(j) == 0 || j[0] == nil {
		return obj, nil
	}

	in := j[0].(map[string]interface{})

	if v, ok := in["active_deadline_seconds"].(int); ok && v > 0 {
		obj.ActiveDeadlineSeconds = ptrToInt64(int64(v))
	}

	if v, ok := in["backoff_limit"].(int); ok && v != 6 {
		obj.BackoffLimit = ptrToInt32(int32(v))
	}

	if v, ok := in["completions"].(int); ok && v > 0 {
		obj.Completions = ptrToInt32(int32(v))
	}

	if v, ok := in["manual_selector"]; ok {
		obj.ManualSelector = ptrToBool(v.(bool))
	}

	if v, ok := in["parallelism"].(int); ok && v > 0 {
		obj.Parallelism = ptrToInt32(int32(v))
	}

	if v, ok := in["selector"].([]interface{}); ok && len(v) > 0 {
		obj.Selector = expandLabelSelector(v)
	}

	template, err := expandPodTemplate(in["template"].([]interface{}))
	if err != nil {
		return obj, err
	}
	obj.Template = *template

	return obj, nil
}

func patchJobSpec(pathPrefix, prefix string, d *schema.ResourceData) (PatchOperations, error) {
	ops := make([]PatchOperation, 0)

	if d.HasChange(prefix + "active_deadline_seconds") {
		v := d.Get(prefix + "active_deadline_seconds").(int)
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/activeDeadlineSeconds",
			Value: v,
		})
	}

	return ops, nil
}
