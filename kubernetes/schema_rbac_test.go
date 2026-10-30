// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestRBACRoleRefAPIGroupValidation(t *testing.T) {
	resources := map[string]*schema.Resource{
		"kubernetes_role_binding_v1":         resourceKubernetesRoleBindingV1(""),
		"kubernetes_cluster_role_binding_v1": resourceKubernetesClusterRoleBindingV1(""),
	}
	cases := map[string]struct {
		apiGroup  string
		expectErr bool
	}{
		"kubernetes":     {apiGroup: "rbac.authorization.k8s.io"},
		"openshift role": {apiGroup: "roles.authorization.openshift.io"},
		"empty":          {apiGroup: "", expectErr: true},
	}

	for resourceName, resource := range resources {
		for name, tc := range cases {
			t.Run(resourceName+"/"+name, func(t *testing.T) {
				raw := map[string]interface{}{
					"metadata": []interface{}{map[string]interface{}{
						"name":      "example",
						"namespace": "default",
					}},
					"role_ref": []interface{}{map[string]interface{}{
						"api_group": tc.apiGroup,
						"kind":      "ClusterRole",
						"name":      "admin",
					}},
					"subject": []interface{}{map[string]interface{}{
						"kind": "User",
						"name": "example",
					}},
				}
				if resourceName == "kubernetes_cluster_role_binding_v1" {
					delete(raw["metadata"].([]interface{})[0].(map[string]interface{}), "namespace")
				}

				diags := resource.Validate(terraform.NewResourceConfigRaw(raw))
				var apiGroupErr bool
				for _, d := range diags {
					for _, step := range d.AttributePath {
						if attr, ok := step.(cty.GetAttrStep); ok && attr.Name == "api_group" {
							apiGroupErr = true
						}
					}
				}
				if apiGroupErr != tc.expectErr {
					t.Fatalf("expected api_group error: %t, got diagnostics: %#v", tc.expectErr, diags)
				}
			})
		}
	}
}
