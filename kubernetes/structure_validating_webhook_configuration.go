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
	att["rules"] = rules

	if in.SideEffects != nil {
		att["side_effects"] = *in.SideEffects
	}

	if in.TimeoutSeconds != nil {
		att["timeout_seconds"] = *in.TimeoutSeconds
	}

	return []interface{}{att}, nil
}
