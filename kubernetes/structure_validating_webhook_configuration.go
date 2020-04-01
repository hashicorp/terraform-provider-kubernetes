package kubernetes

import (
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
)

func flattenServiceReference(in admissionregistrationv1.ServiceReference) []interface{} {
	att := map[string]interface{}{}

	att["name"] = in.Name
	att["namespace"] = in.Namespace

	if in.Path != nil {
		att["path"] = in.Path
	}

	if in.Port != nil {
		att["port"] = *in.Port
	}

	return []interface{}{att}
}

func expandServiceReference(l []interface{}) *admissionregistrationv1.ServiceReference {
	obj := &admissionregistrationv1.ServiceReference{}

	if len(l) == 0 || l[0] == nil {
		return obj
	}

	in := l[0].(map[string]interface{})

	if v, ok := in["name"].(string); ok {
		obj.Name = v
	}

	if v, ok := in["namespace"].(string); ok {
		obj.Namespace = v
	}

	if v, ok := in["path"].(string); ok && v != "" {
		obj.Path = ptrToString(v)
	}

	if v, ok := in["port"].(int); ok {
		obj.Port = ptrToInt32(int32(v))
	}

	return obj
}

func flattenWebhookClientConfig(in admissionregistrationv1.WebhookClientConfig) []interface{} {
	att := map[string]interface{}{}

	if len(in.CABundle) > 0 {
		att["ca_bundle"] = string(in.CABundle)
	}

	if in.Service != nil {
		att["service"] = flattenServiceReference(*in.Service)
	}

	if in.URL != nil {
		att["url"] = *in.URL
	}

	return []interface{}{att}
}

func expandWebhookClientConfig(l []interface{}) admissionregistrationv1.WebhookClientConfig {
	obj := admissionregistrationv1.WebhookClientConfig{}

	if len(l) == 0 || l[0] == nil {
		return obj
	}

	in := l[0].(map[string]interface{})

	if v, ok := in["ca_bundle"].(string); ok {
		obj.CABundle = []byte(v)
	}

	if v, ok := in["service"].([]interface{}); ok {
		obj.Service = expandServiceReference(v)
	}

	if v, ok := in["url"].(string); ok && v != "" {
		obj.URL = ptrToString(v)
	}

	return obj
}

func flattenRuleWithOperations(in admissionregistrationv1.RuleWithOperations) map[string]interface{} {
	att := map[string]interface{}{}

	att["api_groups"] = in.APIGroups
	att["api_versions"] = in.APIVersions
	att["operations"] = in.Operations
	att["resources"] = in.Resources

	if in.Scope != nil {
		att["scope"] = *in.Scope
	}

	return att
}

func expandRuleWithOperations(in map[string]interface{}) admissionregistrationv1.RuleWithOperations {
	obj := admissionregistrationv1.RuleWithOperations{}

	if v, ok := in["api_groups"].([]interface{}); ok {
		obj.APIGroups = expandStringSlice(v)
	}

	if v, ok := in["api_versions"].([]interface{}); ok {
		obj.APIVersions = expandStringSlice(v)
	}

	if v, ok := in["operations"].([]interface{}); ok {
		for _, op := range v {
			if op != nil {
				obj.Operations = append(obj.Operations, admissionregistrationv1.OperationType(op.(string)))
			}
		}
	}

	if v, ok := in["resources"].([]interface{}); ok {
		obj.Resources = expandStringSlice(v)
	}

	if v, ok := in["scope"].(string); ok {
		scope := admissionregistrationv1.ScopeType(v)
		obj.Scope = &scope
	}

	return obj
}

func flattenValidatingWebhook(in admissionregistrationv1.ValidatingWebhook) map[string]interface{} {
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

func expandValidatingWebhook(in map[string]interface{}) admissionregistrationv1.ValidatingWebhook {
	obj := admissionregistrationv1.ValidatingWebhook{}

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

func expandValidatingWebhooks(in []interface{}) []admissionregistrationv1.ValidatingWebhook {
	webhooks := []admissionregistrationv1.ValidatingWebhook{}
	for _, h := range in {
		webhooks = append(webhooks, expandValidatingWebhook(h.(map[string]interface{})))
	}
	return webhooks
}

func flattenValidatingWebhooks(in []admissionregistrationv1.ValidatingWebhook) []interface{} {
	webhooks := []interface{}{}
	for _, h := range in {
		webhooks = append(webhooks, flattenValidatingWebhook(h))
	}
	return webhooks
}
