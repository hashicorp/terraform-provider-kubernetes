// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func idParts(id string) (string, string, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		err := fmt.Errorf("Unexpected ID format (%q), expected %q.", id, "namespace/name")
		return "", "", err
	}

	return parts[0], parts[1], nil
}

func buildId(meta metav1.ObjectMeta) string {
	return meta.Namespace + "/" + meta.Name
}

func buildIdWithVersionKind(meta metav1.ObjectMeta, apiVersion, kind string) string {
	id := fmt.Sprintf("apiVersion=%v,kind=%v,name=%s",
		apiVersion, kind, meta.Name)
	if meta.Namespace != "" {
		id += fmt.Sprintf(",namespace=%v", meta.Namespace)
	}
	return id
}

func expandMetadata(in []interface{}) metav1.ObjectMeta {
	meta := metav1.ObjectMeta{}
	if len(in) == 0 || in[0] == nil {
		return meta
	}

	m := in[0].(map[string]interface{})

	if v, ok := m["annotations"].(map[string]interface{}); ok && len(v) > 0 {
		meta.Annotations = expandStringMap(m["annotations"].(map[string]interface{}))
	}

	if v, ok := m["labels"].(map[string]interface{}); ok && len(v) > 0 {
		meta.Labels = expandStringMap(m["labels"].(map[string]interface{}))
	}

	if v, ok := m["generate_name"]; ok {
		meta.GenerateName = v.(string)
	}
	if v, ok := m["name"]; ok {
		meta.Name = v.(string)
	}
	if v, ok := m["namespace"]; ok {
		meta.Namespace = v.(string)
	}

	return meta
}

func patchMetadata(keyPrefix, pathPrefix string, d *schema.ResourceData) PatchOperations {
	ops := make([]PatchOperation, 0)
	if d.HasChange(keyPrefix + "annotations") {
		oldV, newV := d.GetChange(keyPrefix + "annotations")
		diffOps := diffStringMap(pathPrefix+"annotations", oldV.(map[string]interface{}), newV.(map[string]interface{}))
		ops = append(ops, diffOps...)
	}
	if d.HasChange(keyPrefix + "labels") {
		oldV, newV := d.GetChange(keyPrefix + "labels")
		diffOps := diffStringMap(pathPrefix+"labels", oldV.(map[string]interface{}), newV.(map[string]interface{}))
		ops = append(ops, diffOps...)
	}
	return ops
}

func expandStringMap(m map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		result[k] = v.(string)
	}
	return result
}

func expandBase64MapToByteMap(m map[string]interface{}) map[string][]byte {
	result := make(map[string][]byte)
	for k, v := range m {
		b, err := base64.StdEncoding.DecodeString(v.(string))
		if err == nil {
			result[k] = b
		}
	}
	return result
}

func expandStringSlice(s []interface{}) []string {
	result := make([]string, len(s))
	for k, v := range s {
		// Handle the Terraform parser bug which turns empty strings in lists to nil.
		if v == nil {
			result[k] = ""
		} else {
			result[k] = v.(string)
		}
	}
	return result
}

// flattenMetadataFields flattens all metadata fields.
func flattenMetadataFields(meta metav1.ObjectMeta) []interface{} {
	m := make(map[string]interface{})
	m["annotations"] = meta.Annotations
	if meta.GenerateName != "" {
		m["generate_name"] = meta.GenerateName
	}
	m["generation"] = meta.Generation
	m["labels"] = meta.Labels
	m["name"] = meta.Name
	if meta.Namespace != "" {
		m["namespace"] = meta.Namespace
	}
	m["resource_version"] = meta.ResourceVersion
	m["uid"] = string(meta.UID)

	return []interface{}{m}
}

func flattenMetadata(meta metav1.ObjectMeta, d *schema.ResourceData, providerMetadata interface{}) []interface{} {
	metadata := expandMetadata(d.Get("metadata").([]interface{}))

	ignoreAnnotations := providerMetadata.(kubeClientsets).IgnoreAnnotations
	removeInternalKeys(meta.Annotations, metadata.Annotations)
	removeKeys(meta.Annotations, metadata.Annotations, ignoreAnnotations)

	ignoreLabels := providerMetadata.(kubeClientsets).IgnoreLabels
	removeInternalKeys(meta.Labels, metadata.Labels)
	removeKeys(meta.Labels, metadata.Labels, ignoreLabels)

	return flattenMetadataFields(meta)
}

