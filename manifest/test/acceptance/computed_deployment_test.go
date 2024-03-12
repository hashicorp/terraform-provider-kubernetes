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
	tfstatehelper "github.com/hashicorp/terraform-provider-kubernetes/manifest/test/helper/state"
)

func TestKubernetesManifest_ComputedDeploymentFields(t *testing.T) {
	ctx := context.Background()

	reattachInfo, err := provider.ServeTest(ctx, hclog.Default(), t)
	if err != nil {
		t.Errorf("Failed to create provider instance: %q", err)
	}

	name := strings.ToLower(randName())
	namespace := strings.ToLower(randName())

	k8shelper.CreateNamespace(t, namespace)
	defer k8shelper.DeleteResource(t, namespace, kubernetes.NewGroupVersionResource("v1", "namespaces"))

	tfvars := TFVARS{
		"name":      name,
		"namespace": namespace,
	}

	tf := tfhelper.RequireNewWorkingDir(ctx, t)
	tf.SetReattachInfo(ctx, reattachInfo)
	defer func() {
		tf.Destroy(ctx)
		tf.Close()
		k8shelper.AssertNamespacedResourceDoesNotExist(t, "apps/v1", "deployments", namespace, name)
	}()

	tfconfig := loadTerraformConfig(t, "ComputedFields/computed_deployment.tf", tfvars)
	tf.SetConfig(ctx, string(tfconfig))
	tf.Init(ctx)
	tf.Apply(ctx)

	s, err := tf.State(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve terraform state: %q", err)
	}
	tfstate := tfstatehelper.NewHelper(s)
	tfstate.AssertAttributeValues(t, tfstatehelper.AttributeValues{
		"kubernetes_manifest.deployment_resource_diff.object.metadata.name":                                           name,
		"kubernetes_manifest.deployment_resource_diff.object.metadata.namespace":                                      namespace,
		"kubernetes_manifest.deployment_resource_diff.object.spec.template.spec.containers.0.resources.limits.cpu":    "250m",
		"kubernetes_manifest.deployment_resource_diff.object.spec.template.spec.containers.0.resources.limits.memory": "512Mi",
	})
}
