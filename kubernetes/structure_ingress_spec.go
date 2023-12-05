// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Flatteners

func flattenIngressRule(in []v1beta1.IngressRule) []interface{} {
	att := make([]interface{}, len(in))
	for i, r := range in {
		m := make(map[string]interface{})

		m["host"] = r.Host
		m["http"] = flattenIngressRuleHttp(r.HTTP)
		att[i] = m
	}
	return att
}

func flattenIngressRuleHttp(in *v1beta1.HTTPIngressRuleValue) []interface{} {
	if in == nil {
		return []interface{}{}
	}
	pathAtts := make([]interface{}, len(in.Paths))
	for i, p := range in.Paths {
		path := map[string]interface{}{
			"path":    p.Path,
			"backend": flattenIngressBackend(&p.Backend),
		}
		pathAtts[i] = path
	}

	httpAtt := map[string]interface{}{
		"path": pathAtts,
	}

	return []interface{}{httpAtt}
}

func flattenIngressBackend(in *v1beta1.IngressBackend) []interface{} {
	att := make([]interface{}, 1)

	m := make(map[string]interface{})
	m["service_name"] = in.ServiceName
	m["service_port"] = in.ServicePort.String()

	att[0] = m

	return att
}

func flattenIngressSpec(in v1beta1.IngressSpec) []interface{} {
	att := make(map[string]interface{})

	if in.IngressClassName != nil {
		att["ingress_class_name"] = in.IngressClassName
	}

	if in.Backend != nil {
		att["backend"] = flattenIngressBackend(in.Backend)
	}

	if len(in.Rules) > 0 {
		att["rule"] = flattenIngressRule(in.Rules)
	}

	if len(in.TLS) > 0 {
		att["tls"] = flattenIngressTLS(in.TLS)
	}

	return []interface{}{att}
}

func flattenIngressTLS(in []v1beta1.IngressTLS) []interface{} {
	att := make([]interface{}, len(in))

	for i, v := range in {
		m := make(map[string]interface{})
		m["hosts"] = v.Hosts
		m["secret_name"] = v.SecretName

		att[i] = m
	}

	return att
}

func flattenIngressStatus(in v1beta1.IngressLoadBalancerStatus) []interface{} {
	out := make([]interface{}, len(in.Ingress))
	for i, ingress := range in.Ingress {
		out[i] = map[string]interface{}{
			"ip":       ingress.IP,
			"hostname": ingress.Hostname,
		}
	}

	return []interface{}{
		map[string][]interface{}{
			"ingress": out,
		},
	}
}

// Expanders

func expandIngressRule(l []interface{}) []v1beta1.IngressRule {
	if len(l) == 0 || l[0] == nil {
		return []v1beta1.IngressRule{}
	}
	obj := make([]v1beta1.IngressRule, len(l))
	for i, n := range l {
		cfg := n.(map[string]interface{})

		var paths []v1beta1.HTTPIngressPath

		if httpCfg, ok := cfg["http"]; ok {
			httpList := httpCfg.([]interface{})
			for _, h := range httpList {
				http := h.(map[string]interface{})
				if v, ok := http["path"]; ok {
					pathList := v.([]interface{})
					paths = make([]v1beta1.HTTPIngressPath, len(pathList))
					for i, path := range pathList {
						p := path.(map[string]interface{})
						hip := v1beta1.HTTPIngressPath{
							Path:    p["path"].(string),
							Backend: *expandIngressBackend(p["backend"].([]interface{})),
						}
						paths[i] = hip
					}
				}
			}
		}

		obj[i] = v1beta1.IngressRule{
			Host: cfg["host"].(string),
			IngressRuleValue: v1beta1.IngressRuleValue{
				HTTP: &v1beta1.HTTPIngressRuleValue{
					Paths: paths,
				},
			},
		}
	}
	return obj
}

func expandIngressSpec(l []interface{}) v1beta1.IngressSpec {
	if len(l) == 0 || l[0] == nil {
		return v1beta1.IngressSpec{}
	}
	in := l[0].(map[string]interface{})
	obj := v1beta1.IngressSpec{}

	if v, ok := in["ingress_class_name"].(string); ok && len(v) > 0 {
		obj.IngressClassName = &v
	}

	if v, ok := in["backend"].([]interface{}); ok && len(v) > 0 {
		obj.Backend = expandIngressBackend(v)
	}

	if v, ok := in["rule"].([]interface{}); ok && len(v) > 0 {
		obj.Rules = expandIngressRule(v)
	}

	if v, ok := in["tls"].([]interface{}); ok && len(v) > 0 {
		obj.TLS = expandIngressTLS(v)
	}

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

	if v, ok := in["service_port"].(string); ok {
		obj.ServicePort = intstr.Parse(v)
	}

	return obj
}

func expandIngressTLS(l []interface{}) []v1beta1.IngressTLS {
	if len(l) == 0 {
		return nil
	}

	tlsList := make([]v1beta1.IngressTLS, len(l))
	for i, t := range l {
		if t == nil {
			t = map[string]interface{}{}
		}
		in := t.(map[string]interface{})
		obj := v1beta1.IngressTLS{}

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
