package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	networking "k8s.io/api/networking/v1"
)

// Flatteners

func flattenIngressV1Rule(in []networking.IngressRule) []interface{} {
	att := make([]interface{}, len(in), len(in))
	for i, r := range in {
		m := make(map[string]interface{})

		m["host"] = r.Host
		m["http"] = flattenIngressV1RuleHttp(r.HTTP)
		att[i] = m
	}
	return att
}

func flattenIngressV1RuleHttp(in *networking.HTTPIngressRuleValue) []interface{} {
	if in == nil {
		return []interface{}{}
	}
	pathAtts := make([]interface{}, len(in.Paths), len(in.Paths))
	for i, p := range in.Paths {
		path := map[string]interface{}{
			"path":      p.Path,
			"path_type": p.PathType,
			"backend":   flattenIngressV1Backend(&p.Backend),
		}
		pathAtts[i] = path
	}

	httpAtt := map[string]interface{}{
		"path": pathAtts,
	}

	return []interface{}{httpAtt}
}

func flattenIngressV1Backend(in *networking.IngressBackend) []interface{} {
	p := make(map[string]interface{})

	if in.Service.Port.Number != 0 {
		p["number"] = in.Service.Port.Number
	}

	if in.Service.Port.Name != "" {
		p["name"] = in.Service.Port.Name
	}

	s := make(map[string]interface{})
	s["port"] = []interface{}{p}
	s["name"] = in.Service.Name

	m := make(map[string]interface{})
	m["service"] = []interface{}{s}

	return []interface{}{m}
}

func flattenIngressV1Spec(in networking.IngressSpec) []interface{} {
	att := make(map[string]interface{})

	if in.IngressClassName != nil {
		att["ingress_class_name"] = in.IngressClassName
	}

	if in.DefaultBackend != nil {
		att["default_backend"] = flattenIngressV1Backend(in.DefaultBackend)
	}

	if len(in.Rules) > 0 {
		att["rule"] = flattenIngressV1Rule(in.Rules)
	}

	if len(in.TLS) > 0 {
		att["tls"] = flattenIngressV1TLS(in.TLS)
	}

	return []interface{}{att}
}

func flattenIngressV1TLS(in []networking.IngressTLS) []interface{} {
	att := make([]interface{}, len(in), len(in))

	for i, v := range in {
		m := make(map[string]interface{})
		m["hosts"] = v.Hosts
		m["secret_name"] = v.SecretName

		att[i] = m
	}

	return att
}

// Expanders

func expandIngressV1Rule(l []interface{}) []networking.IngressRule {
	if len(l) == 0 || l[0] == nil {
		return []networking.IngressRule{}
	}
	obj := make([]networking.IngressRule, len(l), len(l))
	for i, n := range l {
		cfg := n.(map[string]interface{})

		var paths []networking.HTTPIngressPath

		if httpCfg, ok := cfg["http"]; ok {
			httpList := httpCfg.([]interface{})
			for _, h := range httpList {
				http := h.(map[string]interface{})
				if v, ok := http["path"]; ok {
					pathList := v.([]interface{})
					paths = make([]networking.HTTPIngressPath, len(pathList), len(pathList))
					for i, path := range pathList {
						p := path.(map[string]interface{})
						t := networking.PathType(p["path_type"].(string))
						hip := networking.HTTPIngressPath{
							Path:     p["path"].(string),
							PathType: &t,
							Backend:  *expandIngressV1Backend(p["backend"].([]interface{})),
						}
						paths[i] = hip
					}
				}
			}
		}

		obj[i] = networking.IngressRule{
			Host: cfg["host"].(string),
			IngressRuleValue: networking.IngressRuleValue{
				HTTP: &networking.HTTPIngressRuleValue{
					Paths: paths,
				},
			},
		}
	}
	return obj
}

func expandIngressV1Spec(l []interface{}) networking.IngressSpec {
	if len(l) == 0 || l[0] == nil {
		return networking.IngressSpec{}
	}
	in := l[0].(map[string]interface{})
	obj := networking.IngressSpec{}

	if v, ok := in["ingress_class_name"].(string); ok && len(v) > 0 {
		obj.IngressClassName = &v
	}

	if v, ok := in["default_backend"].([]interface{}); ok && len(v) > 0 {
		obj.DefaultBackend = expandIngressV1Backend(v)
	}

	if v, ok := in["rule"].([]interface{}); ok && len(v) > 0 {
		obj.Rules = expandIngressV1Rule(v)
	}

	if v, ok := in["tls"].([]interface{}); ok && len(v) > 0 {
		obj.TLS = expandIngressV1TLS(v)
	}

	return obj
}

func expandIngressV1Backend(l []interface{}) *networking.IngressBackend {
	if len(l) == 0 || l[0] == nil {
		return &networking.IngressBackend{}
	}

	in := l[0].(map[string]interface{})
	l, ok := in["service"].([]interface{})
	if !ok || len(l) == 0 || l[0] == nil {
		return &networking.IngressBackend{}
	}

	obj := &networking.IngressBackend{}
	obj.Service = &networking.IngressServiceBackend{}
	service := l[0].(map[string]interface{})
	if v, ok := service["name"].(string); ok {
		obj.Service.Name = v
	}

	l, ok = service["port"].([]interface{})
	if !ok || len(l) == 0 || l[0] == nil {
		return obj
	}

	servicePort := l[0].(map[string]interface{})
	if v, ok := servicePort["number"].(int); ok {
		obj.Service.Port.Number = int32(v)
	}

	if v, ok := servicePort["name"].(string); ok {
		obj.Service.Port.Name = v
	}
	return obj
}

func expandIngressV1TLS(l []interface{}) []networking.IngressTLS {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	tlsList := make([]networking.IngressTLS, len(l), len(l))
	for i, t := range l {
		in := t.(map[string]interface{})
		obj := networking.IngressTLS{}

		if v, ok := in["hosts"]; ok {
			obj.Hosts = expandStringSlice(v.([]interface{}))
		}

		if v, ok := in["secret_name"].(string); ok {
			obj.SecretName = v
		}
		tlsList[i] = obj
	}

	return tlsList
}

// Patch Ops

func patchIngressV1Spec(keyPrefix, pathPrefix string, d *schema.ResourceData) PatchOperations {
	ops := make([]PatchOperation, 0, 0)
	if d.HasChange(keyPrefix + "backend") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "backend",
			Value: expandIngressV1Backend(d.Get(keyPrefix + "backend").([]interface{})),
		})
	}

	if d.HasChange(keyPrefix + "rule") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "rules",
			Value: expandIngressV1Rule(d.Get(keyPrefix + "rule").([]interface{})),
		})
	}

	if d.HasChange(keyPrefix + "tls") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "tls",
			Value: expandIngressV1TLS(d.Get(keyPrefix + "tls").([]interface{})),
		})
	}

	return ops
}
