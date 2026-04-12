// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func flattenHTTPRouteSpec(in gatewayv1.HTTPRouteSpec) []interface{} {
	att := make(map[string]interface{})

	if len(in.ParentRefs) > 0 {
		parentRefs := make([]interface{}, len(in.ParentRefs))
		for i, p := range in.ParentRefs {
			parentRefs[i] = flattenParentReference(p)
		}
		att["parent_refs"] = parentRefs
	}

	if len(in.Hostnames) > 0 {
		hostnames := make([]string, len(in.Hostnames))
		for i, h := range in.Hostnames {
			hostnames[i] = string(h)
		}
		att["hostnames"] = hostnames
	}

	if in.UseDefaultGateways != "" {
		att["use_default_gateways"] = string(in.UseDefaultGateways)
	}

	if len(in.Rules) > 0 {
		rules := make([]interface{}, len(in.Rules))
		for i, rule := range in.Rules {
			rules[i] = flattenHTTPRouteRule(rule)
		}
		att["rules"] = rules
	}

	return []interface{}{att}
}

func flattenHTTPRouteRule(in gatewayv1.HTTPRouteRule) map[string]interface{} {
	rule := make(map[string]interface{})

	if in.Name != nil {
		rule["name"] = string(*in.Name)
	}

	if len(in.Matches) > 0 {
		matches := make([]interface{}, len(in.Matches))
		for i, m := range in.Matches {
			matches[i] = flattenHTTPRouteMatch(m)
		}
		rule["matches"] = matches
	}

	if len(in.Filters) > 0 {
		filters := make([]interface{}, len(in.Filters))
		for i, f := range in.Filters {
			filters[i] = flattenHTTPRouteFilter(f)
		}
		rule["filters"] = filters
	}

	if len(in.BackendRefs) > 0 {
		backendRefs := make([]interface{}, len(in.BackendRefs))
		for i, br := range in.BackendRefs {
			backendRefs[i] = flattenHTTPBackendRef(br)
		}
		rule["backend_refs"] = backendRefs
	}

	if in.Timeouts != nil {
		rule["timeouts"] = flattenHTTPRouteTimeouts(in.Timeouts)
	}

	if in.Retry != nil {
		rule["retry"] = flattenHTTPRouteRetry(in.Retry)
	}

	if in.SessionPersistence != nil {
		rule["session_persistence"] = flattenSessionPersistence(in.SessionPersistence)
	}

	return rule
}

func flattenHTTPRouteMatch(in gatewayv1.HTTPRouteMatch) map[string]interface{} {
	match := make(map[string]interface{})

	if in.Path != nil {
		match["path"] = flattenHTTPPathMatch(*in.Path)
	}

	if len(in.Headers) > 0 {
		headers := make([]interface{}, len(in.Headers))
		for i, h := range in.Headers {
			headers[i] = flattenHTTPHeaderMatch(h)
		}
		match["headers"] = headers
	}

	if len(in.QueryParams) > 0 {
		queryParams := make([]interface{}, len(in.QueryParams))
		for i, q := range in.QueryParams {
			queryParams[i] = flattenHTTPQueryParamMatch(q)
		}
		match["query_params"] = queryParams
	}

	if in.Method != nil {
		match["method"] = string(*in.Method)
	}

	return match
}

func flattenHTTPPathMatch(in gatewayv1.HTTPPathMatch) []interface{} {
	path := make(map[string]interface{})
	if in.Type != nil {
		path["type"] = string(*in.Type)
	}
	if in.Value != nil {
		path["value"] = *in.Value
	}
	return []interface{}{path}
}

func flattenHTTPHeaderMatch(in gatewayv1.HTTPHeaderMatch) map[string]interface{} {
	header := make(map[string]interface{})
	header["name"] = string(in.Name)
	header["value"] = in.Value
	if in.Type != nil {
		header["type"] = string(*in.Type)
	}
	return header
}

func flattenHTTPQueryParamMatch(in gatewayv1.HTTPQueryParamMatch) map[string]interface{} {
	qp := make(map[string]interface{})
	qp["name"] = string(in.Name)
	qp["value"] = in.Value
	if in.Type != nil {
		qp["type"] = string(*in.Type)
	}
	return qp
}

