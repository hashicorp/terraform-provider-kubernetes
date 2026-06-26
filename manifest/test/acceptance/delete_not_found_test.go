// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance
// +build acceptance

package acceptance

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/kubernetes"
)

func TestKubernetesManifest_DeletionNotFound(t *testing.T) {
	ctx := context.Background()

	reattachInfo, err := provider.ServeTest(ctx, hclog.Default(), t)
	if err != nil {
		t.Fatalf("Failed to create provider instance: %v", err)
	}

	name := randName()
	namespace := randName()

	tf := tfhelper.RequireNewWorkingDir(ctx, t)
	tf.SetReattachInfo(ctx, reattachInfo)

	k8shelper.CreateNamespace(t, namespace)
	t.Logf("Verifying if namespace %s exists", namespace)
	k8shelper.AssertResourceExists(t, "v1", "namespaces", namespace)

	defer func() {
		tf.Destroy(ctx)
		tf.Close()
		k8shelper.DeleteResource(t, namespace, kubernetes.NewGroupVersionResource("v1", "namespaces"))
		k8shelper.AssertResourceDoesNotExist(t, "v1", "namespaces", namespace)
	}()

	tfvars := TFVARS{
		"namespace": namespace,
		"name":      name,
	}

	// Load the Terraform config that will create the ConfigMap
	tfconfig := loadTerraformConfig(t, "DeleteNotFoundTest/resource.tf", tfvars)
	tf.SetConfig(ctx, tfconfig)

	t.Log("Applying Terraform configuration to create ConfigMap")
	if err := tf.Apply(ctx); err != nil {
		t.Fatalf("Terraform apply failed: %v", err)
	}

	state, err := tf.State(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve Terraform state: %v", err)
	}
	t.Logf("Terraform state: %v", state)

	time.Sleep(2 * time.Second)

	t.Logf("Checking if ConfigMap %s in namespace %s was created", name, namespace)
	k8shelper.AssertNamespacedResourceExists(t, "v1", "configmaps", namespace, name)

	// Simulating the deletion of the resource outside of Terraform
	k8shelper.DeleteNamespacedResource(t, name, namespace, kubernetes.NewGroupVersionResource("v1", "configmaps"))

	// Running tf destroy in order to check if we are handling "404 Not Found" gracefully
	tf.Destroy(ctx)

	// Ensuring that the ConfigMap no longer exists
	k8shelper.AssertNamespacedResourceDoesNotExist(t, "v1", "configmaps", namespace, name)
}
