// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance
// +build acceptance

package acceptance

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/provider"
	tfstatehelper "github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/state"
)

func TestKubernetesManifest_CustomResourceDefinition(t *testing.T) {
	ctx := context.Background()

	reattachInfo, err := provider.ServeTest(ctx, hclog.Default(), t)
	if err != nil {
		t.Errorf("Failed to create provider instance: %q", err)
	}

	kind := strings.Title(randString(8))
	plural := strings.ToLower(kind) + "s"
	group := "terraform.io"
	version := "v1"
	name := fmt.Sprintf("%s.%s", plural, group)

	tf := tfhelper.RequireNewWorkingDir(ctx, t)
	tf.SetReattachInfo(ctx, reattachInfo)
	defer func() {
		tf.Destroy(ctx)
		tf.Close()
		k8shelper.AssertResourceDoesNotExist(t, "apiextensions.k8s.io/v1", "customresourcedefinitions", name)
	}()

	tfvars := TFVARS{
		"kind":       kind,
		"plural":     plural,
		"group":      group,
		"cr_version": version,
	}
	tfconfig := loadTerraformConfig(t, "CustomResourceDefinition/customresourcedefinition.tf", tfvars)
	tf.SetConfig(ctx, tfconfig)
	tf.Init(ctx)
	tf.Apply(ctx)

	k8shelper.AssertResourceExists(t, "apiextensions.k8s.io/v1", "customresourcedefinitions", name)

	s, err := tf.State(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve terraform state: %q", err)
	}
	tfstate := tfstatehelper.NewHelper(s)
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.name":           name,
		"kubernetes_manifest.test.object.spec.group":              group,
		"kubernetes_manifest.test.object.spec.names.kind":         kind,
		"kubernetes_manifest.test.object.spec.names.plural":       plural,
		"kubernetes_manifest.test.object.spec.scope":              "Namespaced",
		"kubernetes_manifest.test.object.spec.versions.0.name":    version,
		"kubernetes_manifest.test.object.spec.versions.0.served":  true,
		"kubernetes_manifest.test.object.spec.versions.0.storage": true,
		"kubernetes_manifest.test.object.spec.versions.0.schema": map[string]interface{}{
			"openAPIV3Schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"data": map[string]interface{}{
						"type": "string",
					},
					"refs": map[string]interface{}{
						"type": "number",
					},
					"otherData": map[string]interface{}{
						"type": "string",
					},
					"stuff": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"foo": map[string]interface{}{
									"type": "string",
								},
							},
						},
					},
					"limits": map[string]interface{}{
						"type": "object",
						"additionalProperties": map[string]interface{}{
							"x-kubernetes-int-or-string": true,
							"anyOf": []interface{}{
								map[string]interface{}{"type": "integer"},
								map[string]interface{}{"type": "string"},
							},
						},
					},
				},
			},
		},
		"kubernetes_manifest.test.object.spec.versions.1.name":    version + "beta1",
		"kubernetes_manifest.test.object.spec.versions.1.served":  true,
		"kubernetes_manifest.test.object.spec.versions.1.storage": false,
		"kubernetes_manifest.test.object.spec.versions.1.schema": map[string]interface{}{
			"openAPIV3Schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"data": map[string]interface{}{
						"type": "string",
					},
					"otherData": map[string]interface{}{
						"type": "string",
					},
					"refs": map[string]interface{}{
						"type": "number",
					},
				},
			},
		},
	})
}
