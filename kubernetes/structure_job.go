// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/utils/ptr"
)

func flattenJobV1Spec(in batchv1.JobSpec, d *schema.ResourceData, meta interface{}, prefix ...string) ([]interface{}, error) {
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

	if in.CompletionMode != nil {
		att["completion_mode"] = string(*in.CompletionMode)
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

	removeGeneratedLabels(in.Template.ObjectMeta.Labels)

	podSpec, err := flattenPodTemplateSpec(in.Template)
	if err != nil {
		return nil, err
	}
	att["template"] = podSpec

	if in.TTLSecondsAfterFinished != nil {
		att["ttl_seconds_after_finished"] = strconv.Itoa(int(*in.TTLSecondsAfterFinished))
	}

	return []interface{}{att}, nil
}

func expandJobV1Spec(j []interface{}) (batchv1.JobSpec, error) {
	obj := batchv1.JobSpec{}

	if len(j) == 0 || j[0] == nil {
		return obj, nil
	}

	in := j[0].(map[string]interface{})

	if v, ok := in["active_deadline_seconds"].(int); ok && v > 0 {
		obj.ActiveDeadlineSeconds = ptr.To(int64(v))
	}

	if v, ok := in["backoff_limit"].(int); ok && v >= 0 {
		obj.BackoffLimit = ptr.To(int32(v))
	}

	if v, ok := in["completions"].(int); ok && v > 0 {
		obj.Completions = ptr.To(int32(v))
	}

	if v, ok := in["completion_mode"].(string); ok && v != "" {
		m := batchv1.CompletionMode(v)
		obj.CompletionMode = &m
	}

	if v, ok := in["manual_selector"]; ok {
		obj.ManualSelector = ptr.To(v.(bool))
	}

	if v, ok := in["parallelism"].(int); ok && v >= 0 {
		obj.Parallelism = ptr.To(int32(v))
	}

	if v, ok := in["selector"].([]interface{}); ok && len(v) > 0 {
		obj.Selector = expandLabelSelector(v)
	}

	template, err := expandPodTemplate(in["template"].([]interface{}))
	if err != nil {
		return obj, err
	}
	obj.Template = *template

	if v, ok := in["ttl_seconds_after_finished"].(string); ok && v != "" {
		i, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return obj, err
		}
		obj.TTLSecondsAfterFinished = ptr.To(int32(i))
	}

	return obj, nil
}

func patchJobV1Spec(pathPrefix, prefix string, d *schema.ResourceData) PatchOperations {
	ops := make([]PatchOperation, 0)

	if d.HasChange(prefix + "active_deadline_seconds") {
		v := d.Get(prefix + "active_deadline_seconds").(int)
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/activeDeadlineSeconds",
			Value: v,
		})
	}

	if d.HasChange(prefix + "backoff_limit") {
		v := d.Get(prefix + "backoff_limit").(int)
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/backoffLimit",
			Value: v,
		})
	}

	if d.HasChange(prefix + "manual_selector") {
		v := d.Get(prefix + "manual_selector").(bool)
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/manualSelector",
			Value: v,
		})
	}

	if d.HasChange(prefix + "parallelism") {
		v := d.Get(prefix + "parallelism").(int)
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/parallelism",
			Value: v,
		})
	}

	return ops
}

// removeGeneratedLabels removes server-generated labels
func removeGeneratedLabels(labels map[string]string) map[string]string {
	// The Jobs controller adds the following labels to the template block dynamically
	// and thus we have to ignore them to avoid perpetual diff:
	// - 'batch.kubernetes.io/controller-uid'
	// - 'batch.kubernetes.io/job-name'
	// - 'controller-uid' // deprecated starting from Kubernetes 1.27
	// - 'job-name'  // deprecated starting from Kubernetes 1.27
	//
	// More information: https://kubernetes.io/docs/reference/labels-annotations-taints/
	generatedLabels := []string{
		"batch.kubernetes.io/controller-uid",
		"batch.kubernetes.io/job-name",
		// Starting from Kubernetes 1.27, the following labels are deprecated.
		"controller-uid",
		"job-name",
	}
	for _, l := range generatedLabels {
		delete(labels, l)
	}

	return labels
}