func removeInternalKeys(m map[string]string, d map[string]string) {
	for k := range m {
		if isInternalKey(k) && !isKeyInMap(k, d) {
			delete(m, k)
		}
	}
}

// removeKeys removes given Kubernetes metadata(annotations and labels) keys.
// In that case, they won't be available in the TF state file and will be ignored during apply/plan operations.
func removeKeys(m map[string]string, d map[string]string, ignoreKubernetesMetadataKeys []string) {
	for k := range m {
		if ignoreKey(k, ignoreKubernetesMetadataKeys) && !isKeyInMap(k, d) {
			delete(m, k)
		}
	}
}

func isKeyInMap(key string, d map[string]string) bool {
	_, ok := d[key]
	return ok
}

func isInternalKey(annotationKey string) bool {
	u, err := url.Parse("//" + annotationKey)
	if err != nil {
		return false
	}

	// allow user specified application specific keys
	if u.Hostname() == "app.kubernetes.io" {
		return false
	}

	// allow AWS load balancer configuration annotations
	if u.Hostname() == "service.beta.kubernetes.io" {
		return false
	}

	// internal *.kubernetes.io keys
	if strings.HasSuffix(u.Hostname(), "kubernetes.io") {
		return true
	}

	// Specific to DaemonSet annotations, generated & controlled by the server.
	if strings.Contains(annotationKey, "deprecated.daemonset.template.generation") {
		return true
	}
	return false
}

// ignoreKey reports whether the Kubernetes metadata(annotations and labels) key contains
// any match of the regular expression pattern from the expressions slice.
func ignoreKey(key string, expressions []string) bool {
	for _, e := range expressions {
		if ok, _ := regexp.MatchString(e, key); ok {
			return true
		}
	}

	return false
}

func flattenByteMapToBase64Map(m map[string][]byte) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		result[k] = base64.StdEncoding.EncodeToString([]byte(v))
	}
	return result
}

func flattenByteMapToStringMap(m map[string][]byte) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		result[k] = string(v)
	}
	return result
}

func sliceOfString(slice []interface{}) []string {
	result := make([]string, len(slice))
	for i, s := range slice {
		result[i] = s.(string)
	}
	return result
}

func base64EncodeStringMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		value := v.(string)
		result[k] = base64.StdEncoding.EncodeToString([]byte(value))
	}
	return result
}

func base64EncodeByteMap(m map[string][]byte) map[string]interface{} {
	result := map[string]interface{}{}
	for k, v := range m {
		result[k] = base64.StdEncoding.EncodeToString(v)
	}
	return result
}

func base64DecodeStringMap(m map[string]interface{}) (map[string][]byte, error) {
	mm := map[string][]byte{}
	for k, v := range m {
		d, err := base64.StdEncoding.DecodeString(v.(string))
		if err != nil {
			return nil, err
		}
		mm[k] = []byte(d)
	}
	return mm, nil
}

func flattenResourceList(l api.ResourceList) map[string]string {
	m := make(map[string]string)
	for k, v := range l {
		m[string(k)] = v.String()
	}
	return m
}

func expandMapToResourceList(m map[string]interface{}) (*api.ResourceList, error) {
	out := make(api.ResourceList)
	for stringKey, origValue := range m {
		key := api.ResourceName(stringKey)
		var value resource.Quantity

		if v, ok := origValue.(int); ok {
			q := resource.NewQuantity(int64(v), resource.DecimalExponent)
			value = *q
		} else if v, ok := origValue.(string); ok {
			var err error
			value, err = resource.ParseQuantity(v)
			if err != nil {
				return &out, err
			}
		} else {
			return &out, fmt.Errorf("Unexpected value type: %#v", origValue)
		}

		out[key] = value
	}
	return &out, nil
}

func flattenPersistentVolumeAccessModes(in []api.PersistentVolumeAccessMode) *schema.Set {
	var out = make([]interface{}, len(in))
	for i, v := range in {
		out[i] = string(v)
	}
	return schema.NewSet(schema.HashString, out)
}

