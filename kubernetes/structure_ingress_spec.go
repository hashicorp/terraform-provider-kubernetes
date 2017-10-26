package kubernetes

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

// Flatteners
func flattenIngressRule(in []v1beta1.IngressRule) []interface{} {
	att := make([]interface{}, len(in), len(in))
	for i, n := range in {
		// rulePrefix := fmt.Sprintf("rule.%d.")
		m := make(map[string]interface{})

		m["host"] = n.Host

		for i, p := range n.HTTP.Paths {
			pathPrefix := fmt.Sprintf("http.0.path.%d.", i)
			m[pathPrefix+"path"] = p.Path
			m[pathPrefix+"backend.0"] = flattenIngressBackend(p.Backend)
		}

		att[i] = m
	}
	return att
}

func flattenIngressBackend(in v1beta1.IngressBackend) map[string]interface{} {
	m := make(map[string]interface{})

	m["service_name"] = in.ServiceName
	m["service_port"] = in.ServicePort

	return m
}

func flattenIngressSpec(in v1beta1.IngressSpec) []interface{} {
	att := make(map[string]interface{})
	if len(in.Rules) > 0 {
		att["rule"] = flattenIngressRule(in.Rules)
	}

	return []interface{}{att}
}

// Expanders

func expandIngressRule(l []interface{}) []v1beta1.IngressRule {
	if len(l) == 0 || l[0] == nil {
		return []v1beta1.IngressRule{}
	}
	obj := make([]v1beta1.IngressRule, len(l), len(l))
	for i, n := range l {
		cfg := n.(map[string]interface{})
		obj[i] = v1beta1.IngressRule{
			Host: cfg["host"].(string),
		}
		// if v, ok := cfg["name"].(string); ok {
		// 	obj[i].Name = v
		// }

	}
	return obj
}

func expandIngressSpec(l []interface{}) v1beta1.IngressSpec {
	if len(l) == 0 || l[0] == nil {
		return v1beta1.IngressSpec{}
	}
	in := l[0].(map[string]interface{})
	obj := v1beta1.IngressSpec{}

	if v, ok := in["backend"].([]interface{}); ok && len(v) > 0 {
		obj.Backend = expandIngressBackend(v)
	}
	// if v, ok := in["selector"].(map[string]interface{}); ok && len(v) > 0 {
	// 	obj.Selector = expandStringMap(v)
	// }

	return obj
}

func expandIngressBackend(l []interface{}) *v1beta1.IngressBackend {
	if len(l) == 0 || l[0] == nil {
		return &v1beta1.IngressBackend{}
	}
	in := l[0].(map[string]interface{})
	obj := &v1beta1.IngressBackend{}

	if v, ok := in["service_name"].(string); ok {
		obj.ServiceName = v
	}

	if v, ok := in["service_port"].(int); ok {
		obj.ServicePort = expandIntOrString(v)
	}

	return obj
}

// Patch Ops

func patchIngressSpec(keyPrefix, pathPrefix string, d *schema.ResourceData) PatchOperations {
	ops := make([]PatchOperation, 0, 0)
	if d.HasChange(keyPrefix + "backend") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "backend",
			Value: d.Get(keyPrefix + "backend").(map[string]interface{}),
		})
	}

	return ops
}
