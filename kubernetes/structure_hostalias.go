package kubernetes

import "k8s.io/api/core/v1"

func flattenHostaliases(in []v1.HostAlias) []interface{} {
	att := make([]interface{}, len(in))
	for i, v := range in {
		ha := make(map[string]interface{})
		ha["ip"] = v.IP
		if len(v.Hostnames) > 0 {
			ha["hostnames"] = v.Hostnames
		}
		att[i] = ha
	}
	return att
}
func expandHostaliases(hostalias []interface{}) ([]v1.HostAlias, error) {
	if len(hostalias) == 0 {
		return []v1.HostAlias{}, nil
	}

	hs := make([]v1.HostAlias, len(hostalias))
	for i, ha := range hostalias {
		hoas := ha.(map[string]interface{})

		if ip, ok := hoas["ip"]; ok {
			hs[i].IP = ip.(string)
		}

		if hostnames, ok := hoas["hostnames"].([]interface{}); ok {
			hs[i].Hostnames = expandStringSlice(hostnames)
		}
	}
	return hs, nil
}