func expandPersistentVolumeAccessModes(s []interface{}) []api.PersistentVolumeAccessMode {
	out := make([]api.PersistentVolumeAccessMode, len(s))
	for i, v := range s {
		out[i] = api.PersistentVolumeAccessMode(v.(string))
	}
	return out
}

func flattenResourceQuotaSpec(in api.ResourceQuotaSpec) []interface{} {
	out := make([]interface{}, 1)

	m := make(map[string]interface{}, 0)
	m["hard"] = flattenResourceList(in.Hard)
	m["scopes"] = flattenResourceQuotaScopes(in.Scopes)

	if in.ScopeSelector != nil {
		m["scope_selector"] = flattenResourceQuotaScopeSelector(in.ScopeSelector)
	}

	out[0] = m
	return out
}

func expandResourceQuotaSpec(s []interface{}) (*api.ResourceQuotaSpec, error) {
	out := &api.ResourceQuotaSpec{}
	if len(s) < 1 {
		return out, nil
	}
	m := s[0].(map[string]interface{})

	if v, ok := m["hard"]; ok {
		list, err := expandMapToResourceList(v.(map[string]interface{}))
		if err != nil {
			return out, err
		}
		out.Hard = *list
	}

	if v, ok := m["scopes"]; ok {
		out.Scopes = expandResourceQuotaScopes(v.(*schema.Set).List())
	}

	if v, ok := m["scope_selector"]; ok {
		out.ScopeSelector = expandResourceQuotaScopeSelector(v.([]interface{}))
	}

	return out, nil
}

func flattenResourceQuotaScopes(in []api.ResourceQuotaScope) *schema.Set {
	out := make([]string, len(in))
	for i, scope := range in {
		out[i] = string(scope)
	}
	return newStringSet(schema.HashString, out)
}

func expandResourceQuotaScopes(s []interface{}) []api.ResourceQuotaScope {
	out := make([]api.ResourceQuotaScope, len(s))
	for i, scope := range s {
		out[i] = api.ResourceQuotaScope(scope.(string))
	}
	return out
}

func expandResourceQuotaScopeSelector(s []interface{}) *api.ScopeSelector {
	if len(s) < 1 {
		return nil
	}
	m := s[0].(map[string]interface{})

	att := &api.ScopeSelector{}

	if v, ok := m["match_expression"].([]interface{}); ok {
		att.MatchExpressions = expandResourceQuotaScopeSelectorMatchExpressions(v)
	}

	return att
}

func expandResourceQuotaScopeSelectorMatchExpressions(s []interface{}) []api.ScopedResourceSelectorRequirement {
	out := make([]api.ScopedResourceSelectorRequirement, len(s))

	for i, raw := range s {
		matchExp := raw.(map[string]interface{})

		if v, ok := matchExp["scope_name"].(string); ok {
			out[i].ScopeName = api.ResourceQuotaScope(v)
		}

		if v, ok := matchExp["operator"].(string); ok {
			out[i].Operator = api.ScopeSelectorOperator(v)
		}

		if v, ok := matchExp["values"].(*schema.Set); ok && v.Len() > 0 {
			out[i].Values = sliceOfString(v.List())
		}
	}
	return out
}

func flattenResourceQuotaScopeSelector(in *api.ScopeSelector) []interface{} {
	out := make([]interface{}, 1)

	m := make(map[string]interface{}, 0)
	m["match_expression"] = flattenResourceQuotaScopeSelectorMatchExpressions(in.MatchExpressions)

	out[0] = m
	return out
}

func flattenResourceQuotaScopeSelectorMatchExpressions(in []api.ScopedResourceSelectorRequirement) []interface{} {
	if len(in) == 0 {
		return []interface{}{}
	}
	out := make([]interface{}, len(in))

	for i, l := range in {
		m := make(map[string]interface{}, 0)
		m["operator"] = string(l.Operator)
		m["scope_name"] = string(l.ScopeName)

		if l.Values != nil && len(l.Values) > 0 {
			m["values"] = newStringSet(schema.HashString, l.Values)
		}

		out[i] = m
	}
	return out
}

