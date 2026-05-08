// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance
// +build acceptance

package acceptance

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/kubernetes"
	tfstatehelper "github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/state"
)

func TestKubernetesManifest_fieldManager(t *testing.T) {
	ctx := context.Background()

	reattachInfo, err := provider.ServeTest(ctx, hclog.Default(), t)
	if err != nil {
		t.Errorf("Failed to create provider instance: %q", err)
	}

	name := randName()
	namespace := randName()

	tf := tfhelper.RequireNewWorkingDir(ctx, t)
	tf.SetReattachInfo(ctx, reattachInfo)
	defer func() {
		tf.Destroy(ctx)
		tf.Close()
		k8shelper.AssertNamespacedResourceDoesNotExist(t, "v1", "configmaps", namespace, name)
	}()

	k8shelper.CreateNamespace(t, namespace)
	defer k8shelper.DeleteResource(t, namespace, kubernetes.NewGroupVersionResource("v1", "namespaces"))

	// 1. Create the resource
	tfvars := TFVARS{
		"namespace":     namespace,
		"name":          name,
		"field_manager": "tftest",
		"force":         false,
		"data":          "bar",
	}
	tfconfig := loadTerraformConfig(t, "FieldManager/field_manager.tf", tfvars)
	tf.SetConfig(ctx, tfconfig)
	tf.Init(ctx)
	tf.Apply(ctx)

	k8shelper.AssertNamespacedResourceExists(t, "v1", "configmaps", namespace, name)

	s, err := tf.State(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve terraform state: %q", err)
	}
	tfstate := tfstatehelper.NewHelper(s)
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.namespace":       namespace,
		"kubernetes_manifest.test.object.metadata.name":            name,
		"kubernetes_manifest.test.object.data.foo":                 "bar",
		"kubernetes_manifest.test.field_manager.0.name":            "tftest",
		"kubernetes_manifest.test.field_manager.0.force_conflicts": false,
	})

	// 2. Try to change the resource with a new field manager name, should give a conflict
	tfvars = TFVARS{
		"namespace":     namespace,
		"name":          name,
		"field_manager": "tftest-newmanager",
		"force":         false,
		"data":          "foobar",
	}
	tfconfig = loadTerraformConfig(t, "FieldManager/field_manager.tf", tfvars)
	tf.SetConfig(ctx, tfconfig)
	tf.Init(ctx)
	err = tf.Apply(ctx) // this should fail
	if err == nil || !strings.Contains(err.Error(), "There was a field manager conflict when trying to apply the manifest") {
		t.Log(err.Error())
		t.Fatal("Expected terraform apply to cause a field manager conflict")
	}

	// 3. Try again with force_conflicts set to true, should succeed
	tfvars = TFVARS{
		"namespace":     namespace,
		"name":          name,
		"field_manager": "tftest-newmanager",
		"force":         true,
		"data":          "foobar",
	}
	tfconfig = loadTerraformConfig(t, "FieldManager/field_manager.tf", tfvars)
	tf.SetConfig(ctx, tfconfig)
	tf.Init(ctx)
	tf.Apply(ctx)

	k8shelper.AssertNamespacedResourceExists(t, "v1", "configmaps", namespace, name)

	s, err = tf.State(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve terraform state: %q", err)
	}
	tfstate = tfstatehelper.NewHelper(s)
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.namespace":       namespace,
		"kubernetes_manifest.test.object.metadata.name":            name,
		"kubernetes_manifest.test.object.data.foo":                 "foobar",
		"kubernetes_manifest.test.field_manager.0.name":            "tftest-newmanager",
		"kubernetes_manifest.test.field_manager.0.force_conflicts": true,
	})
}
