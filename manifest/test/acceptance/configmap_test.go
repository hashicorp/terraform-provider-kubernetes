// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance
// +build acceptance

package acceptance

import (
	"context"
	"testing"

	"github.com/hashicorp/go-hclog"
	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/kubernetes"
	tfstatehelper "github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/state"
)

// This test case tests a ConfigMap but also is a demonstration of some the assert functions
// available in the test helper
func TestKubernetesManifest_ConfigMap(t *testing.T) {
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

	tfvars := TFVARS{
		"namespace": namespace,
		"name":      name,
	}
	tfconfig := loadTerraformConfig(t, "ConfigMap/configmap.tf", tfvars)
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
		"kubernetes_manifest.test.object.metadata.namespace": namespace,
		"kubernetes_manifest.test.object.metadata.name":      name,
		"kubernetes_manifest.test.object.data.foo":           "bar",
	})

	tfconfigModified := loadTerraformConfig(t, "ConfigMap/configmap_modified.tf", tfvars)
	tf.SetConfig(ctx, tfconfigModified)
	tf.Apply(ctx)

	s2, err := tf.State(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve terraform state: %q", err)
	}
	tfstate = tfstatehelper.NewHelper(s2)
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.test.object.metadata.namespace":        namespace,
		"kubernetes_manifest.test.object.metadata.name":             name,
		"kubernetes_manifest.test.object.metadata.annotations.test": "1",
		"kubernetes_manifest.test.object.metadata.labels.test":      "2",
		"kubernetes_manifest.test.object.data.foo":                  "bar",
	})

	tfstate.AssertAttributeEqual(t, "kubernetes_manifest.test.object.data.fizz", "buzz")

	tfstate.AssertAttributeLen(t, "kubernetes_manifest.test.object.metadata.labels", 1)
	tfstate.AssertAttributeLen(t, "kubernetes_manifest.test.object.metadata.annotations", 1)

	tfstate.AssertAttributeNotEmpty(t, "kubernetes_manifest.test.object.metadata.labels.test")

	tfstate.AssertAttributeDoesNotExist(t, "kubernetes_manifest.test.spec")

	tfversion, err := tf.Version(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve terraform version: %v", err)
	}
	constraint, _ := version.NewConstraint(">= 1.12.0")
	if constraint.Check(tfversion) {
		tfstate.AssertIdentityValueEqual(t, "kubernetes_manifest.test", "api_version", "v1")
		tfstate.AssertIdentityValueEqual(t, "kubernetes_manifest.test", "kind", "ConfigMap")
		tfstate.AssertIdentityValueEqual(t, "kubernetes_manifest.test", "name", name)
		tfstate.AssertIdentityValueEqual(t, "kubernetes_manifest.test", "namespace", namespace)
	} else {
		t.Logf("Skipping identity assertions because terraform version %s is less than 1.12.0", tfversion)
	}
}