func flattenHTTPRouteFilter(in gatewayv1.HTTPRouteFilter) map[string]interface{} {
	filter := make(map[string]interface{})
	filter["type"] = string(in.Type)

	if in.RequestHeaderModifier != nil {
		filter["request_header_modifier"] = flattenHTTPHeaderFilter(in.RequestHeaderModifier)
	}

	if in.ResponseHeaderModifier != nil {
		filter["response_header_modifier"] = flattenHTTPHeaderFilter(in.ResponseHeaderModifier)
	}

	if in.RequestRedirect != nil {
		filter["request_redirect"] = flattenHTTPRequestRedirectFilter(in.RequestRedirect)
	}

	if in.URLRewrite != nil {
		filter["url_rewrite"] = flattenHTTPURLRewriteFilter(in.URLRewrite)
	}

	if in.RequestMirror != nil {
		filter["request_mirror"] = flattenHTTPRequestMirrorFilter(in.RequestMirror)
	}

	if in.CORS != nil {
		filter["cors"] = flattenHTTPCORSFilter(in.CORS)
	}

	if in.ExtensionRef != nil {
		filter["extension_ref"] = flattenLocalObjectReferenceHTTPRoute(*in.ExtensionRef)
	}

	return filter
}

func flattenHTTPHeaderFilter(in *gatewayv1.HTTPHeaderFilter) []interface{} {
	filter := make(map[string]interface{})

	if len(in.Set) > 0 {
		set := make([]interface{}, len(in.Set))
		for i, h := range in.Set {
			set[i] = flattenHTTPHeaderHTTPRoute(h)
		}
		filter["set"] = set
	}

	if len(in.Add) > 0 {
		add := make([]interface{}, len(in.Add))
		for i, h := range in.Add {
			add[i] = flattenHTTPHeaderHTTPRoute(h)
		}
		filter["add"] = add
	}

	if len(in.Remove) > 0 {
		filter["remove"] = in.Remove
	}

	return []interface{}{filter}
}

func flattenHTTPHeaderHTTPRoute(in gatewayv1.HTTPHeader) map[string]interface{} {
	header := make(map[string]interface{})
	header["name"] = string(in.Name)
	header["value"] = in.Value
	return header
}

func flattenHTTPRequestRedirectFilter(in *gatewayv1.HTTPRequestRedirectFilter) []interface{} {
	redirect := make(map[string]interface{})

	if in.Scheme != nil {
		redirect["scheme"] = *in.Scheme
	}

	if in.Hostname != nil {
		redirect["hostname"] = string(*in.Hostname)
	}

	if in.Path != nil {
		redirect["path"] = flattenHTTPPathModifier(in.Path)
	}

	if in.Port != nil {
		redirect["port"] = *in.Port
	}

	if in.StatusCode != nil {
		redirect["status_code"] = *in.StatusCode
	}

	return []interface{}{redirect}
}

func flattenHTTPPathModifier(in *gatewayv1.HTTPPathModifier) []interface{} {
	mod := make(map[string]interface{})
	if in.Type != "" {
		mod["type"] = string(in.Type)
	}
	if in.ReplaceFullPath != nil {
		mod["replace_full_path"] = *in.ReplaceFullPath
	}
	if in.ReplacePrefixMatch != nil {
		mod["replace_prefix_match"] = *in.ReplacePrefixMatch
	}
	return []interface{}{mod}
}

func flattenHTTPURLRewriteFilter(in *gatewayv1.HTTPURLRewriteFilter) []interface{} {
	rewrite := make(map[string]interface{})

	if in.Hostname != nil {
		rewrite["hostname"] = string(*in.Hostname)
	}

	if in.Path != nil {
		rewrite["path"] = flattenHTTPPathModifier(in.Path)
	}

	return []interface{}{rewrite}
}

func flattenHTTPRequestMirrorFilter(in *gatewayv1.HTTPRequestMirrorFilter) []interface{} {
	mirror := make(map[string]interface{})
	mirror["backend_ref"] = flattenBackendObjectReference(in.BackendRef)

	if in.Percent != nil {
		mirror["percent"] = *in.Percent
	}

	return []interface{}{mirror}
}

