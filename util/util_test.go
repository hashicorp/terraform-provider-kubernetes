// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package util

import (
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestParseResourceID(t *testing.T) {
	cases := []struct {
		id        string
		namespace string
		name      string
		gvk       schema.GroupVersionKind
		err       error
	}{
		{
			id:        "apiVersion=v1,kind=ConfigMap,name=test",
			namespace: "default",
			name:      "test",
			gvk:       schema.FromAPIVersionAndKind("v1", "ConfigMap"),
		},
		{
			id:        "apiVersion=v1,kind=ConfigMap,name=test,namespace=kube-system",
			namespace: "kube-system",
			name:      "test",
			gvk:       schema.FromAPIVersionAndKind("v1", "ConfigMap"),
		},
		{
			id:        "apiVersion=apps/v1,kind=Deployment,name=app,namespace=test",
			namespace: "test",
			name:      "app",
			gvk:       schema.FromAPIVersionAndKind("apps/v1", "Deployment"),
		},
		{
			id:  "apiVersion=apps/v1,kind=Deployment,name=app,junk=test",
			err: fmt.Errorf(`could not parse ID: "apiVersion=apps/v1,kind=Deployment,name=app,junk=test". ID contained unknown key "junk"`),
		},
		{
			id:  "apiVersion_apps/v1,kind=Deployment,name=app",
			err: fmt.Errorf(`could not parse ID: "apiVersion_apps/v1,kind=Deployment,name=app". ID must be in key=value format`),
		},
		{
			id:  "junk",
			err: fmt.Errorf(`could not parse ID: "junk". ID must contain apiVersion, kind, and name`),
		},
	}

	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			gvk, n, ns, err := ParseResourceID(tc.id)
			if err != nil && tc.err.Error() != err.Error() {
				t.Errorf("expected error %q got %q", tc.err, err)
			}
			if tc.namespace != ns {
				t.Errorf("expected namespace %q got %q", tc.namespace, ns)
			}
			if tc.name != n {
				t.Errorf("expected name %q got %q", tc.name, n)
			}
			if tc.gvk != gvk {
				t.Errorf("expected GroupVersionKind %#v got %#v", tc.gvk, gvk)
			}
		})
	}
}