func newStringSet(f schema.SchemaSetFunc, in []string) *schema.Set {
	var out = make([]interface{}, len(in))
	for i, v := range in {
		out[i] = v
	}
	return schema.NewSet(f, out)
}
func newInt64Set(f schema.SchemaSetFunc, in []int64) *schema.Set {
	var out = make([]interface{}, len(in))
	for i, v := range in {
		out[i] = int(v)
	}
	return schema.NewSet(f, out)
}

func resourceListEquals(x, y api.ResourceList) bool {
	for k, v := range x {
		yValue, ok := y[k]
		if !ok {
			return false
		}
		if v.Cmp(yValue) != 0 {
			return false
		}
	}
	for k, v := range y {
		xValue, ok := x[k]
		if !ok {
			return false
		}
		if v.Cmp(xValue) != 0 {
			return false
		}
	}
	return true
}

func expandLimitRangeSpec(s []interface{}, isNew bool) (*api.LimitRangeSpec, error) {
	out := &api.LimitRangeSpec{}
	if len(s) < 1 || s[0] == nil {
		return out, nil
	}
	m := s[0].(map[string]interface{})

	if limits, ok := m["limit"].([]interface{}); ok {
		newLimits := make([]api.LimitRangeItem, len(limits))

		for i, l := range limits {
			lrItem := api.LimitRangeItem{}
			limit := l.(map[string]interface{})

			if v, ok := limit["type"]; ok {
				lrItem.Type = api.LimitType(v.(string))
			}

			// defaultRequest is forbidden for Pod limits, even though it's set & returned by API
			// this is how we avoid sending it back
			if v, ok := limit["default_request"]; ok {
				drm := v.(map[string]interface{})
				if lrItem.Type == api.LimitTypePod && len(drm) > 0 {
					if isNew {
						return out, fmt.Errorf("limit.%d.default_request cannot be set for Pod limit", i)
					}
				} else {
					el, err := expandMapToResourceList(drm)
					if err != nil {
						return out, err
					}
					lrItem.DefaultRequest = *el
				}
			}

			if v, ok := limit["default"]; ok {
				el, err := expandMapToResourceList(v.(map[string]interface{}))
				if err != nil {
					return out, err
				}
				lrItem.Default = *el
			}
			if v, ok := limit["max"]; ok {
				el, err := expandMapToResourceList(v.(map[string]interface{}))
				if err != nil {
					return out, err
				}
				lrItem.Max = *el
			}
			if v, ok := limit["max_limit_request_ratio"]; ok {
				el, err := expandMapToResourceList(v.(map[string]interface{}))
				if err != nil {
					return out, err
				}
				lrItem.MaxLimitRequestRatio = *el
			}
			if v, ok := limit["min"]; ok {
				el, err := expandMapToResourceList(v.(map[string]interface{}))
				if err != nil {
					return out, err
				}
				lrItem.Min = *el
			}

			newLimits[i] = lrItem
		}

		out.Limits = newLimits
	}

	return out, nil
}

func flattenLimitRangeSpec(in api.LimitRangeSpec) []interface{} {
	if len(in.Limits) == 0 {
		return []interface{}{}
	}

	out := make([]interface{}, 1)
	limits := make([]interface{}, len(in.Limits))

	for i, l := range in.Limits {
		m := make(map[string]interface{}, 0)
		m["default"] = flattenResourceList(l.Default)
		m["default_request"] = flattenResourceList(l.DefaultRequest)
		m["max"] = flattenResourceList(l.Max)
		m["max_limit_request_ratio"] = flattenResourceList(l.MaxLimitRequestRatio)
		m["min"] = flattenResourceList(l.Min)
		m["type"] = string(l.Type)

		limits[i] = m
	}
	out[0] = map[string]interface{}{
		"limit": limits,
	}
	return out
}

func schemaSetToStringArray(set *schema.Set) []string {
	array := make([]string, 0, set.Len())
	for _, elem := range set.List() {
		e := elem.(string)
		array = append(array, e)
	}
	return array
}

func schemaSetToInt64Array(set *schema.Set) []int64 {
	array := make([]int64, 0, set.Len())
	for _, elem := range set.List() {
		e := elem.(int)
		array = append(array, int64(e))
	}
	return array
}

func flattenLocalObjectReferenceArray(in []api.LocalObjectReference) []interface{} {
	att := []interface{}{}
	for _, v := range in {
		m := map[string]interface{}{
			"name": v.Name,
		}
		att = append(att, m)
	}
	return att
}