func flattenHTTPCORSFilter(in *gatewayv1.HTTPCORSFilter) []interface{} {
	cors := make(map[string]interface{})

	if len(in.AllowOrigins) > 0 {
		origins := make([]string, len(in.AllowOrigins))
		for i, o := range in.AllowOrigins {
			origins[i] = string(o)
		}
		cors["allow_origins"] = origins
	}

	if in.AllowCredentials != nil {
		cors["allow_credentials"] = *in.AllowCredentials
	}

	if len(in.AllowMethods) > 0 {
		methods := make([]string, len(in.AllowMethods))
		for i, m := range in.AllowMethods {
			methods[i] = string(m)
		}
		cors["allow_methods"] = methods
	}

	if len(in.AllowHeaders) > 0 {
		headers := make([]string, len(in.AllowHeaders))
		for i, h := range in.AllowHeaders {
			headers[i] = string(h)
		}
		cors["allow_headers"] = headers
	}

	if len(in.ExposeHeaders) > 0 {
		exposeHeaders := make([]string, len(in.ExposeHeaders))
		for i, h := range in.ExposeHeaders {
			exposeHeaders[i] = string(h)
		}
		cors["expose_headers"] = exposeHeaders
	}

	if in.MaxAge > 0 {
		cors["max_age"] = in.MaxAge
	}

	return []interface{}{cors}
}

func flattenSessionPersistence(in *gatewayv1.SessionPersistence) []interface{} {
	sp := make(map[string]interface{})

	if in.SessionName != nil {
		sp["session_name"] = *in.SessionName
	}

	if in.Type != nil {
		sp["type"] = string(*in.Type)
	}

	if in.CookieConfig != nil {
		sp["cookie_config"] = flattenCookieConfig(in.CookieConfig)
	}

	if in.AbsoluteTimeout != nil {
		sp["absolute_timeout"] = *in.AbsoluteTimeout
	}

	if in.IdleTimeout != nil {
		sp["idle_timeout"] = *in.IdleTimeout
	}

	return []interface{}{sp}
}

func flattenCookieConfig(in *gatewayv1.CookieConfig) []interface{} {
	cc := make(map[string]interface{})

	if in.LifetimeType != nil {
		cc["lifetime_type"] = string(*in.LifetimeType)
	}

	return []interface{}{cc}
}

func flattenHTTPBackendRef(in gatewayv1.HTTPBackendRef) map[string]interface{} {
	ref := make(map[string]interface{})

	if in.Group != nil {
		ref["group"] = string(*in.Group)
	}

	if in.Kind != nil {
		ref["kind"] = string(*in.Kind)
	}

	ref["name"] = string(in.Name)

	if in.Namespace != nil {
		ref["namespace"] = string(*in.Namespace)
	}

	if in.Port != nil {
		ref["port"] = int(*in.Port)
	}

	if in.Weight != nil {
		ref["weight"] = int(*in.Weight)
	}

	if len(in.Filters) > 0 {
		filters := make([]interface{}, len(in.Filters))
		for i, f := range in.Filters {
			filters[i] = flattenHTTPRouteFilter(f)
		}
		ref["filters"] = filters
	}

	return ref
}

func flattenBackendObjectReference(in gatewayv1.BackendObjectReference) []interface{} {
	ref := make(map[string]interface{})

	if in.Group != nil {
		ref["group"] = string(*in.Group)
	}

	if in.Kind != nil {
		ref["kind"] = string(*in.Kind)
	}

	ref["name"] = string(in.Name)

	if in.Namespace != nil {
		ref["namespace"] = string(*in.Namespace)
	}

	if in.Port != nil {
		ref["port"] = *in.Port
	}

	return []interface{}{ref}
}

func flattenLocalObjectReferenceHTTPRoute(in gatewayv1.LocalObjectReference) []interface{} {
	ref := make(map[string]interface{})
	ref["name"] = in.Name
	return []interface{}{ref}
}

func flattenHTTPRouteTimeouts(in *gatewayv1.HTTPRouteTimeouts) []interface{} {
	timeouts := make(map[string]interface{})

	if in.Request != nil {
		timeouts["request"] = string(*in.Request)
	}

	if in.BackendRequest != nil {
		timeouts["backend_request"] = string(*in.BackendRequest)
	}

	return []interface{}{timeouts}
}

func flattenHTTPRouteRetry(in *gatewayv1.HTTPRouteRetry) []interface{} {
	retry := make(map[string]interface{})

	if len(in.Codes) > 0 {
		codes := make([]int, len(in.Codes))
		for i, c := range in.Codes {
			codes[i] = int(c)
		}
		retry["codes"] = codes
	}

	if in.Attempts != nil {
		retry["attempts"] = *in.Attempts
	}

	if in.Backoff != nil {
		retry["backoff"] = string(*in.Backoff)
	}

	return []interface{}{retry}
}

func flattenHTTPRouteStatus(in gatewayv1.HTTPRouteStatus) []interface{} {
	status := make(map[string]interface{})

	if len(in.Parents) > 0 {
		status["parents"] = flattenRouteParentStatuses(in.Parents)
	}

	return []interface{}{status}
}

