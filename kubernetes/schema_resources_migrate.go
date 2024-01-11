// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// NOTE these functions are for patching the schemas for resources with containers
// to use the old schema for the resources block so we can migrate it to the new format
// without having to duplicate the entire schema

func patchJobTemplatePodSpecWithResourcesFieldV0(m map[string]*schema.Schema) map[string]*schema.Schema {
	spec := m["spec"].Elem.(*schema.Resource)
	jobTemplate := spec.Schema["job_template"].Elem.(*schema.Resource)
	jobSpec := jobTemplate.Schema["spec"].Elem.(*schema.Resource)
	template := jobSpec.Schema["template"].Elem.(*schema.Resource)
	template.Schema = patchPodSpecWithResourcesFieldV0(template.Schema)
	return m
}

func patchTemplatePodSpecWithResourcesFieldV0(m map[string]*schema.Schema) map[string]*schema.Schema {
	spec := m["spec"].Elem.(*schema.Resource)
	template := spec.Schema["template"].Elem.(*schema.Resource)
	template.Schema = patchPodSpecWithResourcesFieldV0(template.Schema)
	return m
}

func patchPodSpecWithResourcesFieldV0(m map[string]*schema.Schema) map[string]*schema.Schema {
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

func upgradeJobTemplatePodSpecWithResourcesFieldV0(ctx context.Context, rawState map[string]interface{}, meta interface{}) map[string]interface{} {
	s, ok := rawState["spec"].([]interface{})
	if !ok || len(s) == 0 {
		return rawState
	}

	spec := s[0].(map[string]interface{})
	jt, ok := spec["job_template"].([]interface{})
	if !ok || len(jt) == 0 {
		return rawState
	}

	jobTemplate := jt[0].(map[string]interface{})
	js, ok := jobTemplate["spec"].([]interface{})
	if !ok || len(js) == 0 {
		return rawState
	}

	jobSpec := js[0].(map[string]interface{})
	t, ok := jobSpec["template"].([]interface{})
	if !ok || len(t) == 0 {
		return rawState
	}

	template := t[0].(map[string]interface{})
	ps, ok := template["spec"].([]interface{})
	if !ok || len(ps) == 0 {
		return rawState
	}

	podSpec := ps[0].(map[string]interface{})
	template["spec"] = []interface{}{upgradeContainers(podSpec)}

	return rawState
}

func upgradeTemplatePodSpecWithResourcesFieldV0(ctx context.Context, rawState map[string]interface{}, meta interface{}) map[string]interface{} {
	s, ok := rawState["spec"].([]interface{})
	if !ok || len(s) == 0 {
		return rawState
	}

	spec := s[0].(map[string]interface{})
	t, ok := spec["template"].([]interface{})
	if !ok || len(t) == 0 {
		return rawState
	}

	template := t[0].(map[string]interface{})
	ps, ok := template["spec"].([]interface{})

	if !ok || len(ps) == 0 {
		return rawState
	}

	podSpec := ps[0].(map[string]interface{})
	template["spec"] = []interface{}{upgradeContainers(podSpec)}

	return rawState
}

func upgradePodSpecWithResourcesFieldV0(ctx context.Context, rawState map[string]interface{}, meta interface{}) map[string]interface{} {
	s, ok := rawState["spec"].([]interface{})
	if !ok || len(s) == 0 {
		return rawState
	}

	spec := s[0].(map[string]interface{})
	rawState["spec"] = []interface{}{upgradeContainers(spec)}

	return rawState
}

func upgradeContainers(rawState map[string]interface{}) map[string]interface{} {
	if initContainers, ok := rawState["init_container"].([]interface{}); ok && len(initContainers) > 0 {
		for _, c := range initContainers {
			initContainer := c.(map[string]interface{})
			if r, ok := initContainer["resources"].([]interface{}); ok && len(r) > 0 {
				resources := r[0].(map[string]interface{})
				if req, ok := resources["requests"].([]interface{}); ok && len(req) > 0 {
					requests := req[0].(map[string]interface{})
					resources["requests"] = requests
				} else {
					resources["requests"] = map[string]interface{}{}
				}

				if lim, ok := resources["limits"].([]interface{}); ok && len(lim) > 0 {
					limits := lim[0].(map[string]interface{})
					resources["limits"] = limits
				} else {
					resources["limits"] = map[string]interface{}{}
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
				} else {
					resources["requests"] = map[string]interface{}{}
				}

				if lim, ok := resources["limits"].([]interface{}); ok && len(lim) > 0 {
					limits := lim[0].(map[string]interface{})
					resources["limits"] = limits
				} else {
					resources["limits"] = map[string]interface{}{}
				}
			}
		}
	}
	return rawState
}
