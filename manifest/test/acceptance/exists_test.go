// Copyright (c) HashiCorp, Inc.
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
)

func TestKubernetesManifest_alreadyExists(t *testing.T) {
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
		k8shelper.AssertNamespacedResourceDoesNotExist(t,
			"v1", "configmaps", namespace, name)
	}()

	k8shelper.CreateNamespace(t, namespace)
	defer k8shelper.DeleteResource(t, namespace, kubernetes.NewGroupVersionResource("v1", "namespaces"))

	tfvars := TFVARS{
		"namespace": namespace,
		"name":      name,
	}
	tfconfig := loadTerraformConfig(t, "alreadyExists/configmap.tf", tfvars)
	tf.SetConfig(ctx, tfconfig)
	tf.Init(ctx)
	tf.Apply(ctx)

	k8shelper.AssertNamespacedResourceExists(t,
		"v1", "configmaps", namespace, name)

	// Make a new working dir and apply again
	tf2 := tfhelper.RequireNewWorkingDir(ctx, t)
	tf2.SetReattachInfo(ctx, reattachInfo)
	tfconfigModified := loadTerraformConfig(t, "alreadyExists/configmap.tf", tfvars)
	tf2.SetConfig(ctx, tfconfigModified)
	err = tf2.Apply(ctx)

	if err == nil {
		t.Fatal("Creating a resource that already exists should cause an error")
	}

	errMsg := "Error: Cannot create resource that already exists"
	if err != nil && !strings.Contains(err.Error(), errMsg) {
		t.Errorf("Expected error to contain %q. Actual error:", errMsg)
		t.Log(err)
	}
}