func flattenRouteParentStatuses(in []gatewayv1.RouteParentStatus) []interface{} {
	result := make([]interface{}, len(in))
	for i, p := range in {
		result[i] = flattenRouteParentStatus(p)
	}
	return result
}

func flattenRouteParentStatus(in gatewayv1.RouteParentStatus) map[string]interface{} {
	parent := make(map[string]interface{})

	parent["parent_ref"] = []interface{}{flattenParentReference(in.ParentRef)}
	parent["controller_name"] = string(in.ControllerName)

	if len(in.Conditions) > 0 {
		parent["conditions"] = flattenConditions(in.Conditions)
	}

	return parent
}

func flattenParentReference(in gatewayv1.ParentReference) map[string]interface{} {
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

	return ref
}

func flattenConditions(in []metav1.Condition) []interface{} {
	result := make([]interface{}, len(in))
	for i, c := range in {
		condition := make(map[string]interface{})
		condition["type"] = c.Type
		condition["status"] = string(c.Status)
		condition["message"] = c.Message
		condition["reason"] = c.Reason
		if c.LastTransitionTime.IsZero() == false {
			condition["last_transition_time"] = c.LastTransitionTime.Format("2006-01-02T15:04:05Z")
		}
		if c.ObservedGeneration != 0 {
			condition["observed_generation"] = c.ObservedGeneration
		}
		result[i] = condition
	}
	return result
}

func expandHTTPRouteSpec(l []interface{}) gatewayv1.HTTPRouteSpec {
	if len(l) == 0 || l[0] == nil {
		return gatewayv1.HTTPRouteSpec{}
	}

	in := l[0].(map[string]interface{})
	obj := gatewayv1.HTTPRouteSpec{}

	if v, ok := in["parent_refs"].([]interface{}); ok && len(v) > 0 {
		obj.ParentRefs = expandParentReferences(v)
	}

	if v, ok := in["hostnames"].([]interface{}); ok && len(v) > 0 {
		hostnames := make([]gatewayv1.Hostname, len(v))
		for i, h := range v {
			hostnames[i] = gatewayv1.Hostname(h.(string))
		}
		obj.Hostnames = hostnames
	}

	if v, ok := in["use_default_gateways"].(string); ok && v != "" {
		obj.UseDefaultGateways = gatewayv1.GatewayDefaultScope(v)
	}

	if v, ok := in["rules"].([]interface{}); ok && len(v) > 0 {
		obj.Rules = expandHTTPRouteRules(v)
	}

	return obj
}

func expandParentReferences(l []interface{}) []gatewayv1.ParentReference {
	if len(l) == 0 {
		return nil
	}

	result := make([]gatewayv1.ParentReference, len(l))
	for i, item := range l {
		if item == nil {
			continue
		}
		result[i] = expandParentReference(item.(map[string]interface{}))
	}
	return result
}

func expandParentReference(in map[string]interface{}) gatewayv1.ParentReference {
	obj := gatewayv1.ParentReference{}

	if v, ok := in["group"].(string); ok && v != "" {
		g := gatewayv1.Group(v)
		obj.Group = &g
	}

	if v, ok := in["kind"].(string); ok && v != "" {
		k := gatewayv1.Kind(v)
		obj.Kind = &k
	}

	if v, ok := in["namespace"].(string); ok && v != "" {
		ns := gatewayv1.Namespace(v)
		obj.Namespace = &ns
	}

	if v, ok := in["name"].(string); ok && v != "" {
		obj.Name = gatewayv1.ObjectName(v)
	}

	if v, ok := in["section_name"].(string); ok && v != "" {
		sn := gatewayv1.SectionName(v)
		obj.SectionName = &sn
	}

	if v, ok := in["port"].(int); ok && v > 0 {
		p := gatewayv1.PortNumber(v)
		obj.Port = &p
	}

	return obj
}

func expandHTTPRouteRules(l []interface{}) []gatewayv1.HTTPRouteRule {
	if len(l) == 0 {
		return nil
	}

	result := make([]gatewayv1.HTTPRouteRule, len(l))
	for i, item := range l {
		if item == nil {
			continue
		}
		result[i] = expandHTTPRouteRule(item.(map[string]interface{}))
	}
	return result
}

