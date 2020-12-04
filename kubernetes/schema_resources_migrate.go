package kubernetes

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func patchTemplatePodSpecWithResourcesFieldV0(m map[string]*schema.Schema, templateFieldName string) map[string]*schema.Schema {
	spec := m["spec"].Elem.(*schema.Resource)
	template := spec.Schema[templateFieldName].Elem.(*schema.Resource)
	podSpec := template.Schema["spec"].Elem.(*schema.Resource)

	initContainer := podSpec.Schema["init_container"].Elem.(*schema.Resource)
	initContainer.Schema["resources"].Elem = &schema.Resource{
		Schema: resourcesFieldV0(),
	}

	container := podSpec.Schema["container"].Elem.(*schema.Resource)
	container.Schema["resources"].Elem = &schema.Resource{
		Schema: resourcesFieldV0(),
	}
	return m
}

func patchPodSpecWithResourcesFieldV0(m map[string]*schema.Schema, templateFieldName string) map[string]*schema.Schema {
	spec := m["spec"].Elem.(*schema.Resource)

	initContainer := spec.Schema["init_container"].Elem.(*schema.Resource)
	initContainer.Schema["resources"].Elem = &schema.Resource{
		Schema: resourcesFieldV0(),
	}

	container := spec.Schema["container"].Elem.(*schema.Resource)
	container.Schema["resources"].Elem = &schema.Resource{
		Schema: resourcesFieldV0(),
	}
	return m
}

func upgradeTemplatePodSpecWithResourcesFieldV0(ctx context.Context, rawState map[string]interface{}, meta interface{}, templateFieldName string) (map[string]interface{}, error) {
	if s, ok := rawState["spec"].([]interface{}); ok && len(s) > 0 {
		spec := s[0].(map[string]interface{})
		if t, ok := spec[templateFieldName].([]interface{}); ok && len(t) > 0 {
			template := t[0].(map[string]interface{})
			if ps, ok := template["spec"].([]interface{}); ok && len(ps) > 0 {
				podSpec := ps[0].(map[string]interface{})
				upgradedPodSpec := upgradePodSpecWithResourcesFieldV0(podSpec)
				template["spec"] = []interface{}{upgradedPodSpec}
			}
		}
	}
	return rawState, nil
}

func upgradePodSpecWithResourcesFieldV0(rawState map[string]interface{}) map[string]interface{} {
	if initContainers, ok := rawState["init_container"].([]interface{}); ok && len(initContainers) > 0 {
		for _, c := range initContainers {
			initContainer := c.(map[string]interface{})
			if r, ok := initContainer["resources"].([]interface{}); ok && len(r) > 0 {
				resources := r[0].(map[string]interface{})
				if req, ok := resources["requests"].([]interface{}); ok && len(req) > 0 {
					requests := req[0].(map[string]interface{})
					resources["requests"] = requests
				}
				if lim, ok := resources["limits"].([]interface{}); ok && len(lim) > 0 {
					limits := lim[0].(map[string]interface{})
					resources["limits"] = limits
				}
			}
		}
	}
	if containers, ok := rawState["container"].([]interface{}); ok && len(containers) > 0 {
		for _, c := range containers {
			container := c.(map[string]interface{})
			if r, ok := container["resources"].([]interface{}); ok && len(r) > 0 {
				resources := r[0].(map[string]interface{})
				if req, ok := resources["requests"].([]interface{}); ok && len(req) > 0 {
					requests := req[0].(map[string]interface{})
					resources["requests"] = requests
				}
				if lim, ok := resources["limits"].([]interface{}); ok && len(lim) > 0 {
					limits := lim[0].(map[string]interface{})
					resources["limits"] = limits
				}
			}
		}
	}
	return rawState
}
