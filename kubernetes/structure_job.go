// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
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

	if in.PodFailurePolicy != nil {
		att["pod_failure_policy"] = flattenPodFailurePolicy(in.PodFailurePolicy)
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

	if v, ok := in["pod_failure_policy"].([]interface{}); ok && len(v) > 0 {
		obj.PodFailurePolicy = expandPodFailurePolicy(v)
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

func expandPodFailurePolicy(l []interface{}) *batchv1.PodFailurePolicy {
	obj := &batchv1.PodFailurePolicy{}
	if len(l) == 0 || l[0] == nil {
		return obj
	}
	in := l[0].(map[string]interface{})

	if v, ok := in["rule"].([]interface{}); ok && len(v) > 0 {
		rules := expandPodFailurePolicyRules(v)
		obj.Rules = rules
	}
	return obj
}

func expandPodFailurePolicyRules(l []interface{}) []batchv1.PodFailurePolicyRule {
	obj := make([]batchv1.PodFailurePolicyRule, len(l))
	if len(l) == 0 || l[0] == nil {
		return obj
	}
	for i, rule := range l {
		objRule := &batchv1.PodFailurePolicyRule{}

		r := rule.(map[string]interface{})

		if v, ok := r["action"].(string); ok && v != "" {
			objRule.Action = batchv1.PodFailurePolicyAction(v)
		}

		if v, ok := r["on_exit_codes"].([]interface{}); ok && len(v) > 0 {
			onExitCodes := expandPodFailurePolicyOnExitCodesRequirement(v)
			objRule.OnExitCodes = onExitCodes
		}

		if v, ok := r["on_pod_condition"].([]interface{}); ok && len(v) > 0 {
			podConditions := expandPodFailurePolicyOnPodConditionsPattern(v)
			objRule.OnPodConditions = podConditions
		}

		obj[i] = *objRule
	}

	return obj
}

func expandPodFailurePolicyOnExitCodesRequirement(l []interface{}) *batchv1.PodFailurePolicyOnExitCodesRequirement {
	obj := &batchv1.PodFailurePolicyOnExitCodesRequirement{}
	if len(l) == 0 || l[0] == nil {
		return obj
	}
	in := l[0].(map[string]interface{})

	if v, ok := in["container_name"].(string); ok && v != "" {
		obj.ContainerName = &v
	}

	if v, ok := in["operator"].(string); ok && v != "" {
		obj.Operator = batchv1.PodFailurePolicyOnExitCodesOperator(v)
	}

	if v, ok := in["values"].([]interface{}); ok && len(v) > 0 {
		vals := make([]int32, len(v))
		for i := 0; i < len(v); i++ {
			vals[i] = int32(v[i].(int))
		}

		obj.Values = vals
	}

	return obj
}

func expandPodFailurePolicyOnPodConditionsPattern(l []interface{}) []batchv1.PodFailurePolicyOnPodConditionsPattern {
	obj := make([]batchv1.PodFailurePolicyOnPodConditionsPattern, len(l))
	if len(l) == 0 || l[0] == nil {
		return obj
	}
	for i, condition := range l {
		objCondition := &batchv1.PodFailurePolicyOnPodConditionsPattern{}
		c := condition.(map[string]interface{})
		if v, ok := c["status"].(string); ok && v != "" {
			objCondition.Status = v1.ConditionStatus(v)
		}

		if v, ok := c["type"].(string); ok && v != "" {
			objCondition.Type = v1.PodConditionType(v)
		}
		obj[i] = *objCondition
	}
	return obj
}

func flattenPodFailurePolicy(in *batchv1.PodFailurePolicy) []interface{} {
	att := make(map[string]interface{})
	if len(in.Rules) > 0 {
		att["rule"] = flattenPodFailurePolicyRules(in.Rules)
	}
	return []interface{}{att}
}

func flattenPodFailurePolicyRules(in []batchv1.PodFailurePolicyRule) []interface{} {
	att := make([]interface{}, len(in))

	for i, r := range in {
		m := make(map[string]interface{})
		m["action"] = r.Action
		if r.OnExitCodes != nil {
			m["on_exit_codes"] = flattenPodFailurePolicyOnExitCodes(r.OnExitCodes)
		}
		if r.OnPodConditions != nil {
			m["on_pod_condition"] = flattenPodFailurePolicyOnPodConditions(r.OnPodConditions)
		}
		att[i] = m
	}

	return att
}

func flattenPodFailurePolicyOnExitCodes(in *batchv1.PodFailurePolicyOnExitCodesRequirement) []interface{} {
	att := make(map[string]interface{})
	if *in.ContainerName != "" {
		att["container_name"] = *in.ContainerName
	}
	att["operator"] = in.Operator
	if len(in.Values) > 0 {
		vals := make([]int, len(in.Values))
		for i := 0; i < len(vals); i++ {
			vals[i] = int(in.Values[i])
		}
		att["values"] = vals
	}

	return []interface{}{att}
}

func flattenPodFailurePolicyOnPodConditions(in []batchv1.PodFailurePolicyOnPodConditionsPattern) []interface{} {
	att := make([]interface{}, len(in))

	for i, r := range in {
		m := make(map[string]interface{})
		if r.Status != "" {
			m["status"] = r.Status
		}
		if r.Type != "" {
			m["type"] = r.Type
		}
		att[i] = m
	}

	return att
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

	if d.HasChange(prefix + "pod_failure_policy") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/podFailurePolicy",
			Value: expandPodFailurePolicy(d.Get(prefix + "pod_failure_policy").([]interface{})),
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