func expandHTTPRouteRule(in map[string]interface{}) gatewayv1.HTTPRouteRule {
	obj := gatewayv1.HTTPRouteRule{}

	if v, ok := in["name"].(string); ok && v != "" {
		name := gatewayv1.SectionName(v)
		obj.Name = &name
	}

	if v, ok := in["matches"].([]interface{}); ok && len(v) > 0 {
		obj.Matches = expandHTTPRouteMatches(v)
	}

	if v, ok := in["filters"].([]interface{}); ok && len(v) > 0 {
		obj.Filters = expandHTTPRouteFilters(v)
	}

	if v, ok := in["backend_refs"].([]interface{}); ok && len(v) > 0 {
		obj.BackendRefs = expandHTTPBackendRefs(v)
	}

	if v, ok := in["timeouts"].([]interface{}); ok && len(v) > 0 {
		obj.Timeouts = expandHTTPRouteTimeouts(v)
	}

	if v, ok := in["retry"].([]interface{}); ok && len(v) > 0 {
		obj.Retry = expandHTTPRouteRetry(v)
	}

	if v, ok := in["session_persistence"].([]interface{}); ok && len(v) > 0 {
		obj.SessionPersistence = expandSessionPersistence(v)
	}

	return obj
}

func expandHTTPRouteMatches(l []interface{}) []gatewayv1.HTTPRouteMatch {
	if len(l) == 0 {
		return nil
	}

	result := make([]gatewayv1.HTTPRouteMatch, len(l))
	for i, item := range l {
		if item == nil {
			continue
		}
		result[i] = expandHTTPRouteMatch(item.(map[string]interface{}))
	}
	return result
}

func expandHTTPRouteMatch(in map[string]interface{}) gatewayv1.HTTPRouteMatch {
	obj := gatewayv1.HTTPRouteMatch{}

	if v, ok := in["path"].([]interface{}); ok && len(v) > 0 {
		obj.Path = expandHTTPPathMatch(v)
	}

	if v, ok := in["headers"].([]interface{}); ok && len(v) > 0 {
		obj.Headers = expandHTTPHeaderMatches(v)
	}

	if v, ok := in["query_params"].([]interface{}); ok && len(v) > 0 {
		obj.QueryParams = expandHTTPQueryParamMatches(v)
	}

	if v, ok := in["method"].(string); ok && v != "" {
		method := gatewayv1.HTTPMethod(v)
		obj.Method = &method
	}

	return obj
}

func expandHTTPPathMatch(l []interface{}) *gatewayv1.HTTPPathMatch {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	in := l[0].(map[string]interface{})
	obj := &gatewayv1.HTTPPathMatch{}

	if v, ok := in["type"].(string); ok && v != "" {
		t := gatewayv1.PathMatchType(v)
		obj.Type = &t
	}

	if v, ok := in["value"].(string); ok && v != "" {
		obj.Value = &v
	}

	return obj
}

func expandHTTPHeaderMatches(l []interface{}) []gatewayv1.HTTPHeaderMatch {
	if len(l) == 0 {
		return nil
	}

	result := make([]gatewayv1.HTTPHeaderMatch, len(l))
	for i, item := range l {
		if item == nil {
			continue
		}
		result[i] = expandHTTPHeaderMatch(item.(map[string]interface{}))
	}
	return result
}

func expandHTTPHeaderMatch(in map[string]interface{}) gatewayv1.HTTPHeaderMatch {
	obj := gatewayv1.HTTPHeaderMatch{}

	if v, ok := in["name"].(string); ok && v != "" {
		obj.Name = gatewayv1.HTTPHeaderName(v)
	}

	if v, ok := in["value"].(string); ok && v != "" {
		obj.Value = v
	}

	if v, ok := in["type"].(string); ok && v != "" {
		t := gatewayv1.HeaderMatchType(v)
		obj.Type = &t
	}

	return obj
}

func expandHTTPQueryParamMatches(l []interface{}) []gatewayv1.HTTPQueryParamMatch {
	if len(l) == 0 {
		return nil
	}

	result := make([]gatewayv1.HTTPQueryParamMatch, len(l))
	for i, item := range l {
		if item == nil {
			continue
		}
		result[i] = expandHTTPQueryParamMatch(item.(map[string]interface{}))
	}
	return result
}

func expandHTTPQueryParamMatch(in map[string]interface{}) gatewayv1.HTTPQueryParamMatch {
	obj := gatewayv1.HTTPQueryParamMatch{}

	if v, ok := in["name"].(string); ok && v != "" {
		obj.Name = gatewayv1.HTTPHeaderName(v)
	}

	if v, ok := in["value"].(string); ok && v != "" {
		obj.Value = v
	}

	if v, ok := in["type"].(string); ok && v != "" {
		t := gatewayv1.QueryParamMatchType(v)
		obj.Type = &t
	}

	return obj
}

