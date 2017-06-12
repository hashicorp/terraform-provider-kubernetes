package kubernetes

import (
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	api "k8s.io/kubernetes/pkg/apis/rbac/v1"
)

func expandRBACRoleRef(in interface{}) api.RoleRef {
	if in == nil {
		return api.RoleRef{}
	}
	ref := api.RoleRef{}
	m := in.(map[string]interface{})
	if v, ok := m["api_group"]; ok {
		ref.APIGroup = v.(string)
	}
	if v, ok := m["kind"].(string); ok {
		ref.Kind = string(v)
	}
	if v, ok := m["name"]; ok {
		ref.Name = v.(string)
	}

	return ref
}

func expandRBACSubjects(in []interface{}) []api.Subject {
	if len(in) == 0 || in[0] == nil {
		return []api.Subject{}
	}
	subjects := []api.Subject{}
	for i := range in {
		subject := api.Subject{}
		m := in[i].(map[string]interface{})
		if v, ok := m["api_group"]; ok {
			subject.APIGroup = v.(string)
		}
		if v, ok := m["kind"].(string); ok {
			subject.Kind = string(v)
		}
		if v, ok := m["name"]; ok {
			subject.Name = v.(string)
		}
		if v, ok := m["namespace"]; ok {
			subject.Namespace = v.(string)
		}
		subjects = append(subjects, subject)
	}
	return subjects
}

func flattenRBACRoleRef(in api.RoleRef) interface{} {
	att := make(map[string]interface{})

	if in.APIGroup != "" {
		att["api_group"] = in.APIGroup
	}
	att["kind"] = in.Kind
	att["name"] = in.Name
	return att
}

func flattenRBACSubjects(in []api.Subject) []interface{} {
	att := make([]interface{}, len(in), len(in))
	for _, n := range in {
		m := make(map[string]interface{})
		if n.APIGroup != "" {
			m["api_group"] = n.APIGroup
		}
		m["kind"] = n.Kind
		m["name"] = n.Name
		if n.Namespace != "" {
			m["namespace"] = n.Namespace
		}
	}
	return att
}

// Patch Ops
func patchRbacSubject(d *schema.ResourceData) PatchOperations {
	ops := make([]PatchOperation, 0, 0)

	if d.HasChange("subjects") {
		subjects := expandRBACSubjects(d.Get("subjects").([]interface{}))
		for i, v := range subjects {
			ops = append(ops, &ReplaceOperation{
				Path:  "/subjects/" + strconv.Itoa(i),
				Value: v,
			})
		}
	}
	return ops
}
