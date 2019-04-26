package kubernetes

import (
	api "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

func expandCustomResourceDefinitionSpec(l []interface{}) (*api.CustomResourceDefinitionSpec, error) {
	obj := &api.CustomResourceDefinitionSpec{}

	if len(l) == 0 || l[0] == nil {
		return obj, nil
	}

	in := l[0].(map[string]interface{})

	obj.Group = in["group"].(string)
	obj.Version = in["version"].(string)
	if v, ok := in["names"].([]interface{}); ok && len(v) > 0 {
		obj.Names = expandCustomResourceDefinitionNames(v)
	}
	obj.Scope = api.ResourceScope(in["scope"].(string))

	// Intentionally skipping "schema" field; it contains a JSONSchema field that forces
	// a recursive schema, but https://github.com/hashicorp/terraform/issues/18616 says
	// Terraform does not support recursive schemas

	// if v, ok := in["subresource"].([]interface{}); ok && len(v) > 0 {
	// 	obj.Subresources = expandCustomResourceSubresources(v)
	// }

	if v, ok := in["versions"].([]interface{}); ok && len(v) > 0 {
		obj.Versions = expandCustomResourceDefinitionVersions(v)
	}

	// if v, ok := in["additional_printer_column"].([]interface{}); ok && len(v) > 0 {
	// 	obj.AdditionalPrinterColumns = expandCustomResourceColumnDefinition(v)
	// }

	// if v, ok := in["conversion"].([]interface{}); ok && len(v) > 0 {
	// 	obj.Conversion = expandCustomResourceConversion(v)
	// }

	return obj, nil
}

func expandCustomResourceDefinitionNames(l []interface{}) api.CustomResourceDefinitionNames {
	obj := api.CustomResourceDefinitionNames{}

	if len(l) == 0 || l[0] == nil {
		return obj
	}

	in := l[0].(map[string]interface{})

	obj.Plural = in["plural"].(string)

	if v, ok := in["singular"].(string); ok {
		obj.Singular = v
	}

	if v, ok := in["short_names"].([]interface{}); ok && len(v) > 0 {
		shortNames := make([]string, len(v))
		for i, c := range v {
			shortNames[i] = c.(string)
		}
		obj.ShortNames = shortNames
	}

	obj.Kind = in["kind"].(string)

	if v, ok := in["list_kind"].(string); ok {
		obj.ListKind = v
	}

	if v, ok := in["categories"].([]interface{}); ok && len(v) > 0 {
		categories := make([]string, len(v))
		for i, c := range v {
			categories[i] = c.(string)
		}
		obj.Categories = categories
	}

	return obj
}

// func expandCustomResourceSubresources(l []interface{}) *api.CustomResourceSubresources {
// 	obj := &api.CustomResourceSubresources{}

// 	if len(l) == 0 || l[0] == nil {
// 		return obj
// 	}

// 	in := l[0].(map[string]interface{})

// 	if v, ok := in["scale"].([]interface{}); ok && len(v) > 0 {
// 		obj.Scale = expandCustomResourceSubresourceScale(v)
// 	}

// 	return obj
// }

// func expandCustomResourceSubresourceScale(l []interface{}) *api.CustomResourceSubresourceScale {
// 	obj := &api.CustomResourceSubresourceScale{}

// 	if len(l) == 0 || l[0] == nil {
// 		return obj
// 	}

// 	in := l[0].(map[string]interface{})

// 	obj.SpecReplicasPath = in["spec_replicas_path"].(string)
// 	obj.StatusReplicasPath = in["status_replicas_path"].(string)
// 	if v, ok := in["label_selector_path"].(string); ok {
// 		obj.LabelSelectorPath = ptrToString(v)
// 	}

// 	return obj
// }

func expandCustomResourceDefinitionVersions(l []interface{}) []api.CustomResourceDefinitionVersion {
	if len(l) == 0 {
		return []api.CustomResourceDefinitionVersion{}
	}
	obj := make([]api.CustomResourceDefinitionVersion, len(l))
	for i, c := range l {
		m := c.(map[string]interface{})
		obj[i].Name = m["name"].(string)
		obj[i].Served = m["served"].(bool)
		obj[i].Storage = m["storage"].(bool)
		// Skip "schema" field
		// if v, ok := m["subresources"].([]interface{}); ok && len(v) > 0 {
		// 	obj[i].Subresources = expandCustomResourceSubresources(v)
		// }
		// if v, ok := m["additional_printer_column"].([]interface{}); ok && len(v) > 0 {
		// 	obj[i].AdditionalPrinterColumns = expandCustomResourceColumnDefinition(v)
		// }
	}
	return obj
}

// func expandCustomResourceColumnDefinition(l []interface{}) []api.CustomResourceColumnDefinition {
// 	if len(l) == 0 {
// 		return []api.CustomResourceColumnDefinition{}
// 	}
// 	obj := make([]api.CustomResourceColumnDefinition, len(l))
// 	for i, c := range l {
// 		m := c.(map[string]interface{})
// 		obj[i].Name = m["name"].(string)
// 		obj[i].Type = m["type"].(string)
// 		if v, ok := m["format"]; ok {
// 			obj[i].Format = v.(string)
// 		}
// 		if v, ok := m["description"]; ok {
// 			obj[i].Description = v.(string)
// 		}
// 		if v, ok := m["priority"]; ok {
// 			obj[i].Priority = int32(v.(int))
// 		}
// 		obj[i].JSONPath = m["json_path"].(string)
// 	}
// 	return obj
// }

// func expandCustomResourceConversion(l []interface{}) *api.CustomResourceConversion {
// 	obj := &api.CustomResourceConversion{}

// 	if len(l) == 0 || l[0] == nil {
// 		return obj
// 	}

// 	in := l[0].(map[string]interface{})
// 	obj.Strategy = api.ConversionStrategyType(in["strategy"].(string))
// 	if v, ok := in["webhook_client_config"].([]interface{}); ok && len(v) > 0 {
// 		obj.WebhookClientConfig = expandWebhookClientConfig(v)
// 	}

// 	// if v, ok := in["conversion_review_versions"].([]interface{}); ok && len(v) > 0 {
// 	// 	conversionReviewVersions := make([]string, len(v))
// 	// 	for i, c := range v {
// 	// 		conversionReviewVersions[i] = c.(string)
// 	// 	}
// 	// 	obj.ConversionReviewVersions = conversionReviewVersions
// 	// }
// 	return obj
// }

// func expandWebhookClientConfig(l []interface{}) *api.WebhookClientConfig {
// 	obj := &api.WebhookClientConfig{}

// 	if len(l) == 0 || l[0] == nil {
// 		return obj
// 	}

// 	in := l[0].(map[string]interface{})

// 	if v, ok := in["url"].(string); ok {
// 		obj.URL = ptrToString(v)
// 	}
// 	if v, ok := in["service"].([]interface{}); ok && len(v) > 0 {
// 		obj.Service = expandServiceReference(v)
// 	}
// 	if v, ok := in["ca_bundle"].(string); ok {
// 		obj.CABundle = bytes.NewBufferString(v).Bytes()
// 	}
// 	return obj
// }

func expandServiceReference(l []interface{}) *api.ServiceReference {
	obj := &api.ServiceReference{}

	if len(l) == 0 || l[0] == nil {
		return obj
	}

	in := l[0].(map[string]interface{})

	obj.Namespace = in["namespace"].(string)
	obj.Name = in["name"].(string)

	if v, ok := in["path"].(string); ok {
		obj.Path = ptrToString(v)
	}
	return obj
}

func flattenCustomResourceDefinitionSpec(in api.CustomResourceDefinitionSpec) []interface{} {
	att := make(map[string]interface{})
	att["group"] = in.Group

	if in.Version != "" {
		att["version"] = in.Version
	}

	att["names"] = flattenCustomResourceDefinitionNames(in.Names)
	att["scope"] = string(in.Scope)
	// Skipping Validation

	// if in.Subresources != nil {
	// 	att["subresources"] = flattenCustomResourceSubresources(*in.Subresources)
	// }

	att["versions"] = flattenCustomResourceDefinitionVersions(in.Versions)
	// att["additional_printer_column"] = flattenCustomResourceColumnDefinition(in.AdditionalPrinterColumns)
	// if in.Conversion != nil {
	// 	att["conversion"] = flattenCustomResourceConversion(*in.Conversion)
	// }

	return []interface{}{att}
}

func flattenCustomResourceDefinitionNames(in api.CustomResourceDefinitionNames) []interface{} {
	att := make(map[string]interface{})
	att["plural"] = in.Plural
	att["singular"] = in.Singular

	shortNames := make([]string, len(in.ShortNames))
	for i, v := range in.ShortNames {
		shortNames[i] = v
	}
	att["short_names"] = shortNames
	att["kind"] = in.Kind
	att["list_kind"] = in.ListKind
	categories := make([]string, len(in.Categories))
	for i, v := range in.Categories {
		categories[i] = v
	}
	att["categories"] = categories

	return []interface{}{att}
}

// func flattenCustomResourceSubresources(in api.CustomResourceSubresources) []interface{} {
// 	att := make(map[string]interface{})
// 	if in.Scale != nil {
// 		m := make(map[string]interface{})
// 		m["spec_replicas_path"] = in.Scale.SpecReplicasPath
// 		m["status_replicas_path"] = in.Scale.StatusReplicasPath
// 		if in.Scale.LabelSelectorPath != nil {
// 			m["label_selector_path"] = in.Scale.LabelSelectorPath
// 		}
// 		att["scale"] = []interface{}{m}
// 	}
// 	// TODO(mbarrien): Handle in.Status; allow empty map?
// 	return []interface{}{att}
// }

func flattenCustomResourceDefinitionVersions(in []api.CustomResourceDefinitionVersion) []interface{} {
	att := make([]interface{}, len(in))
	for i, v := range in {
		m := map[string]interface{}{}
		m["name"] = v.Name
		m["served"] = v.Served
		m["storage"] = v.Storage
		// Skipping "schema"
		// if v.Subresources != nil {
		// 	m["subresources"] = flattenCustomResourceSubresources(*v.Subresources)
		// }
		// m["additional_printer_column"] = flattenCustomResourceColumnDefinition(v.AdditionalPrinterColumns)
		att[i] = m
	}
	return att
}

// func flattenCustomResourceColumnDefinition(in []api.CustomResourceColumnDefinition) []interface{} {
// 	att := make([]interface{}, len(in))
// 	for i, v := range in {
// 		m := map[string]interface{}{}
// 		m["name"] = v.Name
// 		m["type"] = v.Type
// 		m["format"] = v.Format
// 		m["description"] = v.Description
// 		m["priority"] = int(v.Priority)
// 		m["json_path"] = v.JSONPath
// 		att[i] = m
// 	}
// 	return att
// }

// func flattenCustomResourceConversion(in api.CustomResourceConversion) []interface{} {
// 	att := make(map[string]interface{})
// 	att["strategy"] = string(in.Strategy)
// 	if in.WebhookClientConfig != nil {
// 		att["webhook_client_config"] = flattenWebhookClientConfig(*in.WebhookClientConfig)
// 	}
// 	// conversionReviewVersions := make([]string, len(in.ConversionReviewVersions))
// 	// for i, v := range in.ConversionReviewVersions {
// 	// 	conversionReviewVersions[i] = v
// 	// }
// 	// att["conversion_review_versions"] = conversionReviewVersions
// 	return []interface{}{att}
// }

// func flattenWebhookClientConfig(in api.WebhookClientConfig) []interface{} {
// 	att := make(map[string]interface{})
// 	if in.URL != nil {
// 		att["url"] = *in.URL
// 	}
// 	if in.Service != nil {
// 		att["service"] = *in.Service
// 	}
// 	att["ca_bundle"] = string(in.CABundle)
// 	return []interface{}{att}
// }