func expandHTTPRouteFilters(l []interface{}) []gatewayv1.HTTPRouteFilter {
	if len(l) == 0 {
		return nil
	}

	result := make([]gatewayv1.HTTPRouteFilter, len(l))
	for i, item := range l {
		if item == nil {
			continue
		}
		result[i] = expandHTTPRouteFilter(item.(map[string]interface{}))
	}
	return result
}

func expandHTTPRouteFilter(in map[string]interface{}) gatewayv1.HTTPRouteFilter {
	obj := gatewayv1.HTTPRouteFilter{}

	if v, ok := in["type"].(string); ok && v != "" {
		obj.Type = gatewayv1.HTTPRouteFilterType(v)
	}

	if v, ok := in["request_header_modifier"].([]interface{}); ok && len(v) > 0 {
		obj.RequestHeaderModifier = expandHTTPHeaderFilter(v)
	}

	if v, ok := in["response_header_modifier"].([]interface{}); ok && len(v) > 0 {
		obj.ResponseHeaderModifier = expandHTTPHeaderFilter(v)
	}

	if v, ok := in["request_redirect"].([]interface{}); ok && len(v) > 0 {
		obj.RequestRedirect = expandHTTPRequestRedirectFilter(v)
	}

	if v, ok := in["url_rewrite"].([]interface{}); ok && len(v) > 0 {
		obj.URLRewrite = expandHTTPURLRewriteFilter(v)
	}

	if v, ok := in["request_mirror"].([]interface{}); ok && len(v) > 0 {
		obj.RequestMirror = expandHTTPRequestMirrorFilter(v)
	}

	if v, ok := in["cors"].([]interface{}); ok && len(v) > 0 {
		obj.CORS = expandHTTPCORSFilter(v)
	}

	if v, ok := in["extension_ref"].([]interface{}); ok && len(v) > 0 {
		ref := expandLocalObjectReferenceHTTPRoute(v)
		obj.ExtensionRef = &ref
	}

	return obj
}

func expandHTTPHeaderFilter(l []interface{}) *gatewayv1.HTTPHeaderFilter {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	in := l[0].(map[string]interface{})
	obj := &gatewayv1.HTTPHeaderFilter{}

	if v, ok := in["set"].([]interface{}); ok && len(v) > 0 {
		set := make([]gatewayv1.HTTPHeader, len(v))
		for i, h := range v {
			set[i] = expandHTTPHeaderHTTPRoute(h.(map[string]interface{}))
		}
		obj.Set = set
	}

	if v, ok := in["add"].([]interface{}); ok && len(v) > 0 {
		add := make([]gatewayv1.HTTPHeader, len(v))
		for i, h := range v {
			add[i] = expandHTTPHeaderHTTPRoute(h.(map[string]interface{}))
		}
		obj.Add = add
	}

	if v, ok := in["remove"].([]interface{}); ok && len(v) > 0 {
		remove := make([]string, len(v))
		for i, r := range v {
			remove[i] = r.(string)
		}
		obj.Remove = remove
	}

	return obj
}

func expandHTTPHeaderHTTPRoute(in map[string]interface{}) gatewayv1.HTTPHeader {
	obj := gatewayv1.HTTPHeader{}

	if v, ok := in["name"].(string); ok && v != "" {
		obj.Name = gatewayv1.HTTPHeaderName(v)
	}

	if v, ok := in["value"].(string); ok {
		obj.Value = v
	}

	return obj
}

func expandHTTPRequestRedirectFilter(l []interface{}) *gatewayv1.HTTPRequestRedirectFilter {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	in := l[0].(map[string]interface{})
	obj := &gatewayv1.HTTPRequestRedirectFilter{}

	if v, ok := in["scheme"].(string); ok && v != "" {
		obj.Scheme = &v
	}

	if v, ok := in["hostname"].(string); ok && v != "" {
		hostname := gatewayv1.PreciseHostname(v)
		obj.Hostname = &hostname
	}

	if v, ok := in["path"].([]interface{}); ok && len(v) > 0 {
		obj.Path = expandHTTPPathModifier(v)
	}

	if v, ok := in["port"].(int); ok && v > 0 {
		port := gatewayv1.PortNumber(v)
		obj.Port = &port
	}

	if v, ok := in["status_code"].(int); ok && v > 0 {
		statusCode := v
		obj.StatusCode = &statusCode
	}

	return obj
}

