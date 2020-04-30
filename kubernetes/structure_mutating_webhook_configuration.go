package kubernetes

import (
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
)

func flattenMutatingWebhook(in admissionregistrationv1.MutatingWebhook) map[string]interface{} {
	att := map[string]interface{}{}

	att["admission_review_versions"] = in.AdmissionReviewVersions

	att["client_config"] = flattenWebhookClientConfig(in.ClientConfig)

	if in.FailurePolicy != nil {
		att["failure_policy"] = *in.FailurePolicy
	}

	if in.MatchPolicy != nil {
		att["match_policy"] = *in.MatchPolicy
	}

	att["name"] = in.Name

	if in.NamespaceSelector != nil {
		if in.NamespaceSelector.MatchExpressions != nil || in.NamespaceSelector.MatchLabels != nil {
			att["namespace_selector"] = flattenLabelSelector(in.NamespaceSelector)
		}
	}

	if in.ObjectSelector != nil {
		if in.ObjectSelector.MatchExpressions != nil || in.ObjectSelector.MatchLabels != nil {
			att["object_selector"] = flattenLabelSelector(in.ObjectSelector)
		}
	}

	if in.ReinvocationPolicy != nil {
		att["reinvocation_policy"] = *in.ReinvocationPolicy
	}

	rules := []interface{}{}
	for _, rule := range in.Rules {
		rules = append(rules, flattenRuleWithOperations(rule))
	}
	att["rule"] = rules

	if in.SideEffects != nil {
		att["side_effects"] = *in.SideEffects
	}

	if in.TimeoutSeconds != nil {
		att["timeout_seconds"] = *in.TimeoutSeconds
	}

	return att
}

func expandMutatingWebhook(in map[string]interface{}) admissionregistrationv1.MutatingWebhook {
	obj := admissionregistrationv1.MutatingWebhook{}

	if v, ok := in["admission_review_versions"].([]interface{}); ok {
		obj.AdmissionReviewVersions = expandStringSlice(v)
	}

	if v, ok := in["client_config"].([]interface{}); ok {
		obj.ClientConfig = expandWebhookClientConfig(v)
	}

	if v, ok := in["failure_policy"].(string); ok {
		policy := admissionregistrationv1.FailurePolicyType(v)
		obj.FailurePolicy = &policy
	}

	if v, ok := in["match_policy"].(string); ok {
		policy := admissionregistrationv1.MatchPolicyType(v)
		obj.MatchPolicy = &policy
	}

	if v, ok := in["name"].(string); ok {
		obj.Name = v
	}

	if v, ok := in["namespace_selector"].([]interface{}); ok && len(v) != 0 {
		obj.NamespaceSelector = expandLabelSelector(v)
	}

	if v, ok := in["object_selector"].([]interface{}); ok && len(v) != 0 {
		obj.ObjectSelector = expandLabelSelector(v)
	}

	if v, ok := in["reinvocation_policy"].(string); ok {
		policy := admissionregistrationv1.ReinvocationPolicyType(v)
		obj.ReinvocationPolicy = &policy
	}

	if v, ok := in["rule"].([]interface{}); ok {
		rules := []admissionregistrationv1.RuleWithOperations{}
		for _, r := range v {
			rules = append(rules, expandRuleWithOperations(r.(map[string]interface{})))
		}
		obj.Rules = rules
	}

	if v, ok := in["side_effects"].(string); ok {
		sideEffects := admissionregistrationv1.SideEffectClass(v)
		obj.SideEffects = &sideEffects
	}

	if v, ok := in["timeout_seconds"].(int); ok {
		obj.TimeoutSeconds = ptrToInt32(int32(v))
	}

	return obj
}

func expandMutatingWebhooks(in []interface{}) []admissionregistrationv1.MutatingWebhook {
	webhooks := []admissionregistrationv1.MutatingWebhook{}
	for _, h := range in {
		webhooks = append(webhooks, expandMutatingWebhook(h.(map[string]interface{})))
	}
	return webhooks
}

func flattenMutatingWebhooks(in []admissionregistrationv1.MutatingWebhook) []interface{} {
	webhooks := []interface{}{}
	for _, h := range in {
		webhooks = append(webhooks, flattenMutatingWebhook(h))
	}
	return webhooks
}