func expandLocalObjectReferenceArray(in []interface{}) []api.LocalObjectReference {
	att := []api.LocalObjectReference{}
	if len(in) < 1 {
		return att
	}
	att = make([]api.LocalObjectReference, len(in))
	for i, c := range in {
		p := c.(map[string]interface{})
		if name, ok := p["name"]; ok {
			att[i].Name = name.(string)
		}
	}
	return att
}

func flattenServiceAccountSecrets(in []api.ObjectReference, defaultSecretName string) []interface{} {
	att := make([]interface{}, 0)
	for _, v := range in {
		if v.Name == defaultSecretName {
			continue
		}
		m := map[string]interface{}{}
		if v.Name != "" {
			m["name"] = v.Name
		}
		att = append(att, m)
	}
	return att
}

func expandServiceAccountSecrets(in []interface{}, defaultSecretName string) []api.ObjectReference {
	att := make([]api.ObjectReference, 0)

	for _, c := range in {
		p := c.(map[string]interface{})
		if name, ok := p["name"]; ok {
			att = append(att, api.ObjectReference{Name: name.(string)})
		}
	}
	if defaultSecretName != "" {
		att = append(att, api.ObjectReference{Name: defaultSecretName})
	}

	return att
}

func flattenNodeSelectorRequirementList(in []api.NodeSelectorRequirement) []map[string]interface{} {
	att := make([]map[string]interface{}, len(in))
	for i, v := range in {
		m := map[string]interface{}{}
		m["key"] = v.Key
		m["values"] = newStringSet(schema.HashString, v.Values)
		m["operator"] = string(v.Operator)
		att[i] = m
	}
	return att
}

func expandNodeSelectorRequirementList(in []interface{}) []api.NodeSelectorRequirement {
	att := []api.NodeSelectorRequirement{}
	if len(in) < 1 {
		return att
	}
	att = make([]api.NodeSelectorRequirement, len(in))
	for i, c := range in {
		p := c.(map[string]interface{})
		att[i].Key = p["key"].(string)
		att[i].Operator = api.NodeSelectorOperator(p["operator"].(string))
		att[i].Values = expandStringSlice(p["values"].(*schema.Set).List())
	}
	return att
}

func flattenNodeSelectorTerm(in api.NodeSelectorTerm) []interface{} {
	att := make(map[string]interface{})
	if len(in.MatchExpressions) > 0 {
		att["match_expressions"] = flattenNodeSelectorRequirementList(in.MatchExpressions)
	}
	if len(in.MatchFields) > 0 {
		att["match_fields"] = flattenNodeSelectorRequirementList(in.MatchFields)
	}
	return []interface{}{att}
}

func expandNodeSelectorTerm(l []interface{}) *api.NodeSelectorTerm {
	if len(l) == 0 || l[0] == nil {
		return &api.NodeSelectorTerm{}
	}
	in := l[0].(map[string]interface{})
	obj := api.NodeSelectorTerm{}
	if v, ok := in["match_expressions"].([]interface{}); ok && len(v) > 0 {
		obj.MatchExpressions = expandNodeSelectorRequirementList(v)
	}
	if v, ok := in["match_fields"].([]interface{}); ok && len(v) > 0 {
		obj.MatchFields = expandNodeSelectorRequirementList(v)
	}
	return &obj
}

func flattenNodeSelectorTerms(in []api.NodeSelectorTerm) []interface{} {
	att := make([]interface{}, len(in))
	for i, n := range in {
		att[i] = flattenNodeSelectorTerm(n)[0]
	}
	return att
}

func expandNodeSelectorTerms(l []interface{}) []api.NodeSelectorTerm {
	if len(l) == 0 || l[0] == nil {
		return []api.NodeSelectorTerm{}
	}
	obj := make([]api.NodeSelectorTerm, len(l))
	for i, n := range l {
		obj[i] = *expandNodeSelectorTerm([]interface{}{n})
	}
	return obj
}

func flattenPersistentVolumeMountOptions(in []string) *schema.Set {
	var out = make([]interface{}, len(in))
	for i, v := range in {
		out[i] = string(v)
	}
	return schema.NewSet(schema.HashString, out)
}
