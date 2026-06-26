// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

//go:build acceptance
// +build acceptance

package acceptance

import (
	"context"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/kubernetes"

	tfstatehelper "github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/state"
)

func TestDataSourceKubernetesResourceStatus_Deployment(t *testing.T) {
	ctx := context.Background()

	reattachInfo, err := provider.ServeTest(ctx, hclog.Default(), t)
	if err != nil {
		t.Errorf("Failed to create provider instance: %q", err)
	}

	name := randName()
	namespace := randName()

	k8shelper.CreateNamespace(t, namespace)
	defer k8shelper.DeleteResource(t, namespace, kubernetes.NewGroupVersionResource("v1", "namespaces"))

	// STEP 1: Create a Deployment to use as a data source
	tf := tfhelper.RequireNewWorkingDir(ctx, t)
	tf.SetReattachInfo(ctx, reattachInfo)
	defer func() {
		tf.Destroy(ctx)
		tf.Close()
		k8shelper.AssertNamespacedResourceDoesNotExist(t, "apps/v1", "deployments", namespace, name)
	}()

	tfvars := TFVARS{
		"name":      name,
		"namespace": namespace,
	}
	tfconfig := loadTerraformConfig(t, "datasource-resource-status/step1.tf", tfvars)
	tf.SetConfig(ctx, tfconfig)
	tf.Init(ctx)
	tf.Apply(ctx)

	k8shelper.AssertNamespacedResourceExists(t, "apps/v1", "deployments", namespace, name)

	state, err := tf.State(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve terraform state: %q", err)
	}
	tfstate := tfstatehelper.NewHelper(state)

	// STEP 2: Read the Deployment from step 1 using a kubernetes_resource data source
	reattachInfo2, err := provider.ServeTest(ctx, hclog.Default(), t)
	if err != nil {
		t.Errorf("Failed to create additional provider instance: %q", err)
	}
	step2 := tfhelper.RequireNewWorkingDir(ctx, t)
	step2.SetReattachInfo(ctx, reattachInfo2)
	defer func() {
		step2.Destroy(ctx)
		step2.Close()
		k8shelper.AssertNamespacedResourceDoesNotExist(t, "apps/v1", "deployments", namespace, name)
	}()

	tfconfig = loadTerraformConfig(t, "datasource-resource-status/step2.tf", tfvars)
	step2.SetConfig(ctx, string(tfconfig))
	step2.Init(ctx)
	step2.Apply(ctx)

	s2, err := step2.State(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve terraform state: %q", err)
	}
	tfstate = tfstatehelper.NewHelper(s2)

	// check that the data source has the status field defined
	tfstate.AssertAttributeNotEmpty(t, "data.kubernetes_resource.test_deploy.object.status")
}
