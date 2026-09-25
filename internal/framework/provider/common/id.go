// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildID mirrors SDKv2 buildId: "namespace/name", including empty components.
// Cluster-scoped resources use the bare name instead.
func BuildID(meta metav1.ObjectMeta) string {
	return meta.Namespace + "/" + meta.Name
}

// ParseID mirrors SDKv2 idParts, requiring exactly two components and preserving its error text.
func ParseID(id string) (namespace, name string, err error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("Unexpected ID format (%q), expected %q.", id, "namespace/name")
	}

	return parts[0], parts[1], nil
}
