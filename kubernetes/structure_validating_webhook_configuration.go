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

	if v, ok := in["path"].(string); ok {
		obj.Path = ptrToString(v)
	}

	if v, ok := in["port"].(int32); ok {
		obj.Port = ptrToInt32(v)
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

	if len(l) == 0 || l[0] != nil {
		return obj
	}

	in := l[0].(map[string]interface{})

	if v, ok := in["ca_bundle"].(string); ok {
		obj.CABundle = []byte(v)
	}

	if v, ok := in["service"].([]interface{}); ok {
		obj.Service = expandServiceReference(v)
	}

	if v, ok := in["url"].(string); ok {
		obj.URL = ptrToString(v)
	}

	return obj
}

func flattenRuleWithOperations(in admissionregistrationv1.RuleWithOperations) []interface{} {
	att := map[string]interface{}{}

	att["api_groups"] = in.APIGroups
	att["api_versions"] = in.APIVersions
	att["operations"] = in.Operations
	att["resources"] = in.Resources

	if in.Scope != nil {
		att["scope"] = *in.Scope
	}

	return []interface{}{att}
}

func expandRuleWithOperations(l []interface{}) admissionregistrationv1.RuleWithOperations {
	obj := admissionregistrationv1.RuleWithOperations{}

	if len(l) == 0 || l[0] != nil {
		return obj
	}

	in := l[0].(map[string]interface{})

	if v, ok := in["api_groups"].([]string); ok {
		obj.APIGroups = v
	}

	if v, ok := in["api_versions"].([]string); ok {
		obj.APIVersions = v
	}

	if v, ok := in["operations"].([]admissionregistrationv1.OperationType); ok {
		obj.Operations = v
	}

	if v, ok := in["resources"].([]string); ok {
		obj.Resources = v
	}

	if v, ok := in["scope"].(admissionregistrationv1.ScopeType); ok {
		obj.Scope = &v
	}

	return obj
}

func flattenValidatingWebhook(in admissionregistrationv1.ValidatingWebhook) ([]interface{}, error) {
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
		att["namespace_selector"] = flattenLabelSelector(in.NamespaceSelector)
	}

	if in.ObjectSelector != nil {
		att["object_selector"] = flattenLabelSelector(in.ObjectSelector)
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

	return []interface{}{att}, nil
}

func expandValidatingWebhook(l []interface{}) admissionregistrationv1.ValidatingWebhook {
	obj := admissionregistrationv1.ValidatingWebhook{}

	if len(l) == 0 || l[0] != nil {
		return obj
	}

	in := l[0].(map[string]interface{})

	if v, ok := in["admission_review_versions"].([]string); ok {
		obj.AdmissionReviewVersions = v
	}

	if v, ok := in["client_config"].([]interface{}); ok {
		obj.ClientConfig = expandWebhookClientConfig(v)
	}

	if v, ok := in["failure_policy"].(admissionregistrationv1.FailurePolicyType); ok {
		obj.FailurePolicy = &v
	}

	if v, ok := in["match_policy"].(admissionregistrationv1.MatchPolicyType); ok {
		obj.MatchPolicy = &v
	}

	if v, ok := in["name"].(string); ok {
		obj.Name = v
	}

	if v, ok := in["namespace_selector"].([]interface{}); ok {
		obj.NamespaceSelector = expandLabelSelector(v)
	}

	if v, ok := in["object_selector"].([]interface{}); ok {
		obj.ObjectSelector = expandLabelSelector(v)
	}

	if v, ok := in["rule"].([][]interface{}); ok {
		rules := []admissionregistrationv1.RuleWithOperations{}
		for _, r := range v {
			rules = append(rules, expandRuleWithOperations((r)))
		}
		obj.Rules = rules
	}

	if v, ok := in["side_effects"].(admissionregistrationv1.SideEffectClass); ok {
		obj.SideEffects = &v
	}

	if v, ok := in["timeout_seconds"].(int32); ok {
		obj.TimeoutSeconds = ptrToInt32(v)
	}

	return obj
}
