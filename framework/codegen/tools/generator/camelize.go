package main

import "strings"

// Camelize converts a string containing snake_case into camelCase
// FIXME this wont work for variables containing ancronyms, e.g: pod_cidr, cluster_ip
// we should add a map of overrides so we can explicitly convert these
func Camelize(in string) string {
	out := ""
	cap := false
	for _, ch := range in {
		if ch == '_' {
			cap = true
			continue
		}
		if cap {
			out += strings.ToUpper(string(ch))
			cap = false
		} else {
			out += string(ch)
		}
	}
	return out
}

func UpperCamelize(in string) string {
	return strings.Title(Camelize(in))
}
