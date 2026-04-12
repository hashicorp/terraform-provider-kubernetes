// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func flattenBackendTLSPolicySpec(in gatewayv1.BackendTLSPolicySpec) []interface{} {
	att := make(map[string]interface{})

	if len(in.TargetRefs) > 0 {
		targets := make([]interface{}, len(in.TargetRefs))
		for i, t := range in.TargetRefs {
			targets[i] = flattenLocalPolicyTargetReferenceWithSectionName(t)
		}
		att["target_refs"] = targets
	}

	att["validation"] = flattenBackendTLSPolicyValidation(in.Validation)

	if len(in.Options) > 0 {
		options := make(map[string]string)
		for k, v := range in.Options {
			options[string(k)] = string(v)
		}
		att["options"] = options
	}

	return []interface{}{att}
}

func flattenLocalPolicyTargetReferenceWithSectionName(in gatewayv1.LocalPolicyTargetReferenceWithSectionName) map[string]interface{} {
	ref := make(map[string]interface{})

	ref["group"] = string(in.Group)
	ref["kind"] = string(in.Kind)
	ref["name"] = string(in.Name)

	if in.SectionName != nil {
		ref["section_name"] = string(*in.SectionName)
	}

	return ref
}

func flattenBackendTLSPolicyValidation(in gatewayv1.BackendTLSPolicyValidation) []interface{} {
	val := make(map[string]interface{})

	if len(in.CACertificateRefs) > 0 {
		certs := make([]interface{}, len(in.CACertificateRefs))
		for i, c := range in.CACertificateRefs {
			certs[i] = flattenLocalObjectReferenceGateway(c)
		}
		val["ca_certificate_refs"] = certs
	}

	if in.WellKnownCACertificates != nil {
		val["well_known_ca_certificates"] = string(*in.WellKnownCACertificates)
	}

	val["hostname"] = string(in.Hostname)

	if len(in.SubjectAltNames) > 0 {
		sans := make([]interface{}, len(in.SubjectAltNames))
		for i, s := range in.SubjectAltNames {
			sans[i] = flattenSubjectAltName(s)
		}
		val["subject_alt_names"] = sans
	}

	return []interface{}{val}
}

func flattenLocalObjectReferenceGateway(in gatewayv1.LocalObjectReference) map[string]interface{} {
	ref := make(map[string]interface{})
	ref["group"] = string(in.Group)
	ref["kind"] = string(in.Kind)
	ref["name"] = string(in.Name)
	return ref
}

func flattenSubjectAltName(in gatewayv1.SubjectAltName) map[string]interface{} {
	san := make(map[string]interface{})

	san["type"] = string(in.Type)

	if in.Hostname != "" {
		san["hostname"] = string(in.Hostname)
	}

	if in.URI != "" {
		san["uri"] = string(in.URI)
	}

	return san
}

func flattenBackendTLSPolicyStatus(in gatewayv1.PolicyStatus) []interface{} {
	status := make(map[string]interface{})

	if len(in.Ancestors) > 0 {
		ancestors := make([]interface{}, len(in.Ancestors))
		for i, a := range in.Ancestors {
			ancestors[i] = flattenPolicyAncestorStatus(a)
		}
		status["ancestors"] = ancestors
	}

	return []interface{}{status}
}

func flattenPolicyAncestorStatus(in gatewayv1.PolicyAncestorStatus) map[string]interface{} {
	ancestor := make(map[string]interface{})

	ancestor["ancestor_ref"] = flattenParentReferenceBackendTLS(in.AncestorRef)
	ancestor["controller_name"] = string(in.ControllerName)

	if len(in.Conditions) > 0 {
		ancestor["conditions"] = flattenConditions(in.Conditions)
	}

	return ancestor
}

func flattenParentReferenceBackendTLS(in gatewayv1.ParentReference) []interface{} {
	ref := make(map[string]interface{})

	if in.Group != nil {
		ref["group"] = string(*in.Group)
	}

	if in.Kind != nil {
		ref["kind"] = string(*in.Kind)
	}

	if in.Namespace != nil {
		ref["namespace"] = string(*in.Namespace)
	}

	ref["name"] = string(in.Name)

	if in.SectionName != nil {
		ref["section_name"] = string(*in.SectionName)
	}

	if in.Port != nil {
		ref["port"] = *in.Port
	}

	return []interface{}{ref}
}