func expandHTTPPathModifier(l []interface{}) *gatewayv1.HTTPPathModifier {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	in := l[0].(map[string]interface{})
	obj := &gatewayv1.HTTPPathModifier{}

	if v, ok := in["type"].(string); ok && v != "" {
		obj.Type = gatewayv1.HTTPPathModifierType(v)
	}

	if v, ok := in["replace_full_path"].(string); ok && v != "" {
		obj.ReplaceFullPath = &v
	}

	if v, ok := in["replace_prefix_match"].(string); ok && v != "" {
		obj.ReplacePrefixMatch = &v
	}

	return obj
}

func expandHTTPURLRewriteFilter(l []interface{}) *gatewayv1.HTTPURLRewriteFilter {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	in := l[0].(map[string]interface{})
	obj := &gatewayv1.HTTPURLRewriteFilter{}

	if v, ok := in["hostname"].(string); ok && v != "" {
		hostname := gatewayv1.PreciseHostname(v)
		obj.Hostname = &hostname
	}

	if v, ok := in["path"].([]interface{}); ok && len(v) > 0 {
		obj.Path = expandHTTPPathModifier(v)
	}

	return obj
}

func expandHTTPRequestMirrorFilter(l []interface{}) *gatewayv1.HTTPRequestMirrorFilter {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	in := l[0].(map[string]interface{})
	obj := &gatewayv1.HTTPRequestMirrorFilter{}

	if v, ok := in["backend_ref"].([]interface{}); ok && len(v) > 0 {
		obj.BackendRef = expandBackendObjectReference(v)
	}

	if v, ok := in["percent"].(int); ok && v > 0 {
		percent := int32(v)
		obj.Percent = &percent
	}

	return obj
}

func expandHTTPCORSFilter(l []interface{}) *gatewayv1.HTTPCORSFilter {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	in := l[0].(map[string]interface{})
	obj := &gatewayv1.HTTPCORSFilter{}

	if v, ok := in["allow_origins"].([]interface{}); ok && len(v) > 0 {
		origins := make([]gatewayv1.CORSOrigin, len(v))
		for i, o := range v {
			origins[i] = gatewayv1.CORSOrigin(o.(string))
		}
		obj.AllowOrigins = origins
	}

	if v, ok := in["allow_credentials"].(bool); ok {
		obj.AllowCredentials = &v
	}

	if v, ok := in["allow_methods"].([]interface{}); ok && len(v) > 0 {
		methods := make([]gatewayv1.HTTPMethodWithWildcard, len(v))
		for i, m := range v {
			methods[i] = gatewayv1.HTTPMethodWithWildcard(m.(string))
		}
		obj.AllowMethods = methods
	}

	if v, ok := in["allow_headers"].([]interface{}); ok && len(v) > 0 {
		headers := make([]gatewayv1.HTTPHeaderName, len(v))
		for i, h := range v {
			headers[i] = gatewayv1.HTTPHeaderName(h.(string))
		}
		obj.AllowHeaders = headers
	}

	if v, ok := in["expose_headers"].([]interface{}); ok && len(v) > 0 {
		headers := make([]gatewayv1.HTTPHeaderName, len(v))
		for i, h := range v {
			headers[i] = gatewayv1.HTTPHeaderName(h.(string))
		}
		obj.ExposeHeaders = headers
	}

	if v, ok := in["max_age"].(int); ok && v > 0 {
		obj.MaxAge = int32(v)
	}

	return obj
}

func expandSessionPersistence(l []interface{}) *gatewayv1.SessionPersistence {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	in := l[0].(map[string]interface{})
	obj := &gatewayv1.SessionPersistence{}

	if v, ok := in["session_name"].(string); ok && v != "" {
		obj.SessionName = &v
	}

	if v, ok := in["type"].(string); ok && v != "" {
		t := gatewayv1.SessionPersistenceType(v)
		obj.Type = &t
	}

	if v, ok := in["cookie_config"].([]interface{}); ok && len(v) > 0 {
		obj.CookieConfig = expandCookieConfig(v)
	}

	if v, ok := in["absolute_timeout"].(string); ok && v != "" {
		d := gatewayv1.Duration(v)
		obj.AbsoluteTimeout = &d
	}

	if v, ok := in["idle_timeout"].(string); ok && v != "" {
		d := gatewayv1.Duration(v)
		obj.IdleTimeout = &d
	}

	return obj
}

func expandCookieConfig(l []interface{}) *gatewayv1.CookieConfig {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	in := l[0].(map[string]interface{})
	obj := &gatewayv1.CookieConfig{}

	if v, ok := in["lifetime_type"].(string); ok && v != "" {
		lt := gatewayv1.CookieLifetimeType(v)
		obj.LifetimeType = &lt
	}

	return obj
}

