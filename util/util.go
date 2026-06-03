// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package util

// This package contains utility functions that are shared
// between the manifest provider and the main provider

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ParseResourceID processes the resource ID string and extracts
// the values for GVK, name and (optionally) namespace of the target resource
//
// The expected format for the resource ID is:
// "apiVersion=<value>,kind=<value>,name=<value>[,namespace=<value>"]
//
// where 'namespace' is only required for resources that expect a namespace.
// Example: "apiVersion=v1,kind=Secret,namespace=default,name=default-token-qgm6s"
func ParseResourceID(id string) (schema.GroupVersionKind, string, string, error) {
	parts := strings.Split(id, ",")
	if len(parts) < 3 || len(parts) > 4 {
		return schema.GroupVersionKind{}, "", "",
			fmt.Errorf("could not parse ID: %q. ID must contain apiVersion, kind, and name", id)
	}

	namespace := "default"
	var apiVersion, kind, name string
	for _, p := range parts {
		pp := strings.Split(p, "=")
		if len(pp) != 2 {
			return schema.GroupVersionKind{}, "", "",
				fmt.Errorf("could not parse ID: %q. ID must be in key=value format", id)
		}
		key := pp[0]
		val := pp[1]
		switch key {
		case "apiVersion":
			apiVersion = val
		case "kind":
			kind = val
		case "name":
			name = val
		case "namespace":
			namespace = val
		default:
			return schema.GroupVersionKind{}, "", "",
				fmt.Errorf("could not parse ID: %q. ID contained unknown key %q", id, key)
		}
	}

	gvk := schema.FromAPIVersionAndKind(apiVersion, kind)
	return gvk, name, namespace, nil
}

// ExpandHome expands a leading "~" in path to the current user's home directory.
func ExpandHome(path string) (string, error) {
	if path == "" || path[0] != '~' {
		return path, nil
	}

	if len(path) > 1 && path[1] != '/' && path[1] != '\\' {
		return "", fmt.Errorf("cannot expand user-specific home directory in path %q", path)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	if path == "~" {
		return home, nil
	}

	return filepath.Join(home, path[2:]), nil
}