func expandBackendTLSPolicySpec(l []interface{}) gatewayv1.BackendTLSPolicySpec {
	if len(l) == 0 || l[0] == nil {
		return gatewayv1.BackendTLSPolicySpec{}
	}

	in := l[0].(map[string]interface{})
	obj := gatewayv1.BackendTLSPolicySpec{}

	if v, ok := in["target_refs"].([]interface{}); ok && len(v) > 0 {
		obj.TargetRefs = expandLocalPolicyTargetReferenceWithSectionName(v)
	}

	if v, ok := in["validation"].([]interface{}); ok && len(v) > 0 {
		obj.Validation = expandBackendTLSPolicyValidation(v)
	}

	if v, ok := in["options"].(map[string]interface{}); ok && len(v) > 0 {
		options := make(map[gatewayv1.AnnotationKey]gatewayv1.AnnotationValue)
		for key, val := range v {
			options[gatewayv1.AnnotationKey(key)] = gatewayv1.AnnotationValue(val.(string))
		}
		obj.Options = options
	}

	return obj
}

func expandLocalPolicyTargetReferenceWithSectionName(l []interface{}) []gatewayv1.LocalPolicyTargetReferenceWithSectionName {
	if len(l) == 0 {
		return nil
	}

	result := make([]gatewayv1.LocalPolicyTargetReferenceWithSectionName, len(l))
	for i, item := range l {
		if item == nil {
			continue
		}
		in := item.(map[string]interface{})
		ref := gatewayv1.LocalPolicyTargetReferenceWithSectionName{
			LocalPolicyTargetReference: gatewayv1.LocalPolicyTargetReference{
				Group: gatewayv1.Group(in["group"].(string)),
				Kind:  gatewayv1.Kind(in["kind"].(string)),
				Name:  gatewayv1.ObjectName(in["name"].(string)),
			},
		}

		if v, ok := in["section_name"].(string); ok && v != "" {
			sn := gatewayv1.SectionName(v)
			ref.SectionName = &sn
		}

		result[i] = ref
	}
	return result
}

func expandBackendTLSPolicyValidation(l []interface{}) gatewayv1.BackendTLSPolicyValidation {
	if len(l) == 0 || l[0] == nil {
		return gatewayv1.BackendTLSPolicyValidation{}
	}

	in := l[0].(map[string]interface{})
	obj := gatewayv1.BackendTLSPolicyValidation{}

	if v, ok := in["ca_certificate_refs"].([]interface{}); ok && len(v) > 0 {
		certs := make([]gatewayv1.LocalObjectReference, len(v))
		for i, c := range v {
			m := c.(map[string]interface{})
			certs[i] = gatewayv1.LocalObjectReference{
				Group: gatewayv1.Group(m["group"].(string)),
				Kind:  gatewayv1.Kind(m["kind"].(string)),
				Name:  gatewayv1.ObjectName(m["name"].(string)),
			}
		}
		obj.CACertificateRefs = certs
	}

	if v, ok := in["well_known_ca_certificates"].(string); ok && v != "" {
		wkc := gatewayv1.WellKnownCACertificatesType(v)
		obj.WellKnownCACertificates = &wkc
	}

	if v, ok := in["hostname"].(string); ok && v != "" {
		obj.Hostname = gatewayv1.PreciseHostname(v)
	}

	if v, ok := in["subject_alt_names"].([]interface{}); ok && len(v) > 0 {
		obj.SubjectAltNames = expandSubjectAltNames(v)
	}

	return obj
}

func expandSubjectAltNames(l []interface{}) []gatewayv1.SubjectAltName {
	if len(l) == 0 {
		return nil
	}

	result := make([]gatewayv1.SubjectAltName, len(l))
	for i, item := range l {
		if item == nil {
			continue
		}
		in := item.(map[string]interface{})
		san := gatewayv1.SubjectAltName{
			Type: gatewayv1.SubjectAltNameType(in["type"].(string)),
		}

		if v, ok := in["hostname"].(string); ok && v != "" {
			san.Hostname = gatewayv1.Hostname(v)
		}

		if v, ok := in["uri"].(string); ok && v != "" {
			san.URI = gatewayv1.AbsoluteURI(v)
		}

		result[i] = san
	}
	return result
}