func expandHTTPBackendRefs(l []interface{}) []gatewayv1.HTTPBackendRef {
	if len(l) == 0 {
		return nil
	}

	result := make([]gatewayv1.HTTPBackendRef, len(l))
	for i, item := range l {
		if item == nil {
			continue
		}
		result[i] = expandHTTPBackendRef(item.(map[string]interface{}))
	}
	return result
}

func expandHTTPBackendRef(in map[string]interface{}) gatewayv1.HTTPBackendRef {
	obj := gatewayv1.HTTPBackendRef{}

	if v, ok := in["group"].(string); ok && v != "" {
		g := gatewayv1.Group(v)
		obj.Group = &g
	}

	if v, ok := in["kind"].(string); ok && v != "" {
		k := gatewayv1.Kind(v)
		obj.Kind = &k
	}

	if v, ok := in["name"].(string); ok && v != "" {
		obj.Name = gatewayv1.ObjectName(v)
	}

	if v, ok := in["namespace"].(string); ok && v != "" {
		ns := gatewayv1.Namespace(v)
		obj.Namespace = &ns
	}

	if v, ok := in["port"].(int); ok && v > 0 {
		p := gatewayv1.PortNumber(v)
		obj.Port = &p
	}

	if v, ok := in["weight"].(int); ok && v > 0 {
		w := int32(v)
		obj.Weight = &w
	}

	if v, ok := in["filters"].([]interface{}); ok && len(v) > 0 {
		obj.Filters = expandHTTPRouteFilters(v)
	}

	return obj
}

func expandBackendObjectReference(l []interface{}) gatewayv1.BackendObjectReference {
	if len(l) == 0 || l[0] == nil {
		return gatewayv1.BackendObjectReference{}
	}

	in := l[0].(map[string]interface{})
	obj := gatewayv1.BackendObjectReference{}

	if v, ok := in["group"].(string); ok && v != "" {
		g := gatewayv1.Group(v)
		obj.Group = &g
	}

	if v, ok := in["kind"].(string); ok && v != "" {
		k := gatewayv1.Kind(v)
		obj.Kind = &k
	}

	if v, ok := in["name"].(string); ok && v != "" {
		obj.Name = gatewayv1.ObjectName(v)
	}

	if v, ok := in["namespace"].(string); ok && v != "" {
		ns := gatewayv1.Namespace(v)
		obj.Namespace = &ns
	}

	if v, ok := in["port"].(int); ok && v > 0 {
		p := gatewayv1.PortNumber(v)
		obj.Port = &p
	}

	return obj
}

func expandLocalObjectReferenceHTTPRoute(l []interface{}) gatewayv1.LocalObjectReference {
	if len(l) == 0 || l[0] == nil {
		return gatewayv1.LocalObjectReference{}
	}

	in := l[0].(map[string]interface{})
	obj := gatewayv1.LocalObjectReference{}

	if v, ok := in["name"].(string); ok && v != "" {
		obj.Name = gatewayv1.ObjectName(v)
	}

	return obj
}

func expandHTTPRouteTimeouts(l []interface{}) *gatewayv1.HTTPRouteTimeouts {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	in := l[0].(map[string]interface{})
	obj := &gatewayv1.HTTPRouteTimeouts{}

	if v, ok := in["request"].(string); ok && v != "" {
		d := gatewayv1.Duration(v)
		obj.Request = &d
	}

	if v, ok := in["backend_request"].(string); ok && v != "" {
		d := gatewayv1.Duration(v)
		obj.BackendRequest = &d
	}

	return obj
}

func expandHTTPRouteRetry(l []interface{}) *gatewayv1.HTTPRouteRetry {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	in := l[0].(map[string]interface{})
	obj := &gatewayv1.HTTPRouteRetry{}

	if v, ok := in["codes"].([]interface{}); ok && len(v) > 0 {
		codes := make([]gatewayv1.HTTPRouteRetryStatusCode, len(v))
		for i, c := range v {
			codes[i] = gatewayv1.HTTPRouteRetryStatusCode(c.(int))
		}
		obj.Codes = codes
	}

	if v, ok := in["attempts"].(int); ok && v > 0 {
		a := v
		obj.Attempts = &a
	}

	if v, ok := in["backoff"].(string); ok && v != "" {
		d := gatewayv1.Duration(v)
		obj.Backoff = &d
	}

	return obj
}
